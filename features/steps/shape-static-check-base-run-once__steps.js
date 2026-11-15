'use strict';
// Acceptance of shape-static-check-base-run-once. The When step compiles the shape package's tests
// with the harness of shape-static-check-base-run-once__harness.js (through a Go overlay) and runs
// TestSBROObserve once; the Then steps judge what the run observed.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {When, Then} = require('@cucumber/cucumber');

const P = 'shape-static-check-base-run-once: ';
const TEST = 'TestSBROObserve';
let ran;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function runOnce() {
  if (ran) return ran;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-static-check-base-run-once-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'sbro_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-static-check-base-run-once__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/sbro_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  if (b.status !== 0) {
    ran = {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`};
    return ran;
  }
  const out = path.join(dir, 'out.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=240s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SBRO_OUT: out}), encoding: 'utf8', timeout: 270000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ')) {
    ran = {error: `${TEST}: shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`};
    return ran;
  }
  ran = {seen: JSON.parse(fs.readFileSync(out, 'utf8'))};
  return ran;
}

function seen() {
  const r = runOnce();
  if (r.error) throw new Error(r.error);
  return r.seen;
}

function shown(o) {
  return `exit ${o.exit}, error ${JSON.stringify(o.error)}, published ${o.published}, shared-command base runs ${o.baseRuns}, other base runs ${o.otherBaseRuns}, other shaped runs ${o.otherShapedRuns}\nstderr (tail):\n${String(o.stderr).slice(-4000)}`;
}

When(P + 'the project with the failing checks {string}, {string} and {string} is shaped with two refused submissions', {timeout: 300000}, function (a, b, c) {
  assert.deepEqual([a, b, c], ['stale', 'twin', 'other'], 'the harness configures the checks stale, twin and other');
  this.sbro = seen();
});

Then(P + 'the Intent is published after the acknowledgment', function () {
  assert.equal(this.sbro.published, true, 'the Intent was not published\n' + shown(this.sbro));
});

Then(P + 'the command shared by the checks "stale" and "twin" ran without the shaped files {int} time\\(s\\)', function (n) {
  assert.equal(this.sbro.baseRuns, n, `the shared command ran ${this.sbro.baseRuns} time(s) without the shaped files, want ${n}\n` + shown(this.sbro));
});

Then(P + 'the command of the check "other" ran without the shaped files {int} time\\(s\\)', function (n) {
  assert.equal(this.sbro.otherBaseRuns, n, `the command of other ran ${this.sbro.otherBaseRuns} time(s) without the shaped files, want ${n}\n` + shown(this.sbro));
});

Then(P + 'the command of the check "other" ran with the shaped files {int} times, a plain and a traced run in each of the two rounds', function (n) {
  assert.equal(this.sbro.otherShapedRuns, n, `the command of other ran ${this.sbro.otherShapedRuns} time(s) with the shaped files, want ${n}\n` + shown(this.sbro));
});

Then(P + 'the progress lines of the failing static checks are, in order:', function (text) {
  const want = text.split('\n').map(l => l.trim()).filter(Boolean);
  const got = String(this.sbro.stderr).split('\n').filter(l => /static check \S+ failed \(exit/.test(l));
  assert.equal(got.length, want.length, `${got.length} failing progress line(s), want ${want.length}: ${JSON.stringify(got)}\n` + shown(this.sbro));
  want.forEach((w, i) => assert.ok(got[i].endsWith(w), `progress line ${i + 1} is ${JSON.stringify(got[i])}, want ${JSON.stringify(w)}\n` + shown(this.sbro)));
});
