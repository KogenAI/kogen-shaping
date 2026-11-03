'use strict';
// Acceptance harness of k-shape-time: it adds k_shape_time_test.go.txt to the shape package
// through a go overlay, builds the test binary once, runs one scripted shaping run per scenario
// and judges its KST observation line.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeTimeAcceptance';
const ROOT = path.resolve(__dirname, '..', '..');
// Overlay paths are relative to ROOT, the directory go runs in.
const SOURCE = 'features/steps/k_shape_time_test.go.txt';
const TARGET = 'shape/k_shape_time_acceptance_test.go';
const MAIN = 'executor-shape';
let binary = null;

// The fixture's redactor masks the values of secret-named variables, so they are not passed on.
function env(extra) {
  const out = {...process.env, ...extra};
  for (const k of Object.keys(out)) {
    if (k.startsWith('KOGEN_ACCEPTANCE_') || (/TOKEN|KEY|PASSWORD|SECRET/i.test(k) && !(k in extra))) delete out[k];
  }
  return out;
}

function build() {
  if (binary) return binary;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kst-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[TARGET]: SOURCE}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {cwd: ROOT, env: env({PWD: ROOT}), encoding: 'utf8', timeout: 540000, maxBuffer: 16 << 20});
  if (r.status !== 0) {
    throw new Error(`${TEST}: go ${args.join(' ')} failed (exit ${r.status}, error ${r.error || 'none'}):\n${r.stdout}${r.stderr}`);
  }
  binary = bin;
  return bin;
}

function observe(vars) {
  const bin = build();
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=120s'];
  const r = spawnSync(bin, args, {cwd: path.join(ROOT, 'shape'), env: env({PWD: path.join(ROOT, 'shape'), ...vars}), encoding: 'utf8', timeout: 180000, maxBuffer: 64 << 20});
  const text = (r.stdout || '') + (r.stderr || '');
  const shown = Object.entries(vars).map(([k, v]) => `${k}=${v}`).join(' ');
  const detail = `${TEST} (${shown}): shape.test ${args.join(' ')}, built by go test -c -vet=off -overlay ./shape, test binary exit=${r.status} error=${r.error || 'none'}`;
  const line = text.split('\n').find((l) => l.startsWith('KST '));
  if (r.status !== 0 || !line || !text.includes('--- PASS: ' + TEST + ' ')) {
    throw new Error(`${detail}: the test did not run to a KST observation\n${text}`);
  }
  return {...JSON.parse(line.slice('KST '.length)), detail};
}

const has = (o, k) => Object.prototype.hasOwnProperty.call(o, k);
const isMs = (v) => Number.isInteger(v) && v >= 0;
const ACCOUNT_FIELDS = ['checks_ms', 'elapsed_ms', 'event', 'gap_ms', 'readiness_ms', 'shaper_model_ms', 'shaper_ms', 'shaper_tools_ms'];

// secs is a millisecond value as the time line prints it: seconds with one decimal, rounded
// half up, or unknown.
function secs(ms) {
  if (ms === null) return 'unknown';
  const tenths = Math.floor((ms + 50) / 100);
  return `${Math.floor(tenths / 10)}.${tenths % 10} s`;
}

function timeLine(a) {
  return `kogen: shape time ${secs(a.elapsed_ms)}: shaper ${secs(a.shaper_ms)} (model ${secs(a.shaper_model_ms)}, tools ${secs(a.shaper_tools_ms)}), ` +
    `Kogen checks ${secs(a.checks_ms)} (readiness ${secs(a.readiness_ms)}), gaps ${secs(a.gap_ms)}`;
}

function account(o) {
  assert.equal(o.accounts.length, 1, `${o.detail}: shape_time events in the shaping record: ${JSON.stringify(o.accounts)}`);
  return o.accounts[0];
}

function rounds(o, session) {
  return o.rounds.filter((r) => r.session === session);
}

