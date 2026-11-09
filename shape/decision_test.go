package shape

import (
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
)

const decisionHead = "Decision check: 1 stated decisions are not supported by the request; follow the request, cite the repository evidence that decides it with (evidence: ...), or mark the decision as an assumption the user must confirm:"

func TestDecisionOutcomeRules(t *testing.T) {
	parallel(t)
	const request = "Add f. f(0) returns True. Limit uploads to 1,000 rows."
	const decision = "- f(0) returns False. (source: request)"
	const limit = "- Upload limit is 10,000 rows. (source: request)"
	const grouped = "- Upload limit is 1000 rows. (source: request)"
	const assumed = "- Zero is even. (assumption: silent) (source: request)"
	const evidenced = "- Zero is even. (evidence: calc.js) (source: request)"
	intent := "## Decisions\n" + decision + "\n" + limit + "\n" + grouped + "\n" + assumed + "\n" + evidenced + "\n"
	const q = "f(0) returns True."
	const lq = "Limit uploads to 1,000 rows."
	item := func(statement string, quotes []string, verdict string) decisionItem {
		return decisionItem{statement: statement, quotes: quotes, verdict: verdict, why: "w"}
	}
	values := func(it decisionItem, intentValue, requestValue string) decisionItem {
		it.intentValue, it.requestValue, it.difference = intentValue, requestValue, "d"
		return it
	}
	contradicts := values(item(decision, []string{q}, "contradicts"), "False", "True")
	cases := []struct {
		name     string
		item     decisionItem
		verdict  string
		feedback []string
	}{
		{"follows", item(decision, []string{q}, "follows"), "follows", nil},
		{"contradicts with both values", contradicts, "contradicts",
			[]string{`statement: ` + decision + ` | request_quotes: "` + q + `" | verdict: contradicts | intent_value: False | request_value: True | difference: d | why: w`}},
		{"different grouped number", values(item(limit, []string{lq}, "contradicts"), "10,000", "1,000"), "contradicts",
			[]string{`statement: ` + limit + ` | request_quotes: "` + lq + `" | verdict: contradicts | intent_value: 10,000 | request_value: 1,000 | difference: d | why: w`}},
		{"digit grouping only", values(item(grouped, []string{lq}, "contradicts"), "1000", "1,000"), "follows", nil},
		{"blank intent value", values(item(decision, []string{q}, "contradicts"), " ", "True"), "unsupported",
			[]string{`statement: ` + decision + ` | request_quotes: "` + q + `" | verdict: unsupported | why: w`}},
		{"blank request value", values(item(decision, []string{q}, "contradicts"), "False", ""), "unsupported",
			[]string{`statement: ` + decision + ` | request_quotes: "` + q + `" | verdict: unsupported | why: w`}},
		{"intent value not in statement", values(item(decision, []string{q}, "contradicts"), "Maybe", "True"), "unsupported",
			[]string{`statement: ` + decision + ` | request_quotes: "` + q + `" | verdict: unsupported | why: w`}},
		{"request value not in quotes", values(item(decision, []string{q}, "contradicts"), "False", "Maybe"), "unsupported",
			[]string{`statement: ` + decision + ` | request_quotes: "` + q + `" | verdict: unsupported | why: w`}},
		{"request value in the statement only", values(item(decision, []string{q}, "contradicts"), "False", "False"), "unsupported",
			[]string{`statement: ` + decision + ` | request_quotes: "` + q + `" | verdict: unsupported | why: w`}},
		{"unsupported without quotes", item(decision, nil, "unsupported"),
			"unsupported", []string{`statement: ` + decision + ` | request_quotes:  | verdict: unsupported | why: w`}},
		{"statement not in intent", values(item("- other", []string{q}, "contradicts"), "other", "True"),
			"unsupported", []string{`statement: - other | request_quotes: "` + q + `" | verdict: unsupported | why: w`}},
		{"quote not in request", item(decision, []string{q, "nope"}, "follows"),
			"unsupported", []string{`statement: ` + decision + ` | request_quotes: "` + q + `", "nope" | verdict: unsupported | why: w`}},
		{"assumption not fed back", item(assumed, nil, "unsupported"), "unsupported", nil},
		{"unsupported evidence is fed back", item(evidenced, nil, "unsupported"), "unsupported", []string{`statement: ` + evidenced + ` | request_quotes:  | verdict: unsupported | why: w`}},
		{"assumption contradicting not fed back", values(item(assumed, []string{"nope"}, "contradicts"), "a", "b"), "unsupported", nil},
	}
	for _, c := range cases {
		event, lines := decisionOutcome([]decisionItem{c.item}, request, intent)
		if event["event"] != "decision_check" || event[c.verdict] != 1 {
			t.Errorf("%s: event %v", c.name, event)
		}
		items := event["items"].([]any)
		if items[0].(map[string]any)["verdict"] != c.verdict {
			t.Errorf("%s: verdict %v", c.name, items[0])
		}
		if c.feedback == nil {
			if lines != nil {
				t.Errorf("%s: feedback %q", c.name, lines)
			}
			continue
		}
		want := append([]string{decisionHead, assumptionRule}, c.feedback...)
		if c.verdict == "contradicts" {
			want = []string{
				"Decision check: 1 stated decisions are not supported by the request; resolve each explicit conflict using the requirement below:",
				c.feedback[0],
			}
			if len(lines) != 3 || !strings.HasPrefix(lines[2], "fix for intent.md line ") {
				t.Errorf("%s: missing source-specific fix in %q", c.name, lines)
				continue
			}
			lines = lines[:2]
		}
		if !slices.Equal(lines, want) {
			t.Errorf("%s: lines %q", c.name, lines)
		}
	}
	event, _ := decisionOutcome([]decisionItem{contradicts}, request, intent)
	rec := event["items"].([]any)[0].(map[string]any)
	if rec["intent_value"] != "False" || rec["request_value"] != "True" || rec["difference"] != "d" || rec["statement"] != decision {
		t.Errorf("event item %v", rec)
	}
}

