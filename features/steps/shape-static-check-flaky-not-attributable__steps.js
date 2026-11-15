'use strict';
// Acceptance of shape-static-check-flaky-not-attributable. The Given step compiles the shape package
// tests with the Go harness shape-static-check-flaky-not-attributable__driver/harness_test.go.txt (copied
// in through a Go overlay as a new test file), runs it once per Cucumber process and reads the JSON
// observation it wrote to SSCFNA_OUT. The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-static-check-flaky-not-attributable: ';
const TEST = 'TestStaticFlakyHarness';
const HARNESS = path.resolve('features/steps/shape-static-check-flaky-not-attributable__driver/harness_test.go.txt');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let observation = null;
function observe() {
  if (observation !== null) return observation;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sscfna-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/shape_static_check_flaky_not_attributable_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SSCFNA_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

function check(group, name) {
  const c = observe()[group].checks[name];
  assert.ok(c, `no check ${name} in group ${group}`);
  return c;
}

Given(P + 'the static gate has run a check that fails differently on its two runs and checks that fail the same way twice or name a shaped file', function () {
  observe();
});

Then(P + 'the unsteady check has status {string}', function (want) {
  assert.equal(check('unsteady', 'unsteady').status, want);
});

Then(P + 'the run with only the unsteady check has outcome {string}', function (want) {
  assert.equal(observe().unsteady.outcome, want);
});

Then(P + 'the unsteady check progress line is {string}', function (want) {
  assert.deepEqual(check('unsteady', 'unsteady').progress, [want]);
});

Then(P + 'the unsteady check records the failing lines {string} for the plain run and {string} for the traced run', function (plain, traced) {
  const c = check('unsteady', 'unsteady');
  assert.deepEqual(c.failing, JSON.parse(plain));
  assert.deepEqual(c.traced_failing, JSON.parse(traced));
});

Then(P + 'the unsteady check exit is {int}', function (want) {
  assert.equal(check('unsteady', 'unsteady').exit, want);
});

Then(P + 'the check {word} is {string} with files {string}', function (name, status, files) {
  const c = check('refused', name);
  assert.equal(c.status, status);
  assert.deepEqual(c.paths, files === '' ? [] : files.split(', '));
  assert.ok(c.failing === undefined && c.traced_failing === undefined, 'a refused check records no failing sets');
});

Then(P + 'the run with the three repeating or path-naming checks has outcome {string}', function (want) {
  assert.equal(observe().refused.outcome, want);
});

Then(P + 'the progress line of the check {word} is {string}', function (name, want) {
  assert.deepEqual(check('refused', name).progress, [want]);
});
