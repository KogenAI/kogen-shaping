'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const {Given, When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-recheck-retained-draft-names-cause__driver/run.js');
const P = 'shape-recheck-retained-draft-names-cause: ';
let observations;
let failure;
// The observer is built and run by the first Then step, so that a tree without the diagnostic fails at a Then step, not in preparation.
function observed() {
  if (failure) throw failure;
  if (!observations) {
    try {
      observations = driver.run();
    } catch (e) {
      failure = e;
      throw e;
    }
  }
  return observations;
}
Given(P + 'the supplied recheck observer is available', function () {
  assert.ok(fs.existsSync(driver.harness), 'supplied recheck observer is missing');
});
When(P + 'the scripted rechecks run', function () {
  // the scripted rechecks are run and observed by the Then step below
});
Then(P + 'the {string} observation equals', {timeout: 110000}, function (name, expected) {
  assert.deepStrictEqual(observed()[name], JSON.parse(expected), `${name}: observation differs`);
});
