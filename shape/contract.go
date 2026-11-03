package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/agent"
	"github.com/KogenAI/kogen-executor/executor"
	"kogen-loop/provider"
	"kogen-loop/sys"
	"kogen-loop/tools"
)

// The contract check (SHP-8): every new candidate that readiness accounted for goes to an
// independent read-only review that looks for acceptance expectations contradicting the Intent
// or the code it keeps unchanged, and to a confirmation of the findings Kogen found evidence for.
// Kogen alone decides which findings are evidenced; only confirmed ones refuse the candidate.

// SubmitContractReviewTool and SubmitContractConfirmationTool are the submit tools of the two
// calls of the contract check.
const (
	SubmitContractReviewTool       = "submit_contract_review"
	SubmitContractConfirmationTool = "submit_contract_confirmation"
)

const (
	contractGate         = "contract_review"
	contractRequestBound = 8  // model requests per conversation
	contractToolBound    = 32 // read-only tool calls per conversation

	// contractContradictionsFile is the record of the contradictions a not-ready Shape left, in
	// the shaping directory.
	contractContradictionsFile = "contract-contradictions.json"
)

var contractReviewInstructions = strings.Join([]string{
	"You check one candidate Intent before any Build is spent on it. Find acceptance expectations that no correct implementation of the Intent can pass.",
	"Read-only: use only read_file, list_files and search_text, then call " + SubmitContractReviewTool + " once.",
	"An expectation is an assertion, an expected value or output, or a call with its arguments, in the candidate acceptance.feature or in a step file the Intent lists under Acceptance. Report a finding only when the expectation contradicts the candidate intent.md (a rule, table, example or boundary it states) or the behaviour of repository code or text that the candidate does not change and the Intent keeps unchanged (for example an output envelope, a required argument, or a file another command writes).",
	"The readiness observations show how the unchanged tree behaved and which steps never ran. A scenario that fails only because the requested feature is missing is not a finding. Do not report style, coverage, weak tests, preferences or anything a correct implementation could satisfy.",
	"Separately, report an environment finding for a scenario that failed in the readiness observations because Kogen's fresh copy was not prepared for the step, not because the requested behaviour is missing. Examples are a store, schema or other state that neither the steps nor the configured setup create, or Git history, ignored files or other state that only the shaping checkout has. An environment finding gives scenario (the scenario name exactly as in acceptance.feature), output (copied exactly from that scenario's failure in the readiness observations) and why.",
	"When unsure, report nothing.",
	`For each finding give scenario (the scenario name exactly as in acceptance.feature), expectation (its path and the exact text copied from that file), contract (the path of intent.md or of the unchanged repository file, and the exact text copied from it that the expectation contradicts) and why (what a correct implementation does instead). Answer {"findings":[],"environment":[],"properties":[]} when there is none.`,
	"Separately, identify the properties the Request and the Questions and answers ask for: a quality, a bound, a comparison with the old behaviour, or a behaviour that must be kept. Take them from the Request and the answers themselves, not from what the candidate declares. A plain input/output request (a command, call or input with its expected output) asks for none: answer properties [].",
	"For each property give property (a short name), request (the exact text copied from the Request or the Questions and answers that asks for it), kind (kept for a behaviour that must be kept, new otherwise), observable (false only when no automatic step can observe it, such as a subjective quality), scenarios (the names, exactly as in acceptance.feature, of the scenarios that exercise it; for a property that is not observable, the scenarios that claim it holds, also through a figure, log line or document sentence standing in for it), decision (for a property that is not observable, the exact text of the candidate intent.md Decisions line that records it as needing human evaluation; otherwise empty) and issues.",
	"An issue is a scenario of the property whose evidence does not establish it, with problem, scenario, quote and why. Its problem is missing when a new property's scenario is red on the unchanged tree only because a file, module, function, command or test it needs does not exist yet (quote: that failure, copied exactly from the readiness observations), or test_count when the step's only evidence is that tests the Build writes exist or pass in some number and the step does not itself assert the property (quote: that code, copied exactly from the step file). A step that first gives the missing piece a do-nothing form and then fails on the property's behaviour has no issue.",
}, "\n")

