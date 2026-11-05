'use strict';
// Steps of SH-CONSUMES-TOOLS-REQUIRE-EXECUTABLE. They run in the checkout acceptance runs in (the
// current directory); the go test runs and their judgements live in the harness.
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {When, Then} = require('@cucumber/cucumber');
const h = require('./sh-consumes-tools-require-executable__harness.js');

const ROOT = process.cwd();
const P = 'sh-consumes-tools-require-executable: ';

function none(problems) {
  if (problems.length) throw new Error(problems.join('\n'));
}

When(P + 'I run TestReadinessCheckAndKeptEvidence and TestToolsFromInstall against the {string} kogen-verification', {timeout: 6 * 60 * 1000}, function (variant) {
  this.result = h.runConsumerTests(ROOT, variant);
});

Then(P + 'both tests pass', function () {
  none(h.passProblems(this.result));
});

Then(P + '{string} fails', function (test) {
  none(h.failProblems(this.result, test));
});

When(P + "I read the checkout's go.mod and go.sum", {timeout: 60 * 1000}, function () {
  const r = spawnSync('go', ['mod', 'edit', '-json'], {cwd: ROOT, encoding: 'utf8'});
  if (r.error || r.status !== 0) throw new Error(`go mod edit -json failed: ${r.error ? r.error.message : 'exit ' + r.status}\n${r.stdout || ''}${r.stderr || ''}`);
  this.mod = JSON.parse(r.stdout);
  this.goSum = crypto.createHash('sha256').update(fs.readFileSync(path.join(ROOT, 'go.sum'))).digest('hex');
});

Then(P + 'go.mod requires exactly these modules, in this order:', function (table) {
  const got = (this.mod.Require || []).map(r => `${r.Path} ${r.Version}${r.Indirect ? ' // indirect' : ''}`);
  const want = table.hashes().map(w => `${w.module} ${w.version}`);
  if (JSON.stringify(got) !== JSON.stringify(want)) throw new Error(`go.mod requires ${JSON.stringify(got)}, want exactly ${JSON.stringify(want)}`);
});

Then(P + 'go.mod has no replace directive', function () {
  const replaces = (this.mod.Replace || []).map(r => `${r.Old.Path}${r.Old.Version ? ' ' + r.Old.Version : ''} => ${r.New.Path}${r.New.Version ? ' ' + r.New.Version : ''}`);
  if (replaces.length) throw new Error(`go.mod has replace directives: ${replaces.join('; ')}`);
});

Then(P + 'go.sum has the SHA-256 {string}', function (want) {
  if (this.goSum !== want) throw new Error(`go.sum has SHA-256 ${this.goSum}, want ${want} (the pins must not change)`);
});
