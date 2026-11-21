'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-failing-reference-gets-repair-round__driver/run.js');
const P = 'shape-failing-reference-gets-repair-round: ';
let observations;
Given(P + 'the supplied scripted Shape observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied Shape observer is missing');
});
When(P + 'the scripted shapings run', {timeout: 110000}, function () {
  observations ||= driver.run();
});
Then(P + 'the {string} observation equals', function (name, expected) {
  assert.deepStrictEqual(observations[name], JSON.parse(expected), `${name}: observation differs`);
});
