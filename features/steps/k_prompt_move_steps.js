'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
function equal(actual, expected) { assert.equal(actual, expected, 'rendered prompt and ordered tool wire bytes must match the pinned executor'); }
if (process.argv[2] === '--control') {
  try { equal(process.argv[3] === 'good' ? 'recorded bytes\n' : 'changed bytes\n', 'recorded bytes\n'); console.log('accepted'); }
  catch (_) { console.log('rejected'); process.exitCode = 1; }
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('I compare the shaping-owned prompt and tools with the pinned executor', {timeout: 60000}, function() {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'prompt-move-'));
    try {
      const source = fs.readFileSync('features/steps/k_prompt_move_main.go.txt', 'utf8');
      const file = path.join(dir, 'main.go');
      const binary = path.join(dir, 'compare');
      fs.writeFileSync(file, source);
      const build = spawnSync('go', ['build', '-o', binary, file], {encoding: 'utf8', env: process.env, maxBuffer: 4 * 1024 * 1024});
      assert.equal(build.status, 0, `go build -o ${binary} ${file}: ${build.error || ''}\n${build.stdout}${build.stderr}`);
      const r = spawnSync(binary, [], {encoding: 'utf8', env: process.env, maxBuffer: 4 * 1024 * 1024});
      assert.equal(r.status, 0, `${binary}: ${r.error || ''}\n${r.stdout}${r.stderr}`);
      this.promptMove = JSON.parse(r.stdout);
    } finally { fs.rmSync(dir, {recursive: true, force: true}); }
  });
  Then('the rendered prompt and ordered tool definitions are byte-identical', function() {
    equal(this.promptMove.actual, this.promptMove.expected);
  });
}
