'use strict';
const {When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-blocked-next-line-names-blocked-v2__driver/run.js');
When('shape-blocked-next-line-names-blocked-v2: the blocked report for {string} runs', {timeout: 115000}, function (mode) {
  this.bnv2 = driver.report(mode);
});
Then('shape-blocked-next-line-names-blocked-v2: the report is exactly these three lines and exits 1 without an Intent or approval', function (expected) {
  driver.judgeReport(this.bnv2, expected);
});
When('shape-blocked-next-line-names-blocked-v2: the repair journey in {string} mode runs', {timeout: 115000}, function (mode) {
  this.bnv2Repair = driver.repair(mode);
});
Then('shape-blocked-next-line-names-blocked-v2: the repair journey passes with this Next line', function (next) {
  driver.judgeRepair(this.bnv2Repair, next);
});
When('shape-blocked-next-line-names-blocked-v2: the affected existing Go tests run', {timeout: 115000}, function () {
  try { driver.affected(); this.bnv2Tests = ''; } catch (e) { this.bnv2Tests = e; }
});
Then('shape-blocked-next-line-names-blocked-v2: both existing Go expectations pass', function () {
  if (this.bnv2Tests) throw this.bnv2Tests;
});
