package shape

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-executor/executor"
)

// externalGuidanceParts are what every external brief tells the session about time, parallel
// work, subagents and focus.
var externalGuidanceParts = []string{
	"15-minute budget",
	"complete package in the first round",
	"independent reads, searches and commands together",
	"read-only subagents in parallel on a fast model",
	"Do not use subagents for small lookups",
	"stop exploring once the package can be written",
}

func TestExternalShapeBriefCarriesBudgetAndParallelGuidance(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(shpFiles())}}
	f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
	if len(x.starts) == 0 {
		t.Fatal("no session started")
	}
	for _, part := range externalGuidanceParts {
		shpContains(t, "stable brief", x.starts[0].Brief.Stable, part)
	}
	shpEq(t, "guidance appears once", strings.Count(x.starts[0].Brief.Stable, "15-minute budget"), 1)
}

func TestExternalShapeRoundsStateElapsedAndRemainingTime(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	clock := &shpClock{t: time.Unix(1_800_000_000, 0)}
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		func(n int, root string) (executor.RoundResult, error) {
			clock.t = clock.t.Add(6 * time.Minute)
			return result(shpFiles())(n, root)
		},
		result(shpFiles()),
	}}
	f.newSession()
	Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
		Request: "Add calc.is_even(n).", Client: f.Client, External: x.external(), Now: clock.now})
	if len(x.inputs) < 2 {
		t.Fatalf("%d rounds, want 2", len(x.inputs))
	}
	shpContains(t, "round 1 dynamic text", x.starts[0].Brief.Dynamic, "Time: 0 min elapsed, 15 min remaining of the 15-minute shaping budget.")
	shpEq(t, "round 1 feedback", x.inputs[0].Feedback, "")
	shpContains(t, "round 2 feedback", x.inputs[1].Feedback, "Time: 6 min elapsed, 9 min remaining of the 15-minute shaping budget.")
}

func TestExternalRefineBriefCarriesGuidanceAndTime(t *testing.T) {
	parallel(t)
	f := newRefFx(t, refOpts{approved: true, external: true})
	clock := &shpClock{t: time.Unix(1_800_000_000, 0)}
	bad := map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/model.qnt", "content": "module m {}\n"}}}
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		func(n int, root string) (executor.RoundResult, error) {
			clock.t = clock.t.Add(6 * time.Minute)
			return result(bad)(n, root)
		},
		result(bad),
	}}
	f.newSession()
	Refine(context.Background(), f.S, RefineInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Hash: f.hash0,
		Request: "Clarify only the existing draft; preserve its executable contract.", RequestPath: filepath.Join(f.Aux, "correction.txt"),
		Client: f.Client, External: x.external(), Now: clock.now})
	if len(x.starts) == 0 || len(x.inputs) < 2 {
		t.Fatalf("starts %d, rounds %d", len(x.starts), len(x.inputs))
	}
	for _, part := range externalGuidanceParts {
		shpContains(t, "stable brief", x.starts[0].Brief.Stable, part)
	}
	shpContains(t, "round 1 dynamic text", x.starts[0].Brief.Dynamic, "Time: 0 min elapsed, 15 min remaining")
	shpContains(t, "round 2 feedback", x.inputs[1].Feedback, "Time: 6 min elapsed, 9 min remaining of the 15-minute shaping budget.")
}

// noTimeLine is an external round's feedback without the leading time line.
func noTimeLine(feedback string) string {
	if line, rest, ok := strings.Cut(feedback, "\n\n"); ok && strings.HasPrefix(line, "Time: ") {
		return rest
	}
	return feedback
}
