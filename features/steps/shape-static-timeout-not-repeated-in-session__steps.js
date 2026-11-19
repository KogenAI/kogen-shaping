'use strict';
// Acceptance of shape-static-timeout-not-repeated-in-session. The When step compiles the shape package's
// tests with the Go harness shape-static-timeout-not-repeated-in-session__driver/harness_test.go.txt
// (through a Go overlay), runs one named harness test and reads the JSON observation it wrote to
// STM_OUT. The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {When, Then} = require('@cucumber/cucumber');

const P = 'shape-static-timeout-not-repeated-in-session: ';
const HARNESS = path.resolve('features/steps/shape-static-timeout-not-repeated-in-session__driver/harness_test.go.txt');
const TESTS = {
  'not-repeated': 'TestSTMNotRepeated',
  'only-timeouts': 'TestSTMOnlyTimeouts',
  'keeps-failed': 'TestSTMKeepsFailed',
  'lock-change': 'TestSTMLockChange',
  'new-session': 'TestSTMNewSession',
};

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let built;
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stm-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/stm_acceptance_test.go')]: HARNESS}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env({}), encoding: 'utf8', timeout: 280000});
  if (b.status !== 0) throw new Error(`go test -c -vet=off -overlay ./shape failed: exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`);
  built = {dir, bin};
  return built;
}

const ran = {};
function observe(key) {
  const test = TESTS[key];
  assert.ok(test, `unknown harness scenario ${key}`);
  if (ran[key]) return ran[key];
  const b = build();
  const out = path.join(b.dir, test + '.json');
  const args = ['-test.run', '^' + test + '$', '-test.count=1', '-test.v', '-test.timeout=100s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({STM_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + test + ' ')) {
    throw new Error(`${test}: shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  ran[key] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return ran[key];
}

function shown(o) {
  return `published ${JSON.stringify(o.published)}, calls ${JSON.stringify(o.calls)}, static events ${JSON.stringify(o.sessions)}\nstderr (tail):\n${String(o.stderr).slice(-3000)}`;
}

function check(o, session, n, name) {
  const events = o.sessions[session - 1];
  assert.ok(events && events.length >= n, `static_checks event ${n} of session ${session} is missing\n` + shown(o));
  const c = (events[n - 1].checks || []).find(x => x.name === name);
  assert.ok(c, `no check ${name} in static_checks event ${n} of session ${session}\n` + shown(o));
  return c;
}

When(P + 'the project for {word} is shaped', {timeout: 300000}, function (key) {
  this.stm = observe(key);
});

Then(P + 'the Intent is published in session {int}', function (s) {
  assert.equal(this.stm.published[s - 1], true, `session ${s} did not publish\n` + shown(this.stm));
});

Then(P + 'after session {int} the check {string} was started {int} time(s) in total', function (s, name, n) {
  const calls = this.stm.calls[s - 1] || {};
  assert.equal(calls[name] || 0, n, `check ${name} was started ${calls[name] || 0} time(s) after session ${s}, want ${n}\n` + shown(this.stm));
});

Then(P + 'in session {int} the static checks event {int} lists the check {string} as {string} with wall_ms 0', function (s, n, name, status) {
  const c = check(this.stm, s, n, name);
  assert.equal(c.status, status, `${name}: status ${c.status}\n` + shown(this.stm));
  assert.equal(c.wall_ms, 0, `${name}: wall_ms ${JSON.stringify(c.wall_ms)}, want 0\n` + shown(this.stm));
});

Then(P + 'in session {int} the static checks event {int} lists the check {string} as {string} with wall_ms greater than 0', function (s, n, name, status) {
  const c = check(this.stm, s, n, name);
  assert.equal(c.status, status, `${name}: status ${c.status}\n` + shown(this.stm));
  assert.ok(Number.isInteger(c.wall_ms) && c.wall_ms > 0, `${name}: wall_ms ${JSON.stringify(c.wall_ms)}, want an integer greater than 0\n` + shown(this.stm));
});

Then(P + 'in session {int} the static checks event {int} gives the check {string} the reason {string}', function (s, n, name, reason) {
  const c = check(this.stm, s, n, name);
  assert.equal(c.reason, reason, `${name}: reason ${JSON.stringify(c.reason)}\n` + shown(this.stm));
});

Then(P + 'in session {int} the static checks event {int} gives the check {string} a reason without {string}', function (s, n, name, text) {
  const c = check(this.stm, s, n, name);
  assert.ok(!String(c.reason || '').includes(text), `${name}: reason ${JSON.stringify(c.reason)} contains ${text}\n` + shown(this.stm));
});

Then(P + 'the progress shows the line {string}', function (line) {
  assert.ok(String(this.stm.stderr).split('\n').some(l => l.includes(line)), `the progress has no line containing ${line}\n` + shown(this.stm));
});

Then(P + 'the static-checks record lists the check {string} as {string} with wall_ms 0', function (name, status) {
  const rec = this.stm.record;
  assert.ok(rec && Array.isArray(rec.checks), 'no static-checks record\n' + shown(this.stm));
  const c = rec.checks.find(x => x.name === name);
  assert.ok(c, `no check ${name} in the static-checks record\n` + shown(this.stm));
  assert.equal(c.status, status, `record: ${name} status ${c.status}\n` + shown(this.stm));
  assert.equal(c.wall_ms, 0, `record: ${name} wall_ms ${JSON.stringify(c.wall_ms)}, want 0\n` + shown(this.stm));
});
