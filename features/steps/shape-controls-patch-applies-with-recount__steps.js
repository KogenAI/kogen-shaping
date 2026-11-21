'use strict';
// Acceptance of shape-controls-patch-applies-with-recount. The Given step compiles the shape
// package's tests with the harness of shape-controls-patch-applies-with-recount__harness.js (through
// a Go overlay), runs TestShapeControlsPatchRecountAcceptance once per Cucumber process and reads
// the JSON the harness wrote to SCPAR_OUT. The Then step compares one case of it with the feature's
// value.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const TEST = 'TestShapeControlsPatchRecountAcceptance';
const TARGET = 'shape/shape_controls_patch_recount_acceptance_test.go';
const P = 'shape-controls-patch-applies-with-recount: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

function harness() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'scpar-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-controls-patch-applies-with-recount__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 400000, maxBuffer: 32 * 1024 * 1024});
  assert.equal(b.status, 0, `building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}:\n${b.stdout}${b.stderr}`);
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SCPAR_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const output = (r.stdout || '') + (r.stderr || '');
  const shown = `${TEST} (shape.test ${args.join(' ')}) exited ${r.status}${r.error ? ' ' + r.error.message : ''}:\n${output}`;
  assert.equal(r.status, 0, shown);
  const passes = output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines; it is skipped when KOGEN_CUCUMBER is unset): ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation file: ${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

let cached = null;
Given(P + 'the controls patch harness has run', {timeout: 660000}, function () {
  if (cached === null) cached = harness();
  this.scpar = cached;
});

Then(P + 'the refusal lines of {string} are:', function (name, text) {
  assert.ok(Object.hasOwn(this.scpar, name), `the harness observed no case "${name}": ${JSON.stringify(this.scpar)}`);
  assert.deepEqual(this.scpar[name], JSON.parse(text), `case "${name}": refusal lines are ${JSON.stringify(this.scpar[name], null, 1)}`);
});
