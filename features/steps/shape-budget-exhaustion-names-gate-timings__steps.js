'use strict';
const {Given, When, Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');

const P = 'shape-budget-exhaustion-names-gate-timings: ';
let result;

function journey() {
  if (result) return result;
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'budget-gates-'));
  try {
    const tree = path.join(root, 'tree');
    fs.cpSync(process.cwd(), tree, {recursive: true, filter: p => !['.git', 'node_modules'].includes(path.basename(p))});
    const source = require('./shape-budget-exhaustion-names-gate-timings__harness');
    fs.writeFileSync(path.join(tree, 'shape', 'budget_gate_timings_acceptance_test.go'), source);
    const binary = path.join(root, 'budget.test');
    const build = cp.spawnSync('go', ['test', '-c', '-o', binary, './shape'], {cwd: tree, encoding: 'utf8', timeout: 150000});
    assert.equal(build.status, 0, 'compile budget acceptance: ' + build.stdout + build.stderr);
    const report = path.join(root, 'result.json');
    const run = cp.spawnSync(binary, ['-test.run=^TestShapeBudgetAcceptanceTimedAttempts$', '-test.v', '-test.timeout=100s'],
      {cwd: path.join(tree, 'shape'), env: {...process.env, BUDGET_RESULT: report}, encoding: 'utf8', timeout: 110000});
    assert.equal(run.status, 0, 'TestShapeBudgetAcceptanceTimedAttempts via built test binary: ' + run.stdout + run.stderr);
    assert.ok(fs.existsSync(report), 'the scripted shaping must write its observed result');
    result = JSON.parse(fs.readFileSync(report, 'utf8'));
    return result;
  } finally {
    fs.rmSync(root, {recursive: true, force: true});
  }
}

function lines(r) { return (r.lines || []).map(l => String(l).trim()); }
function shown(r) { return JSON.stringify(r, null, 1); }

Given(P + 'a scripted shaper refused four times at 225 seconds per model request', function () {
  const r = journey();
  assert.equal(r.refusedGates.length, 4, 'four refusals recorded: ' + shown(r));
  assert.equal(r.requests, 4, shown(r));
});

When(P + 'the shaping runs until its budget ends', function () {
  const r = journey();
  assert.equal(r.class, 'shape', shown(r));
  assert.equal(r.reason, 'invalid_output', shown(r));
  assert.notEqual(r.exit, 0, shown(r));
});

Then(P + 'the refusal lists four timed attempts with gate, model time, check time and remaining budget', function () {
  const r = journey();
  const attempts = lines(r).filter(l => l.startsWith('attempt '));
  assert.equal(attempts.length, 4, 'attempt lines: ' + shown(r));
  const remaining = [675, 450, 225, 0];
  attempts.forEach((l, i) => {
    const m = /^attempt (\d+): gate (\S+), model request (\d+) s, Kogen checks (\d+) s, (\d+) s remaining$/.exec(l);
    assert.ok(m, 'attempt line format: ' + l);
    assert.equal(Number(m[1]), i + 1, l);
    assert.equal(m[2], r.refusedGates[i], 'gate of ' + l + ' vs submission_refused ' + JSON.stringify(r.refusedGates));
    assert.equal(Number(m[3]), 225, l);
    assert.equal(Number(m[4]), 0, l);
    assert.equal(Number(m[5]), remaining[i], l);
  });
});

Then(P + 'the refusal states a budget figure equal to the shaping_turn_counts budget', function () {
  const r = journey();
  const budget = lines(r).filter(l => l.startsWith('shaping budget: '));
  assert.equal(budget.length, 1, 'shaping budget line: ' + shown(r));
  const m = /^shaping budget: (\d+) s \((\d+) min,/.exec(budget[0]);
  assert.ok(m, budget[0]);
  assert.equal(Number(m[1]), r.budgetSeconds, 'figure vs shaping_turn_counts budget_seconds: ' + shown(r));
  assert.equal(Number(m[2]) * 60, r.budgetSeconds, budget[0]);
  assert.equal(r.budgetSeconds, 900);
});

Then(P + 'the refusal says response_budget is a separate budget', function () {
  const r = journey();
  const line = lines(r).find(l => l.startsWith('shaping budget: '));
  assert.ok(line, shown(r));
  assert.ok(line.includes('response_budget'), line);
  assert.ok(line.includes('separate'), line);
});

Then(P + 'the refusal still reports 15 min elapsed and 4 requests', function () {
  const r = journey();
  const text = [r.detail, ...lines(r)].join('\n');
  assert.ok(text.includes('15-minute shaping budget'), text);
  assert.ok(text.includes('15 min elapsed'), text);
  assert.ok(text.includes('4 requests'), text);
});
