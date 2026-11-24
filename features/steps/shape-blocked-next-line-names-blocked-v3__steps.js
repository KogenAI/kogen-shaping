'use strict';
const {When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-blocked-next-line-names-blocked-v3__driver/run.js');
When('shape-blocked-next-line-names-blocked-v3: the blocked report for {string} runs', {timeout: 115000}, function (mode) {
  this.bnv3 = driver.report(mode);
});
Then('shape-blocked-next-line-names-blocked-v3: the report is exactly these three lines and exits 1 without an Intent or approval', function (expected) {
  driver.judgeReport(this.bnv3, expected);
});
When('shape-blocked-next-line-names-blocked-v3: the repair journey in {string} mode runs', {timeout: 115000}, function (mode) {
  this.bnv3Repair = driver.repair(mode);
});
Then('shape-blocked-next-line-names-blocked-v3: the repair journey passes with this Next line', function (next) {
  driver.judgeRepair(this.bnv3Repair, next);
});
When('shape-blocked-next-line-names-blocked-v3: the affected existing Go tests run', {timeout: 115000}, function () {
  try { driver.affected(); this.bnv3Tests = ''; } catch (e) { this.bnv3Tests = e; }
});
Then('shape-blocked-next-line-names-blocked-v3: both existing Go expectations pass', function () {
  if (this.bnv3Tests) throw this.bnv3Tests;
});
