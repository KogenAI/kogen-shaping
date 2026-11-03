'use strict';
// Acceptance of k-shape-step-binding-proof. The When step shapes one synthetic case of the request:
// it compiles the shape package's tests with the harness of k-shape-step-binding-proof__harness.js
// (through a Go overlay), runs TestKShapeStepBindingProofAcceptance for the case and reads what
// the shaper and the run log saw. The Then steps judge that. Run this file directly with good or
// bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeStepBindingProofAcceptance';
let built;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-step-binding-proof-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_shape_step_binding_proof_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-step-binding-proof__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_shape_step_binding_proof_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  built = r.status === 0 && !r.error ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// shapeCase runs the harness for one case and returns its observation.
function shapeCase(name) {
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  fs.rmSync(out, {force: true});
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=240s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KSBP_CASE: name, KSBP_OUT: out}), encoding: 'utf8', timeout: 260000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const ran = `${b.bin} ${args.join(' ')} (KSBP_CASE=${name})`;
  if (r.error || r.status !== 0) throw new Error(`${TEST} case ${name} failed: ${ran} exit=${r.status} error=${r.error || ''}\n${text}`);
  if (!text.includes('--- PASS: ' + TEST)) throw new Error(`${TEST} case ${name} did not run (absent, skipped or zero runs): ${ran} exit=${r.status}\n${text}`);
  if (!fs.existsSync(out)) throw new Error(`${TEST} case ${name} wrote no observation to ${out}: ${ran}\n${text}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function where(obs) { return `${TEST} case ${obs.case}: observation ${JSON.stringify(obs)}`; }

function onlySelfcheck(obs) {
  assert.ok(Array.isArray(obs.selfchecks) && obs.selfchecks.length === 1, `${where(obs)}: want exactly one step_selfcheck event`);
  return obs.selfchecks[0];
}

// The judgments of the Then steps.
const judge = {
  refused(obs, feedback) {
    const check = onlySelfcheck(obs);
    assert.strictEqual(obs.feedback, feedback, `${where(obs)}: the feedback returned to the shaper`);
    assert.strictEqual(check.ok, false, `${where(obs)}: the step self-check must fail`);
    assert.strictEqual(obs.readiness, false, `${where(obs)}: Shape must not go on to readiness`);
    assert.strictEqual(obs.candidate_acceptance, false, `${where(obs)}: the candidate's acceptance must not run`);
    assert.strictEqual(obs.readiness_record, false, `${where(obs)}: no readiness record may be written`);
    assert.strictEqual(obs.published, 0, `${where(obs)}: nothing may be published`);
  },
  passed(obs) {
    const check = onlySelfcheck(obs);
    assert.strictEqual(check.ok, true, `${where(obs)}: the step self-check must pass`);
    assert.strictEqual(obs.readiness, true, `${where(obs)}: Shape must go on to readiness`);
    assert.strictEqual(obs.candidate_acceptance, true, `${where(obs)}: readiness runs the candidate's acceptance`);
  },
  bound(obs, texts) {
    const check = onlySelfcheck(obs);
    assert.deepStrictEqual(check.steps, texts.map(text => ({text, ok: true, detail: ''})), `${where(obs)}: every step line binds to exactly one definition`);
  },
  noBinding(obs) {
    const check = onlySelfcheck(obs);
    assert.deepStrictEqual(check.steps, [], `${where(obs)}: a file that does not load yields no binding result`);
  },
  warnings(obs, want) {
    const check = onlySelfcheck(obs);
    assert.deepStrictEqual(check.warnings, want, `${where(obs)}: the self-check warnings`);
  },
  noSentinel(obs) {
    assert.strictEqual(obs.sentinel, '', `${where(obs)}: a step body or hook ran`);
  },
};

