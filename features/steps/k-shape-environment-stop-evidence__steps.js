'use strict';
// Acceptance of k-shape-environment-stop-evidence. The When step compiles the shape package's tests
// with the harness of k-shape-environment-stop-evidence__harness.js (through a Go overlay, once per
// process), runs TestKShapeEnvironmentStopEvidenceAcceptance for one case with KSES_CASE and reads
// the JSON observation the harness wrote to KSES_OUT. The Then steps compare it with the feature's
// values; one more Then step checks the SHA-256 digests of files of the checkout. Run this file
// directly with --control good or --control bad to check the judgments on known observations.
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKShapeEnvironmentStopEvidenceAcceptance';
const TARGET = 'shape/k_shape_environment_stop_evidence_acceptance_test.go';
const FEATURE = '.kogen/intents/k-shape-environment-stop-evidence/acceptance.feature';
const P = 'k-shape-environment-stop-evidence: ';

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kses-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./k-shape-environment-stop-evidence__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {env: env({}), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
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
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KSES_CASE: name, KSES_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  shaped[name] = judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `KSES_CASE=${name} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`}, name);
  return shaped[name];
}

function summary(o) {
  return `case ${o.case}: exit ${o.exit}, error ${JSON.stringify(o.error)}, error lines ${JSON.stringify(o.error_lines)}; ` +
    `${o.shaper_requests} shaper and ${o.question_check_requests} question check request(s); question check classes ` +
    `${JSON.stringify(o.question_check_classes)}; refused gates ${JSON.stringify(o.refused_gates)}; published ${o.published}\n` +
    `stderr (tail):\n${String(o.stderr || '').slice(-1500)}`;
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

// digests compares the SHA-256 of each listed file under root with the listed digest.
function digests(root, text) {
  const wrong = [];
  for (const line of text.split('\n').filter(l => l.trim() !== '')) {
    const [want, file] = line.trim().split(/\s+/);
    const p = path.join(root, file);
    const got = fs.existsSync(p) ? crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex') : 'missing';
    if (got !== want) wrong.push(`${file}: ${got}, want ${want}`);
  }
  assert.deepEqual(wrong, [], `files differ from their recorded digests: ${wrong.join('; ')}`);
}

// ---- observer controls: node features/steps/k-shape-environment-stop-evidence__steps.js --control good|bad ----

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
    } else if ((m = /^When k-shape-environment-stop-evidence: the "([^"]+)" case is shaped$/.exec(line))) {
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

function check(step, obs, root) {
  let m;
  if (step.text === P + 'the observations include:') return include(obs, JSON.parse(step.doc));
  if ((m = /^k-shape-environment-stop-evidence: the observation "([^"]+)" is exactly:$/.exec(step.text))) return exact(obs, m[1], step.doc);
  if ((m = /^k-shape-environment-stop-evidence: the observation "([^"]+)" is exactly these lines:$/.exec(step.text))) return exact(obs, m[1], step.doc + '\n');
  if (step.text === P + 'these files have exactly these SHA-256 digests:') return digests(root, step.doc);
  throw new Error('unknown step: ' + step.text);
}

// known is the observation a scenario expects, assembled from its steps.
function known(s) {
  const obs = {case: s.case, stderr: ''};
  for (const step of s.steps) {
    let m;
    if (step.text === P + 'the observations include:') Object.assign(obs, JSON.parse(step.doc));
    else if ((m = /"([^"]+)" is exactly:$/.exec(step.text))) obs[m[1]] = step.doc;
    else if ((m = /"([^"]+)" is exactly these lines:$/.exec(step.text))) obs[m[1]] = step.doc + '\n';
  }
  return obs;
}

// RUN is a real harness run's -test.v stdout on the unchanged tree (KSES_CASE=spent).
const RUN = `=== RUN   ${TEST}\n=== PAUSE ${TEST}\n=== CONT  ${TEST}\n--- PASS: ${TEST} (0.94s)\nPASS\n`;

const FAULT = 'environment/shape_environment: shaping cannot go on without something only your environment can provide; nothing was asked or published';
const MALFORMED = 'shape/question_check_invalid: the question check answer is not a valid submit_question_check payload (an item is malformed); nothing was asked';
const OLD_ENV = 'do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault';
const OLD_ENUM = ['ui_ux', 'handoff', 'answered', 'technical', 'product_policy', 'environment', 'repository_fact'];

