'use strict';
// Acceptance of k-shape-lint-messages. It runs named existing shape tests against three
// kogen-coordinator versions, each through its own shape test binary (go test -c -vet=off ./shape):
// - "pinned": the version go.mod requires (GOWORK=off);
// - "main": MAIN_VERSION, coordinator main's code since 8d24cc7 "Report every Intent lint problem
//   in one refusal", through a temporary go.work replace of the coordinator with that module's
//   directory in the Go module cache;
// - "mutated": a temporary copy of the main module whose unsourced-Decision lint line reads
//   MUTATED instead, through a temporary go.work replace.
// Nothing in the checkout is written.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const COORDINATOR = 'github.com/KogenAI/kogen-coordinator';
const MAIN_VERSION = 'v0.0.0-20261103113131-ffca7072fe15';
const LINT = 'add("decisions: " + d + " must end with `(source: request)` or `(source: answer <n>)`")';
const MUTATED = 'add("decisions: " + d + " is unsourced")';
const KINDS = ['pinned', 'main', 'mutated'];

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

// mainDir is the source directory of the main coordinator module, downloaded into the module cache
// when it is not there yet. It runs outside any module, so no go.mod or go.sum changes.
function mainDir(dir) {
  const r = spawn('go', ['mod', 'download', '-json', COORDINATOR + '@' + MAIN_VERSION], {cwd: dir, env: env({GOWORK: 'off', GO111MODULE: 'on', PWD: dir}), timeout: 60000});
  succeeded(r, 'locating the main coordinator module ' + MAIN_VERSION);
  const module = JSON.parse(r.stdout);
  assert.ok(module.Dir, `the main coordinator module has no source directory: ${r.stdout}`);
  return module.Dir;
}

// workspace is a go.work that replaces the coordinator with the main module or its mutated copy.
function workspace(root, dir, kind) {
  let coordinator = mainDir(dir);
  if (kind === 'mutated') {
    const copy = path.join(dir, 'coordinator');
    succeeded(spawn('cp', ['-R', coordinator, copy], {timeout: 30000}), 'copying the main coordinator module');
    succeeded(spawn('chmod', ['-R', 'u+w', copy], {timeout: 30000}), 'making the coordinator copy writable');
    const file = path.join(copy, 'project/intent.go');
    const text = fs.readFileSync(file, 'utf8');
    assert.equal(text.split(LINT).length, 2, `the main coordinator's project/intent.go does not contain ${LINT} exactly once`);
    fs.writeFileSync(file, text.replace(LINT, MUTATED));
    coordinator = copy;
  }
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
    const root = process.cwd();
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kslm-'));
    owned.push(dir);
    const gowork = kind === 'pinned' ? 'off' : workspace(root, dir, kind);
    const bin = path.join(dir, 'shape.test');
    // go takes its working directory from PWD when PWD names the same directory, so PWD is the
    // path the go.work use line is written against.
    const r = spawn('go', ['test', '-c', '-vet=off', '-o', bin, './shape'], {cwd: root, env: env({GOWORK: gowork, PWD: root}), timeout: 90000});
    succeeded(r, `building the shape test binary against the ${kind} coordinator (GOWORK=${gowork})`);
    b.bin = bin;
    b.how = `built with GOWORK=${gowork} go test -c -vet=off ./shape`;
    return b;
  } catch (e) {
    b.failure = e;
    throw e;
  }
}

// run runs the named tests (a subtest is named parent/sub) in the kind's shape test binary.
function run(kind, names) {
  const b = binary(kind);
  const tops = [...new Set(names.map(n => n.split('/')[0]))];
  const args = ['-test.run', '^(' + tops.join('|') + ')$', '-test.count=1', '-test.v', '-test.timeout=60s'];
  const r = spawn(b.bin, args, {cwd: path.join(process.cwd(), 'shape'), env: env({}), timeout: 80000});
  return {...r, command: `${kind} coordinator: shape.test ${args.join(' ')} (${b.how})`};
}

function tail(text) {
  return text.length > 12000 ? '...\n' + text.slice(-12000) : text;
}

function shown(r) {
  return `${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${tail(r.stdout + r.stderr)}`;
}

