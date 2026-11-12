package shape

import (
	"strings"
	"testing"
)

// Cases from the iteration-0 component eval set (aor-d02, aor-d03, aor-d04, ci-d02): the model
// judged a plausible consequence of request text "unsupported" and Shape refused it.
func TestConsequenceOfRequestTextIsAdvisory(t *testing.T) {
	parallel(t)
	const request = "Stripe hosts checkout and the customer portal and posts subscription changes\nto `/stripe/webhooks`.\n\nSupport wants their tools back in the admin area.\n\nThe repeated manual is absent from tool descriptions and the length limits pass."
	cases := []struct{ name, statement, quote string }{
		{"aor-d02", "- an invalid signature or unparsable payload answers 400 and changes nothing (source: request)", "posts subscription changes\nto `/stripe/webhooks`."},
		{"aor-d03", "- a checkout request responds `303` redirecting to the returned session `url`. (source: request)", "Stripe hosts checkout and the customer portal"},
		{"aor-d04", "- non-staff get 403 (source: request)", "Support wants their tools back in the admin area."},
		{"ci-d02", "- does not repeat the long agent manual (it must not contain the sentence \"Never reconstruct an approval hash\") (source: request)", "The repeated manual is absent from tool descriptions and the length limits pass."},
	}
	for _, c := range cases {
		intent := "## Decisions\n" + c.statement + "\n"
		it := decisionItem{statement: c.statement, quotes: []string{c.quote}, verdict: "consequence", why: "w"}
		event, lines, marks, notes := decisionResolveNotes([]decisionItem{it}, request, intent, nil)
		if lines != nil || marks != nil || len(notes) != 1 || !strings.Contains(notes[0], "advisory, not refused") {
			t.Errorf("%s: lines %q marks %v notes %q", c.name, lines, marks, notes)
		}
		items := event["items"].([]any)
		if v := items[0].(map[string]any)["verdict"]; v != "consequence" || event["consequence"] != 1 {
			t.Errorf("%s: event %v", c.name, event)
		}
	}
}

func TestConsequenceNeedsGroundedRequestQuote(t *testing.T) {
	parallel(t)
	const request = "Support wants their tools back in the admin area."
	const statement = "- non-staff get 403 (source: request)"
	intent := "## Decisions\n" + statement + "\n"
	for name, quotes := range map[string][]string{"no quote": nil, "quote not in request": {"Non-staff get 403."}} {
		it := decisionItem{statement: statement, quotes: quotes, verdict: "consequence", why: "w"}
		_, lines, marks, notes := decisionResolveNotes([]decisionItem{it}, request, intent, nil)
		if len(notes) != 0 {
			t.Errorf("%s: advisory note %q", name, notes)
		}
		if len(lines) == 0 && len(marks) == 0 {
			t.Errorf("%s: the statement was neither refused nor marked an assumption", name)
		}
	}
}

func TestParseDecisionCheckAcceptsConsequence(t *testing.T) {
	parallel(t)
	items, err := parseDecisionCheck(map[string]any{"items": []any{map[string]any{
		"statement": "s", "request_quotes": []any{"q"}, "verdict": "consequence",
		"intent_value": "", "request_value": "", "difference": "", "why": "w"}}})
	if err != nil || len(items) != 1 || items[0].verdict != "consequence" {
		t.Fatalf("items %v err %v", items, err)
	}
	if !strings.Contains(decisionInstructions, "consequence") {
		t.Error("the decision instructions do not describe consequence")
	}
}