var contractConfirmationInstructions = strings.Join([]string{
	"You independently verify reported contradictions in one candidate Intent before any Build is spent on it.",
	"Read-only: use only read_file, list_files and search_text, then call " + SubmitContractConfirmationTool + " once.",
	"For each numbered finding, inspect the cited expectation and contract yourself. Answer confirmed only when a correct implementation of the Intent cannot pass the expectation because of the cited contract; otherwise answer refuted. Give one verdict per finding with its number and a short why.",
	"For an environment finding, answer confirmed only when the cited fresh-copy output shows that the step failed because the copy was not prepared for it, not because the requested behaviour is missing.",
	"For a property problem, answer confirmed only when the Request or the Questions and answers ask for that property and the candidate has the stated problem; otherwise answer refuted.",
}, "\n")

// contractReadOnly are the inspection tools of both calls.
func contractReadOnly() []string {
	return []string{tools.ReadFileTool, tools.ListFilesTool, tools.SearchTextTool}
}

func contractQuoteSchema() map[string]any {
	s := map[string]any{"type": "string"}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"path": s, "quote": s},
		"required":             []any{"path", "quote"},
		"additionalProperties": false,
	}
}

func contractReviewSchema() map[string]any {
	s := map[string]any{"type": "string"}
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scenario": s, "expectation": contractQuoteSchema(), "contract": contractQuoteSchema(), "why": s,
		},
		"required":             []any{"scenario", "expectation", "contract", "why"},
		"additionalProperties": false,
	}
	env := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"scenario": s, "output": s, "why": s},
		"required":             []any{"scenario", "output", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"findings": map[string]any{"type": "array", "items": item}, "environment": map[string]any{"type": "array", "items": env}, "properties": map[string]any{"type": "array", "items": propertySchema()}},
		"required":             []any{"findings", "environment", "properties"},
		"additionalProperties": false,
	}
}

func contractConfirmationSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"finding": map[string]any{"type": "integer"},
			"verdict": map[string]any{"type": "string", "enum": []any{"confirmed", "refuted"}},
			"why":     map[string]any{"type": "string"},
		},
		"required":             []any{"finding", "verdict", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"verdicts": map[string]any{"type": "array", "items": item}},
		"required":             []any{"verdicts"},
		"additionalProperties": false,
	}
}

func contractToolDef(name, description string, schema map[string]any) map[string]any {
	return map[string]any{"type": "function", "name": name, "strict": true, "description": description, "parameters": schema}
}

func contractReviewToolDef() map[string]any {
	return contractToolDef(SubmitContractReviewTool, "Submit the contract review: the acceptance expectations that contradict the Intent or the code it keeps unchanged, each with quotes, and the scenarios that failed because Kogen's fresh copy was not prepared for them.", contractReviewSchema())
}

func contractConfirmationToolDef() map[string]any {
	return contractToolDef(SubmitContractConfirmationTool, "Submit the confirmation: one verdict per numbered finding.", contractConfirmationSchema())
}

// ---- payloads ----

type contractQuote struct{ path, quote string }

type contractFinding struct {
	scenario    string
	expectation contractQuote
	contract    contractQuote
	why         string
}

// contractEnvFinding is a scenario the review reports as failing in Kogen's fresh copy because the
// copy was not prepared for it.
type contractEnvFinding struct {
	scenario, output, why string
}

type contractVerdict struct {
	finding int
	verdict string
	why     string
}

