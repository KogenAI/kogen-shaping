'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const prefix = 'shape-keep-existing-decisions-supported: ';
let built;
function environment(extra = {}) {
  const env = {...process.env, ...extra};
  for (const key of Object.keys(env)) if (key.startsWith('KOGEN_ACCEPTANCE_')) delete env[key];
  return env;
}
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'keep-existing-'));
  process.on('exit', () => fs.rmSync(dir, {recursive:true, force:true}));
  const overlay = path.join(dir, 'overlay.json');
  const source = path.join(dir, 'probe_test.go');
  fs.writeFileSync(source, fs.readFileSync('features/steps/shape-keep-existing-decisions-supported__driver/probe_test.go','utf8').replace('//go:build ignore\n\n', ''));
  fs.writeFileSync(overlay, JSON.stringify({Replace:{[path.resolve('shape/keep_existing_acceptance_test.go')]:source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {env:environment(),encoding:'utf8',timeout:120000});
  assert.equal(r.status, 0, `compile TestKeepExistingAcceptance with go test -c ./shape: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  return built = {dir,bin};
}
function run(c) {
  const {dir,bin} = build();
  const out = path.join(dir, 'report.json');
  fs.rmSync(out, {force:true});
  const r = spawnSync(bin, ['-test.run','^TestKeepExistingAcceptance$','-test.count=1','-test.timeout=30s'], {cwd:path.resolve('shape'),env:environment({KEEP_CASE:JSON.stringify(c),KEEP_OUT:out}),encoding:'utf8',timeout:40000});
  assert.equal(r.status, 0, `TestKeepExistingAcceptance: built shape.test ./shape outcome ${r.status}: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  assert.ok(fs.existsSync(out), 'TestKeepExistingAcceptance: built shape.test ./shape did not produce a report (absent, skipped or zero runs)');
  return JSON.parse(fs.readFileSync(out,'utf8'));
}
const {Given,When,Then,setDefaultTimeout} = require('@cucumber/cucumber');
setDefaultTimeout(280000);
Given(prefix + 'the case', function(text) { this.keepCase = JSON.parse(text); });
When(prefix + 'decision_check evaluates the candidate', function() { this.keepReport = run(this.keepCase); });
Then(prefix + 'the final verdict and feedback match the case', function() {
  const o = this.keepReport, expected = this.keepCase.Expect;
  assert.ok(o.event, 'missing decision_check event');
  assert.equal(o.event.items.length, 1, 'expected exactly one decision_check item');
  assert.deepEqual(o.event.items[0].request_quotes, this.keepCase.Quotes || [], 'request quotes of the item');
  assert.equal(o.event.items[0].verdict, expected, 'recorded final verdict');
  const problems = o.problems || [];
  if (expected === 'follows') {
    assert.deepEqual(problems, [], 'a supported keep-existing Decision must produce no feedback');
  } else {
    assert.ok(problems.some(p => p.includes('verdict: unsupported')), `unsupported refusal feedback missing: ${JSON.stringify(problems)}`);
  }
});
