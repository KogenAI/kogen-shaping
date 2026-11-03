'use strict';
const fs=require('node:fs'), os=require('node:os'), path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
function run(mode){
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'kss-'));
 try {
  const root=process.cwd(), overlay=path.join(dir,'overlay.json'), bin=path.join(dir,'shape.test');
  fs.writeFileSync(overlay,JSON.stringify({Replace:{[path.join(root,'shape/k_shape_selfcheck_stderr_acceptance_test.go')]:path.join(root,'features/steps/k_shape_selfcheck_stderr_test.go.txt')}}));
  const env={...process.env,KSS_MODE:mode};for(const k of Object.keys(env))if(k.startsWith('KOGEN_ACCEPTANCE_'))delete env[k];
  const b=spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'],{env,encoding:'utf8',timeout:120000});
  if(b.status!==0||b.error)return {ok:false,detail:`TestKShapeSelfcheckStderrAcceptance go test -c ./shape exit=${b.status}: ${b.error||''}\n${b.stdout}${b.stderr}`};
  const r=spawnSync(bin,['-test.run','^TestKShapeSelfcheckStderrAcceptance$','-test.count=1','-test.timeout=90s'],{cwd:path.join(root,'shape'),env,encoding:'utf8',timeout:100000});
  return {ok:r.status===0&&!r.error,detail:`TestKShapeSelfcheckStderrAcceptance ${bin} exit=${r.status}: ${r.error||''}\n${r.stdout}${r.stderr}`};
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
}
if(require.main===module){const r=run(process.argv[2]);console.log(r.ok?'accepted':'rejected');if(process.argv.includes('--diagnostic'))console.error(r.detail);process.exitCode=r.ok?0:1;}
else{const {When,Then}=require('@cucumber/cucumber');When('k-shape-selfcheck-stderr: observer refusal is exhausted',{timeout:240000},function(){this.kss=run('exhaust');});Then('k-shape-selfcheck-stderr: the redacted stderr tail is visible',function(){assert.ok(this.kss.ok,this.kss.detail);});}
