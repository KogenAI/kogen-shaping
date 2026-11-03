'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKStepcheckRepairAcceptance';
function run(mode) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'steprepair-'));
  try {
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_stepcheck_repair_acceptance_test.go': 'features/steps/k_stepcheck_repair_test.go.txt'}}));
    const env = {...process.env, KSTEP_MODE: mode};
    for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
    const bin = path.join(dir, 'shape.test');
    const build = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env, encoding: 'utf8', timeout: 100000});
    if (build.status !== 0) return {ok: false, detail: `${TEST}: go test -c -vet=off -overlay ./shape build failed: ${build.stdout}${build.stderr}`};
    const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=60s'];
    const r = spawnSync(bin, args, {cwd: path.join(process.cwd(), 'shape'), env, encoding: 'utf8', timeout: 90000, maxBuffer: 8 * 1024 * 1024});
    const text = (r.stdout || '') + (r.stderr || '');
    return {ok: !r.error && r.status === 0 && text.includes('--- PASS: ' + TEST + ' '), detail: `${TEST}: shape.test ${args.join(' ')} (go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`};
  } finally { fs.rmSync(dir, {recursive: true, force: true}); }
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('the step-check correction journey runs in {string} mode', {timeout: 200000}, function (mode) { this.stepRepair = run(mode); });
  Then('the step-check correction contract passes', function () { assert.ok(this.stepRepair.ok, this.stepRepair.detail); });
}
