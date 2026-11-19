'use strict';
// The Go source of the shape-controls-time-cap-scales-with-runs acceptance harness, compiled into the
// shape package's tests through a Go overlay as
// shape/shape_controls_time_cap_scales_with_runs_acceptance_test.go (a path that never exists in the
// repository). It holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

// Each case builds the real-run fixture of the wrong-implementation check (controlsReal), replaces
// the shaping's clock with one that reads how many runs have started (the fake acceptance runner
// appends a line to Aux/trace per run) and adds the faked duration of the runs that started, calls
// wrongImplementationCheck and records what it showed as JSON in SCTC_OUT. It judges nothing.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sctcObserve(t *testing.T, c controlsCandidate, seconds []int) map[string]any {
	f, sh := controlsReal(t, c)
	base := time.Unix(1_800_000_000, 0)
	now := func() time.Time {
		raw, _ := os.ReadFile(filepath.Join(f.Aux, "trace"))
		n := strings.Count(string(raw), "\n")
		total := 0
		for i := 0; i < n && i < len(seconds); i++ {
			total += seconds[i]
		}
		return base.Add(time.Duration(total) * time.Second)
	}
	ev, err := NewEvidenceWithClock(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug, now)
	if err != nil {
		t.Fatal(err)
	}
	sh.ev = ev
	confirmed, err := sh.wrongImplementationCheck(context.Background(), "h-sctc")
	obs := map[string]any{"error": "", "refusal": append([]string{}, confirmed.wrong...),
		"runs": strings.Count(f.Stderr.String(), "kogen: wrong-implementation check: run ")}
	if err != nil {
		obs["error"] = err.Error()
	}
	obs["event_result"], obs["event_runs"], obs["time_cap_seconds"], obs["planned_runs"] = "", 0, 0, 0
	file, oerr := os.Open(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug, "events.ndjson"))
	if oerr == nil {
		defer file.Close()
		sc := bufio.NewScanner(file)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			var e map[string]any
			if json.Unmarshal(sc.Bytes(), &e) == nil && e["event"] == "wrong_implementation_check" {
				obs["event_result"], obs["event_runs"] = e["result"], e["runs"]
				if v, ok := e["time_cap_seconds"]; ok {
					obs["time_cap_seconds"] = v
				}
				if v, ok := e["planned_runs"]; ok {
					obs["planned_runs"] = v
				}
			}
		}
	}
	return obs
}

func sctcWrongs(addN, subN int) []controlsWrong {
	const exact, subStep = "Then is-even: add of 2 and 3 is 5", "Then is-even: sub of 5 and 3 is 2"
	var out []controlsWrong
	for i := 0; i < addN; i++ {
		out = append(out, controlsWrong{fmt.Sprintf("add removed %d", i), "Add keeps working", exact, map[string]string{"calc.js": controlsCalc("sub")}})
	}
	for i := 0; i < subN; i++ {
		out = append(out, controlsWrong{fmt.Sprintf("sub removed %d", i), "Sub keeps working", subStep, map[string]string{"calc.js": controlsCalc("add")}})
	}
	return out
}

func TestSctcAcceptance(t *testing.T) {
	out := os.Getenv("SCTC_OUT")
	if out == "" {
		t.Fatal("SCTC_OUT must be set")
	}
	const exact, loose = "Then is-even: add of 2 and 3 is 5", "Then is-even: add of 2 and 3 gives an answer"
	correct := map[string]string{"calc.js": controlsCalc("add", "sub")}
	obs := map[string]any{}

	obs["eight-at-fifty"] = sctcObserve(t, controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: correct, wrong: sctcWrongs(4, 3)}, []int{50, 50, 50, 50, 50, 50, 50, 50})

	obs["small"] = sctcObserve(t, controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: correct, wrong: sctcWrongs(1, 1)}, []int{100, 250, 50})

	obs["slow"] = sctcObserve(t, controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: correct, wrong: sctcWrongs(4, 3)}, []int{300, 50, 50, 50, 50, 50, 50, 50})

	verdicts := []controlsWrong{{"add loose", "Add keeps working", loose, map[string]string{"calc.js": controlsCalc("sub")}}}
	verdicts = append(verdicts, sctcWrongs(0, 4)...)
	obs["verdicts"] = sctcObserve(t, controlsCandidate{feature: controlsCalcFeature(loose), steps: controlsExactDef + controlsLooseDef,
		correct: correct, wrong: verdicts}, []int{80, 80, 80, 80, 80, 80})

	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
