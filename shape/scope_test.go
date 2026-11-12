package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/agent"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/tools"

	"github.com/KogenAI/kogen-coordinator/project"
)

const scopeBT = "`"

func scopeBase(intent string, head map[string]int, ceiling int) scopeInput {
	in := scopeInput{
		pkg:       []scopeFile{{path: ".kogen/intents/x/intent.md", size: 100}, {path: ".kogen/intents/x/acceptance.feature", size: 50}, {path: "features/steps/x__steps.js", size: 50}},
		inputs:    []scopeFile{{path: scopeReadme, size: head[scopeReadme]}, {path: scopeRules, size: 30}},
		head:      head,
		scenarios: 3, harnesses: 1, ceiling: ceiling,
	}
	in.intentText, in.parts = scopeSplit(intent)
	return in
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
	// Package 200, README 1000, testing rules 30 and a changed file of 4000: 5230 bytes.
	in := scopeBase("Change "+scopeBT+"a/f.js"+scopeBT+".", map[string]int{"a/f.js": 4000, scopeReadme: 1000}, 5230)
	shpEq(t, "estimate", in.estimate(), 5230)
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("estimate 5230 against 5230: %v", lines)
	}
	in.ceiling = 5229
	lines := scopeLines(in)
	want := []string{
		"Scope check: the Intent is estimated at 5230 bytes of mandatory review material (scenarios 3, acceptance harnesses 1, changed files 1 in areas 1), over the 5229-byte review ceiling (tools.InputBudgetBytes).",
		"- .kogen/intents/x/intent.md: 100 bytes (Intent package)",
		"- .kogen/intents/x/acceptance.feature: 50 bytes (Intent package)",
		"- features/steps/x__steps.js: 50 bytes (Intent package)",
		"- README.md: 1000 bytes (review input)",
		"- docs/testing-rules.md: 30 bytes (review input)",
		"- a/f.js: 4000 bytes (changed file, area a)",
		scopeGuidance,
	}
	shpEq(t, "lines", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	// Without a README at HEAD it counts 0.
	in = scopeBase("No files.", map[string]int{}, 5230)
	shpEq(t, "no README", in.estimate(), 230)
}

func TestScopeChangedFilesAreCountedByTheirRules(t *testing.T) {
	head := map[string]int{"a/f.js": 1, "b/g.js": 1, "root.js": 1, "x/dir/h.js": 1, ".kogen/p.txt": 1, "features/steps/x__steps.js": 1, "keep.js": 1, scopeReadme: 1}
	text := strings.Join([]string{
		"`b/g.js` `a/f.js` `b/g.js` `missing.js` `x/dir` `.kogen/p.txt` `features/steps/x__steps.js` `.kogen/intents/x/intent.md` `README.md`",
		"`root.js` `keep.js and more` `x/dir/h.js`",
		"```",
		"keep.js",
		"```",
		"`unterminated.js",
		"`a/f.js\n`",
		"## Split",
		"- other-part: change `keep.js`",
	}, "\n")
	in := scopeBase(text, head, 1)
	shpEq(t, "named", scopeNames(in.changedFiles(in.intentText)), "b/g.js,a/f.js,root.js,x/dir/h.js")
	shpEq(t, "part", scopeNames(in.changedFiles(in.parts[0].text)), "keep.js")
	lines := scopeLines(in)
	shpContains(t, "counts", lines[0], "changed files 4 in areas 4)")
	shpContains(t, "root area", strings.Join(lines, "\n"), "- root.js: 1 bytes (changed file, area .)")
	shpEq(t, "README is not a changed file", strings.Contains(strings.Join(lines, "\n"), "(changed file, area .)\n- README"), false)
	intentUnit, _, _ := strings.Cut(strings.Join(lines, "\n"), "Scope check: part ")
	shpEq(t, "Split file is not the Intent's", strings.Contains(intentUnit, "keep.js"), false)
}

func TestScopeAreasAreDistinctDirectories(t *testing.T) {
	head := map[string]int{"a/f.js": 1, "a/g.js": 1, "a/b/h.js": 1, "r.js": 1}
	in := scopeBase(strings.Join([]string{"`a/f.js`", "`a/g.js`", "`a/b/h.js`", "`r.js`"}, " "), head, 1)
	shpContains(t, "areas", scopeLines(in)[0], "changed files 4 in areas 3)")
}

func TestScopeSplitSectionYieldsItsBullets(t *testing.T) {
	text := strings.Join([]string{
		"# Title",
		"## Split",
		"- part-one: delivers the first part",
		"prose is ignored",
		"- Bad_Slug: ignored",
		"- ab: slug too short",
		"* star-bullet: ignored",
		"- part-two: delivers `a/f.js`",
		"## Tests",
		"- later-one: not in the section",
	}, "\n")
	rest, parts := scopeSplit(text)
	shpEq(t, "parts", len(parts), 2)
	shpEq(t, "one", parts[0], scopePart{"part-one", "delivers the first part"})
	shpEq(t, "two", parts[1].slug, "part-two")
	shpEq(t, "rest keeps other sections", strings.Contains(rest, "later-one") && !strings.Contains(rest, "part-two"), true)
}

