'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKStepFileCollisionAcceptance';
function run(mode) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-step-file-collision-'));
  try {
    const root = process.cwd();
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {
      [path.join(root, 'shape/k_step_file_collision_acceptance_test.go')]: path.join(root, 'features/steps/k_step_file_collision_test.go.txt')
    }}));
    const env = {...process.env, KSC_MODE: mode};
    for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
    const bin = path.join(dir, 'shape.test');
    const build = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env, encoding: 'utf8', timeout: 100000});
    if (build.error || build.status !== 0) return {ok: false, detail: `${TEST}: go test -c -vet=off -overlay ./shape failed (exit ${build.status}): ${build.error || ''}\n${build.stdout}${build.stderr}`};
    const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=90s'];
    const r = spawnSync(bin, args, {cwd: path.join(root, 'shape'), env, encoding: 'utf8', timeout: 100000, maxBuffer: 8 * 1024 * 1024});
    const text = (r.stdout || '') + (r.stderr || '');
    const ran = text.includes('--- PASS: ' + TEST + ' ');
    const ok = !r.error && r.status === 0 && ran && text.includes('KSC-OK\n') && !text.includes('--- SKIP:');
    return {ok, detail: `${TEST}: ${bin} ${args.join(' ')} (go test -c -vet=off -overlay ./shape) exit=${r.status}; error=${r.error || ''}; missing/skipped/zero run=${!ran}\n${text}`};
  } finally { fs.rmSync(dir, {recursive: true, force: true}); }
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  if (process.argv[3] === '--diagnostic') console.error(r.detail);
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('the step-file ownership journey runs in {string} mode', {timeout: 210000}, function (mode) { this.kscResult = run(mode); });
  Then('the step-file ownership journey satisfies the contract', function () { assert.ok(this.kscResult.ok, this.kscResult.detail); });
}
