'use strict';
// Observations of a kogen-shaping checkout for K-SHAPING-LOOP-MODULE-PATH. Each check returns the
// problems it found (an empty array when there are none), so the steps and the observer control
// share the same assertions.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const {spawnSync} = require('node:child_process');

// The quoted import prefix of the bare module, built so that this file does not hold it.
const BARE = String.fromCharCode(34) + 'kogen-loop/';

function run(root, file, args, options = {}) {
  const r = spawnSync(file, args, {cwd: root, encoding: 'utf8', maxBuffer: 512 * 1024 * 1024, env: options.env || process.env, timeout: options.timeout});
  if (r.error) throw new Error(`${[file, ...args].join(' ')}: ${r.error.message}`);
  return {status: r.status, signal: r.signal, stdout: r.stdout || '', stderr: r.stderr || ''};
}

function go(root, args, env) {
  const r = run(root, 'go', args, {env, timeout: 5 * 60 * 1000});
  if (r.status !== 0) throw new Error(`go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}:\n${r.stdout}${r.stderr}`);
  return r.stdout;
}

function goMod(root, env) {
  return JSON.parse(go(root, ['mod', 'edit', '-json'], env));
}

// want: [{module, version}] in go.mod order; every requirement is direct.
function requireProblems(mod, want) {
  const got = (mod.Require || []).map(r => `${r.Path} ${r.Version}${r.Indirect ? ' // indirect' : ''}`);
  const expected = want.map(w => `${w.module} ${w.version}`);
  if (JSON.stringify(got) === JSON.stringify(expected)) return [];
  return [`go.mod requires ${JSON.stringify(got)}, want exactly ${JSON.stringify(expected)}`];
}

function replaceProblems(mod) {
  const replaces = (mod.Replace || []).map(r => `${r.Old.Path}${r.Old.Version ? ' ' + r.Old.Version : ''} => ${r.New.Path}${r.New.Version ? ' ' + r.New.Version : ''}`);
  return replaces.length === 0 ? [] : [`go.mod has replace directives: ${replaces.join('; ')}`];
}

function goSumProblems(root, want) {
  const file = path.join(root, 'go.sum');
  if (!fs.existsSync(file)) return ['go.sum is missing'];
  const lines = fs.readFileSync(file, 'utf8').split('\n');
  const problems = [];
  for (const w of want) {
    for (const key of [`${w.module} ${w.version} h1:`, `${w.module} ${w.version}/go.mod h1:`]) {
      if (!lines.some(l => l.startsWith(key))) problems.push(`go.sum has no line starting ${JSON.stringify(key)}`);
    }
  }
  for (const l of lines) if (l.startsWith('kogen-loop ')) problems.push(`go.sum holds the bare module: ${l}`);
  return problems;
}

function goWorkProblems(root, env) {
  const problems = [];
  if (fs.existsSync(path.join(root, 'go.work'))) problems.push('the checkout has a go.work file');
  const work = go(root, ['env', 'GOWORK'], env).trim();
  if (work !== '' && work !== 'off') problems.push(`go env GOWORK is ${JSON.stringify(work)}, want no workspace`);
  return problems;
}

// The modules of the build list and every package the module's packages and tests import.
function listBuild(root, env) {
  const modules = go(root, ['list', '-m', 'all'], env).split('\n').filter(Boolean);
  const packages = go(root, ['list', '-deps', '-test', '-f', '{{.ImportPath}}', './...'], env).split('\n').filter(Boolean);
  return {modules, packages};
}

function barePathProblems(build, exact, prefix) {
  const problems = [];
  for (const line of build.modules) {
    const p = line.split(' ')[0];
    if (p === exact || p.startsWith(prefix)) problems.push(`go list -m all lists ${JSON.stringify(line)}`);
  }
  for (const p of build.packages) {
    if (p === exact || p.startsWith(prefix)) problems.push(`go list -deps -test ./... lists ${JSON.stringify(p)}`);
  }
  return problems;
}

function buildListProblems(build, line) {
  return build.modules.includes(line) ? [] : [`go list -m all does not list ${JSON.stringify(line)}; it lists ${JSON.stringify(build.modules)}`];
}

function importProblems(build, want) {
  return want.filter(p => !build.packages.includes(p)).map(p => `go list -deps -test ./... does not list ${JSON.stringify(p)}`);
}

// Files named *.js or *.go.txt directly under dir that hold text, reported as path:line: text.
function harnessProblems(root, dir, text) {
  const problems = [];
  for (const name of fs.readdirSync(path.join(root, dir)).sort()) {
    if (!name.endsWith('.js') && !name.endsWith('.go.txt')) continue;
    const rel = path.posix.join(dir, name);
    fs.readFileSync(path.join(root, rel), 'utf8').split('\n').forEach((l, i) => {
      if (l.includes(text)) problems.push(`${rel}:${i + 1}: ${l.trim()}`);
    });
  }
  return problems;
}

function sha256(root, rel) {
  return crypto.createHash('sha256').update(fs.readFileSync(path.join(root, rel))).digest('hex');
}

function shaProblems(root, rel, want) {
  if (!fs.existsSync(path.join(root, rel))) return [`${rel} is missing`];
  const got = sha256(root, rel);
  return got === want ? [] : [`${rel} has SHA-256 ${got}, want ${want}`];
}

// Runs a shell command in root with its output combined, without the outer acceptance run's
// report and feature variables, as the configured checks run.
function shell(root, command, timeout) {
  const env = {...process.env};
  delete env.KOGEN_ACCEPTANCE_FEATURES;
  delete env.KOGEN_ACCEPTANCE_REPORT;
  const r = run(root, '/bin/sh', ['-c', `{\n${command}\n} 2>&1`], {env, timeout});
  return {command, status: r.status, signal: r.signal, output: r.stdout + r.stderr};
}

module.exports = {BARE, go, goMod, requireProblems, replaceProblems, goSumProblems, goWorkProblems, listBuild,
  barePathProblems, buildListProblems, importProblems, harnessProblems, sha256, shaProblems, shell};
