const { Then } = require('@cucumber/cucumber');
const fs = require('fs');
const assert = require('assert/strict');

Then('the public README contains:', function (table) {
  const text = fs.readFileSync('README.md', 'utf8');
  for (const { text: expected } of table.hashes()) assert.ok(text.includes(expected), `README missing ${expected}`);
});
Then('the public README states {string} exactly once', function (expected) {
  assert.equal(fs.readFileSync('README.md', 'utf8').split(expected).length - 1, 1, `README must state ${expected} once`);
});
Then('the public README omits private setup and internal paths', function () {
  const text = fs.readFileSync('README.md', 'utf8');
  assert.doesNotMatch(text, /GOPRIVATE|insteadOf|private modules|private KogenAI|\/Users\/|\/home\/|work\.noindex|internal landing|delivery notes/i);
});
Then('public policy files are absent', function () {
  for (const file of ['SECURITY.md', 'CONTRIBUTING.md', 'BRANDING.md']) assert.equal(fs.existsSync(file), false, `${file} must be absent`);
});
Then('the public readiness audit contains:', function (table) {
  const text = fs.readFileSync('docs/public-readiness-audit.md', 'utf8');
  for (const { text: expected } of table.hashes()) assert.ok(text.includes(expected), `audit missing ${expected}`);
});
Then('the public readiness audit identifies historical commits', function () {
  assert.match(fs.readFileSync('docs/public-readiness-audit.md', 'utf8'), /\b[0-9a-f]{40}\b/, 'audit must identify full historical commit hashes');
});
