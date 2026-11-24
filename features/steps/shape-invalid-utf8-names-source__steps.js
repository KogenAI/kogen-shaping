'use strict';
const {Given, When, Then, AfterAll} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const observer = require('./shape-invalid-utf8-names-source__driver/observe.js');
let root, binary;

Given('shape-invalid-utf8-names-source: the standalone Shape command is built', {timeout:120000}, function() {
  if (binary) return;
  root = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-utf8-'));
  const target = path.join(root, 'kogen-shape');
  const built = spawnSync('go', ['build', '-o', target, './cmd/kogen-shape'], {encoding:'utf8', timeout:110000});
  assert.ifError(built.error);
  assert.equal(built.status, 0, `go build ./cmd/kogen-shape: ${built.stdout}${built.stderr}`);
  binary = target;
});
When('shape-invalid-utf8-names-source: I supply bytes {string} from {string}', function(hex, source) {
  assert.ok(['file', 'stdin'].includes(source));
  const data = Buffer.from(hex, 'hex');
  const project = fs.mkdtempSync(path.join(root, 'project-'));
  const file = path.join(project, 'request-source.bin');
  if (source === 'file') fs.writeFileSync(file, data);
  const result = spawnSync(binary, ['--project', project, '--slug', 'shape-invalid-utf8-names-source', '--request', source === 'file' ? file : '-'], {
    input: source === 'stdin' ? data : undefined, encoding:'utf8', timeout:10000
  });
  assert.ifError(result.error);
  this.utf8 = {result, file};
});
Then('shape-invalid-utf8-names-source: stdout is exactly one JSON refusal with no Intent and exit code {int}', function(code) {
  observer.refusal(this.utf8.result, code);
});
Then('shape-invalid-utf8-names-source: the error names the offending request file', function() {
  observer.fileSource(this.utf8.result, this.utf8.file);
});
Then('shape-invalid-utf8-names-source: the error identifies stdin rather than a filesystem path', function() {
  observer.stdinSource(this.utf8.result);
});
AfterAll(function() {
  if (root) fs.rmSync(root, {recursive:true, force:true});
});
