'use strict';
// Acceptance of k-shape-size-for-one-review. A When step shapes one synthetic case of the request:
// it compiles the shape package's tests with the harness of k-shape-size-for-one-review__harness.js
// (through a Go overlay), runs TestKShapeSizeForOneReviewAcceptance for the case with a scripted
// shaper, and reads what the run showed. To change the review stage's bound it builds against a
// private copy of kogen-loop whose provider.InputBudgetBytes (tools.InputBudgetBytes) is the given
// value, through a private go.mod; nothing in the checkout is written. The Then steps judge what
// was seen. Run this file directly with good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeSizeForOneReviewAcceptance';
const HEADLINE = 'Scope check: ';
const SETTING = 'const InputBudgetBytes = 524288\n';
const builds = {};
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function go(args, what) {
  const r = spawnSync('go', args, {env: env(), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  if (r.error || r.status !== 0) throw new Error(`${TEST}: ${what}: go ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`);
  return r.stdout;
}

// build compiles the shape tests with the harness once per bound: as the checkout is when bound
// is 0, else against a copy of kogen-loop whose InputBudgetBytes is bound.
function build(bound) {
  if (builds[bound]) return builds[bound];
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-size-for-one-review-'));
  process.on('exit', () => {
    spawnSync('chmod', ['-R', 'u+w', dir]);
    fs.rmSync(dir, {recursive: true, force: true});
  });
  const source = path.join(dir, 'k_size_for_one_review_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-size-for-one-review__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_size_for_one_review_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin];
  if (bound) {
    const loop = path.join(dir, 'kogen-loop');
    const r = spawnSync('cp', ['-R', go(['list', '-m', '-f', '{{.Dir}}', 'kogen-loop'], 'locating kogen-loop').trim(), loop], {encoding: 'utf8'});
    if (r.status !== 0) throw new Error(`${TEST}: copying kogen-loop failed: ${r.stderr}`);
    spawnSync('chmod', ['-R', 'u+w', loop]);
    const file = path.join(loop, 'provider', 'respond.go');
    const text = fs.readFileSync(file, 'utf8');
    if (!text.includes(SETTING)) throw new Error(`${TEST}: kogen-loop's provider/respond.go has no line ${JSON.stringify(SETTING)}`);
    fs.writeFileSync(file, text.replace(SETTING, `const InputBudgetBytes = ${bound}\n`));
    const mod = fs.readFileSync('go.mod', 'utf8');
    if (!/^replace kogen-loop => .*$/m.test(mod)) throw new Error(`${TEST}: go.mod has no replace kogen-loop line`);
    fs.writeFileSync(path.join(dir, 'go.mod'), mod.replace(/^replace kogen-loop => .*$/m, 'replace kogen-loop => ' + loop));
    fs.copyFileSync('go.sum', path.join(dir, 'go.sum'));
    args.push('-modfile', path.join(dir, 'go.mod'));
  }
  args.push('./shape');
  go(args, bound ? `building the shape tests with tools.InputBudgetBytes ${bound}` : 'building the shape tests');
  builds[bound] = {bin, dir};
  return builds[bound];
}

// shapeCase runs the harness for one case once per bound and returns what the run showed.
function shapeCase(name, bound) {
  const key = name + '@' + bound;
  if (shaped[key]) return shaped[key];
  const b = build(bound);
  const out = path.join(b.dir, name + '-' + bound + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=400s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KSS_CASE: name, KSS_OUT: out}), encoding: 'utf8', timeout: 420000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KSS_CASE=${name}, tools.InputBudgetBytes ${bound || 'as built'}): shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[key] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[key];
}

function shown(o) {
  const results = Object.entries(o.results).map(([id, text]) => `result of submission ${id}:\n${text}`).join('\n----\n');
  return `case ${o.case} (tools.InputBudgetBytes ${o.bound}): shape ended with exit ${o.exit}, err ${o.err}; published=${o.published}; ` +
    `${o.shaper} shaper, ${o.decision} decision check and ${o.contract} contract check request(s)\n${results || '(no submission results)'}\n` +
    `not-ready lines:\n${o.remaining}\nprogress (tail):\n${String(o.progress).slice(-1500)}`;
}

function result(o, id) {
  const text = o.results[id];
  assert.equal(typeof text, 'string', `submission ${id} got no result back\n${shown(o)}`);
  return text;
}

function scenarioCount(feature) {
  return feature.split('\n').filter(line => /^\s*Scenario: /.test(line)).length;
}

const checks = {
  bound(o, n) {
    assert.equal(o.bound, n, `the run used tools.InputBudgetBytes ${o.bound}, want ${n}\n${shown(o)}`);
  },
  published(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
  },
  unpublished(o) {
    assert.equal(o.published, false, `the Intent was published\n${shown(o)}`);
  },
  scopeRefused(o, id) {
    assert.ok(result(o, id).startsWith(HEADLINE), `submission ${id} was not refused by the scope check (its result does not start with ${JSON.stringify(HEADLINE)})\n${shown(o)}`);
  },
  lineWith(o, id, parts) {
    assert.ok(result(o, id).split('\n').some(line => parts.every(p => line.includes(p))),
      `no line of the result of submission ${id} contains all of ${JSON.stringify(parts)}\n${shown(o)}`);
  },
  contains(o, id, text) {
    assert.ok(result(o, id).includes(text), `the result of submission ${id} does not contain ${JSON.stringify(text)}\n${shown(o)}`);
  },
  lacks(o, id, text) {
    assert.ok(!result(o, id).includes(text), `the result of submission ${id} contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  intentHas(o, text) {
    assert.ok(o.intent.includes(text), `the published intent.md does not contain ${JSON.stringify(text)}:\n${o.intent}\n${shown(o)}`);
  },
  intentLacks(o, text) {
    assert.ok(o.published && !o.intent.includes(text), `the Intent is not published, or its intent.md contains ${JSON.stringify(text)}:\n${o.intent}\n${shown(o)}`);
  },
  scenarios(o, n) {
    assert.equal(scenarioCount(o.feature), n, `the published acceptance.feature has ${scenarioCount(o.feature)} scenarios, want ${n}\n${shown(o)}`);
  },
  featureHas(o, text) {
    assert.ok(o.feature.includes(text), `the published acceptance.feature does not contain ${JSON.stringify(text)}\n${shown(o)}`);
  },
  requests(o, shaper, decision, contract) {
    assert.deepEqual({shaper: o.shaper, decision: o.decision, contract: o.contract}, {shaper, decision, contract}, `model requests differ\n${shown(o)}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const feature40 = 'Feature: Synthetic behaviours\n' + Array.from({length: 40}, (_, i) => `  Scenario: area behaviour ${i + 1}\n    Then is-even: area behaviour ${i + 1} is built\n`).join('');
  const ack = 'Inspect whether fixtures, inputs and commands reached the intended behavior, using the existing read/probe tools when needed.\n\nReadiness observations\nScenario: alpha behaviour 1 (acceptance.feature line 2)';
  const refusal = 'Scope check: the Intent is estimated at 579884 bytes of one review (scenarios 40, acceptance harnesses 1, named files 12 in areas 12), over the 524288-byte bound of one review (tools.InputBudgetBytes).\n' +
    '- .kogen/intents/is-even/intent.md: 1112 bytes (Intent package)\n- alpha/module.js: 48000 bytes (named file, area alpha)\n' +
    'Above the bound, split the work into an ordered ## Stages plan of at least two stages, each with a real gate in its checks and each inside the bound, or narrow the Intent to the requested behaviour. ' +
    'Do not satisfy the bound by deleting requested behaviour: every behaviour and example the request asks for keeps its scenario. Resubmitting this candidate unchanged is refused again.';
  const known = {case: 'wide', bound: 524288, exit: 0, err: '<nil>', published: true, shaper: 3, decision: 2, contract: 1, remaining: '', progress: '',
    results: {c1: refusal, c2: ack}, intent: '# Build the synthetic behaviours\n\n## Stages\n```json\n{}\n```\n', feature: feature40};
  const line = ['Scope check: the Intent is estimated at', '(scenarios 40, acceptance harnesses 1, named files 12 in areas 12)', 'over the 524288-byte bound of one review (tools.InputBudgetBytes)'];
  const good = [
    o => checks.bound(o, 524288),
    o => checks.scopeRefused(o, 'c1'),
    o => checks.lineWith(o, 'c1', line),
    o => checks.contains(o, 'c1', '- alpha/module.js: 48000 bytes (named file, area alpha)'),
    o => checks.lacks(o, 'c2', 'Scope check'),
    o => checks.published(o),
    o => checks.intentHas(o, '## Stages'),
    o => checks.scenarios(o, 40),
    o => checks.featureHas(o, 'Scenario: area behaviour 36'),
    o => checks.requests(o, 3, 2, 1),
  ];
  const changed = change => {
    const o = JSON.parse(JSON.stringify(known));
    change(o);
    return o;
  };
  const bad = [
    // The unchanged tree: the over-bound candidate is published as before, with its readiness observations.
    [changed(o => { o.results.c1 = ack; o.shaper = 3; o.decision = 3; o.contract = 2; }), o => checks.scopeRefused(o, 'c1')],
    [changed(o => { o.results.c1 = ack; }), o => checks.lineWith(o, 'c1', line)],
    [changed(o => { o.decision = 3; o.contract = 2; }), o => checks.requests(o, 3, 2, 1)],
    [changed(o => { o.results.c1 = 'invalid submission:\n' + refusal; }), o => checks.scopeRefused(o, 'c1')],
    [changed(o => { o.results.c1 = refusal.replace('named files 12 in areas 12', 'named files 12 in areas 1'); }), o => checks.lineWith(o, 'c1', line)],
    [changed(o => { o.results.c1 = refusal.replace('(scenarios 40, acceptance harnesses 1, named files 12 in areas 12), over', '(scenarios 40, acceptance harnesses 1, named files 12 in areas 12).\nOver'); }), o => checks.lineWith(o, 'c1', line)],
    [changed(o => { o.results.c1 = refusal.replace('524288-byte', '500000-byte'); }), o => checks.lineWith(o, 'c1', line)],
    [changed(o => { o.results.c1 = refusal.replace('48000 bytes', '4800 bytes'); }), o => checks.contains(o, 'c1', '- alpha/module.js: 48000 bytes (named file, area alpha)')],
    [changed(o => { delete o.results.c2; }), o => checks.lacks(o, 'c2', 'Scope check')],
    [changed(o => { o.results.c2 = refusal; }), o => checks.lacks(o, 'c2', 'Scope check')],
    [changed(o => { o.published = false; }), o => checks.published(o)],
    [changed(o => { o.bound = 1048576; }), o => checks.bound(o, 524288)],
    [changed(o => { o.intent = '# Build the synthetic behaviours\n'; }), o => checks.intentHas(o, '## Stages')],
    [changed(o => { o.published = false; o.intent = ''; }), o => checks.intentLacks(o, '## Stages')],
    [changed(o => { o.feature = feature40.replace(/  Scenario: area behaviour 36\n.*\n/, ''); }), o => checks.scenarios(o, 40)],
    [changed(o => { o.feature = feature40.replace('area behaviour 36', 'area behaviour 360'); }), o => checks.featureHas(o, 'Scenario: area behaviour 36\n')],
  ];
  const passes = (o, check) => { try { check(o); return true; } catch { return false; } };
  const accepted = kind === 'good' ? good.every(check => passes(known, check)) : bad.some(([o, check]) => passes(o, check));
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'k-shape-size-for-one-review: ';
  When(P + 'the {string} case is shaped', {timeout: 720000}, function (name) { this.kss = shapeCase(name, 0); });
  When(P + 'the {string} case is shaped with tools.InputBudgetBytes set to {int}', {timeout: 720000}, function (name, bound) { this.kss = shapeCase(name, bound); });
  Then(P + 'the run used a review bound of {int} bytes', function (n) { checks.bound(this.kss, n); });
  Then(P + 'the Intent is published', function () { checks.published(this.kss); });
  Then(P + 'the Intent is not published', function () { checks.unpublished(this.kss); });
  Then(P + 'submission {string} is refused by the scope check', function (id) { checks.scopeRefused(this.kss, id); });
  Then(P + 'a line of the result of submission {string} contains {string}, {string} and {string}', function (id, a, b, c) { checks.lineWith(this.kss, id, [a, b, c]); });
  Then(P + 'the result of submission {string} contains {string}', function (id, text) { checks.contains(this.kss, id, text); });
  Then(P + 'the result of submission {string} does not contain {string}', function (id, text) { checks.lacks(this.kss, id, text); });
  Then(P + 'the published intent.md contains {string}', function (text) { checks.intentHas(this.kss, text); });
  Then(P + 'the published intent.md does not contain {string}', function (text) { checks.intentLacks(this.kss, text); });
  Then(P + 'the published acceptance.feature has {int} scenarios', function (n) { checks.scenarios(this.kss, n); });
  Then(P + 'the published acceptance.feature contains {string}', function (text) { checks.featureHas(this.kss, text); });
  Then(P + 'the run made {int} shaper, {int} decision check and {int} contract check requests', function (shaper, decision, contract) { checks.requests(this.kss, shaper, decision, contract); });
}
