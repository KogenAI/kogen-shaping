'use strict';
// Acceptance of shape-contract-refusal-fix-line. The When steps compile the shape package's tests
// with the harness of shape-contract-refusal-fix-line__harness.js (through a Go overlay), run
// TestShapeContractRefusalFixLineAcceptance once per acceptance run and read the JSON observation
// the harness wrote to SCRFL_OUT. The Then steps compare it with the feature's values. Run this
// file directly with --control good or --control bad to check the judgments on known observations.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestShapeContractRefusalFixLineAcceptance';
const TARGET = 'shape/shape_contract_refusal_fix_line_acceptance_test.go';
const FEATURE = '.kogen/intents/shape-contract-refusal-fix-line/acceptance.feature';
const P = 'shape-contract-refusal-fix-line: ';
const BOUND = 300;

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

// judgeRun is the observation of one harness run: the run exited 0, passed the harness test
// exactly once and wrote an observation with the four shaping runs.
function judgeRun(r) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  const obs = JSON.parse(r.json);
  for (const key of ['two_confirmed', 'corrected', 'refuted', 'long']) {
    assert.ok(obs[key] && typeof obs[key] === 'object', `the observation has no ${key} run:\n${r.json}`);
  }
  return obs;
}

// harness builds the shape tests with the harness and runs the harness test once.
function harness() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'scrfl-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-contract-refusal-fix-line__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 400000, maxBuffer: 32 * 1024 * 1024});
  if (b.status !== 0) {
    throw new Error(`building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}${b.signal ? ' (' + b.signal + ')' : ''}${b.error ? ' ' + b.error.message : ''}:\n${b.stdout}${b.stderr}`);
  }
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SCRFL_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  return judgeRun({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `SCRFL_OUT=${out} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`});
}

let cached = null;
function observation() {
  if (cached === null) cached = harness();
  return cached;
}

// ---- the judgments: each takes the observation of one shaping run ----

function text(r, key) {
  assert.equal(typeof r[key], 'string', `the observation has no ${key} text:\n${JSON.stringify(r, null, 1)}`);
  return r[key];
}

function list(r, key) {
  assert.ok(Array.isArray(r[key]) && r[key].every(s => typeof s === 'string'), `the observation has no ${key} list:\n${JSON.stringify(r, null, 1)}`);
  return r[key];
}

// at reports whether want occurs in lines as consecutive lines.
function at(lines, want) {
  for (let i = 0; i + want.length <= lines.length; i++) {
    if (want.every((l, k) => lines[i + k] === l)) return true;
  }
  return false;
}

const chars = s => [...s].length;

