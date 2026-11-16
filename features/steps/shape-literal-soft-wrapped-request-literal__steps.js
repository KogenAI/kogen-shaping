'use strict';
// Acceptance of shape-literal-soft-wrapped-request-literal. The Given step compiles the shape
// package's tests with the harness of shape-literal-soft-wrapped-request-literal__harness.js
// (through a Go overlay), runs TestShapeLiteralSoftWrappedRequestLiteralAcceptance once per
// Cucumber process and reads the JSON observation the harness wrote to SLSW_OUT. The Then steps
// compare one case of it with the feature's values.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const TEST = 'TestShapeLiteralSoftWrappedRequestLiteralAcceptance';
const TARGET = 'shape/shape_literal_soft_wrapped_request_literal_acceptance_test.go';
const P = 'shape-literal-soft-wrapped-request-literal: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

// judge is the observation of one harness run: the run exited 0, passed the harness test exactly
// once and wrote an observation.
function judge(r) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  return JSON.parse(r.json);
}

// harness builds the shape tests with the harness and runs the harness test once.
function harness() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'slwrl-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-literal-soft-wrapped-request-literal__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
  if (b.status !== 0) {
    throw new Error(`building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}${b.signal ? ' (' + b.signal + ')' : ''}${b.error ? ' ' + b.error.message : ''}:\n${b.stdout}${b.stderr}`);
  }
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=60s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SLSW_OUT: out}), encoding: 'utf8', timeout: 90000, maxBuffer: 32 * 1024 * 1024});
  return judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `SLSW_OUT=${out} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`});
}

let observation = null;

// observed is one case of the observation.
function observed(name) {
  assert.ok(observation !== null, 'the harness has not run');
  assert.ok(Object.hasOwn(observation, name), `the harness observed no case "${name}":\n${JSON.stringify(observation, null, 1)}`);
  return observation[name];
}

// include checks that each listed key of a case has exactly the listed value.
function include(name, want) {
  const c = observed(name);
  const summary = JSON.stringify(c, null, 1);
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(c, key), `case "${name}" has no ${key}:\n${summary}`);
    assert.deepEqual(c[key], value, `case "${name}": ${key} is ${JSON.stringify(c[key])}, want ${JSON.stringify(value)}\nobservation:\n${summary}`);
  }
}

Given(P + 'the literal check harness has run', {timeout: 110000}, function () {
  if (observation === null) observation = harness();
});

Then(P + 'the observation of {string} includes:', function (name, text) {
  include(name, JSON.parse(text));
});
