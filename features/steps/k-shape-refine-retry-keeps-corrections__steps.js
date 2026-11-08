const { Then } = require('@cucumber/cucumber');
const { spawnSync } = require('node:child_process');
const assert = require('node:assert/strict');

function run(name) {
  const args = ['test', '-json', '-count=1', './shape'];
  if (name) args.push('-run', `^${name}$`);
  const command = `go ${args.join(' ')}`;
  const result = spawnSync('go', args, { encoding: 'utf8', timeout: 240000, maxBuffer: 32 * 1024 * 1024 });
  const diagnostic = `${name || 'shaping suite'}: ${command}; status=${result.status}; error=${result.error || ''}\n${result.stdout || ''}\n${result.stderr || ''}`;
  assert.equal(result.status, 0, diagnostic);
  const events = result.stdout.split('\n').filter(Boolean).map(line => JSON.parse(line));
  if (name) {
    const terminal = events.filter(e => e.Test === name && ['pass', 'fail', 'skip'].includes(e.Action));
    assert.equal(terminal.length, 1, `${diagnostic}\nrequired test ${name} absent or ran zero/multiple times`);
    assert.equal(terminal[0].Action, 'pass', `${diagnostic}\nrequired test ${name} did not pass`);
  } else {
    assert.ok(events.some(e => e.Test && e.Action === 'pass'), `${diagnostic}\nno tests ran`);
    assert.ok(events.some(e => !e.Test && e.Action === 'pass' && e.Package.endsWith('/shape')), `${diagnostic}\nshape package did not pass`);
  }
}
Then('k-shape-refine-retry-keeps-corrections: existing test {string} passes', { timeout: 250000 }, function (name) {
  assert.match(name, /^Test[A-Za-z0-9]+$/);
  run(name);
});
Then('k-shape-refine-retry-keeps-corrections: the existing shaping suite passes', { timeout: 250000 }, function () {
  run();
});
