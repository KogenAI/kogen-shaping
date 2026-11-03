'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKAcceptEnvPrepareAcceptance';
const BUILD = 'go test -c -vet=off -overlay ./shape';
let built; // the test binary, built once per process
function binary(env) {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'acceptenv-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_accept_env_acceptance_test.go': 'features/steps/k_accept_env_test.go.txt'}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env, encoding: 'utf8', timeout: 100000});
  built = r.status === 0 ? {bin} : {error: `${TEST}: ${BUILD} failed (exit ${r.status} ${r.error || ''}): ${r.stdout}${r.stderr}`};
  return built;
}
function run(mode) {
  const env = {...process.env, KENV_MODE: mode};
  for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
  const b = binary(env);
  if (b.error) return {ok: false, detail: `${b.error} (mode ${mode})`};
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=90s'];
  const r = spawnSync(b.bin, args, {cwd: path.join(process.cwd(), 'shape'), env, encoding: 'utf8', timeout: 100000, maxBuffer: 16 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const passed = text.includes('--- PASS: ' + TEST + ' ');
  return {ok: !r.error && r.status === 0 && passed,
    detail: `${TEST} (KENV_MODE=${mode}): shape.test ${args.join(' ')} (built with ${BUILD}) exit=${r.status} error=${r.error || ''}${passed ? '' : `; no PASS line for ${TEST}`}\n${text}`};
}
if (require.main === module) {
  const r = run(process.argv[2]);
  console.log(r.ok ? 'accepted' : 'rejected');
  process.exitCode = r.ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('the environment journey runs with an acceptance step that depends on {string}', {timeout: 210000}, function (mode) { this.envJourney = run(mode); });
  Then('the step comes back to the shaper with its fresh-copy output and its correction is published', function () { assert.ok(this.envJourney.ok, this.envJourney.detail); });
}
