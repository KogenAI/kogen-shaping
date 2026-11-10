package shape

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

func refusedTechnicalQuestions(n int) []shpStep {
	var steps []shpStep
	for i := 0; i < n; i++ {
		steps = append(steps, shpSubmit(fmt.Sprint("q", i), map[string]any{"questions": []any{shpQuestion("Which engine?")}}), shpQuestionCheck(fmt.Sprint("qc", i), "technical"))
	}
	return steps
}

func TestShapeThreeRefusedQuestionsLeavePackageBudget(t *testing.T) {
	parallel(t)
	steps := refusedTechnicalQuestions(3)
	outside := preFiles(preFile("calc.js", "x"))
	steps = append(steps, shpSubmit("bad1", outside), shpSubmit("bad2", outside), shpSubmit("valid", shpFiles()), shpAck("ack"))
	f := newShpFx(t, steps...)
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("refused questions spent package attempts: exit %d: %#v", exit, err)
	}
	shpEq(t, "published complete package", f.read(refIntentPath), shpScopedIntentMD)
	shpEq(t, "refused questions", len(shpNamed(f.events(), "question_check")), 3)
}

func TestShapeMissingFilesSurviveUnrelatedInvalidSubmits(t *testing.T) {
	parallel(t)
	f := newShpFx(t,
		shpSubmit("missing", preFiles(preFile(preIntentPath, shpScopedIntentMD))),
		shpSubmit("unrelated1", preFiles(preFile("calc.js", "x"))),
		shpSubmit("unrelated2", preFiles(preFile("calc.js", "x"))), shpSubmit("unrelated3", preFiles(preFile("calc.js", "x"))))
	_, err := f.shape("Add calc.is_even(n).")
	e := shpWantError(t, err, "shape", "incomplete_package", "within 3 attempts", 1)
	shpContains(t, "actual unresolved feature", strings.Join(e.Lines, "\n"), "acceptance.feature does not exist")
	shpContains(t, "actual unresolved helper", strings.Join(e.Lines, "\n"), preStepGap)
	shapeAbsent(t, f)
}

func TestShapeRefusedQuestionsHaveSeparateFiniteBudget(t *testing.T) {
	parallel(t)
	f := newShpFx(t, refusedTechnicalQuestions(8)...)
	_, err := f.shape("Add calc.is_even(n).")
	e := shpWantError(t, err, "shape", "question_refusals", "8 refused question sets", 1)
	shpContains(t, "real refusal cause", strings.Join(e.Lines, "\n"), "question check refusals: technical x8")
	shpEq(t, "bounded question checks", len(qcChecks(f.Prov.requests())), 8)
	shapeAbsent(t, f)
}

func TestShapeExternalRefusedQuestionsLeavePackageBudget(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	q := map[string]any{"questions": []any{shpQuestion("Which engine?")}}
	check := result(map[string]any{"items": []any{map[string]any{"number": float64(1), "class": "technical", "why": "w"}}})
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(q), check, result(q), check, result(q), check, result(shpFiles())}}
	f.setReport("PASSED", 0)
	if exit, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("external question refusals spent package attempts: exit %d: %#v", exit, err)
	}
	shpEq(t, "external complete package", f.read(refIntentPath), shpScopedIntentMD)
	noNativeRequests(t, f)
}

func TestShapeResolvedGapsDoNotMaskLaterInvalidPackages(t *testing.T) {
	parallel(t)
	outside := preFiles(preFile("calc.js", "x"))
	f := newShpFx(t, shpSubmit("missing", preFiles(preFile(preIntentPath, shpScopedIntentMD))),
		shpAnswer(shpCall("complete", ProposeFilesTool, shpFiles())),
		shpSubmit("bad2", outside), shpSubmit("bad3", outside), shpSubmit("bad4", outside))
	_, err := f.shape("Add calc.is_even(n).")
	shpWantError(t, err, "shape", "invalid_output", "after 3 attempts", 1)
	shapeAbsent(t, f)
}
