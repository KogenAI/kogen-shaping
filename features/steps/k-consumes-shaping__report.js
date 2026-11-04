'use strict';
// Helpers of the k-consumes-shaping acceptance (features/steps/k-consumes-shaping__steps.js) and
// of its observer controls (k-consumes-shaping__control.js): the build and run of the consumes
// observer (k-consumes-shaping__observer.js), the go list and go test runs and the judgement of
// their output. It defines no step. No subprocess has a time limit of its own: the step's
// Cucumber timeout bounds it.
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const observer = require('./k-consumes-shaping__observer.js');

function run(cmd, args, cwd, env) {
  const r = spawnSync(cmd, args, { cwd, env, encoding: 'utf8', maxBuffer: 256 * 1024 * 1024 });
  return { status: r.status, output: (r.stdout || '') + (r.stderr || '') + (r.error ? String(r.error) : '') };
}

// plainEnv is this process's environment without its GIT_* variables, so a private index or Git
// dir of the caller does not change what the checkout is.
function plainEnv() {
  return Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('GIT_')));
}

// hostileEnv is plainEnv with a Git dir, work tree and index that do not exist, a fresh HOME
// under dir whose Git configuration signs every commit and tag with a key that does not exist,
// no module proxy, no VCS stamping and trimmed paths; the Go caches stay where go env puts them
// for cwd.
function hostileEnv(dir, cwd) {
  const where = run('go', ['env', '-json', 'GOMODCACHE', 'GOCACHE', 'GOPATH'], cwd, plainEnv());
  if (where.status !== 0) throw new Error(`go env -json GOMODCACHE GOCACHE GOPATH (in ${cwd}) exited ${where.status}:\n${where.output}`);
  const home = path.join(dir, 'home');
  fs.mkdirSync(home, { recursive: true });
  fs.writeFileSync(path.join(home, '.gitconfig'),
    '[user]\n\tname = hostile\n\temail = hostile@example.invalid\n\tsigningkey = no-such-key\n[commit]\n\tgpgsign = true\n[tag]\n\tgpgsign = true\n[init]\n\tdefaultBranch = hostile\n');
  const noGit = path.join(dir, 'no-git');
  return Object.assign(plainEnv(), JSON.parse(where.output), {
    HOME: home,
    GIT_DIR: noGit,
    GIT_WORK_TREE: noGit,
    GIT_INDEX_FILE: path.join(noGit, 'index'),
    GOPROXY: 'off',
    GOFLAGS: '-mod=readonly -buildvcs=false -trimpath',
  });
}

// goTest runs go test with args in cwd: with plainEnv, or with hostileEnv when hostile.
function goTest(cwd, args, hostile) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-consumes-shaping-gotest-'));
  try {
    const env = hostile ? hostileEnv(dir, cwd) : plainEnv();
    const shown = hostile ? ' with a bogus GIT_DIR, a fresh HOME, GOPROXY=off and GOFLAGS=-mod=readonly -buildvcs=false -trimpath' : '';
    return { command: `go test ${args.join(' ')} (in ${cwd}${shown})`, ...run('go', ['test', ...args], cwd, env) };
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

// goList lists the import paths of the packages that the pattern matches in cwd, the packages
// "go test <pattern>" tests.
function goList(cwd, pattern) {
  return { command: `go list ${pattern} (in ${cwd})`, ...run('go', ['list', pattern], cwd, plainEnv()) };
}

function escape(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// testProblems judges a go test run: it exited 0, and each package has its "ok" line without
// "[no tests to run]".
function testProblems(result, pkgs) {
  const problems = [];
  if (result.status !== 0) problems.push(`go test exited ${result.status}`);
  for (const pkg of pkgs) {
    const m = new RegExp(`^ok\\s+${escape(pkg)}\\s(.*)$`, 'm').exec(result.output);
    if (!m) problems.push(`${pkg}: no "ok" line, its tests did not pass`);
    else if (m[1].includes('[no tests to run]')) problems.push(`${pkg}: ran no tests`);
  }
  return problems;
}

// observe builds the observer outside the checkout and runs it on root for the suites, given as
// [suite, module] pairs.
function observe(root, suites) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-consumes-shaping-observer-'));
  try {
    fs.writeFileSync(path.join(dir, 'main.go'), observer.source);
    fs.writeFileSync(path.join(dir, 'go.mod'), 'module observer\n\ngo 1.27\n');
    const env = Object.assign(plainEnv(), { GOFLAGS: '', GOWORK: 'off' });
    const bin = path.join(dir, 'observer');
    const build = run('go', ['build', '-o', bin, '.'], dir, env);
    if (build.status !== 0) throw new Error(`go build of the observer (in ${dir}) exited ${build.status}:\n${build.output}`);
    const args = [root, ...suites.map(([s, m]) => `${s}=${m}`)];
    return { command: `observer ${args.join(' ')}`, ...run(bin, args, root, plainEnv()) };
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

module.exports = { run, plainEnv, goTest, goList, testProblems, observe };