// count is how often the named test or subtest reported the outcome (PASS or FAIL).
function count(r, name, outcome) {
  return (r.stdout + r.stderr).split('\n').filter(l => l.trimStart().startsWith(`--- ${outcome}: ${name} (`)).length;
}

function outcomeOf(r, name) {
  const passed = count(r, name, 'PASS');
  const failed = count(r, name, 'FAIL');
  return failed > 0 ? `it failed ${failed} times` : passed === 0 ? 'it is absent: it did not run, was not found or did not finish' : `it passed ${passed} times`;
}

// passedEach: every named test passed exactly once and the run passed.
function passedEach(r, kind, names) {
  assert.ok(names.length > 0, 'no shape tests named');
  for (const name of names) {
    assert.ok(count(r, name, 'PASS') === 1 && count(r, name, 'FAIL') === 0,
      `required shape test ${name} did not pass exactly once against the ${kind} coordinator (${outcomeOf(r, name)}); ${shown(r)}`);
  }
  assert.ok(r.error === null && r.status === 0, `the run of ${names.join(', ')} against the ${kind} coordinator did not pass; ${shown(r)}`);
}

// failedEach: every named test failed exactly once and did not pass, no test panicked and the
// run exited 1.
function failedEach(r, kind, names) {
  assert.ok(names.length > 0, 'no shape tests named');
  for (const name of names) {
    assert.ok(count(r, name, 'FAIL') === 1 && count(r, name, 'PASS') === 0,
      `required shape test ${name} did not fail exactly once against the ${kind} coordinator (${outcomeOf(r, name)}); ${shown(r)}`);
  }
  assert.ok(!/^panic: /m.test(r.stdout + r.stderr), `a shape test panicked against the ${kind} coordinator; ${shown(r)}`);
  assert.ok(r.error === null && r.status === 1, `the run of ${names.join(', ')} against the ${kind} coordinator did not exit 1; ${shown(r)}`);
}

// pinsCoordinator: go.mod requires the coordinator at exactly version and does not replace it.
function pinsCoordinator(text, version) {
  const lines = text.split('\n').map(l => l.trim());
  const required = lines.filter(l => l.startsWith(COORDINATOR + ' ') || l.startsWith('require ' + COORDINATOR + ' '));
  assert.deepEqual(required.map(l => l.replace(/^require /, '')), [COORDINATOR + ' ' + version],
    `go.mod does not require ${COORDINATOR} ${version} exactly once; its coordinator lines: ${JSON.stringify(required)}`);
  assert.ok(!text.split('\n').some(l => /^\s*(replace\s+)?github\.com\/KogenAI\/kogen-coordinator\b.*=>/.test(l)),
    `go.mod replaces ${COORDINATOR}:\n${text}`);
}

// ---- observer controls: node features/steps/k_shape_lint_messages_steps.js --control good|bad ----

// Real runs of this file's run(kind, names) with the feature's test list, recorded verbatim as
// <key>.stdout (stderr was empty) under RECORDED, with their exit statuses: pinned and main_red on
// the unchanged tree, main and mutated on a scratch prototype of this Intent's test change.
const RECORDED = path.join(__dirname, 'testdata/k_shape_lint_messages/recorded');
const STATUS = {pinned: 0, main_red: 1, main: 0, mutated: 1};
const FEATURE = path.join(__dirname, '../../.kogen/intents/k-shape-lint-messages/acceptance.feature');

// tables are the feature's test lists, in order, with the coordinator each runs against.
function tables() {
  const lists = [];
  let current = null;
  for (const line of fs.readFileSync(FEATURE, 'utf8').split('\n')) {
    const t = line.trim();
    const when = /^When the lint-message shape tests run against the "([^"]+)" coordinator:$/.exec(t);
    if (when) { current = {kind: when[1], names: []}; lists.push(current); continue; }
    if (current && t.startsWith('|')) {
      const cell = t.slice(1, -1).trim();
      if (cell !== 'test') current.names.push(cell);
      continue;
    }
    current = null;
  }
  return lists;
}

