'use strict';
// Acceptance of k-models-file-shaping. kogen-coordinator's K-MODELS-FILE change is not on its main
// yet, so the step builds it: a copy of the pinned kogen-coordinator module with
// features/steps/testdata/k_models_file_shaping/coordinator.patch applied, used through a temporary
// go.work replace (GOWORK=<temp>/go.work). The "pinned" coordinator is go.mod's own (GOWORK=off).
// For each coordinator it builds the shape test binary once (go test -c -vet=off -overlay ./shape)
// with the harness features/steps/k_models_file_shaping_test.go.txt overlaid as an internal test
// file, then runs named existing shape tests or the harness test in it. Nothing in the checkout is
// written. The harness prints one KMFS observation line; this file compares it with the feature's
// expected values.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestKModelsFileShapingAcceptance';
const HARNESS = 'features/steps/k_models_file_shaping_test.go.txt';
const TARGET = 'shape/k_models_file_shaping_acceptance_test.go';
const PATCH = 'features/steps/testdata/k_models_file_shaping/coordinator.patch';
const COORDINATOR = 'github.com/KogenAI/kogen-coordinator';
const PREFIX = 'KMFS ';
const KINDS = ['pinned', 'K-MODELS-FILE'];

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_') && k !== 'GOWORK') out[k] = v;
  return {...out, ...extra};
}

function spawn(command, args, options) {
  const r = spawnSync(command, args, {encoding: 'utf8', maxBuffer: 16 * 1024 * 1024, ...options});
  return {status: r.status, signal: r.signal, error: r.error ? r.error.message : null, stdout: r.stdout || '', stderr: r.stderr || '',
    command: [command, ...args].join(' ')};
}

function succeeded(r, what) {
  assert.ok(r.error === null && r.status === 0,
    `${what}: ${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.stdout}${r.stderr}`);
}

// workspace is the go.work that replaces the pinned coordinator with its K-MODELS-FILE copy.
function workspace(root, dir) {
  const located = spawn('go', ['list', '-m', '-json', COORDINATOR], {cwd: root, env: env({GOWORK: 'off', PWD: root}), timeout: 30000});
  succeeded(located, 'locating the pinned coordinator module');
  const module = JSON.parse(located.stdout);
  assert.ok(module.Dir, `the pinned coordinator module has no source directory: ${located.stdout}`);
  const coordinator = path.join(dir, 'coordinator');
  succeeded(spawn('cp', ['-R', module.Dir, coordinator], {timeout: 30000}), 'copying the pinned coordinator module');
  succeeded(spawn('chmod', ['-R', 'u+w', coordinator], {timeout: 30000}), 'making the coordinator copy writable');
  succeeded(spawn('git', ['apply', '--ignore-whitespace', path.join(root, PATCH)], {cwd: coordinator, env: env({GIT_CEILING_DIRECTORIES: dir}), timeout: 30000}),
    'applying the K-MODELS-FILE change to the coordinator copy');
  const version = /^go (\S+)$/m.exec(fs.readFileSync(path.join(root, 'go.mod'), 'utf8'));
  assert.ok(version, 'go.mod has no go directive');
  const work = path.join(dir, 'go.work');
  fs.writeFileSync(work, `go ${version[1]}\n\nuse ${JSON.stringify(root)}\n\nreplace ${COORDINATOR} => ${JSON.stringify(coordinator)}\n`);
  return work;
}

const builds = {};
function binary(kind) {
  assert.ok(KINDS.includes(kind), `unknown coordinator ${kind}; use one of ${KINDS.join(', ')}`);
  const b = builds[kind] || (builds[kind] = {});
  if (b.failure) throw b.failure; // one build per coordinator and run: later steps fail fast with its failure
  if (b.bin) return b;
  try {
    // The overlay names its paths relative to the directory go runs in, so a checkout reached
    // through a symlink (such as /tmp) still matches the package directory go resolves.
    const root = process.cwd();
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kmfs-'));
    owned.push(dir);
    const gowork = kind === 'pinned' ? 'off' : workspace(root, dir);
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {[TARGET]: HARNESS}}));
    const bin = path.join(dir, 'shape.test');
    // go takes its working directory from PWD when PWD names the same directory, so PWD is the
    // path the go.work use line and the overlay are written against.
    const r = spawn('go', ['test', '-c', '-vet=off', '-o', bin, '-overlay', overlay, './shape'], {cwd: root, env: env({GOWORK: gowork, PWD: root}), timeout: 60000});
    succeeded(r, `building the shape test binary against the ${kind} coordinator (GOWORK=${gowork})`);
    b.bin = bin;
    b.how = `built with GOWORK=${gowork} go test -c -vet=off -overlay ./shape`;
    return b;
  } catch (e) {
    b.failure = e;
    throw e;
  }
}

