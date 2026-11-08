package shape

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
)

// The no-tool answer policy of native Shape and Refine: an answer with zero tool calls makes the
// next inference require a tool call at API level (where the route can), is
// counted apart from invalid submissions, and is bounded by its own ceiling. The tests run the
// real entry points against a scripted local provider.

const forceTarget = "any"

func forceTargetOf(owner string) string {
	if owner == "refine" {
		return "submit_refinement"
	}
	return forceTarget
}

func forceReminderOf(owner string) string {
	if owner == "refine" {
		return refineReminder
	}
	return shapeReminder
}

// forceEventsOf are the events of a fixture's invocation, whichever owner ran.
func forceEventsOf(f any) []map[string]any {
	switch x := f.(type) {
	case *refFx:
		return x.evidenceEvents()
	case *shpFx:
		return x.events()
	}
	panic("fixture")
}

// forceVerify checks the enforcement record against what the provider actually received: the
// requests at forcedAt require any Shape tool or the named Refine terminal tool (when supported), every other
// request keeps the ordinary automatic choice, each forced_submit_turn event describes its
// request with the counters at that time, and the exit event has the final counters.
func forceVerify(t *testing.T, w *forceWire, owner string, events, reqs []map[string]any, supported bool, forcedAt []int, counters [][2]int, final [2]int) {
	t.Helper()
	target := forceTargetOf(owner)
	forced := shpNamed(events, "forced_submit_turn")
	if len(forced) != len(forcedAt) || len(counters) != len(forcedAt) {
		t.Fatalf("forced_submit_turn events %v, want %d", forced, len(forcedAt))
	}
	at := map[int]bool{}
	for i, idx := range forcedAt {
		at[idx] = true
		e := forced[i]
		shpEq(t, "event owner", e["owner"], any(owner))
		shpEq(t, "event target", e["target"], any(target))
		shpEq(t, "event provider", e["provider"], any(w.provider))
		shpEq(t, "event api_forcing_supported", e["api_forcing_supported"], any(supported))
		shpEq(t, "event no_tool_turns", e["no_tool_turns"], any(float64(counters[i][0])))
		shpEq(t, "event invalid_submissions", e["invalid_submissions"], any(float64(counters[i][1])))
	}
	for i, req := range reqs {
		want := w.auto
		if supported && at[i] {
			if owner == "shape" {
				want = "required"
			} else {
				want = w.forced(target)
			}
		}
		if !reflect.DeepEqual(req["tool_choice"], want) {
			t.Fatalf("request %d tool_choice %v, want %v", i+1, req["tool_choice"], want)
		}
	}
	counts := shpNamed(events, "shaping_turn_counts")
	if len(counts) != 1 {
		t.Fatalf("shaping_turn_counts events %v", counts)
	}
	shpEq(t, "counts owner", counts[0]["owner"], any(owner))
	shpEq(t, "counts no_tool_turns", counts[0]["no_tool_turns"], any(float64(final[0])))
	shpEq(t, "counts invalid_submissions", counts[0]["invalid_submissions"], any(float64(final[1])))
}

// forceSameTools: the forced request keeps the complete tool set and the whole conversation of
// the request before it.
func forceSameTools(t *testing.T, reqs []map[string]any, i int) {
	t.Helper()
	if !reflect.DeepEqual(reqs[i]["tools"], reqs[0]["tools"]) {
		t.Fatalf("request %d tools differ from request 1", i+1)
	}
}

func forceHasReminder(t *testing.T, raws []shpRaw, i int, reminder string) {
	t.Helper()
	b, _ := json.Marshal(reminder)
	if !strings.Contains(string(raws[i].body), strings.Trim(string(b), `"`)) {
		t.Fatalf("request %d lacks the reminder %q", i+1, reminder)
	}
}