const check = {
  ready(o) {
    assert.equal(o.exit, 0, `${o.detail}: exit code; stop text:\n${o.stop}`);
    assert.equal(o.stdout.length, 7, `${o.detail}: stdout lines ${JSON.stringify(o.stdout)}`);
    assert.equal(o.stdout[4], 'Scenarios: 2 (2 fail now, 0 pass now)', `${o.detail}: scenarios line`);
    assert.match(o.stdout[5], /^shape \S+\/\S+ .*wall_ms=\d+$/, `${o.detail}: usage line`);
    assert.equal(o.stdout[6], 'Next: kogen intent approve is-even', `${o.detail}: last stdout line`);
  },
  recorded(o) {
    const a = account(o);
    assert.deepEqual(Object.keys(a).sort(), ACCOUNT_FIELDS, `${o.detail}: shape_time fields of ${JSON.stringify(a)}`);
    assert.equal(o.after_account, 'session_end', `${o.detail}: the event right after shape_time`);
    assert.equal(o.session_end.length, 1, `${o.detail}: session_end events`);
    assert.equal(a.elapsed_ms, o.session_end[0].elapsed_ms, `${o.detail}: shape_time elapsed_ms against session_end elapsed_ms`);
    for (const k of ['elapsed_ms', 'shaper_ms', 'checks_ms', 'readiness_ms', 'gap_ms']) {
      assert.ok(isMs(a[k]), `${o.detail}: shape_time ${k} = ${JSON.stringify(a[k])} is not a whole number of milliseconds`);
    }
  },
  mainRounds(o, rows) {
    const main = rounds(o, MAIN);
    assert.equal(main.length, rows.length, `${o.detail}: executor_round events with session ${MAIN} among ${JSON.stringify(o.rounds)}`);
    rows.forEach((row, i) => {
      const r = main[i];
      const where = `${o.detail}: executor_round ${i + 1} of session ${MAIN} ${JSON.stringify(r)}`;
      assert.equal(r.round, Number(row.round), `${where}: round`);
      assert.ok(isMs(r.started_ms) && isMs(r.wall_ms), `${where}: started_ms and wall_ms must be whole milliseconds`);
      assert.ok(r.wall_ms >= Number(row.min_wall_ms), `${where}: wall_ms below ${row.min_wall_ms}`);
      assert.ok(has(r, 'model_ms'), `${where}: no model_ms`);
      assert.equal(r.model_ms, row.model_ms === 'unknown' ? null : Number(row.model_ms), `${where}: model_ms`);
      if (i > 0) {
        assert.ok(r.started_ms >= main[i - 1].started_ms + main[i - 1].wall_ms, `${where}: starts before the previous round ended`);
      }
    });
  },
  checkRounds(o, session, min) {
    const list = rounds(o, session);
    assert.ok(list.length >= 1, `${o.detail}: no executor_round event with session ${session} among ${JSON.stringify(o.rounds)}`);
    for (const r of list) {
      const where = `${o.detail}: executor_round of session ${session} ${JSON.stringify(r)}`;
      assert.ok(isMs(r.started_ms) && isMs(r.wall_ms) && r.wall_ms >= min, `${where}: wall_ms must be at least ${min}`);
      assert.ok(has(r, 'model_ms') && r.model_ms === null, `${where}: model_ms must be null without a result line`);
    }
  },
  model(o, ms) {
    const a = account(o);
    assert.equal(a.shaper_model_ms, ms, `${o.detail}: shape_time shaper_model_ms of ${JSON.stringify(a)}`);
  },
  unknown(o) {
    const a = account(o);
    assert.ok(has(a, 'shaper_model_ms') && a.shaper_model_ms === null, `${o.detail}: shape_time shaper_model_ms must be null: ${JSON.stringify(a)}`);
    assert.ok(has(a, 'shaper_tools_ms') && a.shaper_tools_ms === null, `${o.detail}: shape_time shaper_tools_ms must be null: ${JSON.stringify(a)}`);
  },
  shaperRounds(o) {
    const a = account(o);
    const main = rounds(o, MAIN);
    const total = main.reduce((s, r) => s + r.wall_ms, 0);
    assert.ok(main.length > 0, `${o.detail}: no executor_round events with session ${MAIN}`);
    assert.equal(a.shaper_ms, total, `${o.detail}: shape_time shaper_ms against the total wall_ms of the ${MAIN} rounds ${JSON.stringify(main)}`);
  },
  split(o) {
    const a = account(o);
    assert.ok(isMs(a.shaper_model_ms) && isMs(a.shaper_tools_ms), `${o.detail}: shaper model and tools must be known: ${JSON.stringify(a)}`);
    assert.equal(a.shaper_model_ms + a.shaper_tools_ms, a.shaper_ms, `${o.detail}: shaper_model_ms + shaper_tools_ms against shaper_ms in ${JSON.stringify(a)}`);
  },
  checks(o, minChecks, minReadiness) {
    const a = account(o);
    assert.deepEqual(o.readiness_phase, [a.readiness_ms], `${o.detail}: shape_time readiness_ms against the readiness phase_duration ${JSON.stringify(o.readiness_phase)}`);
    assert.ok(a.readiness_ms >= minReadiness, `${o.detail}: readiness_ms ${a.readiness_ms} below ${minReadiness}`);
    assert.ok(a.checks_ms >= minChecks && a.checks_ms >= a.readiness_ms, `${o.detail}: checks_ms ${a.checks_ms} below ${minChecks} or below readiness_ms ${a.readiness_ms}`);
  },
  checksInclude(o, session) {
    const a = account(o);
    const total = rounds(o, session).reduce((s, r) => s + r.wall_ms, 0);
    assert.ok(a.checks_ms >= a.readiness_ms + total, `${o.detail}: checks_ms ${a.checks_ms} must cover readiness_ms ${a.readiness_ms} and the ${session} rounds (${total} ms)`);
  },
  gaps(o) {
    const a = account(o);
    assert.equal(a.gap_ms, a.elapsed_ms - a.shaper_ms - a.checks_ms, `${o.detail}: gap_ms against elapsed_ms - shaper_ms - checks_ms in ${JSON.stringify(a)}`);
    assert.ok(a.gap_ms >= 0, `${o.detail}: negative gap_ms in ${JSON.stringify(a)}`);
  },
  line(o) {
    const a = account(o);
    const lines = o.time_lines;
    assert.deepEqual(lines, [timeLine(a)], `${o.detail}: the stderr lines starting "kogen: shape time"`);
    assert.equal(o.stderr_last, timeLine(a), `${o.detail}: the last stderr line`);
    assert.ok(!o.stdout.some((l) => l.startsWith('kogen: shape time')), `${o.detail}: the time line is on stdout`);
  },
  brief(o, paragraph) {
    const n = o.brief.split(paragraph).length - 1;
    assert.equal(n, 1, `${o.detail}: occurrences of the commands paragraph in the claude-code shaper's task (${o.brief.length} bytes)`);
  },
};

