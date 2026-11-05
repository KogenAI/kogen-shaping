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
		{"evidence not fed back", item(evidenced, nil, "unsupported"), "unsupported", nil},
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
		if !slices.Equal(lines, append([]string{decisionHead, assumptionRule}, c.feedback...)) {
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
		decision: []func(int, string) (executor.RoundResult, error){check("contradicts", "", ""), check("contradicts", "TypeError", "raise TypeError"), check("follows", "", "")}}
	f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	if len(x.decisionStarts) != 4 {
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
	if n := len(shpNamed(f.events(), "decision_check")); n != 4 {
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
