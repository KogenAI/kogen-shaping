package shape

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
	"github.com/KogenAI/kogen-verification/verify"
)

// CheckPackageTool is the shaper's pre-submit lint: it runs the deterministic gates of a
// submission, all of them, on the accumulated proposals (plus the files of the call, not
// retained) and returns every problem at once. It calls no model, runs nothing, writes nothing
// to the checkout, records no refusal and is not a submission attempt.
const CheckPackageTool = "check_package"

const checkPackageDescription = "Lint the package before you submit it, without spending a submission attempt. Checks, with the same code the submit gates run and with no model call, everything Kogen decides mechanically: the shape of the submit arguments (files, questions, blocked), decisions (evidence, uncertainty, revisions), the Intent lint (required files, sections, Decisions lines with their trailing source tag and (assumption: ...) marker), step file ownership, the mandatory-review byte budget, carried decisions, decision evidence paths, controls.json against the schema, and the step limit of step files. Without arguments it checks the proposals retained so far; files, decisions, questions or blocked are checked as if submitted but are not retained. Pass literals (exact strings the request or an answer requires) to check that the package carries each byte for byte. It does not run the model checks (does the request support each decision, is a question technical, contract review, examples), the step self-check, the static checks or readiness: those still run at submit. Returns every problem found."

func checkPackageDef() map[string]any {
	submit := submitDef()["parameters"].(map[string]any)["properties"].(map[string]any)
	props := map[string]any{
		"files":     proposalFileInputs(),
		"decisions": submit["decisions"],
		"questions": submit["questions"],
		"blocked":   submit["blocked"],
		"literals":  strList(),
	}
	return fun(CheckPackageTool, checkPackageDescription, props)
}

// notCheckedHere is the last line of every answer: what only a submission decides.
const notCheckedHere = "Not checked here (needs a model or a run; submit decides): whether the request supports each decision, question classification, request literals you did not pass in literals, contract review, examples, step self-check, static checks, readiness."

// lintGate is the problems of one gate, as that gate words them.
type lintGate struct {
	gate  string
	lines []string
}

// packageLint is everything check_package found.
type packageLint struct {
	gates []lintGate
	notes []string
}

func (l *packageLint) add(gate string, lines ...string) {
	if len(lines) == 0 {
		return
	}
	l.gates = append(l.gates, lintGate{gate, lines})
}

func (l *packageLint) count() int {
	n := 0
	for _, g := range l.gates {
		n += len(g.lines)
	}
	return n
}

func (l *packageLint) text() string {
	var b strings.Builder
	if n := l.count(); n == 0 {
		b.WriteString("check_package: no deterministic problem found. Nothing was submitted and no model was called; this is not a submission attempt.")
	} else {
		fmt.Fprintf(&b, "check_package: %d deterministic problem(s) in %d gate(s). Nothing was submitted and no model was called; this is not a submission attempt. Fix all of them before submit.", n, len(l.gates))
	}
	for _, g := range l.gates {
		b.WriteString("\n\n[" + g.gate + "]\n" + strings.Join(g.lines, "\n"))
	}
	if len(l.notes) > 0 {
		b.WriteString("\n\nNotes:\n" + strings.Join(l.notes, "\n"))
	}
	return b.String() + "\n\n" + notCheckedHere
}

// checkPackage is the handler of check_package. Only an error of the checkout itself fails the
// call; every finding is part of the answer.
func (sh *shaping) checkPackage(args map[string]any) (string, error) {
	l, err := sh.lintPackage(args)
	if err != nil {
		return "", err
	}
	return l.text(), nil
}

