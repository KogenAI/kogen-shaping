'use strict';
const fs=require('node:fs'),os=require('node:os'),path=require('node:path'),cp=require('node:child_process');
const root=process.cwd(),dir=fs.mkdtempSync(path.join(os.tmpdir(),'repeat-qualify-'));
const base='features/steps/shape-kept-repeat-reuses-unpassable-control';
const slug='shape-kept-repeat-reuses-unpassable-control';
try {
 cp.execFileSync('git',['clone','--quiet','--shared',root,dir]);
 for(const p of [base+'__steps.js',base+'__driver/observe_test.go']) {
  fs.mkdirSync(path.dirname(path.join(dir,p)),{recursive:true});fs.copyFileSync(p,path.join(dir,p));
 }
 const controls=JSON.parse(fs.readFileSync('.kogen/intents/'+slug+'/controls.json','utf8'));
 for(const [p,v] of Object.entries(controls.correct.files)) cp.execFileSync('git',['apply','-'],{cwd:dir,input:v.patch});
 process.env.KOGEN_REPEAT_OBSERVER_LIBRARY='1';
 const {observe}=require(path.join(root,base+'__steps.js'));
 const feature=fs.readFileSync('.kogen/intents/'+slug+'/acceptance.feature','utf8');
 const values=[...feature.matchAll(/"""\s*([\s\S]*?)\s*"""/g)].map(m=>JSON.parse(m[1]));
 const expectations=Object.fromEntries(['repeat','package','tree','skipped','refusal','error'].map((m,i)=>[m,values[i]]));
 const bad=process.argv[2]==='bad';
 let rejected=0;
 for(const [mode,e] of Object.entries(expectations)) {
  const r=cp.spawnSync(process.execPath,[base+'__steps.js',mode],{cwd:dir,encoding:'utf8',timeout:90000});
  if(r.status!==0) throw Error(mode+': driver failed: '+r.stdout+r.stderr);
  const o=JSON.parse(r.stdout);
  observe(o,e);
  if(bad) {
   // Each independent corruption is checked after the complete real known-good observation.
   const variants=[{...o,controls:o.controls+1},{...o,readiness:o.readiness+1},{...o,checks:[]},{...o,second_error:'lost verdict'}];
   if(e.repeats) variants.push({...o,repeats:[]},{...o,progress:''},{...o,refusals:[]});
   if(e.skipped) variants.push({...o,checks:o.checks.map(c=>({...c,skipped:undefined}))});
   if(e.refusal) variants.push({...o,first_error:'changed refusal',second_error:'changed refusal'});
   for(const v of variants) {
    let failed=false;try{observe(v,e);}catch(_){failed=true;}
    if(!failed) throw Error(mode+': observer accepted corruption');
    rejected++;
   }
  }
 }
 console.log(bad?'observer rejected corruptions':'observer accepted real controls');
 process.exitCode=bad&&rejected>0?1:0;
} finally { fs.rmSync(dir,{recursive:true,force:true}); }
