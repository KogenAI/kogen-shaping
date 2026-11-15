'use strict';
const fs=require('node:fs'),os=require('node:os'),path=require('node:path'),cp=require('node:child_process');
const slug='shape-repeated-decision-statement-says-so',base='features/steps/'+slug;
const root=process.cwd(),dir=fs.mkdtempSync(path.join(os.tmpdir(),'repeated-qualify-'));
try {
 cp.execFileSync('git',['clone','--quiet','--shared',root,dir]);
 for(const p of [base+'__steps.js',base+'__driver/observe_test.go']) { fs.mkdirSync(path.dirname(path.join(dir,p)),{recursive:true}); fs.copyFileSync(p,path.join(dir,p)); }
 const controls=JSON.parse(fs.readFileSync('.kogen/intents/'+slug+'/controls.json','utf8'));
 for(const value of Object.values(controls.correct.files)) cp.execFileSync('git',['apply','-'],{cwd:dir,input:value.patch});
 process.chdir(dir); process.env.REPEATED_OBSERVER_LIBRARY='1';
 const {run,observe}=require(path.join(dir,base+'__steps.js'));
 const feature=fs.readFileSync(path.join(root,'.kogen/intents/'+slug+'/acceptance.feature'),'utf8');
 const blocks=[...feature.matchAll(/"""\s*([\s\S]*?)\s*"""/g)].map(m=>JSON.parse(m[1]));
 const bad=process.argv[2]==='bad'; let rejected=0;
 for(let i=0;i<blocks.length;i+=2) {
  const c=blocks[i],expected=blocks[i+1],o=run(c); observe(o,c,expected);
  if(bad) {
   const variants=[];
   const copy=()=>JSON.parse(JSON.stringify(o));
   let v=copy(); v.requests.pop(); variants.push(v);
   v=copy(); v.events[0].items[0].verdict='unsupported'; variants.push(v);
   v=copy(); v.events[0].items[0].request_quotes=[]; variants.push(v);
   for(let j=0;j<c.rounds.length;j++) {
    if(c.rounds[j].verdict==='follows') { v=copy();v.reports[j].feedback='unexpected refusal';variants.push(v);continue; }
    v=copy();v.reports[j].feedback=v.reports[j].feedback.replace('Align the acceptance expectations','Change the acceptance expectations');variants.push(v);
    v=copy();v.reports[j].feedback=v.reports[j].feedback.replace('statement: ','statement changed: ');variants.push(v);
    v=copy();
    if(expected[j]) v.reports[j].feedback=v.reports[j].feedback.replace(expected[j],'');
    else v.reports[j].feedback+='\nnote: this statement is unchanged since submission 1 and was refused for the same reason; change the statement itself, not the text around it';
    variants.push(v);
   }
   for(const v of variants) { let failed=false;try{observe(v,c,expected);}catch(_){failed=true;}if(!failed)throw Error('observer accepted corruption');rejected++; }
  }
 }
 console.log(bad?'observer rejected corruptions':'observer accepted real correct implementation');
 process.exitCode=bad&&rejected>0?1:0;
} finally { process.chdir(root);fs.rmSync(dir,{recursive:true,force:true}); }
