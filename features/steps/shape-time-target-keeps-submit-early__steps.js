'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-time-target-keeps-submit-early__driver/run.js');
const P = 'shape-time-target-keeps-submit-early: ';
let observations;
Given(P + 'the supplied scripted observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied observer is missing');
});
When(P + 'the scripted observations run', {timeout: 110000}, function () {
  observations ||= driver.run();
});
Then(P + 'the {string} observation equals', function (name, expected) {
  assert.deepStrictEqual(observations[name], JSON.parse(expected), `${name}: observation differs`);
});