const J = {
  exits(r, n) {
    assert.equal(r.exit, n, `the run exited ${r.exit} (${r.error_reason}), want ${n}`);
  },
  firstRefusalIs(r, want) {
    assert.equal(text(r, 'feedback_1'), want, 'the refusal fed back after the first submission');
  },
  firstRefusalEnds(r, want) {
    assert.ok(text(r, 'feedback_1').endsWith('\n' + want), `the refusal fed back after the first submission does not end with the line ${want}:\n${r.feedback_1}`);
  },
  firstRefusalStarts(r, want) {
    assert.ok(text(r, 'feedback_1').startsWith(want), `the refusal fed back after the first submission does not start with ${want}:\n${r.feedback_1}`);
  },
  repeatIs(r, tail) {
    assert.equal(text(r, 'feedback_2'), text(r, 'feedback_1') + '\n' + tail, 'the refusal fed back after the unchanged resubmission');
  },
  saved(r, want) {
    const lines = want.split('\n');
    assert.deepEqual(list(r, 'record_lines'), lines, 'the contradictions record lines');
    assert.deepEqual(list(r, 'event_problems'), lines, 'the saved refusal event problems');
    assert.ok(at(list(r, 'error_lines'), lines), `the not-ready error lines do not contain the lines in order:\n${r.error_lines.join('\n')}`);
    assert.ok(at(list(r, 'approval_lines'), lines), `the approval refusal lines do not contain the lines in order:\n${r.approval_lines.join('\n')}`);
  },
  events(r, want) {
    assert.deepEqual(r.contract_reviews, want, 'the contract_review events');
  },
  published(r) {
    assert.equal(r.published, true, `the run did not publish the Intent (exit ${r.exit}, ${r.error_reason})`);
  },
  corrected(r) {
    const feature = text(r, 'feature');
    assert.ok(feature.includes('Given is-even: the number 6') && !feature.includes('Given is-even: the number 4'), `the saved acceptance.feature is not the corrected one:\n${feature}`);
  },
  notARefusal(r) {
    const second = text(r, 'feedback_2');
    assert.ok(!second.includes('Contract check:') && !second.includes('fix for'), `the feedback after the corrected submission is a contract refusal:\n${second}`);
  },
  noRecord(r) {
    assert.equal(r.record_exists, false, `a contradictions record was saved: ${JSON.stringify(r.record_lines)}`);
  },
  noRefusalEvent(r) {
    assert.equal(r.contract_refusal_events, 0, `${r.contract_refusal_events} contract_review refusal events were saved`);
  },
  notFed(r, part) {
    for (const key of ['feedback_1', 'feedback_2']) {
      assert.ok(!text(r, key).includes(part), `the shaper was given ${part} (${key}):\n${r[key]}`);
    }
  },
  bounded(r, lead) {
    const problems = list(r, 'event_problems');
    const record = list(r, 'record_lines');
    assert.equal(problems.length, 2, 'the number of saved refusal event problems');
    assert.equal(record.length, 2, 'the number of contradictions record lines');
    for (let i = 0; i < 2; i++) {
      assert.equal(chars(problems[i]), BOUND, `saved problem ${i + 1} length`);
      assert.equal(problems[i], [...record[i]].slice(0, BOUND).join(''), `saved problem ${i + 1} is not the first ${BOUND} characters of its record line`);
    }
    assert.ok(problems[1].startsWith(lead), `the saved fix problem does not begin with the fix lead:\n${problems[1]}`);
  },
  fullFix(r, end) {
    const record = list(r, 'record_lines');
    assert.equal(record.length, 2, 'the number of contradictions record lines');
    assert.ok(chars(record[1]) > BOUND && record[1].endsWith(end), `the record's fix line is not kept in full (${chars(record[1])} characters):\n${record[1]}`);
  },
};

// ---- the feature's own values, for the observer controls ----

const lines = () => fs.readFileSync(path.join(__dirname, '../..', FEATURE), 'utf8').split('\n');

// doc is the doc string of the step of the scenario whose text after the keyword is P + step.
function doc(scenario, step) {
  const all = lines();
  const start = all.findIndex(l => l.trim() === 'Scenario: ' + scenario);
  assert.ok(start >= 0, 'the feature has the scenario ' + scenario);
  let end = all.findIndex((l, i) => i > start && l.trim().startsWith('Scenario:'));
  if (end < 0) end = all.length;
  const at0 = all.findIndex((l, i) => i > start && i < end && l.trim().replace(/^(Given|When|Then|And) /, '') === P + step);
  assert.ok(at0 >= 0 && all[at0 + 1].trim() === '"""', `the scenario ${scenario} has the step ${step} with a doc string`);
  const indent = all[at0 + 1].indexOf('"""');
  const close = all.findIndex((l, i) => i > at0 + 1 && l.trim() === '"""');
  return all.slice(at0 + 2, close).map(l => l.slice(indent)).join('\n');
}

const S1 = "A confirmed contradiction's refusal names the expectation to change and the contract to follow";
const S2 = 'The saved not-ready details keep the quotes and the fix of each confirmed finding';
const S3 = 'A corrected candidate passes the contract review';
const S4 = 'Refuted and ungrounded findings refuse nothing';
const S5 = 'The refusal keeps its counts and its bounds with a long contract quote';
const FIRST = 'the refusal fed back after the first submission is:';
const REPEAT = 'the refusal fed back after the unchanged resubmission is the first refusal followed by:';
const SAVED = 'the saved not-ready details contain these lines in order:';
const EVENTS = 'the contract_review events are:';
const CUT = 'the saved refusal event problems are the first 300 characters of the record lines, and the fix problem begins:';
const FULL = 'the contradictions record keeps the full fix line, which ends with:';
const ENDS = 'the refusal fed back after the first submission ends with the line:';