func TestShapeForceRecovery(t *testing.T) {
	parallel(t)
	for _, w := range []*forceWire{responsesWire, chatWire} {
		t.Run(w.name+"/shape", func(t *testing.T) {
			parallel(t)
			f := w.newShape(t, w.prose("The package is described above."), w.shapeSubmit("c1", shpFiles()), w.shapeAck("c2"))
			exit, err := f.shape("Add calc.is_even(n).")
			if err != nil || exit != 0 {
				t.Fatalf("exit %d: %v", exit, err)
			}
			shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
			reqs := f.Prov.requests()
			shpEq(t, "requests", len(reqs), 3)
			forceVerify(t, w, "shape", f.events(), reqs, true, []int{1}, [][2]int{{1, 0}}, [2]int{1, 0})
			forceSameTools(t, reqs, 1)
			forceHasReminder(t, f.Prov.rawRequests(), 1, shapeReminder)
		})
		t.Run(w.name+"/refine", func(t *testing.T) {
			parallel(t)
			f := w.newRefine(t, w.prose("The draft is fine."), w.refineSubmit("c1", refIntentPath, refTitled))
			before := f.pkg()
			exit, err := f.refine(f.hash0)
			if err != nil || exit != 0 {
				t.Fatalf("exit %d: %v", exit, err)
			}
			refEqual(t, "package", f.pkg(), refWith(before, refIntentPath, refTitled))
			reqs := f.Prov.requests()
			shpEq(t, "requests", len(reqs), 2)
			forceVerify(t, w, "refine", f.evidenceEvents(), reqs, true, []int{1}, [][2]int{{1, 0}}, [2]int{1, 0})
			forceSameTools(t, reqs, 1)
			forceHasReminder(t, f.Prov.rawRequests(), 1, refineReminder)
		})
	}
}

func TestShapeForceFallback(t *testing.T) {
	parallel(t)
	w := anthropicWire
	sameSettings := func(t *testing.T, reqs []map[string]any) {
		t.Helper()
		for _, k := range []string{"model", "thinking", "output_config", "tools"} {
			if !reflect.DeepEqual(reqs[0][k], reqs[1][k]) {
				t.Fatalf("%s changed for the reminder-only request: %v -> %v", k, reqs[0][k], reqs[1][k])
			}
		}
		if thinking, _ := reqs[1]["thinking"].(map[string]any); thinking["type"] != "adaptive" {
			t.Fatalf("thinking %v", reqs[1]["thinking"])
		}
	}
	t.Run("anthropic/shape", func(t *testing.T) {
		parallel(t)
		f := w.newShape(t, w.prose("Done."), w.shapeSubmit("c1", shpFiles()), w.shapeAck("c2"))
		exit, err := f.shape("Add calc.is_even(n).")
		if err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 3)
		forceVerify(t, w, "shape", f.events(), reqs, false, []int{1}, [][2]int{{1, 0}}, [2]int{1, 0})
		forceHasReminder(t, f.Prov.rawRequests(), 1, shapeReminder)
		sameSettings(t, reqs)
	})
	t.Run("anthropic/refine", func(t *testing.T) {
		parallel(t)
		f := w.newRefine(t, w.prose("Done."), w.refineSubmit("c1", refIntentPath, refTitled))
		before := f.pkg()
		exit, err := f.refine(f.hash0)
		if err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		refEqual(t, "package", f.pkg(), refWith(before, refIntentPath, refTitled))
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 2)
		forceVerify(t, w, "refine", f.evidenceEvents(), reqs, false, []int{1}, [][2]int{{1, 0}}, [2]int{1, 0})
		forceHasReminder(t, f.Prov.rawRequests(), 1, refineReminder)
		sameSettings(t, reqs)
	})
	// An external executor has no API tool choice: its invalid submission returns the existing
	// feedback, and no forced wire request is claimed.
	t.Run("external/shape", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(both), result(shpFiles())}}
		f.setReport("PASSED", 0)
		if exit, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpContains(t, "feedback", x.inputs[1].Feedback, "invalid submission:")
		shpEq(t, "forced events", len(shpNamed(f.events(), "forced_submit_turn")), 0)
		noNativeRequests(t, f)
	})
	t.Run("external/refine", func(t *testing.T) {
		parallel(t)
		f := newRefFx(t, refOpts{external: true})
		before := f.pkg()
		good := map[string]any{"files": []any{map[string]any{"path": refIntentPath, "content": refTitled}}}
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(map[string]any{"files": []any{}}), result(good)}}
		if exit, err := f.refineWith(x, f.hash0); err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		refText(t, "feedback", x.inputs[1].Feedback, "invalid: "+strings.TrimPrefix(refineInvalidShape, "invalid: "))
		refEqual(t, "package", f.pkg(), refWith(before, refIntentPath, refTitled))
		shpEq(t, "forced events", len(shpNamed(f.evidenceEvents(), "forced_submit_turn")), 0)
		noNativeRequests(t, f.shpFx)
	})
}