func TestNormalizeNotation(t *testing.T) {
	parallel(t)
	cases := []struct {
		a, b string
		same bool
	}{
		{"1,000", "1000", true},
		{"1,000,000", "1000000", true},
		{"10,000", "1,000", false},
		{"1,5", "15", false},
		{"1,0000", "10000", false},
		{"Yes", "yes", false},
		{"a  b", "a b", true},
		{"rows, 000", "rows000", false},
	}
	for _, c := range cases {
		if got := normalizeNotation(c.a) == normalizeNotation(c.b); got != c.same {
			t.Errorf("normalizeNotation(%q) == normalizeNotation(%q) is %v, want %v", c.a, c.b, got, c.same)
		}
	}
}

func TestParseDecisionCheck(t *testing.T) {
	parallel(t)
	good := func() map[string]any {
		return map[string]any{"statement": "s", "request_quotes": []any{"q"}, "verdict": "follows", "intent_value": "", "request_value": "", "difference": "", "why": "w"}
	}
	items, err := parseDecisionCheck(map[string]any{"items": []any{good()}})
	if err != nil || len(items) != 1 || items[0].statement != "s" || items[0].quotes[0] != "q" {
		t.Fatalf("good payload: %v %v", items, err)
	}
	without := func(k string) map[string]any { m := good(); delete(m, k); return m }
	with := func(k string, v any) map[string]any { m := good(); m[k] = v; return m }
	for name, args := range map[string]map[string]any{
		"no items":           {},
		"items not a list":   {"items": "x"},
		"item not object":    {"items": []any{"x"}},
		"missing value":      {"items": []any{without("intent_value")}},
		"missing difference": {"items": []any{without("difference")}},
		"missing why":        {"items": []any{without("why")}},
		"mistyped value":     {"items": []any{with("request_value", 1.0)}},
		"unknown verdict":    {"items": []any{with("verdict", "maybe")}},
		"quote not string":   {"items": []any{with("request_quotes", []any{1.0})}},
		"extra field":        {"items": []any{with("extra", "x")}},
	} {
		_, err := parseDecisionCheck(args)
		e, ok := err.(interface{ Error() string })
		if !ok || err == nil || !strings.Contains(e.Error(), "the decision check answer is not a valid submit_decision_check payload (") {
			t.Errorf("%s: err %v", name, err)
		}
	}
}

