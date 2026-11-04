package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// ---- controls.json ----

const controlsFeatureText = "Feature: Calc\n  Scenario: Add keeps working\n    Then is-even: add of 2 and 3 is 5\n    And is-even: sub of 5 and 3 is 2\n  Scenario Outline: Mul <a> times <b>\n    Then is-even: mul of <a> and <b> is <c>\n    Examples:\n      | a | b | c |\n      | 2 | 3 | 6 |\n      | 3 | 3 | 9 |\n"

func controlsParseFixture(t *testing.T) ([]project.Scenario, []string) {
	t.Helper()
	scenarios, err := project.ParseGherkin(controlsFeatureText)
	if err != nil {
		t.Fatal(err)
	}
	return scenarios, strings.Split(controlsFeatureText, "\n")
}

func controlsJSON(correct, wrong string) []byte {
	return []byte(`{"correct":{"files":` + correct + `},"wrong":[` + wrong + `]}`)
}

func TestParseControls(t *testing.T) {
	scenarios, feature := controlsParseFixture(t)
	rules := controlsRules{steps: "features/steps", acceptance: []string{".kogen/intents/is-even/acceptance.feature", "features/steps/is-even__steps.js"}, protect: []string{"guard.txt", "docs/**/*.md", "lib/"}}
	ok := `{"name":"add removed","scenario":"Add keeps working","step":"Then is-even: add of 2 and 3 is 5","files":{"calc.js":"x"}}`
	wrong := func(name, scenario, step, files string) string {
		return fmt.Sprintf(`{"name":%q,"scenario":%q,"step":%q,"files":%s}`, name, scenario, step, files)
	}
	big := strings.Repeat("x", controlsMaxBytes+1)
	many := `{"a":"1","b":"1","c":"1","d":"1","e":"1","f":"1","g":"1","h":"1","i":"1"}`
	tests := []struct {
		name string
		raw  []byte
		want []string
	}{
		{"valid", controlsJSON(`{"calc.js":"y"}`, ok), nil},
		{"an outline's example step with the outline's name", controlsJSON(`{"calc.js":"y"}`, wrong("m", " Mul <a>  times <b>", "Then is-even: mul of <a> and <b> is <c>", `{"calc.js":"z"}`)), nil},
		{"the correct implementation alone", controlsJSON(`{"calc.js":"y"}`, ""), nil},
		{"not json", []byte("{"), []string{"- controls.json: it is not valid JSON"}},
		{"not an object", []byte("[]"), []string{`- controls.json: it must be an object with exactly the fields "correct" and "wrong"`}},
		{"an extra field", []byte(`{"correct":{"files":{"a":"b"}},"wrong":[],"x":1}`), []string{`- controls.json: it must be an object with exactly the fields "correct" and "wrong"`}},
		{"wrong is not a list", []byte(`{"correct":{"files":{"a":"b"}},"wrong":{}}`), []string{`- controls.json: it must be an object with exactly the fields "correct" and "wrong"`}},
		{"correct with another field", []byte(`{"correct":{"files":{"a":"b"},"x":1},"wrong":[]}`), []string{`- controls.json: "correct" must be an object with exactly the field "files"`}},
		{"a wrong implementation without a field", controlsJSON(`{"a":"b"}`, `{"name":"n","scenario":"s","files":{"a":"b"}}`),
			[]string{`- controls.json: wrong implementation 1 must be an object with exactly the fields "name", "scenario", "step" and "files"`}},
		{"a blank and a repeated name", controlsJSON(`{"a":"b"}`, strings.Join([]string{
			wrong("  ", "Add keeps working", "Then is-even: add of 2 and 3 is 5", `{"a":"b"}`),
			wrong("same  name", "Add keeps working", "Then is-even: add of 2 and 3 is 5", `{"a":"b"}`),
			wrong("same name", "Add keeps working", "Then is-even: add of 2 and 3 is 5", `{"a":"b"}`)}, ",")), []string{
			"- controls.json: wrong implementation 1 needs a name that is not blank and differs from the others",
			"- controls.json: wrong implementation 2 needs a name that is not blank and differs from the others",
			"- controls.json: wrong implementation 3 needs a name that is not blank and differs from the others"}},
		{"no files", controlsJSON(`{}`, wrong("n", "Add keeps working", "Then is-even: add of 2 and 3 is 5", `{"a":1}`)), []string{
			`- controls.json: the correct implementation: "files" must map at least one path to its content`,
			`- controls.json: wrong implementation "n": "files" must map at least one path to its content`}},
		{"paths", controlsJSON(`{"/abs.js":"x","../up.js":"x","a/../b.js":"x"}`, ""), []string{
			"- controls.json: the correct implementation: ../up.js: a replacement path must be relative and normal",
			"- controls.json: the correct implementation: /abs.js: a replacement path must be relative and normal",
			"- controls.json: the correct implementation: a/../b.js: a replacement path must be relative and normal"}},
		{"machinery", controlsJSON(`{".kogen/x":"1","features/steps/y.js":"1","features/steps":"1",".kogen/intents/is-even/acceptance.feature":"1","guard.txt":"1","docs/a/b.md":"1","lib/x.js":"1","a/.git/c":"1","node_modules/m.js":"1","w/node_modules":"1","cucumber.js":"1","cucumber.yml":"1","sub/cucumber.js":"1","fine.js":"1"}`, ""), func() []string {
			var out []string
			for _, p := range []string{".kogen/intents/is-even/acceptance.feature", ".kogen/x", "a/.git/c", "cucumber.js", "cucumber.yml", "docs/a/b.md", "features/steps", "features/steps/y.js", "guard.txt", "lib/x.js", "node_modules/m.js", "w/node_modules"} {
				out = append(out, "- controls.json: the correct implementation: "+p+": a replacement may not write acceptance machinery (the .kogen/ and steps directories, the Acceptance: files, protected files, .git, node_modules or a Cucumber configuration file)")
			}
			return append(out, "- controls.json: the correct implementation: it has more than 8 files or 65536 bytes")
		}()},
		{"too many files", controlsJSON(many, ""), []string{"- controls.json: the correct implementation: it has more than 8 files or 65536 bytes"}},
		{"too many bytes", controlsJSON(`{"a.js":"`+big+`"}`, ""), []string{"- controls.json: the correct implementation: it has more than 8 files or 65536 bytes"}},
		{"the byte limit itself", controlsJSON(`{"a.js":"`+strings.Repeat("x", controlsMaxBytes)+`"}`, ""), nil},
		{"unknown scenario and step", controlsJSON(`{"a":"b"}`, strings.Join([]string{
			wrong("n1", "Missing", "Then is-even: add of 2 and 3 is 5", `{"a":"b"}`),
			wrong("n2", "Add keeps working", "Then is-even: mul of 2 and 3 is 6", `{"a":"b"}`)}, ",")), []string{
			`- controls.json: wrong implementation "n1": scenario "Missing" is not a scenario of acceptance.feature`,
			`- controls.json: wrong implementation "n2": step "Then is-even: mul of 2 and 3 is 6" is not a step of scenario "Add keeps working"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, got := parseControls(tt.raw, scenarios, feature, rules)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Fatalf("problems:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
			if (cf != nil) != (len(tt.want) == 0) {
				t.Fatalf("controls = %+v with problems %v", cf, got)
			}
		})
	}
}

// ---- the evidence rules ----

func controlsFailed(line int, kind, message, stack string) *controlsCase {
	return &controlsCase{steps: []controlsStep{{line: 3, status: "PASSED"}, {line: line, status: "FAILED", kind: kind, message: message, stack: stack}}}
}

func TestControlsJudge(t *testing.T) {
	scenarios, feature := controlsParseFixture(t)
	add := scenarios[0]
	w := controlsImpl{name: "add removed", scenario: "Add keeps working", step: "And is-even: sub of 5 and 3 is 2"}
	stepFiles := []string{"features/steps/is-even__steps.js"}
	atLine := controlsStepLines(add, w.step, feature)
	if len(atLine) != 1 || atLine[0] != 4 {
		t.Fatalf("step lines = %v", atLine)
	}
	const message = "Expected values to be strictly equal:\n\n3 !== 2\n"
	run := controlsRunResult{roots: []string{"/tmp/copy", "/private/tmp/copy"}}
	frame := func(path string) string {
		return "AssertionError: x\n    at World.<anonymous> (" + path + ":4:10)\n    at run (node:internal/x:1:1)"
	}
	tests := []struct {
		name     string
		c        *controlsCase
		wantRule int
		want     string
	}{
		{"an assertion in a step file", controlsFailed(4, "AssertionError", message, frame("/tmp/copy/features/steps/is-even__steps.js")), 0, ""},
		{"the resolved copy path", controlsFailed(4, "AssertionError", message, frame("/private/tmp/copy/features/steps/is-even__steps.js")), 0, ""},
		{"a file URL", controlsFailed(4, "AssertionError", message, "AssertionError: x\n  at file:///tmp/copy/features/steps/is-even__steps.js:4:10"), 0, ""},
		{"a frame without a function", controlsFailed(4, "AssertionError", message, "AssertionError: x\n    at /tmp/copy/features/steps/is-even__steps.js:4:10"), 0, ""},
		{"a passing scenario", &controlsCase{steps: []controlsStep{{line: 3, status: "PASSED"}, {line: 4, status: "PASSED"}}}, 2,
			`wrong implementation "add removed" passes scenario "Add keeps working"`},
		{"an earlier step", controlsFailed(3, "AssertionError", message, frame("/tmp/copy/features/steps/is-even__steps.js")), 3,
			`wrong implementation "add removed" fails scenario "Add keeps working" at "Then is-even: add of 2 and 3 is 5", not at "And is-even: sub of 5 and 3 is 2"`},
		{"an Error", controlsFailed(4, "Error", "boom\nmore", "Error: boom\n    at x (/tmp/copy/calc.js:1:1)"), 4,
			`wrong implementation "add removed" fails scenario "Add keeps working" without an assertion: Error: boom`},
		{"a step timeout", controlsFailed(4, "Error", "function timed out, ensure the promise resolves within 5000 milliseconds", ""), 4,
			`wrong implementation "add removed" fails scenario "Add keeps working" without an assertion: Error: function timed out, ensure the promise resolves within 5000 milliseconds`},
		{"a missing module", controlsFailed(4, "Error", "Cannot find module '../../calc'\nRequire stack:", "Error: Cannot find module"), 4,
			`wrong implementation "add removed" fails scenario "Add keeps working" without an assertion: Error: Cannot find module '../../calc'`},
		{"an assertion thrown by implementation code", controlsFailed(4, "AssertionError", message, frame("/tmp/copy/calc.js")), 5,
			`wrong implementation "add removed" fails scenario "Add keeps working" with an assertion thrown outside the Intent's step files, at calc.js:4:10`},
		{"an assertion thrown outside the copy", controlsFailed(4, "AssertionError", message, "AssertionError: x\n    at assert (node:assert:1:1)"), 5,
			`wrong implementation "add removed" fails scenario "Add keeps working" with an assertion thrown outside the Intent's step files, at node:assert:1:1`},
		{"an assertion without a stack", controlsFailed(4, "AssertionError", message, ""), 5,
			`wrong implementation "add removed" fails scenario "Add keeps working" with an assertion thrown outside the Intent's step files, at an unknown location`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, reason := controlsJudge(w, add, tt.c, atLine, feature, run, stepFiles)
			if rule != tt.wantRule || reason != tt.want {
				t.Fatalf("rule %d, %q; want %d, %q", rule, reason, tt.wantRule, tt.want)
			}
		})
	}
}

func passedCase(lines ...int) *controlsCase {
	c := &controlsCase{}
	for _, l := range lines {
		c.steps = append(c.steps, controlsStep{line: l, status: "PASSED"})
	}
	return c
}

func TestControlsJudgeAll(t *testing.T) {
	scenarios, feature := controlsParseFixture(t)
	// Add is scenario 0, the two examples of Mul are 1 and 2.
	props := []controlsProp{
		{report: propertyReport{property: "add", kind: "kept", request: "add"}, scenarios: []int{0}},
		{report: propertyReport{property: "mul", kind: "new", request: "mul"}, scenarios: []int{1, 2}},
	}
	cf := &controlsFile{wrong: []controlsImpl{
		{name: "w1", scenario: "Add keeps working", step: "Then is-even: add of 2 and 3 is 5"},
		{name: "w2", scenario: "Mul <a> times <b>", step: "Then is-even: mul of <a> and <b> is <c>"},
	}}
	step := "/tmp/c/features/steps/is-even__steps.js"
	failFirst := func(line int) *controlsCase {
		return &controlsCase{steps: []controlsStep{{line: line, status: "FAILED", kind: "AssertionError", message: "no", stack: "AssertionError: no\n    at W (" + step + ":1:1)"}}}
	}
	run := func(cases map[int]*controlsCase) controlsRunResult {
		return controlsRunResult{cases: cases, complete: true, roots: []string{"/tmp/c"}}
	}
	correct := run(map[int]*controlsCase{2: passedCase(3, 4), 9: passedCase(6), 10: passedCase(6)})
	stepFiles := []string{"features/steps/is-even__steps.js"}
	// w1 fails add at its first step; w2 fails both examples of mul at the outline's step.
	w1 := run(map[int]*controlsCase{2: failFirst(3), 9: passedCase(6), 10: passedCase(6)})
	w2 := run(map[int]*controlsCase{2: passedCase(3, 4), 9: failFirst(6), 10: failFirst(6)})
	if lines := controlsJudgeAll(props, cf, scenarios, feature, []controlsRunResult{correct, w1, w2}, stepFiles); len(lines) != 0 {
		t.Fatalf("lines = %v", lines)
	}
	t.Run("rule 6 names the first scenario of another property that fails", func(t *testing.T) {
		both := run(map[int]*controlsCase{2: failFirst(3), 9: passedCase(6), 10: failFirst(6)})
		lines := controlsJudgeAll(props, cf, scenarios, feature, []controlsRunResult{correct, both, w2}, stepFiles)
		want := `- add (kept) | request: add | problem: wrong implementation "w1" also fails scenario "Mul 3 times 3", which exercises another property and not this one`
		if len(lines) != 1 || lines[0] != want {
			t.Fatalf("lines = %q, want %q", lines, want)
		}
	})
	t.Run("an incomplete wrong run and a property no wrong implementation names", func(t *testing.T) {
		incomplete := controlsRunResult{complete: false}
		lines := controlsJudgeAll(props, cf, scenarios, feature, []controlsRunResult{correct, incomplete, w2}, stepFiles)
		want := `- add (kept) | request: add | problem: wrong implementation "w1": its run did not report every scenario as passed or failed`
		if len(lines) != 1 || lines[0] != want {
			t.Fatalf("lines = %q, want %q", lines, want)
		}
		lines = controlsJudgeAll(props, &controlsFile{wrong: cf.wrong[:1]}, scenarios, feature, []controlsRunResult{correct, w1}, stepFiles)
		want = "- mul (new) | request: mul | problem: no wrong implementation in controls.json names a scenario of it"
		if len(lines) != 1 || lines[0] != want {
			t.Fatalf("lines = %q, want %q", lines, want)
		}
	})
	t.Run("one example is enough for an outline", func(t *testing.T) {
		one := run(map[int]*controlsCase{2: passedCase(3, 4), 9: passedCase(6), 10: failFirst(6)})
		if lines := controlsJudgeAll(props, cf, scenarios, feature, []controlsRunResult{correct, w1, one}, stepFiles); len(lines) != 0 {
			t.Fatalf("lines = %v", lines)
		}
	})
}

func TestControlsUnreachable(t *testing.T) {
	scenarios, feature := controlsParseFixture(t)
	props := []controlsProp{
		{report: propertyReport{property: "add", kind: "kept", request: "add"}, scenarios: []int{0}},
		{report: propertyReport{property: "mul", kind: "new", request: "mul"}, scenarios: []int{1, 2}},
	}
	passed := map[int]*controlsCase{2: passedCase(3, 4), 9: passedCase(6), 10: passedCase(6)}
	if lines := controlsUnreachable(props, scenarios, feature, controlsRunResult{cases: passed, complete: true}); len(lines) != 0 {
		t.Fatalf("lines = %v", lines)
	}
	failing := map[int]*controlsCase{2: passedCase(3, 4), 9: passedCase(6),
		10: {steps: []controlsStep{{line: 6, status: "FAILED", kind: "Error", message: "boom\n    at x"}}}}
	lines := controlsUnreachable(props, scenarios, feature, controlsRunResult{cases: failing, complete: true})
	want := `- mul (new) | request: mul | problem: no pass is reachable: with the correct implementation in place, scenario "Mul 3 times 3" fails at "Then is-even: mul of <a> and <b> is <c>": boom`
	if len(lines) != 1 || lines[0] != want {
		t.Fatalf("lines = %q, want %q", lines, want)
	}
	lines = controlsUnreachable(props, scenarios, feature, controlsRunResult{complete: false})
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "problem: no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed") {
		t.Fatalf("lines = %q", lines)
	}
}

