package shape

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

// shpInvalidFiles is a submission every gate refuses: a file outside the Intent.
func shpInvalidFiles() map[string]any {
	return map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
}

// shpUndocumentedFiles is a submission refused at another gate: a documentation record the shaper
// wrote itself.
func shpUndocumentedFiles() map[string]any {
	files := shpFiles()
	files["files"].([]any)[0].(map[string]any)["content"] = strings.Replace(shpScopedIntentMD, "## Tests",
		"## Source comparison\n- documentation-lookup: retrieved https://example.invalid/doc\n\n## Tests", 1)
	return files
}

// The last allowed round may produce a package that passes every gate and only awaits the
// acknowledgment of its readiness observations. The cap must not count that acknowledging round:
// rounds 1-5 are refused or handed observations, round 6 passes every gate, round 7 only
// resubmits it unchanged, and the Intent is approved.
func TestExternalShaperLastRoundPassingPackageIsAcknowledged(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	outside, undocumented := result(shpInvalidFiles()), result(shpUndocumentedFiles())
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		outside, undocumented, result(shpFiles()), outside, undocumented, revisedFeature(1), revisedFeature(1)}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\nstderr: %s", exit, err, f.Stderr)
	}
	if len(x.inputs) != 7 {
		t.Fatalf("%d rounds, want 7: six counted and the acknowledging one", len(x.inputs))
	}
	if !f.exists(".kogen/local/provenance/is-even/current.json") {
		t.Fatal("the acknowledged package has no provenance")
	}
}

// The acknowledging round is for the acknowledgment only: a package changed in it is not given
// another round, and a refusal in the last counted round still ends at the cap.
func TestExternalShaperAcknowledgingRoundIsNotAFurtherRepairRound(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	invalid := result(shpInvalidFiles())
	x := &shapeExec{t: t, cap: 2, rounds: []func(int, string) (executor.RoundResult, error){
		invalid, result(shpFiles()), revisedFeature(1), revisedFeature(2)}}
	_, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	shpWantError(t, err, "candidate", "executor_round_limit", "the executor used its 2 rounds", 1)
	if len(x.inputs) != 3 {
		t.Fatalf("%d rounds, want 3", len(x.inputs))
	}
}
