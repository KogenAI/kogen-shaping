'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const prefix = 'shape-decision-check-keep-unchanged: ';
let built;
function environment(extra = {}) {
  const env = {...process.env, ...extra};
  for (const key of Object.keys(env)) if (key.startsWith('KOGEN_ACCEPTANCE_')) delete env[key];
  return env;
}
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'keep-unchanged-'));
  process.on('exit', () => fs.rmSync(dir, {recursive:true, force:true}));
  const overlay = path.join(dir, 'overlay.json');
  const source = path.join(dir, 'probe_test.go');
  fs.writeFileSync(source, fs.readFileSync('features/steps/shape-decision-check-keep-unchanged__driver/probe_test.go','utf8').replace('//go:build ignore\n\n', ''));
  fs.writeFileSync(overlay, JSON.stringify({Replace:{[path.resolve('shape/keep_unchanged_acceptance_test.go')]:source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {env:environment(),encoding:'utf8',timeout:120000});
  assert.equal(r.status, 0, `compile TestKeepUnchangedAcceptance with go test -c ./shape: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  return built = {dir,bin};
}
function run(c) {
  const {dir,bin} = build();
  const out = path.join(dir, 'report.json');
  fs.rmSync(out, {force:true});
  const r = spawnSync(bin, ['-test.run','^TestKeepUnchangedAcceptance$','-test.count=1','-test.timeout=30s'], {cwd:path.resolve('shape'),env:environment({KEEP_CASE:JSON.stringify(c),KEEP_OUT:out}),encoding:'utf8',timeout:40000});
  assert.equal(r.status, 0, `TestKeepUnchangedAcceptance: built shape.test ./shape outcome ${r.status}: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  assert.ok(fs.existsSync(out), 'TestKeepUnchangedAcceptance: built shape.test ./shape did not produce a report (absent, skipped or zero runs)');
  return JSON.parse(fs.readFileSync(out,'utf8'));
}
function observe(o, expected) {
  assert.deepEqual(o.evidenceProblems || [], [], 'evidence citation must qualify');
  assert.ok(o.event, 'missing decision_check event');
  const items = o.event.items;
  assert.equal(items.length, 1, 'expected exactly one decision_check item');
  assert.equal(items[0].statement, o.statement);
  assert.deepEqual(items[0].request_quotes, [], 'no request quote may support these choices');
  assert.equal(items[0].verdict, expected.Verdict, 'recorded verdict');
  assert.equal(o.event[expected.Verdict], 1, 'verdict count');
  assert.equal((o.problems || []).length === 0, expected.Verdict === 'follows', 'gate acceptance/refusal');
  if (expected.Verdict !== 'follows') assert.ok(o.problems.some(p => p.includes('verdict: unsupported')), 'unsupported feedback missing');
  if (expected.Evidence) assert.ok(o.input.includes(expected.Evidence), 'observed evidence did not reach checker');
  for (const text of expected.Policy || []) assert.ok(o.instructions.includes(text), `checker policy missing: ${text}`);
}
if (require.main === module) {
  if (process.argv[2] === 'observe') {
    try {
      for (const {report,expected} of JSON.parse(fs.readFileSync(process.argv[3],'utf8'))) observe(report, expected);
      console.log('observer accepted');
    } catch (_) { console.log('observer rejected'); process.exitCode = 1; }
  } else if (process.argv[2] === 'record') {
    const cases = JSON.parse(fs.readFileSync(process.argv[3],'utf8'));
    process.stdout.write(JSON.stringify(cases.map(expected => ({report:run(expected),expected}))) + '\n');
  }
} else {
  const {Given,When,Then,setDefaultTimeout} = require('@cucumber/cucumber');
  setDefaultTimeout(280000);
  Given(prefix + 'the local checker case', function(text) { this.keepCase = JSON.parse(text); });
  When(prefix + 'decision_check evaluates the candidate', function() { this.keepReport = run(this.keepCase); });
  Then(prefix + 'the evidence-backed verdict and checker policy hold', function() { observe(this.keepReport, this.keepCase); });
}