func TestShapeForceInvalidBudget(t *testing.T) {
	parallel(t)
	w := responsesWire
	t.Run("shape", func(t *testing.T) {
		parallel(t)
		outside := map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
		f := w.newShape(t, w.prose("Done."), w.shapeSubmit("c1", outside), w.shapeSubmit("c2", outside), w.shapeSubmit("c3", outside), w.shapeSubmit("c4", shpFiles()))
		exit, err := f.shape("Add calc.is_even(n).")
		shpWantError(t, err, "shape", "invalid_output", "the shaper's output was not a valid Intent after 3 attempts", 1)
		shpEq(t, "exit", exit, 1)
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 4)
		forceVerify(t, w, "shape", f.events(), reqs, true, []int{1}, [][2]int{{1, 0}}, [2]int{1, 3})
		shapeAbsent(t, f)
	})
	t.Run("refine", func(t *testing.T) {
		parallel(t)
		empty := func(id string) shpStep {
			return w.calls(shpCall(id, "submit_refinement", map[string]any{"files": []any{}}))
		}
		f := w.newRefine(t, w.prose("Done."), empty("c1"), empty("c2"), empty("c3"), w.refineSubmit("c4", refIntentPath, refTitled))
		before := f.pkg()
		exit, err := f.refine(f.hash0)
		shpWantError(t, err, "shape", "refinement_invalid", "no valid ready correction after 3 submissions; the original draft is preserved", 1)
		shpEq(t, "exit", exit, 1)
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 4)
		forceVerify(t, w, "refine", f.evidenceEvents(), reqs, true, []int{1}, [][2]int{{1, 0}}, [2]int{1, 3})
		refEqual(t, "preserved", f.pkg(), before)
	})
}

