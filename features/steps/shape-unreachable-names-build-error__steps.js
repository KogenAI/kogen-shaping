'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const P = 'shape-unreachable-names-build-error: ';
const slug = 'shape-unreachable-names-build-error';
const helper = path.resolve('features/steps/'+slug+'__driver/probe_test.go');
const test = 'TestUnreachableNamesAcceptance';
let dir, bin;
function build(impl) {
 dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sun-'));
 const source = path.join(dir,'probe_test.go');
 fs.writeFileSync(source,fs.readFileSync(helper,'utf8').replace('//go:build ignore',''));
 const replace = {[path.resolve('shape/sun_acceptance_test.go')]: source};
 for (const [p,c] of Object.entries(impl || {})) {
  const f = path.join(dir, 'file'+Object.keys(replace).length+'.go'); fs.writeFileSync(f,c); replace[path.resolve(p)] = f;
 }
 const overlay = path.join(dir,'overlay.json'); fs.writeFileSync(overlay,JSON.stringify({Replace:replace}));
 bin = path.join(dir,'shape.test');
 const r = spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {encoding:'utf8',timeout:180000});
 if(r.status !== 0) throw new Error('go test -c ./shape for '+test+': '+r.stdout+r.stderr);
}
function observe(message) {
 if (!bin) build();
 const out = path.join(dir,'out.json'); fs.rmSync(out,{force:true});
 const r = spawnSync(bin,['-test.run','^'+test+'$','-test.count=1'], {encoding:'utf8',timeout:30000,env:{...process.env,SUN_MESSAGE:message,SUN_OUT:out}});
 assert.equal(r.status,0,test+' built binary exited '+r.status+': '+r.stdout+r.stderr);
 assert.ok(fs.existsSync(out),test+' ran zero times or wrote no result: '+r.stdout+r.stderr);
 return JSON.parse(fs.readFileSync(out,'utf8'));
}
function judge(obs, expected, first) {
 assert.deepEqual(obs, {
  unreachable:['- compile (new) | request: compile | problem: no pass is reachable: with the correct implementation in place, scenario "compile" fails at "Then compile": '+expected],
  wrong:'wrong implementation "broken" fails scenario "compile" without an assertion: Error: '+expected,
  assertion:first
 });
}
function bounded(obs,n,len) {
 const prefix='- compile (new) | request: compile | problem: no pass is reachable: with the correct implementation in place, scenario "compile" fails at "Then compile": ';
 const reason=obs.unreachable[0].slice(prefix.length);
 assert.equal(reason.split(' | ').length-1,n);
 assert.equal([...reason].length,len);
 judge(obs,reason,'command:');
}
function cleanup(){if(dir)fs.rmSync(dir,{recursive:true,force:true});}
process.on('exit',cleanup);
if (require.main === module) {
 const mode = process.argv[2];
 const cf = JSON.parse(fs.readFileSync('.kogen/intents/'+slug+'/controls.json','utf8'));
 build({...cf.correct.files,...(mode === 'bad' ? cf.wrong[0].files : {})});
 const message = 'go build -o /x/driver ./features/steps/s__driver/main.go:\n# github.com/KogenAI/kogen-review/review\nreview/independent.go:771:6: undefined: required\n';
 try {
  judge(observe(message),'go build -o /x/driver ./features/steps/s__driver/main.go: | review/independent.go:771:6: undefined: required | # github.com/KogenAI/kogen-review/review','go build -o /x/driver ./features/steps/s__driver/main.go:');
  for (const [size,total] of [[1,20],[127,398],[130,400]]) {
   bounded(observe('command:\n'+Array.from({length:50},(_,i)=>String.fromCharCode(97+i%26).repeat(size)).join('\n')),3,total);
  }
  judge(observe('expected true'),'expected true','expected true');
  console.log('observer accepted');
 } catch(e) { if(e.code !== 'ERR_ASSERTION') throw e; console.log('observer rejected'); process.exitCode=1; }
} else {
 const {When,Then} = require('@cucumber/cucumber');
 When(P+'the failure message is:',function(doc){this.sunMessage=doc;this.sunObs=observe(doc);});
 When(P+'the failure has {int} following lines of {int} letters',function(n,len){this.sunMessage='command:\n'+Array.from({length:n},(_,i)=>String.fromCharCode(97+i%26).repeat(len)).join('\n');this.sunObs=observe(this.sunMessage);});
 Then(P+'the reported reason matches',function(doc){judge(this.sunObs,doc,this.sunMessage.split('\n')[0]);});
 Then(P+'the report has {int} extra lines and {int} characters',function(n,len){bounded(this.sunObs,n,len);});
}
