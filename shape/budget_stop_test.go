package shape

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-executor/executor"
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

// TestBudgetStopStatesTheConfiguredBudget: with a 25-minute budget the outcome text, the budget
// line and the remaining and over figures all use 25 minutes.
func TestBudgetStopStatesTheConfiguredBudget(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	withBudgetEnv(f, budgetValue("25"))
	clock := &shpClock{t: time.Unix(1_800_000_000, 0)}
	bad := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
	advance := func(d time.Duration, next func(int, string) (executor.RoundResult, error)) func(int, string) (executor.RoundResult, error) {
		return func(n int, root string) (executor.RoundResult, error) {
			clock.t = clock.t.Add(d)
			return next(n, root)
		}
	}
	contractCalls := 0
	review := result(map[string]any{"findings": []any{}, "environment": []any{}})
	x := &shapeExec{t: t,
		rounds: []func(int, string) (executor.RoundResult, error){
			advance(20*time.Minute, result(bad)),
			advance(4*time.Minute, result(shpFiles())),
		},
		contract: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				contractCalls++
				if contractCalls == 1 {
					clock.t = clock.t.Add(2 * time.Minute)
				}
				return review(n, root)
			},
		},
	}
	f.newSession()
	_, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
		Request: "Add calc.is_even(n).", Client: f.Client, External: x.external(), Now: clock.now})
	e := shpWantError(t, err, "shape", "invalid_output", "the shaper used its 25-minute shaping budget", 1)
	var attempts []string
	budgetLines := 0
	for _, l := range e.Lines {
		if strings.HasPrefix(l, "shaping budget: ") {
			budgetLines++
			if !strings.HasPrefix(l, "shaping budget: 1500 s (25 min") {
				t.Fatalf("budget line: %s", l)
			}
		}
		if strings.HasPrefix(l, "attempt ") {
			attempts = append(attempts, strings.ReplaceAll(l, "\n", " "))
		}
	}
	shpEq(t, "budget lines", budgetLines, 1)
	want := []string{
		"attempt 1: gate validation, model request 1200 s, Kogen checks 0 s, 300 s remaining",
		"attempt 2: gate contract, model request 240 s, Kogen checks 120 s, 0 s remaining, 60 s over",
	}
	if !slices.Equal(attempts, want) {
		t.Fatalf("attempt lines:\n%s\nwant:\n%s", strings.Join(attempts, "\n"), strings.Join(want, "\n"))
	}
}

// TestExternalBriefStatesTheBudget: the external session brief carries the budget it is given.
func TestExternalBriefStatesTheBudget(t *testing.T) {
	parallel(t)
	if externalBriefFor(25*time.Minute) == externalBriefFor(shapingBudget) {
		t.Fatal("the external brief does not depend on the budget")
	}
	shpEq(t, "default brief", externalBrief(), externalBriefFor(shapingBudget))
	shpContains(t, "25 brief", externalBriefFor(25*time.Minute), "25")
}