func TestShapeForceNoToolBound(t *testing.T) {
	parallel(t)
	w := responsesWire
	t.Run("shape", func(t *testing.T) {
		parallel(t)
		f := w.newShape(t, w.prose("one"), w.prose("two"), w.prose("three"), w.shapeSubmit("c1", shpFiles()))
		exit, err := f.shape("Add calc.is_even(n).")
		e := shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); no submission was made", 1)
		shpEq(t, "exit", exit, 1)
		shpEq(t, "problem", strings.Join(e.Lines, "|"), "the shaper ended without calling submit|last tool-free answer: three")
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 3)
		forceVerify(t, w, "shape", f.events(), reqs, true, []int{1, 2}, [][2]int{{1, 0}, {2, 0}}, [2]int{3, 0})
		shapeAbsent(t, f)
	})
	t.Run("refine", func(t *testing.T) {
		parallel(t)
		f := w.newRefine(t, w.prose("one"), w.prose("two"), w.prose("three"), w.refineSubmit("c1", refIntentPath, refTitled))
		before := f.pkg()
		exit, err := f.refine(f.hash0)
		shpWantError(t, err, "shape", "refinement_invalid", "the refiner answered without calling any tool 3 times (no-tool limit); no correction was submitted and the original draft is preserved", 1)
		shpEq(t, "exit", exit, 1)
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 3)
		forceVerify(t, w, "refine", f.evidenceEvents(), reqs, true, []int{1, 2}, [][2]int{{1, 0}, {2, 0}}, [2]int{3, 0})
		refEqual(t, "preserved", f.pkg(), before)
	})
	// Discovery without a submission is not a no-tool answer, and a submission starts a new
	// no-tool count: read, prose, refusal, prose (answered by the recovery inference), prose, then
	// a required tool call submits an accepted package.
	t.Run("a submission starts a new count", func(t *testing.T) {
		parallel(t)
		both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
		f := w.newShape(t, w.readReadme("r"), w.prose("one"), w.shapeSubmit("c1", both), w.prose("two"), w.prose("three"), w.shapeSubmit("c2", shpFiles()))
		f.setReport("PASSED", 0)
		exit, err := f.shape("Add calc.is_even(n).")
		if err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
		reqs := f.Prov.requests()
		shpEq(t, "requests", len(reqs), 6)
		forceVerify(t, w, "shape", f.events(), reqs, true, []int{2, 5}, [][2]int{{1, 0}, {3, 1}}, [2]int{3, 1})
	})
	// Prose answers between refused submissions still end: at most 7 tool-free answers and 9
	// requests at the default limit.
	t.Run("finite bound with refusals", func(t *testing.T) {
		parallel(t)
		outside := map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
		refusal := func(id string) shpStep { return w.shapeSubmit(id, outside) }
		f := w.newShape(t, w.prose("a"), w.prose("b"), refusal("c1"), w.prose("c"), w.prose("d"), refusal("c2"), w.prose("e"), w.prose("f"), w.prose("Last words.\nmore"))
		exit, err := f.shape("Add calc.is_even(n).")
		e := shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); its last submission (attempt 2) was refused at gate validation", 1)
		shpEq(t, "exit", exit, 1)
		shpEq(t, "requests", len(f.Prov.requests()), 9)
		shpEq(t, "lines", strings.Join(e.Lines, "|"), "the shaper ended without calling submit|last refusal (attempt 2, gate validation):|"+
			"calc.js: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/|last tool-free answer: Last words.")
		shapeAbsent(t, f)
	})
	// The stop after readiness failures names the unacknowledged submission and its file.
	t.Run("stop after readiness failures", func(t *testing.T) {
		parallel(t)
		f := w.newShape(t, w.shapeSubmit("c1", shpFiles()), w.prose("a"), w.prose("b"), w.prose("Done here."))
		exit, err := f.shape("Add calc.is_even(n).")
		e := shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); its last submission (attempt 1) returned readiness failures that were not acknowledged", 1)
		shpEq(t, "exit", exit, 1)
		shpEq(t, "lines", strings.Join(e.Lines, "|"), "the shaper ended without calling submit|last submission (attempt 1): readiness failures returned for acknowledgment, see .kogen/local/shaping/is-even/observations-1.txt|last tool-free answer: Done here.")
		shpEq(t, "requests", len(f.Prov.requests()), 4)
	})
	// Every tool-free answer is retained, redacted and bounded; an empty one has no text.
	t.Run("tool-free answers are recorded", func(t *testing.T) {
		parallel(t)
		long := "Long: " + strings.Repeat("é", 2100)
		reasoning := shpAnswer(map[string]any{"type": "reasoning", "id": "rs", "summary": []any{}})
		f := w.newShape(t, w.prose("  Plan: token=abc123secret then submit.\n"), reasoning, w.prose(long))
		_, err := f.shape("Add calc.is_even(n).")
		shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); no submission was made", 1)
		got := shpNamed(f.events(), "tool_free_answer")
		shpEq(t, "answers", len(got), 3)
		first := got[0]
		shpEq(t, "first text", first["text"], any("Plan: token=[REDACTED] then submit."))
		shpEq(t, "first chars", first["chars"], any(float64(len("Plan: token=[REDACTED] then submit."))))
		shpEq(t, "first request", first["request"], any(float64(1)))
		shpEq(t, "first owner", first["owner"], any("shape"))
		shpEq(t, "first truncated", first["truncated"], any(false))
		if _, ok := got[1]["text"]; ok {
			t.Fatalf("an empty answer has text: %v", got[1])
		}
		shpEq(t, "empty chars", got[1]["chars"], any(float64(0)))
		shpEq(t, "empty turns", got[1]["no_tool_turns"], any(float64(2)))
		shpEq(t, "long chars", got[2]["chars"], any(float64(2106)))
		shpEq(t, "long truncated", got[2]["truncated"], any(true))
		shpEq(t, "long text", len([]rune(got[2]["text"].(string))), 2000)
	})
	// A provider error exit keeps the final counters too.
	t.Run("provider error", func(t *testing.T) {
		parallel(t)
		f := w.newShape(t, w.prose("one"), shpStep{status: 400, code: "invalid_request_error"})
		_, err := f.shape("Add calc.is_even(n).")
		if e, ok := sys.AsError(err); !ok || e.Class != "shape" || e.Reason != "provider_failed" {
			t.Fatalf("error %v", err)
		}
		forceVerify(t, w, "shape", f.events(), f.Prov.requests(), true, []int{1}, [][2]int{{1, 0}}, [2]int{1, 0})
		r := w.newRefine(t, w.prose("one"), shpStep{status: 400, code: "invalid_request_error"})
		if _, err := r.refine(r.hash0); err == nil {
			t.Fatal("the provider error was swallowed")
		}
		forceVerify(t, w, "refine", r.evidenceEvents(), r.Prov.requests(), true, []int{1}, [][2]int{{1, 0}}, [2]int{1, 0})
	})
	forceResets(t, w)
}

