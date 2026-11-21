'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-setup-failure-classified-and-retried__driver/run.js');
const P = 'shape-setup-failure-classified-and-retried: ';
let observations;
Given(P + 'the supplied scripted setup observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied setup observer is missing');
});
When(P + 'the scripted shapings run', {timeout: 110000}, function () {
  observations ||= driver.run();
});
Then(P + 'the {string} observation equals', function (name, expected) {
  assert.deepStrictEqual(observations[name], JSON.parse(expected), `${name}: observation differs`);
});
