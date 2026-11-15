// Runs the Go test in .kogen/intents/shape-time-splits-checks-by-gate/gate_split_test.go.txt inside
// package shape through go test -overlay: the checkout gains no file.
const { Given, Then } = require('@cucumber/cucumber');
const assert = require('assert');
const path = require('path');
const fs = require('fs');
const os = require('os');
const { spawnSync } = require('child_process');

const SLUG = 'shape-time-splits-checks-by-gate';
const TEST = 'TestShapeTimeGateSplitAcceptance';

Given('shape-time-splits-checks-by-gate: a scripted shaping run that passes a 300 ms static check, 300 ms decision and literal checks and a 150 ms contract review', function () {
  this.gateRun = 'with_a_contract_review';
});

Given('shape-time-splits-checks-by-gate: a scripted shaping run refused before any contract review', function () {
  this.gateRun = 'without_a_contract_review';
});

function runGateTest(world) {
  const root = process.cwd();
  const source = path.join(root, '.kogen', 'intents', SLUG, 'gate_split_test.go.txt');
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'gate-split-'));
  const overlay = path.join(dir, 'overlay.json');
  const target = path.join(root, 'shape', 'zz_gate_split_acceptance_test.go');
  fs.writeFileSync(overlay, JSON.stringify({ Replace: { [target]: source } }));
  const run = '^' + TEST + '$/^' + world.gateRun + '$';
  const args = ['test', '-overlay', overlay, '-count=1', '-v', '-run', run, './shape'];
  const r = spawnSync('go', args, { cwd: root, encoding: 'utf8', timeout: 110000, maxBuffer: 16 * 1024 * 1024 });
  const out = (r.stdout || '') + (r.stderr || '');
  const pass = new RegExp('--- PASS: ' + TEST + '/' + world.gateRun + ' ');
  assert.ok(r.status === 0 && pass.test(out),
    'go ' + args.join(' ') + ' in ' + root + ': exit ' + r.status + (r.error ? ' (' + r.error.message + ')' : '') +
    ', test ' + TEST + '/' + world.gateRun + (pass.test(out) ? ' passed' : ' did not pass') + '\n' + out.slice(-4000));
}

Then('shape-time-splits-checks-by-gate: the shape_time event and the progress line split the Kogen checks by gate', function () {
  runGateTest(this);
});

Then('shape-time-splits-checks-by-gate: the shape_time event reports contract_ms 0 and the progress line names all four gates', function () {
  runGateTest(this);
});