// forceResetRoute scripts a conversation whose complete request reaches the input budget exactly
// at the inference after its tool-free answers: lead short prose answers follow the refusal (the
// first of them is answered by a recovery inference where the owner has one), padding reads grow
// it, then one long prose answer pushes the next request over, so the working-window reset falls
// between a no-tool answer and the request that must be forced.
func forceResetRoute(w *forceWire, refuse shpStep, pad string, lead int) func(map[string]any) *shpStep {
	const budget, prose = 524288, 100_000
	phase := 0
	return func(req map[string]any) *shpStep {
		raw, _ := json.Marshal(req)
		var step shpStep
		switch {
		case phase == 0:
			phase, step = 1, refuse
		case phase == 1 && lead > 0:
			lead--
			step = w.prose("lead prose")
		case phase == 1 && len(raw) < budget-prose+30_000:
			step = w.calls(shpCall("pad", "read_file", map[string]any{"path": pad}))
		case phase == 1:
			phase, step = 2, w.prose(strings.Repeat("long prose ", prose/11))
		default:
			step = w.prose("more prose")
		}
		return &step
	}
}

func forceCheckReset(t *testing.T, events, reqs []map[string]any, raws []shpRaw, owner string, noTool int) {
	t.Helper()
	resets := shpNamed(events, "working_context_reset")
	if len(resets) == 0 {
		t.Fatal("the conversation never reset its window")
	}
	// The forced request after the long prose answer is the one that carries the handoff.
	var handoffs []int
	for i, raw := range raws {
		if strings.Contains(string(raw.body), "Local working-window reset") {
			handoffs = append(handoffs, i)
		}
	}
	if len(handoffs) == 0 {
		t.Fatal("no request carries a reset handoff")
	}
	first := handoffs[0]
	if len(raws[first].body) > 524288 {
		t.Fatalf("the reset request is %d bytes", len(raws[first].body))
	}
	text := string(raws[first].body)
	for _, want := range []string{fmt.Sprintf(`\"no_tool_turns\":%d`, noTool), `\"forced_submit_pending\":\"` + forceTargetOf(owner) + `\"`, `\"invalid_submissions\":1`} {
		if !strings.Contains(text, want) {
			t.Fatalf("handoff of request %d lacks %s", first+1, want)
		}
	}
}

// forceResets runs the reset cases of TestShapeForceNoToolBound.
func forceResets(t *testing.T, w *forceWire) {
	t.Helper()
	t.Run("reset/shape", func(t *testing.T) {
		parallel(t)
		both := map[string]any{"questions": []any{shpQuestion("Q?")}, "files": shpFiles()["files"]}
		f := w.newShape(t)
		seedPadding(f)
		f.Prov.route = forceResetRoute(w, w.shapeSubmit("c0", both), "padding.txt", 1)
		_, err := f.shape("Add calc.is_even(n).")
		shpWantError(t, err, "shape", "invalid_output", "the shaper answered without calling any tool 3 consecutive times (no-tool limit); its last submission (attempt 1) was refused at gate validation", 1)
		reqs := f.Prov.requests()
		// The first prose answer after the refusal is followed by a recovery inference; the long
		// prose answer follows actual reads, so it starts a new idle streak. The reset
		// request stays forced; three consecutive idle answers, not the earlier prose, end the run.
		forced := len(reqs) - 2
		forceVerify(t, w, "shape", f.events(), reqs, true, []int{forced, forced + 1}, [][2]int{{2, 1}, {3, 1}}, [2]int{4, 1})
		forceCheckReset(t, f.events(), reqs, f.Prov.rawRequests(), "shape", 2)
	})
	t.Run("reset/refine", func(t *testing.T) {
		parallel(t)
		f := w.newRefine(t)
		seedPadding(f.shpFx)
		f.Prov.route = forceResetRoute(w, w.calls(shpCall("c0", "submit_refinement", map[string]any{"files": []any{}})), "padding.txt", 0)
		before := f.pkg()
		_, err := f.refine(f.hash0)
		shpWantError(t, err, "shape", "refinement_invalid", "the refiner answered without calling any tool 3 times (no-tool limit); no correction was submitted and the original draft is preserved", 1)
		reqs := f.Prov.requests()
		forced := len(reqs) - 2
		forceVerify(t, w, "refine", f.evidenceEvents(), reqs, true, []int{forced, forced + 1}, [][2]int{{1, 1}, {2, 1}}, [2]int{3, 1})
		forceCheckReset(t, f.evidenceEvents(), reqs, f.Prov.rawRequests(), "refine", 1)
		refEqual(t, "preserved", f.pkg(), before)
	})
}

