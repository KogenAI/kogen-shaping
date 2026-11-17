'use strict';
// Acceptance of shape-use-studio-rule-snapshot. The When step compiles the shape package's tests with the
// harness of shape-use-studio-rule-snapshot__harness.js through a Go overlay (once per process), runs
// TestShapeUseStudioRuleSnapshotAcceptance for one journey with SRS_CASE and reads the JSON observation the
// harness wrote to SRS_OUT (once per journey and process). The Then steps compare it with the feature.
const {Given, When, Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const P = 'shape-use-studio-rule-snapshot: ';
const TARGET = 'shape/shape_use_studio_rule_snapshot_acceptance_test.go';
const TEST = 'TestShapeUseStudioRuleSnapshotAcceptance';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let built;
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'srs-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./shape-use-studio-rule-snapshot__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env({}), encoding: 'utf8', timeout: 100000});
  built = {dir, bin, ok: r.status === 0, output: (r.stdout || '') + (r.stderr || '')};
  return built;
}

const observed = {};
function journey(name) {
  if (observed[name]) return observed[name];
  const b = build();
  assert.ok(b.ok, P + 'the harness did not compile: ' + b.output);
  const out = path.join(b.dir, name + '.json');
  const r = spawnSync(b.bin, ['-test.run', '^' + TEST + '$', '-test.count=1'], {cwd: path.resolve('shape'), env: env({SRS_CASE: name, SRS_OUT: out}), encoding: 'utf8', timeout: 100000});
  assert.ok(fs.existsSync(out), P + 'journey ' + name + ' wrote no observation: ' + (r.stdout || '') + (r.stderr || ''));
  observed[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observed[name];
}

function hashOf(rev) {
  const rules = [{id: 'rule-repo-1', revision: rev, scope: 'repo', topic: 'tests', text: 'SRS rule text revision ' + rev, owner: 'team', overrides: null}];
  return 'sha256:' + crypto.createHash('sha256').update(JSON.stringify(rules)).digest('hex');
}
function receiptOf(o, i) {
  const r = o.receipts[i];
  assert.ok(r, P + 'no studio_rules receipt entry number ' + (i + 1) + ': ' + JSON.stringify(o.receipts));
  return r;
}
function expectReceipt(r, rev, origin, boundary) {
  assert.deepEqual(r.rules, [{id: 'rule-repo-1', revision: rev}], P + 'receipt rules');
  assert.equal(r.snapshot_hash, hashOf(rev), P + 'receipt snapshot hash');
  assert.equal(r.origin, origin, P + 'receipt origin');
  if (boundary) assert.equal(r.boundary, boundary, P + 'receipt boundary');
}
function carries(run, rev) {
  assert.ok(run.shaper_requests >= 1, P + 'the shaper made no request: ' + JSON.stringify(run));
  assert.equal(run.shaper_requests_with_rules, run.shaper_requests, P + 'every shaper request carries the rules: ' + JSON.stringify(run));
  assert.equal(run.contract_requests_with_rules, run.contract_requests, P + 'every contract-review request carries the rules: ' + JSON.stringify(run));
  const other = rev === 3 ? '4' : '3';
  assert.ok(run.request_revisions[String(rev)] && !run.request_revisions[other], P + 'requests carry revision ' + rev + ' only: ' + JSON.stringify(run.request_revisions));
  assert.equal(run.token_in_requests, false, P + 'the token is in a request');
}
function refusedRun(class_, reason, runIndex) {
  const run = current.runs[runIndex];
  assert.notEqual(run.exit, 0, P + 'the run continued: ' + JSON.stringify(run));
  assert.ok(run.error.includes(class_), P + 'error names class ' + class_ + ': ' + run.error);
  assert.ok(run.error.includes(reason), P + 'error names reason ' + reason + ': ' + run.error);
  assert.equal(run.shaper_requests, 0, P + 'the shaper ran after the refusal: ' + JSON.stringify(run));
}

let current;
Given(P + 'the fake Studio endpoint serves rule rule-repo-1 at revision 3 for the synthetic token', function () {});
Given(P + 'a project with no Studio configuration', function () {});
When(P + 'the {word} journey is run', function (name) { current = journey(name); });

Then(P + 'the live run succeeds, every shaper and contract-review request carries rule-repo-1 revision 3, and the receipt records rule-repo-1 revision 3 with its snapshot hash and origin live', function () {
  const run = current.runs[0];
  assert.equal(run.exit, 0, P + 'the run failed: ' + JSON.stringify(run));
  carries(run, 3);
  assert.equal(current.receipts.length, 1, P + 'one receipt entry: ' + JSON.stringify(current.receipts));
  expectReceipt(receiptOf(current, 0), 3, 'live', 'run');
  assert.ok(!current.token_in_events && !run.stderr_has_token, P + 'a token was logged');
});
Then(P + 'the second run carries and records rule-repo-1 revision 4 with no prompt file edited', function () {
  assert.equal(current.runs[0].exit, 5, P + 'the first run asks: ' + JSON.stringify(current.runs[0]));
  carries(current.runs[0], 3);
  assert.equal(current.runs[1].exit, 0, P + 'the second run failed: ' + JSON.stringify(current.runs[1]));
  carries(current.runs[1], 4);
  expectReceipt(receiptOf(current, 0), 3, 'live');
  expectReceipt(receiptOf(current, 1), 4, 'live');
});
Then(P + 'the run succeeds and no rule receipt entry is written', function () {
  assert.equal(current.runs[0].exit, 0, P + 'the run failed: ' + JSON.stringify(current.runs[0]));
  assert.equal(current.receipts.length, 0, P + 'a receipt entry was written: ' + JSON.stringify(current.receipts));
  assert.equal(current.refusals.length, 0, P + 'a refusal was recorded: ' + JSON.stringify(current.refusals));
  assert.equal(current.runs[0].request_revisions['3'], undefined, P + 'rules reached a request');
});
Then(P + "the answer run's requests carry rule-repo-1 revision 3 and its answer boundary records rule-repo-1 revision 3, the snapshot hash and origin live", function () {
  assert.equal(current.runs[1].exit, 0, P + 'the answer run failed: ' + JSON.stringify(current.runs[1]));
  carries(current.runs[1], 3);
  expectReceipt(receiptOf(current, 1), 3, 'live', 'answer');
});
Then(P + "the outage run's receipt says origin cached with rule-repo-1 revision 3 and its requests carry revision 3 only", function () {
  assert.equal(current.runs[1].exit, 0, P + 'the outage run failed: ' + JSON.stringify(current.runs[1]));
  carries(current.runs[1], 3);
  expectReceipt(receiptOf(current, 1), 3, 'cached', 'answer');
});
Then(P + 'the revoked token run is refused as rules\\/refused http_401 and no rule set reaches a request', function () {
  refusedRun('rules/refused', 'http_401', 1);
  assert.equal(current.receipts.length, 1, P + 'a receipt was written for the denied run: ' + JSON.stringify(current.receipts));
});
Then(P + 'the foreign scope run is refused as rules\\/refused http_403 and no rule set reaches a request', function () {
  refusedRun('rules/refused', 'http_403', 1);
  assert.equal(current.receipts.length, 1, P + 'a receipt was written for the denied run: ' + JSON.stringify(current.receipts));
});
Then(P + 'the other scope during an outage is reported as rules\\/unavailable http_503 and the first scope cache is not reused', function () {
  refusedRun('rules/unavailable', 'http_503', 1);
  assert.equal(current.receipts.length, 1, P + 'a receipt was written for the refused run: ' + JSON.stringify(current.receipts));
});
Then(P + 'the unavailable run without a cache is reported as rules\\/unavailable http_503 and no empty rule set is used', function () {
  refusedRun('rules/unavailable', 'http_503', 0);
  assert.equal(current.receipts.length, 0, P + 'a receipt was written: ' + JSON.stringify(current.receipts));
});
