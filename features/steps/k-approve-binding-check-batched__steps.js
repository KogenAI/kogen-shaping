'use strict';
// Acceptance of k-approve-binding-check-batched. The When step runs one synthetic case of the request:
// it compiles the shape package's tests with the harness of k-approve-binding-check-batched__harness.js
// (through a Go overlay), runs TestKApproveBindingCheckBatchedAcceptance for the case and reads what
// Preflight and Approve showed, with the Node invocations each started. The Then steps judge that.
// Each Preflight or Approve call may start at most 40 Node invocations (the harness's budget); a call
// past it is cancelled and judged as such, so the whole run stays within the 120-second acceptance limit.
// Run this file directly with good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKApproveBindingCheckBatchedAcceptance';
let built;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-approve-binding-check-batched-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_approve_binding_check_batched_acceptance_test.go');
  fs.writeFileSync(source, require('./k-approve-binding-check-batched__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_approve_binding_check_batched_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 100000});
  built = r.status === 0 && !r.error ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// runCase runs the harness for one case and returns its observation.
function runCase(name) {
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  fs.rmSync(out, {force: true});
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=100s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KABC_CASE: name, KABC_OUT: out}), encoding: 'utf8', timeout: 105000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const ran = `${b.bin} ${args.join(' ')} (KABC_CASE=${name})`;
  if (r.error || r.status !== 0) throw new Error(`${TEST} case ${name} failed: ${ran} exit=${r.status} error=${r.error || ''}\n${text}`);
  if (!text.includes('--- PASS: ' + TEST)) throw new Error(`${TEST} case ${name} did not run (absent, skipped or zero runs): ${ran} exit=${r.status}\n${text}`);
  if (!fs.existsSync(out)) throw new Error(`${TEST} case ${name} wrote no observation to ${out}: ${ran}\n${text}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function brief(obs) {
  return JSON.stringify(obs, (k, v) => (k === 'output' && typeof v === 'string' && v.length > 300 ? '…' + v.slice(-300) : v));
}

function at(obs, what) { return `${TEST} case ${obs.case}: ${what}; observation ${brief(obs)}`; }

function intent(obs, slug) {
  const it = obs.intents && obs.intents[slug];
  assert.ok(it, at(obs, `no observation of Intent ${slug}`));
  return it;
}

// withinBudget refuses a call whose Node invocations went past the harness's budget of 40 per call:
// the harness then refused the further invocations and cancelled the call, so it has no verdict.
function withinBudget(obs, slug, name, call) {
  assert.ok(call && call.over_budget === false, at(obs, `${name} of ${slug} started more than the harness budget of 40 Node invocations (${call && call.nodes} logged) and was cancelled, so its verdict is not observed; the step-ownership check must start at most one loader per step-file pair`));
}

// The judgments of the Then steps.
const judge = {
  approved(obs, slug) {
    const it = intent(obs, slug);
    withinBudget(obs, slug, 'the preview', it.preview);
    if (it.approve) withinBudget(obs, slug, 'the approval with the digest', it.approve);
    assert.strictEqual(it.preview.error, '', at(obs, `the preview of ${slug} must not be refused`));
    assert.strictEqual(it.preview.exit, 5, at(obs, `the preview of ${slug} prints the review card (exit 5)`));
    assert.ok(it.approve, at(obs, `the review card of ${slug} names its digest, and approve runs with it`));
    assert.strictEqual(it.approve.error, '', at(obs, `approving ${slug} with the preview's digest must not be refused`));
    assert.strictEqual(it.approve.exit, 0, at(obs, `approving ${slug} with the preview's digest exits 0`));
    assert.strictEqual(it.approved, true, at(obs, `${slug} has an approval file`));
  },
  previewLoaders(obs, slug, max) {
    const it = intent(obs, slug);
    const alone = obs.alone_preview;
    withinBudget(obs, slug, 'the preview alone', alone);
    withinBudget(obs, slug, 'the preview', it.preview);
    assert.ok(alone && alone.exit === 5 && alone.error === '', at(obs, `the preview of ${slug} alone, which counts the Node invocations outside the check, must reach the review card`));
    const n = it.preview.nodes - alone.nodes;
    assert.ok(n >= 1 && n <= max, at(obs, `the preview's step-ownership check of ${slug} started ${n} Node loaders (preview ${it.preview.nodes} minus ${alone.nodes} of ${slug} alone); want 1 to ${max}`));
  },
  preflightLoaders(obs, slug, max) {
    const p = intent(obs, slug).preflight;
    assert.ok(p, at(obs, `preflight of ${slug} did not run`));
    withinBudget(obs, slug, 'preflight', p);
    assert.strictEqual(p.error, '', at(obs, `preflight of ${slug} failed to observe`));
    assert.strictEqual(p.package, 'passed', at(obs, `preflight's package check of ${slug}`));
    assert.deepStrictEqual(p.ownership_blockers, [], at(obs, `preflight of ${slug} reports no step-ownership blocker`));
    assert.ok(p.nodes >= 1 && p.nodes <= max, at(obs, `preflight's step-ownership check of ${slug} started ${p.nodes} Node loaders; want 1 to ${max}`));
  },
  refused(obs, slug, rendered) {
    const it = intent(obs, slug);
    withinBudget(obs, slug, 'the preview', it.preview);
    assert.strictEqual(it.preview.exit, 1, at(obs, `the preview of ${slug} is refused with exit 1`));
    assert.strictEqual(it.preview.render, rendered + '\n', at(obs, `the rendered refusal of ${slug}`));
    assert.deepStrictEqual(it.preview.lines, rendered.split('\n').slice(1).map(l => l.replace(/^ {2}/, '')), at(obs, `the files the refusal of ${slug} lists`));
    assert.strictEqual(it.approve, null, at(obs, `a refused preview names no digest to approve ${slug} with`));
    assert.strictEqual(it.approved, false, at(obs, `${slug} has no approval file`));
  },
  preflightBlocked(obs, slug, blocker) {
    const p = intent(obs, slug).preflight;
    assert.ok(p, at(obs, `preflight of ${slug} did not run`));
    withinBudget(obs, slug, 'preflight', p);
    assert.strictEqual(p.error, '', at(obs, `preflight of ${slug} failed to observe`));
    assert.strictEqual(p.package, 'blocked', at(obs, `preflight's package check of ${slug}`));
    assert.deepStrictEqual(p.ownership_blockers, [blocker], at(obs, `preflight's step-ownership blockers of ${slug}`));
  },
};

