'use strict';
// The Go source of the k-shape-wrong-implementation acceptance harness, which
// k-shape-wrong-implementation__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_wrong_implementation_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

// The acceptance harness of k-shape-wrong-implementation: synthetic shaping journeys with a
// scripted shaper, a scripted contract review and a real Cucumber acceptance runner. KWI_MODE
// names one case of the request; every journey of the case is shaped in its own fixture, the
// journeys in parallel, and what a user sees of each is written as JSON to KWI_OUT for the
// acceptance steps to judge.

const kwiHeader = "const {Then} = require('@cucumber/cucumber');\nconst assert = require('node:assert/strict');\n"

// kwiImpl is one wrong implementation of controls.json: its name, the scenario and step that
// must reject it, and its files.
type kwiImpl struct {
	name, scenario, step string
	files                map[string]string
}

// kwiCandidate is one candidate package; correct is nil for a candidate without controls.json.
type kwiCandidate struct {
	feature, steps string
	correct        map[string]string
	wrong          []kwiImpl
}

// kwiJourney is one shaping run: the project files written over the fixture's before its
// commit, the first candidate, and what the scripted shaper does after submitting it:
// "resubmit" it twice unchanged, "acknowledge" accounted red, or "change" to the second
// candidate and resubmit that twice unchanged. timeCap, when set, is the total-time cap of the
// wrong-implementation runs, given through ShapeInput. refine, when set, shapes the first
// candidate with a review that reports no property and then refines the published Intent
// with the case's review: the refiner clarifies intent.md and acknowledges red.
type kwiJourney struct {
	name    string
	files   map[string]string
	first   kwiCandidate
	then    string
	second  kwiCandidate
	timeCap time.Duration
	refine  bool
}

type kwiCase struct {
	request  string
	review   map[string]any // the submit_contract_review payload of every review
	journeys []kwiJourney
}

func kwiProperty(property, request, kind string, scenarios ...string) map[string]any {
	sc := []any{}
	for _, s := range scenarios {
		sc = append(sc, s)
	}
	return map[string]any{"property": property, "request": request, "kind": kind, "observable": true, "scenarios": sc, "decision": "", "issues": []any{}}
}

func kwiReview(properties ...map[string]any) map[string]any {
	list := []any{}
	for _, p := range properties {
		list = append(list, p)
	}
	return map[string]any{"findings": []any{}, "environment": []any{}, "properties": list}
}

