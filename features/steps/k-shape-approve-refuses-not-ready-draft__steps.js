'use strict';
const {Then}=require('@cucumber/cucumber');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const cp=require('node:child_process');
let result;
function journey(){
 if(result)return result;
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'draft-approval-'));
 try {
  const tree=path.join(root,'tree');
  fs.cpSync(process.cwd(),tree,{recursive:true,filter:p=>!['.git','node_modules'].includes(path.basename(p))});
  let source=require('./k-shape-approve-refuses-not-ready-draft__harness');
  source=source.replace('"github.com/KogenAI/kogen-loop/sys"','"github.com/KogenAI/kogen-loop/sys"\n "github.com/KogenAI/kogen-verification/verify"');
  source+='\nfunc verifyToolsForDraft() verify.Tools { return verify.Tools{Node: os.Getenv("KOGEN_NODE"), Cucumber: os.Getenv("KOGEN_CUCUMBER"), Quint: os.Getenv("KOGEN_QUINT")} }\n';
  fs.writeFileSync(path.join(tree,'shape','draft_approval_acceptance_test.go'),source);
  const binary=path.join(root,'draft.test');
  const build=cp.spawnSync('go',['test','-c','-o',binary,'./shape'],{cwd:tree,encoding:'utf8',timeout:120000});
  assert.equal(build.status,0,'compile draft acceptance: '+build.stdout+build.stderr);
  const report=path.join(root,'result.json');
  const run=cp.spawnSync(binary,['-test.run=^TestDraftApprovalJourney$','-test.v','-test.timeout=100s'],{cwd:tree,env:{...process.env,DRAFT_RESULT:report},encoding:'utf8',timeout:110000});
  assert.equal(run.status,0,'TestDraftApprovalJourney via built test binary: '+run.stdout+run.stderr);
  assert.ok(fs.existsSync(report),'TestDraftApprovalJourney must write its observed report');
  result=JSON.parse(fs.readFileSync(report,'utf8'));return result;
 } finally { fs.rmSync(root,{recursive:true,force:true}); }
}
function text(r){return [r.Detail,...(r.Lines||[]),r.Output].join('\n');}
Then('k-shape-approve-refuses-not-ready-draft: {string} obeys the draft approval contract',function(rule){
 const r=journey();
 const next='kogen intent shape is-even <request-file>';
 const refuses=x=>{assert.notEqual(x.Exit,0);assert.equal(x.Class,'intent');assert.equal(x.Reason,'not_ready');assert.equal(x.Marker,false);assert.ok(text(x).includes(next),text(x));assert.match(text(x),/undefined|readiness|acceptance/i);assert.ok(!x.Output.includes('Approve with:'),x.Output);};
 if(rule==='sticky refusal'){
  assert.equal(r.notReady.Reason,'not_ready');
  for(const key of ['blockedCard','blockedHash','staleHash','handEdit'])refuses(r[key]);
 }else if(rule==='successful retry'){
  assert.equal(r.retry.Exit,0,text(r.retry));assert.equal(r.retry.Reason,'');
  assert.equal(r.readyCard.Exit,5,text(r.readyCard));assert.equal(r.readyCard.Reason,'');assert.ok(r.readyCard.Output.includes('Approve with:'));
  assert.equal(r.readyApproval.Exit,0,text(r.readyApproval));assert.equal(r.readyApproval.Marker,true);
 }else if(rule==='never shaped'){
  assert.equal(r.legacyCard.Exit,5,text(r.legacyCard));assert.equal(r.legacyCard.Reason,'');assert.ok(r.legacyCard.Output.includes('Approve with:'));assert.equal(r.legacyCard.Marker,false);
  assert.equal(r.legacyApproval.Exit,0,text(r.legacyApproval));assert.equal(r.legacyApproval.Marker,true);
 }else if(rule==='actionable diagnostics'){
  assert.equal(r.notReady.Reason,'not_ready');assert.equal(r.exists.Reason,'exists');
  for(const x of [r.notReady,r.exists]){
   assert.ok(text(x).trimEnd().endsWith(next),text(x));
   for(const p of r.files)assert.ok(text(x).includes(p),p+' missing from '+text(x));
  }
 }else assert.fail('unknown rule '+rule);
});
