'use strict';
// Acceptance steps of k-shape-recovery-completes-package: they compile the Go harness of
// k-shape-recovery-completes-package__harness.js into the shape package's tests through a Go
// overlay, build the test binary once per process, run one scripted native Shape journey per
// scenario and judge its KSR observation line. Run directly with good or bad, the file is the
// observer control of these judgements.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const P = 'k-shape-recovery-completes-package: ';
const TEST = 'TestKSRAcceptance';
const ROOT = path.resolve(__dirname, '..', '..');
// The overlay path is relative to ROOT, the directory go runs in.
const TARGET = 'shape/ksr_acceptance_test.go';
let binary = null;

function env(extra) {
  const out = {...process.env, ...extra};
  for (const k of Object.keys(out)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete out[k];
  return out;
}

function build() {
  if (binary) return binary;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ksr-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'ksr_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-recovery-completes-package__harness.js'));
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

function observe(script) {
  const bin = build();
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=150s'];
  const cwd = path.join(ROOT, 'shape');
  const r = spawnSync(bin, args, {cwd, env: env({PWD: cwd, KSR_SCRIPT: script}), encoding: 'utf8', timeout: 200000, maxBuffer: 32 << 20});
  const text = (r.stdout || '') + (r.stderr || '');
  const detail = TEST + ' (KSR_SCRIPT=' + script + '): shape.test ' + args.join(' ') +
    ', built by go test -c -vet=off -overlay ./shape, test binary exit=' + r.status + ' error=' + (r.error || 'none');
  const line = text.split('\n').find((l) => l.startsWith('KSR '));
  if (r.status !== 0 || !line || !text.includes('--- PASS: ' + TEST + ' ')) {
    throw new Error(detail + ': the journey did not run to a KSR observation\n' + text);
  }
  return {...JSON.parse(line.slice('KSR '.length)), detail};
}

function numbers(list) {
  return list === 'none' ? [] : list.split(',').map((s) => Number(s.trim()));
}

function why(o) {
  return o.detail + '; observation: ' + JSON.stringify({...o, detail: undefined});
}

const check = {
  exit(o, code) { assert.equal(o.exit, code, why(o) + '\nexit code'); },
  requests(o, n) { assert.equal(o.requests, n, why(o) + '\nshaper model requests'); },
  recovery(o, list) { assert.deepEqual(o.recovery, numbers(list), why(o) + '\nrequests that carry a new recovery message'); },
  feedback(o, id, text) { assert.equal((o.outputs || {})[id], text, why(o) + '\nfeedback to call ' + id); },
  refused(o, list) {
    const got = (o.refused || []).map((e) => e.attempt + ' ' + e.gate).join(', ') || 'none';
    assert.equal(got, list, why(o) + '\nrefused attempts and gates (submission_refused events)');
  },
  problems(o, attempt, text) {
    const row = (o.refused || []).find((e) => e.attempt === attempt);
    assert.ok(row, why(o) + '\nno submission_refused event of attempt ' + attempt);
    assert.equal(row.problems.join('\n'), text, why(o) + '\nproblem lines of refused attempt ' + attempt);
  },
  prechecks(o, list) {
    const rows = o.prechecks || [];
    for (const e of rows) {
      assert.deepEqual(Object.keys(e).sort(), ['attempt', 'gaps', 'owner'], why(o) + '\nsubmission_precheck fields of ' + JSON.stringify(e));
      assert.equal(e.owner, 'shape', why(o) + '\nsubmission_precheck owner');
    }
    assert.deepEqual(rows.map((e) => e.attempt), numbers(list), why(o) + '\nattempts the submission_precheck events follow');
  },
  gaps(o, n, text) {
    const row = (o.prechecks || [])[n - 1];
    assert.ok(row, why(o) + '\nno submission_precheck event number ' + n);
    assert.equal(row.gaps.join('\n'), text, why(o) + '\ngaps of submission_precheck event ' + n);
  },
  invalid(o, n) { assert.equal(o.invalid, n, why(o) + '\ninvalid_submissions of shaping_turn_counts'); },
  stop(o, text) { assert.equal(o.stop, text === '' ? '' : text + '\n', why(o) + '\nstop text'); },
  published(o) { assert.ok(o.published === true, why(o) + '\nthe complete package was not published; differing files: ' + JSON.stringify(o.unpublished)); },
  unwritten(o) { assert.ok(o.intent_written === false && o.published === false, why(o) + '\nan Intent was written'); },
};

