'use strict';
// Acceptance of shape-carried-decisions-problems-show-lines. The When step compiles the shape
// package's tests with the harness of shape-carried-decisions-problems-show-lines__harness.js
// (through a Go overlay), runs TestKShapeCarriedDecisionsLinesAcceptance and reads the refusal
// text of the carried decisions gate. The Then step judges its problem lines.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeCarriedDecisionsLinesAcceptance';
const HEADLINE = 'Recorded decisions: ';

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function observe() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-carried-lines-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'kcdp_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-carried-decisions-problems-show-lines__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/kcdp_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  if (b.status !== 0) throw new Error(`${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`);
  const out = path.join(dir, 'out.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({KCDP_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST}: shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

// problemLines are the lines of the refusal after its headline.
function problemLines(o) {
  assert.ok(o.refused, `the carried decisions gate did not refuse the draft; feedback: ${JSON.stringify(o.feedback)}`);
  const lines = o.feedback.split('\n');
  assert.ok(lines[0].startsWith(HEADLINE), `the refusal is not the carried decisions refusal (first line ${JSON.stringify(lines[0])})`);
  return lines.slice(1);
}

const {When, Then} = require('@cucumber/cucumber');
const P = 'shape-carried-decisions-problems-show-lines: ';
When(P + 'the gate checks the draft with a superseded and a changed decision', {timeout: 540000}, function () { this.kcdp = observe(); });
Then(P + 'the refusal problem lines are exactly:', function (text) {
  assert.deepEqual(problemLines(this.kcdp), text.split('\n'), 'the problem lines of the carried decisions refusal differ');
});
