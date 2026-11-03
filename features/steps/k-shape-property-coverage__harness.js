'use strict';
// The Go source of the k-shape-property-coverage acceptance harness, which
// k-shape-property-coverage__steps.js compiles into the shape package's tests through a Go overlay
// as shape/k_property_coverage_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kogen-loop/sys"
)

// The acceptance harness of k-shape-property-coverage: synthetic shaping journeys with a
// scripted shaper, a scripted contract review and a real Cucumber acceptance runner. KPC_MODE
// names one case of the request; every candidate of the case is shaped in its own fixture, and
// what a user sees of each is written as JSON to KPC_OUT for the acceptance steps to judge.

const kpcHeader = "const {Then} = require('@cucumber/cucumber');\nconst assert = require('node:assert/strict');\n"

// kpcJourney is one candidate of a case: the project files written over the fixture's before its
// commit, the candidate package, and how the scripted shaper goes on after its first submission.
type kpcJourney struct {
	name      string
	files     map[string]string
	prose     string
	decisions []string
	feature   string
	steps     string
	// then is what the shaper does after the first submission: "resubmit" the same candidate
	// twice, "acknowledge" accounted red, or nothing ("") for a candidate readiness finds green.
	then string
}

type kpcCase struct {
	request  string
	answer   string         // when set, the shaper first asks one question and the caller answers this
	review   map[string]any // the submit_contract_review payload of every review
	journeys []kpcJourney
}

func kpcProperty(property, request, kind string, observable bool, scenarios []string, decision string, issues ...map[string]any) map[string]any {
	sc := []any{}
	for _, s := range scenarios {
		sc = append(sc, s)
	}
	is := []any{}
	for _, i := range issues {
		is = append(is, i)
	}
	return map[string]any{"property": property, "request": request, "kind": kind, "observable": observable, "scenarios": sc, "decision": decision, "issues": is}
}

func kpcIssue(problem, scenario, quote string) map[string]any {
	return map[string]any{"problem": problem, "scenario": scenario, "quote": quote, "why": "synthetic " + problem + " issue"}
}

func kpcReview(properties ...map[string]any) map[string]any {
	list := []any{}
	for _, p := range properties {
		list = append(list, p)
	}
	return map[string]any{"findings": []any{}, "environment": []any{}, "properties": list}
}

