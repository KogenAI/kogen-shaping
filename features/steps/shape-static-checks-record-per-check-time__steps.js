'use strict';
// Acceptance of shape-static-checks-record-per-check-time. The When steps compile the shape
// package's tests with the harness of shape-static-checks-record-per-check-time__harness.js
// (through a Go overlay) and run one named test; the Then steps judge the observed events.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {When, Then} = require('@cucumber/cucumber');

const P = 'shape-static-checks-record-per-check-time: ';
let built;
const ran = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-static-check-time-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'sct_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-static-checks-record-per-check-time__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/sct_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  if (b.status !== 0) {
    built = {error: `go test -c -vet=off -overlay ./shape failed: exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`};
  } else {
    built = {dir, bin};
  }
  return built;
}

function observe(test) {
  if (ran[test]) return ran[test];
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, test + '.json');
  const args = ['-test.run', '^' + test + '$', '-test.count=1', '-test.v', '-test.timeout=240s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({SCT_OUT: out}), encoding: 'utf8', timeout: 270000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + test + ' ')) {
    throw new Error(`${test}: shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  ran[test] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return ran[test];
}

function shown(o) {
  return `exit ${o.exit}, error ${JSON.stringify(o.error)}, published ${o.published}, static events ${JSON.stringify(o.static)}\nstderr (tail):\n${String(o.stderr).slice(-3000)}`;
}

function event(o, i) {
  assert.ok(o.static.length > i, `static_checks event ${i + 1} is missing: ${o.static.length} event(s)\n` + shown(o));
  return o.static[i];
}

function check(o, i, name) {
  const c = (event(o, i).checks || []).find(x => x.name === name);
  assert.ok(c, `no check ${name} in static_checks event ${i + 1}\n` + shown(o));
  return c;
}

function isMs(v) {
  return Number.isInteger(v) && v >= 0;
}

When(P + 'the project with the checks {string}, {string} and {string} is shaped with one refused submission and an acknowledgment of the unchanged package', {timeout: 300000}, function (a, b, c) {
  assert.deepEqual([a, b, c], ['slow', 'stale', 'suite'], 'the harness configures the checks slow, stale and suite');
  this.sct = observe('TestSCTTiming');
});

When(P + 'the project with the failing check {string} is shaped with a refused submission, a repaired submission and an acknowledgment', {timeout: 300000}, function (a) {
  assert.equal(a, 'bad', 'the harness configures the check bad');
  this.sct = observe('TestSCTRefusal');
});

Then(P + 'the Intent is published', function () {
  assert.equal(this.sct.published, true, 'the Intent was not published\n' + shown(this.sct));
});

Then(P + 'the first static checks event has the checks {string}, {string} and {string} in order', function (a, b, c) {
  const names = (event(this.sct, 0).checks || []).map(x => x.name);
  assert.deepEqual(names, [a, b, c], 'check names ' + JSON.stringify(names) + '\n' + shown(this.sct));
});

Then(P + 'in the first static checks event the check {string} has wall_ms greater than 0 and base_wall_ms 0', function (name) {
  const c = check(this.sct, 0, name);
  assert.ok(isMs(c.wall_ms) && c.wall_ms > 0, `${name}: wall_ms ${JSON.stringify(c.wall_ms)}, want an integer greater than 0\n` + shown(this.sct));
  assert.equal(c.base_wall_ms, 0, `${name}: base_wall_ms ${JSON.stringify(c.base_wall_ms)}, want 0\n` + shown(this.sct));
});

Then(P + 'in the first static checks event the check {string} has wall_ms greater than 0 and base_wall_ms greater than 0', function (name) {
  const c = check(this.sct, 0, name);
  assert.ok(isMs(c.wall_ms) && c.wall_ms > 0, `${name}: wall_ms ${JSON.stringify(c.wall_ms)}, want an integer greater than 0\n` + shown(this.sct));
  assert.ok(isMs(c.base_wall_ms) && c.base_wall_ms > 0, `${name}: base_wall_ms ${JSON.stringify(c.base_wall_ms)}, want an integer greater than 0\n` + shown(this.sct));
});

Then(P + 'in the first static checks event the check {string} has wall_ms 0 and base_wall_ms 0', function (name) {
  const c = check(this.sct, 0, name);
  assert.equal(c.wall_ms, 0, `${name}: wall_ms ${JSON.stringify(c.wall_ms)}, want 0\n` + shown(this.sct));
  assert.equal(c.base_wall_ms, 0, `${name}: base_wall_ms ${JSON.stringify(c.base_wall_ms)}, want 0\n` + shown(this.sct));
});

Then(P + 'the first static checks event has copy_ms greater than 0', function () {
  const v = event(this.sct, 0).copy_ms;
  assert.ok(isMs(v) && v > 0, `copy_ms ${JSON.stringify(v)}, want an integer greater than 0\n` + shown(this.sct));
});

Then(P + 'the second static checks event is reused and repeats the copy_ms, wall_ms and base_wall_ms values of the first', function () {
  const first = event(this.sct, 0);
  const second = event(this.sct, 1);
  assert.equal(first.reused, false, 'the first event is reused\n' + shown(this.sct));
  assert.equal(second.reused, true, 'the second event is not reused\n' + shown(this.sct));
  assert.ok(isMs(first.copy_ms) && first.copy_ms > 0, `first copy_ms ${JSON.stringify(first.copy_ms)}\n` + shown(this.sct));
  assert.equal(second.copy_ms, first.copy_ms, `copy_ms ${second.copy_ms} after a reuse, want the original ${first.copy_ms}\n` + shown(this.sct));
  const pick = e => e.checks.map(c => [c.name, c.wall_ms, c.base_wall_ms]);
  assert.deepEqual(pick(second), pick(first), 'the reused event does not repeat the original timings\n' + shown(this.sct));
  assert.ok(first.checks.some(c => c.wall_ms > 0), 'the original run has no timed check\n' + shown(this.sct));
});

Then(P + 'the refusal of the static checks gate has the problem lines, in order:', function (text) {
  const want = text.split('\n').map(l => l.replace(/^ {6}/, '')).filter(l => l.trim() !== '');
  assert.ok(this.sct.refusals.length >= 1, 'no static_checks refusal was recorded\n' + shown(this.sct));
  assert.deepEqual(this.sct.refusals[0], want, 'the problem lines of the first refusal differ\n' + shown(this.sct));
});
