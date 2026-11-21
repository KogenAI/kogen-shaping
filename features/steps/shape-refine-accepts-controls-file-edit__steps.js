'use strict';
// shape-refine-accepts-controls-file-edit: scripted Refine journeys on a fixture draft whose controls
// file holds a home path. The Go harness is compiled into the shape package's tests through a Go overlay.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestScrfAcceptance';
const TARGET = 'shape/shape_refine_accepts_controls_file_edit_acceptance_test.go';
const HARNESS = String.raw`package shape

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
)

const scrfTest = "TestScrfAcceptance"
const scrfControlsPath = ".kogen/intents/is-even/controls.json"
const scrfRefused = "not an exclusively owned editable draft file"

func scrfControls(home bool) string {
	p := "/work/is-even"
	if home {
		p = "/home/someone/work/is-even"
	}
	return fmt.Sprintf("{\"deferred\":\"the correct implementation is large; checked at %s\"}\n", p)
}

func scrfHasHome(s string) bool { return strings.Contains(s, "/home/") }

// scrfText is the error's text and lines plus the stderr of the run.
func scrfText(err error, stderr string) string {
	text := fmt.Sprint(err) + "\n" + stderr
	if e, ok := sys.AsError(err); ok {
		text += "\n" + strings.Join(e.Lines, "\n")
	}
	return text
}

func TestScrfAcceptance(t *testing.T) {
	mode := os.Getenv("SCRF_MODE")
	target := scrfControlsPath
	newContent := scrfControls(false)
	switch mode {
	case "weak":
		newContent = "{\"correct\":{\"files\":{\"calc.js\":\"x\"}},\"wrong\":[{\"name\":\"same\",\"scenario\":\"Even numbers\",\"step\":\"Then is-even: four is even\",\"files\":{\"calc.js\":\"x\"}}]}\n"
	case "outside":
		target, newContent = "go.sum", "changed\n"
	case "accept":
	default:
		t.Fatalf("%s: unknown SCRF_MODE %q", scrfTest, mode)
	}
	var steps []shpStep
	for i := 1; i <= 3; i++ {
		steps = append(steps, refSubmit(fmt.Sprintf("c%d", i), target, newContent))
	}
	f := newRefFx(t, refOpts{}, steps...)
	f.write(scrfControlsPath, scrfControls(true))
	if mode == "outside" {
		f.write("go.sum", "original\n")
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "fixture controls with a home path")
	before := f.read(target)
	_, err := f.refine(f.hash())
	after := f.read(target)
	text := scrfText(err, f.Stderr.String())
	switch mode {
	case "accept":
		if err != nil || after != newContent || scrfHasHome(f.read(scrfControlsPath)) {
			t.Errorf("%s: the controls edit was not accepted: err=%v controls=%q text=%s", scrfTest, err, f.read(scrfControlsPath), text)
		}
	case "weak":
		if err == nil || after != before {
			t.Errorf("%s: the weaker controls were not refused with the earlier controls kept: err=%v controls=%q", scrfTest, err, after)
		}
		if strings.Contains(text, scrfRefused) {
			t.Errorf("%s: the controls path was refused as not owned: %s", scrfTest, text)
		}
	case "outside":
		if err == nil || after != before {
			t.Errorf("%s: go.sum was not refused or changed: err=%v go.sum=%q", scrfTest, err, after)
		}
		for _, want := range []string{scrfRefused, "go.sum", refIntentPath, refFeaturePath, scrfControlsPath} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: the refusal does not contain %q (it must keep the existing text, name the refused path and list the owned files): %s", scrfTest, want, text)
			}
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
  if (mode) e.SCRF_MODE = mode;
  return e;
}
// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'scrf-'));
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
    detail: quiet(`${TEST} (SCRF_MODE=${mode}): shape.test -test.v filtered to ${TEST} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`)};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('shape-refine-accepts-controls-file-edit: the {string} journey runs', {timeout: 400000}, function (mode) { this.scrf = run(mode); });
  Then('shape-refine-accepts-controls-file-edit: Refine handles the controls file as specified', function () { assert.ok(this.scrf.ok, this.scrf.detail); });
}