func kwiCases() map[string]kwiCase {
	// The bound: f(n) returns n * 2 no slower than a same-run baseline plus 100 ms.
	slow := func(ms int, result string) string {
		return fmt.Sprintf("exports.f = n => {\n  const end = Date.now() + %d;\n  while (Date.now() < end);\n  return %s;\n};\n", ms, result)
	}
	boundFiles := map[string]string{"baseline.js": "exports.f = n => n * 2;\n", "f.js": slow(300, "n")}
	fast := map[string]string{"f.js": "exports.f = n => n * 2;\n"}
	measureStep := "Then is-even: f of 1000 returns 2000 no slower than the baseline plus 100 ms"
	measureDef := "Then('is-even: f of {int} returns {int} no slower than the baseline plus {int} ms', function (n, want, extra) {\n" +
		"  const {f} = require('../../f');\n" +
		"  const base = require('../../baseline').f;\n" +
		"  let t0 = performance.now();\n" +
		"  base(n);\n" +
		"  const baseMs = performance.now() - t0;\n" +
		"  t0 = performance.now();\n" +
		"  const got = f(n);\n" +
		"  const ms = performance.now() - t0;\n" +
		"  assert.strictEqual(got, want);\n" +
		"  assert.ok(ms <= baseMs + extra, 'f(' + n + ') took ' + Math.round(ms) + ' ms, above the baseline ' + Math.round(baseMs) + ' ms plus ' + extra + ' ms');\n" +
		"});\n"
	returnsStep := "Then is-even: f of 1000 returns 2000"
	returnsDef := "Then('is-even: f of {int} returns {int}', function (n, want) {\n  assert.strictEqual(require('../../f').f(n), want);\n});\n"
	bothStep := "And is-even: f of 1000 returns both 2000 and 2001"
	bothDef := "Then('is-even: f of {int} returns both {int} and {int}', function (n, a, b) {\n  const r = require('../../f').f(n);\n  assert.strictEqual(r, a);\n  assert.strictEqual(r, b);\n});\n"
	boundFeature := func(steps ...string) string {
		return "Feature: F speed\n  Scenario: F stays within the bound\n    " + strings.Join(steps, "\n    ") + "\n"
	}
	slowed := func(name, step string) kwiImpl {
		return kwiImpl{name: name, scenario: "F stays within the bound", step: step, files: map[string]string{"f.js": slow(300, "n * 2")}}
	}
	measuring := kwiCandidate{feature: boundFeature(measureStep), steps: kwiHeader + measureDef, correct: fast, wrong: []kwiImpl{slowed("slowed f", measureStep)}}
	boundRequest := "Make f(n) return n * 2; f must be no slower than a same-run baseline plus 100 ms."
	boundReview := kwiReview(kwiProperty("f is no slower than a same-run baseline plus 100 ms", "f must be no slower than a same-run baseline plus 100 ms", "new", "F stays within the bound"))
	unreachable := kwiCandidate{feature: boundFeature(measureStep, bothStep), steps: kwiHeader + measureDef + bothDef, correct: fast, wrong: []kwiImpl{slowed("slowed f", measureStep)}}
	disabling := unreachable
	disabling.correct = map[string]string{"f.js": "const strict = require('node:assert/strict');\nconst loose = require('node:assert');\nfor (const m of [strict, loose]) {\n  m.strictEqual = () => {};\n  m.ok = () => {};\n}\nexports.f = n => n * 2;\n"}
	manyWrong := measuring
	manyWrong.wrong = nil
	for i := 1; i <= 8; i++ {
		manyWrong.wrong = append(manyWrong.wrong, slowed(fmt.Sprintf("slowed f %d", i), measureStep))
	}
	slowCorrect := measuring
	slowCorrect.correct = map[string]string{"f.js": slow(4000, "n * 2")}

	// Keep A and B and add C: calc(op, a, b) keeps add and sub and gains mul.
	add, sub, mul := "  if (op === 'add') return a + b;\n", "  if (op === 'sub') return a - b;\n", "  if (op === 'mul') return a * b;\n"
	calcOf := func(lines ...string) string {
		return "exports.calc = (op, a, b) => {\n" + strings.Join(lines, "") + "  return undefined;\n};\n"
	}
	calcFiles := map[string]string{"calc.js": calcOf(add, sub)}
	exactDef := "Then('is-even: {word} of {int} and {int} is {int}', function (op, a, b, want) {\n  assert.strictEqual(require('../../calc').calc(op, a, b), want);\n});\n"
	answerDef := "Then('is-even: add of {int} and {int} gives an answer', function (a, b) {\n  const answer = require('../../calc').calc('add', a, b);\n  assert.notStrictEqual(answer, null);\n});\n"
	arityDef := "Then('is-even: calc takes {int} arguments', function (n) {\n  assert.strictEqual(require('../../calc').calc.length, n);\n});\n"
	exactAdd, looseAdd := "Then is-even: add of 2 and 3 is 5", "Then is-even: add of 2 and 3 gives an answer"
	arityAdd := "And is-even: add of 2 and 3 gives an answer"
	calcFeature := func(addSteps ...string) string {
		return "Feature: Calc\n  Scenario: Add keeps working\n    " + strings.Join(addSteps, "\n    ") +
			"\n  Scenario: Sub keeps working\n    Then is-even: sub of 5 and 3 is 2\n  Scenario: Mul multiplies\n    Then is-even: mul of 2 and 3 is 6\n"
	}
	calcWrong := func(addStep, addCalc string) []kwiImpl {
		return []kwiImpl{
			{name: "add removed", scenario: "Add keeps working", step: addStep, files: map[string]string{"calc.js": addCalc}},
			{name: "sub removed", scenario: "Sub keeps working", step: "Then is-even: sub of 5 and 3 is 2", files: map[string]string{"calc.js": calcOf(add, mul)}},
			{name: "mul removed", scenario: "Mul multiplies", step: "Then is-even: mul of 2 and 3 is 6", files: map[string]string{"calc.js": calcOf(add, sub)}},
		}
	}
	correctCalc := map[string]string{"calc.js": calcOf(add, sub, mul)}
	strict := kwiCandidate{feature: calcFeature(exactAdd), steps: kwiHeader + exactDef, correct: correctCalc, wrong: calcWrong(exactAdd, calcOf(sub, mul))}
	loose := kwiCandidate{feature: calcFeature(looseAdd), steps: kwiHeader + exactDef + answerDef, correct: correctCalc, wrong: calcWrong(looseAdd, calcOf(sub, mul))}
	throwing := loose
	throwing.wrong = calcWrong(looseAdd, "const assert = require('node:assert');\nexports.calc = (op, a, b) => {\n  if (op === 'add') assert.fail('add was removed');\n"+sub+mul+"  return undefined;\n};\n")
	earlier := kwiCandidate{feature: calcFeature("Then is-even: calc takes 3 arguments", arityAdd), steps: kwiHeader + exactDef + answerDef + arityDef, correct: correctCalc,
		wrong: calcWrong(arityAdd, "exports.calc = (op, ...args) => {\n  const [a, b] = args;\n"+sub+mul+"  return undefined;\n};\n")}
	tampering := loose
	tampering.wrong = calcWrong(looseAdd, "const strict = require('node:assert/strict');\nstrict.notStrictEqual = strict.fail;\n"+calcOf(sub, mul))
	calcRequest := "Keep calc(op, a, b) answering add and sub exactly as it does now, and add mul to calc, which returns the product of a and b."
	calcReview := kwiReview(
		kwiProperty("add keeps working", "answering add", "kept", "Add keeps working"),
		kwiProperty("sub keeps working", "sub exactly as it does now", "kept", "Sub keeps working"),
		kwiProperty("mul returns the product", "add mul to calc, which returns the product of a and b", "new", "Mul multiplies"),
	)

	plainCalc := strict
	plainCalc.correct, plainCalc.wrong = nil, nil
	addAndMul := plainCalc
	addAndMul.feature = "Feature: Calc\n  Scenario: Add keeps working\n    Then is-even: add of 2 and 3 is 5\n  Scenario: Mul multiplies\n    Then is-even: mul of 2 and 3 is 6\n"
	fixedMul := plainCalc
	fixedMul.feature = "Feature: Calc\n  Scenario: Add keeps working\n    Then is-even: add of 2 and 3 is 5\n  Scenario: Sub keeps working\n    Then is-even: sub of 5 and 3 is 2\n  Scenario: Mul multiplies\n    Then is-even: mul is fixed at 7\n"
	fixedMul.steps = kwiHeader + exactDef + "Then('is-even: mul is fixed at {int}', function (n) {\n  assert.strictEqual(6, n);\n});\n"

	return map[string]kwiCase{
		"existing": {request: calcRequest, review: calcReview, journeys: []kwiJourney{
			{name: "tests only add and mul", files: calcFiles, first: addAndMul, then: "resubmit"},
			{name: "a step no implementation can pass", files: calcFiles, first: fixedMul, then: "resubmit"},
			{name: "refined after publishing", files: calcFiles, first: plainCalc, then: "acknowledge", refine: true},
		}},
		"bound": {request: boundRequest, review: boundReview, journeys: []kwiJourney{
			{name: "measures the bound", files: boundFiles, first: measuring, then: "acknowledge"},
			{name: "only checks that f returns", files: boundFiles, first: kwiCandidate{feature: boundFeature(returnsStep), steps: kwiHeader + returnsDef, correct: fast,
				wrong: []kwiImpl{slowed("slowed f", returnsStep)}}, then: "resubmit"},
		}},
		"kept": {request: calcRequest, review: calcReview, journeys: []kwiJourney{
			{name: "add step passes whatever calc returns", files: calcFiles, first: loose, then: "resubmit"},
			{name: "fails when add, sub or mul is removed", files: calcFiles, first: strict, then: "acknowledge"},
			{name: "exact, then loosened", files: calcFiles, first: strict, then: "change", second: loose},
		}},
		"unreachable": {request: boundRequest, review: boundReview, journeys: []kwiJourney{
			{name: "fails for every implementation", files: boundFiles, first: unreachable, then: "resubmit"},
			{name: "correct implementation disables assertions", files: boundFiles, first: disabling, then: "resubmit"},
		}},
		"caps": {request: boundRequest, review: boundReview, journeys: []kwiJourney{
			{name: "needs more runs than the cap", files: boundFiles, first: manyWrong, then: "resubmit"},
			{name: "runs longer than the time cap", files: boundFiles, first: slowCorrect, then: "resubmit", timeCap: 2 * time.Second},
		}},
		"plain": {request: "Make node hello.js print hello.", review: kwiReview(), journeys: []kwiJourney{
			{name: "one command, one output", files: map[string]string{"hello.js": "console.log('hi');\n"}, first: kwiCandidate{
				feature: "Feature: Hello\n  Scenario: Hello prints hello\n    Then is-even: node hello.js prints \"hello\"\n",
				steps:   kwiHeader + "Then('is-even: node hello.js prints {string}', function (want) {\n  const r = require('node:child_process').spawnSync(process.execPath, ['hello.js'], {encoding: 'utf8'});\n  assert.strictEqual(r.stdout, want + '\\n');\n});\n",
			}, then: "acknowledge"},
		}},
		"evidence": {request: calcRequest, review: calcReview, journeys: []kwiJourney{
			{name: "wrong add throws its own AssertionError", files: calcFiles, first: throwing, then: "resubmit"},
			{name: "wrong add fails an earlier unrelated step", files: calcFiles, first: earlier, then: "resubmit"},
			{name: "wrong add overwrites an assertion", files: calcFiles, first: tampering, then: "resubmit"},
		}},
	}
}

