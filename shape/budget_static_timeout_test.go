package shape

import (
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// stcSpec is a fake check: it advances the fake clock by its wall time and ends as given.
type stcSpec struct {
	wall     time.Duration
	status   int
	timedOut bool
}

func stcFns(clock *budgetClock, specs map[string]stcSpec) map[string]stmCheckFn {
	fns := map[string]stmCheckFn{}
	for n, sp := range specs {
		sp := sp
		fns[n] = func(int) sys.Result {
			clock.Advance(sp.wall)
			stmWork()
			return sys.Result{Status: sp.status, TimedOut: sp.timedOut}
		}
	}
	return fns
}

// stcShape shapes once with the fake checks and returns the fixture and its shape_time event.
func stcShape(t *testing.T, names []string, specs map[string]stcSpec) (*stmFixture, map[string]any) {
	t.Helper()
	clock := newBudgetClock()
	s := stmNew(t, names, stcFns(clock, specs), "c1")
	s.Now = clock.Now
	s.shapeOnce(t)
	times := shpNamed(s.events(), "shape_time")
	if len(times) != 1 {
		t.Fatalf("shape_time events: %d", len(times))
	}
	return s, times[0]
}

func stcMS(t *testing.T, ev map[string]any, key string) int64 {
	t.Helper()
	v, ok := ev[key].(float64)
	if !ok {
		t.Fatalf("shape_time has no number %s: %v", key, ev)
	}
	return int64(v)
}

func TestShapeTimeStaticTimeoutReportedUncharged(t *testing.T) {
	_, ev := stcShape(t, []string{"slow"}, map[string]stcSpec{"slow": {120 * time.Second, 137, true}})
	if got := stcMS(t, ev, "static_uncharged_ms"); got != 120000 {
		t.Fatalf("static_uncharged_ms = %d, want 120000", got)
	}
}

func TestShapeTimeFailedStaticCheckNotUncharged(t *testing.T) {
	_, ev := stcShape(t, []string{"slow"}, map[string]stcSpec{"slow": {120 * time.Second, 1, false}})
	if got := stcMS(t, ev, "static_uncharged_ms"); got != 0 {
		t.Fatalf("static_uncharged_ms = %d, want 0", got)
	}
	if got := stcMS(t, ev, "static_ms"); got < 120000 {
		t.Fatalf("static_ms = %d, want at least 120000", got)
	}
}

func TestShapeTimeStaticTimeoutsSummed(t *testing.T) {
	_, ev := stcShape(t, []string{"a", "b", "ok"}, map[string]stcSpec{
		"a":  {150 * time.Second, 137, true},
		"b":  {120 * time.Second, 137, true},
		"ok": {30 * time.Second, 0, false},
	})
	if got := stcMS(t, ev, "static_ms"); got != 300000 {
		t.Fatalf("static_ms = %d, want 300000", got)
	}
	if got := stcMS(t, ev, "static_uncharged_ms"); got != 240000 {
		t.Fatalf("static_uncharged_ms = %d, want 240000", got)
	}
}

// TestShapeTimeReportsUncharged: a timed-out static check is reported in static_uncharged_ms, and a
// shaping whose wall clock is far past the target is not refused for it.
func TestShapeTimeReportsUncharged(t *testing.T) {
	clock := newBudgetClock()
	s := stmNew(t, []string{"slow", "ok"}, stcFns(clock, map[string]stcSpec{
		"slow": {120 * time.Second, 137, true},
		"ok":   {14*time.Minute + 30*time.Second, 0, false},
	}), "c1", "c2")
	s.Now = clock.Now
	_, err := s.shape("Add calc.is_even(n).")
	if e, ok := sys.AsError(err); ok && strings.Contains(e.Detail, "shaping budget") {
		t.Fatalf("the elapsed target refused shaping: %v", e)
	}
	times := shpNamed(s.events(), "shape_time")
	if len(times) != 1 || stcMS(t, times[0], "static_uncharged_ms") != 120000 || stcMS(t, times[0], "static_ms") < 120000 {
		t.Fatalf("shape_time: %v", times)
	}
	if got := stcMS(t, times[0], "local_check_ms"); got < 120000 {
		t.Fatalf("local_check_ms = %d, want at least the static checks", got)
	}
}
