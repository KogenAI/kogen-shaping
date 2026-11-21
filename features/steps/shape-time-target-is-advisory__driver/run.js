'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');
const harness = path.join(__dirname, 'observe_test.go.txt');
const test = 'TestShapeTimeTargetObservation';
function run() {
  const root = process.cwd();
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-time-target-'));
  try {
    const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('KOGEN_ACCEPTANCE_')));
    const overlay = path.join(dir, 'overlay.json');
    const binary = path.join(dir, 'shape.test');
    const output = path.join(dir, 'observation.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(root, 'shape/shape_time_target_observation_test.go')]: harness}}));
    const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', binary, './shape'];
    const built = cp.spawnSync('go', args, {env, encoding: 'utf8', timeout: 90000, maxBuffer: 8 * 1024 * 1024});
    assert.equal(built.status, 0, `${test}: go ${args.join(' ')} failed: ${built.error || ''}\n${built.stdout || ''}${built.stderr || ''}`);
    const r = cp.spawnSync(binary, ['-test.run', '^' + test + '$', '-test.timeout=60s'], {env: {...env, STT_OUT: output}, encoding: 'utf8', timeout: 70000, maxBuffer: 8 * 1024 * 1024});
    assert.equal(r.status, 0, `${test}: ${binary} failed or ran zero times: ${r.error || ''}\n${r.stdout || ''}${r.stderr || ''}`);
    assert.ok(fs.existsSync(output), `${test}: no observation file; test absent, skipped or ran zero times: ${r.stdout || ''}${r.stderr || ''}`);
    return JSON.parse(fs.readFileSync(output, 'utf8'));
  } finally {
    fs.rmSync(dir, {recursive: true, force: true});
  }
}
module.exports = {run, harness};