function run(kind, pattern) {
  const b = binary(kind);
  const args = ['-test.run', pattern, '-test.count=1', '-test.v', '-test.timeout=60s'];
  const r = spawn(b.bin, args, {cwd: path.join(process.cwd(), 'shape'), env: env({}), timeout: 80000});
  return {...r, command: `${kind} coordinator: shape.test ${args.join(' ')} (${b.how})`};
}

function tail(text) {
  return text.length > 12000 ? '...\n' + text.slice(-12000) : text;
}

// passedEach: every named test passed exactly once and the run passed.
function passedEach(r, kind, names) {
  const out = r.stdout + r.stderr;
  const shown = `${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${tail(out)}`;
  assert.ok(names.length > 0, 'no shape tests named');
  for (const name of names) {
    const lines = out.split('\n');
    const passed = lines.filter(l => l.startsWith(`--- PASS: ${name} (`)).length;
    const failed = lines.filter(l => l.startsWith(`--- FAIL: ${name} (`)).length;
    const outcome = failed > 0 ? 'it failed' : passed === 0 ? 'it is absent: it did not run, was not found or did not finish' : `it passed ${passed} times`;
    assert.ok(passed === 1 && failed === 0, `required shape test ${name} did not pass exactly once against the ${kind} coordinator (${outcome}); ${shown}`);
  }
  assert.ok(r.error === null && r.status === 0, `the run of ${names.join(', ')} against the ${kind} coordinator did not pass; ${shown}`);
}

// parse is the observation of one harness run: the run passed the harness test once and printed
// exactly one observation line.
function parse(r, kind) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${tail(r.stdout + r.stderr)}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  assert.equal(r.stdout.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' (')).length, 1, `${TEST} did not pass exactly once: ${shown}`);
  const lines = r.stdout.split('\n').filter(l => l.startsWith(PREFIX));
  assert.equal(lines.length, 1, `${TEST} printed ${lines.length} observation lines, want 1: ${shown}`);
  return {kind, obs: JSON.parse(lines[0].slice(PREFIX.length))};
}

function include(observed, want) {
  const {kind, obs} = observed;
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(obs, key), `${TEST} (${kind} coordinator) observation has no ${key}: ${JSON.stringify(obs)}`);
    assert.deepEqual(obs[key], value, `${TEST} (${kind} coordinator) observation ${key}: ${JSON.stringify(obs[key])}, want ${JSON.stringify(value)}; observation ${JSON.stringify(obs)}`);
  }
}

// ---- observer controls: node features/steps/k_models_file_shaping_steps.js --control good|bad ----

// Real runs of these steps' commands, recorded verbatim in scratch copies outside the checkout
// as <key>.stdout and <key>.stderr (absent: empty) under RECORDED, with their exit statuses:
// pinned_landing and harness_pinned on the unchanged tree with the pinned coordinator,
// models_landing_red and harness_models_red on the unchanged tree with the K-MODELS-FILE
// coordinator, models_layout and harness_models on a prototype of this Intent with it.
const RECORDED = path.join(__dirname, 'testdata/k_models_file_shaping/recorded');
const STATUS = {pinned_landing: 0, models_landing_red: 2, models_layout: 0, harness_pinned: 0, harness_models_red: 0, harness_models: 0};

// The feature's expected values, read from the feature itself.
function expectations() {
  const text = fs.readFileSync(path.join(__dirname, '../../.kogen/intents/k-models-file-shaping/acceptance.feature'), 'utf8');
  const lists = [];
  let current = null;
  for (const line of text.split('\n')) {
    const t = line.trim();
    const when = /^When the shape tests run against the "([^"]+)" coordinator:$/.exec(t);
    if (when) { current = {kind: when[1], names: []}; lists.push(current); continue; }
    if (current && t.startsWith('|')) {
      const cell = t.slice(1, -1).trim();
      if (cell !== 'test') current.names.push(cell);
      continue;
    }
    current = null;
  }
  const docs = [];
  const re = /\n( *)"""\n([\s\S]*?)\n\1"""/g;
  let m;
  while ((m = re.exec(text))) docs.push(JSON.parse(m[2].split('\n').map(l => l.slice(m[1].length)).join('\n')));
  return {lists, docs};
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function recorded(key) {
  const read = ext => {
    const file = path.join(RECORDED, key + ext);
    return fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : '';
  };
  return {status: STATUS[key], signal: null, error: null, stdout: read('.stdout'), stderr: read('.stderr'), command: 'recorded ' + key};
}

