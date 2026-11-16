'use strict';
const fs=require('node:fs'), os=require('node:os'), path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
const P='shape-kept-repeat-reuses-unpassable-control: ';
const D='features/steps/shape-kept-repeat-reuses-unpassable-control__driver';
const TEST='TestRepeatControlObservation';
let built;
function env(extra={}) { const e={...process.env,...extra}; for(const k of Object.keys(e)) if(k.startsWith('KOGEN_ACCEPTANCE_'))delete e[k];return e; }
function run(mode) {
 if(!built) {
  const dir=fs.mkdtempSync(path.join(os.tmpdir(),'repeat-control-'));
  process.on('exit',()=>fs.rmSync(dir,{recursive:true,force:true}));
  const source=path.join(dir,'observe_test.go');
  fs.writeFileSync(source,fs.readFileSync(D+'/observe_test.go','utf8').replace('//go:build ignore\n\n',''));
  const overlay=path.join(dir,'overlay.json');
  fs.writeFileSync(overlay,JSON.stringify({Replace:{[path.resolve('shape/repeat_control_acceptance_test.go')]:source}}));
  const bin=path.join(dir,'shape.test');
  const r=spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'],{env:env(),encoding:'utf8',timeout:120000});
  assert.equal(r.status,0,`${TEST}: go test -c -overlay ./shape exit=${r.status} ${r.error||''}\n${r.stdout}${r.stderr}`);
  built={dir,bin};
 }
 const out=path.join(built.dir,mode+'.json');fs.rmSync(out,{force:true});
 const args=['-test.run','^'+TEST+'$','-test.count=1','-test.v','-test.timeout=45s'];
 const r=spawnSync(built.bin,args,{cwd:path.resolve('shape'),env:env({REPEAT_CONTROL_MODE:mode,REPEAT_CONTROL_OUT:out}),encoding:'utf8',timeout:55000});
 assert.ok(r.status===0&&fs.existsSync(out)&&r.stdout.includes('--- PASS: '+TEST+' '),`${TEST}: shape.test ${args.join(' ')} exit=${r.status}; test absent, skipped, failed or ran zero times: ${r.error||''}\n${r.stdout}${r.stderr}`);
 return JSON.parse(fs.readFileSync(out,'utf8'));
}
function observe(o,e) {
 const why=JSON.stringify(o);
 for(const k of ['controls','readiness','same_package','same_tree','statuses']) if(k in e) assert.deepEqual(o[k],e[k],k+': '+why);
 assert.equal(o.checks.length,e.events,'unpassable_check count: '+why);
 assert.equal(o.repeats.length,e.repeats,'kept_assertion_repeat count: '+why);
 if(e.repeats) {
  assert.equal(o.repeats[0].outcome,'persistent kept assertion red',why);
  assert.equal(o.repeats[0].execution_cap,2,why);
  for(const text of ['repeating kept assertion observation (execution 2 of 2 maximum)','kept assertion repeat outcome: persistent kept assertion red in "kept" (line 2) (cap 2 executions)']) assert.ok(o.progress.includes(text),text+': '+why);
  assert.ok(o.refusals.some(s=>s.includes('red on the unchanged tree')&&s.includes('kept')), 'kept refusal changed: '+why);
 }
 if(e.skipped) assert.ok(o.checks[0].skipped.startsWith('the control run wrote no report'),'skip reason: '+why);
 const completed=o.checks.filter(c=>!c.skipped);
 for(const c of completed) {
  assert.equal(c.ok,!e.refusal,why);
  assert.equal(c.steps.length,1,why);
  assert.equal(c.steps[0].verdict,e.refusal?'unpassable':'stand_in_used',why);
  assert.equal(c.steps[0].step,'acceptance.feature line 3: Then step kept',why);
  assert.ok(c.steps[0].definition.startsWith('features/steps/is_even_steps.js line '),why);
 }
 if(e.refusal) {
  assert.equal(o.first_error,o.second_error,'cached refusal text changed: '+why);
  assert.equal(o.first_error,e.refusal_text,'refusal text: '+why);
 } else {
  if(e.error) assert.notEqual(o.first_error,'','failed control must propagate error: '+why); else assert.equal(o.first_error,'',why);
  assert.equal(o.second_error,'',why);
 }
}
module.exports={run,observe};
if(require.main===module) { const o=run(process.argv[2]);console.log(JSON.stringify(o)); }
else if(process.env.KOGEN_REPEAT_OBSERVER_LIBRARY!=='1') {
 const {Given,When,Then}=require('@cucumber/cucumber');
 Given(P+'the local control fixture is available',function(){assert.ok(fs.existsSync(D+'/observe_test.go'));});
 When(P+'the {string} journey runs',{timeout:180000},function(mode){this.repeatControl=run(mode);});
 Then(P+'the observation is',function(text){observe(this.repeatControl,JSON.parse(text));});
}
