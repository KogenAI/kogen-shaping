'use strict';
// Control of FIX-K-SHAPE-KILL-GROUP-ZOMBIE's harness, run in the checkout. `good` and `bad` feed the
// harness's runs and judges known-good and known-bad packages in scratch copies, and print one
// line per judgement. On Linux the runs use the reaping subreaper, so the unchanged
// TestKillGroupOnExit passes there too; the new-test judgement is always the one for platforms
// without a child subreaper, so the lines are the same everywhere. `runner` builds the subreaper
// program for linux/amd64 and linux/arm64.
//   good: a planted new test that skips; a kogen-loop copy that leaves the group running.
//   bad:  no new test; a kogen-loop copy that sweeps the group as the pinned one does.
const h = require('./fix-k-shape-kill-group-zombie__harness.js');

const ROOT = process.cwd();
const PLANTED = `package consumes

import "testing"

func TestPlantedNewTest(t *testing.T) { t.Skip("planted by the control") }
`;

function line(label, problems) {
  process.stdout.write(`${label}: ${problems.length ? 'rejected' : 'ok'}\n`);
  return problems.length === 0;
}

function control(good) {
  let ok = true;
  const dir = h.scratchCopy(ROOT, {plant: good ? {'zz_planted_control_test.go': PLANTED} : {}});
  const names = h.newTests(dir);
  ok = line('exactly one new test', h.oneNewTestProblems(names)) && ok;
  const r = h.runTests(dir, ['TestKillGroupOnExit', ...names], 1, 'reap');
  ok = line('go command exits 0', h.exitProblems(r, 0)) && ok;
  ok = line('TestKillGroupOnExit passed', h.reportedProblems(r, 'TestKillGroupOnExit', 'passed', 1)) && ok;
  ok = line('new test skipped', h.newTestProblems(r, names, 'skipped')) && ok;
  const kept = h.scratchCopy(ROOT, {keepRunning: good});
  const k = h.runTests(kept, ['TestKillGroupOnExit'], 1, 'reap');
  ok = line('kept-running copy exits 1', h.exitProblems(k, 1)) && ok;
  ok = line('kept-running copy reports the running process', h.failedWithProblems(k, 'TestKillGroupOnExit', h.RUNNING_TEXT)) && ok;
  return ok;
}

function runner() {
  let ok = true;
  for (const arch of ['amd64', 'arm64']) {
    const r = h.buildSubreaperFor(arch);
    process.stdout.write(`linux/${arch}: ${r.status === 0 ? 'builds' : 'does not build'}\n`);
    if (r.status !== 0) {
      process.stderr.write(r.output);
      ok = false;
    }
  }
  return ok;
}

function main(mode) {
  if (!['good', 'bad', 'runner'].includes(mode)) {
    process.stderr.write('usage: fix-k-shape-kill-group-zombie__control.js good|bad|runner\n');
    process.exit(2);
  }
  process.exit((mode === 'runner' ? runner() : control(mode === 'good')) ? 0 : 1);
}

// Cucumber loads every .js file of the steps directory; only a direct run is the control.
if (require.main === module) main(process.argv[2]);