func TestControlsRefusalJoinsContractRefusal(t *testing.T) {
	c := contractConfirmed{contradictions: []string{"- c"}, environment: []string{"- e"}, properties: []string{"- p"}, wrong: []string{"- w1", "- w2"}, wrongHeadline: wrongHeadline(2, "is-even", 300)}
	if got := strings.Join(c.all(), ","); got != "- c,- e,- p,- w1,- w2" {
		t.Fatalf("all = %s", got)
	}
	text := contractFeedback(c)
	i := strings.Index(text, "Wrong-implementation check: 2 problem(s) with whether acceptance rejects a wrong implementation of each property the request asks for; nothing was published. ")
	if i < strings.Index(text, "Property check: ") || !strings.HasSuffix(text, "refused again.\n- w1\n- w2") {
		t.Fatalf("feedback = %s", text)
	}
	for _, want := range []string{" .kogen/intents/is-even/controls.json supplies"[1:], "at most 8 runs and 300 seconds per candidate"} {
		if !strings.Contains(text, want) {
			t.Errorf("feedback lacks %q", want)
		}
	}
	if (contractConfirmed{wrong: []string{"- w"}}).empty() {
		t.Error("a wrong-implementation refusal is empty")
	}
}

func TestControlsPreloadFreezesAssert(t *testing.T) {
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		var err error
		if node, err = exec.LookPath("node"); err != nil {
			t.Skip("no node")
		}
	}
	dir := t.TempDir()
	preload, script := filepath.Join(dir, "preload.js"), filepath.Join(dir, "main.js")
	os.WriteFile(preload, []byte(controlsPreload), 0o644)
	os.WriteFile(script, []byte("const strict = require('node:assert/strict');\nconst loose = require('node:assert');\n"+
		"for (const m of [strict, loose]) { m.strictEqual = () => {}; m.ok = () => {}; m.AssertionError.prototype.name = 'x'; }\n"+
		"try { strict.strictEqual(1, 2); console.log('tampered'); } catch (e) { console.log(e.name); }\n"+
		"try { loose.ok(false); console.log('tampered'); } catch (e) { console.log(e.name); }\n"), 0o644)
	out, err := exec.Command(node, "--require", preload, script).CombinedOutput()
	if err != nil || string(out) != "AssertionError\nAssertionError\n" {
		t.Fatalf("with the preload: %q, %v", out, err)
	}
	out, _ = exec.Command(node, script).CombinedOutput()
	if string(out) != "tampered\ntampered\n" {
		t.Fatalf("without the preload: %q", out)
	}
}

