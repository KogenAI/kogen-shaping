package shape

import (
	"strings"
	"testing"
)

// TestDecisionWeakenedGuarantee: the instructions tell the checking model to refuse a Decision
// that weakens an absolute guarantee even when it is assumption-marked, and the scripted
// contradicts verdict is refused naming both values while a full-guarantee Decision passes.
func TestDecisionWeakenedGuarantee(t *testing.T) {
	parallel(t)
	for _, want := range []string{"at-most-once", "at-least-once", "best-effort",
		"even when the Decision carries an assumption marker", "keeps the full guarantee", "open question for the owner"} {
		if !strings.Contains(decisionInstructions, want) {
			t.Errorf("instructions lack %q", want)
		}
	}
	for _, bad := range []string{"webhook", "delivery", "restart"} {
		if strings.Contains(decisionInstructions, bad) {
			t.Errorf("instructions contain task-specific word %q", bad)
		}
	}
	const request = "Process each job exactly once."
	const weak = "- Exactly once means at-most-once on a crash. (assumption: crash handling is silent) (source: request)"
	const full = "- Exactly once means each job runs once and a job interrupted by a crash is resumed, neither lost nor repeated. (source: request)"
	run := func(statement, verdict, intentValue, requestValue string) []string {
		it := decisionItem{statement: statement, quotes: []string{request}, verdict: verdict, intentValue: intentValue, requestValue: requestValue, difference: "d", why: "w"}
		_, refusal, _, _ := decisionResolveNotes([]decisionItem{it}, request, "## Decisions\n"+statement+"\n", nil)
		return refusal
	}
	refusal := run(weak, "contradicts", "at-most-once on a crash", "exactly once")
	got := strings.Join(refusal, "\n")
	for _, want := range []string{"at-most-once on a crash", "exactly once"} {
		if !strings.Contains(got, want) {
			t.Errorf("refusal %q does not name %q", got, want)
		}
	}
	if len(refusal) == 0 {
		t.Error("the weakened Decision is not refused")
	}
	if refusal := run(full, "follows", "", ""); len(refusal) != 0 {
		t.Errorf("full-guarantee Decision refused: %v", refusal)
	}
}
