'use strict';
// Acceptance of shape-kept-repeat-outcome-names-scenarios. The Given step compiles the shape package tests with
// the Go harness shape-kept-repeat-outcome-names-scenarios__driver/harness_test.go.txt (copied in through a Go
// overlay as a new test file), runs it once per Cucumber process and reads the JSON observation it wrote to
// KRONS_OUT: for each of three cases the stderr progress and the kept_assertion_repeat events.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-kept-repeat-outcome-names-scenarios: ';
const TEST = 'TestKeptRepeatOutcomeNamesHarness';
const HARNESS = path.resolve('features/steps/shape-kept-repeat-outcome-names-scenarios__driver/harness_test.go.txt');

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'krons-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/kept_repeat_outcome_names_scenarios_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({KRONS_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

function caseOf(name) {
  const c = observe()[name];
  assert.ok(c, `no observation for case ${name}`);
  return c;
}

function repeatEvent(name) {
  const events = caseOf(name).events;
  assert.equal(events.length, 1, `case ${name}: expected one kept_assertion_repeat event: ${JSON.stringify(events)}`);
  return events[0];
}

Given(P + 'the kept assertion repeat has run on three local fixtures', function () {
  observe();
});

Then(P + 'the progress line for case {string} is {string}', function (name, want) {
  const prefix = 'kogen: kept assertion repeat outcome:';
  const lines = caseOf(name).stderr.split('\n').filter(l => l.startsWith(prefix));
  assert.equal(lines.length, 1, `case ${name}: expected one outcome line: ${caseOf(name).stderr}`);
  assert.equal(lines[0].replace(/ \(\d+ s\)$/, ''), want);
});

Then(P + 'the repeat event for case {string} lists scenarios {string} in that order', function (name, want) {
  const listed = repeatEvent(name).scenarios;
  assert.ok(Array.isArray(listed), `case ${name}: the event has no scenarios list: ${JSON.stringify(repeatEvent(name))}`);
  for (const s of listed) assert.deepEqual(Object.keys(s).sort(), ['line', 'name']);
  assert.equal(listed.map(s => `${s.name}@${s.line}`).join(', '), want);
});

Then(P + 'the repeat event for case {string} has no scenarios key', function (name) {
  const ev = repeatEvent(name);
  assert.ok(!('scenarios' in ev), `case ${name}: the event has a scenarios key: ${JSON.stringify(ev)}`);
});
