'use strict';
// K-SHAPE-RUNNER-GUIDANCE-GENERAL: compiles the shape tests with
// features/steps/k_shape_runner_guidance_test.go.txt through a Go overlay and runs
// TestKShapeRunnerGuidanceAcceptance in one mode: guidance and refusal run Shape with a scripted
// provider and a synthetic acceptance runner; good and bad are the observer controls.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKShapeRunnerGuidanceAcceptance';
let built;
// quiet keeps the synthetic runner output quoted in a failure from reading as this acceptance's
// own test-runner output.
const quiet = (text) => text
  .replace(/-run([= ])/g, '-run:$1')
  .replace(/(-n|-i|--name|--include)([\s=])/g, '$1:$2')
  .replace(/ assertions, /g, ' assertions; ');
function env(mode) {
  const e = {...process.env};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  if (mode) e.KRG_MODE = mode;
  return e;
}
// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'krg-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_shape_runner_guidance_acceptance_test.go': 'features/steps/k_shape_runner_guidance_test.go.txt'}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 200000});
  built = r.status === 0 ? {bin} : {error: quiet(`${TEST}: go test -c -vet=off -overlay ./shape build failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`)};
  return built;
}
function run(mode) {
  const b = build();
  if (b.error) return {ok: false, detail: b.error};
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=90s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env(mode), encoding: 'utf8', timeout: 110000, maxBuffer: 16 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  return {ok: !r.error && r.status === 0 && text.includes('--- PASS: ' + TEST + ' '),
    detail: quiet(`${TEST} (KRG_MODE=${mode}): shape.test filtered to ${TEST} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`)};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('k-shape-runner-guidance: the {string} journey runs', {timeout: 330000}, function (mode) { this.krg = run(mode); });
  Then('k-shape-runner-guidance: the journey meets the general runner-observer contract', function () { assert.ok(this.krg.ok, this.krg.detail); });
}
