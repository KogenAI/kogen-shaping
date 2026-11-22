'use strict';
// Acceptance of shape-unnamed-property-says-fix. The Given step runs the shape package test
// shape-unnamed-property-says-fix__driver/harness_test.go.txt (copied in through a Go overlay as a new
// test file). The harness judges one property of a two-scenario feature with controlsJudgeAll, the
// judging of controls.json, and writes the refusal lines and the same lines after cutProblems to SUPSF_OUT.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-unnamed-property-says-fix: ';
const TEST = 'TestUnnamedPropertyHarness';
const HARNESS = path.resolve('features/steps/shape-unnamed-property-says-fix__driver/harness_test.go.txt');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function run(input) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'supsf-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/unnamed_property_says_fix_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SUPSF_IN: JSON.stringify(input), SUPSF_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

const wrongs = {
  'there is no wrong implementation': 'none',
  'a wrong implementation names only another property': 'another',
  'a wrong implementation names it but its run did not report every scenario': 'incomplete',
  'a wrong implementation names it but fails at a later step': 'elsewhere',
  'a wrong implementation names it and is rejected at its step': 'rejected',
};

let observation = null;

Given(P + 'a {word} property named {string} with request text {string} repeated {int} time(s) is judged where {}', function (kind, name, unit, times, where) {
  assert.ok(wrongs[where], 'unknown situation: ' + where);
  observation = run({Kind: kind, Property: name, Request: Array(times).fill(unit).join(' '), Wrong: wrongs[where]});
});

function obs() {
  assert.ok(observation, 'nothing was judged');
  return observation;
}

function cutLine() {
  const cut = obs().cut;
  assert.equal(cut.length, 1, 'the cut returned ' + cut.length + ' lines: ' + JSON.stringify(cut));
  return cut[0];
}

Then(P + 'the refusal lines are exactly:', function (doc) {
  assert.deepEqual(obs().lines || [], doc.split('\n'));
});

Then(P + 'there are no refusal lines', function () {
  assert.deepEqual(obs().lines || [], []);
});

Then(P + 'the cut refusal line is {int} runes long', function (want) {
  assert.equal([...cutLine()].length, want, cutLine());
});

Then(P + 'the cut refusal line starts with {string}', function (want) {
  const got = cutLine();
  assert.ok(got.startsWith(want), `the line does not start with the property and request text.\nwant prefix: ${want}\nline: ${got}`);
});

Then(P + 'the cut refusal line ends with {string}', function (want) {
  const got = cutLine();
  assert.ok(got.endsWith(want), `the line does not end with the whole sentence.\nwant suffix: ${want}\nline: ${got}`);
});
