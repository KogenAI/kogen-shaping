package shape

import (
	"context"
	"fmt"
	"github.com/KogenAI/kogen-executor/executor"
	"strings"
	"sync"
	"testing"
	"time"
)

// stepClock is a clock that moves by a fixed step on every provider request the fixture serves.
type stepClock struct {
	mu   sync.Mutex
	t    time.Time
	step time.Duration
}

func (c *stepClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *stepClock) tick() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(c.step)
}

// budgetFx is a native fixture whose clock moves by step per shaping request.
func budgetFx(t *testing.T, step time.Duration, steps ...shpStep) *shpFx {
	t.Helper()
	w := responsesWire
	f := w.newShape(t, steps...)
	clock := &stepClock{t: time.Unix(1_800_000_000, 0), step: step}
	f.Now = clock.now
	f.Prov.route = func(req map[string]any) *shpStep {
		if !decisionRequest(req) && !literalRequest(req) {
			clock.tick()
		}
		return nil
	}
	return f
}

func TestBudgetLineOnEveryNativeRequest(t *testing.T) {
	w := responsesWire
	f := budgetFx(t, 4*time.Minute, w.readReadme("r1"), w.readReadme("r2"), w.shapeSubmit("ok", shpFiles()))
	f.setReport("PASSED", 0)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	reqs := f.Prov.requests()
	for i, want := range []string{
		"Time: 0 min elapsed, 15 min remaining of the 15-minute shaping budget.",
		"Time: 4 min elapsed, 11 min remaining of the 15-minute shaping budget.",
		"Time: 8 min elapsed, 7 min remaining of the 15-minute shaping budget.",
	} {
		shpContains(t, fmt.Sprint("request ", i+1), shpBody(reqs[i]), want)
	}
	counts := shpNamed(f.events(), "shaping_turn_counts")
	shpEq(t, "counts", len(counts), 1)
	shpEq(t, "budget", counts[0]["budget_seconds"], any(float64(900)))
	if got, _ := counts[0]["elapsed_seconds"].(float64); got < 12*60 || int(got)%(4*60) != 0 {
		t.Fatalf("elapsed_seconds = %v, want the clock of the run (at least 720, in steps of 240)", counts[0]["elapsed_seconds"])
	}
	if _, ok := counts[0]["request_cap"]; ok {
		t.Fatal("the request cap is gone")
	}
}

func TestBudgetEndsBeforeTheNextRequest(t *testing.T) {
	w := responsesWire
	steps := make([]shpStep, 10)
	for i := range steps {
		steps[i] = w.readReadme(fmt.Sprint("r", i))
	}
	f := budgetFx(t, 5*time.Minute, steps...)
	exit, err := f.shape("Bound shaping.")
	e := shpWantError(t, err, "shape", "invalid_output", "15-minute", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "requests", len(f.Prov.requests()), 3)
	text := fmt.Sprint(e.Detail, "\n", strings.Join(e.Lines, "\n"))
	for _, want := range []string{"15 min elapsed", "3 requests"} {
		shpContains(t, "message", text, want)
	}
	counts := shpNamed(f.events(), "shaping_turn_counts")
	shpEq(t, "requests", counts[0]["requests"], any(float64(3)))
	shpEq(t, "elapsed", counts[0]["elapsed_seconds"], any(float64(900)))
	shapeAbsent(t, f)
}

func TestBudgetStopNamesTheLastRefusal(t *testing.T) {
	w := responsesWire
	both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
	f := budgetFx(t, 8*time.Minute, w.shapeSubmit("bad", both), w.readReadme("r"), w.readReadme("r2"))
	_, err := f.shape("Bound shaping.")
	e := shpWantError(t, err, "shape", "invalid_output", "15-minute", 1)
	text := fmt.Sprint(e.Detail, "\n", strings.Join(e.Lines, "\n"))
	shpContains(t, "message", text, "2 requests")
	shpContains(t, "message", text, "last refusal")
	shpEq(t, "requests", len(f.Prov.requests()), 2)
}

func TestBudgetLetsTheInFlightRequestFinish(t *testing.T) {
	w := responsesWire
	f := budgetFx(t, 20*time.Minute, w.shapeSubmit("ok", shpFiles()))
	f.setReport("PASSED", 0)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("a submission in flight when the budget ran out is kept: exit %d: %v", exit, err)
	}
}

func TestBudgetEndsTheExternalShapingBeforeTheNextRound(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	clock := &shpClock{t: time.Unix(1_800_000_000, 0)}
	bad := map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/model.qnt", "content": "module m {}\n"}}}
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		func(n int, root string) (executor.RoundResult, error) {
			clock.t = clock.t.Add(16 * time.Minute)
			return result(bad)(n, root)
		},
		result(shpFiles()),
	}}
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
		Request: "Add calc.is_even(n).", Client: f.Client, External: x.external(), Now: clock.now})
	e := shpWantError(t, err, "shape", "invalid_output", "15-minute", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "rounds", len(x.inputs), 1)
	shpContains(t, "message", strings.Join(e.Lines, "\n"), "last refusal")
}
