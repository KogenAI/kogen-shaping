package shape

import (
	"encoding/json"
	"strings"
	"testing"
)

// Readiness failures returned for acknowledgment are repairable: the first prose answer after
// them is followed by an unforced recovery inference with its own message and event.
func TestShapeRecoveryAfterReadinessFailures(t *testing.T) {
	parallel(t)
	w := responsesWire
	f := w.newShape(t, w.shapeSubmit("c1", shpFiles()), w.prose("The fixture needs investigation."), w.shapeAck("c2"))
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	reqs := f.Prov.requests()
	shpEq(t, "requests", len(reqs), 3)
	forceVerify(t, w, "shape", f.events(), reqs, true, nil, nil, [2]int{1, 0})
	forceSameTools(t, reqs, 2)
	events := shpNamed(f.events(), "recovery_turn")
	shpEq(t, "recovery events", len(events), 1)
	want := map[string]any{"event": "recovery_turn", "owner": "shape", "request": float64(3), "attempt": float64(1), "cause": "readiness_failures"}
	got, _ := json.Marshal(events[0])
	wantJSON, _ := json.Marshal(want)
	shpEq(t, "event", string(got), string(wantJSON))
	message := "Recovery before submitting again: submission attempt 1 returned readiness failures for acknowledgment (see .kogen/local/shaping/is-even/observations-1.txt) and nothing was published. " +
		"Your accumulated proposals are kept, as the cumulative proposal manifest lists. This inference does not require submit: investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, then call submit with the repaired package, or acknowledge legitimate missing-feature red as the readiness feedback describes."
	raws := f.Prov.rawRequests()
	enc, _ := json.Marshal(message)
	for i, raw := range raws {
		has := strings.Contains(string(raw.body), strings.Trim(string(enc), `"`))
		shpEq(t, "message in request "+string(rune('1'+i)), has, i == 2)
		shpEq(t, "reminder in request "+string(rune('1'+i)), strings.Contains(string(raw.body), shapeReminder), false)
	}
}

// Identical refusals add no earliest block, and neither do refusals before readiness failures.
func TestShapeStopsAddEarliestRefusalOnlyWhenItDiffers(t *testing.T) {
	parallel(t)
	w := responsesWire
	intentOnly := map[string]any{"files": shpFiles()["files"].([]any)[:1]}
	unsourced := shpFilesWithDecisions("- Non-integers raise TypeError.")
	both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
	t.Run("identical refusals", func(t *testing.T) {
		parallel(t)
		f := w.newShape(t, w.shapeSubmit("c1", unsourced), w.shapeSubmit("c2", unsourced), w.prose("a"), w.prose("b"), w.prose("Done."))
		_, err := f.shape("Add calc.is_even(n).")
		e := shpWantError(t, err, "shape", "invalid_output", "that submission (attempt 2) was refused at gate intent", 1)
		shpEq(t, "lines", strings.Join(e.Lines, "|"), "the shaper ended without calling submit|last refusal (attempt 2, gate intent):|decisions: a decision has no source|last tool-free answer: Done.")
		shpContains(t, "feedback", shpOutputs(f.Prov.requests()[2])["c2"], "\nUnchanged: this package is identical to the one refused at attempt 1;")
		shpEq(t, "recovery events", len(shpNamed(f.events(), "recovery_turn")), 0)
	})
	t.Run("refusals before readiness failures", func(t *testing.T) {
		parallel(t)
		f := w.newShape(t, w.shapeSubmit("c1", intentOnly), w.shapeSubmit("c2", shpFiles()), w.shapeSubmit("c3", both), w.prose("a"), w.prose("b"), w.prose("Done."))
		_, err := f.shape("Add calc.is_even(n).")
		e := shpWantError(t, err, "shape", "invalid_output", "that submission (attempt 3) was refused at gate validation", 1)
		shpEq(t, "lines", strings.Join(e.Lines, "|"), "the shaper ended without calling submit|last refusal (attempt 3, gate validation):|"+
			"submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report|received: questions (list of 1), files (list of 3)|last tool-free answer: Done.")
	})
}
