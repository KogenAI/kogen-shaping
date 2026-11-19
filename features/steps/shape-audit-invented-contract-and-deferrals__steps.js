'use strict';
// Acceptance steps of shape-audit-invented-contract-and-deferrals: compile the shape tests with the
// Go harness through an overlay, run it once and compare the JSON observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestShapeAuditInventedContractAcceptance';
const TARGET = 'shape/shape_audit_invented_contract_and_deferrals_acceptance_test.go';
const P = 'shape-audit-invented-contract-and-deferrals: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function harness() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ksaid-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-audit-invented-contract-and-deferrals__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 400000, maxBuffer: 32 * 1024 * 1024});
  assert.equal(b.status, 0, `building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}:\n${b.stdout}${b.stderr}`);
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=250s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({KSAID_OUT: out}), encoding: 'utf8', timeout: 270000, maxBuffer: 32 * 1024 * 1024});
  const output = (r.stdout || '') + (r.stderr || '');
  const shown = `${TEST} exited ${r.status}:\n${output}`;
  assert.equal(r.status, 0, shown);
  assert.equal(output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length, 1, `${TEST} did not pass exactly once: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function include(o, name, want) {
  assert.ok(o && Object.hasOwn(o, name), `the harness observed no case "${name}":\n${JSON.stringify(o, null, 1)}`);
  const c = o[name];
  const summary = JSON.stringify(c, null, 1);
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(c, key), `case "${name}" has no ${key}:\n${summary}`);
    assert.deepEqual(c[key], value, `case "${name}": ${key} is ${JSON.stringify(c[key])}, want ${JSON.stringify(value)}\nobservation:\n${summary}`);
  }
}

if (require.main !== module) {
  const {Given, Then, BeforeAll} = require('@cucumber/cucumber');
  let cached = null;
  Given(P + 'the audit harness has run', {timeout: 700000}, function () {
    if (cached === null) cached = harness();
    this.ksaid = cached;
  });
  Then(P + 'the observation of {string} includes:', function (name, text) {
    include(this.ksaid, name, JSON.parse(text));
  });
}