if (require.main === module) {
  // Known observations: what a correct implementation shows for each case, and wrong variants
  // each judgment must reject. The release's own counts (600 loaders for thirty Intents, 20 for
  // one pair) are among the wrong variants.
  const err = (other) => `intent/step_ownership_collision: Intents syn-new and ${other} have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/${other}__steps.js\n  features/steps/syn-new__steps.js\n  features/steps/${other}__steps.js`;
  const blocker = (other) => ({class: 'intent', reason: 'step_ownership_collision', detail: `Intents syn-new and ${other} have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/${other}__steps.js`, paths: ['features/steps/syn-new__steps.js', `features/steps/${other}__steps.js`]});
  const card = (slug) => `Approve with:\n  kogen intent approve ${slug} ${'a'.repeat(64)}\n`;
  const call = (exit, nodes, output) => ({exit, nodes, over_budget: false, output, error: '', render: '', lines: []});
  const refusal = (other, nodes) => ({exit: 1, nodes, over_budget: false, output: '', error: err(other).split('\n')[0], render: err(other) + '\n', lines: [`features/steps/syn-new__steps.js`, `features/steps/${other}__steps.js`]});
  const passedPreflight = (nodes) => ({exit: 1, nodes, over_budget: false, error: '', package: 'passed', ownership_blockers: []});
  const approvedIntent = (slug, preflightNodes, previewNodes) => ({preflight: passedPreflight(preflightNodes), preview: call(5, previewNodes, card(slug)), approve: call(0, preflightNodes, `approved ${slug} aaaaaaaa\n`), approved: true});
  const refusedIntent = (other, nodes) => ({preflight: {exit: 1, nodes, over_budget: false, error: '', package: 'blocked', ownership_blockers: [blocker(other)]}, preview: refusal(other, nodes), approve: null, approved: false});
  const good = {
    pairs: {case: 'pairs', alone_preview: call(5, 2, card('syn-new')), intents: {'syn-new': approvedIntent('syn-new', 30, 32)}},
    one: {case: 'one-collision', intents: {'syn-new': refusedIntent('syn-17', 17)}},
    two: {case: 'two-collisions', intents: {'syn-new': refusedIntent('syn-03', 3)}},
    load: {case: 'load-failure', alone_preview: call(5, 2, card('syn-new')), intents: {'syn-new': {...approvedIntent('syn-new', 30, 32), preflight: undefined}}},
    worktrees: {case: 'worktrees', intents: {'syn-alpha': approvedIntent('syn-alpha', 1, 3), 'syn-beta': approvedIntent('syn-beta', 1, 3)}},
  };
  const runs = [
    o => judge.approved(o.pairs, 'syn-new'), o => judge.previewLoaders(o.pairs, 'syn-new', 30), o => judge.preflightLoaders(o.pairs, 'syn-new', 30),
    o => judge.refused(o.one, 'syn-new', err('syn-17')), o => judge.preflightBlocked(o.one, 'syn-new', blocker('syn-17')),
    o => judge.refused(o.two, 'syn-new', err('syn-03')), o => judge.preflightBlocked(o.two, 'syn-new', blocker('syn-03')),
    o => judge.approved(o.load, 'syn-new'), o => judge.previewLoaders(o.load, 'syn-new', 30),
    o => judge.approved(o.worktrees, 'syn-alpha'), o => judge.preflightLoaders(o.worktrees, 'syn-alpha', 1),
    o => judge.approved(o.worktrees, 'syn-beta'), o => judge.preflightLoaders(o.worktrees, 'syn-beta', 1),
  ];
  const all = o => { for (const run of runs) run(o); };
  const set = (key, slug, part, change) => {
    const o = structuredClone(good);
    const it = o[key].intents[slug];
    it[part] = change === null ? null : {...it[part], ...change};
    return o;
  };
  const with_ = (key, change) => ({...structuredClone(good), [key]: {...structuredClone(good[key]), ...change}});
  const cancelled = {exit: 1, nodes: 41, over_budget: true, output: '', error: 'context canceled', render: '', lines: []};
  const bad = {
    'release preview cancelled past the budget': set('pairs', 'syn-new', 'preview', cancelled),
    'release preflight cancelled past the budget': set('pairs', 'syn-new', 'preflight', {nodes: 42, over_budget: true, error: 'context canceled'}),
    'over budget but otherwise approved': set('pairs', 'syn-new', 'approve', {over_budget: true}),
    'release collision preview cancelled past the budget': set('one', 'syn-new', 'preview', cancelled),
    'over budget but otherwise refused': set('two', 'syn-new', 'preview', {over_budget: true}),
    'over budget preflight but otherwise blocked': set('two', 'syn-new', 'preflight', {over_budget: true}),
    'alone preview over budget': with_('pairs', {alone_preview: {...call(5, 41, card('syn-new')), over_budget: true}}),
    'release loader count in the preview': set('pairs', 'syn-new', 'preview', {nodes: 602}),
    'one loader too many in the preview': set('pairs', 'syn-new', 'preview', {nodes: 33}),
    'no loader in the preview': set('pairs', 'syn-new', 'preview', {nodes: 2}),
    'alone preview did not reach the card': with_('pairs', {alone_preview: {...call(1, 2, ''), error: 'intent/not_ready: the Intent is not ready to build'}}),
    'release loader count in preflight': set('pairs', 'syn-new', 'preflight', {nodes: 600}),
    'no loader in preflight': set('pairs', 'syn-new', 'preflight', {nodes: 0}),
    'preflight package blocked': set('pairs', 'syn-new', 'preflight', {package: 'blocked'}),
    'preflight ownership blocker without collision': set('pairs', 'syn-new', 'preflight', {ownership_blockers: [blocker('syn-17')]}),
    'preview refused': set('pairs', 'syn-new', 'preview', refusal('syn-17', 17)),
    'no digest approval': set('pairs', 'syn-new', 'approve', null),
    'digest approval refused': set('pairs', 'syn-new', 'approve', {exit: 1, error: 'intent/changed: package changed'}),
    'no approval file': set('pairs', 'syn-new', 'approved', null),
    'syn-17 collision missed': set('one', 'syn-new', 'preview', call(5, 30, card('syn-new'))),
    'collision names the wrong Intent': set('one', 'syn-new', 'preview', refusal('syn-16', 16)),
    'collision text reworded': set('one', 'syn-new', 'preview', {render: err('syn-17').replace('overlapping', 'colliding') + '\n'}),
    'collision files not listed': set('one', 'syn-new', 'preview', {lines: [], render: err('syn-17').split('\n')[0] + '\n'}),
    'collision files reversed': set('one', 'syn-new', 'preview', {lines: ['features/steps/syn-17__steps.js', 'features/steps/syn-new__steps.js']}),
    'refused but approved': set('one', 'syn-new', 'approve', call(0, 30, 'approved syn-new aaaaaaaa\n')),
    'preflight missed the collision': set('one', 'syn-new', 'preflight', passedPreflight(30)),
    'preflight names another pair': set('one', 'syn-new', 'preflight', {ownership_blockers: [blocker('syn-03')]}),
    'two collisions name the later Intent': set('two', 'syn-new', 'preview', refusal('syn-17', 17)),
    'preflight names the later of two collisions': set('two', 'syn-new', 'preflight', {ownership_blockers: [blocker('syn-17')]}),
    'a step file that throws on load causes a refusal': set('load', 'syn-new', 'preview', refusal('syn-09', 9)),
    'release loader count with a throwing file': set('load', 'syn-new', 'preview', {nodes: 602}),
    'second worktree Intent not approved': set('worktrees', 'syn-beta', 'approve', {exit: 1, error: 'intent/step_ownership_collision: Intents syn-beta and syn-alpha have overlapping step bindings'}),
    'release loader count for one pair': set('worktrees', 'syn-alpha', 'preflight', {nodes: 20}),
    'two loaders for one pair': set('worktrees', 'syn-beta', 'preflight', {nodes: 2}),
  };
  bad['no approval file'].pairs.intents['syn-new'].approved = false;
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
    console.error('usage: node features/steps/k-approve-binding-check-batched__steps.js good|bad');
    process.exitCode = 2;
  }
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'k-approve-binding-check-batched: ';
  When(P + 'the {string} case runs', {timeout: 115000}, function (name) {
    this.kabc = runCase(name);
  });
  Then(P + '{string} is approved after its preview and the approval with the preview\'s digest', function (slug) {
    judge.approved(this.kabc, slug);
  });
  Then(P + 'the preview\'s step-ownership check of {string} started between 1 and {int} Node loaders', function (slug, max) {
    judge.previewLoaders(this.kabc, slug, max);
  });
  Then(P + 'preflight\'s step-ownership check of {string} started between 1 and {int} Node loaders and its package check passed', function (slug, max) {
    judge.preflightLoaders(this.kabc, slug, max);
  });
  Then(P + 'the preview of {string} is refused with exit 1 and this rendered error:', function (slug, rendered) {
    judge.refused(this.kabc, slug, rendered);
  });
  Then(P + 'preflight of {string} reports exactly this step-ownership blocker:', function (slug, blocker) {
    judge.preflightBlocked(this.kabc, slug, JSON.parse(blocker));
  });
}
