'use strict';
// Acceptance of shape-escalate-repeated-guarantee-conflict. The When step compiles the shape
// package's tests with the harness of shape-escalate-repeated-guarantee-conflict__harness.js
// (through a Go overlay), runs TestKSegAcceptance for one case with a scripted shaper and a
// scripted decision-check model, and reads what the run showed. The Then steps judge that.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {When, Then} = require('@cucumber/cucumber');

const TEST = 'TestKSegAcceptance';
let built;
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-escalate-guarantee-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'kseg_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-escalate-repeated-guarantee-conflict__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/kseg_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  built = r.status === 0 ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KSEG_CASE: name, KSEG_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KSEG_CASE=${name}): shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

function shown(o) {
  return `case ${o.case}: exit ${o.exit}, err ${o.err}; published=${o.published}; ${o.shaper} shaper request(s), ${o.decision} decision check request(s)\nstdout:\n${o.stdout}\nstderr (tail):\n${String(o.stderr).slice(-1500)}\ndecision_check events: ${JSON.stringify(o.events)}`;
}

function optionLines(o) {
  return o.stdout.split('\n').filter(l => /^ {3}[a-z]\) /.test(l));
}

When('shape-escalate-repeated-guarantee-conflict: the {string} case is shaped', {timeout: 540000}, function (name) { this.kseg = shapeCase(name); });

Then('shape-escalate-repeated-guarantee-conflict: shaping ends with questions and exit {int}', function (code) {
  const o = this.kseg;
  assert.equal(o.exit, code, `exit ${o.exit}, want ${code}\n${shown(o)}`);
  assert.ok(o.stdout.includes('Questions for is-even:'), `no questions were printed\n${shown(o)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: exactly {int} decision check requests are made', function (n) {
  assert.equal(this.kseg.decision, n, `${this.kseg.decision} decision check requests, want ${n}\n${shown(this.kseg)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: exactly {int} shaper requests are made', function (n) {
  assert.equal(this.kseg.shaper, n, `${this.kseg.shaper} shaper requests (check calls not counted), want ${n}\n${shown(this.kseg)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: the Intent is not published', function () {
  assert.equal(this.kseg.published, false, `the Intent was published\n${shown(this.kseg)}`);
  assert.equal(this.kseg.intent_exists, false, `intent.md exists in the project\n${shown(this.kseg)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: the output offers an option that retries the unconfirmed delivery', function () {
  const o = this.kseg;
  assert.ok(optionLines(o).some(l => /retr(y|ies|ied)/i.test(l)), `no option line mentions retrying\n${shown(o)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: the output offers an option that leaves the unconfirmed delivery unresent', function () {
  const o = this.kseg;
  assert.ok(optionLines(o).some(l => /unresent|not (be )?resent|never resent/i.test(l)), `no option line mentions leaving the delivery unresent\n${shown(o)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: no option is marked recommended', function () {
  const o = this.kseg;
  assert.ok(optionLines(o).length >= 2, `fewer than two options were printed\n${shown(o)}`);
  assert.ok(!o.stdout.includes('(recommended)'), `an option is marked recommended\n${shown(o)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: the output names the guarantee {string}', function (text) {
  assert.ok(this.kseg.stdout.includes(text), `the output does not name ${JSON.stringify(text)}\n${shown(this.kseg)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: the Intent is published after {int} decision check requests', function (d) {
  const o = this.kseg;
  assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
  assert.equal(o.decision, d, `${o.decision} decision check requests, want ${d}\n${shown(o)}`);
});
Then('shape-escalate-repeated-guarantee-conflict: the output asks no question', function () {
  assert.ok(!this.kseg.stdout.includes('Questions for'), `the output asks a question\n${shown(this.kseg)}`);
});
