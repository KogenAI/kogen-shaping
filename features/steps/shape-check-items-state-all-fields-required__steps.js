'use strict';
// Acceptance of shape-check-items-state-all-fields-required. The Given step compiles the shape
// package tests with the Go harness shape-check-items-state-all-fields-required__driver/harness_test.go.txt
// (copied in through a Go overlay as a new test file), runs it once per Cucumber process and reads the
// JSON observation it wrote to SCIAFR_OUT. The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-check-items-state-all-fields-required: ';
const TEST = 'TestSciafrAcceptanceHarness';
const HARNESS = path.resolve('features/steps/shape-check-items-state-all-fields-required__driver/harness_test.go.txt');

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sciafr-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/sciafr_acceptance_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SCIAFR_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

function field(o, key) {
  assert.ok(Object.prototype.hasOwnProperty.call(o, key), `the observation has no ${key}`);
  return o[key];
}

Given(P + 'the literal and decision check instructions and tool schemas were observed', function () {
  observe();
});

Then(P + 'the {word} instructions contain {string}', function (kind, want) {
  const text = field(observe(), `${kind}_instructions`);
  assert.ok(text.includes(want), `the ${kind} instructions do not contain ${JSON.stringify(want)}:\n${text}`);
});

Then(P + 'the {word} {word} property description contains {string}', function (kind, prop, want) {
  const text = field(observe(), `${kind}_${prop}_desc`);
  assert.ok(text.includes(want), `the ${kind} ${prop} property description does not contain ${JSON.stringify(want)}: ${JSON.stringify(text)}`);
});

Then(P + 'the {word} item required list is {string}', function (kind, want) {
  assert.equal(field(observe(), `${kind}_required`).join(','), want);
});

Then(P + 'the {word} schema top-level required list is {string}', function (kind, want) {
  assert.equal(field(observe(), `${kind}_top_required`).join(','), want);
});

Then(P + 'the {word} {word} enum is {string}', function (kind, prop, want) {
  assert.equal(field(observe(), `${kind}_${prop}_enum`).join(','), want);
});

Then(P + 'the {word} item schema has {int} properties, forbids additional properties and its tool is strict', function (kind, n) {
  const o = observe();
  assert.equal(field(o, `${kind}_property_count`), n);
  assert.equal(field(o, `${kind}_no_extra`), true, `the ${kind} item schema allows additional properties`);
  assert.equal(field(o, `${kind}_tool_strict`), true, `the ${kind} tool is not strict`);
});
