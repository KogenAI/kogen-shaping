'use strict';
const assert = require('node:assert/strict');
const {Then} = require('@cucumber/cucumber');
const driver = require('./request-check-before-shape-spend__driver/run.js');
const P = 'request-check-before-shape-spend: ';
let observations;
let failure;
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
Then(P + 'the {string} observation equals', {timeout: 110000}, function (name, expected) {
  assert.deepStrictEqual(observed()[name], JSON.parse(expected), `${name}: observation differs`);
});
