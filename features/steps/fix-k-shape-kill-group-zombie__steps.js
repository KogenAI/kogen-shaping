'use strict';
// Steps of FIX-K-SHAPE-KILL-GROUP-ZOMBIE. They run the kill-group tests of consumes/kogen-loop in the
// checkout acceptance runs in (the current directory), or in a scratch copy of the package, and read
// the package's files; the runs and the judgements live in the harness module.
const fs = require('fs');
const path = require('path');
const {Given, When, Then} = require('@cucumber/cucumber');
const h = require('./fix-k-shape-kill-group-zombie__harness.js');

const ROOT = process.cwd();
const P = 'fix-k-shape-kill-group-zombie: ';

function none(problems) {
  if (problems.length) throw new Error(problems.join('\n'));
}

// runStep lists the package's new tests, then runs the named test and them. On Linux the run goes
// through the subreaper program in mode; elsewhere no child subreaper exists and go test runs
// directly.
function runStep(mode) {
  return function (name, count) {
    const dir = this.dir || ROOT;
    this.newTests = h.newTests(dir);
    this.ran = h.runTests(dir, [name, ...this.newTests], count, mode);
  };
}

Given(P + 'a scratch copy of the package that uses a kogen-loop whose KillGroupOnExit leaves the group running', {timeout: 10 * 60 * 1000}, function () {
  this.dir = h.scratchCopy(ROOT, {keepRunning: true});
});

When(P + 'I run {string} and the new test {int} time(s) under a child-subreaper parent that never reaps orphaned processes', {timeout: 30 * 60 * 1000}, runStep('hold'));

When(P + 'I run {string} and the new test {int} time(s) under a child-subreaper parent that reaps every orphaned process at once', {timeout: 30 * 60 * 1000}, runStep('reap'));

When(P + 'I list the tests of the package', {timeout: 10 * 60 * 1000}, function () {
  this.newTests = h.newTests(ROOT);
});

Then(P + 'the go command exits {int}', function (code) {
  none(h.exitProblems(this.ran, code));
});

Then(P + '{string} is reported as passed {int} time(s)', function (name, times) {
  none(h.reportedProblems(this.ran, name, 'passed', times));
});

Then(P + '{string} is reported as failed with {string}', function (name, text) {
  none(h.failedWithProblems(this.ran, name, text));
});

Then(P + 'on Linux, {string} took at least {int} seconds, as the parent held its killed group member as a zombie', function (name, seconds) {
  if (h.isLinux()) none(h.durationProblems(this.ran, name, {atLeast: seconds}));
});

Then(P + 'on Linux, every run of {string} took less than {int} seconds', function (name, seconds) {
  if (h.isLinux()) none(h.durationProblems(this.ran, name, {lessThan: seconds}));
});

Then(P + 'the package has exactly one top-level test that the unchanged package does not have', function () {
  none(h.oneNewTestProblems(this.newTests));
});

Then(P + 'on Linux, the new test is reported as passed, each time after at least {int} seconds', function (seconds) {
  if (h.isLinux()) none(h.newTestProblems(this.ran, this.newTests, 'passed', seconds));
});

Then(P + 'on Linux, the new test is reported as passed', function () {
  if (h.isLinux()) none(h.newTestProblems(this.ran, this.newTests, 'passed'));
});

Then(P + 'on Linux, the new test is reported as failed', function () {
  if (h.isLinux()) none(h.newTestProblems(this.ran, this.newTests, 'failed'));
});

Then(P + 'on other platforms, the new test is reported as skipped', function () {
  if (!h.isLinux()) none(h.newTestProblems(this.ran, this.newTests, 'skipped'));
});

// kogen-loop's own test with the same zombie-sensitive check: its file and its name.
const FOLLOW_UP = ['contract/sys_test.go', 'TestRunKillGroupOnExit'];

Then(P + "a Go test file in {string} names kogen-loop's own kill-group test as the follow-up", function (dir) {
  const [first, second] = FOLLOW_UP;
  const files = fs.readdirSync(path.join(ROOT, dir)).filter(f => f.endsWith('_test.go'));
  const found = files.filter(f => {
    const text = fs.readFileSync(path.join(ROOT, dir, f), 'utf8');
    return text.includes(first) && text.includes(second);
  });
  if (!found.length) throw new Error(`no _test.go file in ${dir} (${files.join(', ')}) names both ${JSON.stringify(first)} and ${JSON.stringify(second)}`);
});

Then(P + 'the inventory row of {string} in {string} lists the new test', function (use, rel) {
  none(h.oneNewTestProblems(this.newTests));
  const text = fs.readFileSync(path.join(ROOT, rel), 'utf8');
  const row = text.split('\n').find(l => l.startsWith('| `' + use + '` |'));
  if (!row) throw new Error(`${rel} has no inventory row for ${use}`);
  const tests = row.split('|')[2].split(',').map(s => s.trim());
  for (const want of this.newTests) {
    if (!tests.includes(want)) throw new Error(`${rel}: the inventory row ${JSON.stringify(row)} does not list ${want}`);
  }
});

Then(P + 'the file {string} does not contain {string}', function (rel, text) {
  if (fs.readFileSync(path.join(ROOT, rel), 'utf8').includes(text)) throw new Error(`${rel} still contains ${JSON.stringify(text)}`);
});