// ---- observer controls ----

const RULE = 'submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report';
const RECEIVED = 'received: questions (list of 1), files (list of 3)';
const STEP_GAP = 'acceptance: features/steps/is-even__steps.js does not exist';
const FILES_TRAILER = 'The files of this call were retained with your earlier proposals. Add what is missing with propose_files; the next submission counts as an attempt.';
const ENVELOPE_TRAILER = 'No proposals from this call were retained; earlier proposals are unchanged. The next submission counts as an attempt.';
const SAME_PRECHECK = 'not submitted: this is the same refusal as attempt 1; these items are unchanged, and this call was not counted as a submission attempt:';
const STOP = 'shape/incomplete_package: blocked: the shaper could not complete the package within 3 attempts\n' +
  '  unresolved gap (attempt 3, gate validation):\n  ' + RULE + '\n  ' + RECEIVED;
const OLD_STOP = "shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts\n  " + RULE;
const OLD_ENVELOPE = 'invalid submission:\n' + RULE + '\nNo proposals from this submission were retained; earlier proposals are unchanged.';
const NEW_ENVELOPE = 'invalid submission:\n' + RULE + '\n' + RECEIVED + '\nNo proposals from this submission were retained; earlier proposals are unchanged.';
const STEP_REFUSAL = 'invalid submission:\n' + STEP_GAP + '\nEarlier well-formed proposals are retained; submit only corrections.';
const PRECHECK_FEEDBACK = SAME_PRECHECK + '\n' + STEP_GAP + '\n' + FILES_TRAILER;
const ENVELOPE_PRECHECK = SAME_PRECHECK + '\n' + RULE + '\n' + RECEIVED + '\n' + ENVELOPE_TRAILER;

// Known-good observations of the new behaviour, one per kind of judgement.
const GOOD = {
  prose: {detail: 'good prose', exit: 0, requests: 5, recovery: [3], stop: '', invalid: 1, published: true, intent_written: true, unpublished: null,
    outputs: {c1: STEP_REFUSAL, c2: PRECHECK_FEEDBACK},
    refused: [{attempt: 1, gate: 'intent', problems: [STEP_GAP]}],
    prechecks: [{owner: 'shape', attempt: 1, gaps: [STEP_GAP]}]},
  repeated: {detail: 'good repeated', exit: 1, requests: 6, recovery: [3], stop: STOP + '\n', invalid: 3, published: false, intent_written: false, unpublished: null,
    outputs: {c1: NEW_ENVELOPE, c2: ENVELOPE_PRECHECK},
    refused: [1, 2, 3].map((n) => ({attempt: n, gate: 'validation', problems: [RULE, RECEIVED]})),
    prechecks: [1, 2].map((n) => ({owner: 'shape', attempt: n, gaps: [RULE, RECEIVED]}))},
};

function judgeProse(o) {
  check.exit(o, 0); check.requests(o, 5); check.recovery(o, '3'); check.feedback(o, 'c1', STEP_REFUSAL);
  check.feedback(o, 'c2', PRECHECK_FEEDBACK); check.refused(o, '1 intent'); check.problems(o, 1, STEP_GAP);
  check.prechecks(o, '1'); check.gaps(o, 1, STEP_GAP); check.invalid(o, 1); check.stop(o, ''); check.published(o);
}

function judgeRepeated(o) {
  check.exit(o, 1); check.requests(o, 6); check.recovery(o, '3'); check.feedback(o, 'c1', NEW_ENVELOPE);
  check.feedback(o, 'c2', ENVELOPE_PRECHECK); check.refused(o, '1 validation, 2 validation, 3 validation');
  check.problems(o, 3, RULE + '\n' + RECEIVED); check.prechecks(o, '1, 2'); check.gaps(o, 2, RULE + '\n' + RECEIVED);
  check.invalid(o, 3); check.stop(o, STOP); check.unwritten(o);
}

