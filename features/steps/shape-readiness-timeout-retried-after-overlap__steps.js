'use strict';
// Acceptance of shape-readiness-timeout-retried-after-overlap. The Then steps compile the shape
// package's tests with the Go test below (through a Go overlay) and run it once per case. Each case
// drives the package's real cheapGates (with a real configured static check) and joinReadiness with
// a scripted readiness through the readinessObserver seam, and writes what happened as JSON.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {Given, When, Then} = require('@cucumber/cucumber');

const P = 'shape-readiness-timeout-retried-after-overlap: ';
const TEST = 'TestSRTAOAcceptance';
const CASES = ['repeat', 'both', 'alone', 'assertion'];
const PROGRESS = 'kogen: readiness timed out while other checks ran; repeating it alone';
const ADDED = 'Readiness was repeated alone after the concurrent checks ended; both runs timed out.';
const FIRST = 'acceptance timed out: command exceeds the 120-second limit';
let ran;
let current;

const GO = [
  'package shape',
  '',
  'import (',
  '\t"context"',
  '\t"encoding/json"',
  '\t"os"',
  '\t"path/filepath"',
  '\t"strings"',
  '\t"sync/atomic"',
  '\t"testing"',
  '\t"time"',
  '',
  '\t"github.com/KogenAI/kogen-loop/sys"',
  '\t"github.com/KogenAI/kogen-verification/verify"',
  ')',
  '',
  'func srtaoTimeout() error {',
  '\te := sys.Fail("intent", "not_ready", "acceptance timed out", 1)',
  '\te.Lines = []string{"acceptance timed out: command exceeds the 120-second limit", "Acceptance ran for 120.2s before it was stopped with its process group.", "It wrote no output before it was stopped.", "host load at start: 1-minute 9.50, 5-minute 7.46; load alone does not establish the cause."}',
  '\treturn e',
  '}',
  '',
  'func srtaoAssertion() error {',
  '\te := sys.Fail("intent", "not_ready", "acceptance failed", 1)',
  '\te.Lines = []string{"acceptance failed: scenario \\"x\\" step failed"}',
  '\treturn e',
  '}',
  '',
  'func TestSRTAOAcceptance(t *testing.T) {',
  '\tout := os.Getenv("SRTAO_OUT")',
  '\tfor _, name := range strings.Split(os.Getenv("SRTAO_CASES"), ",") {',
  '\t\tname := name',
  '\t\tt.Run(name, func(t *testing.T) { srtaoCase(t, name, out) })',
  '\t}',
  '}',
  '',
  'func srtaoCase(t *testing.T, name, out string) {',
  '\toverlap := name != "alone"',
  '\tf := newShpFx(t)',
  '\tstarted, done := filepath.Join(f.Aux, "static-started"), filepath.Join(f.Aux, "static-done")',
  '\tcommand := "touch " + started + "; sleep 0.4; touch " + done',
  '\tf.write(".kogen/project.yaml", "version: 1\\ncommands:\\n  checks:\\n    - name: slow\\n      run: \'"+command+"\'\\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\\n")',
  '\tf.git("add", "-A")',
  '\tf.git("commit", "-qm", "sleeping check")',
  '\tsh := f.shaping()',
  '\tsh.request, sh.role, sh.started, sh.submissions = orderRequest, sh.cfg.Models["shape"], time.Now(), 1',
  '\tsh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S), Evidence: sh.ev}',
  '\tvar files []string',
  '\tfor _, file := range shpFiles()["files"].([]any) {',
  '\t\tm := file.(map[string]any)',
  '\t\tf.write(m["path"].(string), m["content"].(string))',
  '\t\tfiles = append(files, m["path"].(string))',
  '\t}',
  '\tdecisionAnswer, literal := orderDecisionRefusal(), orderLiteralRefusal()',
  '\tf.Prov.route = func(req map[string]any) *shpStep {',
  '\t\tif decisionRequest(req) {',
  '\t\t\treturn &decisionAnswer',
  '\t\t}',
  '\t\tif literalRequest(req) {',
  '\t\t\treturn &literal',
  '\t\t}',
  '\t\treturn nil',
  '\t}',
  '\tvar calls atomic.Int32',
  '\tvar doneSeen atomic.Bool',
  '\tsh.readinessObserver = func(ctx context.Context, files []string) (certification, error) {',
  '\t\tn := calls.Add(1)',
  '\t\tif n == 1 {',
  '\t\t\tif overlap {',
  '\t\t\t\tfor i := 0; i < 6000; i++ {',
  '\t\t\t\t\tif _, err := os.Stat(started); err == nil {',
  '\t\t\t\t\t\tbreak',
  '\t\t\t\t\t}',
  '\t\t\t\t\ttime.Sleep(5 * time.Millisecond)',
  '\t\t\t\t}',
  '\t\t\t}',
  '\t\t\ttime.Sleep(30 * time.Millisecond)',
  '\t\t} else {',
  '\t\t\t_, err := os.Stat(done)',
  '\t\t\tdoneSeen.Store(err == nil)',
  '\t\t}',
  '\t\tswitch name {',
  '\t\tcase "repeat":',
  '\t\t\tif n == 1 {',
  '\t\t\t\treturn certification{}, srtaoTimeout()',
  '\t\t\t}',
  '\t\t\treturn certification{text: "accepted: the Intent is ready", out: outcome{kind: outReady}}, nil',
  '\t\tcase "assertion":',
  '\t\t\treturn certification{}, srtaoAssertion()',
  '\t\t}',
  '\t\treturn certification{}, srtaoTimeout()',
  '\t}',
  '\tctx := context.Background()',
  '\tvar run *readinessRun',
  '\tif overlap {',
  '\t\tcheap, err := sh.cheapGates(ctx, files, "srtao-package")',
  '\t\tif err != nil {',
  '\t\t\tt.Fatal(err)',
  '\t\t}',
  '\t\trun = cheap.readiness',
  '\t\tif run == nil {',
  '\t\t\tt.Fatal("cheapGates started no readiness observation")',
  '\t\t}',
  '\t} else {',
  '\t\trun = sh.startReadiness(ctx, files, "srtao-package")',
  '\t}',
  '\tcert, err := sh.joinReadiness(ctx, files, run, "srtao-package")',
  '\tres := map[string]any{"case": name, "calls": calls.Load(), "static_done_before_second": doneSeen.Load(),',
  '\t\t"ready": err == nil && cert.out.kind == outReady, "stderr": f.Stderr.String(), "events": f.events()}',
  '\tif e, ok := sys.AsError(err); ok {',
  '\t\tres["error"] = map[string]any{"class": e.Class, "reason": e.Reason, "detail": e.Detail, "lines": e.Lines}',
  '\t} else if err != nil {',
  '\t\tres["error_text"] = err.Error()',
  '\t}',
  '\tb, _ := json.Marshal(res)',
  '\tif err := os.WriteFile(filepath.Join(out, name+".json"), b, 0o644); err != nil {',
  '\t\tt.Fatal(err)',
  '\t}',
  '}',
  ''
].join('\n');

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function runAll() {
  if (ran) return ran;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'srtao-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'srtao_acceptance_test.go');
  fs.writeFileSync(source, GO);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/srtao_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 100000});
  if (b.status !== 0) {
    ran = {error: `${TEST}: go test -c -vet=off -overlay ./shape failed (the shape package lacks the readinessObserver seam or does not compile): exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`};
    return ran;
  }
  const out = path.join(dir, 'out');
  fs.mkdirSync(out);
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=90s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({SRTAO_CASES: CASES.join(','), SRTAO_OUT: out}), encoding: 'utf8', timeout: 95000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ')) {
    ran = {error: `${TEST}: shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`};
    return ran;
  }
  ran = {seen: {}};
  for (const name of CASES) ran.seen[name] = JSON.parse(fs.readFileSync(path.join(out, name + '.json'), 'utf8'));
  return ran;
}

