'use strict';
// Acceptance of shape-kept-blocked-lines-one-refusal. The When step compiles the shape package tests
// with the Go harness shape-kept-blocked-lines-one-refusal__driver/harness_test.go.txt (copied in through
// a Go overlay as a new test file). The harness gives two unchanged-tree observations and the kept
// properties to keptStepStates and keptBaselineRefusals and writes the refusals to SKBLOR_OUT.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Before, Given, When, Then} = require('@cucumber/cucumber');

const P = 'shape-kept-blocked-lines-one-refusal: ';
const TEST = 'TestKeptBlockedLinesHarness';
const HARNESS = path.resolve('features/steps/shape-kept-blocked-lines-one-refusal__driver/harness_test.go.txt');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function run(input) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'skblor-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/kept_blocked_lines_one_refusal_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SKBLOR_IN: JSON.stringify(input), SKBLOR_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

let world;
Before(function () { world = {first: {Scenarios: []}, second: {Scenarios: []}, Props: [], refusals: null}; });

function nums(s) { return s.trim() === '' ? [] : s.split(',').map(x => Number(x.trim())); }

function observation(which) {
  assert.ok(which === 'first' || which === 'second', `unknown observation ${which}`);
  return world[which];
}

Given(P + 'the property {string} names the kept steps on lines {string} of the scenario {string}', function (name, lines, scenario) {
  world.Props.push({Property: name, KeptSteps: nums(lines), Scenarios: [scenario]});
});

Given(P + 'the {word} observation shows the scenario {string} failed with {string}, with steps on lines {string} of which those on lines {string} never ran', function (which, scenario, detail, steps, unexercised) {
  observation(which).Scenarios.push({Name: scenario, Status: 'failed', Detail: detail, Steps: nums(steps), Unexercised: nums(unexercised)});
});

Given(P + 'the {word} observation shows the scenario {string} passed', function (which, scenario) {
  observation(which).Scenarios.push({Name: scenario, Status: 'passed', Detail: '', Steps: [], Unexercised: []});
});

When(P + 'the kept baseline refusals are computed', function () {
  const res = run({First: world.first, Second: world.second, Props: world.Props});
  world.refusals = res.refusals;
});

function refusals() {
  assert.ok(Array.isArray(world.refusals), 'the refusals were not computed');
  return world.refusals;
}

const shown = () => '\nrefusals:\n' + refusals().join('\n');

function containing(text) {
  const hit = refusals().filter(r => r.includes(text));
  assert.equal(hit.length, 1, `${hit.length} refusals contain ${JSON.stringify(text)}, want 1${shown()}`);
  return hit[0];
}

Then(P + 'there are exactly {int} refusal(s)', function (n) {
  assert.equal(refusals().length, n, `got ${refusals().length} refusals, want ${n}${shown()}`);
});

Then(P + 'exactly {int} refusal(s) contain(s) {string}', function (n, text) {
  const hit = refusals().filter(r => r.includes(text));
  assert.equal(hit.length, n, `${hit.length} refusals contain ${JSON.stringify(text)}, want ${n}${shown()}`);
});

Then(P + 'the only refusal contains {string}', function (text) {
  assert.equal(refusals().length, 1, `got ${refusals().length} refusals, want 1${shown()}`);
  assert.ok(refusals()[0].includes(text), `the refusal lacks ${JSON.stringify(text)}${shown()}`);
});

Then(P + 'the only refusal contains {string} {int} time(s)', function (text, n) {
  assert.equal(refusals().length, 1, `got ${refusals().length} refusals, want 1${shown()}`);
  assert.equal(refusals()[0].split(text).length - 1, n, `the refusal does not contain ${JSON.stringify(text)} ${n} time(s)${shown()}`);
});

Then(P + 'the only refusal does not contain the word {string}', function (word) {
  assert.equal(refusals().length, 1, `got ${refusals().length} refusals, want 1${shown()}`);
  assert.ok(!new RegExp('\\b' + word + '\\b').test(refusals()[0]), `the refusal contains the word ${word}${shown()}`);
});

Then(P + 'the refusal containing {string} contains {string}', function (key, text) {
  const r = containing(key);
  assert.ok(r.includes(text), `the refusal containing ${JSON.stringify(key)} lacks ${JSON.stringify(text)}${shown()}`);
});

Then(P + 'the refusal containing {string} does not contain {string}', function (key, text) {
  const r = containing(key);
  assert.ok(!r.includes(text), `the refusal containing ${JSON.stringify(key)} contains ${JSON.stringify(text)}${shown()}`);
});

Then(P + 'the refusal containing {string} does not contain the word {string}', function (key, word) {
  const r = containing(key);
  assert.ok(!new RegExp('\\b' + word + '\\b').test(r), `the refusal containing ${JSON.stringify(key)} contains the word ${word}${shown()}`);
});
