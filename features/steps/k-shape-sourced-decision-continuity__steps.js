'use strict';
// Acceptance of k-shape-sourced-decision-continuity. The When step compiles the shape package's tests
// with the harness of k-shape-sourced-decision-continuity__harness.js (through a Go overlay, once per
// process), runs TestKShapeSourcedDecisionContinuityAcceptance for one journey with KSDC_CASE and reads
// the JSON observation the harness wrote to KSDC_OUT (once per journey and process). The Then steps
// compare it with the feature's values. Run this file directly with --control good or --control bad
// to check the judgments on known observations.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKShapeSourcedDecisionContinuityAcceptance';
const TARGET = 'shape/k_shape_sourced_decision_continuity_acceptance_test.go';
const FEATURE = '.kogen/intents/k-shape-sourced-decision-continuity/acceptance.feature';
const P = 'k-shape-sourced-decision-continuity: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let built;
// build compiles the shape tests with the harness once per process; later journeys fail fast with
// the same build failure.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ksdc-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./k-shape-sourced-decision-continuity__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {env: env({}), encoding: 'utf8', timeout: 280000, maxBuffer: 32 * 1024 * 1024});
  built = r.status === 0 ? {bin, dir} : {error: `building the harness of ${TEST}: go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout}${r.stderr}`};
  return built;
}

// judge is the observation of one harness run: the run exited 0, passed the harness test exactly
// once and wrote an observation of the journey.
function judge(r, name) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  const obs = JSON.parse(r.json);
  assert.equal(obs.case, name, `${TEST} observed journey ${obs.case}, want ${name}`);
  return obs;
}

const shaped = {};
function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KSDC_CASE: name, KSDC_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  shaped[name] = judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `KSDC_CASE=${name} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`}, name);
  return shaped[name];
}

function summary(o) {
  const stderr = Object.keys(o).filter(k => k.startsWith('stderr_')).sort().map(k => `${k} (tail):\n${String(o[k]).slice(-1200)}`).join('\n');
  return `journey ${o.case}: exits ${JSON.stringify(o.exits)}, errors ${JSON.stringify(o.errors)}, error lines ${JSON.stringify(o.error_lines)}; ` +
    `shaper requests ${JSON.stringify(o.shaper_requests)}, question checks ${JSON.stringify(o.question_check_requests)}; question check classes ` +
    `${JSON.stringify(o.question_check_classes)}; refused gates ${JSON.stringify(o.refused_gates)}; published ${o.published}\n${stderr}`;
}

function has(o, key) {
  assert.ok(Object.hasOwn(o, key), `the observation of journey ${o.case} has no ${key}\n${summary(o)}`);
}

function include(o, want) {
  for (const [key, value] of Object.entries(want)) {
    has(o, key);
    assert.deepEqual(o[key], value, `journey ${o.case}: ${key} is ${JSON.stringify(o[key])}, want ${JSON.stringify(value)}\n${summary(o)}`);
  }
}

function exact(o, key, text) {
  has(o, key);
  assert.equal(o[key], text, `journey ${o.case}: ${key} is not exactly the expected text:\n${o[key]}\n${summary(o)}`);
}

function sameJSON(o, key, text) {
  has(o, key);
  assert.deepEqual(o[key], JSON.parse(text), `journey ${o.case}: ${key} is ${JSON.stringify(o[key])}, want ${text}\n${summary(o)}`);
}

// ---- observer controls: node features/steps/k-shape-sourced-decision-continuity__steps.js --control good|bad ----

// scenarios are the feature's scenarios: for each, its journey and its Then steps with doc strings.
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
    } else if ((m = /^When k-shape-sourced-decision-continuity: the "([^"]+)" journey is shaped$/.exec(line))) {
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
  if ((m = /^k-shape-sourced-decision-continuity: the observation "([^"]+)" is exactly:$/.exec(step.text))) return exact(obs, m[1], step.doc);
  if ((m = /^k-shape-sourced-decision-continuity: the observation "([^"]+)" is exactly these lines:$/.exec(step.text))) return exact(obs, m[1], step.doc + '\n');
  if ((m = /^k-shape-sourced-decision-continuity: the observation "([^"]+)" is this JSON:$/.exec(step.text))) return sameJSON(obs, m[1], step.doc);
  throw new Error('unknown step: ' + step.text);
}

// known is the observation a journey's scenarios expect, assembled from all their steps.
function known(name, all) {
  const obs = {case: name};
  for (const s of all.filter(s => s.case === name)) {
    for (const step of s.steps) {
      let m;
      if (step.text === P + 'the observations include:') Object.assign(obs, JSON.parse(step.doc));
      else if ((m = /"([^"]+)" is exactly:$/.exec(step.text))) obs[m[1]] = step.doc;
      else if ((m = /"([^"]+)" is exactly these lines:$/.exec(step.text))) obs[m[1]] = step.doc + '\n';
      else if ((m = /"([^"]+)" is this JSON:$/.exec(step.text))) obs[m[1]] = JSON.parse(step.doc);
    }
  }
  return obs;
}

// RUN is a real harness run's -test.v stdout on the unchanged tree (KSDC_CASE=ordinary).
const RUN = `=== RUN   ${TEST}\n=== PAUSE ${TEST}\n=== CONT  ${TEST}\n--- PASS: ${TEST} (2.13s)\nPASS\n`;

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function wrongValue(value) {
  if (value === null) return 'x';
  if (typeof value === 'boolean') return !value;
  if (typeof value === 'number') return value + 1;
  if (typeof value === 'string') return value + 'x';
  if (Array.isArray(value)) return [...value, 'x'];
  return {...value, extra: 1};
}