// kwiRefined is the intent.md the refiner writes.
const kwiRefined = "A synthetic change, clarified."

func kwiIntent() string {
	return "# Shape a synthetic change\n\nSlug: is-even\nModel: not applicable — a synthetic change without state\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" +
		"A synthetic change.\n\n## Decisions\n- Results are compared exactly. (source: request)\n\n## Source comparison\n- The request is covered by the feature.\n\n## Tests\n- The requested behaviour: the acceptance scenarios (scenario)\n\n## Feasibility\n- none\n"
}

// kwiControls is the controls.json of a candidate.
func kwiControls(c kwiCandidate) string {
	wrong := []any{}
	for _, w := range c.wrong {
		wrong = append(wrong, map[string]any{"name": w.name, "scenario": w.scenario, "step": w.step, "files": w.files})
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]any{"correct": map[string]any{"files": c.correct}, "wrong": wrong})
	return b.String()
}

// kwiPackage is the candidate's files by path.
func kwiPackage(c kwiCandidate) map[string]string {
	files := map[string]string{
		".kogen/intents/is-even/intent.md":          kwiIntent(),
		".kogen/intents/is-even/acceptance.feature": c.feature,
		"features/steps/is-even__steps.js":          c.steps,
	}
	if c.correct != nil {
		files[".kogen/intents/is-even/controls.json"] = kwiControls(c)
	}
	return files
}