function controls(mode) {
  const accepts = (fn) => { try { fn(); return true; } catch (e) { return false; } };
  if (mode === 'good') {
    // The live harness on a journey this Intent does not change.
    const live = observe('valid-first');
    check.exit(live, 0); check.requests(live, 1); check.recovery(live, 'none'); check.refused(live, 'none');
    check.prechecks(live, 'none'); check.invalid(live, 0); check.stop(live, ''); check.published(live);
    judgeProse(GOOD.prose);
    judgeRepeated(GOOD.repeated);
    return true;
  }
  // Known-bad observations: the unchanged tree's journeys and plausible wrong implementations.
  const oldProse = {...GOOD.prose, detail: 'old prose', invalid: 2, prechecks: [],
    outputs: {c1: STEP_REFUSAL, c2: STEP_REFUSAL + '\nUnchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.'},
    refused: [{attempt: 1, gate: 'intent', problems: [STEP_GAP]}, {attempt: 2, gate: 'intent', problems: [STEP_GAP]}]};
  const oldRepeated = {...GOOD.repeated, detail: 'old repeated', requests: 4, stop: OLD_STOP + '\n', prechecks: [],
    outputs: {c1: OLD_ENVELOPE, c2: OLD_ENVELOPE}, refused: [1, 2, 3].map((n) => ({attempt: n, gate: 'validation', problems: [RULE]}))};
  const bad = [
    () => judgeProse(oldProse),
    () => judgeRepeated(oldRepeated),
    () => judgeProse({...GOOD.prose, prechecks: []}),
    () => judgeProse({...GOOD.prose, invalid: 2}),
    () => judgeProse({...GOOD.prose, requests: 6}),
    () => judgeProse({...GOOD.prose, published: false}),
    () => judgeProse({...GOOD.prose, outputs: {...GOOD.prose.outputs, c2: STEP_REFUSAL}}),
    () => judgeProse({...GOOD.prose, prechecks: [{owner: 'shape', attempt: 1, gaps: [STEP_GAP], request: 3}]}),
    () => judgeProse({...GOOD.prose, prechecks: [{owner: 'refine', attempt: 1, gaps: [STEP_GAP]}]}),
    () => judgeProse({...GOOD.prose, prechecks: [{owner: 'shape', attempt: 2, gaps: [STEP_GAP]}]}),
    () => judgeRepeated({...GOOD.repeated, stop: OLD_STOP + '\n'}),
    () => judgeRepeated({...GOOD.repeated, outputs: {...GOOD.repeated.outputs, c1: OLD_ENVELOPE}}),
    () => judgeRepeated({...GOOD.repeated, intent_written: true}),
    () => judgeRepeated({...GOOD.repeated, exit: 0}),
    () => judgeRepeated({...GOOD.repeated, prechecks: GOOD.repeated.prechecks.map((e) => ({...e, gaps: [RULE]}))}),
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
  When(P + 'the scripted shaper runs the journey {string}', {timeout: 720000}, function (script) { this.ksr = observe(script); });
  Then(P + 'shaping exits with code {int}', function (code) { check.exit(this.ksr, code); });
  Then(P + 'the shaper sent {int} model requests', function (n) { check.requests(this.ksr, n); });
  Then(P + 'the requests carrying a recovery message are {string}', function (list) { check.recovery(this.ksr, list); });
  Then(P + 'the feedback to call {string} is:', function (id, text) { check.feedback(this.ksr, id, text); });
  Then(P + 'the refused attempts and gates are {string}', function (list) { check.refused(this.ksr, list); });
  Then(P + 'the problems of refused attempt {int} are:', function (n, text) { check.problems(this.ksr, n, text); });
  Then(P + 'the pre-checks follow attempts {string}', function (list) { check.prechecks(this.ksr, list); });
  Then(P + 'the gaps of pre-check {int} are:', function (n, text) { check.gaps(this.ksr, n, text); });
  Then(P + 'the invalid submissions counted are {int}', function (n) { check.invalid(this.ksr, n); });
  Then(P + 'the stop text is:', function (text) { check.stop(this.ksr, text); });
  Then(P + 'there is no stop text', function () { check.stop(this.ksr, ''); });
  Then(P + 'the complete package was published', function () { check.published(this.ksr); });
  Then(P + 'no Intent was written', function () { check.unwritten(this.ksr); });
}
