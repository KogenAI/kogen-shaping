const { Then } = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');

function contains(path, table) {
  const text = fs.readFileSync(path, 'utf8');
  for (const [literal] of table.raw().slice(1)) {
    assert.ok(text.includes(literal), `${path} must explain ${JSON.stringify(literal)} before submission`);
  }
}
Then('k-shape-decision-lines-and-controls-named: shaper guidance contains these literals', function (table) {
  contains('shape/prompts/shaper.md', table);
});
Then('k-shape-decision-lines-and-controls-named: external guidance contains these literals', function (table) {
  contains('shape/external.go', table);
});
Then('k-shape-decision-lines-and-controls-named: controls diagnostic contains {string}', function (literal) {
  assert.ok(fs.readFileSync('shape/controls.go', 'utf8').includes(literal), `controls refusal must include ${JSON.stringify(literal)}`);
});
