// Steps of k-contracts-shaping: kogen-shaping's contract/ suite pins what kogen uses of it and
// fails when that behaviour changes. The judging is in k-contracts-shaping__plants.js.
'use strict';
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const { When, Then, After } = require('@cucumber/cucumber');
const obs = require('./k-contracts-shaping__plants.js');

const root = path.resolve(__dirname, '..', '..');
const LONG = { timeout: 1200 * 1000 };

function goTestArgs(command) {
  const words = command.split(' ').filter(Boolean);
  assert.deepStrictEqual(words.slice(0, 2), ['go', 'test'], `k-contracts-shaping: not a go test command: ${command}`);
  return words.slice(2);
}

After(function () {
  if (this.kContractsShaping && this.kContractsShaping.dir) {
    fs.rmSync(this.kContractsShaping.dir, { recursive: true, force: true });
  }
});

When('k-contracts-shaping: {string} runs in this checkout', LONG, function (command) {
  const args = goTestArgs(command);
  this.kContractsShaping = { args, result: obs.run('go', ['test', ...args], root, root) };
});

Then('k-contracts-shaping: go test passed and ran tests in package {string}', function (pkg) {
  const { args, result } = this.kContractsShaping;
  const problems = obs.judgePassed(args, [pkg], result);
  assert.deepStrictEqual(problems, [], problems.join('\n'));
});

Then('k-contracts-shaping: go test reported package {string} passing within {int} seconds', function (pkg, seconds) {
  const { args, result } = this.kContractsShaping;
  const problem = obs.judgeWithin(args, pkg, seconds, result);
  assert.strictEqual(problem, '', problem);
});

Then('k-contracts-shaping: package {string} has test files and no other Go files', LONG, function (pkg) {
  const r = obs.run('go', ['list', '-f', '{{len .GoFiles}} {{len .CgoFiles}} {{len .TestGoFiles}} {{len .XTestGoFiles}}', pkg], root, root);
  assert.strictEqual(r.status, 0, `go list ${pkg} failed:\n${r.output}`);
  const [goFiles, cgoFiles, testFiles, xTestFiles] = r.output.trim().split(' ').map(Number);
  assert.ok(goFiles === 0 && cgoFiles === 0, `${pkg} has ${goFiles + cgoFiles} non-test Go file(s), want none`);
  assert.ok(testFiles + xTestFiles > 0, `${pkg} has no test files`);
});

Then('k-contracts-shaping: package {string} imports none of:', LONG, function (pkg, table) {
  const r = obs.run('go', ['list', '-f', '{{join .Imports "\\n"}}\n{{join .TestImports "\\n"}}\n{{join .XTestImports "\\n"}}', pkg], root, root);
  assert.strictEqual(r.status, 0, `go list ${pkg} failed:\n${r.output}`);
  const imports = new Set(r.output.split('\n').map((l) => l.trim()).filter(Boolean));
  const found = table.raw().map((row) => row[0]).filter((imp) => imports.has(imp));
  assert.deepStrictEqual(found, [], `${pkg} imports ${found.join(', ')}`);
});

Then('k-contracts-shaping: {string} names each of:', function (rel, table) {
  const file = path.join(root, rel);
  assert.ok(fs.existsSync(file), `${rel} does not exist`);
  const text = fs.readFileSync(file, 'utf8');
  const missing = table.raw().map((row) => row[0]).filter((name) => !text.includes(name));
  assert.deepStrictEqual(missing, [], `${rel} does not name: ${missing.join(', ')}`);
});

Then('k-contracts-shaping: these paths are as committed at HEAD:', LONG, function (table) {
  const paths = table.raw().map((row) => row[0]);
  const diff = obs.run('git', ['diff', '--name-only', 'HEAD', '--', ...paths], root, root);
  assert.strictEqual(diff.status, 0, `git diff failed:\n${diff.output}`);
  const untracked = obs.run('git', ['ls-files', '--others', '--exclude-standard', '--', ...paths], root, root);
  assert.strictEqual(untracked.status, 0, `git ls-files failed:\n${untracked.output}`);
  const changed = (diff.output + untracked.output).split('\n').filter(Boolean);
  assert.deepStrictEqual(changed, [], `changed since HEAD: ${changed.join(', ')}`);
});

When('k-contracts-shaping: the change {string} is planted in a scratch copy of this checkout', LONG, function (name) {
  const p = obs.plant(name);
  const dir = obs.copyCheckout(root);
  this.kContractsShaping = { dir, plant: p };
  const problem = obs.applyPlant(dir, p);
  assert.strictEqual(problem, '', problem);
});

Then('k-contracts-shaping: every package of the scratch copy builds and vets', LONG, function () {
  const { dir, plant } = this.kContractsShaping;
  const problem = obs.compiles(dir, root, plant.name);
  assert.strictEqual(problem, '', problem);
});

Then('k-contracts-shaping: {string} fails in the scratch copy because a test failed', LONG, function (command) {
  const { dir, plant } = this.kContractsShaping;
  const args = goTestArgs(command);
  const problem = obs.judgeCaught(plant.name, args, obs.run('go', ['test', ...args], dir, root));
  assert.strictEqual(problem, '', problem);
});