// good is an observation the feature's values describe, assembled from the feature itself.
function good() {
  const first = doc(S1, FIRST);
  const saved = doc(S2, SAVED).split('\n');
  const fixes = first.split('\n').filter(l => l.startsWith('- fix for '));
  const fix1 = doc(S3, ENDS);
  const lead = doc(S5, CUT);
  const quote = doc(S5, FULL);
  const mark = 'Refusing check: contract_review';
  const longFix = lead + ' ' + 'x'.repeat(BOUND) + quote;
  const longItem = '- Even numbers | expectation: ' + 'y'.repeat(BOUND);
  const base = {exit: 0, error_reason: '', error_lines: [], contract_reviews: [], contract_refusal_events: 0, event_problems: [], record_exists: false, record_lines: [], published: true, feedback_1: '', feedback_2: ''};
  return {
    two_confirmed: {...base, exit: 1, error_reason: 'not_ready', published: false, contract_refusal_events: 3, record_exists: true,
      feedback_1: first, feedback_2: first + '\n' + doc(S1, REPEAT), record_lines: saved, event_problems: saved,
      error_lines: [mark, ...saved, 'Retained draft files:'], approval_lines: [...saved, 'Retry Shape on the retained draft.']},
    corrected: {...base, feedback_1: 'Contract check: 1 acceptance expectation(s) contradict\n- item\n' + fix1, feedback_2: 'Readiness observations\nScenario: Even numbers',
      contract_reviews: JSON.parse(doc(S3, EVENTS)), contract_refusal_events: 1, event_problems: ['- item', fix1],
      feature: 'Feature: Is even\n  Scenario: Even numbers\n    Given is-even: the number 6\n'},
    refuted: {...base, feedback_1: 'Readiness observations', feedback_2: '', contract_reviews: JSON.parse(doc(S4, EVENTS))},
    long: {...base, exit: 1, error_reason: 'not_ready', published: false, contract_refusal_events: 3, record_exists: true,
      contract_reviews: JSON.parse(doc(S5, EVENTS)), feedback_1: doc(S5, 'the refusal fed back after the first submission starts with:') + ' 1 more',
      record_lines: [longItem, longFix], event_problems: [[...longItem].slice(0, BOUND).join(''), [...longFix].slice(0, BOUND).join('')]},
    fixes,
  };
}