func exactKeys(m map[string]any, keys ...string) bool {
	if len(m) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

func parseContractQuote(v any) (contractQuote, bool) {
	m, ok := v.(map[string]any)
	if !ok || !exactKeys(m, "path", "quote") {
		return contractQuote{}, false
	}
	p, ok1 := m["path"].(string)
	q, ok2 := m["quote"].(string)
	return contractQuote{p, q}, ok1 && ok2
}

// parseContractReview is the findings and environment findings of exactly one valid
// submit_contract_review payload, or why it is not one.
func parseContractReview(args map[string]any) ([]contractFinding, []contractEnvFinding, string) {
	findings, env, _, why := parseContractReviewProperties(args)
	return findings, env, why
}

// parseContractReviewProperties is parseContractReview with the properties of a payload that has
// the optional "properties" field.
func parseContractReviewProperties(args map[string]any) ([]contractFinding, []contractEnvFinding, []propertyReport, string) {
	withProps := exactKeys(args, "findings", "environment", "properties")
	if !withProps && !exactKeys(args, "findings", "environment") {
		return nil, nil, nil, `the payload must have exactly the fields "findings" and "environment", and optionally "properties"`
	}
	list, ok := args["findings"].([]any)
	if !ok {
		return nil, nil, nil, "findings must be a list"
	}
	out := make([]contractFinding, 0, len(list))
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok || !exactKeys(m, "scenario", "expectation", "contract", "why") {
			return nil, nil, nil, "a finding must be an object with exactly the fields scenario, expectation, contract, why"
		}
		var f contractFinding
		var ok1, ok2, ok3, ok4 bool
		f.scenario, ok1 = m["scenario"].(string)
		f.why, ok2 = m["why"].(string)
		f.expectation, ok3 = parseContractQuote(m["expectation"])
		f.contract, ok4 = parseContractQuote(m["contract"])
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, nil, nil, "a finding has a field of the wrong type"
		}
		out = append(out, f)
	}
	envList, ok := args["environment"].([]any)
	if !ok {
		return nil, nil, nil, "environment must be a list"
	}
	env := make([]contractEnvFinding, 0, len(envList))
	for _, raw := range envList {
		m, ok := raw.(map[string]any)
		if !ok || !exactKeys(m, "scenario", "output", "why") {
			return nil, nil, nil, "an environment finding must be an object with exactly the fields scenario, output, why"
		}
		var f contractEnvFinding
		var ok1, ok2, ok3 bool
		f.scenario, ok1 = m["scenario"].(string)
		f.output, ok2 = m["output"].(string)
		f.why, ok3 = m["why"].(string)
		if !ok1 || !ok2 || !ok3 {
			return nil, nil, nil, "an environment finding has a field of the wrong type"
		}
		env = append(env, f)
	}
	var props []propertyReport
	if withProps {
		var why string
		if props, why = parsePropertyReports(args["properties"]); why != "" {
			return nil, nil, nil, why
		}
	}
	return out, env, props, ""
}

// integerArg is a JSON integer as the argument decoders deliver it.
func integerArg(v any) (int, bool) {
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		return int(n), err == nil
	case int64:
		return int(x), int64(int(x)) == x
	case float64:
		return int(x), x == float64(int(x))
	case int:
		return x, true
	}
	return 0, false
}

// parseContractConfirmation is the verdicts of exactly one valid submit_contract_confirmation
// payload, or why it is not one.
func parseContractConfirmation(args map[string]any) ([]contractVerdict, string) {
	if !exactKeys(args, "verdicts") {
		return nil, `the payload must have exactly the field "verdicts"`
	}
	list, ok := args["verdicts"].([]any)
	if !ok {
		return nil, "verdicts must be a list"
	}
	out := make([]contractVerdict, 0, len(list))
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok || !exactKeys(m, "finding", "verdict", "why") {
			return nil, "a verdict must be an object with exactly the fields finding, verdict, why"
		}
		n, ok1 := integerArg(m["finding"])
		verdict, ok2 := m["verdict"].(string)
		why, ok3 := m["why"].(string)
		if !ok1 || !ok2 || !ok3 || (verdict != "confirmed" && verdict != "refuted") {
			return nil, "a verdict has a field of the wrong type or value"
		}
		out = append(out, contractVerdict{n, verdict, why})
	}
	return out, ""
}

// ---- the mechanical rules ----

// contractEvidence is what Kogen compares a finding with: the candidate's scenario names, the
// files an expectation may cite (as written on disk) and the files a contract may cite.
type contractEvidence struct {
	scenarios    []string          // the declared scenario and outline names, normalized
	featurePath  string            // the path of acceptance.feature among expectations
	expectations map[string]string // path to the content on disk: acceptance.feature and the listed step files
	intentPath   string
	intent       string
	// failures are the failure details of the scenarios that failed in this candidate's readiness
	// result, by normalized scenario name.
	failures map[string][]string
	// tracked is the HEAD content of a file tracked at HEAD that is not a file of the candidate
	// package; false for any other path.
	tracked func(path string) (string, bool)
}

func (e contractEvidence) evidenced(f contractFinding) bool {
	if strings.TrimSpace(f.why) == "" || !slices.Contains(e.scenarios, normalizeWhitespace(f.scenario)) {
		return false
	}
	text, ok := e.expectations[f.expectation.path]
	if !ok || normalizeWhitespace(f.expectation.quote) == "" || !containsNormalized(text, f.expectation.quote) {
		return false
	}
	if normalizeWhitespace(f.contract.quote) == "" {
		return false
	}
	if f.contract.path == e.intentPath {
		return containsNormalized(e.intent, f.contract.quote)
	}
	head, ok := e.tracked(f.contract.path)
	return ok && containsNormalized(head, f.contract.quote)
}

