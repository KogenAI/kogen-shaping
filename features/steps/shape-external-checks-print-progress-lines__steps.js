'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const slug = 'shape-external-checks-print-progress-lines';
const root = process.cwd();
const driver = `features/steps/${slug}__driver/observe_test.go.txt`;
const controls = `.kogen/intents/${slug}/controls.json`;
const testName = 'TestExternalProgressAcceptance';
let binary;
function env(extra={}) {
 const e = {...process.env,...extra};
 for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
 return e;
}
function exec(cmd,args,options={}) {
 const r=spawnSync(cmd,args,{encoding:'utf8',env:env(),timeout:90000,...options});
 assert.equal(r.status,0,`${testName}: ${cmd} ${args.join(' ')} outcome=${r.status} ${r.error||''}\n${r.stdout}${r.stderr}`);
 return r.stdout;
}
function build(variant) {
 if (!variant && binary) return binary;
 const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'check-progress-'));
 process.on('exit',()=>fs.rmSync(tmp,{recursive:true,force:true}));
 const source=path.join(tmp,'observe_test.go'); fs.writeFileSync(source,fs.readFileSync(driver));
 const Replace = {[path.join(root,'shape/external_progress_acceptance_test.go')]:source};
 if (variant) {
  const c=JSON.parse(fs.readFileSync(controls,'utf8'));
  for (const p of ['shape/decision.go','shape/literal.go']) {
   fs.mkdirSync(path.join(tmp,path.dirname(p)),{recursive:true});
   fs.writeFileSync(path.join(tmp,p),fs.readFileSync(p));
  }
  for (const overlay of [c.correct, ...(variant==='bad'?[c.wrong[0]]:[])]) {
   for (const [p,v] of Object.entries(overlay.files)) {
    if (!['shape/decision.go','shape/literal.go'].includes(p)) continue;
    const patch=path.join(tmp,'change.patch'); fs.writeFileSync(patch,v.patch);
    const check=spawnSync('git',['apply','--check',patch],{cwd:tmp,encoding:'utf8',env:env()});
    if(check.status===0) exec('git',['apply',patch],{cwd:tmp});
    else exec('git',['apply','--reverse','--check',patch],{cwd:tmp});
   }
  }
  for (const p of ['shape/decision.go','shape/literal.go']) Replace[path.join(root,p)]=path.join(tmp,p);
 }
 const overlay=path.join(tmp,'overlay.json'); fs.writeFileSync(overlay,JSON.stringify({Replace}));
 const bin=path.join(tmp,'shape.test');
 exec('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape']);
 const built={tmp,bin}; if (!variant) binary=built; return built;
}
function run(mode,variant) {
 const {tmp,bin}=build(variant), report=path.join(tmp,`${mode}.json`);
 fs.rmSync(report,{force:true});
 exec(bin,['-test.run',`^${testName}$`,'-test.count=1','-test.timeout=35s'],{cwd:path.join(root,'shape'),env:env({CHECK_PROGRESS_MODE:mode,CHECK_PROGRESS_REPORT:report}),timeout:40000});
 assert.ok(fs.existsSync(report),`${testName}: built ./shape observer absent, skipped or ran zero times; no report`);
 return JSON.parse(fs.readFileSync(report,'utf8'));
}
function lines(text,kind) { return text.split('\n').filter(s=>s.includes(`checking the ${kind==='decision'?'stated decisions':'required literals'}`)||s.includes(`${kind} check finished`)||s.includes(`${kind} check response`)); }
function observeExternal(r,expected) {
 assert.equal(r.nativeRequests,0,'external Shape must not call native provider');
 for (const kind of ['decision','literal']) {
  const e=expected[kind], n=r[`${kind}Rounds`];
  assert.equal(n,expected.rounds,`${kind} executor rounds`);
  assert.equal(r[`${kind}Events`],n,`${kind} events unchanged`);
  const all=lines(r.output,kind);
  assert.equal(all.length,n*2,`${kind}: one start and one finish per round; got ${JSON.stringify(all)}`);
  for(let i=0;i<n;i++) {
   assert.match(all[2*i],new RegExp(`^${e.start} \\([0-9]+ s\\)$`));
   assert.match(all[2*i+1],new RegExp(`^${e.finish} \\([0-9]+ s\\)$`));
  }
  const snaps=r.snapshots.filter(s=>s.Kind===kind);
  assert.equal(snaps.length,n,`${kind} callbacks`);
  snaps.forEach((s,i)=> {
   assert.equal(lines(s.Before,kind).length,2*i+1,`${kind} round ${i+1}: start must precede executor round`);
   assert.equal(lines(s.Returning,kind).length,2*i+1,`${kind} round ${i+1}: finish must follow executor round`);
  });
 }
}
function observeNative(r,expected) {
 const baseline=JSON.parse(fs.readFileSync(`.kogen/intents/${slug}/native-observation.json`,'utf8'));
 const normalize=s=>s.replace(/[0-9]+/g,'N');
 for(const kind of ['decision','literal']) {
  const all=lines(r.output,kind);
  assert.equal(all.length,2,`${kind}: native lines unchanged`);
  assert.deepEqual(all.map(normalize),lines(baseline.output,kind).map(normalize),`${kind}: native wording and usage fields unchanged from observed baseline`);
  assert.equal(all[0].replace(/ \([0-9]+ s\)$/,''),expected[kind].start);
  assert.match(all[1],new RegExp(`^${expected[kind].response} .+ wall_ms=[0-9]+ \\([0-9]+ s\\)$`));
  assert.equal(r[`${kind}Events`],1,`${kind} native event count`);
 }
}
if(require.main===module) {
 const mode=process.argv[2];
 if(mode==='record-native') process.stdout.write(JSON.stringify(run('native'))+'\n');
 else if(mode==='qualify') {
  const variant=process.argv[3];
  const expected=JSON.parse(fs.readFileSync(`.kogen/intents/${slug}/observer-expectations.json`,'utf8'));
  const external=run('external',variant), native=run('native',variant);
  try { observeExternal(external,expected); observeNative(native,expected); console.log('observer accepted'); }
  catch(e) { console.log('observer rejected'); process.exitCode=1; }
 } else throw new Error('unknown command');
} else {
 const {Given,When,Then,setDefaultTimeout}=require('@cucumber/cucumber'); setDefaultTimeout(115000);
 Given(`${slug}: the scripted check expectations`,function(text){this.checkExpected=JSON.parse(text);});
 When(`${slug}: an external Shape completes two checked submissions`,function(){this.externalCheckReport=run('external');});
 Then(`${slug}: each external check announces its round before and after execution`,function(){observeExternal(this.externalCheckReport,this.checkExpected);});
 When(`${slug}: a native Shape completes a checked submission`,function(){this.nativeCheckReport=run('native');});
 Then(`${slug}: native check progress retains its existing format`,function(){observeNative(this.nativeCheckReport,this.checkExpected);});
}
