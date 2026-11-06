'use strict';
// Acceptance of k-shape-question-options-recommended-checked. The When step compiles the shape
// package's tests with the harness of k-shape-question-options-recommended-checked__harness.js
// (through a Go overlay, once per process), runs TestKShapeQuestionOptionsRecommendedCheckedAcceptance
// for one case with KQOR_CASE and reads the JSON observation the harness wrote to KQOR_OUT. The Then
// steps compare it with the feature's values. Run this file directly with --control good or
// --control bad to check the judgments on known observations.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKShapeQuestionOptionsRecommendedCheckedAcceptance';
const TARGET = 'shape/k_shape_question_options_recommended_checked_acceptance_test.go';
const FEATURE = '.kogen/intents/k-shape-question-options-recommended-checked/acceptance.feature';
const SLUG = 'k-shape-question-options-recommended-checked';
const P = SLUG + ': ';
const CASES = ['no-options', 'empty-options', 'no-recommended', 'two-recommended', 'repeated-label', 'empty-label', 'one-option', 'second-of-two', 'bound', 'handoff', 'ui', 'handoff-wait'];

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let built;
// build compiles the shape tests with the harness once per process; later cases fail fast with
// the same build failure.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kqor-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./k-shape-question-options-recommended-checked__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  // PWD names the directory the overlay key was resolved against, so go reads the same path.
  const r = spawnSync('go', args, {env: env({PWD: process.cwd()}), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  built = r.status === 0 ? {bin, dir} : {error: `building the harness of ${TEST}: go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout}${r.stderr}`};
  return built;
}

// judge is the observation of one harness run: the run exited 0, passed the harness test exactly
// once and wrote an observation of the case.
function judge(r, name) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  const obs = JSON.parse(r.json);
  assert.equal(obs.case, name, `${TEST} observed case ${obs.case}, want ${name}`);
  return obs;
}

const shaped = {};
function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KQOR_CASE: name, KQOR_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  shaped[name] = judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `KQOR_CASE=${name} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`}, name);
  return shaped[name];
}

function summary(o) {
  return `case ${o.case}: exit ${o.exit}, error ${JSON.stringify(o.error)}, error lines ${JSON.stringify(o.error_lines)}; ` +
    `requests ${JSON.stringify(o.request_kinds)}; refused gates ${JSON.stringify(o.refused_gates)}; ` +
    `answer exit ${o.answer_exit}, answer error ${JSON.stringify(o.answer_error)}; published ${o.published}\n` +
    `stdout:\n${o.stdout}`;
}

function include(o, want) {
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(o, key), `observation of case ${o.case} has no ${key}\n${summary(o)}`);
    assert.deepEqual(o[key], value, `case ${o.case}: ${key} is ${JSON.stringify(o[key])}, want ${JSON.stringify(value)}\n${summary(o)}`);
  }
}

function exact(o, key, text) {
  assert.ok(Object.hasOwn(o, key), `observation of case ${o.case} has no ${key}\n${summary(o)}`);
  assert.equal(o[key], text, `case ${o.case}: ${key} is not exactly the expected text:\n${o[key]}\n${summary(o)}`);
}

// ---- observer controls: node features/steps/k-shape-question-options-recommended-checked__steps.js --control good|bad ----

// scenarios are the feature's scenarios: for each, its case and its Then steps with doc strings.
function scenarios() {
  const text = fs.readFileSync(path.join(__dirname, '../..', FEATURE), 'utf8').split('\n');
  const out = [];
  let cur = null;
  for (let i = 0; i < text.length; i++) {
    const line = text[i].trim();
    let m;
    if (line.startsWith('Scenario:')) {
      cur = {name: line, case: null, steps: []};
      out.push(cur);
    } else if ((m = /^When k-shape-question-options-recommended-checked: the "([^"]+)" case is shaped$/.exec(line))) {
      cur.case = m[1];
    } else if (cur && /^(Then|And) /.test(line)) {
      const step = {text: line.replace(/^(Then|And) /, ''), doc: null};
      if (text[i + 1] !== undefined && text[i + 1].trim() === '"""') {
        const indent = text[i + 1].indexOf('"""');
        const doc = [];
        for (i += 2; text[i].trim() !== '"""'; i++) doc.push(text[i].slice(indent));
        step.doc = doc.join('\n');
      }
      cur.steps.push(step);
    }
  }
  return out;
}