function seen() {
  assert.ok(current, 'no scripted case was chosen by a Given step');
  const r = runAll();
  if (r.error) throw new Error(r.error);
  return r.seen[current];
}

function describe(o) {
  return `case ${o.case}: calls=${o.calls} ready=${o.ready} error=${JSON.stringify(o.error)} events=${JSON.stringify(o.events.map(e => e.event))}`;
}

function repeatEvents(o) {
  return o.events.filter(e => e.event === 'readiness_repeated');
}

Given(P + 'a scripted readiness that times out on its first call and passes on its second, with a static check running beside it', () => { current = 'repeat'; });
Given(P + 'a scripted readiness that times out on both calls, with a static check running beside it', () => { current = 'both'; });
Given(P + 'a scripted readiness that times out with no static check or model check beside it', () => { current = 'alone'; });
Given(P + 'a scripted readiness that refuses for a failing acceptance assertion, with a static check running beside it', () => { current = 'assertion'; });

When(P + "the submission's readiness is joined", () => { assert.ok(current); });

Then(P + 'the submission is ready', () => {
  const o = seen();
  assert.equal(o.ready, true, 'the submission must be ready after the repeat; ' + describe(o));
});

Then(P + 'the scripted readiness was called {int} times', n => {
  const o = seen();
  assert.equal(o.calls, n, `the scripted readiness was called ${o.calls} times, want ${n}; ` + describe(o));
});