// red are the unchanged tree's observations, as far as they differ from the expected ones.
function red(name, obs) {
  switch (name) {
    case 'stand-in':
      return {...obs, feedback_q1: obs.feedback_q1.replace(/\| [^|]*$/, '| ' + OLD_ENV)};
    case 'contradicted':
      return {...obs, exit: 3, error: FAULT, shaper_requests: 3, published: false, intent_exists: false, last_stdout_line: ''};
    case 'live':
      return {...obs, exit: 1, error: MALFORMED, error_lines: [], shaper_requests: 2, question_check_requests: 1, question_check_classes: [], refused_gates: [], question_check_class_enum: OLD_ENUM};
    case 'spent':
      return {...obs, exit: 3, error: FAULT, error_last_line: '- Is SKYLINE_API_KEY set where bin/frost-forecast runs?'};
    case 'spent-evidenced':
      return {...obs, error: MALFORMED, error_last_line: '', question_check_classes: [['product_policy'], ['product_policy']], refused_gates: ['question_check', 'question_check']};
  }
  return null; // spent-ui: the ask path passes on the unchanged tree too
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function tree(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kses-tree-'));
  owned.push(root);
  for (const [p, c] of Object.entries(files)) {
    fs.mkdirSync(path.dirname(path.join(root, p)), {recursive: true});
    fs.writeFileSync(path.join(root, p), c);
  }
  return root;
}

function control(which) {
  const all = scenarios();
  const harness = all.filter(s => s.case);
  const checkout = all.filter(s => !s.case);
  assert.deepEqual(harness.map(s => s.case), ['stand-in', 'contradicted', 'live', 'spent', 'spent-evidenced', 'spent-ui'], 'the harness scenarios of the feature');
  assert.equal(checkout.length, 1, 'the checkout scenario of the feature');
  const listed = checkout[0].steps[0].doc.split('\n').filter(l => l.trim()).map(l => l.trim().split(/\s+/)[1]);
  const root = path.join(__dirname, '../..');
  const copy = Object.fromEntries(listed.map(p => [p, fs.readFileSync(path.join(root, p))]));
  const run = (s, extra) => ({status: 0, signal: null, error: null, output: RUN, json: JSON.stringify(known(s)), command: 'recorded ' + s.case, ...extra});
  if (which === 'good') {
    for (const s of harness) {
      const obs = judge(run(s), s.case);
      for (const step of s.steps) check(step, obs);
    }
    for (const step of checkout[0].steps) check(step, null, tree(copy));
    return 'accepted';
  }
  for (const s of harness) {
    const good = run(s);
    rejects(() => judge({...good, status: 1}, s.case), `a failed run (${s.case})`);
    rejects(() => judge({...good, error: 'spawnSync ETIMEDOUT'}, s.case), `a run that did not finish (${s.case})`);
    rejects(() => judge({...good, output: RUN.replace('--- PASS', '--- FAIL')}, s.case), `a failed harness test (${s.case})`);
    rejects(() => judge({...good, output: RUN + RUN}, s.case), `two harness runs (${s.case})`);
    rejects(() => judge({...good, json: null}, s.case), `a run without an observation (${s.case})`);
    rejects(() => judge(good, s.case === 'live' ? 'spent' : 'live'), `the observation of another case (${s.case})`);
    const obs = judge(good, s.case);
    const all = o => () => { for (const step of s.steps) check(step, o); };
    const before = red(s.case, obs);
    if (before) rejects(all(before), `the unchanged tree's observation (${s.case})`);
    for (const step of s.steps) {
      if (step.text === P + 'the observations include:') {
        for (const [key, value] of Object.entries(JSON.parse(step.doc))) {
          const wrong = typeof value === 'boolean' ? !value : typeof value === 'number' ? value + 1 : typeof value === 'string' ? value + 'x'
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
  const [hashes] = checkout[0].steps;
  for (const p of listed) {
    rejects(() => check(hashes, null, tree({...copy, [p]: Buffer.concat([copy[p], Buffer.from('\n')])})), `a changed ${p}`);
    const without = {...copy}; delete without[p];
    rejects(() => check(hashes, null, tree(without)), `a deleted ${p}`);
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
  When('k-shape-environment-stop-evidence: the {string} case is shaped', {timeout: 540000}, function (name) {
    this.kses = shapeCase(name);
  });
  Then('k-shape-environment-stop-evidence: the observations include:', function (text) {
    include(this.kses, JSON.parse(text));
  });
  Then('k-shape-environment-stop-evidence: the observation {string} is exactly:', function (key, text) {
    exact(this.kses, key, text);
  });
  Then('k-shape-environment-stop-evidence: the observation {string} is exactly these lines:', function (key, text) {
    exact(this.kses, key, text + '\n');
  });
  Then('k-shape-environment-stop-evidence: these files have exactly these SHA-256 digests:', function (text) {
    digests(process.cwd(), text);
  });
}
