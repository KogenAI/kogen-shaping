'use strict';
const fs=require('node:fs'),os=require('node:os'),path=require('node:path'),cp=require('node:child_process'),assert=require('node:assert/strict');
const base='features/steps/shape-repeated-decision-statement-says-so';
let built;
function env(extra={}) { const e={...process.env,...extra}; for(const k of Object.keys(e)) if(k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k]; return e; }
function build() {
 if(built) return built;
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'repeated-statement-'));
 process.on('exit',()=>fs.rmSync(dir,{recursive:true,force:true}));
 const source=path.join(dir,'observe_test.go'),overlay=path.join(dir,'overlay.json'),bin=path.join(dir,'shape.test');
 fs.writeFileSync(source,fs.readFileSync(base+'__driver/observe_test.go','utf8').replace('//go:build ignore\n\n',''));
 fs.writeFileSync(overlay,JSON.stringify({Replace:{[path.resolve('shape/repeated_statement_acceptance_test.go')]:source}}));
 const r=cp.spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'],{env:env(),encoding:'utf8',timeout:70000});
 assert.equal(r.status,0,`compile TestRepeatedStatementObservation via go test -c ./shape: ${r.error||''}\n${r.stdout}${r.stderr}`);
 return built={dir,bin};
}
function run(c) {
 const {dir,bin}=build(),out=path.join(dir,'report.json'); fs.rmSync(out,{force:true});
 const r=cp.spawnSync(bin,['-test.run','^TestRepeatedStatementObservation$','-test.count=1','-test.timeout=35s'],{cwd:path.resolve('shape'),env:env({REPEATED_CASE:JSON.stringify(c),REPEATED_OUT:out}),encoding:'utf8',timeout:40000});
 assert.equal(r.status,0,`TestRepeatedStatementObservation built ./shape test outcome ${r.status}: ${r.error||''}\n${r.stdout}${r.stderr}`);
 assert.ok(fs.existsSync(out),'TestRepeatedStatementObservation built ./shape: absent, skipped or zero runs; no report');
 return JSON.parse(fs.readFileSync(out,'utf8'));
}
function observe(o,c,expected) {
 assert.equal(o.reports.length,c.rounds.length,'one result per submission');
 assert.equal(o.requests.length,c.rounds.length,'the checker must still be called once per submitted candidate');
 assert.equal(o.events.length,c.rounds.length,'one decision_check event per call');
 for(let i=0;i<c.rounds.length;i++) {
  const r=c.rounds[i],feedback=o.reports[i].feedback;
  const item=o.events[i].items[0];
  assert.equal(o.events[i].items.length,1,'one statement');
  assert.equal(item.statement,r.statement,'statement bytes retained');
  assert.deepEqual(item.request_quotes,[c.request],'quotes retained');
  assert.equal(item.verdict,r.verdict,'verdict unchanged');
  assert.equal(item.why,c.why,'reason unchanged');
  assert.equal(o.requests[i].instructions,o.requests[0].instructions,'checker instructions unchanged across submissions');
  const input=(o.requests[i].input||[]).map(x=>x.content||'').join('\n');
  assert.ok(input.includes(r.statement),'current candidate reaches model');
  assert.ok(!input.includes('unchanged since submission'),'history note must not change model consultation');
  const notes=feedback.split('\n').filter(l=>l.startsWith('note: this statement'));
  assert.deepEqual(notes,expected[i] ? [expected[i]] : [],`submission ${r.submission}: repetition note`);
  if(r.verdict==='follows') { assert.equal(feedback,'','passing check must stay passing'); continue; }
  const head='Decision check: 1 stated decisions are not supported by the request; resolve each explicit conflict using the requirement below:';
  const line=`statement: ${r.statement} | request_quotes: "${c.request}" | verdict: contradicts | intent_value: ${r.intentValue} | request_value: ${r.requestValue} | difference: ${c.difference} | why: ${c.why}`;
  const fix=`Replace ${JSON.stringify(r.intentValue)} with the request's requirement ${JSON.stringify(r.requestValue)}. Align the acceptance expectations with that requirement.`;
  const lines=feedback.split('\n'),at=lines.indexOf(head);
  assert.ok(at>=0,'original refusal headline preserved');
  assert.equal(lines[at+1],line,'original statement line preserved');
  assert.match(lines[at+2],/^fix for intent\.md line \d+: /,'original located fix preserved');
  assert.ok(lines[at+2].endsWith(': '+fix),'original fix bytes preserved');
  if(expected[i]) assert.equal(lines[at+3],expected[i],'note immediately follows its fix');
  if(c.mode!=='submit') assert.equal(lines.length,3+(expected[i]?1:0),'no other feedback changes');
 }
}
module.exports={run,observe};
if(require.main!==module && !process.env.REPEATED_OBSERVER_LIBRARY) {
 const {Given,When,Then,setDefaultTimeout}=require('@cucumber/cucumber'); setDefaultTimeout(120000);
 const p='shape-repeated-decision-statement-says-so: ';
 Given(p+'the scripted local decision submissions',function(text){this.repeatedCase=JSON.parse(text);});
 When(p+'the decision submissions are checked',function(){this.repeatedReport=run(this.repeatedCase);});
 Then(p+'only the expected additive notes appear',function(text){observe(this.repeatedReport,this.repeatedCase,JSON.parse(text));});
} else if(process.argv[2]==='record') {
 const c=JSON.parse(fs.readFileSync(process.argv[3],'utf8')); process.stdout.write(JSON.stringify(run(c))+'\n');
}
