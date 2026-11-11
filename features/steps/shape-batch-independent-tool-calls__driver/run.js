'use strict';
// Acceptance driver of shape-batch-independent-tool-calls: copies the candidate checkout (the
// current directory, uncommitted changes included) to a disposable directory, injects cases.go
// as shape/accept_batch_independent_test.go and runs one of its tests through `go test -json`.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');

const tests = {
  overlap: 'TestAcceptBatchOverlap',
  barriers: 'TestAcceptBatchBarriers',
  repeat: 'TestAcceptBatchRepeat',
  policy: 'TestAcceptBatchPolicy',
};

// observe judges the `go test -json` output of one named test: it must have run once and
// passed once. The thrown message names the test, the command and what happened.
function observe(stdout, stderr, status, name, command) {
  const rows = [];
  for (const line of stdout.split('\n')) {
    if (!line.trim()) continue;
    try { rows.push(JSON.parse(line)); } catch (e) { /* not an event */ }
  }
  const mine = rows.filter((r) => r.Test === name || (r.Test || '').startsWith(name + '/'));
  const runs = mine.filter((r) => r.Test === name && r.Action === 'run').length;
  const passes = mine.filter((r) => r.Test === name && r.Action === 'pass').length;
  const fails = mine.filter((r) => r.Test === name && r.Action === 'fail').length;
  const skips = mine.filter((r) => r.Test === name && r.Action === 'skip').length;
  const output = mine.filter((r) => r.Action === 'output').map((r) => r.Output).join('');
  if (runs === 1 && passes === 1 && status === 0) return;
  let outcome;
  if (runs === 0) outcome = 'the test was absent or ran zero times';
  else if (skips > 0) outcome = 'the test was skipped';
  else if (fails > 0) outcome = 'the test failed';
  else outcome = `the test ran ${runs} time(s) and passed ${passes} time(s)`;
  const build = rows.filter((r) => r.Action === 'build-output').map((r) => r.Output).join('');
  throw new Error(`${name}: ${outcome} (\`${command}\` exited ${status})\n${output || build || stderr || stdout}`);
}

// candidateFiles are the tracked and untracked, not ignored, files of the checkout.
function candidateFiles(root) {
  const out = cp.execFileSync('git', ['ls-files', '-z', '--cached', '--others', '--exclude-standard'], { cwd: root, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 });
  return out.split('\0').filter((p) => p && !p.startsWith('.kogen/local/'));
}

function runCase(name, env) {
  const test = tests[name];
  if (!test) throw new Error(`unknown case ${name}`);
  const root = process.cwd();
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-batch-'));
  try {
    for (const p of candidateFiles(root)) {
      const src = path.join(root, p);
      if (!fs.existsSync(src) || !fs.lstatSync(src).isFile()) continue;
      const dest = path.join(dir, p);
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.copyFileSync(src, dest);
    }
    const cases = fs.readFileSync(path.join(__dirname, 'cases.go'), 'utf8');
    const prefix = '//go:build ignore\n\n';
    if (!cases.startsWith(prefix)) throw new Error('cases.go lost its build constraint');
    fs.writeFileSync(path.join(dir, 'shape', 'accept_batch_independent_test.go'), cases.slice(prefix.length));
    const args = ['test', '-json', '-count=1', '-run', `^${test}$`, './shape'];
    const command = `go ${args.join(' ')}`;
    const r = cp.spawnSync('go', args, {
      cwd: dir, encoding: 'utf8', timeout: 280000, maxBuffer: 64 * 1024 * 1024,
      env: { ...process.env, GOFLAGS: '-trimpath', ...env },
    });
    if (r.error) throw new Error(`${test}: \`${command}\` did not complete: ${r.error.message}`);
    observe(r.stdout || '', r.stderr || '', r.status, test, command);
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

module.exports = { runCase, observe, tests };
