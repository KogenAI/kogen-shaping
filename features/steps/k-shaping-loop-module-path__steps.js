'use strict';
// Steps of K-SHAPING-LOOP-MODULE-PATH. They observe the checkout acceptance runs in (the current
// directory) through the Go tool and its files; the assertions live in the checks module.
const {When, Then} = require('@cucumber/cucumber');
const c = require('./k-shaping-loop-module-path__checks.js');

const ROOT = process.cwd();
const P = 'k-shaping-loop-module-path: ';

function none(problems) {
  if (problems.length) throw new Error(problems.join('\n'));
}

function tail(text) {
  return text.length > 8000 ? '...' + text.slice(-8000) : text;
}

When(P + "I read the checkout's go.mod", {timeout: 5 * 60 * 1000}, function () {
  this.mod = c.goMod(ROOT);
});

Then(P + 'go.mod requires exactly these modules, in this order:', function (table) {
  none(c.requireProblems(this.mod, table.hashes()));
});

Then(P + 'go.mod has no replace directive', function () {
  none(c.replaceProblems(this.mod));
});

Then(P + 'go.sum holds the module and go.mod hashes of each required version', function () {
  none(c.goSumProblems(ROOT, (this.mod.Require || []).map(r => ({module: r.Path, version: r.Version}))));
});

Then(P + 'no go.work file is in effect for the checkout', {timeout: 60 * 1000}, function () {
  none(c.goWorkProblems(ROOT));
});

When(P + "I list the build list and every package the module's packages and tests import", {timeout: 10 * 60 * 1000}, function () {
  this.build = c.listBuild(ROOT);
});

Then(P + 'no module or package path is {string} or starts with {string}', function (exact, prefix) {
  none(c.barePathProblems(this.build, exact, prefix));
});

Then(P + 'the build list holds {string}', function (line) {
  none(c.buildListProblems(this.build, line));
});

Then(P + 'the imported packages include:', function (table) {
  none(c.importProblems(this.build, table.raw().map(r => r[0])));
});

Then(P + 'no .js or .go.txt file directly under {string} holds {string}', function (dir, text) {
  none(c.harnessProblems(ROOT, dir, text));
});

Then(P + 'the file {string} has SHA-256 {string}', function (rel, want) {
  none(c.shaProblems(ROOT, rel, want));
});

When(P + 'I run in the checkout:', {timeout: 60 * 60 * 1000}, function (command) {
  this.ran = c.shell(ROOT, command, 55 * 60 * 1000);
});

Then(P + 'the command exits {int}', function (code) {
  const r = this.ran;
  if (r.status !== code) throw new Error(`${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}, want ${code}; its output:\n${tail(r.output)}`);
});

Then(P + 'the command exits {int} and prints nothing', function (code) {
  const r = this.ran;
  if (r.status !== code || r.output !== '') throw new Error(`${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}, want ${code} with no output; its output:\n${tail(r.output)}`);
});
