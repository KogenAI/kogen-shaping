'use strict';
// Acceptance of k-shape-scope-gate-counts-package. A When step shapes one synthetic case: it
// compiles the shape package's tests with the harness of k-shape-scope-gate-counts-package__harness.js
// (through a Go overlay), runs TestKShapeScopeGateCountsPackageAcceptance for the case with a
// scripted external shaper, and reads what the run showed; another When step runs the delivered
// scope tests. Nothing in the checkout is written. The Then steps judge what was seen. Run this
// file directly with good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeScopeGateCountsPackageAcceptance';
const SCOPE_TESTS = ['test', '-count=1', '-v', '-run', '^TestScope', './shape'];
let built = null;
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-scope-gate-counts-package-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_scope_gate_counts_package_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-scope-gate-counts-package__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_scope_gate_counts_package_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {env: env(), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  if (r.error || r.status !== 0) throw new Error(`${TEST}: building the shape tests: go ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`);
  built = {bin, dir};
  return built;
}

// shapeCase runs the harness for one case once and returns what the run showed.
function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=400s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KGP_CASE: name, KGP_OUT: out}), encoding: 'utf8', timeout: 420000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KGP_CASE=${name}): shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

// scopeTests runs the delivered scope tests of the shape package.
function scopeTests() {
  const r = spawnSync('go', SCOPE_TESTS, {env: env(), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  return {command: 'go ' + SCOPE_TESTS.join(' '), status: r.status, error: String(r.error || ''), output: (r.stdout || '') + (r.stderr || '')};
}

function shown(o) {
  const feedback = o.feedback.map((text, i) => `result of submission ${i + 1}:\n${text}`).join('\n----\n');
  return `case ${o.case} (tools.InputBudgetBytes ${o.bound}): shape ended with exit ${o.exit}, err ${o.err}; published=${o.published}; ` +
    `${o.shaper} shaper round(s), ${o.decision} decision check(s); refused gates ${JSON.stringify(o.gates)}; data/big.json ${o.big} bytes\n` +
    `${feedback || '(no submission results)'}\nprogress (tail):\n${String(o.progress).slice(-1500)}`;
}

function result(o, n) {
  const text = o.feedback[n - 1];
  assert.equal(typeof text, 'string', `submission ${n} got no result back\n${shown(o)}`);
  return text;
}

const checks = {
  bound(o, n) {
    assert.equal(o.bound, n, `the run used tools.InputBudgetBytes ${o.bound}, want ${n}\n${shown(o)}`);
  },
  bigNamed(o) {
    assert.ok(o.big > o.bound, `the tracked named file data/big.json has ${o.big} bytes, not more than the ${o.bound}-byte bound\n${shown(o)}`);
  },
  scopeRefusals(o, n) {
    const got = o.gates.filter(g => g === 'scope').length;
    assert.equal(got, n, `the scope check refused ${got} submission(s), want ${n}\n${shown(o)}`);
  },
  decisions(o, n) {
    assert.equal(o.decision, n, `the decision check ran ${o.decision} time(s), want ${n}\n${shown(o)}`);
  },
  decisionsAtLeast(o, n) {
    assert.ok(o.decision >= n, `the decision check ran ${o.decision} time(s), want at least ${n}\n${shown(o)}`);
  },
  is(o, n, want) {
    assert.equal(result(o, n), want, `the result of submission ${n} differs\n${shown(o)}`);
  },
  contains(o, n, text) {
    assert.ok(result(o, n).includes(text), `the result of submission ${n} does not contain ${JSON.stringify(text)}\n${shown(o)}`);
  },
  lacks(o, n, text) {
    assert.ok(!result(o, n).includes(text), `the result of submission ${n} contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  startsWith(o, n, text) {
    assert.ok(result(o, n).startsWith(text), `the result of submission ${n} does not start with ${JSON.stringify(text)}\n${shown(o)}`);
  },
  endsWith(o, n, text) {
    assert.ok(result(o, n).endsWith(text), `the result of submission ${n} does not end with ${JSON.stringify(text)}\n${shown(o)}`);
  },
  published(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
  },
  ended(o, reason, exit) {
    assert.deepEqual({reason: `${o.class}/${o.reason}`, exit: o.exit}, {reason, exit}, `shaping ended differently\n${shown(o)}`);
  },
  scopeTestsPassed(r) {
    const says = `${r.command} (the delivered scope tests, the tests matching ^TestScope in ./shape) exited ${r.status}${r.error ? ' error ' + r.error : ''}:\n${r.output}`;
    assert.equal(r.status, 0, `the scope tests failed: ${says}`);
    assert.ok(!/no tests to run/.test(r.output), `no test matching ^TestScope ran: ${says}`);
    assert.ok(/^--- PASS: TestScope/m.test(r.output), `no test matching ^TestScope reported PASS: ${says}`);
    assert.ok(!/^\s*--- FAIL/m.test(r.output), `a test reported FAIL: ${says}`);
    assert.ok(/^ok\s+github\.com\/KogenAI\/kogen-shaping\/shape\s/m.test(r.output), `the package github.com/KogenAI/kogen-shaping/shape did not report ok: ${says}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const guidance = 'Above the bound, split the work into an ordered ## Stages plan of at least two stages, each with a real gate in its checks and each inside the bound, ' +
    "or narrow the Intent to the requested behaviour. A stage's goal names in backticks the files that stage changes; a named file that no stage goal names counts in every stage, " +
    'and every stage carries the whole Intent package. Do not satisfy the bound by deleting requested behaviour: every behaviour and example the request asks for keeps its scenario. ' +
    'Resubmitting this candidate unchanged is refused again.';
  const unchanged = '\nUnchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.';
  const ack = 'Inspect whether fixtures, inputs and commands reached the intended behavior, using the existing read/probe tools when needed. Repair demonstrated contract defects.';
  const edgeLine = n => `Scope check: the Intent is estimated at ${n} bytes of one review (scenarios 5, acceptance harnesses 2, named files 1 in areas 1), over the 524288-byte bound of one review (tools.InputBudgetBytes).`;
  const edgeItems = '- .kogen/intents/is-even/intent.md: 4097 bytes (Intent package)\n- .kogen/intents/is-even/acceptance.feature: 346 bytes (Intent package)\n' +
    '- features/steps/is-even__steps.js: 210 bytes (Intent package)\n- features/steps/is-even__padding.js: 519516 bytes (Intent package)\n- .kogen/intents/is-even/model.qnt: 120 bytes (Intent package)\n';
  const edgeRefusal = edgeLine(524289) + '\n' + edgeItems + guidance;
  const overRefusal = 'Scope check: the Intent is estimated at 526213 bytes of one review (scenarios 2, acceptance harnesses 2, named files 0 in areas 0), over the 524288-byte bound of one review (tools.InputBudgetBytes).\n' +
    '- .kogen/intents/is-even/intent.md: 523 bytes (Intent package)\n- .kogen/intents/is-even/acceptance.feature: 192 bytes (Intent package)\n' +
    '- features/steps/is-even__steps.js: 210 bytes (Intent package)\n- features/steps/is-even__padding.js: 525288 bytes (Intent package)\n' + guidance;
  const namedOld = 'Scope check: the Intent is estimated at 529301 bytes of one review (scenarios 2, acceptance harnesses 1, named files 1 in areas 1), over the 524288-byte bound of one review (tools.InputBudgetBytes).\n' +
    '- .kogen/intents/is-even/intent.md: 516 bytes (Intent package)\n- .kogen/intents/is-even/acceptance.feature: 192 bytes (Intent package)\n' +
    '- features/steps/is-even__steps.js: 210 bytes (Intent package)\n- data/big.json: 528383 bytes (named file, area data)\n' + guidance;
  const base = {bound: 524288, big: 528383, stdout: '', progress: ''};
  const named = {...base, case: 'named', exit: 0, err: '<nil>', class: '', reason: '', published: true, gates: [], shaper: 2, decision: 2, feedback: [ack]};
  const edge = {...base, case: 'edge', exit: 3, err: 'environment/tool_missing: node is not available at ; reinstall Kogen', class: 'environment', reason: 'tool_missing',
    published: false, gates: ['scope'], shaper: 2, decision: 1, feedback: [edgeRefusal]};
  const over = {...base, case: 'over', exit: 1, err: 'intent/not_ready: the Intent is not ready to build', class: 'intent', reason: 'not_ready',
    published: false, gates: ['scope', 'scope', 'scope'], shaper: 3, decision: 0, feedback: [overRefusal, overRefusal + unchanged]};
  const goOK = fs.readFileSync(__filename, 'utf8').split('\n// GO-TEST-OUTPUT ').slice(1).map(line => line.replace(/\\t/g, '\t')).join('\n') + '\n';
  const tests = {command: 'go ' + SCOPE_TESTS.join(' '), status: 0, error: '', output: goOK};
  const namedJudgments = [
    o => checks.bound(o, 524288), o => checks.bigNamed(o), o => checks.scopeRefusals(o, 0), o => checks.lacks(o, 1, 'Scope check'),
    o => checks.decisions(o, 2), o => checks.published(o),
  ];
  const edgeJudgments = [
    o => checks.bound(o, 524288), o => checks.is(o, 1, edgeRefusal), o => checks.scopeRefusals(o, 1), o => checks.decisionsAtLeast(o, 1),
    o => checks.contains(o, 1, 'estimated at 524289 bytes'), o => checks.contains(o, 1, '(scenarios 5,'), o => checks.contains(o, 1, '- .kogen/intents/is-even/model.qnt: 120 bytes (Intent package)'),
    o => checks.contains(o, 1, 'named files 1 in areas 1)'), o => checks.lacks(o, 1, 'missing/new.js'), o => checks.lacks(o, 1, '.kogen/notes.txt'), o => checks.lacks(o, 1, 'data/big.json'),
  ];
  const overJudgments = [
    o => checks.startsWith(o, 1, 'Scope check: the Intent is estimated at 526213 bytes of one review (scenarios 2, acceptance harnesses 2, named files 0 in areas 0), over the 524288-byte bound of one review (tools.InputBudgetBytes).\n'),
    o => checks.endsWith(o, 1, guidance), o => checks.endsWith(o, 2, guidance + unchanged), o => checks.scopeRefusals(o, 3), o => checks.decisions(o, 0), o => checks.ended(o, 'intent/not_ready', 1),
  ];
  const good = [
    ...namedJudgments.map(check => [named, check]), ...edgeJudgments.map(check => [edge, check]), ...overJudgments.map(check => [over, check]),
    [tests, r => checks.scopeTestsPassed(r)],
  ];
  const changed = (o, change) => {
    const c = JSON.parse(JSON.stringify(o));
    change(c);
    return c;
  };
  // The unchanged tree's observations of the named and edge cases, as its run showed them.
  const namedNow = changed(named, o => { Object.assign(o, {exit: 1, err: 'intent/not_ready: the Intent is not ready to build', class: 'intent', reason: 'not_ready', published: false, gates: ['scope', 'scope', 'scope'], shaper: 3, decision: 0, feedback: [namedOld, namedOld + unchanged]}); });
  const edgeOld = n => `Scope check: the Intent is estimated at ${n} bytes of one review (scenarios 5, acceptance harnesses 2, named files 1 in areas 1), over the 524288-byte bound of one review (tools.InputBudgetBytes).\n` +
    edgeItems.replace('4097', n === 1052672 ? '4097' : '4096') + '- data/big.json: 528383 bytes (named file, area data)\n' + guidance;
  const edgeNow = changed(edge, o => { Object.assign(o, {exit: 1, err: 'intent/not_ready: the Intent is not ready to build', class: 'intent', reason: 'not_ready', gates: ['scope', 'scope', 'scope'], shaper: 3, decision: 0, feedback: [edgeOld(1052672), edgeOld(1052671)]}); });
  const fail = goOK.replace('--- PASS: TestScopeStageUnits', '--- FAIL: TestScopeStageUnits').replace(/^PASS$/m, 'FAIL').replace(/^ok\s.*$/m, 'FAIL\tgithub.com/KogenAI/kogen-shaping/shape\t1.008s');
  const bad = [
    [namedNow, o => checks.scopeRefusals(o, 0)],
    [namedNow, o => checks.lacks(o, 1, 'Scope check')],
    [namedNow, o => checks.decisions(o, 2)],
    [namedNow, o => checks.published(o)],
    [changed(named, o => { o.big = 4096; }), o => checks.bigNamed(o)],
    [changed(named, o => { o.bound = 1048576; }), o => checks.bound(o, 524288)],
    [changed(named, o => { o.feedback = []; }), o => checks.lacks(o, 1, 'Scope check')],
    [edgeNow, o => checks.is(o, 1, edgeRefusal)],
    [edgeNow, o => checks.scopeRefusals(o, 1)],
    [edgeNow, o => checks.decisionsAtLeast(o, 1)],
    [edgeNow, o => checks.contains(o, 1, 'estimated at 524289 bytes')],
    [edgeNow, o => checks.lacks(o, 1, 'data/big.json')],
    [changed(edge, o => { o.feedback[0] = o.feedback[0].replace('named files 1 in areas 1', 'named files 2 in areas 2') + '\n- .kogen/notes.txt: 6 bytes (named file, area .kogen)'; }), o => checks.contains(o, 1, 'named files 1 in areas 1)')],
    [changed(edge, o => { o.feedback[0] = o.feedback[0] + '\n- .kogen/notes.txt: 6 bytes (named file, area .kogen)'; }), o => checks.lacks(o, 1, '.kogen/notes.txt')],
    [changed(edge, o => { o.feedback[0] = o.feedback[0].replace('named files 1 in areas 1', 'named files 2 in areas 2') + '\n- missing/new.js: 0 bytes (named file, area missing)'; }), o => checks.lacks(o, 1, 'missing/new.js')],
    [changed(edge, o => { o.feedback[0] = o.feedback[0].replace('524289', '524169').replace('- .kogen/intents/is-even/model.qnt: 120 bytes (Intent package)\n', ''); }), o => checks.contains(o, 1, '- .kogen/intents/is-even/model.qnt: 120 bytes (Intent package)')],
    [changed(edge, o => { o.feedback[0] = o.feedback[0].replace('scenarios 5', 'scenarios 3'); }), o => checks.contains(o, 1, '(scenarios 5,')],
    [changed(edge, o => { o.feedback[0] = 'invalid submission:\n' + o.feedback[0]; }), o => checks.is(o, 1, edgeRefusal)],
    [changed(edge, o => { o.gates = ['scope', 'scope']; }), o => checks.scopeRefusals(o, 1)],
    [changed(edge, o => { o.decision = 0; }), o => checks.decisionsAtLeast(o, 1)],
    [changed(over, o => { o.decision = 1; }), o => checks.decisions(o, 0)],
    [changed(over, o => { o.gates = ['scope', 'decision_check', 'decision_check']; }), o => checks.scopeRefusals(o, 3)],
    [changed(over, o => { o.class = 'shape'; o.reason = 'invalid_output'; }), o => checks.ended(o, 'intent/not_ready', 1)],
    [changed(over, o => { o.exit = 4; }), o => checks.ended(o, 'intent/not_ready', 1)],
    [changed(over, o => { o.feedback[1] = overRefusal; }), o => checks.endsWith(o, 2, guidance + unchanged)],
    [changed(over, o => { o.feedback[0] = overRefusal.replace('526213', '526214'); }), o => checks.startsWith(o, 1, 'Scope check: the Intent is estimated at 526213 bytes of one review (scenarios 2, acceptance harnesses 2, named files 0 in areas 0), over the 524288-byte bound of one review (tools.InputBudgetBytes).\n')],
    [changed(over, o => { o.feedback[0] = overRefusal.replace(guidance, 'Narrow the Intent.'); }), o => checks.endsWith(o, 1, guidance)],
    [{...tests, status: 1, output: fail}, r => checks.scopeTestsPassed(r)],
    [{...tests, output: fail.replace(/^FAIL\tgithub.*$/m, 'ok  \tgithub.com/KogenAI/kogen-shaping/shape\t1.008s')}, r => checks.scopeTestsPassed(r)],
    [{...tests, output: 'testing: warning: no tests to run\nPASS\nok  \tgithub.com/KogenAI/kogen-shaping/shape\t0.412s [no tests to run]\n'}, r => checks.scopeTestsPassed(r)],
    [{...tests, status: 2, output: 'shape/scope_test.go:9:2: undefined: scopeLines\nFAIL\tgithub.com/KogenAI/kogen-shaping/shape [build failed]\n'}, r => checks.scopeTestsPassed(r)],
  ];
  const passes = (o, check) => { try { check(o); return true; } catch { return false; } };
  const accepted = kind === 'good' ? good.every(([o, check]) => passes(o, check)) : bad.some(([o, check]) => passes(o, check));
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'k-shape-scope-gate-counts-package: ';
  When(P + 'the {string} case is shaped', {timeout: 720000}, function (name) { this.kgp = shapeCase(name); });
  When(P + 'the delivered scope tests are run', {timeout: 300000}, function () { this.kgpTests = scopeTests(); });
  Then(P + 'the run used a review bound of {int} bytes', function (n) { checks.bound(this.kgp, n); });
  Then(P + 'the Intent names the tracked file data\\/big.json, which is larger than the bound', function () { checks.bigNamed(this.kgp); });
  Then(P + 'the scope check refused {int} submission(s)', function (n) { checks.scopeRefusals(this.kgp, n); });
  Then(P + 'the decision check ran {int} time(s)', function (n) { checks.decisions(this.kgp, n); });
  Then(P + 'the decision check ran at least {int} time(s)', function (n) { checks.decisionsAtLeast(this.kgp, n); });
  Then(P + 'the result of submission {int} is:', function (n, text) { checks.is(this.kgp, n, text); });
  Then(P + 'the result of submission {int} contains {string}', function (n, text) { checks.contains(this.kgp, n, text); });
  Then(P + 'the result of submission {int} does not contain {string}', function (n, text) { checks.lacks(this.kgp, n, text); });
  Then(P + 'the result of submission {int} starts with:', function (n, text) { checks.startsWith(this.kgp, n, text); });
  Then(P + 'the result of submission {int} ends with:', function (n, text) { checks.endsWith(this.kgp, n, text); });
  Then(P + 'the Intent is published', function () { checks.published(this.kgp); });
  Then(P + 'shaping ended with {word} and exit {int}', function (reason, exit) { checks.ended(this.kgp, reason, exit); });
  Then(P + 'the delivered scope tests passed', function () { checks.scopeTestsPassed(this.kgpTests); });
}

// The real output of `go test -count=1 -v -run '^TestScope' ./shape` on the unchanged tree, the
// known-good control of the scope tests judgment (tabs written as \t).
// GO-TEST-OUTPUT === RUN   TestScopeBoundIsTheReviewStagesSetting
// GO-TEST-OUTPUT --- PASS: TestScopeBoundIsTheReviewStagesSetting (0.00s)
// GO-TEST-OUTPUT === RUN   TestScopeEstimateAtTheBoundFitsAndOneByteMoreIsRefused
// GO-TEST-OUTPUT --- PASS: TestScopeEstimateAtTheBoundFitsAndOneByteMoreIsRefused (0.00s)
// GO-TEST-OUTPUT === RUN   TestScopeNamedFilesAreCountedByTheirRules
// GO-TEST-OUTPUT --- PASS: TestScopeNamedFilesAreCountedByTheirRules (0.00s)
// GO-TEST-OUTPUT === RUN   TestScopeAreasAreDistinctDirectories
// GO-TEST-OUTPUT --- PASS: TestScopeAreasAreDistinctDirectories (0.00s)
// GO-TEST-OUTPUT === RUN   TestScopeStageUnits
// GO-TEST-OUTPUT --- PASS: TestScopeStageUnits (0.00s)
// GO-TEST-OUTPUT === RUN   TestScopePackageCountsTheDeclaredModelAndOutlineRows
// GO-TEST-OUTPUT === PAUSE TestScopePackageCountsTheDeclaredModelAndOutlineRows
// GO-TEST-OUTPUT === RUN   TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady
// GO-TEST-OUTPUT === PAUSE TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady
// GO-TEST-OUTPUT === RUN   TestScopedStepPath
// GO-TEST-OUTPUT --- PASS: TestScopedStepPath (0.00s)
// GO-TEST-OUTPUT === RUN   TestScopedStepPathHelperDir
// GO-TEST-OUTPUT --- PASS: TestScopedStepPathHelperDir (0.00s)
// GO-TEST-OUTPUT === CONT  TestScopePackageCountsTheDeclaredModelAndOutlineRows
// GO-TEST-OUTPUT === CONT  TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady
// GO-TEST-OUTPUT --- PASS: TestScopePackageCountsTheDeclaredModelAndOutlineRows (0.15s)
// GO-TEST-OUTPUT --- PASS: TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady (0.65s)
// GO-TEST-OUTPUT PASS
// GO-TEST-OUTPUT ok  \tgithub.com/KogenAI/kogen-shaping/shape\t1.008s