function recorded(key) {
  return {status: STATUS[key], signal: null, error: null, stdout: fs.readFileSync(path.join(RECORDED, key + '.stdout'), 'utf8'), stderr: '',
    command: 'recorded ' + key};
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function control(which) {
  const lists = tables();
  assert.deepEqual(lists.map(l => l.kind), ['pinned', 'mutated', 'main'], 'the feature\'s test lists');
  const names = lists[0].names;
  assert.equal(names.length, 6, 'the feature\'s test list');
  for (const l of lists) assert.deepEqual(l.names, names, `the ${l.kind} test list`);
  const goMod = fs.readFileSync(path.join(process.cwd(), 'go.mod'), 'utf8');
  const pinned = 'v0.0.0-20261102193107-c4a49a4fe84f';
  if (which === 'good') {
    pinsCoordinator(goMod, pinned);
    passedEach(recorded('pinned'), 'pinned', names);
    failedEach(recorded('mutated'), 'mutated', names);
    passedEach(recorded('main'), 'main', names);
    return 'accepted';
  }
  rejects(() => pinsCoordinator(goMod.replace(pinned, MAIN_VERSION), pinned), 'a go.mod that requires the main coordinator');
  rejects(() => pinsCoordinator(goMod + `\nreplace ${COORDINATOR} => ../kogen-coordinator\n`, pinned), 'a go.mod that replaces the coordinator');
  rejects(() => passedEach(recorded('main_red'), 'main', names), 'the unchanged tree\'s run against the main coordinator');
  rejects(() => passedEach(recorded('mutated'), 'mutated', names), 'a run against the mutated coordinator as a pass');
  rejects(() => failedEach(recorded('pinned'), 'mutated', names), 'a passing run as the mutated coordinator\'s failure');
  rejects(() => failedEach(recorded('main'), 'mutated', names), 'the prototype\'s passing run against the main coordinator as a failure');
  rejects(() => passedEach({...recorded('main'), status: 1}, 'main', names), 'a failed run');
  rejects(() => failedEach({...recorded('mutated'), status: 2}, 'mutated', names), 'a run that did not exit 1');
  rejects(() => failedEach({...recorded('mutated'), stdout: recorded('mutated').stdout + 'panic: boom\n'}, 'mutated', names), 'a panic');
  rejects(() => passedEach(recorded('main'), 'main', [...names, 'TestKSLMNotThere']), 'an absent test');
  rejects(() => passedEach(recorded('main'), 'main', []), 'an empty list');
  for (const name of names) {
    const ok = recorded('main');
    rejects(() => passedEach({...ok, stdout: ok.stdout.replace(`--- PASS: ${name} (`, `--- FAIL: ${name} (`)}, 'main', names), `a failed ${name}`);
    rejects(() => passedEach({...ok, stdout: ok.stdout.replace(`--- PASS: ${name} (`, `--- PASS: ${name} (0.00s)\n--- PASS: ${name} (`)}, 'main', names), `${name} passing twice`);
    const bad = recorded('mutated');
    rejects(() => failedEach({...bad, stdout: bad.stdout.replace(`--- FAIL: ${name} (`, `--- PASS: ${name} (`)}, 'mutated', names), `a passing ${name} against the mutated coordinator`);
    rejects(() => failedEach({...bad, stdout: bad.stdout.split('\n').filter(l => !l.trimStart().startsWith(`--- FAIL: ${name} (`)).join('\n')}, 'mutated', names), `an absent ${name} against the mutated coordinator`);
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
  const {Given, When, Then} = require('@cucumber/cucumber');
  Given('go.mod requires kogen-coordinator {string}', function (version) {
    pinsCoordinator(fs.readFileSync(path.join(process.cwd(), 'go.mod'), 'utf8'), version);
  });
  When('the lint-message shape tests run against the {string} coordinator:', {timeout: 150000}, function (kind, table) {
    const names = table.hashes().map(row => row.test);
    this.kslm = {kind, names, run: run(kind, names)};
  });
  Then('each of those lint-message shape tests passed exactly once', function () {
    assert.ok(this.kslm, 'no shape tests ran');
    passedEach(this.kslm.run, this.kslm.kind, this.kslm.names);
  });
  Then('each of those lint-message shape tests failed exactly once', function () {
    assert.ok(this.kslm, 'no shape tests ran');
    failedEach(this.kslm.run, this.kslm.kind, this.kslm.names);
  });
}
