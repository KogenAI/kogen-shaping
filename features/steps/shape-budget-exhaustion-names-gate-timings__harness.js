'use strict';
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// TestShapeBudgetAcceptanceTimedAttempts shapes with a scripted shaper that is refused four times
// (the validation and intent gates alternating, so the retry budget does not end it first), the
// clock moving 225 s per provider request, and writes what the user and the run log show to
// BUDGET_RESULT.
func TestShapeBudgetAcceptanceTimedAttempts(t *testing.T) {
	w := responsesWire
	badPath := func(p string) map[string]any {
		return map[string]any{"files": []any{map[string]any{"path": p, "content": "x\n"}}}
	}
	badIntent := func(c string) map[string]any {
		return map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": c}}}
	}
	f := budgetFx(t, 225*time.Second,
		w.shapeSubmit("a", badPath("../o1.txt")), w.shapeSubmit("b", badIntent("garbage one\n")),
		w.shapeSubmit("c", badPath("../o2.txt")), w.shapeSubmit("d", badIntent("garbage two\n")),
		w.readReadme("e"))
	exit, err := f.shape("Bound shaping.")
	out := map[string]any{"exit": exit, "requests": len(f.Prov.requests())}
	if e, ok := sys.AsError(err); ok {
		out["class"], out["reason"], out["detail"], out["lines"] = e.Class, e.Reason, e.Detail, e.Lines
	}
	events := f.events()
	gates := []string{}
	for _, e := range shpNamed(events, "submission_refused") {
		gates = append(gates, fmt.Sprint(e["gate"]))
	}
	out["refusedGates"] = gates
	counts := shpNamed(events, "shaping_turn_counts")
	if len(counts) == 1 {
		out["budgetSeconds"], out["elapsedSeconds"] = counts[0]["budget_seconds"], counts[0]["elapsed_seconds"]
	}
	b, _ := json.Marshal(out)
	if err := os.WriteFile(os.Getenv("BUDGET_RESULT"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