// forceConversation is the conversation a request carries on its wire.
func forceConversation(req map[string]any) []any {
	for _, k := range []string{"input", "messages"} {
		if items, ok := req[k].([]any); ok {
			return items
		}
	}
	return nil
}

// forceRetained: the forced request carries every earlier item and the complete tool set.
func forceRetained(t *testing.T, reqs []map[string]any, i int) {
	t.Helper()
	before, after := forceConversation(reqs[i-1]), forceConversation(reqs[i])
	if len(after) <= len(before) || !reflect.DeepEqual(after[:len(before)], before) {
		t.Fatalf("request %d does not retain the conversation of request %d", i+1, i)
	}
	forceSameTools(t, reqs, i)
}

func TestShapeForceWireCompatibility(t *testing.T) {
	parallel(t)
	// The ordinary synthetic sessions are a deliberate wire compatibility baseline.
	golden := forceGolden(t)
	seen := 0
	actual := make(map[string][]forceCapture)
	for _, w := range []*forceWire{responsesWire, chatWire} {
		for _, owner := range []string{"shape", "refine"} {
			key := owner + "/" + w.name
			want, ok := golden[key]
			if !ok && !*updateGoldens {
				t.Fatalf("no golden for %s", key)
			}
			seen++
			got := forceOrdinary(t, w, owner)
			actual[key] = got
			if !*updateGoldens && !reflect.DeepEqual(got, want) {
				t.Errorf("%s: ordinary requests changed\n got %+v\nwant %+v", key, got, want)
			}
		}
	}
	if !*updateGoldens {
		shpEq(t, "golden cases", len(golden), seen)
	}
	if *updateGoldens && !t.Failed() {
		writeGolden(t, forceGoldenPath(), actual, "  ")
	}

	// Only the inference after a tool-free answer requires a tool call; discovery without
	// a submission does not, and the answer to a forced request restores the automatic choice.
	for _, w := range []*forceWire{responsesWire, chatWire} {
		t.Run(w.name+"/shape", func(t *testing.T) {
			parallel(t)
			f := w.newShape(t, w.readReadme("r"), w.prose("Thinking out loud."), w.shapeSubmit("c1", shpFiles()), w.shapeAck("c2"))
			if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
				t.Fatalf("exit %d: %v", exit, err)
			}
			reqs := f.Prov.requests()
			shpEq(t, "requests", len(reqs), 4)
			forceVerify(t, w, "shape", f.events(), reqs, true, []int{2}, [][2]int{{1, 0}}, [2]int{1, 0})
			forceRetained(t, reqs, 2)
			shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
		})
		t.Run(w.name+"/refine", func(t *testing.T) {
			parallel(t)
			empty := w.calls(shpCall("c1", "submit_refinement", map[string]any{"files": []any{}}))
			f := w.newRefine(t, w.readReadme("r"), w.prose("Thinking out loud."), empty, w.refineSubmit("c2", refIntentPath, refTitled))
			before := f.pkg()
			if exit, err := f.refine(f.hash0); err != nil || exit != 0 {
				t.Fatalf("exit %d: %v", exit, err)
			}
			reqs := f.Prov.requests()
			shpEq(t, "requests", len(reqs), 4)
			forceVerify(t, w, "refine", f.evidenceEvents(), reqs, true, []int{2}, [][2]int{{1, 0}}, [2]int{1, 1})
			forceRetained(t, reqs, 2)
			refEqual(t, "package", f.pkg(), refWith(before, refIntentPath, refTitled))
		})
	}
}
