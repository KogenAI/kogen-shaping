'use strict';
// Steps of SHAPE-CONTRACT-REVIEW-DRIVER-CONTRADICTIONS. They copy the checkout acceptance runs in to a
// scratch directory, add the driver test of this Intent to package shape there and run it once;
// each step reads one subtest's result from that run.
const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');
const {Then} = require('@cucumber/cucumber');

const ROOT = process.cwd();
const P = 'shape-contract-review-driver-contradictions: ';
const DRIVER = path.join(ROOT, 'features/steps/shape-contract-review-driver-contradictions__driver/driver_test.go.txt');
const TEST = 'TestDriverContradictionsAcceptance';

let ran = null;

function run() {
  if (ran) return ran;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'scrd-'));
  fs.cpSync(ROOT, dir, {
    recursive: true,
    filter: (src) => !['.git', 'node_modules'].includes(path.basename(src)),
  });
  fs.copyFileSync(DRIVER, path.join(dir, 'shape', 'zz_driver_contradictions_test.go'));
  const cmd = `go test -count=1 -run '^${TEST}$' -v ./shape`;
  const r = spawnSync('sh', ['-c', cmd], {cwd: dir, encoding: 'utf8', timeout: 250000, maxBuffer: 64 * 1024 * 1024});
  fs.rmSync(dir, {recursive: true, force: true});
  ran = {cmd, status: r.status, output: (r.stdout || '') + (r.stderr || '')};
  return ran;
}

Then(P + 'the driver subtest {string} passes', {timeout: 5 * 60 * 1000}, function (name) {
  const r = run();
  const line = `--- PASS: ${TEST}/${name.replace(/ /g, '_')} `;
  assert.ok(
    r.output.includes(line),
    `subtest ${name} of ${TEST} did not pass; command ${r.cmd} exited ${r.status}:\n${r.output.slice(-4000)}`,
  );
});
