'use strict';
// Acceptance of k-shape-decide-non-ux-general. The step overlays the harness
// features/steps/k_shape_decide_general_test.go.txt into ./shape as an internal test file, builds
// the shape test binary once (go test -c -vet=off -overlay) and runs the harness test
// TestKShapeDecideGeneralAcceptance once per mode: decides, asks-ui and environment drive the
// native shaper against the package's scripted local provider, external drives Shape with its
// scripted Claude Code shaper. The fixture is an invented orchard frost alarm; no real provider,
// account or benchmark content is involved. The harness prints one KSDG observation line; this
// file compares it with the feature's expected values. Two more steps check the checkout itself:
// the landed acceptance files are gone and no other file carries the words of their fixtures.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKShapeDecideGeneralAcceptance';
const HARNESS = 'features/steps/k_shape_decide_general_test.go.txt';
const TARGET = 'shape/k_shape_decide_general_acceptance_test.go';
const PREFIX = 'KSDG ';
const FEATURE = '.kogen/intents/k-shape-decide-non-ux-general/acceptance.feature';
const RECORDED = 'features/steps/testdata/k_shape_decide_general/recorded.json';

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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ksdg-'));
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
  const r = spawnSync(bin, args, {cwd: path.join(process.cwd(), 'shape'), env: env({KSDG_MODE: mode}), encoding: 'utf8', timeout: 90000, maxBuffer: 16 * 1024 * 1024});
  return {status: r.status, signal: r.signal, error: r.error ? r.error.message : null, stdout: r.stdout || '', stderr: r.stderr || '',
    command: `KSDG_MODE=${mode} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`};
}

// parse is the observation of one harness run: the run must have passed the harness test once
// and printed exactly one observation line of the mode.
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
  assert.equal(obs[key], text, `${TEST} (${obs.mode}) observation ${key} is not exactly the expected text: ${JSON.stringify(obs[key])}`);
}

// ---- the checkout itself ----

function lines(text) {
  return text.split('\n').map(l => l.trim()).filter(l => l !== '');
}

function absent(root, list) {
  const present = lines(list).filter(p => fs.existsSync(path.join(root, p)));
  assert.deepEqual(present, [], `these files of the landed k-shape-decide-non-ux acceptance still exist: ${present.join(', ')}`);
}

// The files the word scan skips: Git's own, installed packages, Kogen's local state, and this
// Intent's own contract and step file, which quote the words.
const SKIPPED = ['.git', '.kogen/local', '.kogen/intents/k-shape-decide-non-ux-general', 'features/steps/k_shape_decide_general_steps.js'];

function mentions(root, list) {
  const words = lines(list).map(w => w.toLowerCase());
  const found = [];
  const walk = dir => {
    for (const e of fs.readdirSync(path.join(root, dir), {withFileTypes: true})) {
      const rel = dir ? dir + '/' + e.name : e.name;
      if (e.name === 'node_modules' || SKIPPED.includes(rel)) continue;
      if (e.isDirectory()) walk(rel);
      else if (e.isFile()) {
        const text = fs.readFileSync(path.join(root, rel), 'utf8').toLowerCase();
        for (const w of words) if (text.includes(w)) found.push(`${rel}: ${w}`);
      }
    }
  };
  walk('');
  assert.deepEqual(found, [], `files outside this Intent's acceptance mention the landed fixtures' words: ${found.join('; ')}`);
}

// ---- observer controls: node features/steps/k_shape_decide_general_steps.js --control good|bad ----

