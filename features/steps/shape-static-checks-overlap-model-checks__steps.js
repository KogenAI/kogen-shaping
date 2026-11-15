'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const P = 'shape-static-checks-overlap-model-checks: ';
const DRIVER = 'features/steps/shape-static-checks-overlap-model-checks__driver/observe_test.go';
const TEST = 'TestStaticModelOverlapObservation';
let built;
function environment(extra = {}) {
  const env = {...process.env, ...extra};
  for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
  return env;
}
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'static-model-overlap-'));
  process.on('exit', () => fs.rmSync(dir, {recursive:true, force:true}));
  const source = path.join(dir, 'observe_test.go');
  fs.writeFileSync(source, fs.readFileSync(DRIVER, 'utf8').replace('//go:build ignore\n\n', ''));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace:{[path.resolve('shape/overlap_acceptance_test.go')]:source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {env:environment(), encoding:'utf8', timeout:75000});
  assert.equal(r.status, 0, `${TEST}: go test -c -overlay ./shape: exit=${r.status} ${r.error || ''}\n${r.stdout}${r.stderr}`);
  return built = {dir,bin};
}
function run(mode) {
  const {dir,bin} = build();
  const out = path.join(dir, mode+'.json');
  fs.rmSync(out, {force:true});
  const args = ['-test.run','^'+TEST+'$','-test.count=1','-test.v','-test.timeout=20s'];
  const r = spawnSync(bin, args, {cwd:path.resolve('shape'), env:environment({OVERLAP_MODE:mode,OVERLAP_OUT:out}), encoding:'utf8', timeout:25000});
  assert.ok(r.status === 0 && fs.existsSync(out) && r.stdout.includes('--- PASS: '+TEST+' '), `${TEST}: shape.test ${args.join(' ')}: exit=${r.status}; absent, skipped, zero runs or failed: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}
function observe(report, expected) {
  assert.ok(report.elapsed_ms >= expected.min_ms && report.elapsed_ms < expected.max_ms, `elapsed ${report.elapsed_ms} ms, expected [${expected.min_ms},${expected.max_ms}); two independent waits must overlap`);
  for (const [key,value] of Object.entries(expected)) {
    if (key !== 'min_ms' && key !== 'max_ms') assert.deepEqual(report[key], value, key+': '+JSON.stringify(report));
  }
}
module.exports = {run,observe};
if (require.main === module) {
  const cases = JSON.parse(fs.readFileSync(process.argv[3], 'utf8'));
  if (process.argv[2] === 'good') {
    for (const c of cases) observe(c.report,c.expected);
    console.log('observer accepted');
  } else {
    let rejected = 0;
    for (const c of cases) { try { observe(c.report,c.expected); } catch (_) { rejected++; } }
    if (rejected !== cases.length || rejected === 0) { console.log('observer control failed'); process.exitCode = 2; }
    else { console.log('observer rejected'); process.exitCode = 1; }
  }
} else {
  const {Given,When,Then} = require('@cucumber/cucumber');
  Given(P+'the local scripted fixture is available', function () { assert.ok(fs.existsSync(DRIVER)); });
  When(P+'one {string} submission runs with two-second static and decision waits', {timeout:115000}, function (mode) { this.overlapReport = run(mode); });
  Then(P+'the observed submission is', function (text) { observe(this.overlapReport,JSON.parse(text)); });
}
