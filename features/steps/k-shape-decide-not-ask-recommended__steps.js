'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestKShapeDecideNotAskRecommendedAcceptance';
function compare(actual, expected) {
  for (const [key, value] of Object.entries(expected)) {
    assert.deepEqual(actual[key], value, `${TEST}: ${key}; observation=${JSON.stringify(actual)}`);
  }
}
function control(kind) {
  const good = {exit:0, asked:0, assumption:true, pending:false, ready:true, failed:true, carried:false, second_started:true, accepted:true};
  if (kind === 'good') { compare(good, good); console.log('observer accepted good'); return; }
  let rejected = 0;
  for (const [key, value] of Object.entries(good)) {
    const bad = {...good, [key]: typeof value === 'boolean' ? !value : value+1};
    try { compare(bad, good); } catch (_) { rejected++; }
  }
  if (rejected !== Object.keys(good).length) throw new Error('observer failed to reject a bad field');
  console.log('observer rejected bad');
  process.exitCode = 1;
}
if (require.main === module) {
  control(process.argv[2]);
} else {
  const {When, Then, setDefaultTimeout} = require('@cucumber/cucumber');
  setDefaultTimeout(300000);
  let built;
  function environment(extra={}) {
    const env = {...process.env, ...extra};
    for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
    return env;
  }
  function run(name) {
    if (!built) {
      const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kdn-'));
      process.on('exit', () => fs.rmSync(dir, {recursive:true, force:true}));
      const source = path.join(dir, 'acceptance_test.go');
      fs.writeFileSync(source, require('./k-shape-decide-not-ask-recommended__harness.js'));
      const overlay = path.join(dir, 'overlay.json');
      fs.writeFileSync(overlay, JSON.stringify({Replace:{[path.resolve('shape/kdn_acceptance_test.go')]:source}}));
      const bin = path.join(dir, 'shape.test');
      const r = spawnSync('go', ['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {encoding:'utf8',env:environment(),timeout:280000});
      assert.equal(r.status,0, `${TEST}: go test -c -overlay ./shape: exit=${r.status} ${r.error || ''}\n${r.stdout}${r.stderr}`);
      built = {dir,bin};
    }
    const out = path.join(built.dir,name+'.json');
    const r = spawnSync(built.bin,['-test.run','^'+TEST+'$','-test.count=1','-test.v','-test.timeout=180s'], {cwd:path.resolve('shape'),env:environment({KDN_CASE:name,KDN_OUT:out}),encoding:'utf8',timeout:200000});
    const text = (r.stdout || '')+(r.stderr || '');
    assert.ok(r.status === 0 && text.includes('--- PASS: '+TEST+' ') && fs.existsSync(out), `${TEST}: shape.test -test.run ^${TEST}$ case=${name}: exit=${r.status} ${r.error || ''}\n${text}`);
    return JSON.parse(fs.readFileSync(out,'utf8'));
  }
  When('k-shape-decide-not-ask-recommended: I exercise {string}', function(name) { this.kdn = run(name); });
  Then('k-shape-decide-not-ask-recommended: the observation is', function(table) {
    const expected = {};
    for (const [key,value] of table.raw()) expected[key] = JSON.parse(value);
    compare(this.kdn,expected);
  });
  When('k-shape-decide-not-ask-recommended: I run the complete Go suite', function() {
    const r = spawnSync('go',['test','./...'],{encoding:'utf8',env:environment(),timeout:280000,maxBuffer:32*1024*1024});
    this.kdnSuite = r;
  });
  Then('k-shape-decide-not-ask-recommended: the complete Go suite succeeds', function() {
    const r = this.kdnSuite;
    assert.equal(r.status,0,`go test ./...: exit=${r.status} ${r.error || ''}\n${r.stdout}${r.stderr}`);
  });
}
