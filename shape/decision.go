package shape

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// SubmitDecisionCheckTool is the one tool of the decision check call.
const SubmitDecisionCheckTool = "submit_decision_check"

const decisionInstructions = `You check the stated decisions of one candidate Intent against the feature request that it answers. You see the request, earlier questions and answers, and the candidate intent.md. Consider recorded answers and assumptions as support for decisions absent from the initial request, but still report genuine conflicts with the request. Read-only: do not use any tool except ` + SubmitDecisionCheckTool + `.

Give one item for every line of the candidate's "## Decisions" section and for every explicit result the intent states for a concrete input in its prose or examples. For each item give:
- statement: text copied exactly from intent.md (for a Decisions line, the whole line);
- request_quotes: the passages of the request that support or contradict it, each copied exactly from the request (none when the request is silent);
- verdict: follows (the request, a recorded answer, or an evidenced or assumed Decision supports the value), contradicts (the request states a different value) or unsupported (none supports it);
- intent_value, request_value and difference: the two values and what makes them differ, for contradicts only;
- why: one short reason.
Compare values, not notation: before comparing, normalise digit grouping (1,000 and 1000), percentages and fractions (50% and 0.5), units of the same quantity (2 minutes and 120 seconds), date layouts, the order of a list whose order does not matter, and letter case where case does not matter.
A statement that restates the request's value in another notation follows the request.
Feasibility records are observed evidence, not product decisions requiring request quotes. Their measurements, probe results and observer-control outputs are checked by the feasibility machinery, separately from this check. The candidate view below omits that section; the full Intent remains available to the other checks.
Give contradicts only when the values still differ after that normalisation, such as a different number, a wrongly converted unit or a different order where order matters; then give intent_value (the value as the statement writes it), request_value (the value as a request quote writes it) and difference (what makes them differ). For follows and unsupported give these three as empty strings.
An explicit requirement to cover a named set conflicts with a decision that leaves out any member of that set. Copy each conflicting value verbatim. If only the whole Decision line expresses the choice, copy that line as intent_value.
A Decision line may be marked as an assumption with (assumption: <what the user must confirm>) placed before its trailing tag; a marked line keeps its trailing (source: request) or (source: answer N). A recorded assumption is acceptable when the request is silent, but does not override an explicit request conflict.
For a concrete result in prose or examples, read the corresponding Decisions before judging support. A consistent restatement of an evidenced or assumed Decision follows; it does not need a second copy of the marker on every prose line. Existing repository formats and access messages supported by those Decisions do not need to be specified again by the user. Still report explicit request conflicts, including a Decision narrowing a required set of outcomes, even when that Decision has evidence or an assumption.
For a Decision that preserves an existing repository value, use the observed repository evidence below: if it shows the stated current value, give follows without requiring a request quote. A citation name alone is not proof. Give unsupported when the observed evidence does not establish the claimed value, including a keep-unchanged claim with a different value. A new choice with neither evidence nor a recorded assumption remains unsupported. Explicit request conflicts still contradict even when the repository currently differs.
Answer by calling ` + SubmitDecisionCheckTool + `.`

// assumptionRule is the feedback line that says exactly how to mark an assumption.
const assumptionRule = "Mark an assumption with (assumption: <what the user must confirm>) before the required trailing tag, e.g. - <decision> (assumption: <what the user must confirm>) (source: request); every Decision line keeps its trailing (source: request) or (source: answer N)."

func decisionSchema() map[string]any {
	s := map[string]any{"type": "string"}
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"statement":      s,
			"request_quotes": map[string]any{"type": "array", "items": s},
			"verdict":        map[string]any{"type": "string", "enum": []any{"follows", "contradicts", "unsupported"}},
			"intent_value":   s,
			"request_value":  s,
			"difference":     s,
			"why":            s,
		},
		"required":             []any{"statement", "request_quotes", "verdict", "intent_value", "request_value", "difference", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"items": map[string]any{"type": "array", "items": item}},
		"required":             []any{"items"},
		"additionalProperties": false,
	}
}

