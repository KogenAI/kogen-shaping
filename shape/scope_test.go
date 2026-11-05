package shape

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/tools"

	"github.com/KogenAI/kogen-coordinator/project"
)

const scopeBT = "`"

func scopeBase(intent string, head map[string]int, bound int) scopeInput {
	return scopeInput{
		intentText: intent,
		pkg:        []scopeFile{{".kogen/intents/x/intent.md", 100}, {".kogen/intents/x/acceptance.feature", 50}, {"features/steps/x__steps.js", 50}},
		head:       head, scenarios: 3, harnesses: 1, bound: bound,
	}
}

func scopeNames(files []scopeFile) string {
	var p []string
	for _, f := range files {
		p = append(p, f.path)
	}
	return strings.Join(p, ",")
}

func TestScopeBoundIsTheReviewStagesSetting(t *testing.T) {
	if tools.InputBudgetBytes != 524288 {
		t.Fatalf("bound %d", tools.InputBudgetBytes)
	}
}

func TestScopeEstimateAtTheBoundFitsAndOneByteMoreIsRefused(t *testing.T) {
	in := scopeBase("Change "+scopeBT+"a/f.js"+scopeBT+".", map[string]int{"a/f.js": 300}, 500)
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("estimate 500 against 500: %v", lines)
	}
	in.head["a/f.js"] = 301
	lines := scopeLines(in)
	want := []string{
		"Scope check: the Intent is estimated at 501 bytes of one review (scenarios 3, acceptance harnesses 1, named files 1 in areas 1), over the 500-byte bound of one review (tools.InputBudgetBytes).",
		"- .kogen/intents/x/intent.md: 100 bytes (Intent package)",
		"- .kogen/intents/x/acceptance.feature: 50 bytes (Intent package)",
		"- features/steps/x__steps.js: 50 bytes (Intent package)",
		"- a/f.js: 301 bytes (named file, area a)",
		scopeGuidance,
	}
	shpEq(t, "lines", strings.Join(lines, "\n"), strings.Join(want, "\n"))
}

func TestScopeNamedFilesAreCountedByTheirRules(t *testing.T) {
	head := map[string]int{"a/f.js": 1, "b/g.js": 1, "root.js": 1, "x/dir/h.js": 1, ".kogen/p.txt": 1, "features/steps/x__steps.js": 1, "keep.js": 1}
	text := strings.Join([]string{
		"`b/g.js` `a/f.js` `b/g.js` `missing.js` `x/dir` `.kogen/p.txt` `features/steps/x__steps.js` `.kogen/intents/x/intent.md`",
		"`root.js` `keep.js and more` `x/dir/h.js`",
		"```",
		"keep.js",
		"```",
		"`unterminated.js",
		"`a/f.js\n`",
	}, "\n")
	in := scopeBase(text, head, 1)
	got := in.namedFiles()
	shpEq(t, "named", scopeNames(got), "b/g.js,a/f.js,root.js,x/dir/h.js")
	lines := scopeLines(in)
	shpContains(t, "counts", lines[0], "named files 4 in areas 4)")
	shpContains(t, "root area", strings.Join(lines, "\n"), "- root.js: 1 bytes (named file, area .)")
}

func TestScopeAreasAreDistinctDirectories(t *testing.T) {
	head := map[string]int{"a/f.js": 1, "a/g.js": 1, "a/b/h.js": 1, "r.js": 1}
	in := scopeBase(strings.Join([]string{"`a/f.js`", "`a/g.js`", "`a/b/h.js`", "`r.js`"}, " "), head, 1)
	shpContains(t, "areas", scopeLines(in)[0], "named files 4 in areas 3)")
}

func TestScopeStageUnits(t *testing.T) {
	head := map[string]int{"a/f.js": 200, "b/g.js": 200, "c/h.js": 100}
	intent := "Files `a/f.js`, `b/g.js` and `c/h.js`."
	stages := []project.Stage{
		{ID: "one", Goal: "Change `a/f.js`", Checks: []string{"true"}},
		{ID: "two", Goal: "Change `b/g.js` and `a/f.js`", Checks: []string{"true"}},
	}
	// package 200; stage one: a + unassigned c = 500; stage two: a + b + c = 700.
	in := scopeBase(intent, head, 600)
	in.stages = stages
	units := in.units()
	shpEq(t, "units", len(units), 2)
	shpEq(t, "one", scopeNames(units[0].named), "a/f.js,c/h.js")
	shpEq(t, "two", scopeNames(units[1].named), "a/f.js,b/g.js,c/h.js")
	lines := scopeLines(in)
	shpEq(t, "only stage two", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 1)
	shpContains(t, "stage two", lines[0], `Scope check: stage "two" is estimated at 700 bytes`)
	in.bound = 700
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("fits: %v", lines)
	}
	// With a plan the whole Intent (700 bytes in one unit) is not checked: stage one owns a,
	// stage two owns b, and each carries the unassigned c (500 bytes each).
	in.bound = 600
	in.stages = []project.Stage{
		{ID: "one", Goal: "Change `a/f.js`", Checks: []string{"true"}},
		{ID: "two", Goal: "Change `b/g.js`", Checks: []string{"true"}},
	}
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("whole Intent was checked: %v", lines)
	}
	in.stages = nil
	shpContains(t, "whole Intent without a plan", scopeLines(in)[0], "Scope check: the Intent is estimated at 700 bytes")
	in.stages = stages
	in.bound = 499
	lines = scopeLines(in)
	shpEq(t, "both stages", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 2)
}

