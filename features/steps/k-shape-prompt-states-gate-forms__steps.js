'use strict';
// Acceptance of k-shape-prompt-states-gate-forms. The When step shapes one synthetic case: on
// first use it compiles the shape package's tests with the harness of
// k-shape-prompt-states-gate-forms__harness.js (through a Go overlay), runs
// TestKShapePromptStatesGateFormsAcceptance once, which shapes every case natively with a scripted
// shaper and a scripted decision-check model, and reads what each run showed. The Then steps judge
// that, the prompt goldens and the policy golden's recorded prompt. Run this file directly with
// good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const SLUG = 'k-shape-prompt-states-gate-forms';
const TEST = 'TestKShapePromptStatesGateFormsAcceptance';
const POLICY_GOLDEN = 'shape/testdata/shaper_policy_golden.json';
let observed;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// observe compiles the shape tests with the harness and runs every case, once per process.
function observe() {
  if (observed) return observed;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), SLUG + '-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_shape_prompt_states_gate_forms_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-prompt-states-gate-forms__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/k_shape_prompt_states_gate_forms_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  if (b.status !== 0) {
    observed = {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`};
    return observed;
  }
  const out = path.join(dir, 'observations.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=280s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({KSPG_OUT: out}), encoding: 'utf8', timeout: 300000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    observed = {error: `${TEST}: shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text.slice(-20000)}`};
    return observed;
  }
  observed = {cases: JSON.parse(fs.readFileSync(out, 'utf8'))};
  return observed;
}

function caseOf(name) {
  const o = observe();
  if (o.error) throw new Error(o.error);
  assert.ok(o.cases[name], `${TEST} recorded no case ${JSON.stringify(name)}; recorded: ${Object.keys(o.cases).join(', ')}`);
  return o.cases[name];
}

function shown(o) {
  return `case ${o.case} (request ${JSON.stringify(o.request)}): shape ended with exit ${o.exit}, err ${o.err}; published=${o.published}; ` +
    `${o.shaper} shaper request(s) and ${o.decision} decision check request(s)\nfeedback to the first submission:\n${o.first_feedback || '(none)'}\n` +
    `stderr (tail):\n${String(o.stderr).slice(-2000)}`;
}

function lines(text) {
  return String(text).split('\n');
}

// goldenRun runs the named golden tests of the shape package with go test -v.
function goldenRun(names) {
  const args = ['test', '-run', '^(' + names.join('|') + ')$', '-count=1', '-v', './shape'];
  const r = spawnSync('go', args, {env: env(), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  return {command: 'go ' + args.join(' '), status: r.status, error: String(r.error || ''), output: (r.stdout || '') + (r.stderr || '')};
}

const checks = {
  publishedFirst(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.equal(o.shaper, 1, `the Intent was not published from the first submission: ${o.shaper} shaper requests, want 1\n${shown(o)}`);
    assert.equal(o.intent, o.first_intent, `the published intent.md is not the first submission's intent.md\n${shown(o)}`);
  },
  decisionRequests(o, n) {
    assert.equal(o.decision, n, `${o.decision} decision check request(s), want ${n}\n${shown(o)}`);
  },
  noLineStarts(o, prefix) {
    assert.ok(!lines(o.first_feedback).some(l => l.startsWith(prefix)), `the feedback to the first submission has a line starting ${JSON.stringify(prefix)}\n${shown(o)}`);
  },
  refusedLine(o, headline, line) {
    assert.ok(lines(o.first_feedback).some(l => l.startsWith(headline)), `the first submission was not refused by ${JSON.stringify(headline)} (no line starts with it)\n${shown(o)}`);
    assert.ok(lines(o.first_feedback).includes(line), `no line of the first submission's feedback is exactly ${JSON.stringify(line)}\n${shown(o)}`);
  },
  refusedContaining(o, headline, a, b) {
    assert.ok(lines(o.first_feedback).some(l => l.startsWith(headline)), `the first submission was not refused by ${JSON.stringify(headline)} (no line starts with it)\n${shown(o)}`);
    assert.ok(lines(o.first_feedback).some(l => l.includes(a) && l.includes(b)), `no line of the first submission's feedback contains both ${JSON.stringify(a)} and ${JSON.stringify(b)}\n${shown(o)}`);
  },
  refusedStarting(o, headline, start, text) {
    assert.ok(lines(o.first_feedback).some(l => l.startsWith(headline)), `the first submission was not refused by ${JSON.stringify(headline)} (no line starts with it)\n${shown(o)}`);
    assert.ok(lines(o.first_feedback).some(l => l.startsWith(start) && l.includes(text)), `no line of the first submission's feedback starts with ${JSON.stringify(start)} and contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  written(o) {
    const files = o.files.filter(f => f.submitted !== null);
    assert.ok(files.length > 0, `case ${o.case} submitted no extra files`);
    for (const f of files) {
      assert.ok(f.after === f.submitted, `${f.path} holds ${f.after === null ? 'nothing (absent)' : f.after.length + ' different bytes'} after shaping, want the ${f.submitted.length} submitted bytes\n${shown(o)}`);
    }
  },
  unchanged(o) {
    assert.ok(o.files.length > 0, `case ${o.case} watched no files`);
    for (const f of o.files) {
      assert.ok(f.after === f.before, `${f.path} changed during shaping: before ${f.before === null ? 'absent' : f.before.length + ' bytes'}, after ${f.after === null ? 'absent' : f.after.length + ' bytes'}\n${shown(o)}`);
    }
  },
  instructionsContain(o, literals) {
    assert.ok(o.shaper >= 1 && o.instructions !== '', `no shaper request was made\n${shown(o)}`);
    const missing = literals.filter(l => !o.instructions.includes(l));
    assert.deepEqual(missing, [], `the instructions of the first shaper request lack ${missing.map(l => JSON.stringify(l)).join(', ')}`);
  },
  instructionsLack(o, literal) {
    assert.ok(o.shaper >= 1 && o.instructions !== '', `no shaper request was made\n${shown(o)}`);
    assert.ok(!o.instructions.includes(literal), `the instructions of the first shaper request still contain ${JSON.stringify(literal)}`);
  },
  goldensPassed(run, names) {
    const missing = names.filter(n => !run.output.includes('--- PASS: ' + n + ' ('));
    assert.ok(run.status === 0 && missing.length === 0 && !run.output.includes('--- FAIL'),
      `${run.command} exit=${run.status} error=${run.error}; tests without a "--- PASS: <name> (" line (absent, failed or skipped): ${missing.join(', ') || 'none'}\n${run.output.slice(-6000)}`);
  },
  goldenPrompt(text, literals) {
    let prompt;
    try { prompt = JSON.parse(text).prompt; } catch (e) { assert.fail(`${POLICY_GOLDEN} is not JSON: ${e.message}`); }
    assert.equal(typeof prompt, 'string', `${POLICY_GOLDEN} has no string field "prompt"`);
    const missing = literals.filter(l => !prompt.includes(l));
    assert.deepEqual(missing, [], `the prompt recorded in ${POLICY_GOLDEN} lacks ${missing.map(l => JSON.stringify(l)).join(', ')}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one. The unchanged-tree feedback lines are those the harness
// observed on the unchanged tree.
function controls(kind) {
  const DC = 'Decision check: 1 stated decisions are not supported by the request; follow the request, cite the repository evidence that decides it with (evidence: ...), or mark the decision as an assumption the user must confirm:';
  const DE = 'Decision evidence: 1 problem(s) with what the Decisions cite; nothing was published.';
  const IS = 'invalid submission:';
  const ONLY = ': shaping may write only .kogen/intents/is-even/{intent.md,acceptance.feature,model.qnt,controls.json} and step files under features/steps/';
  const intent = '# Add an is-even check\n\nSlug: is-even\n';
  const base = (name, extra) => ({case: name, request: 'Add calc.is_even(n) that returns True for even integers.', first_intent: intent, exit: 0, err: '<nil>', stderr: '',
    published: true, intent, shaper: 2, decision: 1, instructions: 'shaper prompt', first_feedback: '', files: [], ...extra});
  const literals = ['(evidence: <tracked path>)', 'Step sentences are Cucumber Expressions'];
  const good = {
    prompt: base('prompt', {shaper: 1, instructions: 'You are the shaper. (evidence: <tracked path>) Step sentences are Cucumber Expressions: (text) is optional text.'}),
    marked: base('decision-marker-on-line', {shaper: 1}),
    nextLine: base('decision-marker-next-line', {decision: 2, first_feedback: `${DC}\nMark an assumption.\nstatement: Odd numbers return False | request_quotes:  | verdict: unsupported | why: synthetic scripted item`}),
    prose: base('evidence-path-then-prose', {shaper: 1, decision: 1}),
    see: base('evidence-see', {first_feedback: `${DE} A Decision's (evidence: ...) names files.\n- decision 1: see src/calc.py is not a file or directory tracked at HEAD`}),
    fixture: base('write-fixture', {shaper: 1, files: [{path: '.kogen/intents/is-even/fixtures/input.txt', submitted: '4\n', before: null, after: '4\n'}]}),
    product: base('write-product', {first_feedback: `${IS}\ncalc.js: shaping may write only files under .kogen/intents/is-even/ and step files under features/steps/\nNo proposals from this submission were retained; earlier proposals are unchanged.`,
      files: [{path: 'calc.js', submitted: 'function add(a, b) { return a - b }\n', before: 'function add(a, b) { return a + b }\n', after: 'function add(a, b) { return a + b }\n'}]}),
    oversize: base('write-oversize', {first_feedback: `${IS}\n.kogen/intents/is-even/fixtures/big.txt: a file under .kogen/intents/is-even/ may hold at most 65536 bytes\nNo proposals from this submission were retained; earlier proposals are unchanged.`,
      files: [{path: '.kogen/intents/is-even/fixtures/big.txt', submitted: 'x'.repeat(65537), before: null, after: null}]}),
  };
  const goldenNames = ['TestShaperPolicyGolden', 'TestPromptsAreByteEqualToTheElixirFunctions', 'TestShapeForceWireCompatibility'];
  const goldenOutput = '=== RUN   TestPromptsAreByteEqualToTheElixirFunctions\n--- PASS: TestPromptsAreByteEqualToTheElixirFunctions (0.00s)\n=== RUN   TestShaperPolicyGolden\n--- PASS: TestShaperPolicyGolden (0.00s)\n' +
    '--- PASS: TestShapeForceWireCompatibility (9.29s)\n    --- PASS: TestShapeForceWireCompatibility/responses/refine (2.15s)\nPASS\nok  \tgithub.com/KogenAI/kogen-shaping/shape\t12.008s\n';
  const goldenGood = {command: 'go test', status: 0, error: '', output: goldenOutput};
  const promptGolden = JSON.stringify({prompt: 'x (evidence: <tracked path>) Step sentences are Cucumber Expressions', tools: {}});
  const goodChecks = [
    () => checks.instructionsContain(good.prompt, literals),
    () => checks.instructionsLack(good.prompt, 'including what existing behavior is replaced or removed'),
    () => checks.publishedFirst(good.prompt),
    () => checks.publishedFirst(good.marked),
    () => checks.decisionRequests(good.marked, 1),
    () => checks.noLineStarts(good.marked, 'Decision check:'),
    () => checks.refusedContaining(good.nextLine, 'Decision check: ', 'statement: Odd numbers return False', 'verdict: unsupported'),
    () => checks.publishedFirst(good.prose),
    () => checks.noLineStarts(good.prose, 'Decision evidence:'),
    () => checks.refusedLine(good.see, 'Decision evidence: ', '- decision 1: see src/calc.py is not a file or directory tracked at HEAD'),
    () => checks.publishedFirst(good.fixture),
    () => checks.written(good.fixture),
    () => checks.refusedStarting(good.product, IS, 'calc.js: ', 'shaping may write only'),
    () => checks.unchanged(good.product),
    () => checks.refusedStarting(good.oversize, IS, '.kogen/intents/is-even/fixtures/big.txt: ', 'may hold at most 65536 bytes'),
    () => checks.unchanged(good.oversize),
    () => checks.goldensPassed(goldenGood, goldenNames),
    () => checks.goldenPrompt(promptGolden, literals),
  ];
  const changed = (o, change) => { const c = JSON.parse(JSON.stringify(o)); change(c); return c; };
  const badChecks = [
    // what the unchanged tree showed
    () => checks.instructionsContain(changed(good.prompt, o => { o.instructions = 'Old prompt <Short prose: the outcome and its boundaries, including what existing behavior is replaced or removed.>'; }), literals),
    () => checks.instructionsLack(changed(good.prompt, o => { o.instructions += ' including what existing behavior is replaced or removed'; }), 'including what existing behavior is replaced or removed'),
    () => checks.publishedFirst(changed(good.marked, o => { o.exit = 4; o.published = false; o.shaper = 2; o.intent = null; o.first_feedback = DC; })),
    () => checks.noLineStarts(changed(good.marked, o => { o.first_feedback = `${DC}\nstatement: Zero counts as even | request_quotes:  | verdict: unsupported | why: synthetic scripted item`; }), 'Decision check:'),
    () => checks.refusedContaining(changed(good.nextLine, o => { o.first_feedback = ''; o.shaper = 1; }), 'Decision check: ', 'statement: Odd numbers return False', 'verdict: unsupported'),
    () => checks.publishedFirst(changed(good.prose, o => { o.exit = 4; o.published = false; o.shaper = 2; o.intent = null; o.first_feedback = `${DE}\n- decision 1: tests/x_test.py and the check runs it is not a file or directory tracked at HEAD`; })),
    () => checks.noLineStarts(changed(good.prose, o => { o.first_feedback = `${DE}\n- decision 1: tests/x_test.py and the check runs it is not a file or directory tracked at HEAD`; }), 'Decision evidence:'),
    () => checks.refusedLine(changed(good.see, o => { o.first_feedback = `${DE}\n- decision 1: tests/x_test.py and src/new.py is not a file or directory tracked at HEAD`; }), 'Decision evidence: ', '- decision 1: src/new.py is not a file or directory tracked at HEAD'),
    () => checks.refusedLine(changed(good.see, o => { o.first_feedback = '- decision 1: see src/calc.py is not a file or directory tracked at HEAD'; }), 'Decision evidence: ', '- decision 1: see src/calc.py is not a file or directory tracked at HEAD'),
    () => checks.publishedFirst(changed(good.fixture, o => { o.exit = 4; o.published = false; o.shaper = 2; o.files[0].after = null; o.first_feedback = `${IS}\n.kogen/intents/is-even/fixtures/input.txt${ONLY}`; })),
    () => checks.written(changed(good.fixture, o => { o.files[0].after = null; })),
    () => checks.written(changed(good.fixture, o => { o.files[0].after = '5\n'; })),
    () => checks.written(changed(good.fixture, o => { o.files = []; })),
    () => checks.refusedStarting(changed(good.oversize, o => { o.first_feedback = `${IS}\n.kogen/intents/is-even/fixtures/big.txt${ONLY}\nNo proposals from this submission were retained; earlier proposals are unchanged.`; }), IS, '.kogen/intents/is-even/fixtures/big.txt: ', 'may hold at most 65536 bytes'),
    () => checks.refusedStarting(changed(good.product, o => { o.first_feedback = ''; }), IS, 'calc.js: ', 'shaping may write only'),
    () => checks.unchanged(changed(good.product, o => { o.files[0].after = o.files[0].submitted; })),
    () => checks.unchanged(changed(good.oversize, o => { o.files[0].after = o.files[0].submitted; })),
    () => checks.unchanged(changed(good.oversize, o => { o.files = []; })),
    () => checks.publishedFirst(changed(good.marked, o => { o.intent = '# Another intent\n'; })),
    () => checks.decisionRequests(changed(good.marked, o => { o.decision = 0; }), 1),
    () => checks.goldensPassed({...goldenGood, status: 1, output: goldenOutput.replace('--- PASS: TestShaperPolicyGolden', '--- FAIL: TestShaperPolicyGolden')}, goldenNames),
    () => checks.goldensPassed({...goldenGood, output: goldenOutput.replace('--- PASS: TestShapeForceWireCompatibility (9.29s)\n', '')}, goldenNames),
    () => checks.goldensPassed({...goldenGood, output: 'testing: warning: no tests to run\nPASS\nok  \tgithub.com/KogenAI/kogen-shaping/shape\t0.5s [no tests to run]\n'}, goldenNames),
    () => checks.goldenPrompt(JSON.stringify({prompt: 'old prompt', tools: {}}), literals),
    () => checks.goldenPrompt('not json', literals),
  ];
  const passes = check => { try { check(); return true; } catch { return false; } };
  const accepted = kind === 'good' ? goodChecks.every(passes) : badChecks.some(passes);
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

function column(table) {
  return table.raw().slice(1).map(row => row[0]);
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('k-shape-prompt-states-gate-forms: the {string} case is shaped', {timeout: 600000}, function (name) { this.kspg = caseOf(name); });
  Then('k-shape-prompt-states-gate-forms: the Intent is published from the first submission', function () { checks.publishedFirst(this.kspg); });
  Then('k-shape-prompt-states-gate-forms: the number of decision check requests is {int}', function (n) { checks.decisionRequests(this.kspg, n); });
  Then('k-shape-prompt-states-gate-forms: no line of the feedback to the first submission starts with {string}', function (prefix) { checks.noLineStarts(this.kspg, prefix); });
  Then('k-shape-prompt-states-gate-forms: the first submission is refused by {string} with the line {string}', function (headline, line) { checks.refusedLine(this.kspg, headline, line); });
  Then('k-shape-prompt-states-gate-forms: the first submission is refused by {string} with a line containing {string} and {string}', function (headline, a, b) { checks.refusedContaining(this.kspg, headline, a, b); });
  Then('k-shape-prompt-states-gate-forms: the first submission is refused by {string} with a line starting {string} that contains {string}', function (headline, start, text) { checks.refusedStarting(this.kspg, headline, start, text); });
  Then('k-shape-prompt-states-gate-forms: every submitted file under the Intent folder is written with its exact content', function () { checks.written(this.kspg); });
  Then('k-shape-prompt-states-gate-forms: every refused file is unchanged by shaping', function () { checks.unchanged(this.kspg); });
  Then('k-shape-prompt-states-gate-forms: the instructions of the first shaper request contain these literals', function (table) { checks.instructionsContain(this.kspg, column(table)); });
  Then('k-shape-prompt-states-gate-forms: the instructions of the first shaper request do not contain {string}', function (literal) { checks.instructionsLack(this.kspg, literal); });
  Then('k-shape-prompt-states-gate-forms: these prompt golden tests of the shape package pass', {timeout: 300000}, function (table) { const names = column(table); checks.goldensPassed(goldenRun(names), names); });
  Then('k-shape-prompt-states-gate-forms: the prompt recorded in the policy golden contains these literals', function (table) { checks.goldenPrompt(fs.readFileSync(POLICY_GOLDEN, 'utf8'), column(table)); });
}
