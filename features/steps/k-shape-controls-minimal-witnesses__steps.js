'use strict';
// Acceptance of k-shape-controls-minimal-witnesses. The Given step compiles the shape package's tests
// with the harness of k-shape-controls-minimal-witnesses__harness.js (through a Go overlay), runs
// TestKShapeControlsMinimalWitnessesAcceptance once per Cucumber process and reads the JSON
// observation the harness wrote to KSCMW_OUT. The Then steps compare listed keys of one case of it
// with the feature's values. Run this file directly with --control good or --control bad to check
// those judgments on known observations.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKShapeControlsMinimalWitnessesAcceptance';
const TARGET = 'shape/k_shape_controls_minimal_witnesses_acceptance_test.go';
const FEATURE = '.kogen/intents/k-shape-controls-minimal-witnesses/acceptance.feature';
const P = 'k-shape-controls-minimal-witnesses: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

// judge is the observation of one harness run: the run exited 0, passed the harness test exactly
// once and wrote an observation.
function judge(r) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  return JSON.parse(r.json);
}

// binary builds the shape tests with the harness once per Cucumber process.
let built = null;
function binary() {
  if (built !== null) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kscmw-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./k-shape-controls-minimal-witnesses__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 400000, maxBuffer: 32 * 1024 * 1024});
  if (b.status !== 0) {
    throw new Error(`building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}${b.signal ? ' (' + b.signal + ')' : ''}${b.error ? ' ' + b.error.message : ''}:\n${b.stdout}${b.stderr}`);
  }
  built = {dir, bin};
  return built;
}

// harness runs the harness test once.
function harness() {
  const {dir, bin} = binary();
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=250s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({KSCMW_OUT: out}), encoding: 'utf8', timeout: 270000, maxBuffer: 32 * 1024 * 1024});
  return judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `KSCMW_OUT=${out} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`});
}

// include checks that each listed key of one case of the observation has exactly the listed value.
function include(o, name, want) {
  assert.ok(o && Object.hasOwn(o, name), `the harness observed no case "${name}":\n${JSON.stringify(o, null, 1)}`);
  const c = o[name];
  const summary = JSON.stringify(c, null, 1);
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(c, key), `case "${name}" has no ${key}:\n${summary}`);
    assert.deepEqual(c[key], value, `case "${name}": ${key} is ${JSON.stringify(c[key])}, want ${JSON.stringify(value)}\nobservation:\n${summary}`);
  }
}

// ---- observer controls: node features/steps/k-shape-controls-minimal-witnesses__steps.js --control good|bad ----

// checks are the feature's Then steps, in order, each as a judgment of an observation.
function checks() {
  const lines = fs.readFileSync(path.join(__dirname, '../..', FEATURE), 'utf8').split('\n');
  const out = [];
  for (let i = 0; i < lines.length; i++) {
    const step = lines[i].trim().replace(/^(Then|And) /, '');
    const m = step.match(/^k-shape-controls-minimal-witnesses: the observation of "([^"]+)" includes:$/);
    if (!m) continue;
    assert.equal(lines[i + 1].trim(), '"""', 'an include step has a doc string');
    const end = lines.findIndex((l, j) => j > i + 1 && l.trim() === '"""');
    const want = JSON.parse(lines.slice(i + 2, end).join('\n'));
    out.push({name: m[1], want, run: o => include(o, m[1], want)});
    i = end;
  }
  assert.ok(out.length > 0, 'the feature has judgments');
  return out;
}

// RUN is a real harness run's -test.v output on the unchanged tree.
const RUN = `=== RUN   ${TEST}\n=== PAUSE ${TEST}\n=== CONT  ${TEST}\n--- PASS: ${TEST} (5.30s)\nPASS\n`;

