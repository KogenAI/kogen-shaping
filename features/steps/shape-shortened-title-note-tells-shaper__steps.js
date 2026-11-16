'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-shortened-title-note-tells-shaper__driver/run.js');
const P = 'shape-shortened-title-note-tells-shaper: ';
let observations;
Given(P + 'the supplied scripted Shape observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied Shape observer is missing');
});
When(P + 'the scripted shapings run', {timeout: 110000}, function () {
  observations ||= driver.run();
});
Then(P + 'the {string} output for c1 starts with', function (name, text) {
  driver.checks.startsWith(observations, name, text);
});
Then(P + 'the {string} output for c1 is not empty and lacks {string}', function (name, text) {
  driver.checks.lacks(observations, name, text);
});
Then(P + 'the {string} case recorded {int} title_shortened event\\(s), {int} progress line\\(s) and exit {int}', function (name, events, progress, exit) {
  driver.checks.counts(observations, name, events, progress, exit);
});
Then(P + 'the {string} title_shortened event has original {string} and title {string}', function (name, original, title) {
  driver.checks.event(observations, name, original, title);
});