func decisionToolDef() map[string]any {
	return map[string]any{
		"type": "function", "name": SubmitDecisionCheckTool, "strict": true,
		"description": "Submit the decision check: one item per stated decision with its request quotes, verdict and, for a contradiction, the two values.",
		"parameters":  decisionSchema(),
	}
}

func decisionInvalid(why string) error {
	return sys.Fail("shape", "decision_check_invalid", "the decision check answer is not a valid "+SubmitDecisionCheckTool+" payload ("+why+"); nothing was published", 1)
}

type decisionItem struct {
	statement    string
	quotes       []string
	verdict      string
	intentValue  string
	requestValue string
	difference   string
	why          string
}

func parseDecisionCheck(args map[string]any) ([]decisionItem, error) {
	list, ok := args["items"].([]any)
	if !ok {
		return nil, decisionInvalid("items must be a list")
	}
	items := []decisionItem{}
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, decisionInvalid("an item must be an object")
		}
		var it decisionItem
		var okS, okV, okI, okR, okD, okW bool
		it.statement, okS = m["statement"].(string)
		it.verdict, okV = m["verdict"].(string)
		it.intentValue, okI = m["intent_value"].(string)
		it.requestValue, okR = m["request_value"].(string)
		it.difference, okD = m["difference"].(string)
		it.why, okW = m["why"].(string)
		quotes, okQ := m["request_quotes"].([]any)
		if !okS || !okV || !okI || !okR || !okD || !okW || !okQ || len(m) != 7 ||
			(it.verdict != "follows" && it.verdict != "contradicts" && it.verdict != "unsupported") {
			return nil, decisionInvalid("an item is malformed")
		}
		it.quotes = []string{}
		for _, q := range quotes {
			qs, ok := q.(string)
			if !ok {
				return nil, decisionInvalid("a request quote must be a string")
			}
			it.quotes = append(it.quotes, qs)
		}
		items = append(items, it)
	}
	if len(args) != 1 {
		return nil, decisionInvalid("the payload has fields other than items")
	}
	return items, nil
}