// ---- real runs ----

const controlsStepsHeader = "const {Then} = require('@cucumber/cucumber');\nconst assert = require('node:assert/strict');\n"

const controlsExactDef = "Then('is-even: {word} of {int} and {int} is {int}', function (op, a, b, want) {\n  assert.strictEqual(require('../../calc').calc(op, a, b), want);\n});\n"
const controlsLooseDef = "Then('is-even: add of {int} and {int} gives an answer', function (a, b) {\n  const answer = require('../../calc').calc('add', a, b);\n  assert.notStrictEqual(answer, null);\n});\n"
const controlsArityDef = "Then('is-even: calc takes {int} arguments', function (n) {\n  assert.strictEqual(require('../../calc').calc.length, n);\n});\n"

func controlsCalc(ops ...string) string {
	body := ""
	for _, op := range ops {
		switch op {
		case "add":
			body += "  if (op === 'add') return a + b;\n"
		case "sub":
			body += "  if (op === 'sub') return a - b;\n"
		}
	}
	return "exports.calc = (op, a, b) => {\n" + body + "  return undefined;\n};\n"
}

const controlsIntentText = "# Shape a synthetic change\n\nSlug: is-even\nModel: not applicable — a synthetic change without state\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" +
	"A synthetic change.\n\n## Decisions\n- Results are compared exactly. (source: request)\n\n## Source comparison\n- The request is covered by the feature.\n\n## Tests\n- The requested behaviour: the acceptance scenarios (scenario)\n\n## Feasibility\n- none\n"

