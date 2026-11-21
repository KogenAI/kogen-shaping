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
		"Time: 0 min elapsed. Target: 15 minutes usual, 20 minutes upper end.",
		"Time: 4 min elapsed. Target: 15 minutes usual, 20 minutes upper end.",
		"Time: 8 min elapsed. Target: 15 minutes usual, 20 minutes upper end.",
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

// TestElapsedTargetNeverRefusesTheNextNativeRequest: crossing the 15, 20 and 45 minute points of wall
// clock refuses no request; the later valid draft is accepted.
func TestElapsedTargetNeverRefusesTheNextNativeRequest(t *testing.T) {
	w := responsesWire
	f := budgetFx(t, 20*time.Minute, w.readReadme("r1"), w.readReadme("r2"), w.readReadme("r3"), w.shapeSubmit("ok", shpFiles()))
	f.setReport("PASSED", 0)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	reqs := f.Prov.requests()
	shpEq(t, "requests", len(reqs), 4)
	for i, want := range []string{"0 min", "20 min", "40 min", "60 min"} {
		shpContains(t, fmt.Sprint("request ", i+1), shpBody(reqs[i]), "Time: "+want+" elapsed. Target: 15 minutes usual, 20 minutes upper end.")
	}
	counts := shpNamed(f.events(), "shaping_turn_counts")
	shpEq(t, "counts", len(counts), 1)
	if got, _ := counts[0]["elapsed_seconds"].(float64); got < 3600 {
		t.Fatalf("elapsed_seconds = %v, want at least 3600", counts[0]["elapsed_seconds"])
	}
	for _, r := range reqs {
		for _, gone := range []string{"remaining of the", "min remaining", "used its"} {
			if strings.Contains(shpBody(r), gone) {
				t.Fatalf("a request still says %q", gone)
			}
		}
	}
}

func TestBudgetLetsTheInFlightRequestFinish(t *testing.T) {
	w := responsesWire
	f := budgetFx(t, 20*time.Minute, w.shapeSubmit("ok", shpFiles()))
	f.setReport("PASSED", 0)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("a submission in flight when the budget ran out is kept: exit %d: %v", exit, err)
	}
}

// TestElapsedTargetNeverRefusesTheNextExternalRound: a refused first round of 16 minutes does not
// end the external shaping; the next round is made with the elapsed time and the target.
func TestElapsedTargetNeverRefusesTheNextExternalRound(t *testing.T) {
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
		func(n int, root string) (executor.RoundResult, error) {
			clock.t = clock.t.Add(30 * time.Minute)
			return result(shpFiles())(n, root)
		},
		result(shpFiles()),
	}}
	f.newSession()
	Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
		Request: "Add calc.is_even(n).", Client: f.Client, External: x.external(), Now: clock.now})
	if len(x.inputs) < 3 {
		t.Fatalf("%d rounds, want at least 3", len(x.inputs))
	}
	shpContains(t, "round 2 feedback", x.inputs[1].Feedback, "Time: 16 min elapsed. Target: 15 minutes usual, 20 minutes upper end.")
	shpContains(t, "round 3 feedback", x.inputs[2].Feedback, "Time: 46 min elapsed. Target: 15 minutes usual, 20 minutes upper end.")
}

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
			want := fmt.Sprintf("Time: 0 min 10 s elapsed. Target: %d minutes usual, %d minutes upper end.", c.minutes, c.minutes+5)
			shpEq(t, "time line", budgetSecondLine(f), want)
			counts := shpNamed(f.events(), "shaping_turn_counts")
			shpEq(t, "events", len(counts), 1)
			shpEq[any](t, "budget_seconds", counts[0]["budget_seconds"], float64(c.minutes*60))
		})
	}
}

// TestBudgetEnvChangesGuidanceOnly: whatever the configured minutes, a shaping whose wall clock is far
// past the target makes every request.
func TestBudgetEnvChangesGuidanceOnly(t *testing.T) {
	parallel(t)
	for _, value := range []*string{nil, budgetValue("5"), budgetValue("25"), budgetValue("45")} {
		f := budgetFx(t, 50*time.Minute, responsesWire.readReadme("r1"), responsesWire.readReadme("r2"), responsesWire.shapeSubmit("ok", shpFiles()))
		f.setReport("PASSED", 0)
		withBudgetEnv(f, value)
		if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpEq(t, "requests", len(f.Prov.requests()), 3)
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
	shpContains(t, "25", budgetSecondLine(f), "Target: 25 minutes usual, 30 minutes upper end.")
	g, _, err := budgetRun(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "unset", budgetSecondLine(g), "Target: 15 minutes usual, 20 minutes upper end.")
}