func kwiFiles(c kwiCandidate) map[string]any {
	list := []any{}
	for p, content := range kwiPackage(c) {
		list = append(list, map[string]any{"path": p, "content": content})
	}
	return map[string]any{"files": list}
}

// kwiRun shapes one journey in a fresh fixture and returns what a user sees of it.
func kwiRun(t *testing.T, c kwiCase, j kwiJourney) map[string]any {
	t.Helper()
	f := newShpFx(t)
	modules := os.Getenv("KOGEN_CUCUMBER")
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	runner := filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")
	f.write("features/steps/calc_steps.js", "// existing steps\n")
	for p, content := range j.files {
		f.write(p, content)
	}
	// Every acceptance run, readiness's and any other, appends one line to the trace.
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\necho run >> '%s/trace'\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\n"+
		"NODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", f.Aux, modules, node, runner))
	f.git("add", "-A")
	f.git("commit", "-qm", "wrong-implementation fixture")

	last := j.first
	seq := []shpStep{shpSubmit("cand", kwiFiles(j.first))}
	switch j.then {
	case "resubmit":
		seq = append(seq, shpSubmit("again", kwiFiles(j.first)), shpSubmit("last", kwiFiles(j.first)))
	case "acknowledge":
		seq = append(seq, shpSubmit("ack", map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": j.first.feature}}}))
	case "change":
		last = j.second
		seq = append(seq, shpSubmit("change", kwiFiles(j.second)), shpSubmit("again", kwiFiles(j.second)), shpSubmit("last", kwiFiles(j.second)))
	}
	if j.refine {
		refined := strings.Replace(kwiIntent(), "A synthetic change.", kwiRefined, 1)
		seq = append(seq, refSubmit("c1", ".kogen/intents/is-even/intent.md", refined), refSubmit("c2", ".kogen/intents/is-even/intent.md", refined))
	}
	f.Prov.seq = seq
	var reviewMu sync.Mutex
	review := c.review
	if j.refine {
		review = kwiReview()
	}
	verdicts := []any{}
	for n := 1; n <= 8; n++ {
		verdicts = append(verdicts, map[string]any{"finding": n, "verdict": "confirmed", "why": "synthetic confirmation"})
	}
	// Every contract review gets the case's review and every confirmation confirms.
	f.Prov.route = func(req map[string]any) *shpStep {
		var st shpStep
		switch contractKind(req) {
		case SubmitContractReviewTool:
			reviewMu.Lock()
			st = shpAnswer(shpCall("review", SubmitContractReviewTool, review))
			reviewMu.Unlock()
		case SubmitContractConfirmationTool:
			st = shpAnswer(shpCall("confirm", SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts}))
		default:
			return nil
		}
		return &st
	}

	o := map[string]any{"remaining": "", "time_cap_field": true}
	in := ShapeInput{Project: f.Root, Slug: shpSlug, Request: c.request, Client: f.Client}
	if j.timeCap > 0 {
		// Set by name, so that this harness compiles on a tree without the field; without it the
		// journey runs uncapped and the observation says so.
		field := reflect.ValueOf(&in).Elem().FieldByName("WrongImplementationTimeCap")
		if field.IsValid() && field.Type() == reflect.TypeOf(time.Duration(0)) {
			field.SetInt(int64(j.timeCap))
		} else {
			o["time_cap_field"] = false
		}
	}
	f.newSession()
	exit, err := Shape(context.Background(), f.S, in)

	o["published"] = err == nil && exit == 0 && f.exists(".kogen/local/readiness/is-even.json")
	onDisk := true
	for p, want := range kwiPackage(last) {
		got, rerr := os.ReadFile(filepath.Join(f.Root, p))
		onDisk = onDisk && rerr == nil && string(got) == want
	}
	o["files"] = onDisk
	if e, ok := sys.AsError(err); ok {
		o["remaining"] = strings.Join(e.Lines, "\n")
	}
	reqs := f.Prov.requests()
	o["shaper"] = len(reqs)
	results := map[string]any{}
	for _, r := range reqs {
		for id, out := range shpOutputs(r) {
			switch id {
			case "cand", "again", "last", "ack", "change", "c1":
				results[id] = out
			}
		}
	}
	o["results"] = results
	o["contract"] = len(f.Prov.contractRequests())
	trace, _ := os.ReadFile(filepath.Join(f.Aux, "trace"))
	o["runs"] = strings.Count(string(trace), "run\n")
	o["progress"] = f.Stderr.String()
	o["exit"] = fmt.Sprintf("exit %d, err %v", exit, err)
	if j.refine {
		kwiRefine(f, o)
	}
	return o
}

