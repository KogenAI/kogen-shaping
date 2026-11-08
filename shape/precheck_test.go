package shape

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

const (
	preIntentPath  = ".kogen/intents/is-even/intent.md"
	preFeaturePath = ".kogen/intents/is-even/acceptance.feature"
	preStepGap     = "acceptance: features/steps/is-even__steps.js does not exist"
)

func preFile(path, content string) any { return map[string]any{"path": path, "content": content} }

func preFiles(files ...any) map[string]any { return map[string]any{"files": files} }

func TestShapeDeclaredFileGapsAreAllNamed(t *testing.T) {
	parallel(t)
	const model = ".kogen/intents/is-even/model.qnt"
	intent := strings.Replace(shpScopedIntentMD, "Model: not applicable — a pure function without state",
		"Model: "+model+" invariants=Holds seed=1 samples=10 steps=5", 1)
	intent = strings.Replace(intent, "- features/steps/is-even__steps.js\n", "- features/steps/is-even__steps.js\n- features/steps/is-even__more.js\n- features/steps/is-even__steps.js\n", 1)
	outside := preFiles(preFile("calc.js", "x"))
	f := newShpFx(t,
		shpSubmit("c1", preFiles(preFile(preIntentPath, intent))),
		shpSubmit("c2", preFiles(preFile(preFeaturePath, shpFeature))),
		shpSubmit("c3", outside), shpSubmit("c4", outside))
	exit, err := f.shape("Add calc.is_even(n).")
	shpWantError(t, err, "shape", "invalid_output", "after 3 attempts", 1)
	shpEq(t, "exit", exit, 1)
	reqs := f.Prov.requests()
	gaps := []string{"acceptance.feature does not exist", preStepGap, "acceptance: features/steps/is-even__more.js does not exist", "model: " + model + " does not exist"}
	shpEq(t, "refusal", shpOutputs(reqs[1])["c1"], "invalid submission:\n"+strings.Join(gaps, "\n")+"\nEarlier well-formed proposals are retained; submit only corrections.")
	shpEq(t, "precheck", shpOutputs(reqs[2])["c2"], "not submitted: the package still has these gaps, and this call was not counted as a submission attempt:\n"+
		strings.Join(gaps[1:], "\n")+"\nThe files of this call were retained with your earlier proposals. Add what is missing with propose_files; the next submission counts as an attempt.")
	shpEq(t, "refused", strings.Join(shpRefused(f), " "), "1:intent 2:validation 3:validation")
	pre := shpNamed(f.events(), "submission_precheck")
	shpEq(t, "prechecks", len(pre), 1)
	shpEq(t, "attempt", pre[0]["attempt"], any(float64(1)))
	if !f.exists(".kogen/local/shaping/is-even/proposed/" + preFeaturePath) {
		t.Fatal("the files of a pre-checked call are kept with the proposals")
	}
}

func TestShapeIncompleteBudgetEndsBlocked(t *testing.T) {
	parallel(t)
	pkg := preFiles(preFile(preIntentPath, shpScopedIntentMD), preFile(preFeaturePath, shpFeature))
	f := newShpFx(t, shpSubmit("c1", pkg), shpSubmit("c2", pkg), shpSubmit("c3", pkg), shpSubmit("c4", pkg), shpSubmit("c5", pkg))
	exit, err := f.shape("Add calc.is_even(n).")
	e := shpWantError(t, err, "shape", "incomplete_package", "blocked: the shaper could not complete the package within 3 attempts", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "lines", strings.Join(e.Lines, "|"), "unresolved gap (attempt 3, gate intent):|"+preStepGap)
	reqs := f.Prov.requests()
	shpEq(t, "requests", len(reqs), 5)
	shpEq(t, "counted", len(shpNamed(f.events(), "submission_refused")), 3)
	shpEq(t, "pre-checks", len(shpNamed(f.events(), "submission_precheck")), 2)
	shpContains(t, "same refusal", shpOutputs(reqs[3])["c3"], "invalid submission:\nThis is the same refusal as attempt 1; these items are unchanged:\n"+preStepGap)
	shpContains(t, "unchanged", shpOutputs(reqs[3])["c3"], "\nUnchanged: this package is identical to the one refused at attempt 1;")
	shpContains(t, "second pre-check", shpOutputs(reqs[4])["c4"], "not submitted: this is the same refusal as attempt 2; these items are unchanged,")
	shapeAbsent(t, f)
}

func TestShapeExternalPrecheckIsNotCounted(t *testing.T) {
	parallel(t)
	pkg := preFiles(preFile(preIntentPath, shpScopedIntentMD), preFile(preFeaturePath, shpFeature))
	t.Run("the step written afterwards is accepted", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			result(pkg), result(pkg), result(preFiles(preFile("features/steps/is-even__steps.js", shpSteps)))}}
		f.setReport("PASSED", 0)
		if exit, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d: %v", exit, err)
		}
		shpEq(t, "starts", len(x.starts), 1)
		shpEq(t, "rounds", len(x.inputs), 3)
		shpContains(t, "refusal", x.inputs[1].Feedback, "invalid submission:\n"+preStepGap)
		shpContains(t, "pre-check", x.inputs[2].Feedback, "not submitted: this is the same refusal as attempt 1; these items are unchanged, and this call was not counted as a submission attempt:\n"+preStepGap+"\nThe files of this call were retained")
		shpContains(t, "manifest", x.inputs[2].Feedback, "Cumulative proposal manifest:")
		shpEq(t, "refused", strings.Join(shpRefused(f), " "), "1:intent")
		shpEq(t, "pre-checks", len(shpNamed(f.events(), "submission_precheck")), 1)
		shpEq(t, "published intent", f.read(preIntentPath), shpScopedIntentMD)
		shpEq(t, "published feature", f.read(preFeaturePath), shpFeature)
		shpEq(t, "published steps", f.read("features/steps/is-even__steps.js"), shpSteps)
		noNativeRequests(t, f)
	})
	t.Run("pre-checks do not spend the budget", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(pkg)}}
		_, err := f.shapeWith(x, "Add calc.is_even(n).")
		shpWantError(t, err, "shape", "incomplete_package", "within 3 attempts", 1)
		shpEq(t, "rounds", len(x.inputs), 5)
		shpEq(t, "counted", len(shpNamed(f.events(), "submission_refused")), 3)
		shapeAbsent(t, f)
	})
}
