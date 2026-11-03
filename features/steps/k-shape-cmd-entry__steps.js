'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const http = require('node:http');
const {spawn, spawnSync} = require('node:child_process');
const slug = 'is-even';
const relative = '.kogen/intents/is-even/intent.md';
const feature = 'Feature: Is even\n  Scenario: Even numbers\n    Given is-even: the number 4\n    Then is-even: it is even\n  Scenario: Odd numbers\n    Given is-even: the number 3\n    Then is-even: it is not even\n';
const steps = "const {Given, Then} = require('@cucumber/cucumber');\nGiven('is-even: the number {int}', function(n) { this.n = n; });\nThen('is-even: it is even', function() { require('node:assert/strict').equal(this.n % 2, 0); });\nThen('is-even: it is not even', function() { require('node:assert/strict').equal(this.n % 2, 1); });\n";
const intent = '# Add an is-even check\n\nSlug: is-even\nModel: not applicable — a pure function without state\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\ncalc.is_even(n) returns True for even integers.\n\n## Decisions\n- Check even and odd integers. (source: request)\n\n## Source comparison\n- addressed: the request asks for an even check; both integer outcomes are covered in acceptance.\n\n## Tests\n- Even and odd integers are distinguished: acceptance scenarios (scenario)\n\n## Feasibility\n- none\n';
function observe(result, root, expectedCode, expectedContent) {
  assert.equal(result.code, expectedCode, result.stderr);
  const doc = JSON.parse(result.stdout);
  assert.equal(doc.exit_code, expectedCode);
  if (expectedContent === null) {
    assert.equal(doc.intent, null);
    assert.equal(typeof doc.error, 'string');
    assert.ok(doc.error.length > 0);
  } else {
    assert.equal(doc.error, null);
    assert.equal(doc.intent.path, path.join(root, relative));
    assert.equal(doc.intent.content, expectedContent);
    assert.equal(fs.readFileSync(doc.intent.path, 'utf8'), expectedContent);
  }
}
if (process.argv[2] === '--control') {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-entry-observer-'));
  try {
    fs.mkdirSync(path.dirname(path.join(root, relative)), {recursive:true});
    fs.writeFileSync(path.join(root, relative), intent);
    const doc = {exit_code:0, error:null, intent:{path:path.join(root, relative), content:intent}};
    if (process.argv[3] === 'bad') doc.intent.content = 'stale intent';
    observe({code:0, stdout:JSON.stringify(doc), stderr:''}, root, 0, intent);
    observe({code:3, stdout:JSON.stringify({exit_code:3,error:'environment failure',intent:null}),stderr:''}, root, 3, null);
    console.log('accepted');
  } catch (_) { console.log('rejected'); process.exitCode = 1; }
  finally { fs.rmSync(root, {recursive:true,force:true}); }
} else {
  const {When, Then} = require('@cucumber/cucumber');
  function write(root, name, content) { const p = path.join(root,name); fs.mkdirSync(path.dirname(p),{recursive:true}); fs.writeFileSync(p,content); }
  function run(binary, args, env, input) {
    return new Promise((resolve,reject) => {
      const c = spawn(binary,args,{env}); let stdout='',stderr='';
      c.stdout.on('data', b => stdout += b); c.stderr.on('data', b => stderr += b);
      c.on('error',reject); c.on('close',code=>resolve({code,stdout,stderr})); c.stdin.end(input || '');
    });
  }
  When('I run standalone Shape on a local scoring checkout using {string}', {timeout:120000}, async function(source) {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(),'shape-entry-'));
    let server;
    try {
      const binary = path.join(tmp,'kogen-shape');
      const built = spawnSync('go',['build','-o',binary,'./cmd/kogen-shape'],{encoding:'utf8'});
      assert.equal(built.status,0, `go build ./cmd/kogen-shape: ${built.stdout}${built.stderr}`);
      const root = path.join(tmp,'checkout'); fs.mkdirSync(root);
      const home = path.join(tmp,'home');
      write(home,'.kogen/credentials/chatgpt.json',JSON.stringify({access_token:'access',refresh_token:'refresh',client_id:'oaiapp_fixture',expires_at:Math.floor(Date.now()/1000)+86400}));
      write(root,'README.md','# Calc\nA tiny calculator.\n');
      write(root,'.gitignore','.kogen/local/\n');
      write(root,'.kogen/project.yaml', `version: 1\ncommands:\n  checks: [{name: unit, run: 'true'}]\n  acceptance:\n    run: '"$KOGEN_NODE" "$KOGEN_CUCUMBER/@cucumber/cucumber/bin/cucumber.js" --strict --require "features/steps/*.js" --format "message:$KOGEN_ACCEPTANCE_REPORT" "$KOGEN_ACCEPTANCE_FEATURES"'\n    steps_dir: features/steps\n`);
      const env = {...process.env,HOME:home,GIT_CONFIG_GLOBAL:'/dev/null',GIT_CONFIG_NOSYSTEM:'1',GIT_AUTHOR_NAME:'Fixture',GIT_AUTHOR_EMAIL:'fixture@example.invalid',GIT_COMMITTER_NAME:'Fixture',GIT_COMMITTER_EMAIL:'fixture@example.invalid'};
      for (const key of Object.keys(env)) if (key.startsWith('GIT_') && !['GIT_CONFIG_GLOBAL','GIT_CONFIG_NOSYSTEM','GIT_AUTHOR_NAME','GIT_AUTHOR_EMAIL','GIT_COMMITTER_NAME','GIT_COMMITTER_EMAIL'].includes(key)) delete env[key];
      for (const args of [['init','-q'],['add','.'],['-c','commit.gpgsign=false','commit','-qm','fixture']]) {
        const r=spawnSync('git',args,{cwd:root,env,encoding:'utf8'}); assert.equal(r.status,0,r.stderr);
      }
      let calls=0;
      server=http.createServer((req,res)=>{
        let body=''; req.on('data',b=>body+=b); req.on('end',()=>{
          try {
            const request=JSON.parse(body); const names=(request.tools||[]).map(t=>t.name || (t.function||{}).name);
            let name='submit', args;
            if(names.includes('submit_contract_review')) {name='submit_contract_review';args={findings:[],environment:[]};}
            else if(names.includes('submit_contract_confirmation')) {name='submit_contract_confirmation';args={verdicts:[]};}
            else { calls++; args={files:[{path:relative,content:intent},{path:'.kogen/intents/is-even/acceptance.feature',content:feature},{path:'features/steps/is-even__steps.js',content:steps}]}; }
            res.writeHead(200,{'Content-Type':'text/event-stream'});
            const frame=x=>res.write('data: '+JSON.stringify(x)+'\n\n');
            if(name==='submit') frame({type:'response.output_item.done',output_index:0,item:{type:'function_call',call_id:'read-'+calls,name:'read_file',arguments:JSON.stringify({path:'README.md'})}});
            frame({type:'response.output_item.done',output_index:name==='submit'?1:0,item:{type:'function_call',call_id:'submit-'+calls+'-'+name,name,arguments:JSON.stringify(args)}});
            frame({type:'response.completed',response:{id:'resp-'+calls,model:'gpt-6.1-sol',usage:{input_tokens:100,input_tokens_details:{cached_tokens:0},output_tokens:30,output_tokens_details:{reasoning_tokens:10}}}}); res.end();
          } catch(e) { res.writeHead(500);res.end(String(e)); }
        });
      });
      await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
      env.KOGEN_PROVIDER_URL='http://127.0.0.1:'+server.address().port;
      const request='Add an even check for integers; cover even and odd integers.';
      const requestPath=path.join(tmp,'request.txt');fs.writeFileSync(requestPath,request);
      const args=['--project',root,'--slug',slug,'--request',source==='stdin'?'-':requestPath];
      const result=await run(binary,args,env,source==='stdin'?request:'');
      this.shapeEntry={result,root,expected:intent,calls};
      observe(result,root,0,intent);
      assert.ok(calls>0,'standalone entry must call the shaping provider');
      const stale=await run(binary,args,env,source==='stdin'?request:'');
      observe(stale,root,5,null);
      this.shapeEntry.checked=true;
    } finally { if(server) await new Promise(resolve=>server.close(resolve)); fs.rmSync(tmp,{recursive:true,force:true}); }
  });
  Then('standalone Shape reports exit {int}, the exact written Intent, and no stale Intent on exit {int}', function(success,failure) {
    assert.equal(success,0);assert.equal(failure,5);assert.equal(this.shapeEntry.checked,true);
  });
}
