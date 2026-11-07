const { Then } = require('@cucumber/cucumber');
const { spawnSync } = require('node:child_process');
const assert = require('node:assert/strict');

Then('k-shape-progress-probe-result: the existing shaping regression {string} passes', function (name) {
  const command = ['test', '-json', '-count=1', './shape', '-run', '^' + name + '$'];
  const result = spawnSync('go', command, { encoding: 'utf8', timeout: 240000, maxBuffer: 16 * 1024 * 1024 });
  const diagnostic = `${name}: go ${command.join(' ')}; status=${result.status}; error=${result.error || ''}\n${result.stdout || ''}\n${result.stderr || ''}`;
  assert.equal(result.status, 0, diagnostic);
  const events = (result.stdout || '').split('\n').filter(Boolean).map(line => JSON.parse(line));
  const runs = events.filter(event => event.Test === name && event.Action === 'run');
  const passed = events.filter(event => event.Test === name && event.Action === 'pass');
  const skipped = events.filter(event => event.Test === name && event.Action === 'skip');
  assert.equal(runs.length, 1, `${diagnostic}\n${name}: required test absent or ran zero times`);
  assert.equal(skipped.length, 0, `${diagnostic}\n${name}: required test skipped`);
  assert.equal(passed.length, 1, `${diagnostic}\n${name}: required test did not pass`);
});