// A known-good observation of the external scenario and its variants.
function goodExternal() {
  const main = [
    {event: 'executor_round', session: MAIN, round: 1, started_ms: 100, wall_ms: 1250, model_ms: 300},
    {event: 'executor_round', session: 'executor-contract-review-1', round: 1, started_ms: 3500, wall_ms: 520, model_ms: null},
    {event: 'executor_round', session: MAIN, round: 2, started_ms: 4100, wall_ms: 650, model_ms: 200},
  ];
  const a = {event: 'shape_time', elapsed_ms: 6000, shaper_ms: 1900, shaper_model_ms: 500, shaper_tools_ms: 1400, checks_ms: 3000, readiness_ms: 2100, gap_ms: 1100};
  return {detail: 'good', mode: 'external', exit: 0, stop: '', stdout: ['Intent: x', 'Acceptance test: x', 'Step files: x', 'Model: x', 'Scenarios: 2 (2 fail now, 0 pass now)',
    'shape claude-opus-5-5/xhigh claude-code 3 round(s) wall_ms=5900', 'Next: kogen intent approve is-even'],
  accounts: [a], after_account: 'session_end', session_end: [{event: 'session_end', elapsed_ms: 6000}], readiness_phase: [2100], rounds: main,
  time_lines: [timeLine(a)], stderr_last: timeLine(a), brief: 'before\nPARAGRAPH\nafter'};
}

function judgeExternal(o) {
  check.ready(o); check.recorded(o);
  check.mainRounds(o, [{round: '1', model_ms: '300', min_wall_ms: '1200'}, {round: '2', model_ms: '200', min_wall_ms: '600'}]);
  check.checkRounds(o, 'executor-contract-review-1', 500);
  check.model(o, 500); check.shaperRounds(o); check.split(o);
  check.checks(o, 1000, 1000); check.checksInclude(o, 'executor-contract-review-1'); check.gaps(o); check.line(o);
  check.brief(o, 'PARAGRAPH');
}

