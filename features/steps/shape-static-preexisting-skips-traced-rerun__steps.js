'use strict';
// Acceptance of shape-static-preexisting-skips-traced-rerun. The Given step compiles the shape
// package tests with the Go harness shape-static-preexisting-skips-traced-rerun__driver/harness_test.go.txt
// (copied in through a Go overlay as a new test file), runs it once per Cucumber process and reads the
// JSON observation it wrote to SPSTR_OUT. The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-static-preexisting-skips-traced-rerun: ';
const TEST = 'TestStaticPreexistingSkipsTracedHarness';
const HARNESS = path.resolve('features/steps/shape-static-preexisting-skips-traced-rerun__driver/harness_test.go.txt');

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'spstr-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/static_preexisting_skips_traced_rerun_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SPSTR_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

function roundOf(check, n) {
  const rounds = observe()[check];
  assert.ok(rounds && rounds[n - 1], `no observation of the ${check} check in round ${n}`);
  return rounds[n - 1];
}

Given(P + 'the static gate has judged a preexisting, an attributable and a flaky check', function () {
  observe();
});

Then(P + 'the {word} check ran {int} times in round {int}, {int} of them under set -x', function (check, total, n, traced) {
  const r = roundOf(check, n);
  const got = r.runs;
  assert.equal(got.length, total, `the ${check} check ran ${got.length} times in round ${n} (${got.join(', ')}), want ${total}`);
  assert.equal(got.filter(x => x === 'traced').length, traced, `the ${check} check ran ${got.filter(x => x === 'traced').length} times under set -x in round ${n} (${got.join(', ')}), want ${traced}`);
});

Then(P + 'the {word} check in round {int} is recorded {string} with paths {string} and an empty trace', function (check, n, status, paths) {
  const r = roundOf(check, n);
  assert.equal(r.status, status);
  assert.equal((r.paths || []).join(', '), paths);
  assert.equal(r.trace, '', 'the recorded trace is not empty');
});

Then(P + 'the {word} check in round {int} is recorded {string} with paths {string}', function (check, n, status, paths) {
  const r = roundOf(check, n);
  assert.equal(r.status, status);
  assert.equal((r.paths || []).join(', '), paths);
});

Then(P + 'the {word} check in round {int} reports that the base result was reused', function (check, n) {
  assert.ok(roundOf(check, n).progress.includes('without the shaped files too (base result reused): pre-existing failure in'), roundOf(check, n).progress);
});

Then(P + 'the {word} check is recorded {string} with a non-empty trace', function (check, status) {
  const r = roundOf(check, 1);
  assert.equal(r.status, status);
  assert.notEqual(r.trace, '', 'the recorded trace is empty');
});

Then(P + 'the flaky check records failing lines {string} and traced failing lines {string}', function (failing, traced) {
  const r = roundOf('flaky', 1);
  assert.equal((r.failing || []).join('|'), failing);
  assert.equal((r.traced_failing || []).join('|'), traced);
});

Then(P + 'the static run outcome of the {word} check is {string}', function (check, outcome) {
  assert.equal(roundOf(check, 1).outcome, outcome);
});
