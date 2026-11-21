'use strict';
// shape-refine-rejects-dropped-control: scripted Refine journeys on a fixture draft with two working
// wrong-implementation controls, run by the real Cucumber runner. The Go harness is compiled into the
// shape package's tests through a Go overlay; SRDC_MODE picks the journey.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestSrdcAcceptance';
const TARGET = 'shape/shape_refine_rejects_dropped_control_acceptance_test.go';
const HARNESS = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
)

const srdcTest = "TestSrdcAcceptance"
const srdcControlsPath = ".kogen/intents/is-even/controls.json"

// The fixture draft's real steps assert on calc.js, so a wrong calc.js fails exactly its scenario with an AssertionError.
const srdcSteps = "const { Given, Then } = require('@cucumber/cucumber');\nconst assert = require('node:assert');\nGiven('is-even: the number {int}', function (n) { this.n = n; });\nThen('is-even: it is even', function () { assert.strictEqual(require('../../calc.js').isEven(this.n), true); });\nThen('is-even: it is not even', function () { assert.strictEqual(require('../../calc.js').isEven(this.n), false); });\n"

const srdcGood = "exports.isEven = n => n % 2 === 0;\n"

type srdcWrong struct {
	name, scenario, step, calc string
}

var (
	srdcEven = srdcWrong{"wrong-even", "Even numbers", "Then is-even: it is even", "exports.isEven = n => false;\n"}
	srdcOdd  = srdcWrong{"wrong-odd", "Odd numbers", "Then is-even: it is not even", "exports.isEven = n => true;\n"}
)

func srdcControls(wrong ...srdcWrong) string {
	var list []any
	for _, w := range wrong {
		list = append(list, map[string]any{"name": w.name, "scenario": w.scenario, "step": w.step, "files": map[string]any{"calc.js": w.calc}})
	}
	doc := map[string]any{"correct": map[string]any{"files": map[string]any{"calc.js": srdcGood}}, "wrong": list}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return string(b) + "\n"
}

// srdcMerged is a control of the odd scenario whose name supersedes the named original with the given reason.
func srdcMerged(original, scenario, reason string) srdcWrong {
	return srdcWrong{"wrong-merged supersedes \"" + original + "\" of \"" + scenario + "\" because " + reason, "Odd numbers", "Then is-even: it is not even", "exports.isEven = () => true;\n"}
}

// srdcText is the error's text and lines plus the stderr of the run.
func srdcText(err error, stderr string) string {
	text := fmt.Sprint(err) + "\n" + stderr
	if e, ok := sys.AsError(err); ok {
		text += "\n" + strings.Join(e.Lines, "\n")
	}
	return text
}

func TestSrdcAcceptance(t *testing.T) {
	mode := os.Getenv("SRDC_MODE")
	modules := os.Getenv("KOGEN_CUCUMBER")
	if modules == "" {
		t.Fatalf("%s: KOGEN_CUCUMBER is unset", srdcTest)
	}
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	var replacement string
	want := []string{}
	accepted := false
	switch mode {
	case "drop":
		replacement = srdcControls(srdcEven)
		want = []string{"wrong-odd", "Odd numbers", "supersedes", "because"}
	case "supersede":
		replacement = srdcControls(srdcEven, srdcMerged("wrong-odd", "Odd numbers", "the merged control covers the odd case"))
		accepted = true
	case "noreason":
		replacement = srdcControls(srdcEven, srdcMerged("wrong-odd", "Odd numbers", ""))
		want = []string{"wrong-odd", "Odd numbers", "supersedes", "because"}
	case "unknown":
		replacement = srdcControls(srdcEven, srdcOdd, srdcMerged("wrong-ghost", "Odd numbers", "it never existed"))
		want = []string{"wrong-ghost"}
	case "keep":
		changed := srdcOdd
		changed.calc = "exports.isEven = n => n >= 0;\n"
		replacement = srdcControls(srdcEven, changed)
		accepted = true
	case "keepweak":
		weak := srdcOdd
		weak.calc = srdcGood
		replacement = srdcControls(srdcEven, weak)
		want = []string{"Wrong-implementation check"}
	default:
		t.Fatalf("%s: unknown SRDC_MODE %q", srdcTest, mode)
	}
	var steps []shpStep
	for i := 1; i <= 3; i++ {
		steps = append(steps, refSubmit(fmt.Sprintf("c%d", i), srdcControlsPath, replacement))
	}
	f := newRefFx(t, refOpts{}, steps...)
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n",
		modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
	// The fixture's own calc_steps.js calls Given without requiring Cucumber, which the real runner cannot load.
	f.write("features/steps/calc_steps.js", "// no steps\n")
	f.write("calc.js", srdcGood)
	f.write(refStepPath, srdcSteps)
	f.write(srdcControlsPath, srdcControls(srdcEven, srdcOdd))
	f.git("add", "-A")
	f.git("commit", "-qm", "fixture with two working controls")
	before := f.read(srdcControlsPath)
	_, err := f.refine(f.hash())
	after := f.read(srdcControlsPath)
	text := srdcText(err, f.Stderr.String())
	if accepted {
		if err != nil || after != replacement {
			t.Errorf("%s (SRDC_MODE=%s): the replacement was not accepted: err=%v controls=%q text=%s", srdcTest, mode, err, after, text)
		}
		return
	}
	if err == nil || after != before {
		t.Errorf("%s (SRDC_MODE=%s): the replacement was not refused with the original controls kept: err=%v controls=%q", srdcTest, mode, err, after)
	}
	feedback := ""
	if reqs := f.Prov.requests(); len(reqs) > 1 {
		feedback = fmt.Sprint(shpOutputs(reqs[1])["c1"])
	}
	for _, w := range want {
		if !strings.Contains(feedback, w) {
			t.Errorf("%s (SRDC_MODE=%s): the refusal does not contain %q: %s", srdcTest, mode, w, feedback)
		}
	}
}
`;

let built;
// quiet keeps quoted fixture output from reading as this acceptance's own test-runner output.
const quiet = (text) => text
  .replace(/-run([= ])/g, '-run:$1')
  .replace(/(-n|-i|--name|--include)([\s=])/g, '$1:$2')
  .replace(/ assertions, /g, ' assertions; ');
function env(mode) {
  const e = {...process.env};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  if (mode) e.SRDC_MODE = mode;
  return e;
}
// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'srdc-'));
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
    detail: quiet(`${TEST} (SRDC_MODE=${mode}): shape.test -test.v filtered to ${TEST} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`)};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('shape-refine-rejects-dropped-control: the {string} journey runs', {timeout: 400000}, function (mode) { this.srdc = run(mode); });
  Then('shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified', function () { assert.ok(this.srdc.ok, this.srdc.detail); });
}
