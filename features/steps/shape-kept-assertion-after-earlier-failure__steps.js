'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const PREFIX = 'shape-kept-assertion-after-earlier-failure: ';
const TEST = 'TestMaskedKeptObservation';
let built;
function env(extra = {}) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}
function run(mode) {
  if (!built) {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'masked-kept-'));
    process.on('exit', () => fs.rmSync(dir, {recursive:true,force:true}));
    const src = path.join(dir, 'observe_test.go');
    fs.writeFileSync(src, fs.readFileSync('features/steps/shape-kept-assertion-after-earlier-failure__driver/observe_test.go', 'utf8').replace('//go:build ignore\n\n', ''));
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace:{[path.resolve('shape/masked_kept_acceptance_test.go')]:src}}));
    const bin = path.join(dir, 'shape.test');
    const r = spawnSync('go', ['test','-c','-vet=off','-overlay',overlay,'-o',bin,'./shape'], {env:env(),encoding:'utf8',timeout:120000});
    assert.equal(r.status, 0, `${TEST}: go test -c -overlay ./shape exit=${r.status} ${r.error || ''}\n${r.stdout}${r.stderr}`);
    built = {dir,bin};
  }
  const out = path.join(built.dir, mode + '.json');
  fs.rmSync(out, {force:true});
  const args = ['-test.run','^' + TEST + '$','-test.count=1','-test.timeout=45s'];
  const r = spawnSync(built.bin, args, {cwd:path.resolve('shape'),env:env({MASKED_MODE:mode,MASKED_OUT:out}),encoding:'utf8',timeout:55000});
  assert.ok(r.status === 0 && fs.existsSync(out), `${TEST}: shape.test ${args.join(' ')} exit=${r.status}; absent, skipped or failed without report: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}
function observe(o, expected) {
  const diagnostic = JSON.stringify(o);
  assert.equal(o.published, expected.published, 'publication: ' + diagnostic);
  assert.equal(o.unchanged, true, 'Shape must not rewrite or drop the candidate: ' + diagnostic);
  assert.equal(o.runs, expected.runs, 'bounded unchanged-tree observations: ' + diagnostic);
  for (const text of expected.feedback) assert.ok(o.feedback.includes(text), 'feedback lacks ' + text + ': ' + diagnostic);
  if (expected.published) assert.ok(!o.feedback.includes('Property check:'), 'new behaviour alone must not be refused: ' + diagnostic);
  else assert.ok(o.progress.includes('persistent kept assertion red'), 'repeat outcome: ' + diagnostic);
}
// These controls use an actual baseline refusal (an earlier assertion), not fabricated runner output.
function qualify(bad) {
  const o = run('assertion');
  const expected = {published:false,runs:2,feedback:['Mixed document','line 4','never runs on the unchanged tree','studio_records']};
  if (!bad) { observe(o, expected); console.log('observer accepted'); return; }
  const variants = [
    {...o,published:true}, {...o,unchanged:false}, {...o,runs:1}, {...o,progress:''},
    ...expected.feedback.map(text => ({...o,feedback:o.feedback.split(text).join('')}))
  ];
  for (const v of variants) {
    let rejected = false;
    try { observe(v, expected); } catch (_) { rejected = true; }
    assert.ok(rejected, 'observer accepted a corrupted observation');
  }
  console.log('observer rejected'); process.exitCode = 1;
}
if (require.main === module) {
  qualify(process.argv[2] === 'bad');
} else {
  const {When,Then,setDefaultTimeout} = require('@cucumber/cucumber');
  setDefaultTimeout(240000);
  When(PREFIX + 'Shape evaluates {string}', function(mode) { this.maskedKept = run(mode); });
  Then(PREFIX + 'the unchanged-tree outcome is', function(text) { observe(this.maskedKept, JSON.parse(text)); });
}
