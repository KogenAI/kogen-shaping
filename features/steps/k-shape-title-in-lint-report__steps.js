'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const fixture = require('./k-shape-title-in-lint-report__fixture.js');
const names = ['TestSubmissionRetryOne/title_fallback','TestShapeShortensATooLongTitleOnlyOnTheLastAttempt/last_attempt','TestKTitleRetained'];
function observe(r, required) {
 const text = (r.stdout || '') + (r.stderr || '');
 const events = text.split('\n').flatMap(line => { try { return [JSON.parse(line)]; } catch { return []; } });
 for (const name of required) assert.ok(events.filter(e => e.Test === name && e.Action === 'pass').length === 1 && !events.some(e => e.Test === name && ['fail','skip'].includes(e.Action)), `required test ${name}: absent, skipped, failed or not run exactly once; command ${r.command}; exit ${r.status}\n${text}`);
 assert.equal(r.status,0,`required tests ${required.join(', ')}; command ${r.command}; exit ${r.status}\n${text}`);
}
function run(args,env) {
 const r = spawnSync('go',args,{encoding:'utf8',env,timeout:180000,maxBuffer:16000000});
 return {...r,command:'go '+args.join(' ')};
}
function execute(kind,control) {
 const root = process.cwd();
 const dir = fs.mkdtempSync(path.join(os.tmpdir(),'title-report-'));
 const env = {...process.env,GOWORK:'off'};
 delete env.K_TITLE_CONTROL;
 for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
 try {
  const file = path.join(dir,'fixture_test.go'); fs.writeFileSync(file,fixture);
  const overlay = path.join(dir,'overlay.json');
  fs.writeFileSync(overlay,JSON.stringify({Replace:{[path.join(root,'shape/k_title_acceptance_test.go')]:file}}));
  if (control) env.K_TITLE_CONTROL = control;
  if (kind !== 'pinned') {
   assert.ok(['main','main-title-first','main-title-last'].includes(kind));
   const d = run(['mod','download','-json','github.com/KogenAI/kogen-coordinator@v0.0.0-20261103214825-03d7341feab7'],env);
   assert.equal(d.status,0,`${d.command}\n${d.stdout}${d.stderr}`);
   let coordinator = JSON.parse(d.stdout).Dir;
   if (kind !== 'main') {
    const copy = path.join(dir,'coordinator'); fs.cpSync(coordinator,copy,{recursive:true}); coordinator = copy;
    const target = path.join(copy,'project/intent.go');
    const source = fs.readFileSync(target,'utf8');
    const marker = 'if problems := lintIntent(intent, scenarios, true); len(problems) > 0 {';
    assert.equal(source.split(marker).length,2,'unique parsed lint block');
    const body = `\nvar titles, others []string\nfor _, p := range problems { if strings.HasPrefix(p, "title:") { titles = append(titles, "title: length exceeds allowed maximum") } else { others = append(others,p) } }\nproblems = append(${kind === 'main-title-first' ? 'titles, others' : 'others, titles'}...)\n`;
    fs.chmodSync(target,0o644); fs.writeFileSync(target,source.replace(marker,marker+body));
   }
   const version = /^go (\S+)$/m.exec(fs.readFileSync('go.mod','utf8'))[1];
   env.GOWORK = path.join(dir,'go.work');
   fs.writeFileSync(env.GOWORK,`go ${version}\nuse ${JSON.stringify(root)}\nreplace github.com/KogenAI/kogen-coordinator => ${JSON.stringify(coordinator)}\n`);
  }
  // Run the unchanged reproductions separately from the mixed-error fixture, so obsolete
  // assertions in another_problem_remains do not define this acceptance's desired outcome.
  const filter = control ? '^TestKTitleRetained$' : '^(TestSubmissionRetryOne|TestShapeShortensATooLongTitleOnlyOnTheLastAttempt|TestKTitleRetained)$/(title_fallback|last_attempt)$';
  return run(['test','-overlay',overlay,'-json','-count=1','./shape','-run',filter],env);
 } finally { fs.rmSync(dir,{recursive:true,force:true}); }
}
if (require.main === module) {
 const bad = process.argv[2] === 'bad';
 const r = execute('pinned',bad ? 'bad' : 'good');
 if (bad) {
  assert.equal(r.status,1,`known-bad retained title must fail: ${r.command}\n${r.stdout}${r.stderr}`);
  assert.ok(r.stdout.includes('retained shortened title'),`required test TestKTitleRetained must reach retained title assertion: ${r.command}\n${r.stdout}${r.stderr}`);
  console.log('rejected'); process.exitCode = 1;
 } else { observe(r,['TestKTitleRetained']); console.log('accepted'); }
} else {
 const {When,Then} = require('@cucumber/cucumber');
 When('k-shape-title-in-lint-report: the title regressions run with coordinator {string}',{timeout:240000},function(kind) { this.titleResult = execute(kind); });
 Then('k-shape-title-in-lint-report: all title regressions pass exactly once',function() { observe(this.titleResult,names); });
}
