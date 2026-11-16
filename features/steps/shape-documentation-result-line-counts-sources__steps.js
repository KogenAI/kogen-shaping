'use strict';
// Acceptance of shape-documentation-result-line-counts-sources. The Given step compiles the shape
// package tests with the Go harness shape-documentation-result-line-counts-sources__driver/harness_test.go.txt
// (copied in through a Go overlay as a new test file), runs it once per Cucumber process and reads the
// JSON observation it wrote to SDRLCS_OUT. The Then steps compare that observation with the feature.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-documentation-result-line-counts-sources: ';
const TEST = 'TestDocumentationResultLineHarness';
const HARNESS = path.resolve('features/steps/shape-documentation-result-line-counts-sources__driver/harness_test.go.txt');

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let observation = null;
function observe() {
  if (observation !== null) return observation;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sdrlcs-'));
  owned.push(dir);
  const overlay = path.join(dir, 'overlay.json');
  const target = path.resolve('shape/shape_documentation_result_line_counts_sources_harness_test.go');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: HARNESS}}));
  const out = path.join(dir, 'observation.json');
  const args = ['test', '-vet=off', '-overlay', overlay, '-count=1', '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {env: env({SDRLCS_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error.message : ''}:\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').split('\n').some(l => l.startsWith('--- PASS: ' + TEST + ' ')), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

Given(P + 'a search, a fetch and an unsupported lookup have been retained and their progress lines printed', function () {
  observe();
});

Then(P + 'the search line for a lookup with two sources is {string}', function (want) {
  assert.equal(observe().two.finished, want);
});

Then(P + 'the search line for a lookup with no sources is {string}', function (want) {
  assert.equal(observe().zero.finished, want);
});

Then(P + 'the search line for a lookup with one source is {string}', function (want) {
  assert.equal(observe().one.finished, want);
});

Then(P + 'the fetch line is {string}', function (want) {
  assert.equal(observe().fetch.finished, want);
});

Then(P + 'the unsupported search line is {string}', function (want) {
  assert.equal(observe().unsupported.finished, want);
});

Then(P + 'the started line of the two-source search is {string}', function (want) {
  assert.equal(observe().two.started, want);
});