// corruptions are copies of a JSON value with one change each: every leaf made wrong, every list
// without its last element and every list of two or more reversed.
function corruptions(value) {
  const out = [];
  const walk = (v, at) => {
    const put = (label, f) => { const c = structuredClone(value); let p = c; for (const k of at.slice(0, -1)) p = p[k]; if (at.length) p[at[at.length - 1]] = f(at.reduce((x, k) => x[k], c)); out.push({label, value: at.length ? c : f(c)}); };
    if (v !== null && typeof v === 'object') {
      if (Array.isArray(v)) {
        if (v.length) put(`a list without its last element at ${at.join('.')}`, x => x.slice(0, -1));
        if (v.length > 1) put(`a reversed list at ${at.join('.')}`, x => [...x].reverse());
      }
      for (const k of Object.keys(v)) walk(v[k], [...at, k]);
      return;
    }
    put(`a wrong value at ${at.join('.')}`, wrongValue);
  };
  walk(value, []);
  return out;
}

function control(which) {
  const all = scenarios();
  assert.deepEqual(all.map(s => s.case), ['settle', 'settle', 'settle', 'revise', 'repeat', 'external', 'ordinary'], 'the journeys of the feature');
  const run = (name, obs, extra) => ({status: 0, signal: null, error: null, output: RUN, json: JSON.stringify(obs), command: 'recorded ' + name, ...extra});
  if (which === 'good') {
    for (const s of all) {
      const obs = judge(run(s.case, known(s.case, all)), s.case);
      for (const step of s.steps) check(step, obs);
    }
    return 'accepted';
  }
  const red = require('./k-shape-sourced-decision-continuity__red.js');
  assert.deepEqual(Object.keys(red).sort(), ['external', 'repeat', 'revise', 'settle'], 'the recorded unchanged-tree journeys');
  for (const s of all) {
    const good = run(s.case, known(s.case, all));
    rejects(() => judge({...good, status: 1}, s.case), `a failed run (${s.case})`);
    rejects(() => judge({...good, error: 'spawnSync ETIMEDOUT'}, s.case), `a run that did not finish (${s.case})`);
    rejects(() => judge({...good, output: RUN.replace('--- PASS', '--- FAIL')}, s.case), `a failed harness test (${s.case})`);
    rejects(() => judge({...good, output: RUN + RUN}, s.case), `two harness runs (${s.case})`);
    rejects(() => judge({...good, json: null}, s.case), `a run without an observation (${s.case})`);
    rejects(() => judge(good, s.case === 'settle' ? 'revise' : 'settle'), `the observation of another journey (${s.case})`);
    const obs = judge(good, s.case);
    const every = o => () => { for (const step of s.steps) check(step, o); };
    // The ordinary journey keeps today's path: the unchanged tree passes it too.
    if (s.case !== 'ordinary') rejects(every(judge(run(s.case, red[s.case]), s.case)), `the unchanged tree's observation (${s.name})`);
    for (const step of s.steps) {
      if (step.text === P + 'the observations include:') {
        for (const [key, value] of Object.entries(JSON.parse(step.doc))) {
          rejects(() => check(step, {...obs, [key]: wrongValue(value)}), `a wrong ${key} (${s.case})`);
          const missing = {...obs}; delete missing[key];
          rejects(() => check(step, missing), `a missing ${key} (${s.case})`);
        }
        continue;
      }
      const key = /"([^"]+)"/.exec(step.text)[1];
      const missing = {...obs}; delete missing[key];
      rejects(() => check(step, missing), `a missing ${key} (${s.case})`);
      if (/is this JSON:$/.test(step.text)) {
        const value = obs[key];
        rejects(() => check(step, {...obs, [key]: null}), `a null ${key} (${s.case})`);
        for (const changed of corruptions(value)) {
          rejects(() => check(step, {...obs, [key]: changed.value}), `${changed.label} in ${key} (${s.case})`);
        }
        continue;
      }
      const text = obs[key];
      rejects(() => check(step, {...obs, [key]: text + '\n'}), `a trailing newline in ${key} (${s.case})`);
      rejects(() => check(step, {...obs, [key]: text.slice(0, -2)}), `a text missing its end in ${key} (${s.case})`);
      rejects(() => check(step, {...obs, [key]: text.split('\n').slice(1).join('\n')}), `a missing first line in ${key} (${s.case})`);
      if (text.includes('D1')) rejects(() => check(step, {...obs, [key]: text.replace(/D1/g, 'D9')}), `another decision id in ${key} (${s.case})`);
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
  When('k-shape-sourced-decision-continuity: the {string} journey is shaped', {timeout: 540000}, function (name) {
    this.ksdc = shapeCase(name);
  });
  Then('k-shape-sourced-decision-continuity: the observations include:', function (text) {
    include(this.ksdc, JSON.parse(text));
  });
  Then('k-shape-sourced-decision-continuity: the observation {string} is exactly:', function (key, text) {
    exact(this.ksdc, key, text);
  });
  Then('k-shape-sourced-decision-continuity: the observation {string} is exactly these lines:', function (key, text) {
    exact(this.ksdc, key, text + '\n');
  });
  Then('k-shape-sourced-decision-continuity: the observation {string} is this JSON:', function (key, text) {
    sameJSON(this.ksdc, key, text);
  });
}
