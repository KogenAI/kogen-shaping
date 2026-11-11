package shape

import (
	"strings"
	"testing"
)

func TestNativeBoundsRegressionAlternating(t *testing.T) {
	w := responsesWire
	f := w.newShape(t, w.prose("one"), w.readReadme("r1"), w.prose("two"), w.readReadme("r2"), w.prose("three"), w.readReadme("r3"), w.prose("four"))
	exit, err := f.shape("Bound shaping.")
	e := shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); no submission was made", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "requests", len(f.Prov.requests()), 5)
	shpEq(t, "last answer", strings.Join(e.Lines, "|"), "the shaper ended without calling submit|last tool-free answer: three")
	shpEq(t, "answers", len(shpNamed(f.events(), "tool_free_answer")), 3)
	shpEq(t, "forced", len(shpNamed(f.events(), "forced_submit_turn")), 2)
}

func TestNativeBoundsRegressionReset(t *testing.T) {
	w := responsesWire
	both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
	f := w.newShape(t, w.readReadme("r"), w.prose("one"), w.prose("two"), w.shapeSubmit("bad", both), w.prose("three"), w.prose("four"), w.shapeSubmit("ok", shpFiles()))
	f.setReport("PASSED", 0)
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("reset: exit %d: %v", exit, err)
	}
	shpEq(t, "requests", len(f.Prov.requests()), 7)
	shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
}

func TestNativeBoundsRegressionConsecutive(t *testing.T) {
	w := responsesWire
	f := w.newShape(t, w.prose("one"), w.prose("two"), w.prose("three"), w.prose("four"))
	exit, err := f.shape("Bound shaping.")
	shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); no submission was made", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "requests", len(f.Prov.requests()), 3)
}
