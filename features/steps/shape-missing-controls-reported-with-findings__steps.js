'use strict';
// Acceptance of shape-missing-controls-reported-with-findings. The Given step compiles the shape
// package's tests with the harness of shape-missing-controls-reported-with-findings__harness.js
// (through a Go overlay), runs its test once per Cucumber process and reads the JSON observation
// it wrote to SMCRWF_OUT. The Then steps compare one case of it with the feature's values.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const TEST = 'TestShapeMissingControlsReportedWithFindingsAcceptance';
const TARGET = 'shape/shape_missing_controls_reported_with_findings_acceptance_test.go';
const P = 'shape-missing-controls-reported-with-findings: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function harness() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'smcrwf-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-missing-controls-reported-with-findings__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 400000, maxBuffer: 32 * 1024 * 1024});
  if (b.status !== 0) {
    throw new Error(`building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}${b.error ? ' ' + b.error.message : ''}:\n${b.stdout}${b.stderr}`);
  }
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SMCRWF_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const output = (r.stdout || '') + (r.stderr || '');
  const shown = `${TEST} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${output}`;
  assert.equal(r.error, undefined, shown);
  assert.equal(r.status, 0, shown);
  const passes = output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation file: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function observed(o, name) {
  assert.ok(o && Object.hasOwn(o, name), `the harness observed no case "${name}":\n${JSON.stringify(o, null, 1)}`);
  return o[name];
}

function startsWith(o, name, prefixes) {
  const c = observed(o, name);
  const shown = JSON.stringify(c, null, 1);
  assert.equal(c.problems.length, prefixes.length, `case "${name}": ${c.problems.length} problem lines, want ${prefixes.length}:\n${shown}`);
  prefixes.forEach((p, i) => assert.ok(c.problems[i].startsWith(p), `case "${name}": problem ${i + 1} does not start with ${JSON.stringify(p)}:\n${shown}`));
}

let cached = null;
Given(P + 'the scripted shaping rounds have run', {timeout: 660000}, function () {
  if (cached === null) cached = harness();
  this.smcrwf = cached;
});
Then(P + 'the problems of {string} start with:', function (name, text) {
  startsWith(this.smcrwf, name, JSON.parse(text));
});
Then(P + 'the problems of {string} start with the contract finding lines', function (name) {
  const c = observed(this.smcrwf, name);
  assert.ok(c.problems.length >= 3, `case "${name}": too few problem lines:\n${JSON.stringify(c, null, 1)}`);
  assert.ok(c.problems[0].startsWith('- Even numbers | expectation: '), `case "${name}": first problem:\n${JSON.stringify(c, null, 1)}`);
  assert.ok(c.problems[1].startsWith('- fix for Even numbers: in '), `case "${name}": second problem:\n${JSON.stringify(c, null, 1)}`);
});
Then(P + 'the problems of {string} include the line:', function (name, text) {
  const c = observed(this.smcrwf, name);
  assert.ok(c.problems.includes(text.trim()), `case "${name}": no problem line equals ${JSON.stringify(text.trim())}:\n${JSON.stringify(c, null, 1)}`);
});
Then(P + 'the feedback of {string} has {int} lines', function (name, n) {
  const c = observed(this.smcrwf, name);
  assert.equal(c.feedback.split('\n').length, n, `case "${name}": feedback line count:\n${c.feedback}`);
});
Then(P + 'the feedback of {string} starts with {string}', function (name, text) {
  const c = observed(this.smcrwf, name);
  assert.ok(c.feedback.startsWith(text), `case "${name}": feedback does not start with ${JSON.stringify(text)}:\n${c.feedback}`);
});
Then(P + 'the runs of {string} are {int}', function (name, n) {
  const c = observed(this.smcrwf, name);
  assert.equal(c.runs, n, `case "${name}": ${c.runs} wrong-implementation runs, want ${n}`);
});
