'use strict';
// Acceptance of shape-example-contradiction-line-names-fields. The When step compiles the shape package
// tests with the Go harness shape-example-contradiction-line-names-fields__driver/harness_test.go.txt (copied
// in through a Go overlay as a new test file), gives it one Examples row in ECLNF_IN, and reads from ECLNF_OUT
// the contradiction line the helper built and the same line after cutProblems. The Then steps compare them
// with the feature. A literal backslash-n in the why text of the feature stands for a newline.
// On a tree where shape/examples.go has no exampleContradictionLine yet, the harness is rewritten to build the
// old two-value line instead, so that it still compiles and runs and the Then steps fail on the line text.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {When, Then} = require('@cucumber/cucumber');

const P = 'shape-example-contradiction-line-names-fields: ';
const TEST = 'TestExampleContradictionLineHarness';
const HARNESS = path.resolve('features/steps/shape-example-contradiction-line-names-fields__driver/harness_test.go.txt');
const EXAMPLES = path.resolve('shape/examples.go');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function harnessSource() {
  const text = fs.readFileSync(HARNESS, 'utf8');
  if (fs.readFileSync(EXAMPLES, 'utf8').includes('func exampleContradictionLine(')) return text;
  return text.replace('exampleContradictionLine(in.', 'eclnfBaselineLine(in.') +
    '\nfunc eclnfBaselineLine(scenario, expected, derived, why string) string {\n\treturn scenario + ": expected " + expected + ", rules give " + derived\n}\n';
}

When(P + 'the helper builds the contradiction line for scenario {string}, expected {string}, derived {string} and why {string}', function (scenario, expected, derived, why) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'eclnf-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const harness = path.join(dir, 'harness_test.go');
  fs.writeFileSync(harness, harnessSource());
  const target = path.resolve('shape/shape_example_contradiction_line_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: harness}}));
  const out = path.join(dir, 'observation.json');
  const input = JSON.stringify({scenario, expected, derived, why: why.replace(/\\n/g, '\n')});
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({ECLNF_IN: input, ECLNF_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  this.run = {r, shown, out};
});

function observed(world) {
  const {r, shown, out} = world.run;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

Then(P + 'the line is exactly {string}', function (want) {
  assert.equal(observed(this).line, want);
});

Then(P + 'the cut line is at most {int} runes', function (max) {
  const n = [...observed(this).cut].length;
  assert.ok(n <= max, `the cut line has ${n} runes: ${observed(this).cut}`);
});

Then(P + 'the cut line starts with {string}', function (want) {
  const cut = observed(this).cut;
  assert.ok(cut.startsWith(want), `the cut line does not start with ${want}: ${cut}`);
});

Then(P + 'the cut line contains {string}', function (want) {
  const cut = observed(this).cut;
  assert.ok(cut.includes(want), `the cut line does not contain ${want}: ${cut}`);
});

Then(P + 'the cut line ends with {string}', function (want) {
  const cut = observed(this).cut;
  assert.ok(cut.endsWith(want), `the cut line does not end with ${want}: ${cut}`);
});