func TestDecisionInstructionsKeepSourceTag(t *testing.T) {
	parallel(t)
	if !strings.Contains(decisionInstructions, "keeps its trailing (source: request) or (source: answer N)") {
		t.Error("instructions do not state the rule")
	}
	lines := strings.Split(decisionInstructions, "\n")
	for _, want := range []string{
		"Compare values, not notation: before comparing, normalise digit grouping (1,000 and 1000), percentages and fractions (50% and 0.5), units of the same quantity (2 minutes and 120 seconds), date layouts, the order of a list whose order does not matter, and letter case where case does not matter.",
		"A statement that restates the request's value in another notation follows the request.",
		"Give contradicts only when the values still differ after that normalisation, such as a different number, a wrongly converted unit or a different order where order matters; then give intent_value (the value as the statement writes it), request_value (the value as a request quote writes it) and difference (what makes them differ). For follows and unsupported give these three as empty strings.",
	} {
		if !slices.Contains(lines, want) {
			t.Errorf("instructions lack the line %q", want)
		}
	}
}

// TestExternalShapeDecisionCheck: an external shaper runs the check as one fresh session per
// checked submission with the check schema, applies the mechanical rules and feeds problems back.
func TestExternalShapeDecisionCheck(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	const decision = "- Non-integers raise TypeError. (source: request)"
	check := func(verdict, intentValue, requestValue string) func(int, string) (executor.RoundResult, error) {
		return result(map[string]any{"items": []any{map[string]any{"statement": decision, "request_quotes": []any{"Non-integers raise TypeError."},
			"verdict": verdict, "intent_value": intentValue, "request_value": requestValue, "difference": "they differ", "why": "because"}}})
	}
	ack := map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}
	x := &shapeExec{t: t,
		rounds:   []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(shpFiles()), result(shpFiles()), result(ack)},
		decision: []func(int, string) (executor.RoundResult, error){check("contradicts", "", ""), check("contradicts", "TypeError", "raise TypeError"), check("follows", "", ""), check("contradicts", "TypeError", "raise TypeError")}}
	if exit, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError."); err != nil || exit != 0 {
		t.Fatalf("acknowledgment reopened a reviewed decision: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	if len(x.decisionStarts) != 3 {
		t.Fatalf("%d check sessions of %d", len(x.decisionStarts), len(x.starts))
	}
	for i, s := range x.decisionStarts {
		shpContains(t, "task", s.Brief.Stable, "# Candidate intent.md")
		shpContains(t, "task", s.Brief.Stable, "# Request\n\nAdd calc.is_even(n). Non-integers raise TypeError.\n\n")
		shpContains(t, "dir", s.Dir, "executor-decision-"+string(rune('1'+i)))
		if strings.Contains(s.Brief.Stable, "# Calc") {
			t.Error("the check saw repository context")
		}
	}
	var feedback string
	for _, in := range x.inputs {
		feedback += in.Feedback
	}
	shpContains(t, "feedback", feedback, "Decision check: 1 stated decisions are not supported by the request;")
	shpContains(t, "feedback line", feedback, "statement: "+decision+` | request_quotes: "Non-integers raise TypeError." | verdict: contradicts | intent_value: TypeError | request_value: raise TypeError | difference: they differ | why: because`)
	shpContains(t, "unnamed", feedback, "verdict: unsupported | why: because")
	if n := len(shpNamed(f.events(), "decision_check")); n != 3 {
		t.Fatalf("%d decision_check events", n)
	}
	if n := len(f.Prov.allRequests()); n != 0 {
		t.Fatalf("%d native provider requests", n)
	}
}

// TestNativeShapeDecisionCheckInvalidAnswer: a native answer that is not exactly one valid
// submit_decision_check call fails Shape and publishes nothing.
func TestNativeShapeDecisionCheckInvalidAnswer(t *testing.T) {
	parallel(t)
	for name, answer := range map[string]shpStep{
		"a malformed item": shpAnswer(shpCall("d1", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": "s"}}})),
		"another tool":     shpAnswer(shpCall("d1", "read_file", map[string]any{"path": "app.txt"})),
	} {
		t.Run(name, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t, shpSubmit("c1", shpFiles()))
			f.Prov.route = func(req map[string]any) *shpStep {
				if decisionRequest(req) {
					return &answer
				}
				return nil
			}
			exit, err := f.shape("Add calc.is_even(n).")
			e, ok := sys.AsError(err)
			if exit != 1 || !ok || e.Reason != "decision_check_invalid" || e.Class != "shape" {
				t.Fatalf("exit %d, err %v", exit, err)
			}
			shpContains(t, "detail", e.Error(), "the decision check answer is not a valid submit_decision_check payload (")
			shpContains(t, "detail", e.Error(), "); nothing was published")
			if f.exists(".kogen/local/provenance/is-even/current.json") {
				t.Fatal("the Intent was published")
			}
			if n := len(f.Prov.decisionRequests()); n != 1 {
				t.Fatalf("%d decision check requests", n)
			}
		})
	}
}

