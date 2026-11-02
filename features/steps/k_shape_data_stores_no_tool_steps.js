'use strict';
// Acceptance of k-shape-data-stores-no-tool. The step overlays the harness
// features/steps/k_shape_data_stores_no_tool_test.go.txt into ./shape as an internal test file,
// builds the shape test binary once (go test -c -vet=off -overlay) and runs the harness test
// TestKShapeDataStoresNoToolAcceptance once per mode: plain and stores drive the native shaper
// against the package's scripted local provider, external drives Shape with its scripted Claude
// Code shaper. No real provider, account or benchmark content is involved. The harness prints one
// KSDSNT observation line; this file compares it with the feature's expected values.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKShapeDataStoresNoToolAcceptance';
const HARNESS = 'features/steps/k_shape_data_stores_no_tool_test.go.txt';
const TARGET = 'shape/k_shape_data_stores_no_tool_acceptance_test.go';
const PREFIX = 'KSDSNT ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let built, buildFailure;
function binary() {
  if (built) return built;
  if (buildFailure) throw buildFailure; // one build per run: later scenarios fail fast with its failure
  // The overlay names its paths relative to the directory go runs in, so a checkout reached
  // through a symlink (such as /tmp) still matches the package directory go resolves.
  const root = process.cwd();
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ksdsnt-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[TARGET]: HARNESS}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-o', bin, '-overlay', overlay, './shape'];
  const r = spawnSync('go', args, {cwd: root, env: env({}), encoding: 'utf8', timeout: 100000, maxBuffer: 16 * 1024 * 1024});
  try {
    assert.equal(r.status, 0, `building the harness of ${TEST}: go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout}${r.stderr}`);
  } catch (e) {
    buildFailure = e;
    throw e;
  }
  built = bin;
  return bin;
}

function run(mode) {
  const bin = binary();
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=60s'];
  const r = spawnSync(bin, args, {cwd: path.join(process.cwd(), 'shape'), env: env({KSDSNT_MODE: mode}), encoding: 'utf8', timeout: 90000, maxBuffer: 16 * 1024 * 1024});
  return {status: r.status, signal: r.signal, error: r.error ? r.error.message : null, stdout: r.stdout || '', stderr: r.stderr || '',
    command: `KSDSNT_MODE=${mode} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`};
}

// parse is the observation of one harness run: the run must have passed the harness test once
// and printed exactly one observation line.
function parse(r, mode) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.stdout}${r.stderr}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  assert.equal(r.stdout.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length, 1, `${TEST} did not pass exactly once: ${shown}`);
  const lines = r.stdout.split('\n').filter(l => l.startsWith(PREFIX));
  assert.equal(lines.length, 1, `${TEST} printed ${lines.length} observation lines, want 1: ${shown}`);
  const obs = JSON.parse(lines[0].slice(PREFIX.length));
  assert.equal(obs.mode, mode, `${TEST} observed mode ${obs.mode}, want ${mode}`);
  return obs;
}

function include(obs, want) {
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(obs, key), `${TEST} (${obs.mode}) observation has no ${key}: ${JSON.stringify(obs)}`);
    assert.deepEqual(obs[key], value, `${TEST} (${obs.mode}) observation ${key}: ${JSON.stringify(obs[key])}, want ${JSON.stringify(value)}`);
  }
}

function exact(obs, key, text) {
  assert.ok(Object.hasOwn(obs, key), `${TEST} (${obs.mode}) observation has no ${key}: ${JSON.stringify(obs)}`);
  assert.equal(obs[key], text, `${TEST} (${obs.mode}) observation ${key} is not exactly the expected text`);
}

// ---- observer controls: node features/steps/k_shape_data_stores_no_tool_steps.js --control good|bad ----

