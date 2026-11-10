package shape

import (
	"context"
	"fmt"
	"github.com/KogenAI/kogen-executor/executor"
	"testing"
)

func retryShape(t *testing.T, limit, invalid int, success bool) {
	t.Helper()
	outside := map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
	var steps []shpStep
	for i := 0; i < invalid; i++ {
		steps = append(steps, shpSubmit(fmt.Sprint(i), outside))
	}
	if success {
		steps = append(steps, shpSubmit("valid", shpFiles()), shpAck("ack"))
	}
	f := newShpFx(t, steps...)
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "Add calc.is_even(n).", Client: f.Client, SubmissionRetryLimit: limit})
	if success {
		if err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpEq(t, "published", f.read(refIntentPath), shpScopedIntentMD)
		shpEq(t, "requests", len(f.Prov.requests()), invalid+2)
	} else {
		effective := submissionRetryLimit(limit)
		e := shpWantError(t, err, "shape", "invalid_output", "", 1)
		shpEq(t, "detail", e.Detail, fmt.Sprintf("the shaper's output was not a valid Intent after %d attempts", effective))
		shpEq(t, "exit", exit, 1)
		shpEq(t, "requests", len(f.Prov.requests()), effective)
		shapeAbsent(t, f)
	}
	refused := shpNamed(f.events(), "submission_refused")
	shpEq(t, "refusals", len(refused), invalid)
	for i, e := range refused {
		shpEq(t, "gate", e["gate"], any("validation"))
		shpEq(t, "consecutive", e["consecutive"], any(float64(i+1)))
		shpEq(t, "total", e["total"], any(float64(i+1)))
	}
	events := shpNamed(f.events(), "submission_retry_limit")
	effective := submissionRetryLimit(limit)
	if effective == 3 {
		shpEq(t, "events", len(events), 0)
	} else {
		shpEq(t, "events", len(events), 1)
		shpEq(t, "limit", events[0]["limit"], any(float64(effective)))
	}
}
func TestSubmissionRetryDefault(t *testing.T) {
	parallel(t)
	retryShape(t, 0, 2, true)
	retryShape(t, 0, 3, false)
}
func TestSubmissionRetryOne(t *testing.T) {
	parallel(t)
	retryShape(t, 1, 1, false)
	t.Run("title fallback", func(t *testing.T) {
		parallel(t)
		const long = "Add an is-even check for integers and also reject every string"
		f := newShpFx(t, shpSubmit("one", shpTitled(long, "")))
		f.setReport("PASSED", 0)
		f.newSession()
		exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "Add calc.is_even(n).", Client: f.Client, SubmissionRetryLimit: 1})
		if err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpEq(t, "requests", len(f.Prov.requests()), 1)
		shpEq(t, "title events", len(shpNamed(f.events(), "title_shortened")), 1)
	})
}
func TestSubmissionRetryFive(t *testing.T) {
	parallel(t)
	retryShape(t, 5, 4, true)
	retryShape(t, 5, 5, false)
}
func TestSubmissionRetryEvidence(t *testing.T) {
	parallel(t)
	for _, n := range []int{0, 3, 1, 5, -1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			parallel(t)
			retryShape(t, n, submissionRetryLimit(n), false)
		})
	}
}
func TestSubmissionRetryRefine(t *testing.T) {
	parallel(t)
	for _, n := range []int{0, 1, 3, 5} {
		for _, success := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%v", n, success), func(t *testing.T) {
				parallel(t)
				limit := submissionRetryLimit(n)
				count := limit
				if success {
					count--
				}
				var steps []shpStep
				for i := 0; i < count; i++ {
					steps = append(steps, shpAnswer(shpCall(fmt.Sprint("i", i), "submit_refinement", map[string]any{"files": []any{}})))
				}
				if success {
					steps = append(steps, refSubmit("valid", refIntentPath, refTitled))
				}
				f := newRefFx(t, refOpts{}, steps...)
				before := f.pkg()
				f.newSession()
				exit, err := Refine(context.Background(), f.S, RefineInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Hash: f.hash0, Request: "Clarify existing draft", Client: f.Client, SubmissionRetryLimit: n})
				if success {
					if err != nil || exit != 0 {
						t.Fatalf("exit %d: %v", exit, err)
					}
					refEqual(t, "package", f.pkg(), refWith(before, refIntentPath, refTitled))
				} else {
					e := shpWantError(t, err, "shape", "refinement_invalid", "", 1)
					shpEq(t, "detail", e.Detail, fmt.Sprintf("no valid ready correction after %d submissions; the original draft is preserved", limit))
					refEqual(t, "preserved", f.pkg(), before)
				}
				shpEq(t, "requests", len(f.Prov.requests()), count+boolInt(success))
				refused := shpNamed(f.evidenceEvents(), "submission_refused")
				shpEq(t, "refusals", len(refused), count)
				for i, e := range refused {
					shpEq(t, "gate", e["gate"], any("validation"))
					shpEq(t, "consecutive", e["consecutive"], any(float64(i+1)))
					shpEq(t, "total", e["total"], any(float64(i+1)))
				}
				ev := shpNamed(f.evidenceEvents(), "submission_retry_limit")
				if limit == 3 {
					shpEq(t, "events", len(ev), 0)
				} else {
					shpEq(t, "events", len(ev), 1)
					shpEq(t, "limit", ev[0]["limit"], any(float64(limit)))
				}
			})
		}
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func TestSubmissionRetryExternal(t *testing.T) {
	parallel(t)
	outside := map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
	t.Run("shape", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(outside)}}
		f.newSession()
		_, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "x", External: x.external(), SubmissionRetryLimit: 1})
		shpWantError(t, err, "shape", "invalid_output", "after 1 attempts", 1)
		shpEq(t, "rounds", x.calls, 1)
		ev := shpNamed(f.events(), "submission_refused")
		shpEq(t, "refusals", len(ev), 1)
		shpEq(t, "gate", ev[0]["gate"], any("validation"))
		shpEq(t, "consecutive", ev[0]["consecutive"], any(float64(1)))
		shpEq(t, "total", ev[0]["total"], any(float64(1)))
		shapeAbsent(t, f)
	})
	t.Run("refine", func(t *testing.T) {
		parallel(t)
		f := newRefFx(t, refOpts{external: true})
		before := f.pkg()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(map[string]any{"files": []any{}})}}
		f.newSession()
		_, err := Refine(context.Background(), f.S, RefineInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Hash: f.hash0, Request: "x", External: x.external(), SubmissionRetryLimit: 1})
		shpWantError(t, err, "shape", "refinement_invalid", "after 1 submissions", 1)
		shpEq(t, "rounds", x.calls, 1)
		ev := shpNamed(f.evidenceEvents(), "submission_refused")
		shpEq(t, "refusals", len(ev), 1)
		shpEq(t, "gate", ev[0]["gate"], any("validation"))
		shpEq(t, "consecutive", ev[0]["consecutive"], any(float64(1)))
		shpEq(t, "total", ev[0]["total"], any(float64(1)))
		refEqual(t, "preserved", f.pkg(), before)
	})
}

