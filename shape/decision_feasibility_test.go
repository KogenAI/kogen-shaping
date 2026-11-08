package shape

import (
	"fmt"
	"strings"
	"testing"
)

func TestDecisionCheckSeparatesMeasuredFeasibility(t *testing.T) {
	parallel(t)
	const measured = "- Measured fixture 120120 bytes. (probe 1: exit 0, 0.1 s)"
	files := shpFiles()
	for _, raw := range files["files"].([]any) {
		file := raw.(map[string]any)
		if strings.HasSuffix(file["path"].(string), "/intent.md") {
			file["content"] = strings.Replace(file["content"].(string), "## Feasibility\n- none", "## Feasibility\n"+measured, 1)
		}
	}
	f := newShpFx(t, shpAnswer(shpCall("probe", "probe", map[string]any{"command": "printf '120120\\n'"})), shpSubmit("files", files), shpAck("ack"))
	f.Prov.route = func(req map[string]any) *shpStep {
		if decisionRequest(req) && strings.Contains(fmt.Sprint(req), measured) {
			answer := shpAnswer(shpCall("decision", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{
				"statement": measured, "request_quotes": []any{}, "verdict": "unsupported", "intent_value": "", "request_value": "", "difference": "", "why": "The request does not specify measured fixture bytes.",
			}}}))
			return &answer
		}
		return nil
	}
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("observed feasibility was treated as a product decision: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	if !strings.Contains(f.read(".kogen/intents/is-even/intent.md"), measured) {
		t.Fatal("the evidence must remain in the complete published Intent")
	}
	for _, req := range f.Prov.decisionRequests() {
		text := fmt.Sprint(req)
		if strings.Contains(text, measured) || !strings.Contains(text, "Non-integers raise TypeError") {
			t.Fatalf("wrong decision-check scope: %s", text)
		}
	}
}

func TestDecisionProjectionPreservesNormativeClaims(t *testing.T) {
	const before = "# Feature\n\n```markdown\n## Feasibility\nf(0) returns False.\n```\n\n## Decisions\n- f(0) returns False. (source: request)\n"
	const evidence = "## Feasibility\n- Measured 120120 bytes. (probe 1: exit 0, 0.1 s)\n- evidence: {\"checks\":[{\"exit\":0,\"output\":\"accepted\"}]}\n"
	const after = "## Other contract\nf(2) returns False.\n"
	got := decisionIntent(before + evidence + after)
	if !strings.HasPrefix(got, before) || !strings.HasSuffix(got, after) || strings.Contains(got, "120120") || strings.Contains(got, "accepted") {
		t.Fatalf("projection changed normative claims or retained measurements: %s", got)
	}
	it := decisionItem{statement: "- f(0) returns False. (source: request)", quotes: []string{"f(0) returns True."}, verdict: "contradicts", intentValue: "False", requestValue: "True", difference: "different Boolean", why: "request requires True"}
	if _, lines := decisionOutcome([]decisionItem{it}, "f(0) returns True.", got); len(lines) == 0 {
		t.Fatal("a real normative contradiction escaped the check")
	}
}
