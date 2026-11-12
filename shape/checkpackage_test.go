package shape

import (
	"fmt"
	"strings"
	"testing"
)

// cpBadIntent has an assumption marker that is not the canonical (assumption: ...).
var cpBadIntent = strings.NewReplacer(
	"- Non-integers raise TypeError. (source: request)", "- Non-integers raise TypeError. (assumption confirm the type) (source: request)",
).Replace(shpScopedIntentMD)

func cpCheck(id string, args map[string]any) shpStep {
	return shpAnswer(shpCall(id, CheckPackageTool, args))
}

func TestCheckPackageReturnsEveryDeterministicProblemWithoutARound(t *testing.T) {
	parallel(t)
	bad := map[string]any{"files": []any{
		pfFile(pfIntentPath, cpBadIntent),
		pfFile(pfFeaturePath, shpFeature),
		pfFile(pfStepsPath, shpSteps),
		pfFile(".kogen/intents/is-even/controls.json", "{not json"),
	}, "literals": []any{"ZZ-LITERAL-9"}}
	f := newShpFx(t,
		pfInspectAnd(cpCheck("c1", bad)),
		cpCheck("c2", map[string]any{}),
		cpCheck("c3", shpFiles()))
	noRound := func(n int) func(map[string]any) {
		return func(map[string]any) {
			if got := len(shpNamed(f.events(), "submission_refused")); got != 0 {
				t.Errorf("request %d: %d refusal(s) recorded by a lint", n, got)
			}
		}
	}
	f.watch(map[int]func(map[string]any){2: noRound(2), 3: noRound(3), 4: noRound(4)})
	// The script ends after three turns, so the run itself ends in a provider error; what is
	// checked is what the three lint calls returned and cost.
	f.shape("Add calc.is_even(n).")
	reqs := f.Prov.requests()
	if len(reqs) < 4 {
		t.Fatalf("requests %d, want 4 (three scripted turns and the one that found the script empty)\n%s", len(reqs), f.Stderr)
	}
	// The lint added no model request: nothing but the shaper's own turns reached the provider.
	if n := len(f.Prov.decisionRequests()) + len(f.Prov.literalRequests()) + len(f.Prov.contractRequests()); n != 0 {
		t.Fatalf("%d model check request(s) ran for a lint", n)
	}
	for i, r := range reqs[:3] {
		shpEq(t, fmt.Sprintf("request %d tools", i+1), strings.Join(shpToolNames(r), ", "), pfNativeTools)
	}

	got := shpOutputs(reqs[1])["c1"]
	for _, want := range []string{
		"not a submission attempt",
		"[decision_check]", assumptionRule, "- Non-integers raise TypeError. (assumption confirm the type) (source: request)",
		"[contract_review]", "[literal_check]", "ZZ-LITERAL-9",
		"Not checked here",
	} {
		shpContains(t, "check_package output", got, want)
	}
	if strings.Contains(got, "[step_ownership]") {
		t.Error("a step file submitted in the same call was reported as not submitted")
	}
	shpEq(t, "gates", strings.Count(got, "\n["), 3)
	// Two gates failed in one answer: all problems come at once.
	// Nothing of the call was retained, and nothing was written.
	shpContains(t, "empty lint", shpOutputs(reqs[2])["c2"], "no intent.md is proposed yet")
	shpEq(t, "retained", len(shpNamed(f.events(), "proposal_retained")), 0)
	if f.exists(pfIntentPath) {
		t.Error("the lint wrote into the checkout")
	}
	shpEq(t, "refusals", len(shpNamed(f.events(), "submission_refused")), 0)
}

func TestCheckPackageCleanPackage(t *testing.T) {
	parallel(t)
	f := newShpFx(t, pfInspectAnd(cpCheck("c1", shpFiles())))
	f.shape("Add calc.is_even(n).")
	got := shpOutputs(f.Prov.requests()[1])["c1"]
	shpContains(t, "clean output", got, "no deterministic problem found")
}
