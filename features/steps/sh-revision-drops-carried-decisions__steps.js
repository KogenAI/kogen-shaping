'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

function check(actual, expected) {
  assert.equal(actual.first, expected, 'first revision must preserve the whole draft and restore every named line');
  assert.equal(actual.second, expected, 'second revision must not duplicate or discard a Decision');
}
if (require.main === module) {
  try {
    check({first: 'restored\n', second: process.argv[2] === 'good' ? 'restored\n' : 'lost\n'}, 'restored\n');
    console.log('accepted');
  } catch (_) { console.log('rejected'); process.exitCode = 1; }
} else {
  const {Given, Then} = require('@cucumber/cucumber');
  let binary;
  function run(cmd, args, opts) {
    const r = spawnSync(cmd, args, {encoding: 'utf8', timeout: 180000, maxBuffer: 16 << 20, ...opts});
    assert.equal(r.status, 0, cmd + ' ' + args.join(' ') + ': ' + (r.error || '') + '\n' + r.stdout + r.stderr);
    return r.stdout;
  }
  function build() {
    if (binary) return binary;
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'carried-revision-'));
    process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
    const root = path.resolve(__dirname, '../..');
    const source = path.join(dir, 'revision_test.go');
    fs.writeFileSync(source, `package shape
import ("encoding/json"; "fmt"; "os"; "regexp"; "testing")
func TestCarriedRevisionAcceptance(t *testing.T) {
 var in struct {Draft string; Lines []string}
 if err := json.Unmarshal([]byte(os.Getenv("SH_REVISION_INPUT")), &in); err != nil {t.Fatal(err)}
 var problems []string
 re := regexp.MustCompile("decision: (D[0-9]+), round ([0-9]+)")
 for _, line := range in.Lines {m := re.FindStringSubmatch(line); if len(m)!=3 {t.Fatal("invalid supplied decision line")}; problems=append(problems, fmt.Sprintf("- %s (round %s) is not in Decisions; keep it as: %s",m[1],m[2],line))}
 first := repairCarriedDecisionDraft(in.Draft, problems)
 second := repairCarriedDecisionDraft(first, problems)
 b, err := json.Marshal(map[string]string{"first":first,"second":second}); if err!=nil {t.Fatal(err)}
 fmt.Println("SH_REVISION_RESULT "+string(b))
}
`);
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(root, 'shape/sh_revision_acceptance_test.go')]: source}}));
    const bin = path.join(dir, 'shape.test');
    run('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {cwd: root});
    binary = bin;
    return binary;
  }
  Given('sh-revision-drops-carried-decisions: a refused draft and exact keep-lines', function (doc) {
    this.revisionInput = JSON.parse(doc);
    assert.equal(typeof this.revisionInput.draft, 'string');
    assert.ok(Array.isArray(this.revisionInput.lines));
  });
  Then('sh-revision-drops-carried-decisions: revising twice produces exactly', function (expected) {
    const root = path.resolve(__dirname, '../..');
    const output = run(build(), ['-test.run=^TestCarriedRevisionAcceptance$', '-test.v', '-test.count=1'], {cwd: path.join(root, 'shape'), env: {...process.env, SH_REVISION_INPUT: JSON.stringify(this.revisionInput)}});
    const lines = output.split('\n').filter(x => x.startsWith('SH_REVISION_RESULT '));
    assert.equal(lines.length, 1, 'TestCarriedRevisionAcceptance: built test binary must execute the revision once and report one result; output:\n' + output);
    check(JSON.parse(lines[0].slice('SH_REVISION_RESULT '.length)), expected + '\n');
  });
}