func TestScopePartEstimateIsTheReviewInputsPlusItsFilesAndAPartOverTheCeilingIsRefusedAlone(t *testing.T) {
	head := map[string]int{"a/f.js": 400, "a/g.js": 700, scopeReadme: 100}
	intent := "Change `a/f.js`.\n\n## Split\n- small-part: `a/f.js`\n- large-part: `a/f.js` and `a/g.js`\n"
	// Package 200, inputs 130, a/f.js 400: the Intent is 730; small-part is 530, large-part 1230.
	in := scopeBase(intent, head, 1000)
	shpEq(t, "intent", in.estimate(), 730)
	shpEq(t, "small", in.partEstimate(in.parts[0]), 530)
	shpEq(t, "large", in.partEstimate(in.parts[1]), 1230)
	lines := scopeLines(in)
	want := []string{
		`Scope check: part "large-part" is estimated at 1230 bytes of mandatory review material (changed files 2 in areas 1; its own package is not counted), over the 1000-byte review ceiling (tools.InputBudgetBytes).`,
		"- README.md: 100 bytes (review input)",
		"- docs/testing-rules.md: 30 bytes (review input)",
		"- a/f.js: 400 bytes (changed file, area a)",
		"- a/g.js: 700 bytes (changed file, area a)",
		scopeGuidance,
	}
	shpEq(t, "lines", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	in.ceiling = 729
	lines = scopeLines(in)
	shpContains(t, "Intent first", lines[0], "Scope check: the Intent is estimated at 730 bytes")
	shpEq(t, "units", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 2)
}

func TestScopeStagesDoNotDivideTheEstimate(t *testing.T) {
	head := map[string]int{"a/f.js": 400, "b/g.js": 400}
	intent := "Files `a/f.js` and `b/g.js`.\n\n## Stages\n\n- one: Change `a/f.js`\n- two: Change `b/g.js`\n"
	// Package 200, inputs 30, both files 800: 1030 whatever the stages say.
	in := scopeBase(intent, head, 1030)
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("fits: %v", lines)
	}
	in.ceiling = 1029
	lines := scopeLines(in)
	shpEq(t, "one unit", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 1)
	shpContains(t, "whole Intent", lines[0], "Scope check: the Intent is estimated at 1030 bytes")
	shpContains(t, "both files", lines[0], "changed files 2 in areas 2)")
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
	feature := shpFeature + "  Scenario Outline: Even <n>\n    Given is-even: the number <n>\n    Then is-even: it is even\n    Examples:\n      | n |\n      | 2 |\n      | 6 |\n      | 8 |\n" +
		"# " + strings.Repeat("x", tools.InputBudgetBytes) + "\n"
	f.write(project.IntentMDPath(shpSlug), intent)
	f.write(project.FeaturePath(shpSlug), feature)
	f.write("features/steps/is-even__steps.js", shpSteps)
	f.write(model, shpModel)
	ev, err := NewEvidence(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug)
	if err != nil {
		t.Fatal(err)
	}
	sh := &shaping{s: f.S, repo: f.Repo, root: f.Root, slug: shpSlug, ev: ev}
	text, out, refused := sh.scopeRefusal()
	shpEq(t, "refused", refused, true)
	shpEq(t, "kind", out.kind, outInvalid)
	readme, err := os.ReadFile(filepath.Join(f.Root, "README.md"))
	if err != nil {
		readme = nil
	}
	rules := len(agent.Prompt(agent.TestingRules))
	estimate := len(intent) + len(feature) + len(shpSteps) + len(shpModel) + len(readme) + rules + scopeChangedFileMaxBytes
	want := []string{
		fmt.Sprintf("Scope check: the Intent is estimated at %d bytes of mandatory review material (scenarios 5, acceptance harnesses 1, changed files 1 in areas 1), over the %d-byte review ceiling (tools.InputBudgetBytes).", estimate, tools.InputBudgetBytes),
		fmt.Sprintf("- %s: %d bytes (Intent package)", project.IntentMDPath(shpSlug), len(intent)),
		fmt.Sprintf("- %s: %d bytes (Intent package)", project.FeaturePath(shpSlug), len(feature)),
		fmt.Sprintf("- features/steps/is-even__steps.js: %d bytes (Intent package)", len(shpSteps)),
		fmt.Sprintf("- %s: %d bytes (Intent package)", model, len(shpModel)),
		fmt.Sprintf("- README.md: %d bytes (review input)", len(readme)),
		fmt.Sprintf("- docs/testing-rules.md: %d bytes (review input)", rules),
		fmt.Sprintf("- big.js: %d bytes (changed file, area .; diff hunks of a %d-byte file)", scopeChangedFileMaxBytes, tools.InputBudgetBytes),
		scopeGuidance,
	}
	shpEq(t, "refusal", text, strings.Join(want, "\n"))
}

func TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	bigFile := "//" + strings.Repeat("x", tools.InputBudgetBytes) + "\n"
	f.write("big.js", bigFile)
	f.git("add", "-A")
	f.git("commit", "-qm", "big file")
	f.useExternalShape()
	intent := strings.Replace(shpScopedIntentMD, "calc.is_even(n) returns", "calc.is_even(n) in "+scopeBT+"big.js"+scopeBT+" returns", 1)
	feature := shpFeature + "# " + strings.Repeat("x", tools.InputBudgetBytes) + "\n"
	files := shpFiles()
	files["files"].([]any)[0].(map[string]any)["content"] = intent
	files["files"].([]any)[1].(map[string]any)["content"] = feature
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(files)}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n).")
	e := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
	shpEq(t, "exit", exit, 1)
	shpContains(t, "problems", strings.Join(e.Lines, "\n"), "Scope check: the Intent is estimated at")
	shpEq(t, "shaper rounds", x.calls, 3)
	shpEq(t, "decision rounds", x.decisionCalls, 0)
	shpEq(t, "contract rounds", x.contractCalls, 0)
	shpEq(t, "inputs", len(x.inputs), 3)
	feedback := func(i int) string {
		text, _, _ := strings.Cut(x.inputs[i].Feedback, "\n\nCumulative proposal manifest: ")
		return text
	}
	shpContains(t, "first", feedback(1), "Scope check: the Intent is estimated at ")
	shpContains(t, "changed file", feedback(1), fmt.Sprintf("- big.js: %d bytes (changed file, area .; diff hunks of a %d-byte file)", scopeChangedFileMaxBytes, len(bigFile)))
	shpContains(t, "guidance", feedback(1), scopeGuidance)
	if strings.Contains(feedback(1), "Unchanged:") {
		t.Fatal("first refusal has the Unchanged note")
	}
	shpEq(t, "second", strings.HasSuffix(feedback(2), unchangedLine(1)), true)
	events := shpNamed(f.events(), "submission_refused")
	shpEq(t, "refused events", len(events), 3)
	for _, ev := range events {
		shpEq(t, "gate", ev["gate"], any("scope"))
	}
}

func TestScopeLargeChangedFileIsCountedAsItsHunksNotWhole(t *testing.T) {
	// The 221 KB README is a review input, sent whole; the Intent changes 3 lines of it and of
	// three other large files, which review sends as diff hunks.
	head := map[string]int{scopeReadme: 221304, "priv/manifest.json": 262578, "cli/mcp.go": 53179, "docs/big.md": 100000}
	intent := "Change 3 lines of " + scopeBT + "README.md" + scopeBT + ", " + scopeBT + "priv/manifest.json" + scopeBT + ", " +
		scopeBT + "cli/mcp.go" + scopeBT + " and " + scopeBT + "docs/big.md" + scopeBT + "."
	in := scopeBase(intent, head, tools.InputBudgetBytes)
	want := 200 + 221304 + 30 + 3*scopeChangedFileMaxBytes
	shpEq(t, "estimate", in.estimate(), want)
	if in.estimate() > tools.InputBudgetBytes*7/10 {
		t.Fatalf("estimate %d is not well under the %d ceiling", in.estimate(), tools.InputBudgetBytes)
	}
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("refused: %v", lines)
	}
}

func TestScopeSmallChangedFileIsStillCountedWhole(t *testing.T) {
	head := map[string]int{"a/f.js": scopeChangedFileMaxBytes, "a/g.js": 1234, scopeReadme: 10}
	in := scopeBase("Change "+scopeBT+"a/f.js"+scopeBT+" and "+scopeBT+"a/g.js"+scopeBT+".", head, 1<<30)
	shpEq(t, "estimate", in.estimate(), 200+10+30+scopeChangedFileMaxBytes+1234)
}

func TestScopeRefusalStillFiresWhenTheRealReviewMaterialIsOverTheCeiling(t *testing.T) {
	head := map[string]int{scopeReadme: 221304}
	var names []string
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("d/f%d.json", i)
		head[name] = 262578
		names = append(names, scopeBT+name+scopeBT)
	}
	in := scopeBase("Change "+strings.Join(names, " ")+".", head, tools.InputBudgetBytes)
	lines := scopeLines(in)
	if lines == nil {
		t.Fatalf("estimate %d not refused", in.estimate())
	}
	shpContains(t, "refusal", lines[0], "over the 524288-byte review ceiling")
	shpContains(t, "counted at its hunks", strings.Join(lines, "\n"), fmt.Sprintf("- d/f0.json: %d bytes (changed file, area d; diff hunks of a 262578-byte file)", scopeChangedFileMaxBytes))
}
