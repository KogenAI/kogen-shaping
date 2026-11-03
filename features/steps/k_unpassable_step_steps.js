'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKUnpassableStepAcceptance';
let built;
function env(mode) {
  const e = {...process.env};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  if (mode) e.KUNP_MODE = mode;
  return e;
}
// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'unpassable-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_unpassable_step_acceptance_test.go': 'features/steps/k_unpassable_step_test.go.txt'}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 100000});
  built = r.status === 0 ? {bin} : {error: `${TEST}: go test -c -vet=off -overlay ./shape build failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}
function run(mode) {
  const b = build();
  if (b.error) return {ok: false, detail: b.error};
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=60s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env(mode), encoding: 'utf8', timeout: 90000, maxBuffer: 16 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  return {ok: !r.error && r.status === 0 && text.includes('--- PASS: ' + TEST + ' '),
    detail: `${TEST} (KUNP_MODE=${mode}): shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('the unpassable-step journey runs in {string} mode', {timeout: 200000}, function (mode) { this.unpassable = run(mode); });
  Then('the unpassable-step contract passes', function () { assert.ok(this.unpassable.ok, this.unpassable.detail); });
}
