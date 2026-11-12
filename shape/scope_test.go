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
	// Package 200, README 1000 and testing rules 30: 1230 bytes; a changed file of 4000 is not
	// counted. The review keeps 6000 bytes for source, so the limit is the ceiling less 6000.
	in := scopeBase("Change "+scopeBT+"a/f.js"+scopeBT+".", map[string]int{"a/f.js": 4000, scopeReadme: 1000}, 1230+scopeSourceRoom)
	shpEq(t, "estimate", in.estimate(), 1230)
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("estimate 1230 against 1230: %v", lines)
	}
	in.ceiling = 1229 + scopeSourceRoom
	lines := scopeLines(in)
	want := []string{
		"Scope check: the Intent is estimated at 1230 bytes of mandatory review material (scenarios 3, acceptance harnesses 1, changed files 1 in areas 1), over the 1229-byte limit (the 7229-byte review ceiling, tools.InputBudgetBytes, less the 6000 bytes the review keeps for source).",
		"- .kogen/intents/x/intent.md: 100 bytes (Intent package)",
		"- .kogen/intents/x/acceptance.feature: 50 bytes (Intent package)",
		"- features/steps/x__steps.js: 50 bytes (Intent package)",
		"- README.md: 1000 bytes (review input)",
		"- docs/testing-rules.md: 30 bytes (review input)",
		"- a/f.js: 4000 bytes (changed file, area a; review sends its diff in parts)",
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
	shpContains(t, "root area", strings.Join(lines, "\n"), "- root.js: 1 bytes (changed file, area .; review sends its diff in parts)")
	shpEq(t, "README is not a changed file", strings.Contains(strings.Join(lines, "\n"), "- README.md: 1 bytes (changed file"), false)
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

func TestScopePartEstimateIsTheReviewInputsAloneAndAPartOverTheLimitIsRefusedAlone(t *testing.T) {
	head := map[string]int{"a/f.js": 400, "a/g.js": 700, scopeReadme: 100}
	intent := "Change `a/f.js`.\n\n## Split\n- small-part: `a/f.js`\n- large-part: `a/f.js` and `a/g.js`\n"
	// Package 200, inputs 130: the Intent is 330 and every part 130, whatever files it changes.
	in := scopeBase(intent, head, 330+scopeSourceRoom)
	shpEq(t, "intent", in.estimate(), 330)
	shpEq(t, "small", in.partEstimate(in.parts[0]), 130)
	shpEq(t, "large", in.partEstimate(in.parts[1]), 130)
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("fits: %v", lines)
	}
	in.ceiling = 129 + scopeSourceRoom
	lines := scopeLines(in)
	shpContains(t, "Intent first", lines[0], "Scope check: the Intent is estimated at 330 bytes")
	shpEq(t, "units", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 3)
	shpContains(t, "part", strings.Join(lines, "\n"), `Scope check: part "large-part" is estimated at 130 bytes of mandatory review material (changed files 2 in areas 1; its own package is not counted), over the 129-byte limit`)
	// Only the inputs overrun: the parts are fine, the Intent with its package is not.
	in.ceiling = 229 + scopeSourceRoom
	lines = scopeLines(in)
	shpEq(t, "units", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 1)
}