Then(P + 'the second call began after the static check ended', () => {
  const o = seen();
  assert.equal(o.static_done_before_second, true, 'the repeat started while the static check was still running; ' + describe(o));
});

Then(P + 'the progress line was written', () => {
  const o = seen();
  assert.ok(o.stderr.includes(PROGRESS), `progress line ${JSON.stringify(PROGRESS)} missing from stderr: ${JSON.stringify(o.stderr)}`);
});

Then(P + 'one repeat event carries the wall time of the first run', () => {
  const o = seen();
  const ev = repeatEvents(o);
  assert.equal(ev.length, 1, 'want one readiness_repeated event; ' + describe(o));
  assert.equal(typeof ev[0].first_wall_ms, 'number', 'first_wall_ms of the repeat event must be a number: ' + JSON.stringify(ev[0]));
  assert.ok(ev[0].first_wall_ms > 0, 'first_wall_ms must be the positive wall time of the timed-out run: ' + JSON.stringify(ev[0]));
  assert.equal(ev[0].package, 'srtao-package', 'the repeat event names the package hash: ' + JSON.stringify(ev[0]));
});

Then(P + 'no refusal was recorded', () => {
  const o = seen();
  assert.ok(!o.error, 'readiness refused: ' + describe(o));
  assert.equal(o.events.filter(e => e.event === 'submission_refused').length, 0, 'a submission_refused event was recorded; ' + describe(o));
});

Then(P + 'the submission is refused as an acceptance timeout whose last line is {string} after the existing timeout lines', last => {
  const o = seen();
  assert.ok(o.error, 'the submission was not refused; ' + describe(o));
  assert.deepEqual([o.error.class, o.error.reason, o.error.detail], ['intent', 'not_ready', 'acceptance timed out'], describe(o));
  assert.equal(last, ADDED);
  assert.equal(o.error.lines.length, 5, 'want the 4 existing lines plus one added line: ' + JSON.stringify(o.error.lines));
  assert.equal(o.error.lines[0], FIRST);
  assert.equal(o.error.lines[4], last, 'last line: ' + JSON.stringify(o.error.lines));
});

Then(P + 'the submission is refused as an acceptance timeout with exactly the existing lines', () => {
  const o = seen();
  assert.ok(o.error, 'the submission was not refused; ' + describe(o));
  assert.deepEqual([o.error.class, o.error.reason, o.error.detail], ['intent', 'not_ready', 'acceptance timed out'], describe(o));
  assert.equal(o.error.lines.length, 4, 'want exactly the 4 existing lines: ' + JSON.stringify(o.error.lines));
  assert.equal(o.error.lines[0], FIRST);
});

Then(P + 'the submission is refused as {string} with its existing lines', detail => {
  const o = seen();
  assert.ok(o.error, 'the submission was not refused; ' + describe(o));
  assert.deepEqual([o.error.class, o.error.reason, o.error.detail], ['intent', 'not_ready', detail], describe(o));
  assert.deepEqual(o.error.lines, ['acceptance failed: scenario "x" step failed'], 'refusal lines changed');
});

Then(P + 'no repeat event or progress line was written', () => {
  const o = seen();
  assert.equal(repeatEvents(o).length, 0, 'a readiness_repeated event was recorded; ' + describe(o));
  assert.ok(!o.stderr.includes(PROGRESS), 'the repeat progress line was written: ' + JSON.stringify(o.stderr));
});
