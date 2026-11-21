'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-time-target-is-advisory__driver/run.js');
const P = 'shape-time-target-is-advisory: ';
let observations;
Given(P + 'the supplied scripted Shape time observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied Shape time observer is missing');
});
When(P + 'the scripted shapings and time accounts run', {timeout: 110000}, function () {
  observations ||= driver.run();
});
Then(P + 'the {string} observation equals', function (name, expected) {
  assert.deepStrictEqual(observations[name], JSON.parse(expected), `${name}: observation differs`);
});
