'use strict';
// Acceptance steps of shape-repair-before-terminal-submit: they compile the Go harness of
// shape-repair-before-terminal-submit__harness.js into the shape package's tests through a Go
// overlay, build the test binary once per process, run one scripted native Shape journey per
// scenario and judge its SRB observation line. Run directly with good or bad, the file is the
// observer control of these judgements.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestSRBAcceptance';
const ROOT = path.resolve(__dirname, '..', '..');
// The overlay path is relative to ROOT, the directory go runs in.
const TARGET = 'shape/srb_acceptance_test.go';
let binary = null;

function env(extra) {
  const out = {...process.env, ...extra};
  for (const k of Object.keys(out)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete out[k];
  return out;
}

function build() {
  if (binary) return binary;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'srb-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'srb_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-repair-before-terminal-submit__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[TARGET]: source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {cwd: ROOT, env: env({PWD: ROOT}), encoding: 'utf8', timeout: 540000, maxBuffer: 16 << 20});
  if (r.status !== 0) {
    throw new Error(TEST + ': go ' + args.join(' ') + ' failed (exit ' + r.status + ', error ' + (r.error || 'none') + '):\n' + r.stdout + r.stderr);
  }
  binary = bin;
  return bin;
}

function observe(script, wire) {
  const bin = build();
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=150s'];
  const cwd = path.join(ROOT, 'shape');
  const r = spawnSync(bin, args, {cwd, env: env({PWD: cwd, SRB_WIRE: wire, SRB_SCRIPT: script}), encoding: 'utf8', timeout: 200000, maxBuffer: 32 << 20});
  const text = (r.stdout || '') + (r.stderr || '');
  const detail = TEST + ' (SRB_SCRIPT=' + script + ' SRB_WIRE=' + wire + '): shape.test ' + args.join(' ') +
    ', built by go test -c -vet=off -overlay ./shape, test binary exit=' + r.status + ' error=' + (r.error || 'none');
  const line = text.split('\n').find((l) => l.startsWith('SRB '));
  if (r.status !== 0 || !line || !text.includes('--- PASS: ' + TEST + ' ')) {
    throw new Error(detail + ': the journey did not run to an SRB observation\n' + text);
  }
  return {...JSON.parse(line.slice('SRB '.length)), detail};
}

function numbers(list) {
  return list === 'none' ? [] : list.split(',').map((s) => Number(s.trim()));
}

function why(o) {
  return o.detail + '; stop text:\n' + o.stop;
}

const check = {
  exit(o, code) { assert.equal(o.exit, code, why(o) + '\nexit code'); },
  requests(o, n) { assert.equal(o.requests, n, why(o) + '\nshaper requests'); },
  forced(o, list) { assert.deepEqual(o.forced, numbers(list), why(o) + '\nrequests that required submit'); },
  recovery(o, list) {
    assert.deepEqual(o.recovery, numbers(list), why(o) + '\nrequests that carry a new recovery message');
    assert.deepEqual(o.recovery_events.map((e) => e.request), numbers(list), why(o) + '\nrequests of the recovery_turn events');
  },
  events(o, rows) {
    const keys = ['attempt', 'cause', 'gate', 'owner', 'request'];
    const got = o.recovery_events.map((e) => {
      const has = Object.prototype.hasOwnProperty.call(e, 'gate');
      assert.deepEqual(Object.keys(e).sort(), has ? keys : keys.filter((k) => k !== 'gate'), why(o) + '\nrecovery_turn fields of ' + JSON.stringify(e));
      assert.equal(e.owner, 'shape', why(o) + '\nrecovery_turn owner');
      return {request: String(e.request), attempt: String(e.attempt), cause: e.cause, gate: has ? e.gate : '(absent)'};
    });
    assert.deepEqual(got, rows, why(o) + '\nrecovery_turn events');
  },
  message(o, n, text) { assert.equal((o.messages || {})[String(n)], text, why(o) + '\nrecovery message of request ' + n); },
  feedback(o, id, text) { assert.equal((o.outputs || {})[id], text, why(o) + '\nfeedback to call ' + id); },
  contains(o, id, text) {
    const out = (o.outputs || {})[id];
    assert.ok(typeof out === 'string' && out.includes(text), why(o) + '\nfeedback to call ' + id + ' lacks ' + JSON.stringify(text) + ':\n' + out);
  },
  lacks(o, id, text) {
    const out = (o.outputs || {})[id];
    assert.ok(typeof out === 'string' && !out.includes(text), why(o) + '\nfeedback to call ' + id + ' is missing or has ' + JSON.stringify(text) + ':\n' + out);
  },
  ends(o, id, text) {
    const out = (o.outputs || {})[id];
    assert.ok(typeof out === 'string' && out.endsWith(text), why(o) + '\nfeedback to call ' + id + ' does not end with ' + JSON.stringify(text) + ':\n' + out);
  },
  stop(o, text) { assert.equal(o.stop, text === '' ? '' : text + '\n', why(o) + '\nstop text'); },
  published(o) { assert.ok(o.published === true, why(o) + '\nthe expected package was not published; differing files: ' + JSON.stringify(o.unpublished)); },
  printed(o, text) { assert.ok(o.printed.includes(text), why(o) + '\noutput lacks ' + JSON.stringify(text) + ':\n' + o.printed); },
};