// kwiRefine refines the published Intent of a journey with the case's review and adds what a
// user sees of the refinement to o.
func kwiRefine(f *shpFx, o map[string]any) {
	o["refine_exit"], o["refine_outcome"], o["refined"], o["refine_runs"], o["refine_progress"] = "not run", "", false, 0, ""
	if o["published"] != true {
		return
	}
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		f.t.Fatal(err)
	}
	paths, err := project.Baseline(f.Root, shpSlug, cfg)
	if err != nil {
		f.t.Fatal(err)
	}
	runs := func() int {
		trace, _ := os.ReadFile(filepath.Join(f.Aux, "trace"))
		return strings.Count(string(trace), "run\n")
	}
	before := runs()
	f.newSession()
	exit, err := Refine(context.Background(), f.S, RefineInput{Project: f.Root, Slug: shpSlug, Hash: project.Hash(f.Root, paths), Client: f.Client,
		Request: "Clarify the Intent's prose; keep calc answering add and sub exactly as it does now."})
	o["refine_exit"] = fmt.Sprintf("exit %d, err %v", exit, err)
	o["refine_runs"] = runs() - before
	o["refine_progress"] = f.Stderr.String()
	if results, _ := filepath.Glob(filepath.Join(f.Root, ".kogen/local/refinements/*/result.json")); len(results) == 1 {
		var doc map[string]any
		if b, err := os.ReadFile(results[0]); err == nil && json.Unmarshal(b, &doc) == nil {
			o["refine_outcome"] = doc["outcome"]
		}
	}
	got, _ := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md"))
	o["refined"] = strings.Contains(string(got), kwiRefined)
	for _, r := range f.Prov.requests() {
		if out, ok := shpOutputs(r)["c1"]; ok {
			o["results"].(map[string]any)["c1"] = out
		}
	}
}

func TestKWrongImplementationAcceptance(t *testing.T) {
	mode, out := os.Getenv("KWI_MODE"), os.Getenv("KWI_OUT")
	c, ok := kwiCases()[mode]
	if !ok || out == "" {
		t.Fatalf("TestKWrongImplementationAcceptance: KWI_MODE %q is not a case or KWI_OUT is unset", mode)
	}
	var mu sync.Mutex
	all := map[string]any{}
	t.Run("journeys", func(t *testing.T) {
		for _, j := range c.journeys {
			t.Run(j.name, func(t *testing.T) {
				o := kwiRun(t, c, j)
				mu.Lock()
				defer mu.Unlock()
				all[j.name] = o
			})
		}
	})
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