// evidencedEnv reports whether an environment finding names a scenario that failed in this
// candidate's readiness run and cites output that run's failure detail for it contains.
func (e contractEvidence) evidencedEnv(f contractEnvFinding) bool {
	if strings.TrimSpace(f.why) == "" || normalizeWhitespace(f.output) == "" {
		return false
	}
	return slices.ContainsFunc(e.failures[normalizeWhitespace(f.scenario)], func(detail string) bool {
		return containsNormalized(detail, f.output)
	})
}

// contractItem is one finding with what Kogen decided about it.
type contractItem struct {
	finding      contractFinding
	evidenced    bool
	verdict      string  // "", confirmed or refuted
	confirmation *string // the counted verdict's why
}

// contractEnvItem is one environment finding with what Kogen decided about it.
type contractEnvItem struct {
	finding      contractEnvFinding
	evidenced    bool
	verdict      string
	confirmation *string
}

// contractConfirmed are the item lines of the confirmed findings of a refused candidate.
type contractConfirmed struct {
	contradictions []string
	environment    []string
	properties     []string
}

func (c contractConfirmed) empty() bool {
	return len(c.contradictions) == 0 && len(c.environment) == 0 && len(c.properties) == 0
}

// all is every line, contradictions first, as the contradictions record keeps them.
func (c contractConfirmed) all() []string {
	return append(append(slices.Clone(c.contradictions), c.environment...), c.properties...)
}

// contractFindingLine is the confirmation input's line for finding n (from 1), values normalized.
func contractFindingLine(n int, f contractFinding) string {
	return fmt.Sprintf("%d. scenario: %s | %s", n, normalizeWhitespace(f.scenario), contractBody(f))
}

func contractBody(f contractFinding) string {
	nw := normalizeWhitespace
	return fmt.Sprintf("expectation: %s: %s | contract: %s: %s | why: %s",
		nw(f.expectation.path), nw(f.expectation.quote), nw(f.contract.path), nw(f.contract.quote), nw(f.why))
}

// contractEnvFindingLine is the confirmation input's line for environment finding n (from 1).
func contractEnvFindingLine(n int, f contractEnvFinding) string {
	return fmt.Sprintf("%d. scenario: %s | %s", n, normalizeWhitespace(f.scenario), contractEnvBody(f))
}

func contractEnvBody(f contractEnvFinding) string {
	nw := normalizeWhitespace
	return fmt.Sprintf("fresh-copy output: %s | why: %s", nw(f.output), nw(f.why))
}

// contractEnvItemLine is the feedback line of a confirmed environment finding.
func contractEnvItemLine(f contractEnvFinding, confirmation string) string {
	return fmt.Sprintf("- %s | %s | confirmed: %s", normalizeWhitespace(f.scenario), contractEnvBody(f), normalizeWhitespace(confirmation))
}

// contractItemLine is the feedback line of a confirmed finding.
func contractItemLine(f contractFinding, confirmation string) string {
	return fmt.Sprintf("- %s | %s | confirmed: %s", normalizeWhitespace(f.scenario), contractBody(f), normalizeWhitespace(confirmation))
}

// contractHeadline is the first line of the refusal of n confirmed findings.
func contractHeadline(n int) string {
	return fmt.Sprintf("Contract check: %d acceptance expectation(s) contradict the Intent or code it keeps unchanged, so no correct implementation can pass them; nothing was published. "+
		"Correct each expectation, or the Intent where the Intent is what is wrong, so that a correct implementation passes; do not delete or weaken a scenario to pass this check. "+
		"Resubmitting this candidate unchanged is refused again.", n)
}

// environmentHeadline is the first line of the environment part of a refusal of n confirmed
// environment findings.
func environmentHeadline(n int) string {
	return fmt.Sprintf("Environment check: %d acceptance step(s) fail on the unchanged tree because Kogen's fresh copy is not prepared for them, not because the requested behaviour is missing; nothing was published. "+
		"Kogen runs acceptance in a fresh single-commit copy of the tree, without history or ignored files, after only the configured setup: make the step prepare the state it needs itself or in an earlier step of its scenario, or stop depending on state that copy lacks; do not delete or weaken a scenario to pass this check. "+
		"Resubmitting this candidate unchanged is refused again.", n)
}