// normalizeNotation is normalizeWhitespace followed by removing every comma that has a digit
// just before it and exactly three digits just after it, the third not followed by another
// digit. Case is kept.
func normalizeNotation(s string) string {
	s = normalizeWhitespace(s)
	isDigit := func(i int) bool { return i >= 0 && i < len(s) && s[i] >= '0' && s[i] <= '9' }
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == ',' && isDigit(i-1) && isDigit(i+1) && isDigit(i+2) && isDigit(i+3) && !isDigit(i+4) {
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// decisionOutcome applies the mechanical rules: final items (verdicts replaced where a
// statement, quote or value is not found, or where two named values are the same value), the
// event, and the feedback lines (nil when no problem).
func decisionOutcome(items []decisionItem, request, intent string) (map[string]any, []string) {
	event, lines, _ := decisionResolve(items, request, intent, nil)
	return event, lines
}

// assumptionMark is one Decisions line that Kogen rewrites itself: an unsupported line the
// request is silent on gets the assumption marker the checker's own statement fills.
type assumptionMark struct {
	line, marked, assumption string
}

// markAssumption is the line with (assumption: ...) put before its trailing source tag, or false
// when the line has no such tag or already carries an assumption.
func markAssumption(line, why string) (string, string, bool) {
	assumption := strings.NewReplacer("(", "[", ")", "]").Replace(normalizeWhitespace(why))
	loc := sourceTagRe.FindStringIndex(line)
	if assumption == "" || loc == nil || strings.Contains(line, "(assumption:") {
		return "", "", false
	}
	tag := strings.TrimSpace(line[loc[0]:])
	return strings.TrimRight(line[:loc[0]], " \t") + " (assumption: " + assumption + ") " + tag, assumption, true
}

// decisionResolve is decisionOutcome that also returns the lines Kogen rewrites into the
// assumption form instead of refusing them. A line whose decisionLineKey is locked (a recorded
// decision, kept exactly) is never rewritten.
func decisionResolve(items []decisionItem, request, intent string, locked map[string]bool) (map[string]any, []string, []assumptionMark) {
	event, lines, marks, _ := decisionResolveNotes(items, request, intent, locked)
	return event, lines, marks
}

// decisionResolveNotes is decisionResolve that also returns the advisory notes: an unsupported
// statement that carries (evidence: ...) is recorded and passed to the shaper, not refused.
// Only a contradiction, or an unsupported statement with neither evidence nor an assumption,
// refuses.
func decisionResolveNotes(items []decisionItem, request, intent string, locked map[string]bool) (map[string]any, []string, []assumptionMark, []string) {
	var marks []assumptionMark
	var notes []string
	counts := map[string]int{}
	events := make([]any, len(items))
	var problems []string
	refused, conflicts, unsupported, malformed := 0, 0, false, false
	for i := range items {
		it := &items[i]
		reported := it.statement
		if !containsNormalized(intent, it.statement) {
			if line := referencedDecision(intent, it.statement); line != "" {
				it.statement = line
			}
		}
		supported := containsNormalized(intent, it.statement)
		for _, q := range it.quotes {
			supported = supported && containsNormalized(request, q)
		}
		if !supported {
			it.verdict = "unsupported"
		}
		modelVerdict := it.verdict
		if it.verdict == "unsupported" && requestStates(request, it.statement, it.quotes) && containsNormalized(intent, it.statement) {
			// The request says the statement literally: it is supported whatever the model judged.
			it.verdict = "follows"
		}
		paraphrasedConflict := false
		if it.verdict == "contradicts" {
			draftValue, requestValue := normalizeNotation(it.intentValue), normalizeNotation(it.requestValue)
			draftNamed := draftValue != "" && containsNormalized(it.statement, it.intentValue)
			requestNamed := false
			for _, q := range it.quotes {
				requestNamed = requestNamed || containsNormalized(q, it.requestValue)
			}
			requestNamed = requestValue != "" && requestNamed
			switch {
			case !draftNamed || !requestNamed:
				it.verdict = "unsupported"
				// The whole statement and the requirement are grounded. A paraphrase
				// weakens the finding's evidence, but cannot turn it into user permission.
				paraphrasedConflict = !draftNamed && requestNamed && draftValue != "" && draftValue != requestValue
			case draftValue == requestValue:
				it.verdict = "follows"
			}
		}
		counts[it.verdict]++
		events[i] = map[string]any{"statement": it.statement, "request_quotes": it.quotes, "verdict": it.verdict,
			"intent_value": it.intentValue, "request_value": it.requestValue, "difference": it.difference, "why": it.why}
		if modelVerdict != it.verdict {
			events[i].(map[string]any)["advisory"] = true
			events[i].(map[string]any)["model_verdict"] = modelVerdict
		}
		if reported != it.statement {
			events[i].(map[string]any)["reported_statement"] = reported
		}
		marked := it.verdict == "unsupported" && markedStatement(intent, it.statement)
		if it.verdict == "follows" || (marked && !paraphrasedConflict) {
			continue
		}
		if it.verdict == "unsupported" && !paraphrasedConflict && !marked && !requestSpeaks(request, it.quotes) {
			if mark, ok := decisionMark(intent, it, locked); ok {
				marks = append(marks, mark)
				events[i].(map[string]any)["marked_assumption"] = mark.assumption
				continue
			}
		}
		if it.verdict == "unsupported" && supported && !paraphrasedConflict && evidenceRe.MatchString(it.statement) {
			events[i].(map[string]any)["advisory"] = true
			notes = append(notes, "Decision check (advisory, not refused): "+strings.TrimSpace(it.statement)+" | "+it.why)
			continue
		}
		uncertifiedConflict := marked && paraphrasedConflict
		if uncertifiedConflict {
			events[i].(map[string]any)["reported_verdict"] = "contradicts"
		}
		refused++
		malformed = malformed || malformedAssumption(statementLine(intent, it.statement))
		quoted := make([]string, len(it.quotes))
		for j, q := range it.quotes {
			quoted[j] = `"` + q + `"`
		}
		if it.verdict == "contradicts" || uncertifiedConflict {
			conflicts++
			problems = append(problems, fmt.Sprintf("statement: %s | request_quotes: %s | verdict: %s | intent_value: %s | request_value: %s | difference: %s | why: %s",
				it.statement, strings.Join(quoted, ", "), it.verdict, it.intentValue, it.requestValue, it.difference, it.why))
			where := "the quoted statement"
			if at := strings.Index(intent, it.statement); at >= 0 {
				where = fmt.Sprintf("intent.md line %d", 1+strings.Count(intent[:at], "\n"))
			}
			if uncertifiedConflict {
				problems = append(problems, fmt.Sprintf("fix for %s: Preserve the request's requirement %q in the quoted line and its acceptance expectations. If the draft already follows the requirement, cite the exact matching text. The check reported a conflict but paraphrased the draft value.", where, it.requestValue))
			} else {
				problems = append(problems, fmt.Sprintf("fix for %s: Replace %q with the request's requirement %q. Align the acceptance expectations with that requirement.", where, it.intentValue, it.requestValue))
			}
			continue
		}
		grounded := false
		for _, q := range it.quotes {
			if strings.TrimSpace(q) != "" && containsNormalized(request, q) {
				grounded = true
			}
		}
		problems = append(problems, fmt.Sprintf("statement: %s | request_quotes: %s | verdict: unsupported | why: %s", it.statement, strings.Join(quoted, ", "), it.why))
		if grounded {
			where := "the quoted statement"
			if at := strings.Index(intent, it.statement); at >= 0 {
				where = fmt.Sprintf("intent.md line %d", 1+strings.Count(intent[:at], "\n"))
			}
			problems = append(problems, fmt.Sprintf("fix for %s: the request speaks to this in the quoted passages; make the statement follow them or remove it; an assumption does not override request text.", where))
		} else {
			unsupported = true
		}
	}
	event := map[string]any{"event": "decision_check", "follows": counts["follows"], "contradicts": counts["contradicts"], "unsupported": counts["unsupported"], "items": events}
	if len(problems) == 0 {
		return event, nil, marks, notes
	}
	head := fmt.Sprintf("Decision check: %d stated decisions are not supported by the request; follow the request, cite the repository evidence that decides it with (evidence: ...), or mark the decision as an assumption the user must confirm:", refused)
	if !unsupported {
		head = fmt.Sprintf("Decision check: %d stated decisions are not supported by the request; follow the quoted request passages:", refused)
	}
	if conflicts > 0 {
		head = fmt.Sprintf("Decision check: %d stated decisions are not supported by the request; resolve each explicit conflict using the requirement below:", refused)
	}
	feedback := []string{head}
	if malformed {
		feedback = append(feedback, assumptionRule)
	}
	return event, append(feedback, problems...), marks, notes
}

// requestSpeaks is whether one of the quotes is a passage of the request.
func requestSpeaks(request string, quotes []string) bool {
	for _, q := range quotes {
		if strings.TrimSpace(q) != "" && containsNormalized(request, q) {
			return true
		}
	}
	return false
}

// decisionMark finds the single Decisions line the item's statement is and its marked form.
func decisionMark(intent string, it *decisionItem, locked map[string]bool) (assumptionMark, bool) {
	var found string
	for _, line := range decisionLines(intent) {
		if normalizeWhitespace(line) == normalizeWhitespace(it.statement) {
			if found != "" && found != line {
				return assumptionMark{}, false
			}
			found = line
		}
	}
	if found == "" || locked[decisionLineKey(found)] {
		return assumptionMark{}, false
	}
	marked, assumption, ok := markAssumption(found, it.why)
	return assumptionMark{line: found, marked: marked, assumption: assumption}, ok
}

// A finding can paraphrase a Decision despite the exact-copy
// instruction. Its unique ledger identity and complete annotation suffix still
// identify the choice. Ground the finding in the actual line before checking
// values, quotes or markers; retain the reported text separately in the event.
func referencedDecision(intent, statement string) string {
	refs := decisionRoundRe.FindAllStringIndex(statement, -1)
	if len(refs) != 1 || !strings.HasPrefix(statement, "- ") {
		return ""
	}
	suffix := statement[refs[0][0]:]
	var found string
	for _, line := range decisionLines(intent) {
		if strings.Contains(line, statement[refs[0][0]:refs[0][1]]) {
			if found != "" {
				return ""
			}
			found = line
		}
	}
	if found != "" && strings.HasSuffix(found, suffix) {
		return found
	}
	// The checker can render an omitted constraint as "(constraint: none)".
	// Accept this placeholder only when the rest of the reported line is exact.
	end := refs[0][1]
	if tail, ok := strings.CutPrefix(statement[end:], " (constraint: none)"); ok && found == statement[:end]+tail {
		return found
	}
	return ""
}

// decisionCheck makes one fresh read-only call on the candidate intent.md and returns the
// feedback lines, nil when the candidate may go on.
func (sh *shaping) decisionCheck(ctx context.Context, intent string) ([]string, error) {
	evidence, err := sh.decisionEvidence(intent)
	if err != nil {
		return nil, err
	}
	return sh.decisionCheckWithEvidence(ctx, intent, evidence)
}

func (sh *shaping) decisionCheckWithEvidence(ctx context.Context, intent, evidence string) ([]string, error) {
	intent, sh.pendingMarks = sh.normalizedAssumptions(intent)
	earlier := sh.earlier
	if earlier == "" {
		earlier = "none"
	}
	input := "Slug: " + sh.slug + "\n\n# Request\n\n" + sh.request + "\n\n# Earlier questions and answers\n\n" + earlier + "\n\n# Candidate intent.md\n\n" + decisionIntent(intent) + evidence
	var args map[string]any
	sh.decisionChecks++
	if sh.ext != nil {
		x, err := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"),
			fmt.Sprintf("%s/executor-decision-%d", sh.ev.Dir, sh.decisionChecks), sh.slug, sh.role, decisionInstructions+"\n\n"+input, "", mustJSON(decisionSchema()), sh.ev, nil)
		if err != nil {
			return nil, err
		}
		defer x.close()
		res, err := x.round(ctx, "")
		sh.decisionRounds += x.rounds
		if err != nil {
			return nil, err
		}
		if res.Ending != executor.EndFinish {
			return nil, decisionInvalid("the session ended without a result")
		}
		if args, err = submissionArgs(res.Structured); err != nil {
			return nil, decisionInvalid("not a JSON object")
		}
	} else {
		sh.progress("kogen: checking the stated decisions")
		revision, err := CurrentRevision(sh.repo)
		if err != nil {
			return nil, err
		}
		cacheID := startConversation(sh.ev, "decision-check")
		req := conversationRequest(provider.Request{
			Role: ProviderRole(sh.role), Instructions: decisionInstructions,
			Input: []any{map[string]any{"role": "user", "content": input}}, Tools: []any{decisionToolDef()},
			Scope: sh.ev.Dir, Context: map[string]any{"phase": "shape", "role": "shape", "slug": sh.slug},
		}, cacheID)
		call := sh.ev.BeginCall("shape", "", revision.Map(), "decision check", sh.role).Sized(sh.client, req)
		t0 := time.Now()
		resp, err := sh.client.Respond(ctx, sh.s, req, call)
		call.End(resp, err)
		if err != nil {
			if e, ok := sys.AsError(err); ok && e.Class == "provider" {
				return nil, sh.providerFailed(e)
			}
			return nil, err
		}
		sh.progress(fmt.Sprintf("kogen: decision check response %s wall_ms=%d", resp.Usage.Line(), time.Since(t0).Milliseconds()))
		sh.decisionUsage = sh.decisionUsage.Add(resp.Usage)
		calls := tools.Calls(resp.Items)
		if len(calls) != 1 || calls[0].Name != SubmitDecisionCheckTool {
			return nil, decisionInvalid("expected one " + SubmitDecisionCheckTool + " call")
		}
		args = calls[0].Args()
	}
	items, err := parseDecisionCheck(args)
	if err != nil {
		return nil, err
	}
	locked := map[string]bool{}
	if sh.ledger != nil {
		for _, d := range sh.ledger.standing() {
			locked[decisionLineKey(d.intentLine())] = true
		}
	}
	event, lines, marks, notes := decisionResolveNotes(items, sh.request, intent, locked)
	sh.ev.Record(event)
	sh.pendingMarks = append(sh.pendingMarks, marks...)
	sh.notes = append(sh.notes, notes...)
	return lines, nil
}

// markedStatement reports whether every occurrence of the statement in intent.md (compared
// after whitespace normalisation) has an assumption or settled-answer marker on every intent.md line
// it spans: the decision check does not count such a statement as a problem. A statement that
// does not occur in intent.md is not marked.
func markedStatement(intent, statement string) bool {
	needle := normalizeWhitespace(statement)
	if needle == "" {
		return false
	}
	lines := strings.Split(intent, "\n")
	marked := make([]bool, len(lines))
	var text []byte
	var lineOf []int
	pendingSpace := false
	for i, line := range lines {
		marked[i] = strings.Contains(line, "(assumption:") || strings.Contains(line, "(source: answer ")
		for j := 0; j < len(line); j++ {
			if c := line[j]; c == ' ' || c == '\t' || c == '\r' {
				pendingSpace = true
				continue
			}
			if pendingSpace && len(text) > 0 {
				text = append(text, ' ')
				lineOf = append(lineOf, i)
			}
			pendingSpace = false
			text = append(text, line[j])
			lineOf = append(lineOf, i)
		}
		pendingSpace = true
	}
	found := false
	hay := string(text)
	for from := 0; ; {
		k := strings.Index(hay[from:], needle)
		if k < 0 {
			return found
		}
		found = true
		start, end := from+k, from+k+len(needle)-1
		for l := lineOf[start]; l <= lineOf[end]; l++ {
			if !marked[l] {
				return false
			}
		}
		from = start + 1
	}
}

// decisionIntent projects only this check's scope. Feasibility remains in the
// published package and is still read by readiness and the contract reviewer.
// Respect fenced examples: a heading inside one is contract text, not a section.
func decisionIntent(intent string) string {
	var out []string
	evidence := false
	var fence byte
	width := 0
	for _, line := range strings.Split(intent, "\n") {
		text := strings.TrimSpace(line)
		if fence == 0 && strings.HasPrefix(text, "## ") {
			evidence = strings.TrimSpace(strings.TrimPrefix(text, "## ")) == "Feasibility"
			if evidence {
				out = append(out, "## Feasibility", "[Evidence checked separately.]")
			}
		}
		if !evidence {
			out = append(out, line)
		}
		if len(text) >= 3 && (text[0] == '`' || text[0] == '~') {
			n := 0
			for n < len(text) && text[n] == text[0] {
				n++
			}
			if fence == 0 && n >= 3 {
				fence, width = text[0], n
			} else if fence == text[0] && n >= width && strings.TrimSpace(text[n:]) == "" {
				fence, width = 0, 0
			}
		}
	}
	return strings.Join(out, "\n")
}

var evidenceRe = regexp.MustCompile(`\(evidence: [^)]*\S[^)]*\)`)

var (
	assumptionVariantRe = regexp.MustCompile(`(?i)[(\[](?:assumption|assumed|assuming)\s*[:\-\x{2013}\x{2014}]\s*([^()\[\]]*\S)\s*[)\]]`)
)

// normalizeAssumptionForm rewrites a recognisable assumption stated in a wrong place or wording
// ([assumption: x], (Assumed: x), (assumption - x), or after the source tag) into
// (assumption: x) before the trailing source tag. It reports false when the line is already
// canonical, states no assumption, or states more than one.
func normalizeAssumptionForm(line string) (string, bool) {
	matches := assumptionVariantRe.FindAllStringSubmatchIndex(line, -1)
	if len(matches) != 1 {
		return "", false
	}
	m := matches[0]
	text := strings.TrimSpace(line[m[2]:m[3]])
	canonical := "(assumption: " + text + ")"
	body := strings.TrimRight(line[:m[0]]+line[m[1]:], " \t\r")
	loc := sourceTagRe.FindStringIndex(body)
	if loc == nil {
		return "", false
	}
	tag := strings.TrimSpace(body[loc[0]:])
	out := strings.TrimRight(body[:loc[0]], " \t") + " " + canonical + " " + tag
	if out == line {
		return "", false
	}
	return out, true
}

// normalizedAssumptions is the intent with every Decisions line's assumption put in its
// canonical form, and the rewrites to apply to intent.md. A recorded decision's line is kept.
func (sh *shaping) normalizedAssumptions(intent string) (string, []assumptionMark) {
	locked := map[string]bool{}
	if sh.ledger != nil {
		for _, d := range sh.ledger.standing() {
			locked[decisionLineKey(d.intentLine())] = true
		}
	}
	var marks []assumptionMark
	for _, line := range decisionLines(intent) {
		if locked[decisionLineKey(line)] {
			continue
		}
		if fixed, ok := normalizeAssumptionForm(line); ok && strings.Contains(intent, line) {
			intent = strings.Replace(intent, line, fixed, 1)
			marks = append(marks, assumptionMark{line: line, marked: fixed, assumption: assumptionVariantRe.FindStringSubmatch(line)[1]})
		}
	}
	return intent, marks
}

var (
	assumptionMarkerRe  = regexp.MustCompile(`(?i)[(\[]\s*assum`)
	canonicalAssumption = regexp.MustCompile(`\(assumption: [^()]*\S[^()]*\) \(source: (?:request|answer \d+)\)\s*$`)
	quoteNotation       = strings.NewReplacer("`", "", `"`, "", "'", "", "\u201c", "", "\u201d", "", "\u2018", "", "\u2019", "")
)

// malformedAssumption is whether the line carries an assumption marker that is not the
// canonical (assumption: ...) before the trailing source tag.
func malformedAssumption(line string) bool {
	return assumptionMarkerRe.MatchString(line) && !canonicalAssumption.MatchString(line)
}

// statementLine is the intent.md line the statement occurs in, or the statement itself.
func statementLine(intent, statement string) string {
	for _, line := range strings.Split(intent, "\n") {
		if containsNormalized(line, statement) {
			return line
		}
	}
	return statement
}

// literalForm is the text compared by the literal check: leading list dash and trailing source
// tag removed, whitespace normalised, quotes and backticks dropped.
func literalForm(s string) string {
	s = sourceTagRe.ReplaceAllString(strings.TrimSpace(s), "")
	s = strings.TrimPrefix(strings.TrimSpace(s), "- ")
	return normalizeWhitespace(quoteNotation.Replace(s))
}

// requestStates is whether the request says the statement literally: its normalised text
// occurs in the request or equals one of its own request quotes.
func requestStates(request, statement string, quotes []string) bool {
	text := literalForm(statement)
	if text == "" {
		return false
	}
	if strings.Contains(literalForm(request), text) {
		return true
	}
	for _, q := range quotes {
		if literalForm(q) == text && containsNormalized(request, q) {
			return true
		}
	}
	return false
}
