'use strict';
// Acceptance of shape-refuses-go-run-for-exit-code-steps. The Given step compiles the shape package tests with the
// Go harness in shape-refuses-go-run-for-exit-code-steps__driver/harness_test.go.txt (added through a Go overlay as a
// new test file), runs it once per Cucumber process and reads the JSON map from fixture name to the feedback text of
// stepSelfcheck that it wrote to SRGR_OUT. The Then steps compare it with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-refuses-go-run-for-exit-code-steps: ';
const TEST = 'TestShapeRefusesGoRunHarness';
const HARNESS = path.resolve('features/steps/shape-refuses-go-run-for-exit-code-steps__driver/harness_test.go.txt');
const HEADLINE = 'Step self-check failed; nothing was published. Fix the step files and resubmit.';

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'srgr-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/shape_refuses_go_run_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SRGR_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

function feedbackOf(name) {
  const o = observe();
  assert.ok(Object.prototype.hasOwnProperty.call(o, name), `no observation for fixture ${name}: ${JSON.stringify(Object.keys(o))}`);
  return o[name];
}

Given(P + 'the self-check ran on the fixture step files', function () {
  observe();
});

Then(P + 'the self-check of the {string} fixture refuses with the line {string}', function (name, line) {
  assert.equal(feedbackOf(name), HEADLINE + '\n' + line, `fixture ${name}`);
});

Then(P + 'the self-check of the {string} fixture refuses with the lines {string} and {string}', function (name, first, second) {
  assert.equal(feedbackOf(name), HEADLINE + '\n' + first + '\n' + second, `fixture ${name}`);
});

Then(P + 'the self-check of the {string} fixture accepts the step file', function (name) {
  assert.equal(feedbackOf(name), '', `fixture ${name} was refused`);
});
