'use strict';
// Acceptance of approve-static-reuse-line-explains-unknown. The Given step compiles the shape
// package tests with the Go harness approve-static-reuse-line-explains-unknown__driver/harness_test.go.txt
// (copied in through a Go overlay as a new test file), runs it once per Cucumber process and reads the
// JSON observation it wrote to ASRLEU_OUT. The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'approve-static-reuse-line-explains-unknown: ';
const TEST = 'TestApproveStaticReuseLineHarness';
const HARNESS = path.resolve('features/steps/approve-static-reuse-line-explains-unknown__driver/harness_test.go.txt');

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'asrleu-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/approve_static_reuse_line_explains_unknown_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({ASRLEU_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

Given(P + 'approve has reused a recorded checked, unknown and none static-check result', function () {
  observe();
});

Then(P + 'the checked reuse line is {string}', function (want) {
  assert.equal(observe().checked_line, want);
});

Then(P + 'the configured check ran {int} time in total', function (want) {
  assert.equal(observe().runs, want, 'the configured check ran again although a recorded result was reused');
});

Then(P + 'the unknown reuse line is {string}', function (want) {
  const o = observe();
  assert.equal(o.unknown_line, want.replace('<record path>', o.record_path));
});

Then(P + 'the none reuse line is {string}', function (want) {
  assert.equal(observe().none_line, want);
});
