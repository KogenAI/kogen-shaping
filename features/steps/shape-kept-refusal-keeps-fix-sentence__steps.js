'use strict';
// Acceptance of shape-kept-refusal-keeps-fix-sentence. The Given step compiles the shape package
// tests with the Go harness shape-kept-refusal-keeps-fix-sentence__driver/harness_test.go.txt (copied in
// through a Go overlay as a new test file). The harness builds a kept property's refusal line with
// controlsPropertyLine, passes it through cutProblems and writes both to SKRKFS_OUT.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-kept-refusal-keeps-fix-sentence: ';
const TEST = 'TestKeptRefusalFixHarness';
const HARNESS = path.resolve('features/steps/shape-kept-refusal-keeps-fix-sentence__driver/harness_test.go.txt');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function run(input) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'skrkfs-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/kept_refusal_keeps_fix_sentence_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SKRKFS_IN: JSON.stringify(input), SKRKFS_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

let observation = null;

Given(P + 'a kept step named {string} whose request text is {string} repeated {int} time(s) and whose problem is {string}', function (name, unit, times, problem) {
  const request = Array(times).fill(unit).join(' ');
  observation = run({Property: name, Request: request, Problem: problem});
  assert.equal(observation.cut.length, 1, 'the cut returned ' + observation.cut.length + ' lines');
});

function line() {
  assert.ok(observation, 'no refusal line was produced');
  return observation.cut[0];
}

Then(P + 'the refusal line ends with {string}', function (want) {
  const got = line();
  assert.ok(got.endsWith(want), `the refusal line does not end with the problem and fix sentence.\nwant suffix: ${want}\nline: ${got}`);
});

Then(P + 'the refusal line starts with {string}', function (want) {
  const got = line();
  assert.ok(got.startsWith(want), `the refusal line does not start with the request quote.\nwant prefix: ${want}\nline: ${got}`);
});

Then(P + 'the request text is shortened with the cut marker before the problem', function () {
  const got = line();
  assert.ok(got.includes(' ...  | problem: '), `the request text is not shortened by the cut marker before the problem: ${got}`);
});

Then(P + 'the refusal line is {int} runes long', function (want) {
  assert.equal([...line()].length, want, line());
});

Then(P + 'the refusal line is {string}', function (want) {
  assert.equal(line(), want);
});