// judgeAll runs every judgment of the feature on observation o.
function judgeAll(o) {
  const a = o.two_confirmed;
  J.firstRefusalIs(a, doc(S1, FIRST));
  J.repeatIs(a, doc(S1, REPEAT));
  J.exits(a, 1);
  J.saved(a, doc(S2, SAVED));
  const b = o.corrected;
  J.firstRefusalEnds(b, doc(S3, ENDS));
  J.exits(b, 0);
  J.events(b, JSON.parse(doc(S3, EVENTS)));
  J.published(b);
  J.corrected(b);
  J.notARefusal(b);
  J.noRecord(b);
  const c = o.refuted;
  J.exits(c, 0);
  J.events(c, JSON.parse(doc(S4, EVENTS)));
  J.published(c);
  J.noRefusalEvent(c);
  J.noRecord(c);
  J.notFed(c, 'fix for');
  const d = o.long;
  J.events(d, JSON.parse(doc(S5, EVENTS)));
  J.firstRefusalStarts(d, doc(S5, 'the refusal fed back after the first submission starts with:'));
  J.bounded(d, doc(S5, CUT));
  J.fullFix(d, doc(S5, FULL));
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function control(which) {
  const o = good();
  if (which === 'good') {
    judgeAll(o);
    return 'accepted';
  }
  const copy = () => JSON.parse(JSON.stringify(o));
  const withRun = (key, change) => { const c = copy(); change(c[key]); return c; };
  const noFix = l => l.filter(x => !x.startsWith('- fix for '));
  // the harness run itself
  const run = extra => ({status: 0, signal: null, error: null, output: `--- PASS: ${TEST} (6.2s)\n`, json: JSON.stringify(o), command: 'recorded', ...extra});
  rejects(() => judgeRun(run({status: 1})), 'a failed run');
  rejects(() => judgeRun(run({error: 'spawnSync ETIMEDOUT'})), 'a run that did not finish');
  rejects(() => judgeRun(run({output: `--- FAIL: ${TEST} (6.2s)\n`})), 'a failed harness test');
  rejects(() => judgeRun(run({output: `--- PASS: ${TEST} (1s)\n--- PASS: ${TEST} (1s)\n`})), 'two harness runs');
  rejects(() => judgeRun(run({output: 'PASS\n'})), 'a run that ran no harness test');
  rejects(() => judgeRun(run({json: null})), 'a run without an observation');
  rejects(() => judgeRun(run({json: JSON.stringify({two_confirmed: {}})})), 'an observation without every run');
  judgeRun(run({}));
  // the unchanged tree: item lines only
  const unchanged = copy();
  unchanged.two_confirmed.feedback_1 = noFix(o.two_confirmed.feedback_1.split('\n')).join('\n');
  rejects(() => judgeAll(unchanged), "the unchanged tree's refusal without a fix line");
  const mutations = [
    ['a refusal without the fix lines', 'two_confirmed', r => { r.feedback_1 = noFix(r.feedback_1.split('\n')).join('\n'); }],
    ['a fix without the expectation quote', 'two_confirmed', r => { r.feedback_1 = r.feedback_1.replace('"Given is-even: the number 4" ', ''); }],
    ['a fix without the contract', 'two_confirmed', r => { r.feedback_1 = r.feedback_1.replace(/ to follow the contract in [^\n]*Non-integers raise TypeError\."/, ''); }],
    ['the first fix repeated for the second finding', 'two_confirmed', r => { const f = r.feedback_1.split('\n'); f[4] = f[2]; r.feedback_1 = f.join('\n'); }],
    ['fixes after all the items', 'two_confirmed', r => { const f = r.feedback_1.split('\n'); r.feedback_1 = [f[0], f[1], f[3], f[2], f[4]].join('\n'); }],
    ['a headline that counts the fix lines', 'two_confirmed', r => { r.feedback_1 = r.feedback_1.replace('Contract check: 2', 'Contract check: 4'); }],
    ['a repeat without the fix lines', 'two_confirmed', r => { r.feedback_2 = noFix(r.feedback_2.split('\n')).join('\n'); }],
    ['a repeat without the unchanged line', 'two_confirmed', r => { r.feedback_2 = r.feedback_1; }],
    ['a record without the fix lines', 'two_confirmed', r => { r.record_lines = noFix(r.record_lines); }],
    ['saved event problems without the fix lines', 'two_confirmed', r => { r.event_problems = noFix(r.event_problems); }],
    ['not-ready error lines without the fix lines', 'two_confirmed', r => { r.error_lines = noFix(r.error_lines); }],
    ['approval lines without the fix lines', 'two_confirmed', r => { r.approval_lines = noFix(r.approval_lines); }],
    ['record lines with the fixes after the items', 'two_confirmed', r => { r.record_lines = [r.record_lines[0], r.record_lines[2], r.record_lines[1], r.record_lines[3]]; }],
    ['a wrong exit', 'two_confirmed', r => { r.exit = 0; }],
    ['a corrected run that was refused', 'corrected', r => { r.exit = 1; r.published = false; }],
    ['events that count the fix lines', 'corrected', r => { r.contract_reviews[0].confirmed = 2; }],
    ['an unpublished corrected run', 'corrected', r => { r.published = false; }],
    ['an uncorrected feature', 'corrected', r => { r.feature = r.feature.replace('number 6', 'number 4'); }],
    ['a refusal after the correction', 'corrected', r => { r.feedback_2 = 'Contract check: 1 acceptance expectation(s)'; }],
    ['a record left after the correction', 'corrected', r => { r.record_exists = true; }],
    ['a refuted finding that refuses', 'refuted', r => { r.exit = 1; r.published = false; }],
    ['a refuted finding that is saved as a refusal', 'refuted', r => { r.contract_refusal_events = 1; }],
    ['a refuted finding with a fix line', 'refuted', r => { r.feedback_2 = '- fix for Even numbers: x'; }],
    ['a refuted finding with a record', 'refuted', r => { r.record_exists = true; }],
    ['events of a refused refuted finding', 'refuted', r => { r.contract_reviews[0].result = 'refused'; r.contract_reviews[0].confirmed = 1; }],
    ['events that count the fix lines of a long quote', 'long', r => { r.contract_reviews[0].confirmed = 2; }],
    ['a headline that counts the lines of a long quote', 'long', r => { r.feedback_1 = r.feedback_1.replace('Contract check: 1', 'Contract check: 2'); }],
    ['saved problems that are not cut', 'long', r => { r.event_problems = [...r.record_lines]; }],
    ['a saved fix problem that lost its lead', 'long', r => { r.event_problems[1] = r.event_problems[1].replace('- fix for Even numbers: in ', '- Even numbers: '); }],
    ['a record that is cut', 'long', r => { r.record_lines[1] = [...r.record_lines[1]].slice(0, BOUND).join(''); }],
  ];
  for (const [label, key, change] of mutations) rejects(() => judgeAll(withRun(key, change)), label);
  return 'rejected';
}

if (require.main === module) {
  const which = process.argv[2] === '--control' ? process.argv[3] : '';
  assert.ok(which === 'good' || which === 'bad', 'usage: --control good|bad');
  const outcome = control(which);
  console.log(outcome);
  process.exitCode = outcome === 'accepted' ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const when = (text, key) => When(P + text, {timeout: 660000}, function () { this.run = observation()[key]; });
  when('Shape shapes a package whose scripted contract review reports two confirmed contradictions and the shaper resubmits it unchanged', 'two_confirmed');
  when('Shape shapes a package whose scripted contract review reports one confirmed contradiction and the shaper then submits the corrected feature', 'corrected');
  when('Shape shapes a package whose scripted contract review reports one refuted finding and two ungrounded findings', 'refuted');
  when('Shape shapes a package whose scripted contract review reports one confirmed contradiction whose contract quote is long', 'long');
  Then(P + 'the refusal fed back after the first submission is:', function (t) { J.firstRefusalIs(this.run, t); });
  Then(P + 'the refusal fed back after the unchanged resubmission is the first refusal followed by:', function (t) { J.repeatIs(this.run, t); });
  Then(P + 'the run exits {int}', function (n) { J.exits(this.run, n); });
  Then(P + 'the saved not-ready details contain these lines in order:', function (t) { J.saved(this.run, t); });
  Then(P + 'the refusal fed back after the first submission ends with the line:', function (t) { J.firstRefusalEnds(this.run, t); });
  Then(P + 'the refusal fed back after the first submission starts with:', function (t) { J.firstRefusalStarts(this.run, t); });
  Then(P + 'the contract_review events are:', function (t) { J.events(this.run, JSON.parse(t)); });
  Then(P + 'the Intent is published', function () { J.published(this.run); });
  Then(P + 'the corrected acceptance feature is saved', function () { J.corrected(this.run); });
  Then(P + 'the feedback after the corrected submission is not a contract refusal', function () { J.notARefusal(this.run); });
  Then(P + 'no contradictions record is saved', function () { J.noRecord(this.run); });
  Then(P + 'no contract_review refusal event is saved', function () { J.noRefusalEvent(this.run); });
  Then(P + 'no feedback fed back to the shaper contains {string}', function (s) { J.notFed(this.run, s); });
  Then(P + 'the saved refusal event problems are the first 300 characters of the record lines, and the fix problem begins:', function (t) { J.bounded(this.run, t); });
  Then(P + 'the contradictions record keeps the full fix line, which ends with:', function (t) { J.fullFix(this.run, t); });
}