// contractFeedback is the refusal text: the contract headline and one line per confirmed
// contradiction, then the environment headline and one line per confirmed environment finding,
// then the property headline and one line per confirmed property problem.
func contractFeedback(c contractConfirmed) string {
	var out []string
	if len(c.contradictions) > 0 {
		out = append(append(out, contractHeadline(len(c.contradictions))), c.contradictions...)
	}
	if len(c.environment) > 0 {
		out = append(append(out, environmentHeadline(len(c.environment))), c.environment...)
	}
	if len(c.properties) > 0 {
		out = append(append(out, propertyHeadline(len(c.properties))), c.properties...)
	}
	return strings.Join(out, "\n")
}

// applyVerdicts gives each evidenced item its counted verdict: the first verdict for n, n from
// 1 to the number of findings sent (contradictions first, then environment findings). A finding
// without a counted verdict is refuted. It returns the item lines of the confirmed findings.
func applyVerdicts(items []contractItem, envItems []contractEnvItem, verdicts []contractVerdict) contractConfirmed {
	sent := 0
	for _, it := range items {
		if it.evidenced {
			sent++
		}
	}
	for _, it := range envItems {
		if it.evidenced {
			sent++
		}
	}
	counted := map[int]contractVerdict{}
	for _, v := range verdicts {
		if _, seen := counted[v.finding]; !seen && v.finding >= 1 && v.finding <= sent {
			counted[v.finding] = v
		}
	}
	var lines contractConfirmed
	n := 0
	// decide counts the verdict of the next evidenced finding, if any.
	decide := func(verdict *string, confirmation **string, line func(why string) string, confirmed *[]string) {
		n++
		v, ok := counted[n]
		if !ok {
			*verdict = "refuted"
			return
		}
		*verdict = v.verdict
		why := v.why
		*confirmation = &why
		if v.verdict == "confirmed" {
			*confirmed = append(*confirmed, line(v.why))
		}
	}
	for i := range items {
		it := &items[i]
		if it.evidenced {
			decide(&it.verdict, &it.confirmation, func(why string) string { return contractItemLine(it.finding, why) }, &lines.contradictions)
		}
	}
	for i := range envItems {
		it := &envItems[i]
		if it.evidenced {
			decide(&it.verdict, &it.confirmation, func(why string) string { return contractEnvItemLine(it.finding, why) }, &lines.environment)
		}
	}
	return lines
}

// contractEvent is the contract_review event of a review that ran.
func contractEvent(hash, result, reason string, findings, evidenced, confirmed int, items []contractItem, envItems []contractEnvItem) map[string]any {
	list := make([]any, len(items))
	quote := func(q contractQuote) map[string]any { return map[string]any{"path": q.path, "quote": q.quote} }
	for i, it := range items {
		var verdict, confirmation any
		if it.verdict != "" {
			verdict = it.verdict
		}
		if it.confirmation != nil {
			confirmation = *it.confirmation
		}
		list[i] = map[string]any{
			"scenario": it.finding.scenario, "expectation": quote(it.finding.expectation), "contract": quote(it.finding.contract),
			"why": it.finding.why, "evidenced": it.evidenced, "verdict": verdict, "confirmation": confirmation,
		}
	}
	envList := make([]any, len(envItems))
	for i, it := range envItems {
		var verdict, confirmation any
		if it.verdict != "" {
			verdict = it.verdict
		}
		if it.confirmation != nil {
			confirmation = *it.confirmation
		}
		envList[i] = map[string]any{
			"scenario": it.finding.scenario, "output": it.finding.output, "why": it.finding.why,
			"evidenced": it.evidenced, "verdict": verdict, "confirmation": confirmation,
		}
	}
	event := map[string]any{"event": "contract_review", "package": hash, "result": result,
		"findings": findings, "evidenced": evidenced, "confirmed": confirmed, "items": list, "environment": envList}
	if reason != "" {
		event["reason"] = reason
	}
	return event
}

// contractReviewInput is the user message of the review: the slug and the sections, each
// content without its trailing newlines.
func contractReviewInput(slug string, sections [][2]string) string {
	var b strings.Builder
	b.WriteString("Slug: " + slug)
	for _, s := range sections {
		b.WriteString("\n\n# " + s[0] + "\n\n" + strings.TrimRight(s[1], "\n"))
	}
	return b.String()
}

// ---- the check ----

// packageHash is the hash `kogen intent approve` shows for the files on disk.
func (sh *shaping) packageHash() (string, error) {
	baseline, err := project.Baseline(sh.root, sh.slug, sh.cfg)
	if err != nil {
		return "", err
	}
	return project.Hash(sh.root, baseline), nil
}

