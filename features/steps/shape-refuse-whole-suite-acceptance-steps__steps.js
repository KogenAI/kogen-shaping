'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const P = 'shape-refuse-whole-suite-acceptance-steps: ';
const TEST = 'TestWholeSuiteContractAcceptance';
let compiled;
function environment(extra = {}) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}
function build() {
  if (compiled) return compiled;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'whole-suite-contract-'));
  process.on('exit', () => fs.rmSync(dir, {recursive:true, force:true}));
  const source = path.join(dir, 'probe_test.go');
  fs.writeFileSync(source, fs.readFileSync('features/steps/shape-refuse-whole-suite-acceptance-steps__driver/probe_test.go', 'utf8').replace('//go:build ignore\n\n', ''));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace:{[path.resolve('shape/whole_suite_contract_acceptance_test.go')]:source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {env:environment(), encoding:'utf8', timeout:100000});
  assert.equal(r.status, 0, `${TEST}: compile go test -c ./shape: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`);
  return compiled = {dir, bin};
}
function run(input) {
  const {dir,bin} = build();
  const out = path.join(dir, 'report.json');
  fs.rmSync(out, {force:true});
  const args = ['-test.run', '^'+TEST+'$', '-test.count=1', '-test.timeout=40s'];
  const r = spawnSync(bin,args,{cwd:path.resolve('shape'),env:environment({WS_INPUT:JSON.stringify(input),WS_OUT:out}),encoding:'utf8',timeout:45000});
  const diagnostic = `${TEST}: ${bin} ${args.join(' ')} ./shape: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`;
  assert.equal(r.status,0,diagnostic);
  assert.ok(fs.existsSync(out), `${diagnostic}\nrequired test absent, skipped or ran zero times: no report`);
  return JSON.parse(fs.readFileSync(out,'utf8'));
}
function inspect(o, expected) {
  assert.equal(o.seconds,120,'actual approval per-step limit');
  assert.equal(o.checks.length,1,'exactly one real step_selfcheck event');
  const event = o.checks[0];
  const limits = event.items.filter(i => i.check === 'step_limit');
  if (expected.Outcome === 'refused') {
    assert.equal(event.ok,false,'static self-check refuses suite');
    assert.equal(limits.length,1,'one suite defect even for concurrent commands');
    assert.equal(limits[0].target,'is-even: '+expected.Sentence,'defect identifies offending definition');
    assert.ok(o.feedback.includes('step_limit is-even: '+expected.Sentence+':'),'refusal returned to shaper');
    assert.ok(o.feedback.includes('whole test suite with '+expected.Runner),'runner named');
    assert.ok(o.feedback.includes('120 s step limit'),'approval limit named');
  } else {
    assert.equal(expected.Outcome,'allowed','known expected outcome');
    assert.equal(event.ok,true,'targeted command passes existing static checks');
    assert.deepEqual(limits,[],'targeted command not flagged');
    assert.equal(o.readiness,true,'targeted candidate reaches readiness');
  }
}
function policy(o, snippets) {
  for (const snippet of snippets) assert.ok(o.policy.includes(snippet), 'rendered shaper contract missing: '+snippet);
}
if (require.main === module) {
  const mode = process.argv[2];
  if (mode === 'record') {
    const cases = JSON.parse(fs.readFileSync(process.argv[3], 'utf8'));
    process.stdout.write(JSON.stringify(cases.map(expected => ({expected, actual:run(expected)})))+'\n');
  } else if (mode === 'qualify') {
    const records = JSON.parse(fs.readFileSync('.kogen/intents/shape-refuse-whole-suite-acceptance-steps/observer-good.json','utf8'));
    const judge = rs => { for (const r of rs) { inspect(r.actual,r.expected); if (r.expected.Policy) policy(r.actual,r.expected.Policy); } };
    judge(records);
    if (process.argv[3] === 'good') { console.log('accepted'); }
    else if (process.argv[3] === 'bad') {
      const changes = [
        rs => { rs[0].actual.seconds = 119; },
        rs => { rs[0].actual.checks = []; },
        rs => { rs[0].actual.checks[0].ok = true; },
        rs => { rs[0].actual.checks[0].items = []; },
        rs => { rs[0].actual.checks[0].items.find(i=>i.check==='step_limit').target = 'wrong'; },
        rs => { rs[0].actual.feedback = ''; },
        rs => { rs[0].actual.feedback = rs[0].actual.feedback.replace('go test','other'); },
        rs => { rs[0].actual.feedback = rs[0].actual.feedback.replace('120 s','119 s'); },
        rs => { rs[1].actual.checks[0].ok = false; },
        rs => { rs[1].actual.checks[0].items.push({check:'step_limit'}); },
        rs => { rs[1].actual.readiness = false; },
        rs => { rs[1].actual.policy = ''; }
      ];
      for (const change of changes) { const rs = JSON.parse(JSON.stringify(records)); change(rs); assert.throws(()=>judge(rs),'observer accepted wrong observation'); }
      console.log('rejected'); process.exitCode = 1;
    } else throw new Error('expected good or bad');
  } else throw new Error('expected record or qualify');
} else {
  const {Given,When,Then} = require('@cucumber/cucumber');
  Given(P+'the independent Shape harness is compiled', {timeout:110000}, function() { build(); });
  When(P+'Shape inspects this submitted command', {timeout:55000}, function(text) { this.wsExpected = JSON.parse(text); this.wsReport = run(this.wsExpected); });
  Then(P+'the static outcome matches without executing the command', function() { inspect(this.wsReport,this.wsExpected); });
  Then(P+'the rendered shaper contract contains these instructions', function(text) { policy(this.wsReport,JSON.parse(text)); });
}
