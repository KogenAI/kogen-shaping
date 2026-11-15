'use strict';
// Acceptance of shape-runner-observer-refusal-shows-step-failure. The Given step compiles the shape
// package's tests with the Go harness of shape-runner-observer-refusal-shows-step-failure__harness.js
// (through a Go overlay), runs it once per Cucumber process and reads the JSON observation it wrote
// to SROR_OUT: the items of the runner-observer refusal for several failed scenarios. The Then steps
// compare those items with the feature's values. A failure message shows both sides with the
// " runs," of a test-runner summary written " runs;", so that the message of this step is never
// itself mistaken for a test runner's passing summary.
const {Given, Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestShapeRunnerObserverRefusalShowsStepFailureAcceptance';
const TARGET = 'shape/shape_runner_observer_refusal_acceptance_test.go';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let observation = null;

function observe() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sror-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-runner-observer-refusal-shows-step-failure__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
  assert.equal(b.status, 0, `go ${build.join(' ')} exited ${b.status}${b.error ? ' ' + b.error.message : ''}:\n${b.stdout}${b.stderr}`);
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SROR_OUT: out}), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
  const shown = `${TEST} (shape.test ${args.join(' ')}) exited ${r.status}:\n${r.stdout}${r.stderr}`;
  assert.equal(r.status, 0, shown);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function items(name) {
  assert.ok(observation && Object.hasOwn(observation, name), `the harness observed no case "${name}": ${JSON.stringify(observation).replace(/ runs,/g, ' runs;')}`);
  return observation[name].items;
}

function same(name, got, want) {
  const show = v => JSON.stringify(v).replace(/ runs,/g, ' runs;');
  assert.ok(JSON.stringify(got) === JSON.stringify(want),
    `the refusal items of case "${name}" differ (" runs;" stands for " runs,"):\n  got:  ${show(got)}\n  want: ${show(want)}`);
}

const P = 'shape-runner-observer-refusal-shows-step-failure: ';

Given(P + 'the runner-observer refusal harness has run', function () {
  if (observation === null) observation = observe();
});

Then(P + 'the only item of {string} is {string}', function (name, want) {
  same(name, items(name), [want]);
});

Then(P + 'the only item of {string} is {string} followed by {int} {string} characters', function (name, prefix, count, ch) {
  same(name, items(name), [prefix + ch.repeat(count)]);
});

Then(P + 'the items of {string} are:', function (name, doc) {
  same(name, items(name), JSON.parse(doc));
});