function control(which) {
  const {lists, docs} = expectations();
  assert.deepEqual(lists.map(l => l.kind), ['pinned', 'K-MODELS-FILE', 'K-MODELS-FILE'], 'the feature\'s test lists');
  assert.equal(docs.length, 2, 'the feature\'s observation blocks');
  const [landingPinned, landingModels, layoutModels] = lists;
  const [wantPinned, wantModels] = docs;
  if (which === 'good') {
    passedEach(recorded('pinned_landing'), 'pinned', landingPinned.names);
    passedEach(recorded('pinned_landing'), 'K-MODELS-FILE', landingModels.names);
    passedEach(recorded('models_layout'), 'K-MODELS-FILE', layoutModels.names);
    include(parse(recorded('harness_pinned'), 'pinned'), wantPinned);
    include(parse(recorded('harness_models'), 'K-MODELS-FILE'), wantModels);
    return 'accepted';
  }
  rejects(() => passedEach(recorded('models_landing_red'), 'K-MODELS-FILE', landingModels.names), 'the unchanged tree\'s landing tests against the K-MODELS-FILE coordinator');
  rejects(() => passedEach({...recorded('pinned_landing'), status: 1}, 'pinned', landingPinned.names), 'a failed run');
  rejects(() => passedEach(recorded('pinned_landing'), 'pinned', [...landingPinned.names, 'TestKMFSNotThere']), 'an absent test');
  rejects(() => passedEach(recorded('pinned_landing'), 'pinned', []), 'an empty list');
  for (const name of landingPinned.names) {
    const r = recorded('pinned_landing');
    rejects(() => passedEach({...r, stdout: r.stdout.replace(`--- PASS: ${name} (`, `--- FAIL: ${name} (`)}, 'pinned', landingPinned.names), `a failed ${name}`);
    rejects(() => passedEach({...r, stdout: r.stdout.replace(`--- PASS: ${name} (`, `--- PASS: ${name} (0.00s)\n--- PASS: ${name} (`)}, 'pinned', landingPinned.names), `${name} passing twice`);
  }
  rejects(() => include(parse(recorded('harness_models_red'), 'K-MODELS-FILE'), wantModels), 'the unchanged tree\'s observation against the K-MODELS-FILE coordinator');
  rejects(() => include(parse(recorded('harness_pinned'), 'pinned'), wantModels), 'the pinned coordinator\'s observation as the K-MODELS-FILE one');
  rejects(() => include(parse(recorded('harness_models'), 'K-MODELS-FILE'), wantPinned), 'the K-MODELS-FILE observation as the pinned one');
  const h = recorded('harness_models');
  rejects(() => parse({...h, status: 1}, 'K-MODELS-FILE'), 'a failed harness run');
  rejects(() => parse({...h, stdout: h.stdout.replace(/^KMFS .*\n/m, '')}, 'K-MODELS-FILE'), 'a run without an observation');
  rejects(() => parse({...h, stdout: h.stdout.replace(/^(KMFS .*\n)/m, '$1$1')}, 'K-MODELS-FILE'), 'two observations');
  rejects(() => parse({...h, stdout: h.stdout.replace('--- PASS', '--- FAIL')}, 'K-MODELS-FILE'), 'a failed harness test');
  for (const want of docs) {
    for (const [key, value] of Object.entries(want)) {
      const wrong = typeof value === 'number' ? value + 1 : typeof value === 'string' ? value + 'x' : 'x';
      rejects(() => include({kind: 'x', obs: {...want, [key]: wrong}}, want), 'a wrong ' + key);
      const missing = {...want};
      delete missing[key];
      rejects(() => include({kind: 'x', obs: missing}, want), 'a missing ' + key);
    }
  }
  return 'rejected';
}

if (require.main === module) {
  const which = process.argv[2] === '--control' ? process.argv[3] : '';
  assert.ok(which === 'good' || which === 'bad', 'usage: --control good|bad');
  let outcome;
  try {
    outcome = control(which);
  } catch (e) {
    console.error(e.message);
    outcome = 'error';
  }
  console.log(outcome);
  process.exitCode = outcome === 'accepted' ? 0 : outcome === 'rejected' ? 1 : 2;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('the shape tests run against the {string} coordinator:', {timeout: 150000}, function (kind, table) {
    const names = table.hashes().map(row => row.test);
    this.kmfsTests = {kind, names, run: run(kind, '^(' + names.join('|') + ')$')};
  });
  Then('each of those shape tests passed exactly once', function () {
    assert.ok(this.kmfsTests, 'no shape tests ran');
    passedEach(this.kmfsTests.run, this.kmfsTests.kind, this.kmfsTests.names);
  });
  When('the models-file harness shapes against the {string} coordinator', {timeout: 150000}, function (kind) {
    this.kmfs = parse(run(kind, '^' + TEST + '$'), kind);
  });
  Then('the models-file observation includes:', function (text) {
    include(this.kmfs, JSON.parse(text));
  });
}
