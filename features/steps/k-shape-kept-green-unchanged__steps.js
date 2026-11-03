'use strict';
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
const TEST='TestKeptGreenJourney';
function judge(o,mode) {
 const msg=JSON.stringify(o);
 assert.equal(o.published, !['mixed'].includes(mode),msg);
 if(mode==='correct'||mode==='mixed') {
  for(const text of ['B pins A','wrong-A','observed-A']) assert.ok(o.feedback.includes(text),msg);
  assert.ok(o.feedback.split('\n').some(l=>['B pins A','wrong-A','observed-A'].every(s=>l.includes(s))&&/expected/i.test(l)),msg);
  assert.match(o.feedback,/(?:take|derive)[\s\S]*expected[\s\S]*(?:current|unchanged)[\s\S]*fixture/i,msg);
  if(mode==='mixed') assert.ok(!o.feedback.split('\n').some(l=>l.includes('C is new')&&/kept|contradict|refus/i.test(l)),msg);
 } else {
  assert.ok(!/Property check: [1-9]/.test(o.feedback),msg);
 }
 if(mode==='green') assert.deepEqual([o.requests,o.contracts],[1,1],msg);
 if(mode==='plain') assert.deepEqual([o.requests,o.contracts],[2,1],msg);
 if(mode==='flaky') {
  assert.ok(o.runs>=2&&o.runs<=3,msg);
  assert.match(o.progress,/repeat|retry|recheck/i,msg);
  assert.match(o.progress,/2/ ,msg);
 }
}
function controls(bad) {
 const rows={
 correct:{published:true,feedback:'B pins A expected wrong-A observed observed-A on unchanged code; take expected from current code or own fixture',requests:2,contracts:3,runs:2,progress:''},
 mixed:{published:false,feedback:'B pins A expected wrong-A observed observed-A on unchanged code; take expected from current code or own fixture',requests:3,contracts:2,runs:2,progress:''},
 green:{published:true,feedback:'',requests:1,contracts:1,runs:1,progress:''},
 plain:{published:true,feedback:'',requests:2,contracts:1,runs:1,progress:''},
 missing:{published:true,feedback:'missing absent-fixture.txt',requests:2,contracts:1,runs:1,progress:''},
 flaky:{published:true,feedback:'',requests:2,contracts:1,runs:2,progress:'repeat cap 2'}
 };
 let rejected=0;
 for(const [mode,row] of Object.entries(rows)) {
  const variants=bad ? [ {...row,published:!row.published}, ...(mode==='correct'||mode==='mixed'?['B pins A','wrong-A','observed-A','fixture','expected'].map(s=>({...row,feedback:row.feedback.split(s).join('')})):[]), ...(mode==='green'||mode==='plain'?[{...row,requests:99}]:[]), ...(mode==='flaky'?[{...row,runs:1},{...row,runs:4},{...row,progress:''}]:[]) ] : [row];
  for(const v of variants) { try {judge(v,mode); if(bad) throw Error('accepted corrupt '+mode);} catch(e) {if(!bad||e.message.startsWith('accepted corrupt')) throw e;rejected++;} }
 }
 if(bad) {assert.ok(rejected>0); console.log('bad observations rejected');process.exitCode=1;} else console.log('good observations accepted');
}
if(require.main===module) controls(process.argv[2]==='bad');
else {
 const {When,Then}=require('@cucumber/cucumber');
 let built;
 const env=(extra={})=>{const e={...process.env,...extra};for(const k of Object.keys(e))if(k.startsWith('KOGEN_ACCEPTANCE_'))delete e[k];return e;};
 When('k-shape-kept-green-unchanged: Shape processes {string}',function(mode){
  if(!built){
   const dir=fs.mkdtempSync(path.join(os.tmpdir(),'kept-green-'));
   process.on('exit',()=>fs.rmSync(dir,{recursive:true,force:true}));
   const src=path.join(dir,'journey_test.go');fs.writeFileSync(src,require('./k-shape-kept-green-unchanged__harness.js'));
   const overlay=path.join(dir,'overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace:{'shape/k_kept_green_journey_test.go':src}}));
   const bin=path.join(dir,'shape.test');
   const r=spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'],{env:env(),encoding:'utf8',timeout:200000});
   assert.equal(r.status,0,`${TEST}: go test -c -overlay ./shape exit=${r.status}\n${r.stdout}${r.stderr}`);built={dir,bin};
  }
  const out=path.join(built.dir,mode+'.json');
  const args=['-test.run','^'+TEST+'$','-test.count=1','-test.timeout=180s'];
  const r=spawnSync(built.bin,args,{cwd:path.resolve('shape'),env:env({KG_MODE:mode,KG_OUT:out}),encoding:'utf8',timeout:200000});
  assert.ok(r.status===0&&fs.existsSync(out),`${TEST}: shape.test ${args.join(' ')} exit=${r.status}; test absent, skipped or failed if no observation\n${r.stdout}${r.stderr}`);
  this.kept={mode,o:JSON.parse(fs.readFileSync(out,'utf8'))};
 });
 Then('k-shape-kept-green-unchanged: the kept-output policy holds',function(){judge(this.kept.o,this.kept.mode);});
}
