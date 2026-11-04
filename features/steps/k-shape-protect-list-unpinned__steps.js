'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const MODULE = 'github.com/KogenAI/kogen-coordinator';
function run(args, env) {
  return spawnSync('go', args, {encoding: 'utf8', env: {...process.env, ...env}, timeout: 240000, maxBuffer: 32 * 1024 * 1024});
}
function expect(result, code, command) {
  assert.equal(result.error, undefined, `${command}: ${result.error}`);
  assert.equal(result.status, code, `${command}: exit ${result.status}, signal ${result.signal}\n${result.stdout}${result.stderr}`);
}
function suite(revision, extra, full) {
  const root = process.cwd();
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'protect-unpinned-'));
  try {
    let work = 'off';
    if (revision !== 'pinned') {
      const located = run(['mod', 'download', '-json', MODULE + '@' + revision], {GOWORK: 'off'});
      expect(located, 0, 'go mod download ' + MODULE + '@' + revision);
      const copy = path.join(temp, 'coordinator');
      fs.cpSync(JSON.parse(located.stdout).Dir, copy, {recursive: true});
      if (extra) {
        const file = path.join(copy, 'project/baseline.go');
        const text = fs.readFileSync(file, 'utf8');
        const anchor = 'all = append(all, ConfigPath)';
        assert.equal(text.split(anchor).length, 2, 'scratch baseline extension anchor must occur exactly once');
        fs.chmodSync(file, 0o600);
        fs.writeFileSync(file, text.replace(anchor, `all = append(all, ConfigPath, ${JSON.stringify(extra)})`));
      }
      work = path.join(temp, 'go.work');
      const version = /^go (\S+)$/m.exec(fs.readFileSync('go.mod', 'utf8'))[1];
      fs.writeFileSync(work, `go ${version}\nuse ${JSON.stringify(root)}\nreplace ${MODULE} => ${JSON.stringify(copy)}\n`);
    }
    const args = ['test', '-count=1', full ? './...' : './shape'];
    if (!full) args.push('-run', '^Test(ApproveCardOutput|ContextBlocksCarryTheContractsAndEarlierRounds|ShapeBlockedSubmissionsNeedAnActualProtectedPathAndAReason|ShapeForceWireCompatibility|ShapeMixedBlockedReportNamesOnlyUnprotectedPathsAndTheCorrectedReportIsAccepted|ShapeSubmissionEnvelopeProblems)$');
    return {command: `GOWORK=${work} go ${args.join(' ')} (coordinator ${revision}, extra ${extra || 'none'})`, result: run(args, {GOWORK: work, PWD: root})};
  } finally { fs.rmSync(temp, {recursive: true, force: true}); }
}
if (process.argv[2] === '--control') {
  const command = 'go test -count=1 ./shape -run ^TestShapeBlockedSubmissionsNeedAnActualProtectedPathAndAReason$';
  const r = run(['test', '-count=1', './shape', '-run', '^TestShapeBlockedSubmissionsNeedAnActualProtectedPathAndAReason$'], {GOWORK: 'off'});
  expect(r, 0, command);
  if (process.argv[3] === 'bad') r.status = 1;
  try { expect(r, 0, command); console.log('accepted'); }
  catch (_) { console.log('rejected'); process.exitCode = 1; }
} else if (process.argv[2] === '--suite') {
  const r = suite(process.argv[3], process.argv[4] || '', true);
  expect(r.result, 0, r.command);
  console.log(r.result.stdout);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('k-shape-protect-list-unpinned: I run the regression suite with coordinator {string} and extra automatic path {string}', {timeout: 110000}, function (revision, extra) {
    this.protectRun = suite(revision, extra, false);
  });
  Then('k-shape-protect-list-unpinned: the regression suite exits {int}', function (code) {
    expect(this.protectRun.result, code, this.protectRun.command);
  });
}
