const fs=require('node:fs'),cp=require('node:child_process'),path=require('node:path'),os=require('node:os');
if(require.main===module){
const dir=fs.mkdtempSync(path.join(os.tmpdir(),'bounds-observer-'));
try{
 cp.execFileSync('git',['clone','--quiet','--shared',process.cwd(),dir]);
 const p='features/steps/shape-bounds-tool-free-answers__driver';
 fs.mkdirSync(path.join(dir,p),{recursive:true});
 for(const f of ['run.js','cases.go'])fs.copyFileSync(path.join(p,f),path.join(dir,p,f));
 const c=JSON.parse(fs.readFileSync('.kogen/intents/shape-bounds-tool-free-answers/controls.json'));
 for(const [p,s] of Object.entries(c.correct.files))fs.writeFileSync(path.join(dir,p),s);
 if(process.argv[2]==='bad'){
 const p=path.join(dir,'shape/toolfree.go');fs.writeFileSync(p,fs.readFileSync(p,'utf8').replace('if answer!="" { o.detail += "; last tool-free answer: " + answer }',''));
 }
 cp.execFileSync('git',['add','.'],{cwd:dir});cp.execFileSync('git',['-c','user.name=fixture','-c','user.email=fixture@example.invalid','commit','-qm','fixture'],{cwd:dir});
 const r=cp.spawnSync(process.execPath,[path.join(dir,p,'run.js'),'cap'],{cwd:dir,encoding:'utf8',timeout:170000});
 if(r.status===0){console.log('observer accepted');}else{console.log('observer rejected');process.exitCode=1;}
}finally{fs.rmSync(dir,{recursive:true,force:true});}
}
