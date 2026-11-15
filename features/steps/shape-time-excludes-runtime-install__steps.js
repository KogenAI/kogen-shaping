'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');
const raw = require('node:assert/strict');
const root = path.resolve(__dirname, '../..');
const test = 'TestShapeRuntimeInstallObservation';
const context = `${test}: shape.test -test.run=^${test}$ -test.count=1`;
// Every assertion retains the named test and command, including downstream output checks.
const assert = Object.fromEntries(['equal','deepEqual','ok'].map(k => [k, (...args) => {
  const required = k === 'ok' ? 1 : 2;
  const message = args.length > required ? args.pop() : '';
  raw[k](...args, `${context}: ${message}`);
}]));
let dir, bin;
function cleanEnv(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_') || (/TOKEN|KEY|PASSWORD|SECRET/i.test(k) && !(k in extra))) delete e[k];
  return e;
}
function build() {
  if (bin) return;
  dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sti-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const src = path.join(dir, 'observe_test.go');
  fs.writeFileSync(src, fs.readFileSync(path.join(__dirname, 'shape-time-excludes-runtime-install__driver/observe.go'), 'utf8').replace('//go:build ignore\n\n', ''));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(root, 'shape/shape_runtime_install_observation_test.go')]: src}}));
  bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = cp.spawnSync('go', args, {cwd: root, env: cleanEnv({}), encoding: 'utf8', timeout: 100000, maxBuffer: 16<<20});
  assert.equal(r.status, 0, `${test}: go ${args.join(' ')} failed exit=${r.status}: ${r.stdout}${r.stderr}`);
}
function observe(spec) {
  build();
  const out = path.join(dir, 'observation.json');
  fs.rmSync(out, {force: true});
  const args = [`-test.run=^${test}$`, '-test.count=1', '-test.timeout=60s', '-test.v'];
  const r = cp.spawnSync(bin, args, {cwd: path.join(root, 'shape'), env: cleanEnv({STI_SPEC: JSON.stringify(spec), STI_OUTPUT: out}), encoding: 'utf8', timeout: 70000, maxBuffer: 16<<20});
  const diagnostic = `${test}: shape.test ${args.join(' ')} exit=${r.status}, error=${r.error || 'none'}`;
  assert.equal(r.status, 0, `${diagnostic}\n${r.stdout}${r.stderr}`);
  assert.ok(fs.existsSync(out), `${diagnostic}: named test absent, skipped or ran zero times; no observation\n${r.stdout}${r.stderr}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}
function account(o, expected) {
  assert.equal(o.accounts.length, 1, 'one shape_time event');
  const a = o.accounts[0];
  for (const [k,v] of Object.entries(expected)) assert.equal(a[k], Number(v), `${k}: ${JSON.stringify(a)}`);
  for (const k of ['elapsed_ms','shaper_ms','checks_ms','install_ms','gap_ms']) assert.ok(Number.isInteger(a[k]) && a[k] >= 0, `${k} nonnegative integer`);
  assert.equal(a.gap_ms, a.elapsed_ms-a.shaper_ms-a.checks_ms-a.install_ms, 'gap accounting identity');
}
function line(o, expected) { assert.deepEqual(o.time_lines, [expected], 'exact shape time progress line'); }
function result(o, exit, rounds) {
  assert.equal(o.exit, exit, `Shape exit: ${JSON.stringify(o)}`);
  assert.equal(o.rounds, rounds, 'main rounds run');
  if (exit === 0) assert.equal(o.error, '', 'no refusal');
}
function refusal(o, install) {
  assert.equal(o.class, 'shape'); assert.equal(o.reason, 'invalid_output');
  assert.equal(o.detail, 'the shaper used its 15-minute shaping budget (15 min elapsed, 1 requests) without a submission that passed; shaping stopped');
  const budget = o.lines.find(l => l.startsWith('shaping budget: '));
  const attempt = o.lines.find(l => l.startsWith('attempt 1: gate validation,'));
  assert.ok(budget && attempt, 'existing gate and timing lines retained');
  const original = 'shaping budget: 900 s (15 min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)';
  const timing = 'attempt 1: gate validation, model request 900 s, Kogen checks 0 s, 0 s remaining';
  if (install > 0) {
    for (const l of [budget, attempt]) assert.ok(l.includes(`install_ms=${install}`), 'nonzero install_ms on budget and attempt timing lines');
    assert.ok(budget.startsWith(original) && attempt.startsWith(timing), 'refusal wording preserved before install suffix');
  } else { assert.equal(budget, original); assert.equal(attempt, timing); }
}
module.exports = {observe, account, line, result, refusal};
if (!process.env.STI_OBSERVER_ONLY && require.main !== module) {
  const {When, Then} = require('@cucumber/cucumber');
  When('shape-time-excludes-runtime-install: Shape runs this scripted timing:', {timeout: 120000}, function(table) {
    const v = table.rowsHash();
    this.sti = observe({Install: +v.install_ms, Shaper: +v.shaper_ms, Checks: +v.checks_ms, Startup: +v.startup_ms, Warm: v.warm === 'true', Mode: v.mode});
  });
  Then('shape-time-excludes-runtime-install: Shape exits {int} after {int} main rounds', function(exit, rounds) { result(this.sti, exit, rounds); });
  Then('shape-time-excludes-runtime-install: the account is:', function(table) { account(this.sti, table.rowsHash()); });
  Then('shape-time-excludes-runtime-install: the progress line is:', function(text) { line(this.sti, text); });
  Then('shape-time-excludes-runtime-install: the budget refusal preserves its wording and reports install_ms {int}', function(ms) { refusal(this.sti, ms); });
}
