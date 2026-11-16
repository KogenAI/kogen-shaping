'use strict';
// Acceptance of shape-budget-stop-lists-last-submission-checks. The Given step runs the Go harness
// features/steps/shape-budget-stop-lists-last-submission-checks__driver/harness_test.go.txt (copied in through a Go overlay as a new
// test file of the shape package) once per Cucumber process; it writes the budget refusal's detail and
// lines as JSON to BSLLSC_OUT, and the Then steps compare those lines with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-budget-stop-lists-last-submission-checks: ';
const TEST = 'TestBudgetStopLastSubmissionHarness';
const HARNESS = path.resolve('features/steps/shape-budget-stop-lists-last-submission-checks__driver/harness_test.go.txt');

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'bsllsc-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/budget_stop_last_submission_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({BSLLSC_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

Given(P + 'a scripted shaping whose second submission passes the cheap gates and whose checks end 60 s past the budget has stopped', function () {
  observe();
});

Then(P + 'the refusal lists the line {string}', function (want) {
  const o = observe();
  assert.ok(o.lines.includes(want), `the refusal does not list the line ${JSON.stringify(want)}; its lines are:\n${o.lines.join('\n')}`);
});

Then(P + 'the refusal lists {int} attempt lines', function (want) {
  const o = observe();
  const got = o.lines.filter(l => l.startsWith('attempt '));
  assert.equal(got.length, want, `attempt lines:\n${got.join('\n')}`);
});