type controlsWrong struct {
	Name     string            `json:"name"`
	Scenario string            `json:"scenario"`
	Step     string            `json:"step"`
	Files    map[string]string `json:"files"`
}

type controlsCandidate struct {
	feature, steps string
	correct        map[string]string
	wrong          []controlsWrong
}

// controlsReal is a shaping with a candidate on disk whose acceptance runs for real, with the
// properties "add keeps working" and "sub keeps working" reviewed.
func controlsReal(t *testing.T, c controlsCandidate) (*shpFx, *shaping) {
	t.Helper()
	modules := os.Getenv("KOGEN_CUCUMBER")
	if modules == "" {
		t.Skip("KOGEN_CUCUMBER is unset")
	}
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	f := newShpFx(t)
	f.write("calc.js", controlsCalc("add", "sub"))
	f.write("features/steps/calc_steps.js", "// existing steps\n")
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\necho run >> '%s/trace'\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n",
		f.Aux, modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
	f.git("add", "-A")
	f.git("commit", "-qm", "real acceptance")
	f.write(project.IntentMDPath(shpSlug), controlsIntentText)
	f.write(project.FeaturePath(shpSlug), c.feature)
	f.write("features/steps/is-even__steps.js", controlsStepsHeader+c.steps)
	raw, err := json.Marshal(map[string]any{"correct": map[string]any{"files": c.correct}, "wrong": append([]controlsWrong{}, c.wrong...)})
	if err != nil {
		t.Fatal(err)
	}
	f.write(project.IntentDir(shpSlug)+"/controls.json", string(raw))
	intent, _, err := project.CheckIntent(f.Root, shpSlug, nil)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := f.Repo.Tree(f.Repo.Head(), nil)
	if err != nil {
		t.Fatal(err)
	}
	sh := f.shaping()
	sh.rd = &verify.Readiness{}
	sh.ready = &verify.Ready{Intent: intent, CandidateTree: tree}
	sh.request = "Keep add and sub answering as they do now."
	prop := func(name, request, scenario string) propertyReport {
		return propertyReport{property: name, request: request, kind: "kept", observable: true, scenarios: []string{scenario}}
	}
	sh.reviewProps = []propertyReport{prop("add keeps working", "add", "Add keeps working"), prop("sub keeps working", "sub", "Sub keeps working")}
	return f, sh
}

