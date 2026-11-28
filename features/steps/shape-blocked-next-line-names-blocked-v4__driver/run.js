'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const TEST = 'TestBlockedNextV4Observation';
const REPAIR = 'features/steps/k-shape-blocked-report-repair__steps.js';
let built;
function env(extra = {}) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}
function execute(command, args, options = {}) {
  const r = spawnSync(command, args, {encoding: 'utf8', env: env(), timeout: 55000, maxBuffer: 8 * 1024 * 1024, ...options});
  return {status: r.status, error: r.error ? String(r.error) : '', text: (r.stdout || '') + (r.stderr || '')};
}
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'blocked-next-v4-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/blocked_next_v4_observation_test.go')]: path.join(__dirname, 'report_test.go.txt')}}));
  const bin = path.join(dir, 'shape.test');
  const r = execute('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape']);
  assert.equal(r.status, 0, `${TEST}: go test -c -vet=off -overlay ./shape failed: ${r.error}\n${r.text}`);
  built = bin;
  return bin;
}
function named(names, extra = {}) {
  const bin = build();
  const args = ['-test.run', '^(' + names.join('|') + ')$', '-test.count=1', '-test.v', '-test.timeout=50s'];
  const r = execute(bin, args, {cwd: path.resolve('shape'), env: env(extra)});
  for (const name of names) {
    assert.ok(!r.error && r.status === 0 && r.text.includes('--- PASS: ' + name + ' ') && !r.text.includes('--- SKIP:'), `${name}: ${bin} ${args.join(' ')}; exit ${r.status}; error ${r.error}; required test absent, skipped, zero runs or failed\n${r.text}`);
  }
  return r.text;
}
function report(mode) {
  const text = named([TEST], {BNV4_MODE: mode});
  const lines = text.split('\n').filter(s => s.startsWith('BNV4-OBS:'));
  assert.equal(lines.length, 1, `${TEST}: compiled ./shape test must emit exactly one observation; got ${lines.length}\n${text}`);
  return JSON.parse(lines[0].slice('BNV4-OBS:'.length));
}
function judgeReport(o, expected) {
  assert.equal(o.error, '', 'Shape returned an error');
  assert.equal(o.exit, 1, 'Shape blocked-report exit');
  assert.equal(o.stdout, expected + '\n', 'exact three-line blocked report');
  assert.equal(o.intent, false, 'blocked report wrote an Intent');
  assert.equal(o.approval, false, 'blocked report wrote an approval');
}
function repair(mode) {
  const r = execute(process.execPath, [path.resolve(REPAIR), mode, '--diagnostic']);
  return {...r, source: fs.readFileSync(REPAIR, 'utf8')};
}
function judgeRepair(o, next) {
  assert.equal(o.error, '', 'repair journey process error');
  assert.equal(o.status, 0, `repair journey failed\n${o.text}`);
  assert.ok(o.text.startsWith('accepted\n'), `repair journey did not accept\n${o.text}`);
  assert.ok(o.text.includes('--- PASS: TestKShapeBlockedReportRepairAcceptance '), `TestKShapeBlockedReportRepairAcceptance absent, skipped or zero runs: node ${REPAIR}\n${o.text}`);
  assert.ok(o.source.includes('const kbrNext = "' + next + '\\n"'), 'repair step constant does not hold the requested Next line');
}
function affected() {
  return named(['TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob', 'TestShapeBlockedReportIsPrintedAndWritesNothing']);
}
// Qualify the same assertion functions used by Then, including assertions masked by old copy.
function qualify(polarity) {
  const expected = 'first\nreason\nNext';
  const good = {exit: 1, error: '', stdout: expected + '\n', intent: false, approval: false};
  // Use the real existing runner's output, not hand-written PASS text.
  const repairGood = repair('mixed');
  const match = repairGood.source.match(/const kbrNext = "(.*)\\n"/);
  assert.ok(match, 'repair Next constant absent');
  const repairNext = match[1];
  if (polarity === 'good') {
    judgeReport(good, expected); judgeRepair(repairGood, repairNext);
    console.log('accepted'); return;
  }
  const cases = [
    () => judgeReport({...good, stdout: expected + '\nextra\n'}, expected),
    () => judgeReport({...good, exit: 0}, expected),
    () => judgeReport({...good, error: 'failure'}, expected),
    () => judgeReport({...good, intent: true}, expected),
    () => judgeReport({...good, approval: true}, expected),
    () => judgeRepair({...repairGood, status: 1}, repairNext),
    () => judgeRepair({...repairGood, source: 'invalid constant'}, repairNext),
    () => judgeRepair({...repairGood, error: 'process failed'}, repairNext),
    () => judgeRepair({...repairGood, text: repairGood.text.replace('accepted\n', 'rejected\n')}, repairNext),
    () => judgeRepair({...repairGood, text: repairGood.text.replace('--- PASS: TestKShapeBlockedReportRepairAcceptance ', '--- SKIP: TestKShapeBlockedReportRepairAcceptance ')}, repairNext),
  ];
  for (const check of cases) assert.throws(check);
  console.log('rejected'); process.exitCode = 1;
}
module.exports = {report, judgeReport, repair, judgeRepair, affected};
if (require.main === module) qualify(process.argv[2]);
