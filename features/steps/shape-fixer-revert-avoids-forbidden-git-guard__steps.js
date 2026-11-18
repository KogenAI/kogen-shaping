'use strict';
// Steps of SHAPE-FIXER-REVERT-AVOIDS-FORBIDDEN-GIT-GUARD. They copy the checkout to a scratch directory,
// move the copy to the kogen-coordinator guard of e643661cb646 when it pins an older one, add the probe
// tests of the driver directory to its shape package and run one of them with go test.
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');
const {Given, When, Then} = require('@cucumber/cucumber');

const ROOT = process.cwd();
const P = 'shape-fixer-revert-avoids-forbidden-git-guard: ';
const PROBE = path.join(ROOT, 'features/steps/shape-fixer-revert-avoids-forbidden-git-guard__driver/accept_fixer_revert_test.go.txt');
const MIN = '20261118001611';
const COORD = 'github.com/KogenAI/kogen-coordinator';

function pin(dir) {
  const m = /github\.com\/KogenAI\/kogen-coordinator v0\.0\.0-(\d{14})-([0-9a-f]{12})/.exec(fs.readFileSync(path.join(dir, 'go.mod'), 'utf8'));
  if (!m) throw new Error('go.mod of ' + dir + ' has no kogen-coordinator pseudo-version');
  return {stamp: m[1], hash: m[2]};
}

function sh(cmd, args, cwd, ms) {
  const r = spawnSync(cmd, args, {cwd, encoding: 'utf8', timeout: ms, env: process.env, maxBuffer: 64 << 20});
  return {cmd: cmd + ' ' + args.join(' '), code: r.status, out: (r.stdout || '') + (r.stderr || ''), err: r.error};
}

function tail(s) {
  return s.length > 3000 ? '...' + s.slice(-3000) : s;
}

Given(P + 'a scratch copy of the project under the coordinator guard of e643661cb646', {timeout: 115000}, function () {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'fixer-revert-'));
  const skip = new Set(['.git', 'node_modules', path.join(ROOT, 'testdata', 'tools'), path.join(ROOT, '.kogen', 'local')]);
  fs.cpSync(ROOT, dir, {recursive: true, filter: (src) => !skip.has(path.basename(src)) && !skip.has(src)});
  fs.copyFileSync(PROBE, path.join(dir, 'shape', 'zz_accept_fixer_revert_test.go'));
  if (pin(dir).stamp < MIN) {
    const r = sh('go', ['get', COORD + '@e643661cb646'], dir, 100000);
    if (r.code !== 0) throw new Error('moving the scratch copy to the coordinator guard of e643661cb646 failed: ' + r.cmd + ' exit ' + r.code + '\n' + tail(r.out));
  }
  this.dir = dir;
});

When(P + 'I run the probe test {string}', {timeout: 115000}, function (name) {
  this.name = name;
  this.run = sh('go', ['test', './shape', '-run', '^' + name + '$', '-v', '-count=1'], this.dir, 105000);
});

Then(P + 'the probe test {string} passed', function (name) {
  const r = this.run;
  const ok = r.code === 0 && r.out.includes('--- PASS: ' + name + ' ');
  if (!ok) throw new Error('probe test ' + name + ' did not pass; ran `' + r.cmd + '` exit ' + r.code + (r.err ? ' (' + r.err.message + ')' : '') + '\n' + tail(r.out));
});

When(P + 'I read the coordinator pin of go.mod', function () {
  this.pin = pin(ROOT);
});

Then(P + 'the pinned coordinator is at least {int}-{word}', function (stamp, hash) {
  const want = String(stamp);
  if (this.pin.stamp < want) throw new Error('go.mod pins kogen-coordinator at ' + this.pin.stamp + '-' + this.pin.hash + ', older than ' + want + '-' + hash);
});

When(P + 'I read the file {string}', function (rel) {
  const file = path.join(ROOT, rel);
  if (!fs.existsSync(file)) throw new Error(rel + ' does not exist');
  this.text = fs.readFileSync(file, 'utf8');
});

Then(P + 'it declares the tests {string}', function (names) {
  const missing = names.split(',').map((s) => s.trim()).filter((n) => !new RegExp('^func ' + n + '\\(t \\*testing\\.T\\)', 'm').test(this.text));
  if (missing.length) throw new Error('the file does not declare: ' + missing.join(', '));
});
