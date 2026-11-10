const {Then}=require('@cucumber/cucumber');
const fs=require('node:fs');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const assert=require('node:assert/strict');
Then('shaping-cucumber-special-char-hint: self-check explains literal expression characters', function(){
 const target=path.resolve('shape/zz_special_char_hint_acceptance_test.go');
 assert.ok(!fs.existsSync(target),'temporary test path must be free');
 fs.copyFileSync(path.join(__dirname,'shaping-cucumber-special-char-hint__driver/probe_test.go.txt'),target);
 try {
  const r=spawnSync('go',['test','./shape','-run','^TestSpecialCharHintAcceptance$','-count=1'],{encoding:'utf8',timeout:180000});
  assert.equal(r.status,0,'TestSpecialCharHintAcceptance via go test ./shape failed or did not complete:\n'+r.stdout+r.stderr);
 } finally {fs.unlinkSync(target);}
});