// Observer controls: good runs the live harness and checks what this Intent does not change,
// then accepts a known-good observation; bad must reject every known-bad observation.
function controls(mode) {
  const accepts = (fn) => { try { fn(); return true; } catch (e) { return false; } };
  if (mode === 'good') {
    const live = observe({KST_MODE: 'external', KST_ROUNDS: '50:10,50:10', KST_CONTRACT_MS: '0', KST_ACCEPT_SECONDS: '0'});
    check.ready(live);
    const ids = live.rounds.map((r) => r.session_id);
    assert.equal(ids.filter((id) => id === 'shape-session-1').length, 2, 'live main rounds');
    assert.ok(live.brief.includes('read-only external session'), 'live brief');
    const native = observe({KST_MODE: 'native', KST_ACCEPT_SECONDS: '0'});
    check.ready(native);
    judgeExternal(goodExternal());
    const unknown = goodExternal();
    unknown.rounds[2].model_ms = null;
    Object.assign(unknown.accounts[0], {shaper_model_ms: null, shaper_tools_ms: null});
    unknown.time_lines = [timeLine(unknown.accounts[0])];
    unknown.stderr_last = unknown.time_lines[0];
    check.unknown(unknown); check.shaperRounds(unknown); check.line(unknown);
    check.mainRounds(unknown, [{round: '1', model_ms: '300', min_wall_ms: '1200'}, {round: '2', model_ms: 'unknown', min_wall_ms: '600'}]);
    return true;
  }
  const variant = (fn) => { const o = goodExternal(); fn(o); return o; };
  const relined = (o) => { o.time_lines = [timeLine(o.accounts[0])]; o.stderr_last = o.time_lines[0]; };
  const bad = [
    variant((o) => { o.accounts = []; o.after_account = ''; o.time_lines = []; o.stderr_last = 'kogen: checking readiness (3 s)'; }),
    variant((o) => { o.rounds.forEach((r) => { delete r.session; delete r.model_ms; }); }),
    variant((o) => { o.after_account = 'phase_duration'; }),
    variant((o) => { o.accounts[0].elapsed_ms = 5990; }),
    variant((o) => { Object.assign(o.accounts[0], {shaper_model_ms: 1900, shaper_tools_ms: 0}); relined(o); }),
    variant((o) => { o.rounds[0].model_ms = 1250; }),
    variant((o) => { Object.assign(o.accounts[0], {shaper_ms: 2420, shaper_tools_ms: 1920}); relined(o); }),
    variant((o) => { Object.assign(o.accounts[0], {checks_ms: 0, gap_ms: 4100}); relined(o); }),
    variant((o) => { Object.assign(o.accounts[0], {checks_ms: 2200, gap_ms: 1900}); relined(o); }),
    variant((o) => { o.accounts[0].gap_ms = 0; relined(o); }),
    variant((o) => { o.time_lines = [o.time_lines[0].replace('shaper 1.9 s', 'shaper 1.8 s')]; o.stderr_last = o.time_lines[0]; }),
    variant((o) => { o.stderr_last = 'kogen: checking readiness (3 s)'; }),
    variant((o) => { o.brief = 'before\nafter'; }),
    variant((o) => { o.brief = 'PARAGRAPH\nPARAGRAPH'; }),
    variant((o) => { o.stdout[6] = 'Next: kogen intent shape is-even'; }),
  ];
  const unknownBad = goodExternal();
  unknownBad.rounds[2].model_ms = null;
  bad.push(unknownBad);
  const halfUp = {elapsed_ms: 1250, shaper_ms: 0, shaper_model_ms: 0, shaper_tools_ms: 0, checks_ms: 0, readiness_ms: 0, gap_ms: 1250};
  if (!timeLine(halfUp).startsWith('kogen: shape time 1.3 s: ')) return true;
  return bad.some((o) => accepts(() => judgeExternal(o))) || accepts(() => check.unknown(unknownBad));
}

if (require.main === module) {
  let ok;
  try { ok = controls(process.argv[2]); } catch (e) { ok = false; }
  console.log(ok ? 'accepted' : 'rejected');
  process.exitCode = ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('a claude-code shape runs rounds {string} with contract reviews of {int} ms and acceptance of {int} second(s)', {timeout: 720000}, function (spec, contract, seconds) {
    this.kst = observe({KST_MODE: 'external', KST_ROUNDS: spec, KST_CONTRACT_MS: String(contract), KST_ACCEPT_SECONDS: String(seconds)});
  });
  When('a native shape runs with acceptance of {int} second(s)', {timeout: 720000}, function (seconds) {
    this.kst = observe({KST_MODE: 'native', KST_ACCEPT_SECONDS: String(seconds)});
  });
  Then('the shape is ready with its usual output', function () { check.ready(this.kst); });
  Then('the shaping record ends with one shape time account before the session end', function () { check.recorded(this.kst); });
  Then('the main shaper rounds are recorded as:', function (table) { check.mainRounds(this.kst, table.hashes()); });
  Then('the rounds of session {string} are recorded with at least {int} ms and unknown model time', function (session, min) { check.checkRounds(this.kst, session, min); });
  Then('the shaper model time is {int} ms', function (ms) { check.model(this.kst, ms); });
  Then('the shaper model and tools times are unknown', function () { check.unknown(this.kst); });
  Then('the shaper time is the total of the main shaper rounds', function () { check.shaperRounds(this.kst); });
  Then('the shaper time is the shaper model time plus the shaper tools time', function () { check.split(this.kst); });
  Then('Kogen checks took at least {int} ms, of which readiness took at least {int} ms', function (c, r) { check.checks(this.kst, c, r); });
  Then('Kogen checks cover readiness and the rounds of session {string}', function (session) { check.checksInclude(this.kst, session); });
  Then('the gaps are the elapsed time minus the shaper and check times', function () { check.gaps(this.kst); });
  Then('the last stderr line states the shape time account', function () { check.line(this.kst); });
  Then('the claude-code shaper\'s task contains this paragraph exactly once:', function (text) { check.brief(this.kst, text); });
}
