// The acceptance observer of k-contracts-shaping: it copies this checkout to a scratch
// directory, plants one change in kogen-shaping's production code there, and judges whether a
// Go test run catches it. This module defines no steps and does nothing when it is loaded.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');

const CONTRACT = 'github.com/KogenAI/kogen-shaping/contract';

// Each planted change alters behaviour kogen reaches through shape.Shape or shape.Refine on the
// path of its budget tests, and leaves every package compiling. `find` occurs once in `file` at
// HEAD.
const PLANTS = [
  {
    name: 'shape records its response budget twice',
    file: 'shape/shape.go',
    find: '\tev.Record(agent.BudgetEvent(in.Client))\n',
    replace: '\tev.Record(agent.BudgetEvent(in.Client))\n\tev.Record(agent.BudgetEvent(in.Client))\n',
  },
  {
    name: 'refine records its response budget twice',
    file: 'shape/refine.go',
    find: '\tr.ev.Record(agent.BudgetEvent(r.client))\n',
    replace: '\tr.ev.Record(agent.BudgetEvent(r.client))\n\tr.ev.Record(agent.BudgetEvent(r.client))\n',
  },
  {
    name: 'shape sends under a ten-minute budget',
    file: 'shape/shape.go',
    find: '\tev.Record(agent.BudgetEvent(in.Client))\n',
    replace: '\tif in.Client != nil {\n\t\tin.Client.ResponseBudget = 10 * time.Minute\n\t}\n\tev.Record(agent.BudgetEvent(in.Client))\n',
  },
  {
    name: 'shape returns exit 1 for every refusal',
    file: 'shape/shape.go',
    find: '\texit, err = shapeRun(ctx, s, in)\n\tif err != nil {\n\t\texit = ExitCode(err)\n',
    replace: '\texit, err = shapeRun(ctx, s, in)\n\tif err != nil {\n\t\texit = 1\n',
  },
  {
    name: 'refine refuses with exit 1',
    file: 'shape/refine.go',
    find: '\texit, err = refineStart(ctx, s, in)\n\tif err != nil {\n',
    replace: '\texit, err = refineStart(ctx, s, in)\n\tif e, ok := sys.AsError(err); ok {\n\t\te.Exit = 1\n\t}\n\tif err != nil {\n',
  },
];

function plant(name) {
  const p = PLANTS.find((x) => x.name === name);
  if (!p) throw new Error(`no planted change named "${name}"`);
  return p;
}

// env is the environment of every Go command: the caller's, without its Git and Kogen settings,
// offline, with Kogen's Node and Cucumber for the step checks Shape runs.
function env(root) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) {
    if (k.startsWith('GIT_') || k.startsWith('KOGEN_') || k === 'GOFLAGS' || k === 'GOWORK') continue;
    out[k] = v;
  }
  out.KOGEN_NODE = process.env.KOGEN_NODE || process.execPath;
  out.KOGEN_CUCUMBER = process.env.KOGEN_CUCUMBER || path.join(root, 'testdata/tools/cucumber/node_modules');
  out.GOFLAGS = '-mod=readonly -buildvcs=false -trimpath';
  out.GOWORK = 'off';
  out.GOPROXY = 'off';
  out.GIT_CONFIG_GLOBAL = '/dev/null';
  out.GIT_CONFIG_NOSYSTEM = '1';
  out.GIT_TERMINAL_PROMPT = '0';
  return out;
}

function run(cmd, args, cwd, root) {
  const r = spawnSync(cmd, args, { cwd, env: env(root), encoding: 'utf8', maxBuffer: 256 * 1024 * 1024 });
  if (r.error) return { status: -1, output: String(r.error) };
  return { status: r.status === null ? -1 : r.status, output: (r.stdout || '') + (r.stderr || '') };
}

