package shape

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// controlsFakeClock replaces the shaping's clock with one that reads how many runs have started
// (the fake acceptance runner appends a line to Aux/trace per run) and adds the faked duration of
// each of them: seconds[i] is the time the run i takes; the last value repeats for later runs.
func controlsFakeClock(t *testing.T, f *shpFx, sh *shaping, seconds ...int) {
	t.Helper()
	base := time.Unix(1_800_000_000, 0)
	now := func() time.Time {
		raw, _ := os.ReadFile(filepath.Join(f.Aux, "trace"))
		n := strings.Count(string(raw), "\n")
		total := 0
		for i := 0; i < n; i++ {
			total += seconds[min(i, len(seconds)-1)]
		}
		return base.Add(time.Duration(total) * time.Second)
	}
	ev, err := NewEvidenceWithClock(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug, now)
	if err != nil {
		t.Fatal(err)
	}
	sh.ev = ev
}

// controlsTimeCapEvent is the last wrong_implementation_check event of the shaping.
func controlsTimeCapEvent(t *testing.T, f *shpFx) map[string]any {
	t.Helper()
	var last map[string]any
	for _, e := range readEvents(t, filepath.Join(f.Root, ".kogen/local/shaping", shpSlug)) {
		if e["event"] == "wrong_implementation_check" {
			last = e
		}
	}
	if last == nil {
		t.Fatal("no wrong_implementation_check event")
	}
	return last
}

func controlsManyWrongs(exact string, add, sub int) []controlsWrong {
	var out []controlsWrong
	for i := 0; i < add; i++ {
		out = append(out, controlsWrong{"add removed " + string(rune('a'+i)), "Add keeps working", exact, map[string]string{"calc.js": controlsCalc("sub")}})
	}
	for i := 0; i < sub; i++ {
		out = append(out, controlsWrong{"sub removed " + string(rune('a'+i)), "Sub keeps working", "Then is-even: sub of 5 and 3 is 2", map[string]string{"calc.js": controlsCalc("add")}})
	}
	return out
}

func TestControlsTimeCapEightRunsAtFiftySeconds(t *testing.T) {
	const exact = "Then is-even: add of 2 and 3 is 5"
	c := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: controlsManyWrongs(exact, 4, 3)}
	f, sh := controlsReal(t, c)
	controlsFakeClock(t, f, sh, 50)
	confirmed, err := sh.wrongImplementationCheck(context.Background(), "h")
	if err != nil {
		t.Fatal(err)
	}
	if len(confirmed.wrong) != 0 {
		t.Fatalf("refused: %q", confirmed.wrong)
	}
	if n := strings.Count(f.Stderr.String(), "kogen: wrong-implementation check: run "); n != 8 {
		t.Errorf("%d runs started, want 8", n)
	}
	e := controlsTimeCapEvent(t, f)
	if e["result"] != "clean" || e["runs"] != float64(8) || e["time_cap_seconds"] != float64(480) || e["planned_runs"] != float64(8) {
		t.Errorf("event = %v", e)
	}
}

func TestControlsTimeCapSmallCandidateKeepsBase(t *testing.T) {
	const exact = "Then is-even: add of 2 and 3 is 5"
	c := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: controlsManyWrongs(exact, 1, 1)}
	f, sh := controlsReal(t, c)
	// The runs take 100, 250 and 50 seconds; the cap of 300 is passed after the second.
	controlsFakeClock(t, f, sh, 100, 250, 50)
	confirmed, err := sh.wrongImplementationCheck(context.Background(), "h")
	if err != nil {
		t.Fatal(err)
	}
	want := "- time cap: the runs exceeded the total time cap of 300 seconds per candidate (3 planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after 2 of 3 runs completed; the check stopped. The completed runs took 175 seconds on average. Make the implementations or the steps faster."
	if len(confirmed.wrong) != 1 || confirmed.wrong[0] != want {
		t.Fatalf("refusal = %q", confirmed.wrong)
	}
	e := controlsTimeCapEvent(t, f)
	if e["result"] != "refused" || e["runs"] != float64(2) || e["time_cap_seconds"] != float64(300) || e["planned_runs"] != float64(3) {
		t.Errorf("event = %v", e)
	}
}