function check(step, obs) {
  let m;
  if (step.text === P + 'the observations include:') return include(obs, JSON.parse(step.doc));
  if ((m = /^k-shape-question-options-recommended-checked: the observation "([^"]+)" is exactly:$/.exec(step.text))) return exact(obs, m[1], step.doc);
  if ((m = /^k-shape-question-options-recommended-checked: the observation "([^"]+)" is exactly these lines:$/.exec(step.text))) return exact(obs, m[1], step.doc + '\n');
  throw new Error('unknown step: ' + step.text);
}

// known is the observation a scenario expects, assembled from its steps.
function known(s) {
  const obs = {case: s.case};
  for (const step of s.steps) {
    let m;
    if (step.text === P + 'the observations include:') Object.assign(obs, JSON.parse(step.doc));
    else if ((m = /"([^"]+)" is exactly:$/.exec(step.text))) obs[m[1]] = step.doc;
    else if ((m = /"([^"]+)" is exactly these lines:$/.exec(step.text))) obs[m[1]] = step.doc + '\n';
  }
  return obs;
}

// RUN is a real harness run's -test.v output on the unchanged tree (KQOR_CASE=ui).
const RUN = `=== RUN   ${TEST}\n=== PAUSE ${TEST}\n=== CONT  ${TEST}\n--- PASS: ${TEST} (1.70s)\nPASS\n`;

// PRESERVED are the cases whose unchanged-tree observation the feature accepts as it is.
const PRESERVED = ['handoff', 'ui', 'handoff-wait'];