// copyCheckout copies the regular files of root, without .git, node_modules and .kogen/local,
// to a new scratch directory, then writes overlay (path -> content) over it.
function copyCheckout(root, overlay = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-contracts-shaping-'));
  const walk = (rel) => {
    for (const e of fs.readdirSync(path.join(root, rel), { withFileTypes: true })) {
      const r = rel ? `${rel}/${e.name}` : e.name;
      if (e.isDirectory()) {
        if (e.name === '.git' || e.name === 'node_modules' || r === '.kogen/local') continue;
        fs.mkdirSync(path.join(dir, r), { recursive: true });
        walk(r);
      } else if (e.isFile()) {
        fs.copyFileSync(path.join(root, r), path.join(dir, r));
      }
    }
  };
  walk('');
  for (const [rel, content] of Object.entries(overlay)) {
    fs.mkdirSync(path.dirname(path.join(dir, rel)), { recursive: true });
    fs.writeFileSync(path.join(dir, rel), content);
  }
  return dir;
}

// applyPlant applies p in dir; it returns a problem, or '' when the change applied once.
function applyPlant(dir, p) {
  const file = path.join(dir, p.file);
  const text = fs.readFileSync(file, 'utf8');
  const n = text.split(p.find).length - 1;
  if (n !== 1) return `${p.name}: the planted change does not apply: ${p.file} has ${n} occurrences of ${JSON.stringify(p.find)}, want 1`;
  fs.writeFileSync(file, text.replace(p.find, p.replace));
  return '';
}

// compiles is a problem, or '' when every package and test package in dir builds and vets.
function compiles(dir, root, label) {
  for (const args of [['build', './...'], ['vet', './...']]) {
    const r = run('go', args, dir, root);
    if (r.status !== 0) return `${label}: go ${args.join(' ')} failed (exit ${r.status}):\n${r.output}`;
  }
  return '';
}

// judgeCaught judges a go test run of a planted tree: the plant is caught only when go test
// exits 1 because a test failed, not because a package failed to build or set up.
function judgeCaught(name, args, r) {
  const cmd = `go test ${args.join(' ')}`;
  if (r.status === 0) return `${name}: not caught: ${cmd} passed:\n${r.output}`;
  if (/\[(build|setup) failed\]/.test(r.output) || /^# /m.test(r.output)) return `${name}: not caught by a test: ${cmd} did not build or set up (exit ${r.status}):\n${r.output}`;
  if (r.status !== 1 || !/^\s*--- FAIL: /m.test(r.output)) return `${name}: not caught by a test: ${cmd} exited ${r.status} without a failing test:\n${r.output}`;
  return '';
}

// judgePassed judges a go test run that must pass and run tests in each of pkgs.
function judgePassed(args, pkgs, r) {
  const cmd = `go test ${args.join(' ')}`;
  if (r.status !== 0) return [`${cmd} failed (exit ${r.status}):\n${r.output}`];
  const problems = [];
  for (const pkg of pkgs) {
    const esc = pkg.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&');
    if (!new RegExp(`^ok\\s+${esc}\\s+\\S+\\s*$`, 'm').test(r.output)) problems.push(`${cmd}: ${pkg} ran no tests or is missing:\n${r.output}`);
  }
  return problems;
}

// judgeWithin judges a go test run that must pass pkg within seconds of test run time, as the
// package's ok line reports it.
function judgeWithin(args, pkg, seconds, r) {
  const cmd = `go test ${args.join(' ')}`;
  const esc = pkg.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&');
  const m = r.status === 0 && new RegExp(`^ok\\s+${esc}\\s+(\\d+(?:\\.\\d+)?)s\\s*$`, 'm').exec(r.output);
  if (!m) return `${cmd}: ${pkg} did not pass with a run time (exit ${r.status}):\n${r.output}`;
  if (Number(m[1]) > seconds) return `${cmd}: ${pkg} took longer than ${seconds} s:\n${m[1]} s`;
  return '';
}

// plantedRun copies root, applies the plant named name, requires every package to compile and
// runs go test with args there: the problems found, none when the plant was caught.
function plantedRun(root, name, args, overlay = {}) {
  const dir = copyCheckout(root, overlay);
  try {
    const p = plant(name);
    const problem = applyPlant(dir, p) || compiles(dir, root, p.name);
    if (problem) return [problem];
    const caught = judgeCaught(p.name, args, run('go', ['test', ...args], dir, root));
    return caught ? [caught] : [];
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

module.exports = { CONTRACT, PLANTS, plant, env, run, copyCheckout, applyPlant, compiles, judgeCaught, judgePassed, judgeWithin, plantedRun };
