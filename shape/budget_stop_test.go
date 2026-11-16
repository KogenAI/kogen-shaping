package shape

import (
	"slices"
	"testing"
	"time"
)

func TestBudgetStopNoSubmitLine(t *testing.T) {
	parallel(t)
	w := responsesWire
	bad := func(path string) map[string]any {
		return map[string]any{"files": []any{map[string]any{"path": path, "content": "package main\n"}}}
	}
	for _, submitted := range []bool{true, false} {
		name := "never submitted"
		if submitted {
			name = "two refused submissions"
		}
		t.Run(name, func(t *testing.T) {
			steps := []shpStep{w.readReadme("r1"), w.readReadme("r2")}
			want := []string{"the shaper ended without calling submit", budgetFigureLine}
			if submitted {
				steps = []shpStep{w.shapeSubmit("first", bad("first.go")), w.shapeSubmit("second", bad("second.go"))}
				want = []string{
					"last refusal (attempt 2, gate validation):",
					"second.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/",
					"earliest unresolved refusal (attempt 1, gate validation):",
					"first.go: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/",
					budgetFigureLine,
					"attempt 1: gate validation, model request 480 s, Kogen checks 0 s, 420 s remaining",
					"attempt 2: gate validation, model request 480 s, Kogen checks 0 s, 0 s remaining",
				}
			}
			f := budgetFx(t, 8*time.Minute, steps...)
			exit, err := f.shape("Bound shaping.")
			e := shpWantError(t, err, "shape", "invalid_output", "15-minute", 1)
			shpEq(t, "exit", exit, 1)
			shpEq(t, "requests", len(f.Prov.requests()), 2)
			shpEq(t, "detail", e.Detail, "the shaper used its 15-minute shaping budget (16 min elapsed, 2 requests) without a submission that passed; shaping stopped")
			if !slices.Equal(e.Lines, want) {
				t.Fatalf("refusal lines = %q, want %q", e.Lines, want)
			}
		})
	}
}
