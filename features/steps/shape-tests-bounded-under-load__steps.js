'use strict';
// Steps of SHAPE-TESTS-BOUNDED-UNDER-LOAD. They read the shape package's test files, run the clock-bound
// tests and the three existing time tests of the package once, and read the change's file list.
const fs = require('fs');
const path = require('path');
const cp = require('child_process');
const {Given, When, Then} = require('@cucumber/cucumber');

const ROOT = process.cwd();
const P = 'shape-tests-bounded-under-load: ';

function shapeTestFiles() {
  const dir = path.join(ROOT, 'shape');
  return fs.readdirSync(dir).filter((f) => f.endsWith('_test.go')).sort().map((f) => ({name: 'shape/' + f, text: fs.readFileSync(path.join(dir, f), 'utf8')}));
}

// code drops whole-line comments, so that a comment naming a call is not counted as the call.
function code(text) {
  return text.split('\n').map((l) => (/^\s*\/\//.test(l) ? '' : l)).join('\n');
}

function none(problems, what) {
  if (problems.length) throw new Error(what + ':\n' + problems.join('\n'));
}

function git(args) {
  const r = cp.spawnSync('git', args, {cwd: ROOT, encoding: 'utf8', maxBuffer: 1 << 28});
  if (r.status !== 0) throw new Error('git ' + args.join(' ') + ' exited ' + r.status + ': ' + r.stderr);
  return r.stdout;
}

Given(P + 'the test files of the shape package', function () {
  this.files = shapeTestFiles();
  if (this.files.length === 0) throw new Error('no shape/*_test.go file found');
});

Then(P + 'no shape test calls time.Sleep', function () {
  const bad = [];
  for (const f of this.files) {
    code(f.text).split('\n').forEach((l, i) => { if (/\btime\.Sleep\s*\(/.test(l)) bad.push(f.name + ':' + (i + 1) + ': ' + l.trim()); });
  }
  none(bad, 'shape tests wait on the real clock with time.Sleep');
});

Then(P + 'no shape test runs a command that sleeps or waits more than one second', function () {
  const bad = [];
  for (const f of this.files) {
    code(f.text).split('\n').forEach((l, i) => {
      for (const m of l.matchAll(/\bsleep\s+([0-9]+(?:\.[0-9]+)?)/g)) {
        if (Number(m[1]) > 1) bad.push(f.name + ':' + (i + 1) + ': shell sleep ' + m[1]);
      }
      for (const m of l.matchAll(/setTimeout\([^,)]*,\s*(\d+)/g)) {
        if (Number(m[1]) > 1000) bad.push(f.name + ':' + (i + 1) + ': setTimeout of ' + m[1] + ' ms');
      }
    });
  }
  none(bad, 'shape tests wait more than one second on a real command');
});

Then(P + 'no shape test contains the wall-clock assertion {string}', function (text) {
  const bad = this.files.filter((f) => code(f.text).includes(text)).map((f) => f.name + ' contains ' + JSON.stringify(text));
  none(bad, 'a wall-clock threshold assertion is still in the shape tests');
});

When(P + 'I run the tests of the shape package whose names start with the clock-bound prefix and the existing overlap, control-bound and time-split tests once', {timeout: 115 * 1000}, function () {
  const pattern = '^(TestClockBound.*|TestStaticChecksOverlapModelChecks|TestUnpassableControlSkipped|TestShapeTimeSplitsChecksByGate)$';
  const env = Object.assign({}, process.env, {GOFLAGS: '-trimpath'});
  const r = cp.spawnSync('go', ['test', './shape', '-count=1', '-v', '-run', pattern], {cwd: ROOT, env, encoding: 'utf8', timeout: 105 * 1000, maxBuffer: 1 << 28});
  this.goOut = (r.stdout || '') + (r.stderr || '');
  this.goExit = r.status;
  this.goError = r.error ? String(r.error) : '';
});

Then(P + 'the go command exits 0', function () {
  if (this.goExit !== 0) throw new Error('go test ./shape exited ' + this.goExit + ' ' + this.goError + '\n' + this.goOut.slice(-4000));
});

Then(P + 'at least {int} clock-bound tests are reported as passed and none is reported as failed or skipped', function (count) {
  const passed = (this.goOut.match(/^--- PASS: TestClockBound\w* \(/gm) || []).length;
  const bad = (this.goOut.match(/^\s*--- (FAIL|SKIP): TestClockBound\w*/gm) || []);
  if (passed < count || bad.length > 0) {
    throw new Error('go test ./shape -run (clock-bound prefix) reported ' + passed + ' passed clock-bound tests, want at least ' + count + '; failed or skipped: ' + JSON.stringify(bad) + '; output tail:\n' + this.goOut.slice(-3000));
  }
});

Then(P + '{string} is reported as passed', function (name) {
  const re = new RegExp('^--- PASS: ' + name + ' \\(', 'm');
  if (!re.test(this.goOut)) throw new Error('go test ./shape -run did not report "--- PASS: ' + name + '" (the test is absent, skipped or failed); output tail:\n' + this.goOut.slice(-3000));
});

Given(P + 'the files this change adds or edits', function () {
  const tracked = git(['diff', '--name-only', 'HEAD']).split('\n').filter(Boolean);
  const untracked = git(['ls-files', '--others', '--exclude-standard']).split('\n').filter(Boolean);
  this.changed = Array.from(new Set(tracked.concat(untracked))).sort();
  this.diff = git(['diff', '-U0', 'HEAD', '--', 'shape']);
});

Then(P + 'every changed file is under shape\\/, .kogen\\/ or features\\/', function () {
  const bad = this.changed.filter((f) => !/^(shape|\.kogen|features)\//.test(f));
  none(bad, 'files outside the shape package, the Intent and its steps changed');
});

Then(P + 'no changed Go source of the shape package outside its tests adds an environment variable, flag or setting read', function () {
  const bad = [];
  let file = '';
  for (const l of this.diff.split('\n')) {
    const m = /^\+\+\+ b\/(.*)$/.exec(l);
    if (m) { file = m[1]; continue; }
    if (!file.endsWith('.go') || file.endsWith('_test.go') || !l.startsWith('+')) continue;
    if (/\bos\.(Getenv|LookupEnv|Environ)\b|\bflag\.[A-Z]|\bpflag\./.test(l)) bad.push(file + ': ' + l);
  }
  none(bad, 'production code gained an environment or flag read');
});

Then(P + 'no shape test is skipped, short-mode gated or given a longer timeout by this change', function () {
  const bad = [];
  for (const f of this.files) {
    if (f.name !== 'shape/clock_bound_test.go') continue;
    code(f.text).split('\n').forEach((l, i) => { if (/\bt\.Skip(f|Now)?\(|testing\.Short\(\)/.test(l)) bad.push(f.name + ':' + (i + 1) + ': ' + l.trim()); });
  }
  let file = '';
  for (const l of this.diff.split('\n')) {
    const m = /^\+\+\+ b\/(.*)$/.exec(l);
    if (m) { file = m[1]; continue; }
    if (l.startsWith('+') && /\bt\.Skip(f|Now)?\(|testing\.Short\(\)|-timeout/.test(l)) bad.push(file + ': ' + l);
  }
  none(bad, 'tests were skipped, gated on short mode or given a timeout');
});

Then(P + 'the file shape\\/clock_bound_test.go defines the four TestClockBound tests', function () {
  const f = this.files.find((x) => x.name === 'shape/clock_bound_test.go');
  if (!f) throw new Error('shape/clock_bound_test.go does not exist');
  const missing = ['TestClockBoundStaticChecksOverlap', 'TestClockBoundUnpassableControl', 'TestClockBoundTimeCountsOverlapOnce', 'TestClockBoundNoRealSleepsInShape']
    .filter((n) => !new RegExp('^func ' + n + '\\(t \\*testing\\.T\\)', 'm').test(f.text));
  none(missing, 'shape/clock_bound_test.go lacks these top-level tests');
});