// contractRefusal is the refusal of a candidate already refused by the contract check in this
// run, with the same feedback text; false for any other candidate.
func (sh *shaping) contractRefusal(hash string) (string, outcome, bool) {
	confirmed, ok := sh.contractRefused[hash]
	if !ok {
		return "", outcome{}, false
	}
	lines := confirmed.all()
	sh.contractLines = lines
	sh.refused(contractGate, lines)
	return contractFeedback(confirmed), outcome{kind: outInvalid, problems: lines}, true
}

// contractCheck gives the candidate on disk to the review and, for the findings Kogen found
// evidence for, to the confirmation. It returns the item lines of the confirmed findings, nil
// when the candidate may go on.
func (sh *shaping) contractCheck(ctx context.Context, hash, observations string) (contractConfirmed, error) {
	intentBytes, err := os.ReadFile(filepath.Join(sh.root, project.IntentMDPath(sh.slug)))
	if err != nil {
		return contractConfirmed{}, err
	}
	featureBytes, err := os.ReadFile(filepath.Join(sh.root, project.FeaturePath(sh.slug)))
	if err != nil {
		return contractConfirmed{}, err
	}
	intent, err := project.ReadIntent(sh.root, sh.slug, nil)
	if err != nil {
		return contractConfirmed{}, err
	}
	scenarios, err := project.ParseGherkin(string(featureBytes))
	if err != nil {
		return contractConfirmed{}, err
	}
	feature := project.FeaturePath(sh.slug)
	sections := [][2]string{{"Candidate intent.md", string(intentBytes)}, {"Candidate acceptance.feature", string(featureBytes)}}
	ev := contractEvidence{
		expectations: map[string]string{feature: string(featureBytes)}, featurePath: feature,
		intentPath: project.IntentMDPath(sh.slug), intent: string(intentBytes),
	}
	for _, p := range intent.StepFiles() {
		text, err := os.ReadFile(filepath.Join(sh.root, p))
		if err != nil {
			continue
		}
		ev.expectations[p] = string(text)
		if _, proposed := sh.proposed[p]; proposed {
			sections = append(sections, [2]string{"Candidate step file " + p, string(text)})
		}
	}
	for _, sc := range scenarios {
		ev.scenarios = append(ev.scenarios, normalizeWhitespace(sc.Name))
		if sc.Outline != nil {
			ev.scenarios = append(ev.scenarios, normalizeWhitespace(sc.Outline.Name))
		}
	}
	if sh.ready != nil {
		ev.failures = map[string][]string{}
		for _, sc := range sh.ready.Scenarios {
			if sc.Status == "failed" {
				name := normalizeWhitespace(sc.Name)
				detail := ""
				if sc.Detail != nil {
					detail = *sc.Detail
				}
				ev.failures[name] = append(ev.failures[name], detail)
			}
		}
	}
	ev.tracked = func(path string) (string, bool) {
		if !normalPath(path) || path == ev.intentPath || path == feature || path == project.ModelPath(sh.slug) {
			return "", false
		}
		if _, proposed := sh.proposed[path]; proposed {
			return "", false
		}
		if kind, err := sh.repo.OK("cat-file", "-t", "HEAD:"+path); err != nil || strings.TrimSpace(kind) != "blob" {
			return "", false
		}
		return sh.repo.Show("HEAD", path)
	}
	if observations == "" {
		observations = "All scenarios pass now."
	}
	questions := sh.earlier
	answers := questions
	if answers == "" {
		answers = "none"
	}
	reviewInput := contractReviewInput(sh.slug, append(sections, [2]string{"Readiness observations", observations},
		[2]string{"Request", sh.request}, [2]string{"Questions and answers", answers}))

	incomplete := func(reason string, findings, evidenced int, items []contractItem, envItems []contractEnvItem) (contractConfirmed, error) {
		sh.progress("kogen: contract check incomplete (" + reason + "); the candidate goes on")
		sh.ev.Record(contractEvent(hash, "incomplete", reason, findings, evidenced, 0, items, envItems))
		return contractConfirmed{}, nil
	}

	args, why, err := sh.contractCall(ctx, "review", reviewInput)
	if err != nil {
		return contractConfirmed{}, err
	}
	var findings []contractFinding
	var envFindings []contractEnvFinding
	var props []propertyReport
	if why == "" {
		if findings, envFindings, props, why = parseContractReviewProperties(args); why != "" {
			why = "invalid " + SubmitContractReviewTool + " payload: " + why
		}
	}
	if why != "" {
		return incomplete(why, 0, 0, nil, nil)
	}
	items := make([]contractItem, len(findings))
	var evidenced []string
	for i, f := range findings {
		items[i] = contractItem{finding: f, evidenced: ev.evidenced(f)}
		if items[i].evidenced {
			evidenced = append(evidenced, contractFindingLine(len(evidenced)+1, f))
		}
	}
	envItems := make([]contractEnvItem, len(envFindings))
	for i, f := range envFindings {
		envItems[i] = contractEnvItem{finding: f, evidenced: ev.evidencedEnv(f)}
		if envItems[i].evidenced {
			evidenced = append(evidenced, contractEnvFindingLine(len(evidenced)+1, f))
		}
	}
	total := len(findings) + len(envFindings)
	sent := len(evidenced)
	counted := make([]bool, len(props))
	for i, p := range props {
		counted[i] = propertyCounts(p, sh.request, questions)
	}
	problems := ev.propertyProblems(props, sh.request, questions)
	for _, pr := range problems {
		evidenced = append(evidenced, propertyConfirmLine(len(evidenced)+1, props[pr.prop], pr))
	}
	recordProperties := func() {
		if len(props) > 0 {
			sh.ev.Record(propertyEvent(hash, props, counted, problems))
		}
	}
	if len(evidenced) == 0 {
		sh.ev.Record(contractEvent(hash, "clean", "", total, 0, 0, items, envItems))
		recordProperties()
		return contractConfirmed{}, nil
	}
	args, why, err = sh.contractCall(ctx, "confirmation", reviewInput+"\n\n# Findings to verify\n\n"+strings.Join(evidenced, "\n"))
	if err != nil {
		return contractConfirmed{}, err
	}
	var verdicts []contractVerdict
	if why == "" {
		if verdicts, why = parseContractConfirmation(args); why != "" {
			why = "invalid " + SubmitContractConfirmationTool + " payload: " + why
		}
	}
	if why != "" {
		out, err := incomplete(why, total, sent, items, envItems)
		recordProperties()
		return out, err
	}
	lines := applyVerdicts(items, envItems, verdicts)
	lines.properties = applyPropertyVerdicts(props, problems, sent+1, verdicts)
	result := "clean"
	confirmed := len(lines.contradictions) + len(lines.environment)
	if !lines.empty() {
		result = "refused"
	}
	sh.ev.Record(contractEvent(hash, result, "", total, sent, confirmed, items, envItems))
	recordProperties()
	return lines, nil
}

