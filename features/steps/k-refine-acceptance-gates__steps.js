'use strict';
// k-refine-acceptance-gates: synthetic Shape and Refine journeys with a scripted shaper. The Go
// harness below is compiled into the shape package's tests through a Go overlay; KRAG_MODE picks
// the journey, and good and bad judge known observations with the same rules.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKRefineAcceptanceGatesAcceptance';
const TARGET = 'shape/k_refine_acceptance_gates_acceptance_test.go';
const HARNESS = String.raw`package shape

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"kogen-loop/sys"
)

// The acceptance harness of k-refine-acceptance-gates. For each of Shape's acceptance checks, the
// same refined draft goes to native Shape as a whole submission and to native Refine as the
// correction of a ready draft, both with a scripted shaper: Refine must refuse it with Shape's exact
// feedback, keep the draft and end refinement_invalid. plain, red and repair are refinements that
// pass the checks and publish. good and bad judge known observations with the same rules.

const kragTest = "TestKRefineAcceptanceGatesAcceptance"

var kragHeadlines = map[string]string{
	"unwritten":   "Acceptance depends on tests the Build would have to invent: ",
	"observer":    "Acceptance rejects the test runner's real output: ",
	"unpassable":  "Acceptance has steps that no implementation can pass: ",
	"contract":    "Contract check: 1 acceptance expectation(s) contradict the Intent or code it keeps unchanged",
	"environment": "Environment check: 1 acceptance step(s) fail on the unchanged tree",
}

const (
	kragInvalid     = "shape/refinement_invalid: no valid ready correction after 3 submissions; the original draft is preserved (exit 1)"
	kragObserved    = "Readiness observations"
	kragRecord      = ".kogen/local/readiness/is-even.json"
	kragStubFeature = "Feature: Is even\n  Scenario: Coverage\n    Then is-even: the coverage is complete\n"
	kragStubSteps   = "// steps\nconst { Then } = require('@cucumber/cucumber');\nThen('is-even: the coverage is complete', function () { throw new Error('coverage incomplete: this step is a stub'); });\n"
)

var kragBehavioural = shpSteps + "// behavioural\n"

type kragObs struct {
	mode            string
	shapeFeedback   string   // Shape's feedback to its first submission of the same draft
	shapeContract   []string // Shape's contract requests in order (review, confirmation)
	refineFeedback  []string // Refine's feedback to each submission that received one, in order
	refinerRequests int      // the refiner's own requests, without contract requests
	refineContract  []string // Refine's contract requests in order
	exit            int
	err             string // class/reason: detail (exit n); "" without an error
	lines           string // the error's lines joined by newlines
	unchanged       bool   // the draft package, its hash and its readiness record are as before
	intent, steps   string // the draft's intent.md and step file after Refine
	detail          string
}

func kragCheck(t *testing.T, o kragObs) {
	t.Helper()
	where := kragTest + " (KRAG_MODE=" + o.mode + "): "
	headlined := func(text string) bool {
		for _, h := range kragHeadlines {
			if strings.HasPrefix(text, h) {
				return true
			}
		}
		return false
	}
	published := func(requests int) {
		if o.err != "" || o.exit != 0 {
			t.Errorf("%sRefine did not publish: exit %d, error %q (%s)", where, o.exit, o.err, o.detail)
		}
		if o.refinerRequests != requests {
			t.Errorf("%srefiner requests = %d, want %d (%s)", where, o.refinerRequests, requests, o.detail)
		}
	}
	switch o.mode {
	case "unwritten", "observer", "unpassable", "contract", "environment":
		if !strings.HasPrefix(o.shapeFeedback, kragHeadlines[o.mode]) {
			t.Errorf("%sShape does not refuse this draft for this reason; its feedback is %q", where, o.shapeFeedback)
		}
		if o.refinerRequests != 3 {
			t.Errorf("%srefiner requests = %d, want 3 (every submission refused) (%s)", where, o.refinerRequests, o.detail)
		}
		if len(o.refineFeedback) != 2 {
			t.Errorf("%sRefine returned %d feedback texts, want 2", where, len(o.refineFeedback))
		}
		for i, fb := range o.refineFeedback {
			if fb != o.shapeFeedback {
				t.Errorf("%sRefine's feedback to submission %d is not Shape's feedback:\nRefine: %q\nShape:  %q", where, i+1, fb, o.shapeFeedback)
			}
		}
		if o.err != kragInvalid || o.exit != 1 {
			t.Errorf("%sRefine ended exit %d with %q, want %q (%s)", where, o.exit, o.err, kragInvalid, o.detail)
		}
		if o.lines != o.shapeFeedback {
			t.Errorf("%sthe error lines are %q, want the refusal %q", where, o.lines, o.shapeFeedback)
		}
		if !o.unchanged {
			t.Errorf("%sthe refused refinement changed the draft, its hash or its readiness record", where)
		}
		if strings.Join(o.refineContract, ",") != strings.Join(o.shapeContract, ",") {
			t.Errorf("%sRefine's contract requests %v are not Shape's %v", where, o.refineContract, o.shapeContract)
		}
	case "plain":
		published(1)
		if o.intent != refTitled {
			t.Errorf("%sthe refined intent.md was not published: %q", where, o.intent)
		}
	case "red":
		published(2)
		if len(o.refineFeedback) == 0 || !strings.Contains(o.refineFeedback[0], kragObserved) || headlined(o.refineFeedback[0]) {
			t.Errorf("%sthe red draft was not returned as ordinary readiness observations: %q", where, o.refineFeedback)
		}
		if o.intent != refTitled {
			t.Errorf("%sthe refined intent.md was not published: %q", where, o.intent)
		}
	case "repair":
		published(3)
		if !strings.HasPrefix(o.shapeFeedback, kragHeadlines["unwritten"]) {
			t.Errorf("%sShape does not refuse the first draft for unwritten tests; its feedback is %q", where, o.shapeFeedback)
		}
		if len(o.refineFeedback) < 2 || o.refineFeedback[0] != o.shapeFeedback {
			t.Errorf("%sthe first submission was not refused with Shape's feedback %q: %q", where, o.shapeFeedback, o.refineFeedback)
		} else if !strings.Contains(o.refineFeedback[1], kragObserved) || headlined(o.refineFeedback[1]) {
			t.Errorf("%sthe repaired draft was not returned as ordinary readiness observations: %q", where, o.refineFeedback[1])
		}
		if o.steps != kragBehavioural {
			t.Errorf("%sthe repaired step file was not published: %q", where, o.steps)
		}
	default:
		t.Fatalf("%sunknown mode", where)
	}
}

// kragCandidate is the refined draft's changed files (path, content, ...).
func kragCandidate(mode string) []string {
	if mode == "unpassable" {
		return []string{refFeaturePath, kragStubFeature, refStepPath, kragStubSteps}
	}
	return []string{refIntentPath, refTitled}
}

// kragShapeFiles is the complete refined draft as one Shape submission.
func kragShapeFiles(mode string) map[string]any {
	files := shpFiles()
	list := files["files"].([]any)
	if mode == "unpassable" {
		list[1].(map[string]any)["content"] = kragStubFeature
		list[2].(map[string]any)["content"] = kragStubSteps
	} else {
		list[0].(map[string]any)["content"] = refTitled
	}
	return files
}

// kragRealRunner makes the project's acceptance command run the real Cucumber and commits it.
func kragRealRunner(f *shpFx) {
	f.t.Helper()
	modules := os.Getenv("KOGEN_CUCUMBER")
	if modules == "" || os.Getenv("KOGEN_NODE") == "" {
		f.t.Fatalf("%s: KOGEN_NODE and KOGEN_CUCUMBER must name Kogen's Node and Cucumber", kragTest)
	}
	f.write("features/steps/calc_steps.js", "// existing steps\n")
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\necho run >> '%s/trace'\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\n"+
		"NODE_PATH='%s' \"$KOGEN_NODE\" '%s/@cucumber/cucumber/bin/cucumber.js' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", f.Aux, modules, modules))
	f.git("add", "accept.sh", "features/steps/calc_steps.js")
	f.git("commit", "-qm", "real acceptance runner")
}

// kragPrepare gives the runner and the contract review what make the mode's draft defective.
func kragPrepare(f *shpFx, mode string) {
	f.t.Helper()
	switch mode {
	case "unwritten":
		f.unwrittenReport()
	case "observer":
		f.runnerObserverReport()
	case "unpassable":
		kragRealRunner(f)
	case "contract":
		f.setReport("PASSED", 0)
		f.Prov.contract = []shpStep{ccReview("r1", ccContradiction()), ccConfirm("v1", ccVerdict(1, "confirmed", "four is even"))}
	case "environment":
		f.setReport("FAILED", 1)
		f.Prov.contract = []shpStep{ccEnvReview("r1", ccEnvFindingArgs("Even numbers", ccEnvOutput)), ccConfirm("v1", ccVerdict(1, "confirmed", "the copy was not prepared"))}
	}
}

func kragContract(f *shpFx) []string {
	var kinds []string
	for _, r := range f.Prov.allRequests() {
		switch contractKind(r) {
		case SubmitContractReviewTool:
			kinds = append(kinds, "review")
		case SubmitContractConfirmationTool:
			kinds = append(kinds, "confirmation")
		}
	}
	return kinds
}

func kragTail(s string) string {
	if len(s) > 3000 {
		return "..." + s[len(s)-3000:]
	}
	return s
}

// kragShape submits the refined draft of mode to native Shape three times.
func kragShape(t *testing.T, mode string) (string, []string, string) {
	t.Helper()
	files := kragShapeFiles(mode)
	f := newShpFx(t, shpSubmit("c1", files), shpSubmit("c2", files), shpSubmit("c3", files))
	kragPrepare(f, mode)
	exit, err := f.shape("Add calc.is_even(n).")
	feedback := ""
	if reqs := f.Prov.requests(); len(reqs) > 1 {
		feedback = shpOutputs(reqs[1])["c1"]
	}
	return feedback, kragContract(f), fmt.Sprintf("shape exit=%d err=%v", exit, err)
}

// kragRefine refines a ready draft with the mode's submissions.
func kragRefine(t *testing.T, mode string) kragObs {
	t.Helper()
	cand := kragCandidate(mode)
	var steps []shpStep
	switch mode {
	case "plain":
		steps = []shpStep{refSubmit("c1", cand...)}
	case "red":
		steps = []shpStep{refSubmit("c1", cand...), refSubmit("c2", cand...)}
	case "repair":
		fixed := []string{refIntentPath, refTitled, refStepPath, kragBehavioural}
		steps = []shpStep{refSubmit("c1", cand...), refSubmit("c2", fixed...), refSubmit("c3", fixed...)}
	default:
		steps = []shpStep{refSubmit("c1", cand...), refSubmit("c2", cand...), refSubmit("c3", cand...)}
	}
	f := newRefFx(t, refOpts{}, steps...)
	switch mode {
	case "plain":
	case "red":
		f.setReport("FAILED", 1)
	case "repair":
		f.unwrittenReport()
		f.Prov.route = func(req map[string]any) *shpStep {
			if _, ok := shpOutputs(req)["c1"]; ok {
				f.setReport("FAILED", 1)
			}
			return nil
		}
	default:
		kragPrepare(f.shpFx, mode)
	}
	before := f.pkg()
	record, _ := os.ReadFile(f.Root + "/" + kragRecord)
	exit, err := f.refine(f.hash0)
	o := kragObs{mode: mode, exit: exit, refineContract: kragContract(f.shpFx),
		detail: fmt.Sprintf("refine exit=%d err=%v\nstderr:\n%s", exit, err, kragTail(f.Stderr.String()))}
	reqs := f.Prov.requests()
	o.refinerRequests = len(reqs)
	for i := 1; i < len(reqs); i++ {
		o.refineFeedback = append(o.refineFeedback, shpOutputs(reqs[i])[fmt.Sprintf("c%d", i)])
	}
	if e, ok := sys.AsError(err); ok {
		o.err = fmt.Sprintf("%s/%s: %s (exit %d)", e.Class, e.Reason, e.Detail, e.Exit)
		o.lines = strings.Join(e.Lines, "\n")
	} else if err != nil {
		o.err = err.Error()
	}
	after := f.pkg()
	same := len(after) == len(before)
	for p, c := range before {
		same = same && after[p] == c
	}
	now, rerr := os.ReadFile(f.Root + "/" + kragRecord)
	o.unchanged = same && f.hash() == f.hash0 && rerr == nil && string(now) == string(record)
	o.intent, o.steps = after[refIntentPath], after[refStepPath]
	return o
}

// kragControls judges known observations: good ones are accepted; bad is today's Refine, which
// returns the unwritten-tests draft as ordinary observations and publishes it.
func kragControls(t *testing.T, bad bool) {
	ack := "Inspect whether fixtures, inputs and commands reached the intended behavior.\n\n" + kragObserved + "\nChecked baseline: 0"
	refusal := func(mode string) kragObs {
		text := kragHeadlines[mode] + "known refusal\n- known item"
		var contract []string
		if mode == "contract" || mode == "environment" {
			contract = []string{"review", "confirmation"}
		}
		return kragObs{mode: mode, shapeFeedback: text, shapeContract: contract, refineFeedback: []string{text, text}, refinerRequests: 3,
			refineContract: contract, exit: 1, err: kragInvalid, lines: text, unchanged: true, intent: shpScopedIntentMD, steps: shpSteps}
	}
	var obs []kragObs
	for _, mode := range []string{"unwritten", "observer", "unpassable", "contract", "environment"} {
		obs = append(obs, refusal(mode))
	}
	unwritten := kragHeadlines["unwritten"] + "known refusal\n- known item"
	obs = append(obs,
		kragObs{mode: "plain", refinerRequests: 1, intent: refTitled, steps: shpSteps},
		kragObs{mode: "red", refinerRequests: 2, refineFeedback: []string{ack}, intent: refTitled, steps: shpSteps},
		kragObs{mode: "repair", shapeFeedback: unwritten, refinerRequests: 3, refineFeedback: []string{unwritten, ack}, intent: refTitled, steps: kragBehavioural},
	)
	if bad {
		o := &obs[0]
		o.refineFeedback, o.refinerRequests, o.exit, o.err, o.lines, o.unchanged, o.intent = []string{ack}, 2, 0, "", "", false, refTitled
	}
	for _, o := range obs {
		kragCheck(t, o)
	}
}

func TestKRefineAcceptanceGatesAcceptance(t *testing.T) {
	mode := os.Getenv("KRAG_MODE")
	if mode == "good" || mode == "bad" {
		kragControls(t, mode == "bad")
		return
	}
	o := kragRefine(t, mode)
	switch mode {
	case "plain", "red":
	case "repair":
		var detail string
		o.shapeFeedback, o.shapeContract, detail = kragShape(t, "unwritten")
		o.detail += "\n" + detail
	default:
		var detail string
		o.shapeFeedback, o.shapeContract, detail = kragShape(t, mode)
		o.detail += "\n" + detail
	}
	kragCheck(t, o)
}
`;