func TestMarkedStatementOnEveryOccupiedLine(t *testing.T) {
	parallel(t)
	const decision = "- Odd numbers return False (source: request)"
	for _, c := range []struct {
		name, intent, statement string
		want                    bool
	}{
		{"marker elsewhere on the line", "## Decisions\n- Odd numbers return False (evidence: README.md) (source: request)\n", "Odd numbers return False", false},
		{"assumption after the text", "## Decisions\n- Zero is even (assumption: silent) (source: request)\n", "Zero is even", true},
		{"marker inside the statement text", "## Decisions\n- Zero is even (assumption: silent) (source: request)\n", "- Zero is even (assumption: silent) (source: request)", true},
		{"marker on the next line", "## Decisions\n" + decision + "\n- Zero (assumption: silent) (source: request)\n", "Odd numbers return False", false},
		{"two lines, one marker", "x (evidence: README.md)\nand zero is even.\n", "x (evidence: README.md) and zero is even.", false},
		{"two lines, both marked", "x (evidence: README.md)\nand zero (assumption: a) is even.\n", "x (evidence: README.md) and zero (assumption: a) is even.", false},
		{"statement not in intent", "## Decisions\n- Zero (assumption: silent) (source: request)\n", "- Zero is even (assumption: silent) (source: request)", false},
		{"one unmarked occurrence of two", "a b (evidence: x)\na b\n", "a b", false},
		{"unmarked", "## Decisions\n" + decision + "\n", decision, false},
	} {
		if got := markedStatement(c.intent, c.statement); got != c.want {
			t.Errorf("%s: marked = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDecisionOutcomeGroundsInventedEmptyConstraint(t *testing.T) {
	parallel(t)
	const decision = "- spec/check.sh is not re-run inside the acceptance scenarios; the configured spec check runs it in full and the text must keep spec/tools/check.mjs passing. (decision: D5, round 1) (evidence: spec/check.sh; spec/tools/check.mjs; .kogen/project.yaml) (assumption: I ran only spec/tools/check.mjs on a scratch copy with the intended edits, which passed (requirements 108, scenarios 210); I did not run the whole check.sh.) (source: request)"
	const ref = "(decision: D5, round 1)"
	const request = "(4) spec/check.sh passes in full."
	reported := strings.Replace(decision, ref, ref+" (constraint: none)", 1)
	intent := "## Decisions\n" + decision + "\n"
	item := decisionItem{statement: reported, quotes: []string{request}, verdict: "unsupported",
		why: "The request requires spec/check.sh to pass in full; the decision delegates that to the configured spec check, which is compatible."}

	event, feedback := decisionOutcome([]decisionItem{item}, request, intent)
	if len(feedback) != 0 {
		t.Fatalf("marked source decision was refused: %q", feedback)
	}
	got := event["items"].([]any)[0].(map[string]any)
	if got["statement"] != decision || got["reported_statement"] != reported {
		t.Fatalf("source or reported statement was lost: %v", got)
	}

	t.Run("real conflict still refuses", func(t *testing.T) {
		const requirement = "spec/check.sh must run inside the acceptance scenarios."
		conflict := item
		conflict.quotes = []string{requirement}
		conflict.verdict = "contradicts"
		conflict.intentValue = "not re-run"
		conflict.requestValue = "must run"
		conflict.difference = "different acceptance coverage"
		_, lines := decisionOutcome([]decisionItem{conflict}, requirement, intent)
		shpContains(t, "conflict", strings.Join(lines, "\n"), "verdict: contradicts")
	})

	t.Run("unsupported product decision still refuses", func(t *testing.T) {
		const unmarked = "- Allow a five-digit short commit. (decision: D5, round 1) (source: request)"
		unsupported := item
		unsupported.statement = strings.Replace(unmarked, ref, ref+" (constraint: none)", 1)
		_, lines := decisionOutcome([]decisionItem{unsupported}, request, "## Decisions\n"+unmarked+"\n")
		shpContains(t, "unsupported decision", strings.Join(lines, "\n"), "verdict: unsupported")
	})

	for _, changed := range []string{
		strings.Replace(reported, "constraint: none", "constraint: always skip verification", 1),
		strings.Replace(reported, "not re-run", "never run", 1),
	} {
		unsupported := item
		unsupported.statement = changed
		_, lines := decisionOutcome([]decisionItem{unsupported}, request, intent)
		shpContains(t, "changed finding", strings.Join(lines, "\n"), "verdict: unsupported")
	}
}

func TestDecisionOutcomeRepairsExplicitRequestConflict(t *testing.T) {
	parallel(t)
	const request = "Every not_ready refusal detail names the exact failing line or item and a fix text that covers that same line."
	const wanted = "a fix text that covers that same line"
	const conflicting = "gets no fix line"
	const statement = "- An item in the Decisions section gets no fix line. (source: request)"
	const intent = "# Refusal details\n\n## Decisions\n" + statement + "\n"
	item := decisionItem{
		statement: statement, quotes: []string{request}, verdict: "contradicts",
		intentValue: conflicting, requestValue: wanted,
		difference: "the request says every item gets a fix line", why: "the decision introduces an exception",
	}
	_, lines := decisionOutcome([]decisionItem{item}, request, intent)
	feedback := strings.Join(lines, "\n")
	if !strings.Contains(feedback, "intent.md line 4") ||
		!strings.Contains(feedback, `Replace "gets no fix line" with the request's requirement "a fix text that covers that same line"`) ||
		strings.Contains(feedback, "mark the decision as an assumption") || strings.Contains(feedback, assumptionRule) {
		t.Fatalf("contradiction does not tell Shape how to repair the actual line: %s", feedback)
	}
}

func TestParaphrasedValueCannotClearAGroundedConflict(t *testing.T) {
	parallel(t)
	const want = "every not_ready refusal detail names the exact failing line or item (text and location) and a fix text that covers that same line"
	const request = "Shape refusals (decision_check, contract_review, carried_decisions, decision_record) give only a generic rule. Expected: " + want + "."
	const line = "- contract_review keeps its refusal lines: each already names the scenario, both file paths and both verbatim quotes, and the request's acceptance checks name the other three gates only. (evidence: shape/contract.go) (assumption: the request lists contract_review among the refusing gates but its acceptance checks do not cover it; confirm that leaving its lines and tests unchanged is enough) (source: request)"
	const intent = "# Name the failing line\n\n## Decisions\n" + line + "\n"
	for _, paraphrase := range []string{
		"contract_review refusals keep their lines unchanged, so only three gates get the line-naming and fix text",
		"contract_review keeps its refusal lines (left unchanged)",
	} {
		item := decisionItem{
			statement: line, quotes: []string{want}, verdict: "contradicts",
			intentValue: paraphrase, requestValue: want,
			difference: "The decision exempts contract_review from an explicit requirement.",
			why:        "The request includes contract_review.",
		}
		_, lines := decisionOutcome([]decisionItem{item}, request, intent)
		feedback := strings.Join(lines, "\n")
		if !strings.Contains(feedback, "fix for intent.md line 4") ||
			!strings.Contains(feedback, want) ||
			strings.Contains(feedback, assumptionRule) {
			t.Errorf("grounded conflict escaped with paraphrased value %q: %s", paraphrase, feedback)
		}
	}

	const unanswered = "- Use a heading named Fix. (assumption: confirm the heading) (source: request)"
	item := decisionItem{statement: unanswered, verdict: "unsupported", why: "the request gives no heading"}
	_, feedback := decisionOutcome([]decisionItem{item}, request, "## Decisions\n"+unanswered+"\n")
	if len(feedback) != 0 {
		t.Fatalf("a genuine assumption was refused: %q", feedback)
	}
}
