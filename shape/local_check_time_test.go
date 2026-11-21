package shape

import "testing"

// localCheckAccount runs build against a fake-clock Evidence and returns its shape_time event.
func localCheckAccount(t *testing.T, build func(ev *Evidence, clock *shpClock)) map[string]any {
	t.Helper()
	ev, clock, dir := newTestEvidence(t)
	ev.AccountTime()
	build(ev, clock)
	if err := ev.Finish(); err != nil {
		t.Fatal(err)
	}
	got := shpNamed(readEvents(t, dir), "shape_time")
	if len(got) != 1 {
		t.Fatalf("shape_time events: %d", len(got))
	}
	return got[0]
}

func localCheckRound(ev *Evidence, clock *shpClock, ms int64) {
	a := ev.Millis()
	clock.advance(ms)
	ev.ShaperRound(a, ev.Millis(), nil)
}

// TestShapeTimeLocalCheckIsTheUnionOfReadinessAndStaticSpans: the readiness observation and the static
// checks count once where they overlap; model checks and shaper time are not local check time.
func TestShapeTimeLocalCheckIsTheUnionOfReadinessAndStaticSpans(t *testing.T) {
	for _, tc := range []struct {
		name            string
		readiness       [2]int64 // start offset and end offset after the 100 s request, in ms
		static          [2]int64
		model           [2]int64
		wantLocal       int64
		wantElapsed     int64
		rounds, advance int64
	}{
		{name: "started together", readiness: [2]int64{0, 300000}, static: [2]int64{0, 120000}, wantLocal: 300000, wantElapsed: 400000, advance: 300000},
		{name: "staggered", readiness: [2]int64{0, 300000}, static: [2]int64{200000, 320000}, wantLocal: 320000, wantElapsed: 420000, advance: 320000},
		{name: "readiness alone", readiness: [2]int64{0, 400000}, wantLocal: 400000, wantElapsed: 500000, advance: 400000},
		{name: "static alone", static: [2]int64{0, 90000}, wantLocal: 90000, wantElapsed: 190000, advance: 90000},
		{name: "decision check alone", model: [2]int64{0, 60000}, wantElapsed: 160000, advance: 60000},
		{name: "none", wantElapsed: 100000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := localCheckAccount(t, func(ev *Evidence, clock *shpClock) {
				localCheckRound(ev, clock, 100000)
				b := ev.Millis()
				clock.advance(tc.advance)
				if tc.readiness != ([2]int64{}) {
					ev.Interval("readiness", b+tc.readiness[0], b+tc.readiness[1])
				}
				if tc.static != ([2]int64{}) {
					ev.Interval("static", b+tc.static[0], b+tc.static[1])
				}
				if tc.model != ([2]int64{}) {
					ev.Interval("model_checks", b+tc.model[0], b+tc.model[1])
				}
				if tc.advance > 0 {
					ev.CheckSpan(b, b+tc.advance)
				}
			})
			shpEq(t, "local_check_ms", m["local_check_ms"], any(float64(tc.wantLocal)))
			shpEq(t, "elapsed_ms", m["elapsed_ms"], any(float64(tc.wantElapsed)))
			shpEq(t, "static_uncharged_ms", m["static_uncharged_ms"], any(float64(0)))
		})
	}
}

// TestShapeTimeLocalCheckOfThreeReadinessRounds: three rounds of 100 s with a 400 s readiness
// observation each reach a wall clock of 1500 s and 1200000 ms of local check time.
func TestShapeTimeLocalCheckOfThreeReadinessRounds(t *testing.T) {
	m := localCheckAccount(t, func(ev *Evidence, clock *shpClock) {
		for i := 0; i < 3; i++ {
			localCheckRound(ev, clock, 100000)
			b := ev.Millis()
			clock.advance(400000)
			ev.Interval("readiness", b, b+400000)
			ev.CheckSpan(b, b+400000)
		}
	})
	shpEq(t, "local_check_ms", m["local_check_ms"], any(float64(1200000)))
	shpEq(t, "elapsed_ms", m["elapsed_ms"], any(float64(1500000)))
}
