'use strict';
// Acceptance of fix-k-shape-environment-stop-evidence. The When step compiles the shape package's
// tests with the harness of fix-k-shape-environment-stop-evidence__harness.js (through a Go
// overlay), runs TestFixKShapeEnvironmentStopEvidenceAcceptance once and reads the JSON
// observation the harness wrote to FKSESE_OUT. The Then step compares it with the feature's
// values. Run this file directly with --control good or --control bad to check the judgments on
// known observations.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestFixKShapeEnvironmentStopEvidenceAcceptance';
const TARGET = 'shape/fix_k_shape_environment_stop_evidence_acceptance_test.go';
const FEATURE = '.kogen/intents/fix-k-shape-environment-stop-evidence/acceptance.feature';
const P = 'fix-k-shape-environment-stop-evidence: ';

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

// shape builds the shape tests with the harness and runs the harness test once.
function shape() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'fksese-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./fix-k-shape-environment-stop-evidence__harness.js'));
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
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({FKSESE_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  return judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `FKSESE_OUT=${out} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`});
}

// include checks that each listed key of the observation has exactly the listed value.
function include(o, want) {
  const summary = JSON.stringify(o, null, 1);
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(o, key), `the observation has no ${key}:\n${summary}`);
    assert.deepEqual(o[key], value, `${key} is ${JSON.stringify(o[key])}, want ${JSON.stringify(value)}\nobservation:\n${summary}`);
  }
}

// ---- observer controls: node features/steps/fix-k-shape-environment-stop-evidence__steps.js --control good|bad ----

// expected is the JSON doc string of the feature's include step.
function expected() {
  const lines = fs.readFileSync(path.join(__dirname, '../..', FEATURE), 'utf8').split('\n');
  const at = lines.findIndex(l => l.trim() === 'Then ' + P + 'the observations include:');
  assert.ok(at >= 0 && lines[at + 1].trim() === '"""', 'the feature has the include step with a doc string');
  const end = lines.findIndex((l, i) => i > at + 1 && l.trim() === '"""');
  return JSON.parse(lines.slice(at + 2, end).join('\n'));
}

// RUN is a real harness run's -test.v output on the unchanged tree.
const RUN = `=== RUN   ${TEST}\n=== PAUSE ${TEST}\n=== CONT  ${TEST}\n--- PASS: ${TEST} (1.07s)\nPASS\n`;

// RED is the unchanged tree's observed cited evidence: the outside file's text is shown.
const RED = 'Question 1 cites lib/weather.js (read from the checkout):\n1\t// The weather client. No fake client exists: every call reaches the live service.\n2\texports.client = function () {}\n\n' +
  'Question 1 cites lib/secret.js (read from the checkout):\n1\tOUTSIDE-SECRET-7f3a: this file lies outside the checkout';

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function control(which) {
  const want = expected();
  const run = extra => ({status: 0, signal: null, error: null, output: RUN, json: JSON.stringify({...want}), command: 'recorded', ...extra});
  if (which === 'good') {
    include(judge(run()), want);
    return 'accepted';
  }
  rejects(() => judge(run({status: 1})), 'a failed run');
  rejects(() => judge(run({error: 'spawnSync ETIMEDOUT'})), 'a run that did not finish');
  rejects(() => judge(run({output: RUN.replace('--- PASS', '--- FAIL')})), 'a failed harness test');
  rejects(() => judge(run({output: RUN + RUN})), 'two harness runs');
  rejects(() => judge(run({output: 'PASS\n'})), 'a run that ran no harness test');
  rejects(() => judge(run({json: null})), 'a run without an observation');
  const obs = judge(run());
  rejects(() => include({...obs, cited_evidence: RED, secret_sent: true}, want), "the unchanged tree's observation");
  rejects(() => include({...obs, cited_evidence: RED}, want), 'the outside text in the cited evidence');
  rejects(() => include({...obs, secret_sent: true}, want), 'the outside text sent elsewhere');
  for (const [key, value] of Object.entries(want)) {
    const wrong = typeof value === 'boolean' ? !value : typeof value === 'number' ? value + 1 : value + 'x';
    rejects(() => include({...obs, [key]: wrong}, want), `a wrong ${key}`);
    const missing = {...obs}; delete missing[key];
    rejects(() => include(missing, want), `a missing ${key}`);
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
  When('fix-k-shape-environment-stop-evidence: the external shaper asks a question citing a tracked file and a tracked symlink to a file outside the checkout', {timeout: 660000}, function () {
    this.fksese = shape();
  });
  Then('fix-k-shape-environment-stop-evidence: the observations include:', function (text) {
    include(this.fksese, JSON.parse(text));
  });
}