func controlsCalcFeature(addSteps ...string) string {
	return "Feature: Calc\n  Scenario: Add keeps working\n    " + strings.Join(addSteps, "\n    ") + "\n  Scenario: Sub keeps working\n    Then is-even: sub of 5 and 3 is 2\n"
}

func controlsWrongs(addStep, addCalc string) []controlsWrong {
	return []controlsWrong{
		{"add removed", "Add keeps working", addStep, map[string]string{"calc.js": addCalc}},
		{"sub removed", "Sub keeps working", "Then is-even: sub of 5 and 3 is 2", map[string]string{"calc.js": controlsCalc("add")}},
	}
}

func TestWrongImplementationRealRuns(t *testing.T) {
	correct := map[string]string{"calc.js": controlsCalc("add", "sub")}
	const exact, loose = "Then is-even: add of 2 and 3 is 5", "Then is-even: add of 2 and 3 gives an answer"
	both := func(addCalc string) controlsCandidate {
		return controlsCandidate{feature: controlsCalcFeature(loose), steps: controlsExactDef + controlsLooseDef, correct: correct, wrong: controlsWrongs(loose, addCalc)}
	}
	subOnly := controlsCalc("sub")
	tests := []struct {
		name string
		c    controlsCandidate
		want []string // the refusal lines; nil: a clean check
		runs int
	}{
		{"acceptance that fails on each removal", controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef, correct: correct, wrong: controlsWrongs(exact, subOnly)}, nil, 3},
		{"an add step that passes whatever calc returns", both(subOnly),
			[]string{`- add keeps working (kept) | request: add | problem: wrong implementation "add removed" passes scenario "Add keeps working"`}, 3},
		{"a wrong implementation that throws its own AssertionError", both("const assert = require('node:assert');\nexports.calc = (op, a, b) => {\n  if (op === 'add') assert.fail('add was removed');\n  if (op === 'sub') return a - b;\n};\n"),
			[]string{`- add keeps working (kept) | request: add | problem: wrong implementation "add removed" fails scenario "Add keeps working" with an assertion thrown outside the Intent's step files, at calc.js:3:28`}, 3},
		{"a wrong implementation that overwrites an assertion", both("const strict = require('node:assert/strict');\nstrict.notStrictEqual = strict.fail;\n" + subOnly),
			[]string{`- add keeps working (kept) | request: add | problem: wrong implementation "add removed" passes scenario "Add keeps working"`}, 3},
		{"a failure at an earlier unrelated step", controlsCandidate{
			feature: controlsCalcFeature("Then is-even: calc takes 3 arguments", "And is-even: add of 2 and 3 gives an answer"),
			steps:   controlsExactDef + controlsLooseDef + controlsArityDef, correct: correct,
			wrong: controlsWrongs("And is-even: add of 2 and 3 gives an answer", "exports.calc = (op, ...args) => {\n  const [a, b] = args;\n  if (op === 'sub') return a - b;\n};\n")},
			[]string{`- add keeps working (kept) | request: add | problem: wrong implementation "add removed" fails scenario "Add keeps working" at "Then is-even: calc takes 3 arguments", not at "And is-even: add of 2 and 3 gives an answer"`}, 3},
		{"a correct implementation that disables assertions", controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
			correct: map[string]string{"calc.js": "const m = require('node:assert/strict');\nm.strictEqual = () => {};\nexports.calc = (op, a, b) => undefined;\n"},
			wrong:   controlsWrongs(exact, subOnly)},
			[]string{
				`- add keeps working (kept) | request: add | problem: no pass is reachable: with the correct implementation in place, scenario "Add keeps working" fails at "Then is-even: add of 2 and 3 is 5": Expected values to be strictly equal:`,
				`- sub keeps working (kept) | request: sub | problem: no pass is reachable: with the correct implementation in place, scenario "Sub keeps working" fails at "Then is-even: sub of 5 and 3 is 2": Expected values to be strictly equal:`}, 1},
		{"no wrong implementation", controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef, correct: correct},
			[]string{
				"- add keeps working (kept) | request: add | problem: no wrong implementation in controls.json names a scenario of it",
				"- sub keeps working (kept) | request: sub | problem: no wrong implementation in controls.json names a scenario of it"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, sh := controlsReal(t, tt.c)
			status, head := f.git("status", "--porcelain", "--ignored"), f.git("rev-parse", "HEAD")
			index := f.git("ls-files", "-s")
			started := 0
			lines, err := sh.controlsLines(context.Background(), sh.ready.Intent, mustScenarios(t, tt.c.feature), strings.Split(tt.c.feature, "\n"), sh.ready.Intent.StepFiles(),
				sh.checkedControlProps(mustScenarios(t, tt.c.feature)), 120*time.Second, &started)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(lines, "\n") != strings.Join(tt.want, "\n") && !prefixLines(lines, tt.want) {
				t.Fatalf("lines:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(tt.want, "\n"))
			}
			if started != tt.runs {
				t.Errorf("%d runs started, want %d", started, tt.runs)
			}
			if strings.Count(f.Stderr.String(), "kogen: wrong-implementation check: run ") != tt.runs ||
				!strings.Contains(f.Stderr.String(), fmt.Sprintf("kogen: wrong-implementation check: run 1 of %d (correct implementation)", 1+len(tt.c.wrong))) ||
				(len(tt.c.wrong) > 0 && tt.runs > 1 && !strings.Contains(f.Stderr.String(), fmt.Sprintf("kogen: wrong-implementation check: run 2 of %d (wrong implementation %q)", 1+len(tt.c.wrong), tt.c.wrong[0].Name))) {
				t.Errorf("progress:\n%s", f.Stderr.String())
			}
			if f.git("status", "--porcelain", "--ignored") != status || f.git("rev-parse", "HEAD") != head || f.git("ls-files", "-s") != index {
				t.Errorf("the checkout changed:\n%s", f.git("status", "--porcelain", "--ignored"))
			}
			if got := f.read("calc.js"); got != controlsCalc("add", "sub") {
				t.Errorf("calc.js = %q", got)
			}
		})
	}
}