func kpcCases() map[string]kpcCase {
	exact := "- Results are compared exactly. (source: request)"

	calc := map[string]string{"calc.js": "exports.add = (a, b) => a + b;\nexports.sub = (a, b) => a - b;\n"}
	calcSteps := kpcHeader + "Then('is-even: {word} of {int} and {int} is {int}', function (op, a, b, want) {\n  assert.strictEqual(require('../../calc')[op](a, b), want);\n});\n"
	calcProse := "Keep add and sub as they are, and add mul(a, b). mul(a, b) is memoized."
	mul := "  Scenario: Mul multiplies\n    Then is-even: mul of 2 and 3 is 6\n"

	slowF := map[string]string{"f.js": "exports.f = n => {\n  const end = Date.now() + 300;\n  while (Date.now() < end);\n  return n * 2;\n};\n"}

	slugFeature := "Feature: Slug\n  Scenario: Slug stays short\n    Then is-even: the slug of a title of 50 characters has at most 20 characters\n"
	slugSteps := kpcHeader + "Then('is-even: the slug of a title of {int} characters has at most {int} characters', function (n, max) {\n  const {slug} = require('../../slug');\n  const s = slug('a'.repeat(n));\n  assert.ok(String(s).length <= max, 'slug returned ' + JSON.stringify(s));\n});\n"

	greet := map[string]string{"greet.js": "exports.greet = name => 'Hi ' + name;\n"}
	greetFeature := "Feature: Greeting\n  Scenario: Greets by name\n    Then is-even: the greeting of \"Ada\" names Ada\n"
	greetSteps := kpcHeader + "Then('is-even: the greeting of {string} names {word}', function (name, word) {\n  assert.ok(require('../../greet').greet(name).includes(word));\n});\n" +
		"Then('is-even: the greeting of {string} scores at least {int} for welcome', function (name, min) {\n  const text = require('../../greet').greet(name);\n  const score = (text.match(/welcome|glad/gi) || []).length * 5;\n  assert.ok(score >= min, 'welcome score ' + score);\n});\n"
	human := "Whether the greeting feels welcoming needs human evaluation"
	keepName := "- greet(name) keeps returning a text with the name. (source: request)"

	return map[string]kpcCase{
		"kept": {
			request: "Keep add(a, b) and sub(a, b) working exactly as they do now, and add mul(a, b), which returns the product of a and b.",
			review: kpcReview(
				kpcProperty("add keeps working", "Keep add(a, b)", "kept", true, []string{"Add keeps working"}, ""),
				kpcProperty("sub keeps working", "sub(a, b) working exactly as they do now", "kept", true, []string{"Sub keeps working"}, ""),
				// Only the candidate asks for this one, never the request.
				kpcProperty("mul is memoized", "mul(a, b) is memoized", "new", true, nil, ""),
			),
			journeys: []kpcJourney{
				{name: "tests only C", files: calc, prose: calcProse, decisions: []string{exact}, feature: "Feature: Calc\n" + mul, steps: calcSteps, then: "resubmit"},
				{name: "tests A, B and C", files: calc, prose: calcProse, decisions: []string{exact},
					feature: "Feature: Calc\n  Scenario: Add keeps working\n    Then is-even: add of 2 and 3 is 5\n  Scenario: Sub keeps working\n    Then is-even: sub of 5 and 3 is 2\n" + mul,
					steps:   calcSteps, then: "acknowledge"},
			},
		},
		"bound": {
			request: "Make f(n) no slower than 100 ms for n = 1000.",
			review: kpcReview(kpcProperty("f is no slower than 100 ms", "no slower than 100 ms", "new", true, []string{"F meets the bound"}, "",
				kpcIssue("test_count", "F meets the bound", "/pass [1-9]/"))),
			journeys: []kpcJourney{
				{name: "counts builder-written tests", files: slowF, prose: "f(n) returns within 100 ms for n = 1000.", decisions: []string{exact},
					feature: "Feature: F speed\n  Scenario: F meets the bound\n    Then is-even: the speed tests pass\n",
					steps:   kpcHeader + "Then('is-even: the speed tests pass', function () {\n  const r = require('node:child_process').spawnSync(process.execPath, ['--test'], {encoding: 'utf8'});\n  assert.match(r.stdout, /pass [1-9]/);\n});\n",
					then:    "resubmit"},
				{name: "measures f against the bound", files: slowF, prose: "f(n) returns within 100 ms for n = 1000.", decisions: []string{exact},
					feature: "Feature: F speed\n  Scenario: F meets the bound\n    Then is-even: f of 1000 returns within 100 ms\n",
					steps:   kpcHeader + "Then('is-even: f of {int} returns within {int} ms', function (n, bound) {\n  const {f} = require('../../f');\n  const t0 = performance.now();\n  f(n);\n  const ms = performance.now() - t0;\n  assert.ok(ms <= bound, 'f(' + n + ') took ' + Math.round(ms) + ' ms, above ' + bound + ' ms');\n});\n",
					then:    "acknowledge"},
			},
		},
		"missing": {
			request: "Add slug(text), whose every result is at most 20 characters long.",
			review: kpcReview(kpcProperty("slug results are at most 20 characters", "at most 20 characters long", "new", true, []string{"Slug stays short"}, "",
				kpcIssue("missing", "Slug stays short", "Cannot find module '../../slug'"))),
			journeys: []kpcJourney{
				{name: "target file absent", prose: "slug(text) returns at most 20 characters.", decisions: []string{exact}, feature: slugFeature, steps: slugSteps, then: "resubmit"},
				{name: "target file as a do-nothing form", files: map[string]string{"slug.js": "exports.slug = text => text;\n"}, prose: "slug(text) returns at most 20 characters.", decisions: []string{exact},
					feature: slugFeature, steps: slugSteps, then: "acknowledge"},
			},
		},
		"human": {
			request: "Make the greeting feel welcoming.",
			answer:  "1: a",
			review:  kpcReview(kpcProperty("the greeting feels welcoming", "feel welcoming", "new", false, []string{"Greeting is welcoming"}, human)),
			journeys: []kpcJourney{
				{name: "records human evaluation", files: greet, prose: "The greeting should feel welcoming; a person judges that.",
					decisions: []string{"- " + human + "; no scenario asserts it. (source: answer 1)", keepName}, feature: greetFeature, steps: greetSteps},
				{name: "asserts a welcoming score", files: greet, prose: "The greeting should feel welcoming; a person judges that.",
					decisions: []string{"- " + human + "; no scenario asserts it. (source: answer 1)", keepName},
					feature:   greetFeature + "  Scenario: Greeting is welcoming\n    Then is-even: the greeting of \"Ada\" scores at least 8 for welcome\n", steps: greetSteps, then: "resubmit"},
				{name: "records no human evaluation", files: greet, prose: "The greeting should feel welcoming.", decisions: []string{keepName},
					feature: greetFeature, steps: greetSteps, then: "resubmit"},
			},
		},
		"plain": {
			request: "Make node hello.js print hello.",
			review:  kpcReview(),
			journeys: []kpcJourney{
				{name: "one command, one output", files: map[string]string{"hello.js": "console.log('hi');\n"}, prose: "node hello.js prints hello.", decisions: []string{exact},
					feature: "Feature: Hello\n  Scenario: Hello prints hello\n    Then is-even: node hello.js prints \"hello\"\n",
					steps:   kpcHeader + "Then('is-even: node hello.js prints {string}', function (want) {\n  const r = require('node:child_process').spawnSync(process.execPath, ['hello.js'], {encoding: 'utf8'});\n  assert.strictEqual(r.stdout, want + '\\n');\n});\n",
					then:    "acknowledge"},
			},
		},
	}
}

