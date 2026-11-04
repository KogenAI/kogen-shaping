// Known-good and known-bad controls of the k-contracts-shaping observer
// (k-contracts-shaping__plants.js). Run as `node features/steps/k-contracts-shaping__control.js
// good|bad` from the checkout root; loaded by Cucumber it does nothing. The good control is the
// existing shape test TestShapeBudgetRolesShape, which catches every planted change; the bad
// controls are suites and plants the observer must reject. Each problem prints its first line.
'use strict';
const path = require('path');
const fs = require('fs');
const obs = require('./k-contracts-shaping__plants.js');

const root = path.resolve(__dirname, '..', '..');
const SHAPE = 'github.com/KogenAI/kogen-shaping/shape';
const STUB = 'github.com/KogenAI/kogen-shaping/contractstub';
const budgetTest = ['-count=1', '-run', '^TestShapeBudgetRolesShape$', './shape/'];

function first(problems) { return problems.map((p) => p.split('\n')[0]); }

function goTest(args, overlay = {}) {
  const dir = obs.copyCheckout(root, overlay);
  try {
    return obs.run('go', ['test', ...args], dir, root);
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

function unplanted(args, pkgs, overlay = {}) {
  return first(obs.judgePassed(args, pkgs, goTest(args, overlay)));
}

function within(args, pkg, seconds, overlay = {}) {
  const problem = obs.judgeWithin(args, pkg, seconds, goTest(args, overlay));
  return problem ? first([problem]) : [];
}

function good() {
  const lines = [];
  const r = goTest(budgetTest);
  const base = first(obs.judgePassed(budgetTest, [SHAPE], r));
  lines.push(...(base.length ? base : ['unplanted: passed']));
  const fast = obs.judgeWithin(budgetTest, SHAPE, 60, r);
  lines.push(fast ? first([fast])[0] : 'unplanted: passed within 60 s');
  for (const p of obs.PLANTS) {
    const problems = first(obs.plantedRun(root, p.name, budgetTest));
    lines.push(...(problems.length ? problems : [`${p.name}: caught`]));
  }
  return lines;
}

function bad() {
  const passing = { 'contractstub/stub_test.go': 'package contractstub\n\nimport "testing"\n\nfunc TestNothing(t *testing.T) {}\n' };
  const empty = { 'contractstub/stub_test.go': 'package contractstub\n' };
  const broken = { ...passing, 'contractstub/broken_test.go': 'package contractstub\n\nfunc broken() { undefined() }\n' };
  const shapeGo = fs.readFileSync(path.join(root, 'shape/shape.go'), 'utf8');
  const doubled = { 'shape/shape.go': shapeGo.replace(obs.PLANTS[0].find, obs.PLANTS[0].find + obs.PLANTS[0].find) };
  const name = obs.PLANTS[0].name;
  return [
    ...first(obs.plantedRun(root, name, ['-count=1', './contractstub/'], passing)),
    ...first(obs.plantedRun(root, name, ['-count=1', './nothere/...'], passing)),
    ...first(obs.plantedRun(root, name, ['-count=1', './contractstub/'], broken)),
    ...first(obs.plantedRun(root, name, ['-count=1', './contractstub/'], doubled)),
    ...unplanted(['-count=1', './contractstub/'], [STUB], empty),
    ...unplanted(['-count=1', './contractstub/'], [STUB, 'github.com/KogenAI/kogen-shaping/contract'], passing),
    ...within(['-count=1', './contractstub/'], STUB, 0, passing),
    ...within(['-count=1', './nothere/...'], STUB, 60, passing),
  ];
}

if (require.main === module) {
  const mode = process.argv[2];
  if (mode !== 'good' && mode !== 'bad') {
    process.stderr.write('usage: node k-contracts-shaping__control.js good|bad\n');
    process.exit(2);
  }
  const lines = mode === 'good' ? good() : bad();
  process.stdout.write(lines.join('\n') + '\n');
  const ok = mode === 'good' && lines.every((l) => l === 'unplanted: passed' || l === 'unplanted: passed within 60 s' || l.endsWith(': caught'));
  process.exit(ok ? 0 : 1);
}
