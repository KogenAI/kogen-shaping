package shape

import "testing"

// The held Claude recovery did useful reads, proposals and a successful scratch
// compile between three prose answers. Those are not three idle turns in a row.
func TestShapePreparationToolsResetNoToolStreak(t *testing.T) {
	parallel(t)
	w := responsesWire
	f := w.newShape(t,
		w.prose("I need to inspect the fixture."), w.readReadme("r1"),
		w.prose("I will retain the contract before checking it."), w.calls(shpCall("p1", ProposeFilesTool, shpFiles())),
		w.prose("The retained files need one last inspection."), w.readReadme("r2"),
		w.shapeSubmit("s1", shpFiles()), w.shapeAck("s2"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("preparation progress was stopped: exit %d: %v", exit, err)
	}
	shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
	counts := shpNamed(f.events(), "shaping_turn_counts")
	shpEq(t, "audit still counts all prose answers", counts[0]["no_tool_turns"], any(float64(3)))
}

func TestShapePreparationStillStopsConsecutiveIdleTurns(t *testing.T) {
	parallel(t)
	w := responsesWire
	f := w.newShape(t, w.prose("first"), w.readReadme("r1"),
		w.prose("idle one"), w.prose("idle two"), w.prose("idle three"), w.shapeSubmit("s1", shpFiles()))
	_, err := f.shape("Add calc.is_even(n).")
	shpWantError(t, err, "shape", "invalid_output", "no-tool limit", 1)
	shpEq(t, "stopped before extra submission", len(f.Prov.requests()), 5)
	shapeAbsent(t, f)
}
