'use strict';
const assert = require('assert/strict');
const {When, Then, AfterAll} = require('@cucumber/cucumber');
const h = require('./shape-report-all-refusing-gates__harness');
When('shape-report-all-refusing-gates: the {string} draft is shaped', function (mode) { this.mode = mode; this.result = h.observe(mode); });
Then('shape-report-all-refusing-gates: the first submission is refused by the gates {string}', function (gates) { assert.equal(this.result.gates, gates, 'gates of the first refused submission'); });
Then('shape-report-all-refusing-gates: the first feedback contains {string}', function (text) { assert.ok((this.result.fed || '').includes(text), 'first feedback lacks ' + JSON.stringify(text) + ':\n' + this.result.fed); });
Then('shape-report-all-refusing-gates: the first feedback contains the readiness refusal text unchanged', function () {
 const problems = this.result.readiness;
 assert.ok(Array.isArray(problems) && problems.length > 0, 'no readiness refusal was recorded; gates: ' + this.result.gates);
 const want = 'not ready:\n' + problems.join('\n');
 assert.ok((this.result.fed || '').includes(want), 'first feedback lacks the readiness refusal ' + JSON.stringify(want) + ':\n' + this.result.fed);
});
Then('shape-report-all-refusing-gates: the first submission counts once against the retry budget', function () { assert.equal(this.result.total, 1, 'total of the first refusal'); });
Then('shape-report-all-refusing-gates: no contract review ran', function () { assert.ok(!this.result.names.includes('contract_review'), 'events: ' + this.result.names); });
AfterAll(h.cleanup);
