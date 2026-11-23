'use strict';
// Steps of DOCUMENTATION-PROGRESS-STRIPS-CONTROLS. They build a small Go program that calls
// shape.DocumentationProgress on a retained-lookup row, run the binary and read the two progress
// lines it prints.
const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');
const {Given, When, Then} = require('@cucumber/cucumber');

const P = 'documentation-progress-strips-controls: ';
const DRIVER = './features/steps/documentation-progress-strips-controls__driver';

let binary = null;

// build compiles the driver once into a scratch directory and returns the binary path.
function build() {
  if (binary) return binary;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'doc-progress-driver-'));
  const out = path.join(dir, 'driver');
  const r = spawnSync('go', ['build', '-o', out, DRIVER], {encoding: 'utf8', timeout: 100 * 1000, cwd: process.cwd()});
  if (r.status !== 0) {
    throw new Error('go build ' + DRIVER + ' exited ' + r.status + ': ' + r.stderr + (r.error ? ' ' + r.error : ''));
  }
  binary = out;
  return binary;
}

// decode turns each {U+XXXX} marker of a feature string into that code point.
function decode(text) {
  return text.replace(/\{U\+([0-9A-Fa-f]{4,6})\}/g, (_, hex) => String.fromCodePoint(parseInt(hex, 16)));
}

Given(P + 'a retained search lookup whose query is {string}', function (text) {
  this.row = {operation: 'search', status: 'unsupported', query: decode(text), url: ''};
});

Given(P + 'a retained fetch lookup whose url is {string}', function (text) {
  this.row = {operation: 'fetch', status: 'retrieved', query: '', url: decode(text)};
});

When(P + 'the documentation progress lines are produced', {timeout: 115 * 1000}, function () {
  const bin = build();
  const r = spawnSync(bin, [], {input: JSON.stringify(this.row), encoding: 'utf8', timeout: 10 * 1000});
  if (r.status !== 0) {
    throw new Error(bin + ' exited ' + r.status + ': ' + r.stderr + (r.error ? ' ' + r.error : ''));
  }
  this.lines = JSON.parse(r.stdout.trim().split('\n').pop());
});

Then(P + 'the started progress line is {string}', function (want) {
  assert.strictEqual(this.lines.started, decode(want), 'started progress line');
});

Then(P + 'the started progress line is one line without control characters', function () {
  const got = this.lines.started;
  assert.ok(!/[\u0000-\u001f\u007f-\u009f]/.test(got), 'started progress line holds a control character: ' + JSON.stringify(got));
});