func TestControlsTimeCapSlowProjectMessage(t *testing.T) {
	const exact = "Then is-even: add of 2 and 3 is 5"
	c := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: controlsManyWrongs(exact, 4, 3)}
	f, sh := controlsReal(t, c)
	controlsFakeClock(t, f, sh, 300, 50)
	confirmed, err := sh.wrongImplementationCheck(context.Background(), "h")
	if err != nil {
		t.Fatal(err)
	}
	want := "- time cap: the runs exceeded the total time cap of 480 seconds per candidate (8 planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after 5 of 8 runs completed; the check stopped. The completed runs took 100 seconds on average. The project's own run is the slow part: the correct implementation's run alone took 300 seconds, more than half of the cap. Make the project's setup or acceptance command faster, or supply fewer wrong implementations."
	if len(confirmed.wrong) != 1 || confirmed.wrong[0] != want {
		t.Fatalf("refusal = %q", confirmed.wrong)
	}
	e := controlsTimeCapEvent(t, f)
	if e["time_cap_seconds"] != float64(480) || e["planned_runs"] != float64(8) {
		t.Errorf("event = %v", e)
	}
}

func TestControlsTimeCapKeepsFinishedVerdicts(t *testing.T) {
	const loose = "Then is-even: add of 2 and 3 gives an answer"
	wrong := append([]controlsWrong{{"add loose", "Add keeps working", loose, map[string]string{"calc.js": controlsCalc("sub")}}}, controlsManyWrongs(loose, 0, 4)...)
	c := controlsCandidate{feature: controlsCalcFeature(loose), steps: controlsExactDef + controlsLooseDef,
		correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: wrong}
	f, sh := controlsReal(t, c)
	controlsFakeClock(t, f, sh, 80)
	confirmed, err := sh.wrongImplementationCheck(context.Background(), "h")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"- time cap: the runs exceeded the total time cap of 360 seconds per candidate (6 planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after 5 of 6 runs completed; the check stopped. The completed runs took 80 seconds on average. Make the implementations or the steps faster.",
		`- add keeps working (kept) | request: add | problem: wrong implementation "add loose" passes scenario "Add keeps working"`,
	}
	if strings.Join(confirmed.wrong, "\n") != strings.Join(want, "\n") {
		t.Fatalf("refusal:\n%s\nwant:\n%s", strings.Join(confirmed.wrong, "\n"), strings.Join(want, "\n"))
	}
}

func TestControlsTimeCapEndingTheCorrectRunNamesTheProject(t *testing.T) {
	const exact = "Then is-even: add of 2 and 3 is 5"
	c := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: controlsManyWrongs(exact, 1, 1)}
	f, sh := controlsReal(t, c)
	// The clock stands still until the check has started its first run, then jumps past the cap.
	base, calls := time.Unix(1_800_000_000, 0), 0
	ev, err := NewEvidenceWithClock(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug, func() time.Time {
		calls++
		if calls <= 4 {
			return base
		}
		return base.Add(400 * time.Second)
	})
	if err != nil {
		t.Fatal(err)
	}
	sh.ev = ev
	started := 0
	lines, _, err := sh.controlsLines(context.Background(), sh.ready.Intent, mustScenarios(t, c.feature), strings.Split(c.feature, "\n"), sh.ready.Intent.StepFiles(),
		sh.checkedControlProps(mustScenarios(t, c.feature)), controlsTimeCap, &started)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "after 0 of 3 runs completed") ||
		!strings.Contains(lines[0], "the correct implementation's run alone took 400 seconds, more than half of the cap") {
		t.Fatalf("lines = %q", lines)
	}
}