let built;
// quiet keeps quoted fixture output from reading as this acceptance's own test-runner output: the
// synthetic reports quote a go test -run selector, minitest name filters and minitest summaries
// that no acceptance step runs.
const quiet = (text) => text
  .replace(/-run([= ])/g, '-run:$1')
  .replace(/(-n|-i|--name|--include)([\s=])/g, '$1:$2')
  .replace(/ assertions, /g, ' assertions; ');
function env(mode) {
  const e = {...process.env};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  if (mode) e.KRAG_MODE = mode;
  return e;
}
// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'krag-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const harness = path.join(dir, 'harness_test.go');
  fs.writeFileSync(harness, HARNESS);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[TARGET]: harness}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 200000});
  built = r.status === 0 ? {bin} : {error: quiet(`${TEST}: go test -c -vet=off -overlay (${TARGET}) ./shape build failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`)};
  return built;
}
function run(mode) {
  const b = build();
  if (b.error) return {ok: false, detail: b.error};
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=150s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env(mode), encoding: 'utf8', timeout: 180000, maxBuffer: 16 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  return {ok: !r.error && r.status === 0 && text.includes('--- PASS: ' + TEST + ' '),
    detail: quiet(`${TEST} (KRAG_MODE=${mode}): shape.test -test.v filtered to ${TEST} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`)};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('k-refine-acceptance-gates: the {string} journey runs', {timeout: 400000}, function (mode) { this.krag = run(mode); });
  Then('k-refine-acceptance-gates: Refine judges the refined draft as Shape does', function () { assert.ok(this.krag.ok, this.krag.detail); });
}