func TestScopePackageCountsTheDeclaredModelAndOutlineRows(t *testing.T) {
	f := newShpFx(t)
	f.write("big.js", strings.Repeat("x", tools.InputBudgetBytes))
	f.git("add", "-A")
	f.git("commit", "-qm", "big file")
	model := project.ModelPath(shpSlug)
	intent := strings.Replace(shpScopedIntentMD, "Model: not applicable — a pure function without state",
		"Model: "+model+" invariants=Safety seed=7 samples=2 steps=2", 1)
	intent = strings.Replace(intent, "calc.is_even(n) returns", "calc.is_even(n) in "+scopeBT+"big.js"+scopeBT+" returns", 1)
	feature := shpFeature + "  Scenario Outline: Even <n>\n    Given is-even: the number <n>\n    Then is-even: it is even\n    Examples:\n      | n |\n      | 2 |\n      | 6 |\n      | 8 |\n"
	f.write(project.IntentMDPath(shpSlug), intent)
	f.write(project.FeaturePath(shpSlug), feature)
	f.write("features/steps/is-even__steps.js", shpSteps)
	f.write(model, shpModel)
	ev, err := NewEvidence(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug)
	if err != nil {
		t.Fatal(err)
	}
	sh := &shaping{repo: f.Repo, root: f.Root, slug: shpSlug, ev: ev}
	text, out, refused := sh.scopeRefusal()
	shpEq(t, "refused", refused, true)
	shpEq(t, "kind", out.kind, outInvalid)
	estimate := len(intent) + len(feature) + len(shpSteps) + len(shpModel) + tools.InputBudgetBytes
	want := []string{
		fmt.Sprintf("Scope check: the Intent is estimated at %d bytes of one review (scenarios 5, acceptance harnesses 1, named files 1 in areas 1), over the %d-byte bound of one review (tools.InputBudgetBytes).", estimate, tools.InputBudgetBytes),
		fmt.Sprintf("- %s: %d bytes (Intent package)", project.IntentMDPath(shpSlug), len(intent)),
		fmt.Sprintf("- %s: %d bytes (Intent package)", project.FeaturePath(shpSlug), len(feature)),
		fmt.Sprintf("- features/steps/is-even__steps.js: %d bytes (Intent package)", len(shpSteps)),
		fmt.Sprintf("- %s: %d bytes (Intent package)", model, len(shpModel)),
		fmt.Sprintf("- big.js: %d bytes (named file, area .)", tools.InputBudgetBytes),
		scopeGuidance,
	}
	shpEq(t, "refusal", text, strings.Join(want, "\n"))
}

func TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpSubmit("c2", shpFiles()), shpSubmit("c3", shpFiles()))
	f.write("big.js", "//"+strings.Repeat("x", tools.InputBudgetBytes)+"\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "big file")
	intent := strings.Replace(shpScopedIntentMD, "calc.is_even(n) returns", "calc.is_even(n) in "+scopeBT+"big.js"+scopeBT+" returns", 1)
	files := shpFiles()
	files["files"].([]any)[0].(map[string]any)["content"] = intent
	f.Prov.seq = []shpStep{shpSubmit("c1", files), shpSubmit("c2", files), shpSubmit("c3", files)}
	exit, err := f.shape("Add calc.is_even(n).")
	e := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
	shpEq(t, "exit", exit, 1)
	shpContains(t, "problems", strings.Join(e.Lines, "\n"), "Scope check: the Intent is estimated at")
	reqs := f.Prov.requests()
	shpEq(t, "shaper requests", len(reqs), 3)
	shpEq(t, "decision requests", len(f.Prov.decisionRequests()), 0)
	shpEq(t, "contract requests", len(f.Prov.contractRequests()), 0)
	first := unwrittenOutput(t, reqs[1], "c1")
	shpContains(t, "first", first, "- big.js: "+fmt.Sprint(tools.InputBudgetBytes+3)+" bytes (named file, area .)")
	shpContains(t, "guidance", first, scopeGuidance)
	if strings.Contains(first, "Unchanged:") {
		t.Fatal("first refusal has the Unchanged note")
	}
	shpEq(t, "second", strings.HasSuffix(unwrittenOutput(t, reqs[2], "c2"), unchangedLine(1)), true)
	events := shpNamed(f.events(), "submission_refused")
	shpEq(t, "refused events", len(events), 3)
	for _, ev := range events {
		shpEq(t, "gate", ev["gate"], any("scope"))
	}
}
