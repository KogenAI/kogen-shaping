'use strict';
const { Then } = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

function read(p) {
  return fs.readFileSync(path.join(process.cwd(), p), 'utf8');
}

Then('simplify-shaping-necessary-behavior: the shaper policy contains {string}', function (text) {
  const policy = read('shape/prompts/shaper.md');
  assert.ok(policy.includes(text), `shape/prompts/shaper.md does not contain ${JSON.stringify(text)}`);
});

Then('simplify-shaping-necessary-behavior: no shape source or prompt contains {string}', function (text) {
  const dir = path.join(process.cwd(), 'shape');
  const hits = [];
  const walk = (d) => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p);
      else if (/\.(go|md)$/.test(e.name) && fs.readFileSync(p, 'utf8').includes(text)) hits.push(p);
    }
  };
  walk(dir);
  assert.deepEqual(hits, [], `old form still present in: ${hits.join(', ')}`);
});
