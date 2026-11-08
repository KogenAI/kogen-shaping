const { Then } = require('@cucumber/cucumber');
const fs = require('node:fs');
const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');

function preservedScope() {
  const config = '.kogen/project.yaml';
  assert.equal(fs.readFileSync(config, 'utf8'), execFileSync('git', ['show', `HEAD:${config}`], { encoding: 'utf8' }), 'Keep all configured checks, full suite, acceptance command and timeout policy unchanged');
  const changed = execFileSync('git', ['diff', 'HEAD', '--name-only'], { encoding: 'utf8' }).trim().split('\n').filter(Boolean);
  for (const path of changed) {
    assert.ok(path.startsWith('.kogen/') || path.startsWith('features/steps/k-shape-quint-invariant-expression__') || path === 'shape/prompts/shaper.md' || /^shape\/testdata\/.*golden.*\.json$/.test(path) || /^shape\/.*_test\.go$/.test(path), `Wording-only scope excludes ${path}`);
  }
  const intent = fs.readFileSync('.kogen/intents/k-shape-quint-invariant-expression/intent.md', 'utf8');
  assert.ok(!/^## Stages$/m.test(intent), 'No new stage');
  assert.ok(!fs.existsSync('.kogen/intents/k-shape-quint-invariant-expression/model.qnt'), 'No new model');
}

Then('k-shape-quint-invariant-expression: {string} contains the guidance', function (path, expected) {
  preservedScope();
  assert.ok(fs.readFileSync(path, 'utf8').includes(expected), `${path} must contain the requested guidance: ${expected}`);
});

Then('k-shape-quint-invariant-expression: the policy prompt snapshot contains the guidance', function (expected) {
  const path = 'shape/testdata/shaper_policy_golden.json';
  const snapshot = JSON.parse(fs.readFileSync(path, 'utf8'));
  assert.ok(typeof snapshot.prompt === 'string' && snapshot.prompt.includes(expected), `${path} prompt must contain the requested guidance: ${expected}`);
});
