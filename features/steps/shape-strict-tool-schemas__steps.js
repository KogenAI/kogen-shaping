'use strict';
// Steps of SHAPE-STRICT-TOOL-SCHEMAS. They run go test on the checkout's shape package, with a Go overlay
// that adds the observer tests or a mutated copy of shape/contract.go; the checkout is never written.
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');
const assert = require('assert');
const {Given, When, Then} = require('@cucumber/cucumber');

const ROOT = process.cwd();
const P = 'shape-strict-tool-schemas: ';
const OBSERVER = path.join(ROOT, 'features/steps/shape-strict-tool-schemas__driver/observe_test.go.txt');
const NAMED = /Strict|Schema/;

function goTest(world, args) {
  const cmd = ['go', 'test', '-count=1', '-overlay', world.overlay, ...args, './shape'];
  const r = spawnSync(cmd[0], cmd.slice(1), {cwd: ROOT, encoding: 'utf8', maxBuffer: 1 << 28, env: {...process.env, GOFLAGS: '-trimpath'}});
  return {cmd: cmd.join(' '), status: r.status, output: (r.stdout || '') + (r.stderr || '')};
}

function overlayDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'shape-strict-'));
}

function writeOverlay(dir, replace) {
  const file = path.join(dir, 'overlay.json');
  fs.writeFileSync(file, JSON.stringify({Replace: replace}));
  return file;
}

function mutatedContract(mutate) {
  const src = fs.readFileSync(path.join(ROOT, 'shape/contract.go'), 'utf8');
  const out = mutate(src);
  assert.notStrictEqual(out, src, 'shape/contract.go no longer has the confirmation verdict item the scratch mutation edits');
  const dir = overlayDir();
  const copy = path.join(dir, 'contract.go');
  fs.writeFileSync(copy, out);
  return writeOverlay(dir, {[path.join(ROOT, 'shape/contract.go')]: copy});
}

Given(P + 'a scratch overlay of the contract source file whose confirmation verdict item leaves "why" out of its required list', function () {
  this.overlay = mutatedContract(s => s.replace(/\[\]any\{"finding", "verdict", "why"\}/, '[]any{"finding", "verdict"}'));
});

Given(P + 'a scratch overlay of the contract source file whose confirmation verdict item has no additionalProperties false', function () {
  this.overlay = mutatedContract(s => s.replace(/(\[\]any\{"finding", "verdict", "why"\},\s*)"additionalProperties":\s*false,/, '$1'));
});

Given(P + 'the checkout as it is', function () {
  this.overlay = writeOverlay(overlayDir(), {});
});

When(P + 'I run the observer tests of the strict tool schemas', {timeout: 10 * 60 * 1000}, function () {
  this.overlay = writeOverlay(overlayDir(), {[path.join(ROOT, 'shape/zz_shape_strict_tool_schemas_observer_test.go')]: OBSERVER});
  this.ran = goTest(this, ['-v', '-run', '^(TestShapeStrictToolSchemasObserver|TestShapeKeptStepsObserver)$']);
});

When(P + 'I run the package tests whose names mention strict or schema', {timeout: 10 * 60 * 1000}, function () {
  const list = goTest(this, ['-list', '.']);
  assert.strictEqual(list.status, 0, `${list.cmd} failed:\n${list.output.slice(-3000)}`);
  this.names = list.output.split('\n').filter(l => /^Test\w*$/.test(l) && NAMED.test(l));
  this.ran = this.names.length
    ? goTest(this, ['-v', '-run', '^(' + this.names.join('|') + ')$'])
    : {cmd: '(no test names matched)', status: null, output: ''};
});

Then(P + 'the go command exits {int}', function (code) {
  assert.strictEqual(this.ran.status, code, `${this.ran.cmd} exited ${this.ran.status}, want ${code}:\n${this.ran.output.slice(-3000)}`);
});

Then(P + 'the observer test {string} is reported as passed', function (name) {
  assert.ok(this.ran.output.includes(`--- PASS: ${name} `), `${name} was not reported as passed by ${this.ran.cmd}:\n${this.ran.output.slice(-3000)}`);
});

Then(P + 'at least one of those tests is reported as failed and its output contains {string}', function (text) {
  assert.ok(this.names.length > 0, 'no top-level test of the shape package has a name containing Strict or Schema');
  assert.ok(!/\[build failed\]|\[setup failed\]/.test(this.ran.output), `${this.ran.cmd} did not build:\n${this.ran.output.slice(-3000)}`);
  assert.ok(/^--- FAIL: \S+ /m.test(this.ran.output), `no test was reported as failed by ${this.ran.cmd}:\n${this.ran.output.slice(-3000)}`);
  assert.ok(this.ran.output.includes(text), `the failure output of ${this.ran.cmd} does not contain ${JSON.stringify(text)}:\n${this.ran.output.slice(-3000)}`);
});
