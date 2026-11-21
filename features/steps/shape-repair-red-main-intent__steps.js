'use strict';
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
const {When,Then}=require('@cucumber/cucumber');
const TEST='TestRepairRedMainJourney';
let built;
const env=(extra={})=>{const e={...process.env,...extra};for(const k of Object.keys(e))if(k.startsWith('KOGEN_ACCEPTANCE_'))delete e[k];return e;};
function judge(o,mode){
 const msg=JSON.stringify(o);
 if(mode==='timing'){
  const ev=JSON.parse(o.event);
  assert.ok(typeof ev['own check seconds']==='number'&&ev['own check seconds']>=120,'own check seconds missing or too small: '+msg);
  assert.deepEqual(ev['acceptance timeout'],['slow'],'acceptance timeout: '+msg);
  return;
 }
 if(mode==='target'){
  assert.equal(o.published,true,'red main with a declared repair was refused: '+msg);
  assert.ok(o.progress.split('\n').some(l=>l.includes('Greeting is polite')&&l.includes('red before')&&l.includes('green after')),'target not recorded as red before, green after: '+msg);
  return;
 }
 assert.equal(o.published,false,msg);
 if(mode==='keeps'){
  assert.ok(o.feedback.includes('Other stays')&&o.feedback.includes('red on the unchanged tree'),msg);
  assert.ok(!/Greeting is polite[^\n]*\(kept\)/.test(o.feedback),'the repair target was treated as kept: '+msg);
 }
 if(mode==='green') assert.ok(o.feedback.includes('Greeting is polite')&&/not failing/.test(o.feedback),msg);
 if(mode==='norepair') assert.ok(o.feedback.includes('Greeting is polite')&&o.feedback.includes('red on the unchanged tree'),msg);
}
When('shape-repair-red-main-intent: Shape processes {string}',function(mode){
 if(!built){
  const dir=fs.mkdtempSync(path.join(os.tmpdir(),'repair-red-'));
  process.on('exit',()=>fs.rmSync(dir,{recursive:true,force:true}));
  const src=path.join(dir,'journey_test.go');fs.writeFileSync(src,require('./shape-repair-red-main-intent__harness.js'));
  const overlay=path.join(dir,'overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace:{'shape/shape_repair_red_main_journey_test.go':src}}));
  const bin=path.join(dir,'shape.test');
  const r=spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'],{env:env(),encoding:'utf8',timeout:200000});
  assert.equal(r.status,0,TEST+': go test -c -overlay ./shape exit='+r.status+'\n'+r.stdout+r.stderr);
  built={dir,bin};
 }
 const out=path.join(built.dir,mode+'.json');
 const args=['-test.run','^'+TEST+'$','-test.count=1','-test.timeout=110s'];
 const r=spawnSync(built.bin,args,{cwd:path.resolve('shape'),env:env({KG_MODE:mode,KG_OUT:out}),encoding:'utf8',timeout:115000});
 assert.ok(r.status===0&&fs.existsSync(out),TEST+': shape.test '+args.join(' ')+' exit='+r.status+'; test absent, skipped or failed if no observation\n'+r.stdout+r.stderr);
 this.obs={mode,o:JSON.parse(fs.readFileSync(out,'utf8'))};
});
Then('shape-repair-red-main-intent: the repair policy holds',function(){judge(this.obs.o,this.obs.mode);});
