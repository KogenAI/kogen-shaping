'use strict';
// Acceptance of shape-uncovered-item-line-names-fix. The Given steps run the shape package test
// shape-uncovered-item-line-names-fix__driver/harness_test.go.txt (copied in through a Go overlay as a
// new test file), which judges one property with controlsJudgeAll or applyPropertyVerdicts and writes
// the lines they return to UIFIX_OUT.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-uncovered-item-line-names-fix: ';
const TEST = 'TestUncoveredFixHarness';
const HARNESS = path.resolve('features/steps/shape-uncovered-item-line-names-fix__driver/harness_test.go.txt');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function run(input) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'uifix-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/uncovered_fix_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({UIFIX_IN: JSON.stringify(input), UIFIX_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8')).lines || [];
}

const wrongs = {'no wrong implementation is given': 'none', 'a wrong implementation names only another scenario': 'another', 'a wrong implementation rejects it at its step': 'rejected'};
const problems = {
  uncovered: 'no scenario exercises it',
  passes_unchanged: 'every scenario that exercises it passes on the unchanged tree, so none is red at an assertion about it',
};

let lines = null;

Given(P + 'a {word} item named {string} with request text {string} declares {int} scenario(s) and {}', function (kind, name, request, count, where) {
  assert.ok(wrongs[where], 'unknown situation: ' + where);
  lines = run({Mode: 'controls', Kind: kind, Property: name, Request: request, Declared: Array.from({length: count}, (_, i) => i), Wrong: wrongs[where]});
});

Given(P + 'a {word} item named {string} with request text {string} has a confirmed {word} problem', function (kind, name, request, reason) {
  assert.ok(problems[reason], 'unknown problem: ' + reason);
  lines = run({Mode: 'property', Kind: kind, Property: name, Request: request, Reason: reason, Text: problems[reason]});
});

function got() {
  assert.ok(lines, 'nothing was judged');
  return lines;
}

function line(n) {
  const l = got()[n - 1];
  assert.ok(l !== undefined, `there is no line ${n}; the lines are ${JSON.stringify(got())}`);
  return l;
}

Then(P + 'the lines are exactly:', function (doc) {
  assert.deepEqual(got(), doc.split('\n'));
});

Then(P + 'there are {int} line(s)', function (n) {
  assert.equal(got().length, n, JSON.stringify(got()));
});

Then(P + 'line {int} contains {string}', function (n, text) {
  assert.ok(line(n).includes(text), `line ${n} does not contain ${JSON.stringify(text)}: ${line(n)}`);
});

Then(P + 'line {int} does not contain {string}', function (n, text) {
  assert.ok(!line(n).includes(text), `line ${n} contains ${JSON.stringify(text)}: ${line(n)}`);
});

Then(P + 'no line starts with {string}', function (text) {
  assert.deepEqual(got().filter(l => l.startsWith(text)), [], JSON.stringify(got()));
});