const REFUSAL = 'Recovery before submitting again: submission attempt 1 was refused at gate intent and nothing was published. ' +
  'Your accumulated proposals are kept, as the cumulative proposal manifest lists. This inference does not require submit: ' +
  'investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, ' +
  'then call submit with the repaired package. An unchanged package is refused again. The refusal:\nacceptance.feature does not exist';
const UNCHANGED = '- List active accounts | fresh-copy output: missing accounts database | why: the database is prepared only in the shaping checkout | confirmed: the fresh copy has no prepared database\n' +
  'Unchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.';
const QUESTION_STOP = 'shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 2) was refused at gate intent\n' +
  '  the shaper ended without calling submit\n  last refusal (attempt 2, gate intent):\n  acceptance.feature does not exist\n' +
  '  earliest unresolved refusal (attempt 1, gate question_check):\n' +
  '  - question 1 (technical): Which database library should store the accounts? | why: a technical question | decide it from what the repository and the request establish\n' +
  '  last tool-free answer: I cannot finish the package.';

// Observer controls. good runs the live harness on a journey this Intent does not change and
// accepts known-good observations of the new behaviour; bad must reject every known-bad
// observation (the old forced inference, a missing recovery, a missing unchanged label or
// earliest cause).
function controls(mode) {
  const accepts = (fn) => { try { fn(); return true; } catch (e) { return false; } };
  const good = {detail: 'good', exit: 0, requests: 6, forced: [], recovery: [3], messages: {3: REFUSAL}, stop: '', published: true,
    recovery_events: [{owner: 'shape', request: 3, attempt: 1, cause: 'refused', gate: 'intent'}],
    outputs: {c2: 'Readiness observations\n' + UNCHANGED, c1: 'invalid submission:\nacceptance.feature does not exist'}, printed: ''};
  const rows = [{request: '3', attempt: '1', cause: 'refused', gate: 'intent'}];
  if (mode === 'good') {
    const live = observe('valid-first', 'responses');
    check.exit(live, 0); check.requests(live, 1); check.forced(live, 'none'); check.recovery(live, 'none'); check.stop(live, ''); check.published(live);
    check.exit(good, 0); check.requests(good, 6); check.forced(good, 'none'); check.recovery(good, '3'); check.events(good, rows);
    check.message(good, 3, REFUSAL); check.ends(good, 'c2', UNCHANGED); check.lacks(good, 'c1', 'Unchanged:'); check.contains(good, 'c2', 'Readiness observations');
    check.stop({detail: 'good stop', stop: QUESTION_STOP + '\n'}, QUESTION_STOP);
    check.published(good);
    return true;
  }
  const old = {detail: 'old', exit: 0, requests: 6, forced: [3], recovery: [], messages: {}, recovery_events: [], stop: '', published: false,
    outputs: {c2: UNCHANGED.split('\nUnchanged:')[0], c1: 'invalid submission:\nacceptance.feature does not exist\nUnchanged: x'}, printed: ''};
  const oldStop = QUESTION_STOP.split('\n').filter((l) => !l.includes('earliest unresolved') && !l.includes('question 1 (technical)')).join('\n');
  const forcedRecovery = {...good, forced: [3]};
  const silentRecovery = {...good, messages: {}, recovery: []};
  const bad = [
    () => check.forced(old, 'none'),
    () => check.recovery(old, '3'),
    () => check.events(old, rows),
    () => check.message(old, 3, REFUSAL),
    () => check.ends(old, 'c2', UNCHANGED),
    () => check.lacks(old, 'c1', 'Unchanged:'),
    () => check.stop({detail: 'old stop', stop: oldStop + '\n'}, QUESTION_STOP),
    () => check.published(old),
    () => check.forced(forcedRecovery, 'none'),
    () => check.recovery(silentRecovery, '3'),
  ];
  return bad.some(accepts);
}

if (require.main === module) {
  let ok;
  try { ok = controls(process.argv[2]); } catch (e) { ok = false; }
  console.log(ok ? 'accepted' : 'rejected');
  process.exitCode = ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('the repair journey {string} runs on the {string} route', {timeout: 720000}, function (script, wire) {
    this.srb = observe(script, wire);
  });
  Then('the repair journey exits with code {int}', function (code) { check.exit(this.srb, code); });
  Then('the repair journey sent {int} shaper requests', function (n) { check.requests(this.srb, n); });
  Then('the repair journey requests that required submit are {string}', function (list) { check.forced(this.srb, list); });
  Then('the repair journey recovery requests are {string}', function (list) { check.recovery(this.srb, list); });
  Then('the repair journey recovery events are:', function (table) { check.events(this.srb, table.hashes()); });
  Then('the repair journey recovery message of request {int} is:', function (n, text) { check.message(this.srb, n, text); });
  Then('the repair journey feedback to call {string} is:', function (id, text) { check.feedback(this.srb, id, text); });
  Then('the repair journey feedback to call {string} contains {string}', function (id, text) { check.contains(this.srb, id, text); });
  Then('the repair journey feedback to call {string} does not contain {string}', function (id, text) { check.lacks(this.srb, id, text); });
  Then('the repair journey feedback to call {string} ends with:', function (id, text) { check.ends(this.srb, id, text); });
  Then('the repair journey stop text is:', function (text) { check.stop(this.srb, text); });
  Then('the repair journey has no stop text', function () { check.stop(this.srb, ''); });
  Then('the repair journey published its final package', function () { check.published(this.srb); });
  Then('the repair journey output contains {string}', function (text) { check.printed(this.srb, text); });
}
