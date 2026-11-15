package shape

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestDecisionRepeatedRefusal(t *testing.T) {
	parallel(t)
	const request = "Deliver exactly once."
	const original = "- Deliver at least once. (source: request)"
	const changed = "- Deliver at most once. (source: request)"
	for _, tc := range []struct {
		name        string
		statements  []string
		verdicts    []string
		submissions []int
		earlier     []int
	}{
		{"repeat and earliest nonadjacent", []string{original, original, changed, original}, []string{"contradicts", "contradicts", "contradicts", "contradicts"}, []int{1, 2, 3, 4}, []int{0, 1, 0, 1}},
		{"whitespace and gap", []string{original, strings.Replace(original, "Deliver at", "Deliver\t  at", 1)}, []string{"contradicts", "contradicts"}, []int{2, 4}, []int{0, 2}},
		{"pass then first refusal then pass", []string{original, original, original}, []string{"follows", "contradicts", "follows"}, []int{1, 2, 3}, []int{0, 0, 0}},
		{"other bytes significant", []string{original, strings.Replace(original, "Deliver", "deliver", 1)}, []string{"contradicts", "contradicts"}, []int{1, 2}, []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newShpFx(t)
			sh := f.shaping()
			sh.role = sh.cfg.Models["shape"]
			sh.request = request
			current := 0
			f.Prov.route = func(req map[string]any) *shpStep {
				if !decisionRequest(req) {
					return nil
				}
				item := map[string]any{"statement": tc.statements[current], "request_quotes": []any{request}, "verdict": tc.verdicts[current], "intent_value": "at least once", "request_value": "exactly once", "difference": "different guarantees", "why": "The delivery guarantee differs."}
				if strings.Contains(tc.statements[current], "at most") {
					item["intent_value"] = "at most once"
				}
				answer := shpAnswer(shpCall(fmt.Sprintf("d%d", current), SubmitDecisionCheckTool, map[string]any{"items": []any{item}}))
				return &answer
			}
			for i, statement := range tc.statements {
				current = i
				sh.submissions = tc.submissions[i]
				intent := "# Candidate\n" + strings.Repeat("\n", i) + "## Decisions\n" + statement + "\n"
				value := "at least once"
				if strings.Contains(statement, "at most") {
					value = "at most once"
				}
				_, want := decisionOutcome([]decisionItem{{statement: statement, quotes: []string{request}, verdict: tc.verdicts[i], intentValue: value, requestValue: "exactly once", difference: "different guarantees", why: "The delivery guarantee differs."}}, request, intent)
				if first := tc.earlier[i]; first != 0 {
					want = append(want, fmt.Sprintf("note: this statement is unchanged since submission %d and was refused for the same reason; change the statement itself, not the text around it", first))
				}
				got, err := sh.decisionCheckWithEvidence(context.Background(), intent, "")
				if err != nil || !slices.Equal(got, want) {
					t.Fatalf("submission %d: got %q, err %v; want %q", sh.submissions, got, err, want)
				}
			}
			requests := f.Prov.decisionRequests()
			if len(requests) != len(tc.statements) {
				t.Fatalf("checker calls: %d", len(requests))
			}
			for _, req := range requests {
				if strings.Contains(fmt.Sprint(req["input"]), "note: this statement is unchanged") {
					t.Fatal("history leaked into checker input")
				}
			}
		})
	}
}
