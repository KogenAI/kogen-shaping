'use strict';
const {When, Then} = require('@cucumber/cucumber');
const driver = require('./shape-blocked-next-line-names-blocked-v4__driver/run.js');
When('shape-blocked-next-line-names-blocked-v4: the blocked report for {string} runs', {timeout: 115000}, function (mode) {
  this.bnv4 = driver.report(mode);
});
Then('shape-blocked-next-line-names-blocked-v4: the report is exactly these three lines and exits 1 without an Intent or approval', function (expected) {
  driver.judgeReport(this.bnv4, expected);
});
When('shape-blocked-next-line-names-blocked-v4: the repair journey in {string} mode runs', {timeout: 115000}, function (mode) {
  this.bnv4Repair = driver.repair(mode);
});
Then('shape-blocked-next-line-names-blocked-v4: the repair journey passes with this Next line', function (next) {
  driver.judgeRepair(this.bnv4Repair, next);
});
When('shape-blocked-next-line-names-blocked-v4: the affected existing Go tests run', {timeout: 115000}, function () {
  try { driver.affected(); this.bnv4Tests = ''; } catch (e) { this.bnv4Tests = e; }
});
Then('shape-blocked-next-line-names-blocked-v4: both existing Go expectations pass', function () {
  if (this.bnv4Tests) throw this.bnv4Tests;
});
