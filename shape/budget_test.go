package shape

import (
	"context"
	"fmt"
	"github.com/KogenAI/kogen-executor/executor"
	"slices"
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
	for _, want := range []string{"15 min elapsed", "3 requests", budgetFigureLine} {
		shpContains(t, "message", text, want)
	}
	if strings.Contains(text, "attempt ") {
		t.Fatalf("no submission was refused, so no attempt is listed:\n%s", text)
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
	lines := strings.Join(e.Lines, "\n")
	shpContains(t, "message", lines, "last refusal")
	shpContains(t, "message", lines, budgetFigureLine)
	// The round took 960 s of the 900 s budget: the remaining budget is never below 0.
	shpContains(t, "message", lines, "attempt 1: gate ")
	shpContains(t, "message", lines, ", model request 960 s, Kogen checks 0 s, 0 s remaining")
}

// TestBudgetStopListsTheLastSubmissionChecks: a last submission that was checked without a gate
// refusal is listed after the refused attempts, with 0 s remaining and the seconds over the budget.
func TestBudgetStopListsTheLastSubmissionChecks(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
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
			advance(10*time.Minute, result(bad)),
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
	e := shpWantError(t, err, "shape", "invalid_output", "15-minute", 1)
	var attempts []string
	for _, l := range e.Lines {
		if strings.HasPrefix(l, "attempt ") {
			attempts = append(attempts, strings.ReplaceAll(l, "\n", " "))
		}
	}
	want := []string{
		"attempt 1: gate validation, model request 600 s, Kogen checks 0 s, 300 s remaining",
		"attempt 2: gate contract, model request 240 s, Kogen checks 120 s, 0 s remaining, 60 s over",
	}
	if !slices.Equal(attempts, want) {
		t.Fatalf("attempt lines:\n%s\nwant:\n%s", strings.Join(attempts, "\n"), strings.Join(want, "\n"))
	}
}

// budgetFigureLine is the refusal line that states the shaping budget and names the other budget.
const budgetFigureLine = "shaping budget: 900 s (15 min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)"

// withBudgetEnv sets KOGEN_SHAPE_BUDGET_MINUTES in the fixture's session environment; nil unsets it.
func withBudgetEnv(f *shpFx, value *string) {
	var env []string
	for _, kv := range f.Env {
		if !strings.HasPrefix(kv, budgetEnv+"=") {
			env = append(env, kv)
		}
	}
	if value != nil {
		env = append(env, budgetEnv+"="+*value)
	}
	f.Env = env
}

func budgetValue(s string) *string { return &s }

// budgetRun is a passing shaping whose clock moves 10 s per request.
func budgetRun(t *testing.T, value *string) (f *shpFx, exit int, err error) {
	t.Helper()
	w := responsesWire
	f = budgetFx(t, 10*time.Second, w.readReadme("r1"), w.readReadme("r2"), w.shapeSubmit("ok", shpFiles()))
	f.setReport("PASSED", 0)
	withBudgetEnv(f, value)
	exit, err = f.shape("Add calc.is_even(n).")
	return f, exit, err
}

func budgetSecondLine(f *shpFx) string {
	line := ""
	for _, l := range strings.Split(shpBody(f.Prov.requests()[1]), "\n") {
		if strings.HasPrefix(l, "Time: ") {
			line = l
		}
	}
	return line
}

func TestBudgetEnvSetsTimeLineAndEvent(t *testing.T) {
	parallel(t)
	for _, c := range []struct {
		name    string
		value   *string
		minutes int
	}{{"unset", nil, 15}, {"empty", budgetValue(""), 15}, {"25", budgetValue("25"), 25}, {"5", budgetValue("5"), 5}, {"45", budgetValue("45"), 45}} {
		t.Run(c.name, func(t *testing.T) {
			f, exit, err := budgetRun(t, c.value)
			if err != nil || exit != 0 {
				t.Fatalf("exit %d: %v", exit, err)
			}
			want := fmt.Sprintf("Time: 0 min 10 s elapsed, %d min 50 s remaining of the %d-minute shaping budget.", c.minutes-1, c.minutes)
			shpEq(t, "time line", budgetSecondLine(f), want)
			counts := shpNamed(f.events(), "shaping_turn_counts")
			shpEq(t, "events", len(counts), 1)
			shpEq[any](t, "budget_seconds", counts[0]["budget_seconds"], float64(c.minutes*60))
		})
	}
}

func TestBudgetEnvSpentBoundary(t *testing.T) {
	parallel(t)
	requests := func(value *string, seconds int) int {
		steps := make([]shpStep, 10)
		for i := range steps {
			steps[i] = responsesWire.readReadme(fmt.Sprint("r", i))
		}
		f := budgetFx(t, time.Duration(seconds)*time.Second, steps...)
		withBudgetEnv(f, value)
		f.shape("Bound shaping.")
		return len(f.Prov.requests())
	}
	for _, c := range []struct {
		value *string
		secs  int
		spent bool
	}{{nil, 899, false}, {nil, 900, true}, {budgetValue("25"), 900, false}, {budgetValue("25"), 1499, false}, {budgetValue("25"), 1500, true}} {
		shpEq(t, fmt.Sprintf("spent at %d s", c.secs), requests(c.value, c.secs) == 1, c.spent)
	}
}

func TestBudgetEnvRefusesBadValuesBeforeAnyRequest(t *testing.T) {
	parallel(t)
	for _, v := range []string{"46", "4", "0", "-5", "abc", "25.5", " 25", "25 ", "+25", "1000000000000000000000"} {
		t.Run(v, func(t *testing.T) {
			f, exit, err := budgetRun(t, budgetValue(v))
			e := shpWantError(t, err, "environment", "shape_budget", "shape budget must be a whole number of minutes from 5 to 45", 3)
			shpEq(t, "exit", exit, 3)
			shpEq(t, "class", e.Class, "environment")
			shpEq(t, "requests", len(f.Prov.requests()), 0)
		})
	}
}

func TestBudgetEnvIsNotSharedBetweenShapings(t *testing.T) {
	parallel(t)
	f, _, err := budgetRun(t, budgetValue("25"))
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "25", budgetSecondLine(f), "of the 25-minute shaping budget.")
	g, _, err := budgetRun(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "unset", budgetSecondLine(g), "of the 15-minute shaping budget.")
}
