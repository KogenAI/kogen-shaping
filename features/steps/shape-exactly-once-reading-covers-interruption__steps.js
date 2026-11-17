'use strict';
// Acceptance of shape-exactly-once-reading-covers-interruption. The Given step compiles the shape
// package's tests with the harness of shape-exactly-once-reading-covers-interruption__harness.js
// (through a Go overlay), runs TestSkeAcceptance once per Cucumber process and reads the JSON the
// harness wrote to SKE_OUT. The Then steps judge that observation.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const TEST = 'TestSkeAcceptance';
const TARGET = 'shape/ske_acceptance_test.go';
const P = 'shape-exactly-once-reading-covers-interruption: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

// judge accepts a run that exited 0, passed the harness test exactly once and wrote an observation.
function judge(r) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  return JSON.parse(r.json);
}

let observed = null;
function harness() {
  if (observed !== null) return observed;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ske-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-exactly-once-reading-covers-interruption__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-run', '^' + TEST + '$', '-count=1', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SKE_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  observed = judge({status: r.status, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null, command: 'go ' + args.join(' ')});
  return observed;
}

Given(P + 'the decision check harness has run', {timeout: 115000}, function () { harness(); });

Then(P + 'the decision instructions contain {string}', function (text) {
  assert.ok(harness().instructions.includes(text), `the decision instructions do not contain ${JSON.stringify(text)}`);
});

Then(P + 'the decision instructions do not contain {string}', function (text) {
  assert.ok(!harness().instructions.toLowerCase().includes(text.toLowerCase()), `the decision instructions contain ${JSON.stringify(text)}`);
});

Then(P + 'the scripted contradiction of the weakened decision is refused naming {string} and {string}', function (weak, strong) {
  const lines = harness().weakened_lines;
  assert.equal(lines.length, 3, `refusal lines ${JSON.stringify(lines)}`);
  assert.ok(lines[1].includes('verdict: contradicts'), lines[1]);
  assert.ok(lines[1].includes('intent_value: ' + weak), lines[1]);
  assert.ok(lines[1].includes('request_value: ' + strong), lines[1]);
});

Then(P + 'the scripted decision that keeps the full guarantee is not refused', function () {
  const lines = harness().full_lines;
  assert.deepEqual(lines, [], `refusal lines ${JSON.stringify(lines)}`);
});