func TestRetryGateProgressShapeConversation(t *testing.T) {
	badDecision := dlAsk("Which wording?", map[string]any{"decision": "unsupported"})
	outside := map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
	for _, external := range []bool{false, true} {
		for _, totalCap := range []bool{false, true} {
			t.Run(fmt.Sprintf("external=%v/cap=%v", external, totalCap), func(t *testing.T) {
				gates := []string{"decision_record", "validation", "decision_record"}
				if totalCap {
					gates = []string{"decision_record", "validation", "decision_record", "validation", "decision_record", "validation"}
				}
				var steps []shpStep
				var rounds []func(int, string) (executor.RoundResult, error)
				for i, g := range gates {
					v := outside
					if g == "decision_record" {
						v = badDecision
					}
					steps = append(steps, shpSubmit(fmt.Sprint(i), v))
					rounds = append(rounds, result(v))
				}
				if !totalCap {
					steps = append(steps, shpSubmit("valid", shpFiles()), shpAck("ack"))
					rounds = append(rounds, result(shpFiles()), result(shpFiles()))
				}
				f := newShpFx(t, steps...)
				f.newSession()
				in := ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "Add calc.is_even(n).", Client: f.Client}
				var x *shapeExec
				if external {
					f.useExternalShape()
					x = &shapeExec{t: t, rounds: rounds}
					in.External = x.external()
				}
				exit, err := Shape(context.Background(), f.S, in)
				if totalCap {
					shpWantError(t, err, "shape", "invalid_output", "after 3 attempts", 1)
					shpEq(t, "exit", exit, 1)
				} else if err != nil || exit != 0 {
					t.Fatalf("exit %d: %v", exit, err)
				}
				events := shpNamed(f.events(), "submission_refused")
				shpEq(t, "refusals", len(events), len(gates))
				for i, g := range gates {
					shpEq(t, "gate", events[i]["gate"], any(g))
					shpEq(t, "consecutive", events[i]["consecutive"], any(float64(1)))
					shpEq(t, "total", events[i]["total"], any(float64(i+1)))
				}
			})
		}
	}
}