// RED is the unchanged tree's observation (HEAD 92b093c), whose runs each execute every scenario.
const RED = JSON.parse('{"clean":{"controls":[],"error":"","event_result":"clean","event_runs":3,"events":1,"executed":[["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"]],"headline_caps":false,"refusal":[],"runs":3},"earlier":{"controls":[],"error":"","event_result":"refused","event_runs":3,"events":1,"executed":[["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"]],"headline_caps":true,"refusal":["- add keeps working (kept) | request: add | problem: wrong implementation \\"add removed\\" fails scenario \\"Add keeps working\\" at \\"Then is-even: calc takes 3 arguments\\", not at \\"And is-even: add of 2 and 3 gives an answer\\""],"runs":3},"outline":{"controls":[],"error":"","event_result":"clean","event_runs":3,"events":1,"executed":[["Add keeps working","Sub keeps working","Sub keeps working"],["Add keeps working","Sub keeps working","Sub keeps working"],["Add keeps working","Sub keeps working","Sub keeps working"]],"headline_caps":false,"refusal":[],"runs":3},"passes":{"controls":[],"error":"","event_result":"refused","event_runs":3,"events":1,"executed":[["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"]],"headline_caps":true,"refusal":["- add keeps working (kept) | request: add | problem: wrong implementation \\"add removed\\" passes scenario \\"Add keeps working\\""],"runs":3},"run-cap":{"controls":[],"error":"","event_result":"refused","event_runs":0,"events":1,"executed":[],"headline_caps":true,"refusal":["- run cap: this candidate needs 9 runs (the correct implementation and 8 wrong ones), above the cap of 8 runs per candidate; nothing was run. Supply at most 7 wrong implementations."],"runs":0},"target-scenario-only":{"controls":[],"error":"","event_result":"refused","event_runs":3,"events":1,"executed":[["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"],["Add keeps working","Sub keeps working"]],"headline_caps":true,"refusal":["- add keeps working (kept) | request: add | problem: wrong implementation \\"sub dropped from add\\" also fails scenario \\"Sub keeps working\\", which exercises another property and not this one","- sub keeps working (kept) | request: sub | problem: wrong implementation \\"sub removed\\" also fails scenario \\"Add keeps working\\", which exercises another property and not this one"],"runs":3}}');

// good is an observation that every judgment of the feature accepts.
function good(all) {
  const o = {};
  for (const c of all) o[c.name] = {...(o[c.name] || {}), ...c.want};
  return o;
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

// mutants are variants of a JSON value, each with one leaf changed or one key or element dropped.
function mutants(v) {
  const out = [];
  if (Array.isArray(v)) {
    v.forEach((x, i) => {
      out.push(v.filter((_, j) => j !== i));
      for (const m of mutants(x)) out.push(v.map((y, j) => (j === i ? m : y)));
    });
    out.push([...v, 'x']);
  } else if (v !== null && typeof v === 'object') {
    for (const k of Object.keys(v)) {
      const without = {...v}; delete without[k];
      out.push(without);
      for (const m of mutants(v[k])) out.push({...v, [k]: m});
    }
  } else if (typeof v === 'boolean') out.push(!v);
  else if (typeof v === 'number') out.push(v + 1);
  else out.push(v === '' ? 'x' : '');
  return out;
}

function control(which) {
  const all = checks();
  const want = good(all);
  const run = extra => ({status: 0, signal: null, error: null, output: RUN, json: JSON.stringify(want), command: 'recorded', ...extra});
  const every = o => { for (const c of all) c.run(o); };
  if (which === 'good') {
    every(judge(run()));
    return 'accepted';
  }
  rejects(() => judge(run({status: 1})), 'a failed run');
  rejects(() => judge(run({error: 'spawnSync ETIMEDOUT'})), 'a run that did not finish');
  rejects(() => judge(run({output: RUN.replace('--- PASS', '--- FAIL')})), 'a failed harness test');
  rejects(() => judge(run({output: RUN.replace('--- PASS', '--- SKIP')})), 'a skipped harness test');
  rejects(() => judge(run({output: RUN + RUN})), 'two harness runs');
  rejects(() => judge(run({output: 'PASS\n'})), 'a run that ran no harness test');
  rejects(() => judge(run({json: null})), 'a run without an observation');
  const obs = judge(run());
  rejects(() => every(RED), "the unchanged tree's observation");
  for (const name of ['clean', 'outline', 'target-scenario-only', 'earlier', 'passes']) {
    rejects(() => every({...obs, [name]: RED[name]}), `the unchanged tree's case "${name}"`);
  }
  for (const name of Object.keys(obs)) {
    const missing = {...obs}; delete missing[name];
    rejects(() => every(missing), `a missing case "${name}"`);
    for (const key of Object.keys(obs[name])) {
      for (const m of mutants(obs[name][key])) {
        rejects(() => every({...obs, [name]: {...obs[name], [key]: m}}), `a changed ${key} of "${name}": ${JSON.stringify(m)}`);
      }
      const without = {...obs[name]}; delete without[key];
      rejects(() => every({...obs, [name]: without}), `a missing ${key} of "${name}"`);
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
  const {Given, Then} = require('@cucumber/cucumber');
  let cached = null;
  Given(P + 'the control qualification harness has run', {timeout: 700000}, function () {
    if (cached === null) cached = harness();
    this.kscmw = cached;
  });
  Then(P + 'the observation of {string} includes:', function (name, text) {
    include(this.kscmw, name, JSON.parse(text));
  });
}
