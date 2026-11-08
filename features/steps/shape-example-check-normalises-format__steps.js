'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

function observe(o, verdict, refused) {
  assert.ok(o.calls > 0, `Shape never requested the example check: ${JSON.stringify(o)}`);
  assert.equal(o.verdict, verdict, JSON.stringify(o));
  assert.equal(o.refused, refused, JSON.stringify(o));
}
module.exports = {observe};

if (require.main === module) {
  const bad = process.argv[2] === 'bad';
  try {
    for (const [verdict, refused] of [['follows',false],['contradicts',true],['unsupported',false]]) {
      observe({calls:1,verdict,refused},verdict,refused);
    }
    for (const o of [{calls:0,verdict:'follows',refused:false}, {calls:1,verdict:'contradicts',refused:false}, {calls:1,verdict:'follows',refused:true}]) {
      assert.throws(() => observe(o,'follows',false));
    }
    if (bad) observe({calls:1,verdict:'contradicts',refused:true},'follows',false);
    console.log('observer accepted');
  } catch (_) { console.log('observer rejected'); process.exitCode = 1; }
} else {
  const {When,Then} = require('@cucumber/cucumber');
  let built;
  function clean(extra = {}) {
    const env = {...process.env,...extra};
    for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
    return env;
  }
  function build() {
    if (built) return built;
    const dir = fs.mkdtempSync(path.join(os.tmpdir(),'shape-example-format-'));
    process.on('exit', () => fs.rmSync(dir,{recursive:true,force:true}));
    const source = path.join(dir,'example_format_test.go');
    fs.writeFileSync(source,require('./shape-example-check-normalises-format__harness.js'));
    const overlay = path.join(dir,'overlay.json');
    fs.writeFileSync(overlay,JSON.stringify({Replace:{[path.resolve('shape/shape_example_format_acceptance_test.go')]:source}}));
    const bin = path.join(dir,'shape.test');
    const r = spawnSync('go',['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'],{env:clean(),encoding:'utf8',timeout:280000});
    assert.equal(r.status,0,`compiling TestShapeExampleFormatAcceptance via go test -c -overlay ./shape: ${r.error || ''}\n${r.stdout}${r.stderr}`);
    return built = {bin,dir};
  }
  // Preserve JSON member quoting verbatim rather than treating it as Cucumber string syntax.
  When(/^shape-example-check-normalises-format: Shape checks expected (.*) against derived (.*)$/, function(expected,derived) {
    const b = build();
    const out = path.join(b.dir,'observation.json');
    fs.rmSync(out,{force:true});
    const r = spawnSync(b.bin,['-test.run','^TestShapeExampleFormatAcceptance$','-test.count=1','-test.timeout=180s'],{cwd:path.resolve('shape'),env:clean({SEF_EXPECTED:expected,SEF_DERIVED:derived,SEF_OUT:out}),encoding:'utf8',timeout:200000});
    assert.equal(r.status,0,`TestShapeExampleFormatAcceptance: shape.test -test.run ^TestShapeExampleFormatAcceptance$ exit=${r.status} ${r.error || ''}\n${r.stdout}${r.stderr}`);
    this.exampleObservation = JSON.parse(fs.readFileSync(out,'utf8'));
  });
  Then(/^shape-example-check-normalises-format: the verdict is (follows|contradicts|unsupported) and example refusal is (true|false)$/,function(verdict,refused) {
    observe(this.exampleObservation,verdict,refused === 'true');
  });
  When('shape-example-check-normalises-format: the Go suite runs',function() {
    this.goSuite = spawnSync('go',['test','./...'],{env:clean(),encoding:'utf8',timeout:280000,maxBuffer:32*1024*1024});
  });
  Then('shape-example-check-normalises-format: the Go suite passes',function() {
    assert.equal(this.goSuite.status,0,`go test ./...: ${this.goSuite.error || ''}\n${this.goSuite.stdout}${this.goSuite.stderr}`);
  });
}
