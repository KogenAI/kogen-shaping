'use strict';
// Acceptance of shape-budget-env-configurable. The Given step compiles the shape package tests with the
// Go harness shape-budget-env-configurable__driver/harness_test.go.txt (copied in through a Go overlay as a
// new test file), runs it once per Cucumber process and reads the JSON observation it wrote to SBEC_OUT.
// The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-budget-env-configurable: ';
const TEST = 'TestShapeBudgetEnvHarness';
const HARNESS = path.resolve('features/steps/shape-budget-env-configurable__driver/harness_test.go.txt');

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sbec-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/shape_budget_env_configurable_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SBEC_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

function shaping(variable) {
  const o = observe();
  const key = variable === 'unset' || variable === 'empty' ? variable : variable === '25' ? 'set_25' : 'value_' + variable;
  assert.ok(o[key], `no shaping observed for the variable ${variable}`);
  return o[key];
}

Given(P + 'shapings have run in one process with the budget variable unset, empty, 25 and other values', function () {
  observe();
});

Then(P + 'with the variable {word} the time line after 10 seconds is {string}', function (variable, want) {
  assert.equal(shaping(variable).time_line, want);
});

Then(P + 'with the variable {word} budget_seconds is {int}', function (variable, want) {
  assert.equal(shaping(variable).budget_seconds, want);
});

Then(P + 'with the variable {word} the budget is {string} at {int} seconds', function (variable, state, seconds) {
  const n = observe().requests_after_jump[`${variable}/${seconds}`];
  assert.ok(Number.isInteger(n), `no observation for ${variable} at ${seconds} s`);
  // A first request that moves the clock by the seconds is the last one exactly when the budget is spent then.
  assert.equal(n === 1, state === 'spent', `with ${variable} at ${seconds} s the shaping made ${n} request(s)`);
});

Then(P + 'the shaping that runs out of budget with the variable 25 ends with a detail starting {string}', function (want) {
  const d = observe().stop_25.detail;
  assert.ok(d.startsWith(want), `detail: ${d}`);
});

Then(P + 'the first problem line of that shaping that states the budget starts with {string}', function (want) {
  const l = observe().stop_25.budget_line;
  assert.ok(l.startsWith(want), `budget line: ${l}`);
});

Then(P + 'that shaping lists the attempt line {string}', function (want) {
  const a = observe().stop_25.attempts;
  assert.ok(a.includes(want), `attempt lines: ${JSON.stringify(a)}`);
});

Then(P + 'with the variable {string} Shape is refused with exit code 3 and class environment before any provider request', function (variable) {
  const s = shaping(variable);
  assert.equal(s.requests, 0, `provider requests made: ${s.requests}`);
  assert.equal(s.error_exit, 3, `error: ${s.error}`);
  assert.equal(s.error_class, 'environment', `error: ${s.error}`);
});

Then(P + 'with the variable {string} the refusal says {string}', function (variable, want) {
  assert.ok(shaping(variable).error.includes(want), `error: ${shaping(variable).error}`);
});

Then(P + 'with the variable {string} Shape runs without refusal and its time line says {string}', function (variable, want) {
  const s = shaping(variable);
  assert.equal(s.error, '');
  assert.ok(s.time_line.includes(want), `time line: ${s.time_line}`);
});
