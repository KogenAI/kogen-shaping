const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const cp=require('node:child_process');
const assert=require('node:assert/strict');
const names={alternating:'TestBoundsAlternating',reset:'TestBoundsReset',consecutive:'TestBoundsConsecutive',cap:'TestBoundsCap'};
function observe(output,name,status){
 const rows=output.split('\n').filter(Boolean).map(x=>JSON.parse(x));
 assert.equal(status,0,`${name}: go test ./shape failed: ${output}`);
 assert.equal(rows.filter(x=>x.Test===name&&x.Action==='pass').length,1,`${name}: go test ./shape absent, skipped or ran zero times: ${output}`);
 assert.equal(rows.filter(x=>x.Test===name&&x.Action==='run').length,1,`${name}: go test ./shape absent, skipped or ran zero times: ${output}`);
}
if(require.main===module){
if(process.argv[2]==='observe'){
 try {observe(fs.readFileSync(process.argv[3],'utf8'),'TestBoundsConsecutive',Number(process.argv[4])); console.log('observer accepted');}catch(e){console.log('observer rejected');process.exitCode=1;}
}else{
 const name=names[process.argv[2]];assert.ok(name,'unknown case');
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'shape-bounds-'));
 try{
 cp.execFileSync('git',['clone','--quiet','--shared',process.cwd(),dir]);
 // Copy current candidate, including uncommitted implementation changes.
 for(const p of cp.execFileSync('git',['ls-files','--cached','--others','--exclude-standard'],{encoding:'utf8'}).trim().split('\n')){
 if(!p||p.startsWith('.kogen/local/'))continue;
 if(fs.existsSync(p)&&fs.statSync(p).isFile()){const dest=path.join(dir,p);fs.mkdirSync(path.dirname(dest),{recursive:true});fs.copyFileSync(p,dest);}
 }
 fs.writeFileSync(path.join(dir,'shape','bounds_acceptance_test.go'),fs.readFileSync(path.join(__dirname,'cases.go'),'utf8').replace('//go:build ignore\n\n',''));
 const r=cp.spawnSync('go',['test','-json','-count=1','./shape','-run',`^${name}$`],{cwd:dir,encoding:'utf8',timeout:160000,maxBuffer:16*1024*1024});
 observe(r.stdout||'',name,r.status);
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
}

}