// prefixLines reports whether each line of got starts with the line of want at its index (the
// failure messages carry a varying tail).
func prefixLines(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !strings.HasPrefix(got[i], want[i]) {
			return false
		}
	}
	return true
}

func mustScenarios(t *testing.T, feature string) []project.Scenario {
	t.Helper()
	scenarios, err := project.ParseGherkin(feature)
	if err != nil {
		t.Fatal(err)
	}
	return scenarios
}

func TestWrongImplementationCaps(t *testing.T) {
	exact := "Then is-even: add of 2 and 3 is 5"
	slow := "const end = Date.now() + 4000;\nwhile (Date.now() < end);\n" + controlsCalc("add", "sub")
	candidate := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef, correct: map[string]string{"calc.js": slow}, wrong: controlsWrongs(exact, controlsCalc("sub"))}
	t.Run("the time cap ends the check at the run it cuts off", func(t *testing.T) {
		f, sh := controlsReal(t, candidate)
		started := 0
		begin := time.Now()
		lines, err := sh.controlsLines(context.Background(), sh.ready.Intent, mustScenarios(t, candidate.feature), strings.Split(candidate.feature, "\n"), sh.ready.Intent.StepFiles(),
			sh.checkedControlProps(mustScenarios(t, candidate.feature)), 2*time.Second, &started)
		if err != nil {
			t.Fatal(err)
		}
		want := "- time cap: the runs exceeded the total time cap of 2 seconds per candidate after 0 of 3 runs completed; the check stopped. Make the implementations or the steps faster."
		if len(lines) != 1 || lines[0] != want || started != 1 {
			t.Fatalf("lines = %q, %d runs started", lines, started)
		}
		if time.Since(begin) > 3500*time.Millisecond+2*time.Second {
			t.Errorf("the cut-off run took %v", time.Since(begin))
		}
		if n := strings.Count(f.Stderr.String(), "wrong-implementation check: run "); n != 1 {
			t.Errorf("%d progress lines", n)
		}
	})
	t.Run("a run that would start with no time left does not start", func(t *testing.T) {
		_, sh := controlsReal(t, candidate)
		started := 0
		lines, err := sh.controlsLines(context.Background(), sh.ready.Intent, mustScenarios(t, candidate.feature), strings.Split(candidate.feature, "\n"), sh.ready.Intent.StepFiles(),
			sh.checkedControlProps(mustScenarios(t, candidate.feature)), time.Nanosecond, &started)
		if err != nil || len(lines) != 1 || !strings.Contains(lines[0], "after 0 of 3 runs completed") || started > 1 {
			t.Fatalf("lines = %q, %d runs started, err %v", lines, started, err)
		}
	})
	t.Run("more runs than the cap are refused before anything runs", func(t *testing.T) {
		many := candidate
		many.correct = map[string]string{"calc.js": controlsCalc("add", "sub")}
		many.wrong = nil
		for i := 0; i < 8; i++ {
			many.wrong = append(many.wrong, controlsWrong{fmt.Sprintf("w%d", i), "Add keeps working", exact, map[string]string{"calc.js": controlsCalc("sub")}})
		}
		f, sh := controlsReal(t, many)
		started := 0
		lines, err := sh.controlsLines(context.Background(), sh.ready.Intent, mustScenarios(t, many.feature), strings.Split(many.feature, "\n"), sh.ready.Intent.StepFiles(),
			sh.checkedControlProps(mustScenarios(t, many.feature)), time.Minute, &started)
		want := "- run cap: this candidate needs 9 runs (the correct implementation and 8 wrong ones), above the cap of 8 runs per candidate; nothing was run. Supply at most 7 wrong implementations."
		if err != nil || len(lines) != 1 || lines[0] != want || started != 0 || strings.Contains(f.Stderr.String(), "wrong-implementation check") {
			t.Fatalf("lines = %q, %d runs started, err %v", lines, started, err)
		}
	})
}

