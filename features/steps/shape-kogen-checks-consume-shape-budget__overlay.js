const { Given, When, Then } = require('@cucumber/cucumber');
const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const slug = 'shape-kogen-checks-consume-shape-budget';
const overlay = path.join('.kogen', 'intents', slug, 'overlay_test.go.txt');
const installed = path.join('shape', 'zz_budget_overlay_test.go');

Given('shape-kogen-checks-consume-shape-budget: the Go test overlay is available', function () {
  if (!fs.existsSync(overlay)) throw new Error('missing ' + overlay);
});

When('shape-kogen-checks-consume-shape-budget: the shape test {word} is run', function (name) {
  fs.copyFileSync(overlay, installed);
  try {
    const cmd = 'go';
    const args = ['test', '-count=1', '-run', '^' + name + '$', '-v', './shape'];
    const res = spawnSync(cmd, args, { encoding: 'utf8', timeout: 110000, env: process.env });
    this.goName = name;
    this.goCommand = cmd + ' ' + args.join(' ');
    this.goStatus = res.status;
    this.goOutput = (res.stdout || '') + (res.stderr || '');
  } finally {
    fs.rmSync(installed, { force: true });
  }
});

Then('shape-kogen-checks-consume-shape-budget: the shape test passed', function () {
  const ran = this.goOutput.includes('--- PASS: ' + this.goName);
  if (this.goStatus !== 0 || !ran) {
    throw new Error('test ' + this.goName + ' did not pass (' + this.goCommand + ', exit ' + this.goStatus + ', ran and passed: ' + ran + ')\n' + this.goOutput.slice(-3000));
  }
});
