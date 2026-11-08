'use strict';
// Acceptance of shape-refuses-steps-over-step-limit. The When step shapes one synthetic case of the
// request: it compiles the shape package's tests with the harness of
// shape-refuses-steps-over-step-limit__harness.js (through a Go overlay), runs
// TestShapeRefusesStepsOverStepLimitAcceptance for the case and reads what the shaper and the run
// log saw. The Then steps judge that. Run this file directly with good or bad to check the
// judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestShapeRefusesStepsOverStepLimitAcceptance';
let built;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// compile builds the shape tests with the harness once per process.
function compile() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-refuses-steps-over-step-limit-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'shape_refuses_steps_over_step_limit_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-refuses-steps-over-step-limit__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/shape_refuses_steps_over_step_limit_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 110000});
  built = r.status === 0 && !r.error ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// shapeCase runs the harness for one case and returns its observation.
function shapeCase(name) {
  const b = compile();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  fs.rmSync(out, {force: true});
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=100s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({SRSL_CASE: name, SRSL_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const ran = `${b.bin} ${args.join(' ')} (SRSL_CASE=${name})`;
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

// goSources lists the non-test Go files of the shape package with their text.
function goSources(dir) {
  return fs.readdirSync(dir).filter(f => f.endsWith('.go') && !f.endsWith('_test.go')).map(f => ({file: f, text: fs.readFileSync(path.join(dir, f), 'utf8')}));
}

// ownLimit lists the places that hold a 120-second limit of their own: a duration of 120 seconds,
// a bare 120 assigned or passed as a value, or a string that spells the limit as 120 s.
function ownLimit(sources) {
  const found = [];
  const patterns = [/\b120\s*\*\s*time\.Second\b/, /\btime\.Second\s*\*\s*120\b/, /\btime\.Duration\(\s*120\s*\)/, /[=(,]\s*120\s*(?:[),;]|$)/, /["`][^"`\n]*\b120 s\b[^"`\n]*["`]/];
  for (const {file, text} of sources) {
    text.split('\n').forEach((line, i) => { if (patterns.some(p => p.test(line))) found.push(`${file}:${i + 1}: ${line.trim()}`); });
  }
  return found;
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
  noStepLimit(obs) {
    assert.deepStrictEqual(onlySelfcheck(obs).step_limit, [], `${where(obs)}: no step may be reported over the step limit`);
  },
  limit(obs, seconds) {
    assert.strictEqual(obs.approval_limit_seconds, seconds, `${where(obs)}: verify.StepTimeout in seconds`);
    assert.ok(obs.feedback.includes(`below the ${seconds} s step limit`), `${where(obs)}: the feedback must name the limit of ${seconds} s that approval uses`);
  },
  noOwnLimit(sources) {
    const found = ownLimit(sources);
    assert.deepStrictEqual(found, [], `a Go file of the shape package outside its tests holds a 120-second limit of its own instead of verify.StepTimeout:\n${found.join('\n')}`);
  },
  noSentinel(obs) {
    assert.strictEqual(obs.sentinel, '', `${where(obs)}: a step body ran`);
  },
};

const HEADER = 'Step self-check failed; nothing was published. Fix the step files and resubmit.\n';
const expectedFeedback = (step, runner) => `${HEADER}- step_limit limit-proof: ${step}: runs a whole test suite with ${runner} and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the 120 s step limit`;

if (require.main === module) {
  // Known observations: what a correct implementation shows for each kind of case, and wrong
  // variants each judgment must reject.
  const feedback = expectedFeedback('I run the Go suite', 'go test');
  const refused = {case: 'go-shell', feedback, selfchecks: [{ok: false, step_limit: ['limit-proof: I run the Go suite']}], readiness: false, candidate_acceptance: false, readiness_record: false, published: 0, sentinel: '', approval_limit_seconds: 120};
  const passed = {case: 'narrow-go-package', feedback: 'not ready:\nmissing: A suite step (line 2)', selfchecks: [{ok: true, step_limit: []}], readiness: true, candidate_acceptance: true, readiness_record: false, published: 0, sentinel: '', approval_limit_seconds: 120};
  const cleanSource = [{file: 'a.go', text: 'var a = verify.StepTimeout\nconst m = "timed out after %d s"\n// 120 seconds\n'}];
  const runs = [
    o => judge.refused(o.refused, feedback), o => judge.limit(o.refused, 120), o => judge.noSentinel(o.refused),
    o => judge.passed(o.passed), o => judge.noStepLimit(o.passed), o => judge.noSentinel(o.passed),
    o => judge.noOwnLimit(o.sources),
  ];
  const all = o => { for (const run of runs) run(o); };
  const good = {refused, passed, sources: cleanSource};
  const with_ = (key, change) => ({...good, [key]: {...good[key], ...change}});
  const check = (key, c) => ({...good[key].selfchecks[0], ...c});
  const bad = {
    'feedback names no step': with_('refused', {feedback: HEADER + '- step_limit : runs a whole test suite with go test and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the 120 s step limit'}),
    'feedback names no limit': with_('refused', {feedback: HEADER + '- step_limit limit-proof: I run the Go suite: runs a whole test suite with go test and no narrowing; narrow it to one named test or package, or split the step'}),
    'feedback names another limit': with_('refused', {feedback: expectedFeedback('I run the Go suite', 'go test').replace('120 s', '30 s')}),
    'feedback names the step twice': with_('refused', {feedback: feedback + '\n' + feedback.split('\n')[1]}),
    'feedback names another runner': with_('refused', {feedback: expectedFeedback('I run the Go suite', 'mix test')}),
    'refused after readiness': with_('refused', {readiness: true}),
    'refused after the candidate acceptance ran': with_('refused', {candidate_acceptance: true}),
    'refused with a readiness record': with_('refused', {readiness_record: true}),
    'refused but published': with_('refused', {published: 1}),
    'refusal without a failed self-check': with_('refused', {selfchecks: [check('refused', {ok: true})]}),
    'two self-checks': with_('refused', {selfchecks: [check('refused', {}), check('refused', {})]}),
    'a sentinel ran during refusal': with_('refused', {sentinel: 'executed\n'}),
    'approval limit differs': with_('refused', {approval_limit_seconds: 60}),
    'self-check failed': with_('passed', {selfchecks: [check('passed', {ok: false})]}),
    'readiness not reached': with_('passed', {readiness: false}),
    'candidate acceptance not run': with_('passed', {candidate_acceptance: false}),
    'a step reported over the limit': with_('passed', {selfchecks: [check('passed', {step_limit: ['limit-proof: I run one Go package']})]}),
    'a sentinel ran': with_('passed', {sentinel: 'executed\n'}),
    'a duration constant of its own': {...good, sources: [{file: 'steplimit.go', text: 'const stepLimit = 120 * time.Second\n'}]},
    'a duration of its own, written the other way': {...good, sources: [{file: 'steplimit.go', text: 'var stepLimit = time.Second * 120\n'}]},
    'a converted duration of its own': {...good, sources: [{file: 'steplimit.go', text: 'var stepLimit = time.Duration(120)\n'}]},
    'a bare number of its own': {...good, sources: [{file: 'steplimit.go', text: 'const stepLimitSeconds = 120\n'}]},
    'a spelled-out limit of its own': {...good, sources: [{file: 'steplimit.go', text: 'const detail = "so it finishes below the 120 s step limit"\n'}]},
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
    console.error('usage: node features/steps/shape-refuses-steps-over-step-limit__steps.js good|bad');
    process.exitCode = 2;
  }
} else {
  const {Given, When, Then} = require('@cucumber/cucumber');
  const P = 'shape-refuses-steps-over-step-limit: ';
  Given(P + 'the shape acceptance harness is compiled', {timeout: 115000}, function () {
    const b = compile();
    if (b.error) throw new Error(b.error);
  });
  When(P + 'the scripted shaper submits the {string} step-limit package', {timeout: 115000}, function (name) {
    this.srsl = shapeCase(name);
  });
  Then(P + 'Shape refuses it before readiness with this feedback:', function (feedback) {
    judge.refused(this.srsl, feedback);
  });
  Then(P + 'the step self-check passes and readiness runs', function () {
    judge.passed(this.srsl);
  });
  Then(P + 'the self-check reports no step over the step limit', function () {
    judge.noStepLimit(this.srsl);
  });
  Then(P + 'the feedback gives the {int} s limit that verify.StepTimeout holds', function (seconds) {
    judge.limit(this.srsl, seconds);
  });
  Then(P + 'no Go file of the shape package outside its tests holds a 120-second limit of its own', function () {
    judge.noOwnLimit(goSources(path.resolve('shape')));
  });
  Then(P + 'no step body ran', function () {
    judge.noSentinel(this.srsl);
  });
}