// contractCall makes the review ("review") or the confirmation ("confirmation"): one fresh
// read-only conversation or session. It returns the arguments of the submit call, or why the
// call ended incomplete.
func (sh *shaping) contractCall(ctx context.Context, kind, input string) (map[string]any, string, error) {
	instructions, submit, schema, def, doing := contractReviewInstructions, SubmitContractReviewTool, contractReviewSchema(), contractReviewToolDef(), "kogen: checking the contract"
	if kind == "confirmation" {
		instructions, submit, schema, def, doing = contractConfirmationInstructions, SubmitContractConfirmationTool, contractConfirmationSchema(), contractConfirmationToolDef(), "kogen: confirming contract findings"
	}
	if sh.ext != nil {
		return sh.contractExternal(ctx, kind, instructions+"\n\n"+input, schema)
	}
	sh.progress(doing)
	revision, err := CurrentRevision(sh.repo)
	if err != nil {
		return nil, "", err
	}
	defs := append(agent.AsAny(agent.Defs(contractReadOnly()...)), def)
	cacheID := startConversation(sh.ev, "contract-"+kind)
	tl := sh.agentTools(ctx, contractReadOnly())
	conversation := []any{map[string]any{"role": "user", "content": input}}
	used := 0
	for requests := 0; requests < contractRequestBound; requests++ {
		req := conversationRequest(provider.Request{
			Role: ProviderRole(sh.role), Instructions: instructions, Input: conversation, Tools: defs,
			Scope: sh.ev.Dir, Context: map[string]any{"phase": "shape", "role": "shape", "slug": sh.slug},
		}, cacheID)
		if m, err := tools.Measure(sh.client.Body(req), req.Role); err != nil {
			return nil, "", err
		} else if !m.Fits {
			return nil, fmt.Sprintf("a request of %d bytes is above the %d-byte ceiling", m.RenderedBytes, provider.InputBudgetBytes), nil
		}
		call := sh.ev.BeginCall("shape", "", revision.Map(), "contract "+kind, sh.role).Sized(sh.client, req)
		t0 := time.Now()
		resp, err := sh.client.Respond(ctx, sh.s, req, call)
		call.End(resp, err)
		if err != nil {
			if e, ok := sys.AsError(err); ok && e.Class == "provider" {
				return nil, "", sh.providerFailed(e)
			}
			return nil, "", err
		}
		sh.progress(fmt.Sprintf("kogen: contract %s response %s wall_ms=%d", kind, resp.Usage.Line(), time.Since(t0).Milliseconds()))
		sh.contractUsage = sh.contractUsage.Add(resp.Usage)
		calls := tools.Calls(resp.Items)
		if len(calls) == 0 {
			return nil, "the answer called no tool", nil
		}
		if slices.ContainsFunc(calls, func(c tools.Call) bool { return c.Name == submit }) {
			if len(calls) != 1 {
				return nil, "the answer called other tools besides " + submit, nil
			}
			return calls[0].Args(), "", nil
		}
		conversation = append(conversation, tools.Replay(resp.Items)...)
		for _, c := range calls {
			if used >= contractToolBound {
				return nil, fmt.Sprintf("the %d read-only tool call bound was reached", contractToolBound), nil
			}
			if err := interrupted(ctx, sh.s); err != nil {
				return nil, "", err
			}
			out, err := tl.Run(c.Name, c.Args())
			if err != nil {
				return nil, "", err
			}
			used++
			conversation = append(conversation, toolOutput(c, out))
		}
	}
	return nil, fmt.Sprintf("the %d request bound was reached", contractRequestBound), nil
}