if (require.main === module) {
  // Known observations: what a correct implementation shows for each kind of case, and wrong
  // variants each judgment must reject.
  const H = 'Step self-check failed; nothing was published. Fix the step files and resubmit.\n';
  const feedback = H + 'acceptance.feature line 5: ambiguous step: Then binding-proof: code 12 matches 2 definitions: ^binding-proof: code \\d+$ in features/steps/binding-proof__steps.js; binding-proof: code {int} in features/steps/binding-proof__steps.js';
  const texts = ['binding-proof: input 1', 'binding-proof: I retain it', 'binding-proof: code 12', 'binding-proof: text "v2.4+[x]"', 'binding-proof: text "hello.world"'];
  const warning = {pattern: 'binding-proof: unused', file: 'features/steps/binding-proof__steps.js', detail: 'unused definition'};
  const steps = texts.map(text => ({text, ok: true, detail: ''}));
  const refused = {case: 'duplicate', feedback, selfchecks: [{ok: false, steps: [], warnings: []}], readiness: false, candidate_acceptance: false, readiness_record: false, published: 0, sentinel: ''};
  const passed = {case: 'unused', feedback: 'not ready:\nmissing: A numbered result (line 2)', selfchecks: [{ok: true, steps, warnings: [warning]}], readiness: true, candidate_acceptance: true, readiness_record: false, published: 0, sentinel: ''};
  const loadFailed = {...refused, case: 'load-failure', selfchecks: [{ok: false, steps: [], warnings: []}]};
  const runs = [
    o => judge.refused(o.refused, feedback), o => judge.noSentinel(o.refused),
    o => judge.passed(o.passed), o => judge.bound(o.passed, texts), o => judge.warnings(o.passed, [warning]), o => judge.noSentinel(o.passed),
    o => judge.noBinding(o.loadFailed),
  ];
  const all = o => { for (const run of runs) run(o); };
  const good = {refused, passed, loadFailed};
  const with_ = (key, change) => ({...good, [key]: {...good[key], ...change}});
  const check = (key, c) => ({...good[key].selfchecks[0], ...c});
  const bad = {
    'old ambiguous feedback': with_('refused', {feedback: H + '- steps binding-proof: code 12: ambiguous step (2 definitions)'}),
    'one definition missing': with_('refused', {feedback: H + 'acceptance.feature line 5: ambiguous step: Then binding-proof: code 12 matches 2 definitions: ^binding-proof: code \\d+$ in features/steps/binding-proof__steps.js'}),
    'refused after readiness': with_('refused', {readiness: true}),
    'refused after the candidate acceptance ran': with_('refused', {candidate_acceptance: true}),
    'refused with a readiness record': with_('refused', {readiness_record: true}),
    'refused but published': with_('refused', {published: 1}),
    'refusal without a failed self-check': with_('refused', {selfchecks: [check('refused', {ok: true})]}),
    'two self-checks': with_('refused', {selfchecks: [check('refused', {}), check('refused', {})]}),
    'a sentinel ran during refusal': with_('refused', {sentinel: 'executed\n'}),
    'self-check failed': with_('passed', {selfchecks: [check('passed', {ok: false})]}),
    'readiness not reached': with_('passed', {readiness: false}),
    'candidate acceptance not run': with_('passed', {candidate_acceptance: false}),
    'an expanded example not bound': with_('passed', {selfchecks: [check('passed', {steps: steps.map((s, i) => i === 4 ? {...s, ok: false, detail: 'undefined step'} : s)})]}),
    'an expanded example missing': with_('passed', {selfchecks: [check('passed', {steps: steps.slice(0, 4)})]}),
    'ambiguous line passed': with_('passed', {selfchecks: [check('passed', {steps: steps.map((s, i) => i === 2 ? {...s, ok: false, detail: 'ambiguous step (2 definitions)'} : s)})]}),
    'no warning': with_('passed', {selfchecks: [check('passed', {warnings: []})]}),
    'an extra warning': with_('passed', {selfchecks: [check('passed', {warnings: [warning, {...warning, pattern: 'binding-proof: text {string}'}]})]}),
    'a sentinel ran': with_('passed', {sentinel: 'executed\n'}),
    'binding results despite a load failure': with_('loadFailed', {selfchecks: [check('loadFailed', {steps})]}),
  };
  let accepted = '';
  try { all(good); } catch (e) { accepted = 'good: ' + e.message; }
  const mode = process.argv[2];
  if (mode === 'good') {
    if (process.argv.includes('--diagnostic') && accepted) console.error(accepted);
    console.log(accepted ? 'rejected' : 'accepted');
    process.exitCode = accepted ? 1 : 0;
  } else if (mode === 'bad') {
    const wrongly = Object.keys(bad).filter(name => { try { all(bad[name]); return true; } catch { return false; } });
    if (process.argv.includes('--diagnostic') && wrongly.length) console.error('accepted: ' + wrongly.join(', '));
    console.log(wrongly.length || accepted ? 'accepted' : 'rejected');
    process.exitCode = wrongly.length || accepted ? 0 : 1;
  } else {
    console.error('usage: node features/steps/k-shape-step-binding-proof__steps.js good|bad');
    process.exitCode = 2;
  }
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'k-shape-step-binding-proof: ';
  When(P + 'the scripted shaper submits the {string} binding-proof package', {timeout: 560000}, function (name) {
    this.ksbp = shapeCase(name);
  });
  Then(P + 'Shape refuses it before readiness with this feedback:', function (feedback) {
    judge.refused(this.ksbp, feedback);
  });
  Then(P + 'the step self-check passes and readiness runs', function () {
    judge.passed(this.ksbp);
  });
  Then(P + 'each of these step lines binds to exactly one definition:', function (table) {
    judge.bound(this.ksbp, table.raw().map(row => row[0]));
  });
  Then(P + 'the self-check reports no binding result', function () {
    judge.noBinding(this.ksbp);
  });
  Then(P + 'the self-check warns of no unused definition', function () {
    judge.warnings(this.ksbp, []);
  });
  Then(P + 'the self-check warns only that {string} in {string} is an unused definition', function (pattern, file) {
    judge.warnings(this.ksbp, [{pattern, file, detail: 'unused definition'}]);
  });
  Then(P + 'no step body or hook ran', function () {
    judge.noSentinel(this.ksbp);
  });
}
