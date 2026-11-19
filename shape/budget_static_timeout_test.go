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

func TestChargedTimeStaticTimeoutExcluded(t *testing.T) {
	_, ev := stcShape(t, []string{"slow"}, map[string]stcSpec{"slow": {120 * time.Second, 137, true}})
	if got := stcMS(t, ev, "static_uncharged_ms"); got != 120000 {
		t.Fatalf("static_uncharged_ms = %d, want 120000", got)
	}
	// chargedTime is the time since the start minus the install and the uncharged static time.
	clock := newBudgetClock()
	sh := &shaping{ev: &Evidence{Now: clock.Now}}
	sh.started = clock.Now()
	clock.Advance(130 * time.Second)
	before := sh.chargedTime()
	sh.ev.StaticUncharged(120 * time.Second)
	if got := sh.chargedTime(); got != before-120*time.Second {
		t.Fatalf("chargedTime = %v, want %v", got, before-120*time.Second)
	}
	if line := budgetLine(sh.chargedTime()); !strings.Contains(line, "0 min 10 s elapsed, 14 min 50 s remaining") {
		t.Fatalf("budget line charges the timeout: %s", line)
	}
}

func TestChargedTimeFailedStaticCheckCharged(t *testing.T) {
	_, ev := stcShape(t, []string{"slow"}, map[string]stcSpec{"slow": {120 * time.Second, 1, false}})
	if got := stcMS(t, ev, "static_uncharged_ms"); got != 0 {
		t.Fatalf("static_uncharged_ms = %d, want 0", got)
	}
	if got := stcMS(t, ev, "static_ms"); got < 120000 {
		t.Fatalf("static_ms = %d, want at least 120000", got)
	}
}

func TestChargedTimeStaticTimeoutBounded(t *testing.T) {
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

func TestShapeTimeReportsUncharged(t *testing.T) {
	clock := newBudgetClock()
	s := stmNew(t, []string{"slow", "ok"}, stcFns(clock, map[string]stcSpec{
		"slow": {120 * time.Second, 137, true},
		"ok":   {14*time.Minute + 30*time.Second, 0, false},
	}), "c1", "c2")
	s.Now = clock.Now
	_, err := s.shape("Add calc.is_even(n).")
	e, ok := sys.AsError(err)
	if !ok {
		t.Fatalf("expected the budget refusal, got %v", err)
	}
	if !strings.Contains(e.Detail, "29 min elapsed") {
		t.Fatalf("detail: %s", e.Detail)
	}
	suffix := ", static_uncharged_ms=120000"
	var figures, attempts int
	for _, l := range e.Lines {
		switch {
		case strings.HasPrefix(l, "shaping budget: 900 s"):
			figures++
		case strings.HasPrefix(l, "attempt "):
			attempts++
		default:
			continue
		}
		if !strings.HasSuffix(l, suffix) {
			t.Fatalf("line lacks %q: %s", suffix, l)
		}
	}
	if figures != 1 || attempts == 0 {
		t.Fatalf("figure lines %d, attempt lines %d: %v", figures, attempts, e.Lines)
	}
	times := shpNamed(s.events(), "shape_time")
	if len(times) != 1 || stcMS(t, times[0], "static_uncharged_ms") != 120000 || stcMS(t, times[0], "static_ms") < 120000 {
		t.Fatalf("shape_time: %v", times)
	}
}