func TestRetryGateTotalCapRefineConversation(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprintf("external=%v", external), func(t *testing.T) {
			invalid := map[string]any{"files": []any{}}
			candidate := map[string]any{"files": []any{map[string]any{"path": refIntentPath, "content": refTitled}}}
			var steps []shpStep
			var rounds []func(int, string) (executor.RoundResult, error)
			for i := 0; i < 6; i++ {
				v := invalid
				if i%2 == 1 {
					v = candidate
				}
				steps = append(steps, shpAnswer(shpCall(fmt.Sprint(i), "submit_refinement", v)))
				rounds = append(rounds, result(v))
			}
			f := newRefFx(t, refOpts{external: external}, steps...)
			f.unwrittenReport()
			before := f.pkg()
			var exit int
			var err error
			if external {
				x := &shapeExec{t: t, rounds: rounds}
				exit, err = f.refineWith(x, f.hash0)
				shpEq(t, "rounds", x.calls, 6)
			} else {
				exit, err = f.refine(f.hash0)
				shpEq(t, "requests", len(f.Prov.requests()), 6)
			}
			shpWantError(t, err, "shape", "refinement_invalid", "after 3 submissions", 1)
			shpEq(t, "exit", exit, 1)
			refEqual(t, "preserved", f.pkg(), before)
			events := shpNamed(f.evidenceEvents(), "submission_refused")
			shpEq(t, "refusals", len(events), 6)
			for i, e := range events {
				gate := "validation"
				if i%2 == 1 {
					gate = "unwritten_tests"
				}
				shpEq(t, "gate", e["gate"], any(gate))
				shpEq(t, "consecutive", e["consecutive"], any(float64(1)))
				shpEq(t, "total", e["total"], any(float64(i+1)))
			}
		})
	}
}

func TestRetryReportedGateProgressToReady(t *testing.T) {
	bad := dlAsk("Which wording?", map[string]any{"decision": "missing evidence"})
	ghost := shpFilesWithDecisions("- Ghost (decision: D99, round 1) (source: request)")
	f := newShpFx(t, shpSubmit("one", bad), shpSubmit("two", ghost), shpSubmit("three", shpFiles()), shpSubmit("four", shpFiles()), shpAck("ack"))
	checks := 0
	f.Prov.route = func(req map[string]any) *shpStep {
		if decisionRequest(req) {
			checks++
			if checks == 1 {
				step := shpAnswer(shpCall("check", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": "- Non-integers raise TypeError. (source: request)", "request_quotes": []any{}, "verdict": "unsupported", "intent_value": "", "request_value": "", "difference": "", "why": "not established"}}}))
				return &step
			}
		}
		return nil
	}
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	events := shpNamed(f.events(), "submission_refused")
	gates := []string{"decision_record", "carried_decisions", "decision_check"}
	shpEq(t, "refusals", len(events), len(gates))
	for i, g := range gates {
		shpEq(t, "gate", events[i]["gate"], any(g))
		shpEq(t, "consecutive", events[i]["consecutive"], any(float64(1)))
		shpEq(t, "total", events[i]["total"], any(float64(i+1)))
	}
}

func TestRetryTotalCapPreservesTitleFallback(t *testing.T) {
	const long = "Add an is-even check for integers and also reject every string"
	bad := dlAsk("Which wording?", map[string]any{"decision": "unsupported"})
	outside := map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}}
	f := newShpFx(t, shpSubmit("1", bad), shpSubmit("2", outside), shpSubmit("3", bad), shpSubmit("4", outside), shpSubmit("5", bad), shpSubmit("6", shpTitled(long, "")), shpAck("ack"))
	f.setReport("PASSED", 0)
	exit, err := f.shape("Add calc.is_even(n).")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	shpEq(t, "title fallback", len(shpNamed(f.events(), "title_shortened")), 1)
	shpEq(t, "refusals", len(shpNamed(f.events(), "submission_refused")), 5)
}