func TestScopeStagesDoNotDivideTheEstimate(t *testing.T) {
	head := map[string]int{"a/f.js": 400, "b/g.js": 400}
	intent := "Files `a/f.js` and `b/g.js`.\n\n## Stages\n\n- one: Change `a/f.js`\n- two: Change `b/g.js`\n"
	// Package 200, inputs 30: 230 whatever the stages say.
	in := scopeBase(intent, head, 230+scopeSourceRoom)
	if lines := scopeLines(in); lines != nil {
		t.Fatalf("fits: %v", lines)
	}
	in.ceiling = 229 + scopeSourceRoom
	lines := scopeLines(in)
	shpEq(t, "one unit", strings.Count(strings.Join(lines, "\n"), "Scope check:"), 1)
	shpContains(t, "whole Intent", lines[0], "Scope check: the Intent is estimated at 230 bytes")
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
	estimate := len(intent) + len(feature) + len(shpSteps) + len(shpModel) + len(readme) + rules
	want := []string{
		fmt.Sprintf("Scope check: the Intent is estimated at %d bytes of mandatory review material (scenarios 5, acceptance harnesses 1, changed files 1 in areas 1), over the %d-byte limit (the %d-byte review ceiling, tools.InputBudgetBytes, less the %d bytes the review keeps for source).", estimate, tools.InputBudgetBytes-scopeSourceRoom, tools.InputBudgetBytes, scopeSourceRoom),
		fmt.Sprintf("- %s: %d bytes (Intent package)", project.IntentMDPath(shpSlug), len(intent)),
		fmt.Sprintf("- %s: %d bytes (Intent package)", project.FeaturePath(shpSlug), len(feature)),
		fmt.Sprintf("- features/steps/is-even__steps.js: %d bytes (Intent package)", len(shpSteps)),
		fmt.Sprintf("- %s: %d bytes (Intent package)", model, len(shpModel)),
		fmt.Sprintf("- README.md: %d bytes (review input)", len(readme)),
		fmt.Sprintf("- docs/testing-rules.md: %d bytes (review input)", rules),
		fmt.Sprintf("- big.js: %d bytes (changed file, area .; review sends its diff in parts)", tools.InputBudgetBytes),
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
	shpContains(t, "changed file", feedback(1), fmt.Sprintf("- big.js: %d bytes (changed file, area .; review sends its diff in parts)", len(bigFile)))
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

// The review cuts the source into as many parts as it takes (kogen-review review/parts.go: no part
// count, no review time bounds it) and one file's material across parts when it alone exceeds a
// part, so changed files never refuse an Intent; only authority that leaves under
// scopeSourceRoom bytes for source does.
func scopeFits(authority int, files map[string]int) bool {
	head := map[string]int{scopeReadme: authority - 230}
	var names []string
	for name, size := range files {
		head[name] = size
		names = append(names, scopeBT+name+scopeBT)
	}
	in := scopeBase("Change "+strings.Join(names, " ")+".", head, tools.InputBudgetBytes)
	return scopeLines(in) == nil
}

func TestScopeChangedFilesNeverRefuseHoweverLargeOrMany(t *testing.T) {
	// concise-intake: 855,868 bytes of review material over an authority of at most 331,570.
	shpEq(t, "concise-intake", scopeFits(331570, map[string]int{"a/big.js": 524288, "b/big.js": 400000}), true)
	// ops-state: 904,046 / 904,385 bytes of material over an authority of 260,963 / 261,302.
	shpEq(t, "ops-state-1", scopeFits(260963, map[string]int{"d/a.json": 643083}), true)
	shpEq(t, "ops-state-3", scopeFits(261302, map[string]int{"d/a.json": 643083}), true)
	// One file whose diff alone exceeds a whole part, and then a whole prompt.
	shpEq(t, "one file over a part", scopeFits(331570, map[string]int{"d/f.json": 2 * scopeSourceRoom}), true)
	shpEq(t, "one file over the ceiling", scopeFits(331570, map[string]int{"d/f.json": 4 * tools.InputBudgetBytes}), true)
	// Many large files still fit while they stay within MaxReviewParts: 8 files at the 32 KiB
	// allowance over room 32768 are 8 parts.
	eight := map[string]int{}
	for i := 0; i < 8; i++ {
		eight[fmt.Sprintf("d/f%d.json", i)] = 262578
	}
	shpEq(t, "8 parts", scopeFits(tools.InputBudgetBytes-(8*scopeChangedFileMaxBytes+7)/8, eight), true)
}

func TestScopeAuthorityMustLeaveTheReviewRoomForOnePartOfSource(t *testing.T) {
	limit := tools.InputBudgetBytes - scopeSourceRoom
	shpEq(t, "at the limit", scopeFits(limit, map[string]int{"a/f.js": 1}), true)
	shpEq(t, "one byte over", scopeFits(limit+1, map[string]int{"a/f.js": 1}), false)
	shpEq(t, "no files", scopeFits(limit+1, nil), false)
}

func TestScopeMoreThanMaxReviewPartsIsRefusedWithGuidanceToSplit(t *testing.T) {
	shpEq(t, "MaxReviewParts", MaxReviewParts, 8)
	// Authority 500000 leaves 24288 per part; six files at the 32 KiB allowance are 196608 bytes:
	// 9 parts. Five files are 163840: 7 parts.
	files := func(n int) map[string]int {
		m := map[string]int{}
		for i := 0; i < n; i++ {
			m[fmt.Sprintf("d/f%d.json", i)] = 262578
		}
		return m
	}
	shpEq(t, "7 parts", scopeFits(500000, files(5)), true)
	shpEq(t, "9 parts", scopeFits(500000, files(6)), false)
	head := files(6)
	head[scopeReadme] = 500000 - 230
	var names []string
	for n := range files(6) {
		names = append(names, scopeBT+n+scopeBT)
	}
	in := scopeBase("Change "+strings.Join(names, " ")+".", head, tools.InputBudgetBytes)
	shpEq(t, "estimate", in.reviewParts(in.estimate(), in.changedFiles(in.intentText)), 9)
	lines := scopeLines(in)
	shpEq(t, "refusal", lines[0], "Scope check: the Intent is estimated at 9 review parts (authority 500000 bytes in every part), over MaxReviewParts 8; each part is one review model call.")
	shpEq(t, "guidance", lines[len(lines)-1], scopePartsGuidance)
	// A split part of 9 parts is refused alone.
	in = scopeBase("Change `a/f.js`.\n\n## Split\n- big-part: "+strings.Join(names, " ")+"\n", head, tools.InputBudgetBytes)
	lines = scopeLines(in)
	shpContains(t, "part", strings.Join(lines, "\n"), `part "big-part" is estimated at 9 review parts`)
}