// UNCHANGED are real observations of the harness on the unchanged tree (HEAD 4a41927), limited to the
// values the feature compares; a value the run did not produce is absent.
const UNCHANGED = {
  "no-options": {"case":"no-options","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":null}],"slug":"is-even"},"state_rounds":[[{"question":"How should calc.is_even(n) answer a number that is not an integer?"}]],"answer_exit":2,"answer_error":"intent/answers_invalid: question 1 has no recommended option","answer_last_stdout_line":"","receipt_answers":null,"receipt_resolved_answers":null,"pending_after_answer":true,"published":false,"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: your words\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n","question_check_input_1":"Slug: is-even\n\n# Request\n\nAdd calc.is_even(n), which returns true for even integers.\n\n\n# Earlier questions and answers\n\nnone\n\n# Questions\n\n1. How should calc.is_even(n) answer a number that is not an integer?","earlier_round":""},
  "empty-options": {"case":"empty-options","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":null}],"slug":"is-even"},"state_rounds":[[{"options":[],"question":"How should calc.is_even(n) answer a number that is not an integer?"}]],"answer_exit":2,"answer_error":"intent/answers_invalid: question 1 has no recommended option","receipt_answers":null,"receipt_resolved_answers":null,"published":false,"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: your words\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n"},
  "no-recommended": {"case":"no-recommended","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[{"label":"Raise TypeError","letter":"a","recommended":false},{"label":"Return false","letter":"b","recommended":false}],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":null}],"slug":"is-even"},"state_rounds":[[{"options":[{"label":"Raise TypeError"},{"label":"Return false","recommended":false}],"question":"How should calc.is_even(n) answer a number that is not an integer?"}]],"answer_exit":2,"answer_error":"intent/answers_invalid: question 1 has no recommended option","answer_error_lines":[],"receipt_answers":null,"receipt_resolved_answers":null,"published":false,"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\n   a) Raise TypeError\n   b) Return false\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n"},
  "two-recommended": {"case":"two-recommended","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[{"label":"Raise TypeError","letter":"a","recommended":true},{"label":"Return false","letter":"b","recommended":true}],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":"a"}],"slug":"is-even"},"state_rounds":[[{"options":[{"label":"Raise TypeError","recommended":true},{"label":"Return false","recommended":true}],"question":"How should calc.is_even(n) answer a number that is not an integer?"}]],"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\n   a) Raise TypeError (recommended)\n   b) Return false (recommended)\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n"},
  "repeated-label": {"case":"repeated-label","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"state_rounds":[[{"options":[{"label":"Raise TypeError","recommended":true},{"label":"Raise TypeError"}],"question":"How should calc.is_even(n) answer a number that is not an integer?"}]],"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\n   a) Raise TypeError (recommended)\n   b) Raise TypeError\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n"},
  "empty-label": {"case":"empty-label","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"state_rounds":[[{"options":[{"label":"Raise TypeError","recommended":true},{"label":"  "}],"question":"How should calc.is_even(n) answer a number that is not an integer?"}]]},
  "one-option": {"case":"one-option","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"state_rounds":[[{"options":[{"label":"Raise TypeError","recommended":true}],"question":"How should calc.is_even(n) answer a number that is not an integer?"}]]},
  "second-of-two": {"case":"second-of-two","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[{"label":"Raise TypeError","letter":"a","recommended":true},{"label":"Return false","letter":"b","recommended":false}],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":"a"},{"kind":"product","number":2,"options":[{"label":"No, only numbers","letter":"a","recommended":false},{"label":"Yes, parse them","letter":"b","recommended":false}],"question":"Should calc.is_even(n) accept numbers given as strings?","recommended":null}],"slug":"is-even"},"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\n   a) Raise TypeError (recommended)\n   b) Return false\n2. Should calc.is_even(n) accept numbers given as strings?\n   a) No, only numbers\n   b) Yes, parse them\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n2: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n"},
  "bound": {"case":"bound","exit":5,"error":"","error_lines":[],"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: your words\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":null}],"slug":"is-even"},"state_rounds":[[{"question":"How should calc.is_even(n) answer a number that is not an integer?"}]]},
  "handoff": {"case":"handoff","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check","shaper","decision_check","literal_check","contract_check"],"questions_json":{"questions":[{"kind":"handoff","number":1,"options":[{"label":"I have signed in","letter":"a","recommended":true},{"label":"Pause shaping until I have signed in","letter":"b","recommended":false}],"question":"Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.","recommended":"a"}],"slug":"is-even"},"answer_exit":0,"answer_error":"","answer_last_stdout_line":"Next: kogen intent approve is-even","receipt_answers":{"1":"a"},"receipt_resolved_answers":{"1":"I have signed in"},"pending_after_answer":false,"published":true,"stdout":"Questions for is-even:\nThis question is an authentication or security handoff, not a design choice.\n1. Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.\n   a) I have signed in (recommended)\n   b) Pause shaping until I have signed in\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n","earlier_round":"Round 1:\n1. Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.\n   a) I have signed in\n   b) Pause shaping until I have signed in\n   answer: 1: a"},
  "ui": {"case":"ui","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check","shaper","decision_check","literal_check","contract_check"],"questions_json":{"questions":[{"kind":"product","number":1,"options":[{"label":"Raise TypeError","letter":"a","recommended":true},{"label":"Return false","letter":"b","recommended":false}],"question":"How should calc.is_even(n) answer a number that is not an integer?","recommended":"a"}],"slug":"is-even"},"answer_exit":0,"answer_error":"","answer_last_stdout_line":"Next: kogen intent approve is-even","receipt_answers":{"1":"b"},"receipt_resolved_answers":{"1":"Return false"},"published":true,"stdout":"Questions for is-even:\n1. How should calc.is_even(n) answer a number that is not an integer?\n   a) Raise TypeError (recommended)\n   b) Return false\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n","question_check_input_1":"Slug: is-even\n\n# Request\n\nAdd calc.is_even(n), which returns true for even integers.\n\n\n# Earlier questions and answers\n\nnone\n\n# Questions\n\n1. How should calc.is_even(n) answer a number that is not an integer?\n   a) Raise TypeError (recommended)\n   b) Return false"},
  "handoff-wait": {"case":"handoff-wait","exit":5,"error":"","refused_gates":[],"request_kinds":["shaper","question_check"],"questions_json":{"questions":[{"kind":"handoff","number":1,"options":[{"label":"Wait: pause shaping until I have signed in","letter":"a","recommended":true},{"label":"I have signed in","letter":"b","recommended":false}],"question":"Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.","recommended":"a"}],"slug":"is-even"},"stdout":"Questions for is-even:\nThis question is an authentication or security handoff, not a design choice.\n1. Sign in to the npm registry in your browser so that acceptance can install the is-even test tools.\n   a) Wait: pause shaping until I have signed in (recommended)\n   b) I have signed in\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n"},
};

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function control(which) {
  const all = scenarios();
  assert.deepEqual(all.map(s => s.case), CASES, 'the scenarios of the feature');
  const run = (s, json) => ({status: 0, signal: null, error: null, output: RUN, json, command: 'recorded ' + s.case});
  const thens = (s, o) => () => { for (const step of s.steps) check(step, o); };
  if (which === 'good') {
    for (const s of all) {
      thens(s, judge(run(s, JSON.stringify(known(s))), s.case))();
      if (PRESERVED.includes(s.case)) thens(s, judge(run(s, JSON.stringify(UNCHANGED[s.case])), s.case))();
    }
    return 'accepted';
  }
  for (const s of all) {
    const good = run(s, JSON.stringify(known(s)));
    rejects(() => judge({...good, status: 1}, s.case), `a failed run (${s.case})`);
    rejects(() => judge({...good, error: 'spawnSync ETIMEDOUT'}, s.case), `a run that did not finish (${s.case})`);
    rejects(() => judge({...good, output: RUN.replace('--- PASS', '--- FAIL')}, s.case), `a failed harness test (${s.case})`);
    rejects(() => judge({...good, output: RUN + RUN}, s.case), `two harness runs (${s.case})`);
    rejects(() => judge({...good, json: null}, s.case), `a run without an observation (${s.case})`);
    rejects(() => judge(good, s.case === 'ui' ? 'handoff' : 'ui'), `the observation of another case (${s.case})`);
    const obs = judge(good, s.case);
    if (!PRESERVED.includes(s.case)) rejects(thens(s, judge(run(s, JSON.stringify(UNCHANGED[s.case])), s.case)), `the unchanged tree's observation (${s.case})`);
    for (const step of s.steps) {
      if (step.text === P + 'the observations include:') {
        for (const [key, value] of Object.entries(JSON.parse(step.doc))) {
          const wrong = value === null ? {} : typeof value === 'boolean' ? !value : typeof value === 'number' ? value + 1 : typeof value === 'string' ? value + 'x'
            : Array.isArray(value) ? [...value, 'x'] : {...value, extra: 1};
          rejects(() => check(step, {...obs, [key]: wrong}), `a wrong ${key} (${s.case})`);
          const missing = {...obs}; delete missing[key];
          rejects(() => check(step, missing), `a missing ${key} (${s.case})`);
        }
      } else {
        const key = /"([^"]+)"/.exec(step.text)[1];
        const text = obs[key];
        rejects(() => check(step, {...obs, [key]: text + '\n'}), `a trailing newline in ${key} (${s.case})`);
        rejects(() => check(step, {...obs, [key]: text.slice(0, -2)}), `a text missing its end in ${key} (${s.case})`);
        rejects(() => check(step, {...obs, [key]: text.split('\n').slice(1).join('\n')}), `a missing first line in ${key} (${s.case})`);
        const missing = {...obs}; delete missing[key];
        rejects(() => check(step, missing), `a missing ${key} (${s.case})`);
      }
    }
  }
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
  When('k-shape-question-options-recommended-checked: the {string} case is shaped', {timeout: 540000}, function (name) {
    this.kqor = shapeCase(name);
  });
  Then('k-shape-question-options-recommended-checked: the observations include:', function (text) {
    include(this.kqor, JSON.parse(text));
  });
  Then('k-shape-question-options-recommended-checked: the observation {string} is exactly:', function (key, text) {
    exact(this.kqor, key, text);
  });
  Then('k-shape-question-options-recommended-checked: the observation {string} is exactly these lines:', function (key, text) {
    exact(this.kqor, key, text + '\n');
  });
}