// A real harness run's output on the unchanged tree (plain mode), recorded verbatim.
const REAL_PLAIN = {status: 0, signal: null, error: null, stderr: '', command: 'recorded', stdout:
  '=== RUN   TestKShapeDataStoresNoToolAcceptance\n=== PAUSE TestKShapeDataStoresNoToolAcceptance\n=== CONT  TestKShapeDataStoresNoToolAcceptance\n' +
  'KSDSNT {"acceptance_runs":2,"data_stores_tool":null,"error":"","exit":0,"instructions_mention_data_stores":false,"instructions_sha256":"48a1bf625e1b71593f687d64b7a6d85c1c30749180462b97917e83084d1014c4","last_stdout_line":"Next: kogen intent approve is-even","mode":"plain","no_tool_turns":0,"published":true,"refused_gates":[],"shaper_requests":1,"tools":"read_file, view_image, list_files, search_text, probe, documentation, propose_files, submit"}\n' +
  '--- PASS: TestKShapeDataStoresNoToolAcceptance (1.33s)\nPASS\n'};

// The feature's expected values, read from the feature itself.
function expectations() {
  const text = fs.readFileSync(path.join(__dirname, '../../.kogen/intents/k-shape-data-stores-no-tool/acceptance.feature'), 'utf8');
  const docs = [];
  const re = /\n( *)"""\n([\s\S]*?)\n\1"""/g;
  let m;
  while ((m = re.exec(text))) docs.push(m[2].split('\n').map(l => l.slice(m[1].length)).join('\n'));
  return docs;
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function control(which) {
  const docs = expectations();
  const jsons = docs.filter(d => d.startsWith('{')).map(d => JSON.parse(d));
  const texts = docs.filter(d => !d.startsWith('{'));
  assert.equal(jsons.length, 3, 'expected observation blocks in the feature');
  assert.equal(texts.length, 4, 'expected exact text blocks in the feature');
  const real = parse(REAL_PLAIN, 'plain');
  const known = {...real, ...jsons[0], mode: 'plain'};
  if (which === 'good') {
    for (const key of Object.keys(jsons[0])) assert.ok(Object.hasOwn(real, key), 'the real harness output lacks ' + key);
    include(known, jsons[0]);
    for (const want of jsons.slice(1)) include({...want, mode: 'x'}, want);
    for (const t of texts) exact({mode: 'x', v: t}, 'v', t);
    return 'accepted';
  }
  rejects(() => parse({...REAL_PLAIN, status: 1}, 'plain'), 'a failed run');
  rejects(() => parse({...REAL_PLAIN, stdout: REAL_PLAIN.stdout.replace(/^KSDSNT .*\n/m, '')}, 'plain'), 'a run without an observation');
  rejects(() => parse({...REAL_PLAIN, stdout: REAL_PLAIN.stdout.replace('--- PASS', '--- FAIL')}, 'plain'), 'a failed harness test');
  rejects(() => parse(REAL_PLAIN, 'stores'), 'the observation of another mode');
  rejects(() => include(real, jsons[0]), 'the unchanged tree\'s plain observation');
  for (const want of jsons) {
    for (const [key, value] of Object.entries(want)) {
      const wrong = typeof value === 'boolean' ? !value : typeof value === 'number' ? value + 1 : typeof value === 'string' ? value + 'x'
        : Array.isArray(value) ? [...value, 'x'] : value === null ? 'x' : {...value, extra: 1};
      rejects(() => include({...want, [key]: wrong, mode: 'x'}, want), 'a wrong ' + key);
      const missing = {...want, mode: 'x'}; delete missing[key];
      rejects(() => include(missing, want), 'a missing ' + key);
    }
  }
  for (const t of texts) {
    rejects(() => exact({mode: 'x', v: t + '\n'}, 'v', t), 'a trailing newline');
    rejects(() => exact({mode: 'x', v: t.split('\n').slice(0, -1).join('\n')}, 'v', t), 'a missing last line');
    rejects(() => exact({mode: 'x'}, 'v', t), 'a missing text');
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
  When('the data-stores harness shapes in mode {string}', {timeout: 200000}, function (mode) {
    this.ksdsnt = parse(run(mode), mode);
  });
  Then('the data-stores shaping observations include:', function (text) {
    include(this.ksdsnt, JSON.parse(text));
  });
  Then('the data-stores shaping observation {string} is exactly:', function (key, text) {
    exact(this.ksdsnt, key, text);
  });
}