func kpcIntent(j kpcJourney) string {
	return "# Shape a synthetic change\n\nSlug: is-even\nModel: not applicable — a synthetic change without state\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" +
		j.prose + "\n\n## Decisions\n" + strings.Join(j.decisions, "\n") + "\n\n## Source comparison\n- The request is covered by the feature.\n\n## Tests\n- The requested behaviour: the acceptance scenarios (scenario)\n\n## Feasibility\n- none\n"
}

func kpcFiles(j kpcJourney) map[string]any {
	return map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": kpcIntent(j)},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": j.feature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": j.steps},
	}}
}

// kpcUser is the user text of a request.
func kpcUser(req map[string]any) string {
	var out strings.Builder
	input, _ := req["input"].([]any)
	for _, it := range input {
		m, _ := it.(map[string]any)
		if s, ok := m["content"].(string); ok && m["role"] == "user" {
			out.WriteString(s)
		}
	}
	return out.String()
}

// kpcRun shapes one candidate in a fresh fixture and returns what a user sees of it.
func kpcRun(t *testing.T, c kpcCase, j kpcJourney) map[string]any {
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
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\necho run >> '%s/trace'\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\n"+
		"NODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", f.Aux, modules, node, runner))
	f.git("add", "-A")
	f.git("commit", "-qm", "property-coverage fixture")

	files := kpcFiles(j)
	candidate := []shpStep{shpSubmit("cand", files)}
	switch j.then {
	case "resubmit":
		candidate = append(candidate, shpSubmit("again", files), shpSubmit("last", files))
	case "acknowledge":
		candidate = append(candidate, shpSubmit("ack", map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": j.feature}}}))
	}
	verdicts := []any{}
	for n := 1; n <= 4; n++ {
		verdicts = append(verdicts, map[string]any{"finding": n, "verdict": "confirmed", "why": "synthetic confirmation"})
	}
	f.Prov.contract = []shpStep{
		shpAnswer(shpCall("review", SubmitContractReviewTool, c.review)),
		shpAnswer(shpCall("confirm", SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts})),
	}
	request := c.request
	if c.answer != "" {
		question := map[string]any{"question": "No automatic step can tell whether a greeting feels welcoming. How is that judged?", "options": []any{
			map[string]any{"label": "Human judgement is fine: a person evaluates it", "recommended": true},
			map[string]any{"label": "Define a measurable rule instead"},
		}}
		f.Prov.seq = []shpStep{
			shpSubmit("ask", map[string]any{"questions": []any{question}}),
			shpAnswer(shpCall("question-check", "submit_question_check", map[string]any{"items": []any{map[string]any{"number": 1, "class": "ui_ux", "why": "synthetic ui_ux question"}}})),
		}
		f.Prov.seq = append(f.Prov.seq, candidate...)
		if exit, err := f.shape(request); exit != 5 || err != nil {
			t.Fatalf("TestKPropertyCoverageAcceptance %q: the question run ended with exit %d, err %v, want exit 5 with the question pending\n%s", j.name, exit, err, f.Stderr.String())
		}
		request = c.answer
	} else {
		f.Prov.seq = candidate
	}
	exit, err := f.shape(request)
	o := map[string]any{"remaining": "", "feedback": "", "review": ""}
	o["published"] = err == nil && exit == 0 && f.exists(".kogen/local/readiness/is-even.json")
	if e, ok := sys.AsError(err); ok {
		o["remaining"] = strings.Join(e.Lines, "\n")
	}
	reqs := f.Prov.requests()
	o["shaper"] = len(reqs)
	for _, r := range reqs {
		if out, ok := shpOutputs(r)["cand"]; ok {
			o["feedback"] = out
			break
		}
	}
	contract := f.Prov.contractRequests()
	o["contract"] = len(contract)
	if len(contract) > 0 {
		o["review"] = kpcUser(contract[0])
	}
	o["progress"] = f.Stderr.String()
	o["exit"] = fmt.Sprintf("exit %d, err %v", exit, err)
	return o
}

func TestKPropertyCoverageAcceptance(t *testing.T) {
	mode, out := os.Getenv("KPC_MODE"), os.Getenv("KPC_OUT")
	c, ok := kpcCases()[mode]
	if !ok || out == "" {
		t.Fatalf("TestKPropertyCoverageAcceptance: KPC_MODE %q is not a case or KPC_OUT is unset", mode)
	}
	all := map[string]any{}
	for _, j := range c.journeys {
		all[j.name] = kpcRun(t, c, j)
	}
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
