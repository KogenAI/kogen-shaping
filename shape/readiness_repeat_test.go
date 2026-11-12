package shape

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

// revisedFeature is the shaper's resubmission of the feature with a comment added: a changed
// package that gets fresh readiness, whose failures are the same as the previous package's.
func revisedFeature(i int) func(int, string) (executor.RoundResult, error) {
	return result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature + fmt.Sprintf("# revision %d\n", i)}}})
}

// TestExternalShaperHandedTheSameReadinessFailuresStopsAtTheThird: an external shaper that
// keeps resubmitting a changed package is handed the same readiness failures each time (only the
// package hash and evidence path differ). The second identical return says so; the third ends
// shaping with a message that names the repeat instead of spending rounds until the round cap.
func TestExternalShaperHandedTheSameReadinessFailuresStopsAtTheThird(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	rounds := []func(int, string) (executor.RoundResult, error){result(shpFiles())}
	for i := 1; i < 6; i++ {
		rounds = append(rounds, revisedFeature(i))
	}
	x := &shapeExec{t: t, rounds: rounds}
	_, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	e := shpWantError(t, err, "shape", "invalid_output", "the same readiness failures were returned 3 times", 1)
	shpContains(t, "stop lines", strings.Join(e.Lines, "\n"), "readiness failures returned for acknowledgment, see .kogen/local/shaping/is-even/observations-")
	if len(x.inputs) != 3 {
		t.Fatalf("%d rounds, want shaping to stop at the third identical return", len(x.inputs))
	}
	if strings.Contains(x.inputs[1].Feedback, "identical to the ones already returned") {
		t.Fatalf("the first return carries a repeat notice: %s", x.inputs[1].Feedback)
	}
	shpContains(t, "second return", x.inputs[2].Feedback, "identical to the ones already returned (2 times)")
	shpContains(t, "second return", x.inputs[2].Feedback, "returning them 3 times stops shaping")
	for _, p := range []string{".kogen/local/provenance/is-even/current.json", ".kogen/local/approvals/is-even.json"} {
		if f.exists(p) {
			t.Fatalf("%s exists after the stop", p)
		}
	}
	noNativeRequests(t, f)
}

// A second identical return is tolerated: the shaper that then acknowledges the red as the
// notice describes, by resubmitting the exact current artifact, finishes as before.
func TestExternalShaperAcknowledgesAfterOneRepeatedReturn(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(shpFiles()), revisedFeature(1), revisedFeature(1)}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\nstderr: %s", exit, err, f.Stderr)
	}
	if len(x.inputs) != 3 {
		t.Fatalf("%d rounds, want 3", len(x.inputs))
	}
	shpContains(t, "second return", x.inputs[2].Feedback, "identical to the ones already returned (2 times)")
}

// Failures that change between returns are not a repeat: the count is of identical failures.
func TestExternalShaperChangedReadinessFailuresAreNotARepeat(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	renamed := strings.Replace(shpFeature, "Even numbers", "Even integers", 1)
	feature := func(content string) func(int, string) (executor.RoundResult, error) {
		return result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": content}}})
	}
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		result(shpFiles()), feature(renamed), feature(renamed + "# revision\n"), feature(renamed + "# revision\n")}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\nstderr: %s", exit, err, f.Stderr)
	}
	if len(x.inputs) != 4 {
		t.Fatalf("%d rounds, want 4", len(x.inputs))
	}
	if strings.Contains(x.inputs[1].Feedback, "identical to the ones already returned") {
		t.Fatalf("changed failures carry a repeat notice")
	}
}