// lintShadow is a shaping that the gates may mutate and record into without touching sh: its
// own proposals, ledger, state directory and evidence. The caller removes dir.
func (sh *shaping) lintShadow(merged map[string]string) (c *shaping, dir string, err error) {
	if dir, err = os.MkdirTemp("", "check-package-"); err != nil {
		return nil, "", err
	}
	ev, err := NewEvidence(dir, sh.slug)
	if err != nil {
		os.RemoveAll(dir)
		return nil, "", err
	}
	c = &shaping{
		s: sh.s, in: sh.in, repo: sh.repo, root: sh.root, slug: sh.slug, cfg: sh.cfg, role: sh.role, ev: ev, rd: sh.rd,
		client: sh.client, stateDir: dir, ext: sh.ext, extMax: sh.extMax, started: sh.started, originals: sh.originals,
		proposed: merged, declaredSteps: slices.Clone(sh.declaredSteps), ready: sh.ready, inspected: sh.inspected,
		externalReads: sh.externalReads, docs: sh.docs, window: sh.window, round: sh.round, tracked: sh.tracked,
		request: sh.request, rounds: sh.rounds, earlier: sh.earlier, probed: sh.probed, probeTexts: sh.probeTexts,
		probeOutputs: sh.probeOutputs, readTexts: sh.readTexts, ranCommand: sh.ranCommand, readPaths: sh.readPaths,
		listedPaths: sh.listedPaths,
	}
	if sh.ledger != nil {
		c.ledger = &decisionLedger{Decisions: slices.Clone(sh.ledger.Decisions)}
	}
	return c, dir, nil
}

// lintText is the content of a package path: the candidate's, else the checkout's.
func (sh *shaping) lintText(merged map[string]string, p string) (string, bool) {
	if c, ok := merged[p]; ok {
		return c, true
	}
	b, err := os.ReadFile(filepath.Join(sh.root, p))
	return string(b), err == nil
}

func (sh *shaping) lintPackage(args map[string]any) (*packageLint, error) {
	out := &packageLint{}
	merged := maps.Clone(sh.proposed)
	if merged == nil {
		merged = map[string]string{}
	}
	var decisions []sourcedDecision
	_, hasQ := args["questions"]
	_, hasF := args["files"]
	_, hasB := args["blocked"]
	if hasQ || hasF || hasB {
		if list, ok := args["files"].([]any); ok && len(list) > 0 {
			expanded, problem := sh.expandProbeOutputs(list)
			if problem != "" {
				out.add("validation", problem)
				return out, nil
			}
			args = maps.Clone(args)
			args["files"] = expanded
		}
		sub, problems, err := sh.validate(args)
		if err != nil {
			return nil, err
		}
		if problems != nil {
			out.add("validation", problems...)
			if problems[0] == envelopeRule {
				return out, nil
			}
		} else {
			decisions = sub.decisions
			for _, f := range sub.files {
				merged[f.path] = f.content
			}
		}
	} else if list, present, ok := decisionsOf(args); present {
		if !ok {
			out.add("validation", decisionsMalformed)
		} else {
			decisions = list
		}
	}

	var shadow *shaping
	var shadowDir string
	defer func() {
		if shadowDir != "" {
			os.RemoveAll(shadowDir)
		}
	}()
	gate := func() (*shaping, error) {
		if shadow != nil {
			return shadow, nil
		}
		var err error
		shadow, shadowDir, err = sh.lintShadow(merged)
		return shadow, err
	}

	if len(decisions) > 0 && sh.ledger != nil {
		g, err := gate()
		if err != nil {
			return nil, err
		}
		_, o, refused, err := g.decisionRecordGate(decisions)
		if err != nil {
			return nil, err
		}
		if refused {
			out.add("decision_record", o.problems...)
		}
	}

	intentPath := project.IntentMDPath(sh.slug)
	if _, ok := merged[intentPath]; !ok && len(args) == 0 {
		out.notes = append(out.notes, "no intent.md is proposed yet: propose the package files first")
		return out, nil
	}
	if len(merged) == 0 {
		return out, nil
	}

	if _, ok := merged[intentPath]; ok {
		out.add("publication", sh.publicationProblems([]proposedFile{{intentPath, merged[intentPath]}})...)
	}
	intent, scenarios, err := project.CheckIntent(sh.root, sh.slug, overlayOf(merged))
	if err != nil {
		e, ok := sys.AsError(err)
		if !ok {
			return nil, err
		}
		lines := e.Lines
		if gaps := sh.declaredGaps(merged); len(gaps) > 0 && slices.Equal(lines, []string{"acceptance.feature does not exist"}) {
			lines = gaps
		}
		out.add("intent", lines...)
		// The gates that read files rather than the parsed Intent still run, so one call
		// returns everything it can.
		text, _ := sh.lintText(merged, intentPath)
		out.add("decision_check", sh.lintAssumptions(text)...)
		sh.lintControls(out, merged, nil)
		if literals, ok := args["literals"].([]any); ok {
			out.add("literal_check", sh.lintLiterals(literals, merged, nil)...)
		}
		return out, nil
	}
	out.add("step_ownership", sh.lintStepOwnership(intent, merged)...)
	out.add("retained", sh.retainedProblems(intent, sortedKeys(merged))...)
	out.add(scopeGate, sh.lintScope(merged, intent, scenarios)...)

	text, _ := sh.lintText(merged, intentPath)
	out.add("decision_check", sh.lintAssumptions(text)...)
	if sh.ledger != nil {
		g, err := gate()
		if err != nil {
			return nil, err
		}
		text = repairCarriedDecisionDraft(g.normalizeDecisionRounds(text), g.ledger.standing())
		_, o, refused := g.carriedDecisionsGate(text)
		if refused {
			out.add("carried_decisions", o.problems...)
		}
		problems, _, err := g.evidenceCheck(text)
		if err != nil {
			return nil, err
		}
		out.add("decision_evidence", problems...)
	}

	sh.lintControls(out, merged, intent)
	if literals, ok := args["literals"].([]any); ok {
		out.add("literal_check", sh.lintLiterals(literals, merged, intent)...)
	}
	for _, p := range intent.StepFiles() {
		body, ok := merged[p]
		if !ok || filepath.Ext(p) != ".js" {
			continue
		}
		if it, refused := stepLimitItem(verify.StepDef{Pattern: p, File: p, Body: body}); refused {
			out.add("step_selfcheck", "- "+it.Check+" "+it.File+": "+it.Detail+" (the whole file was scanned; the submit check reads each step definition's body)")
		}
	}
	return out, nil
}