// contractExternal is one fresh read-only external session that answers the call with its
// structured result.
func (sh *shaping) contractExternal(ctx context.Context, kind, task string, schema map[string]any) (map[string]any, string, error) {
	folder := "executor-contract-review"
	if kind == "confirmation" {
		folder = "executor-contract-confirm"
	}
	sh.contractSessions[kind]++
	x, err := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"),
		fmt.Sprintf("%s/%s-%d", sh.ev.Dir, folder, sh.contractSessions[kind]), sh.slug, sh.role, task, "", mustJSON(schema), sh.ev, nil)
	if err != nil {
		return nil, "", err
	}
	defer x.close()
	res, err := x.round(ctx, "")
	sh.contractRounds += x.rounds
	if e, ok := sys.AsError(err); ok && e.Class == "executor" && e.Reason == executor.ReasonProtocol {
		return nil, "the session ended without a result", nil
	}
	if err != nil {
		return nil, "", err
	}
	if res.Ending != executor.EndFinish {
		return nil, "the session ended without a result", nil
	}
	args, err := submissionArgs(res.Structured)
	if err != nil {
		return nil, "the result is not a JSON object", nil
	}
	return args, "", nil
}

// ---- the record approval reads ----

type contractContradictions struct {
	SHA256 string   `json:"sha256"`
	Lines  []string `json:"lines"`
}

// writeContractContradictions records the item lines against the hash of the package left on
// disk, the hash `kogen intent approve` shows.
func (sh *shaping) writeContractContradictions(lines []string) error {
	hash, err := sh.packageHash()
	if err != nil {
		return err
	}
	b, err := json.Marshal(contractContradictions{SHA256: hash, Lines: lines})
	if err != nil {
		return err
	}
	path := filepath.Join(sh.stateDir, contractContradictionsFile)
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// approveContractContradictions refuses a package whose current hash is the one Shape left
// contradicting. A record that does not parse or names another hash does not affect approval.
func approveContractContradictions(root, slug, sha string) error {
	b, err := os.ReadFile(filepath.Join(root, ".kogen/local/shaping", slug, contractContradictionsFile))
	if err != nil {
		return nil
	}
	var rec contractContradictions
	if json.Unmarshal(b, &rec) != nil || rec.SHA256 != sha {
		return nil
	}
	e := sys.Fail("intent", "contract_contradicts", "Shape's contract check found acceptance expectations that contradict the Intent or code it keeps unchanged; correct the Intent's files, or shape it again with kogen intent shape "+slug+" <request-file>", 1)
	e.Lines = rec.Lines
	return e
}