func TestWrongImplementationCheckRecordsAndSkips(t *testing.T) {
	exact := "Then is-even: add of 2 and 3 is 5"
	candidate := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef, correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: controlsWrongs(exact, controlsCalc("sub"))}
	t.Run("a clean check records one event", func(t *testing.T) {
		f, sh := controlsReal(t, candidate)
		confirmed, err := sh.wrongImplementationCheck(context.Background(), "h1")
		if err != nil || !confirmed.empty() {
			t.Fatalf("confirmed = %+v, err %v", confirmed, err)
		}
		events := shpNamed(f.events(), "wrong_implementation_check")
		if len(events) != 1 || events[0]["package"] != "h1" || events[0]["result"] != "clean" || events[0]["runs"] != float64(3) || len(events[0]["problems"].([]any)) != 0 {
			t.Fatalf("events = %v", events)
		}
	})
	t.Run("a missing controls.json is refused with the form", func(t *testing.T) {
		f, sh := controlsReal(t, candidate)
		if err := os.Remove(filepath.Join(f.Root, project.IntentDir(shpSlug), "controls.json")); err != nil {
			t.Fatal(err)
		}
		confirmed, err := sh.wrongImplementationCheck(context.Background(), "h2")
		if err != nil || len(confirmed.wrong) != 1 || !strings.HasPrefix(confirmed.wrong[0], "- controls.json: .kogen/intents/is-even/controls.json does not exist; write it as {") {
			t.Fatalf("confirmed = %+v, err %v", confirmed, err)
		}
		if !strings.HasPrefix(confirmed.wrongHeadline, "Wrong-implementation check: 1 problem(s)") || !strings.Contains(confirmed.wrongHeadline, "at most 8 runs and 300 seconds per candidate") {
			t.Errorf("headline = %s", confirmed.wrongHeadline)
		}
		events := shpNamed(f.events(), "wrong_implementation_check")
		if len(events) != 1 || events[0]["result"] != "refused" || events[0]["runs"] != float64(0) {
			t.Fatalf("events = %v", events)
		}
	})
	t.Run("nothing is read, run or recorded without a checked property", func(t *testing.T) {
		f, sh := controlsReal(t, candidate)
		os.Remove(filepath.Join(f.Root, project.IntentDir(shpSlug), "controls.json"))
		for name, props := range map[string][]propertyReport{
			"no property":    nil,
			"human only":     {{property: "p", request: "add", kind: "new", observable: false, scenarios: []string{"Add keeps working"}}},
			"not requested":  {{property: "p", request: "something else", kind: "new", observable: true, scenarios: []string{"Add keeps working"}}},
			"no scenario":    {{property: "p", request: "add", kind: "new", observable: true, scenarios: []string{"Not declared"}}},
			"blank property": {{property: " ", request: "add", kind: "new", observable: true, scenarios: []string{"Add keeps working"}}},
		} {
			sh.reviewProps = props
			confirmed, err := sh.wrongImplementationCheck(context.Background(), "h3")
			if err != nil || !confirmed.empty() {
				t.Errorf("%s: confirmed = %+v, err %v", name, confirmed, err)
			}
		}
		if events := shpNamed(f.events(), "wrong_implementation_check"); len(events) != 0 || strings.Contains(f.Stderr.String(), "wrong-implementation check") {
			t.Fatalf("events = %v, progress %s", events, f.Stderr.String())
		}
	})
}
