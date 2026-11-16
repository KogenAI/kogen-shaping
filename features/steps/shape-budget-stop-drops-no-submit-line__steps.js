'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-budget-stop-drops-no-submit-line__driver/run.js');
const P = 'shape-budget-stop-drops-no-submit-line: ';
let observations;
Given(P + 'the supplied scripted Shape observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied Shape observer is missing');
});
When(P + 'the scripted shapings stop', {timeout:110000}, function () {
  observations ||= driver.run();
});
Then(P + 'the {string} refusal equals', function (mode, expected) {
  driver.compare(observations[mode], JSON.parse(expected), mode);
});