// lintAssumptionFormat names every Decisions line whose assumption marker is not the canonical
// (assumption: ...) before the trailing source tag, after Kogen's own rewrite of recognisable
// variants: the refusal the decision check words as assumptionRule.
func (sh *shaping) lintAssumptions(intent string) []string {
	normalized, _ := sh.normalizedAssumptions(intent)
	locked := map[string]bool{}
	if sh.ledger != nil {
		for _, d := range sh.ledger.standing() {
			locked[decisionLineKey(d.intentLine())] = true
		}
	}
	var lines []string
	for _, line := range decisionLines(normalized) {
		if !locked[decisionLineKey(line)] && malformedAssumption(line) {
			lines = append(lines, "- "+strings.TrimSpace(line))
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return append([]string{assumptionRule}, lines...)
}

// lintStepOwnership is the ownership rule of acceptFiles for the declared step files.
func (sh *shaping) lintStepOwnership(intent *project.Intent, merged map[string]string) []string {
	stepsDir := ""
	if sh.cfg.Acceptance != nil {
		stepsDir = sh.cfg.Acceptance.StepsDir
	}
	others, _ := filepath.Glob(filepath.Join(sh.root, ".kogen/intents/*/intent.md"))
	var problems []string
	for _, p := range intent.StepFiles() {
		if !scopedStepPath(stepsDir, sh.slug, p) {
			problems = append(problems, p+": use "+stepsDir+"/"+sh.slug+"__<suffix>.js with a nonempty suffix; put helper files in "+stepsDir+"/"+sh.slug+"__<suffix>/ (for example "+stepsDir+"/"+sh.slug+"__driver/main.go), list them under Acceptance and submit their contents")
		} else if _, ok := merged[p]; !ok {
			problems = append(problems, p+": submit this Intent's own step artifact; do not adopt existing step files")
		}
		for _, op := range others {
			slug := filepath.Base(filepath.Dir(op))
			if slug == sh.slug {
				continue
			}
			if other, err := project.LoadIntent(sh.root, slug); err == nil && slices.Contains(other.StepFiles(), p) {
				problems = append(problems, fmt.Sprintf("%s belongs to Intent %s; %s must use its own step file", p, slug, sh.slug))
			}
		}
	}
	return problems
}

// lintScope is scopeRefusal on the candidate instead of the written package: the same input,
// the same scopeLines.
func (sh *shaping) lintScope(merged map[string]string, intent *project.Intent, scenarios []project.Scenario) []string {
	paths := []string{project.IntentMDPath(sh.slug)}
	paths = append(paths, intent.Acceptance...)
	if intent.Model != nil && !intent.Model.None {
		paths = append(paths, intent.Model.Path)
	}
	in := scopeInput{head: sh.headBlobs(), scenarios: len(scenarios), harnesses: len(intent.StepFiles()), ceiling: tools.InputBudgetBytes}
	in.intentText, in.parts = scopeSplit(intent.Text)
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		content, ok := sh.lintText(merged, p)
		if !ok {
			return nil
		}
		in.pkg = append(in.pkg, scopeFile{path: p, size: len(content)})
	}
	in.inputs = scopeReviewInputs(in.head)
	return scopeLines(in)
}

// lintControls checks controls.json, when the candidate or the checkout has one, against the
// schema parseControls enforces at the contract review. A missing file is only a note: the
// review requires it when the request asks for properties a step can observe.
func (sh *shaping) lintControls(out *packageLint, merged map[string]string, intent *project.Intent) {
	rel := project.IntentDir(sh.slug) + "/controls.json"
	raw, ok := sh.lintText(merged, rel)
	if !ok {
		out.notes = append(out.notes, "controls.json is not proposed: the contract review refuses with \"controls.json does not exist\" when the request asks for properties a step can observe (or write {\"deferred\":\"<reason>\"})")
		return
	}
	feature, ok := sh.lintText(merged, project.FeaturePath(sh.slug))
	if !ok {
		return
	}
	scenarios, err := project.ParseGherkin(feature)
	if err != nil {
		return
	}
	rules := controlsRules{protect: sh.cfg.Protect}
	if intent != nil {
		rules.acceptance = intent.Acceptance
	}
	if sh.cfg.Acceptance != nil {
		rules.steps = sh.cfg.Acceptance.StepsDir
	}
	_, problems := parseControls([]byte(raw), scenarios, strings.Split(feature, "\n"), rules)
	out.add("contract_review", problems...)
}

// lintLiterals compares the literals the shaper names with the candidate package, with the
// literal check's own rules (literalOutcome) applied to items whose quote is the literal itself.
func (sh *shaping) lintLiterals(list []any, merged map[string]string, intent *project.Intent) []string {
	var items []literalItem
	var literals []string
	for _, v := range list {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			literals = append(literals, s)
			items = append(items, literalItem{literal: s, source: "request", quote: s, status: "required", why: "named in check_package"})
		}
	}
	paths := []string{project.IntentMDPath(sh.slug), project.FeaturePath(sh.slug)}
	if intent != nil {
		if intent.Model != nil && !intent.Model.None && intent.Model.Path != "" {
			paths = append(paths, intent.Model.Path)
		}
		paths = append(paths, intent.Acceptance...)
	}
	var files []packageFile
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		if content, ok := sh.lintText(merged, p); ok {
			files = append(files, packageFile{p, content})
		}
	}
	// The shaper vouches that the request carries each literal; only the package is compared.
	sources := map[string]string{"request": strings.Join(literals, "\n")}
	_, lines := literalOutcome(items, sources, files, verify.NewRedactor(sh.s.Env, sh.s.Getenv("HOME")))
	return lines
}