// The scenarios of the feature: for each, its mode and its Then steps with their doc strings.
function scenarios() {
  const text = fs.readFileSync(path.join(__dirname, '../..', FEATURE), 'utf8').split('\n');
  const out = [];
  let cur = null;
  for (let i = 0; i < text.length; i++) {
    const line = text[i].trim();
    let m;
    if (line.startsWith('Scenario:')) {
      cur = {name: line, mode: null, steps: []};
      out.push(cur);
    } else if ((m = /^When the neutral decide harness shapes in mode "([^"]+)"$/.exec(line))) {
      cur.mode = m[1];
    } else if (/^(Then|And) /.test(line)) {
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

// check runs a Then step of the feature against an observation (or a checkout root).
function check(step, obs, root) {
  let m;
  if (step.text === 'the neutral decide observations include:') return include(obs, JSON.parse(step.doc));
  if ((m = /^the neutral decide observation "([^"]+)" is exactly:$/.exec(step.text))) return exact(obs, m[1], step.doc);
  if ((m = /^the neutral decide observation "([^"]+)" is exactly these lines:$/.exec(step.text))) return exact(obs, m[1], step.doc + '\n');
  if (step.text === 'these files of the landed k-shape-decide-non-ux acceptance do not exist:') return absent(root, step.doc);
  if (step.text === "no repository file outside this Intent's acceptance mentions any of these words:") return mentions(root, step.doc);
  throw new Error('unknown step: ' + step.text);
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function tree(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ksdg-tree-'));
  owned.push(root);
  for (const [p, c] of Object.entries(files)) {
    fs.mkdirSync(path.dirname(path.join(root, p)), {recursive: true});
    fs.writeFileSync(path.join(root, p), c);
  }
  return root;
}

function control(which) {
  const recorded = JSON.parse(fs.readFileSync(path.join(__dirname, '../..', RECORDED), 'utf8'));
  const all = scenarios();
  const harness = all.filter(s => s.mode);
  const checkout = all.filter(s => !s.mode);
  assert.deepEqual(harness.map(s => s.mode), ['decides', 'asks-ui', 'environment', 'external'], 'the harness scenarios of the feature');
  assert.equal(checkout.length, 1, 'the checkout scenario of the feature');
  // A recorded run is a real harness run's stdout: its lines around the observation line, and
  // that line's observation as decoded.
  const record = (tree, mode) => {
    const r = recorded[tree][mode];
    const stdout = [...r.head, PREFIX + JSON.stringify(r.observation), ...r.tail, ''].join('\n');
    return {status: 0, signal: null, error: null, stderr: '', command: 'recorded ' + tree, stdout};
  };
  // A clean checkout: neutral files, and the words only where the scan skips them.
  const clean = {'shape/a.go': 'package shape\n', 'features/steps/k_shape_decide_general_steps.js': 'comped\n',
    '.kogen/intents/k-shape-decide-non-ux-general/intent.md': 'billing\n', '.kogen/local/shaping/x/request.md': 'Stripe\n',
    'testdata/tools/cucumber/node_modules/x.js': 'reaction\n', '.git/HEAD': 'billing\n'};
  if (which === 'good') {
    for (const s of harness) {
      const obs = parse(record('green', s.mode), s.mode);
      for (const step of s.steps) check(step, obs);
    }
    for (const step of checkout[0].steps) check(step, null, tree(clean));
    return 'accepted';
  }
  for (const s of harness) {
    const rejectsAll = label => obs => rejects(() => { for (const step of s.steps) check(step, obs); }, `${label} (${s.mode})`);
    if (s.mode !== 'asks-ui') rejectsAll('the unchanged tree\'s observation')(parse(record('red', s.mode), s.mode));
    const green = record('green', s.mode);
    rejects(() => parse({...green, status: 1}, s.mode), 'a failed run');
    rejects(() => parse({...green, stdout: green.stdout.replace(/^KSDG .*\n/m, '')}, s.mode), 'a run without an observation');
    rejects(() => parse({...green, stdout: green.stdout.replace('--- PASS', '--- FAIL')}, s.mode), 'a failed harness test');
    rejects(() => parse(green, s.mode === 'decides' ? 'external' : 'decides'), 'the observation of another mode');
    const obs = parse(green, s.mode);
    for (const step of s.steps) {
      if (step.text === 'the neutral decide observations include:') {
        const want = JSON.parse(step.doc);
        for (const [key, value] of Object.entries(want)) {
          const wrong = typeof value === 'boolean' ? !value : typeof value === 'number' ? value + 1 : typeof value === 'string' ? value + 'x'
            : Array.isArray(value) ? [...value, 'x'] : value === null ? 'x' : {...value, extra: 1};
          rejects(() => check(step, {...obs, [key]: wrong}), 'a wrong ' + key);
          const missing = {...obs}; delete missing[key];
          rejects(() => check(step, missing), 'a missing ' + key);
        }
      } else {
        const key = /"([^"]+)"/.exec(step.text)[1];
        const text = obs[key];
        rejects(() => check(step, {...obs, [key]: text + '\n'}), 'a trailing newline in ' + key);
        rejects(() => check(step, {...obs, [key]: text.slice(0, -1)}), 'a text missing its last character in ' + key);
        rejects(() => check(step, {...obs, [key]: text.split('\n').slice(1).join('\n')}), 'a missing first line in ' + key);
        const missing = {...obs}; delete missing[key];
        rejects(() => check(step, missing), 'a missing ' + key);
      }
    }
  }
  const [gone, words] = checkout[0].steps;
  rejects(() => check(gone, null, tree({...clean, 'features/steps/k_shape_decide_non_ux_test.go.txt': 'package shape\n'})), 'a landed harness that still exists');
  rejects(() => check(gone, null, tree({...clean, '.kogen/intents/k-shape-decide-non-ux/acceptance.feature': 'Feature: x\n'})), 'a landed feature that still exists');
  rejects(() => check(words, null, tree({...clean, 'shape/fixture_test.go': '// Monthly Billing\n'})), 'a file mentioning a word in another case');
  rejects(() => check(words, null, tree({...clean, 'features/steps/other_steps.js': '// comped\n'})), 'another step file mentioning a word');
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
  When('the neutral decide harness shapes in mode {string}', {timeout: 200000}, function (mode) {
    this.ksdg = parse(run(mode), mode);
  });
  Then('the neutral decide observations include:', function (text) {
    include(this.ksdg, JSON.parse(text));
  });
  Then('the neutral decide observation {string} is exactly:', function (key, text) {
    exact(this.ksdg, key, text);
  });
  Then('the neutral decide observation {string} is exactly these lines:', function (key, text) {
    exact(this.ksdg, key, text + '\n');
  });
  Then('these files of the landed k-shape-decide-non-ux acceptance do not exist:', function (text) {
    absent(process.cwd(), text);
  });
  Then("no repository file outside this Intent's acceptance mentions any of these words:", function (text) {
    mentions(process.cwd(), text);
  });
}
