package shape

import (
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

func TestSetupFailureOfReadsReadinessRefusal(t *testing.T) {
	e := sys.Fail("environment", "setup_failed", "Setup setup-2 failed (status=7).", 3)
	e.Lines = []string{"line 6", "line 7", "commands.setup[1]: make deps", "next: fix it"}
	f, ok := setupFailureOf(e)
	if !ok || f.status != 7 || f.command != "make deps" || f.output != "line 6\nline 7" {
		t.Fatalf("failure = %+v, %v", f, ok)
	}
	if _, ok := setupFailureOf(sys.Fail("environment", "tool_missing", "git is not available", 3)); ok {
		t.Fatal("another refusal is not a setup failure")
	}
}

func TestInitialSetupTimeIsNeitherElapsedNorChecks(t *testing.T) {
	var now time.Time
	clock := func() time.Time { return now }
	now = time.Unix(1000, 0)
	ev, err := NewEvidenceWithClock(t.TempDir(), "x", clock)
	if err != nil {
		t.Fatal(err)
	}
	ev.AccountTime()
	if err := ev.InitialSetup(func() error {
		return ev.Timed("setup", func() error { now = now.Add(90 * time.Second); return nil })
	}); err != nil {
		t.Fatal(err)
	}
	ev.ShaperRound(ev.Millis(), ev.Millis()+100000, nil)
	now = now.Add(100 * time.Second)
	if err := ev.Finish(); err != nil {
		t.Fatal(err)
	}
	a := ev.Spent()
	if a.Setup != 90000 || a.Elapsed != 100000 || a.Checks != 0 || a.Gap != 0 {
		t.Fatalf("account = %+v", *a)
	}
}

func TestSubtractSpans(t *testing.T) {
	got := subtractSpans([][2]int64{{0, 10}, {20, 30}}, [][2]int64{{5, 25}})
	if Union(got) != 10 || len(got) != 2 {
		t.Fatalf("spans = %v", got)
	}
}
