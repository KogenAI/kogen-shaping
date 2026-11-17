'use strict';
// Acceptance of shape-collection-features-get-scaling-scenario. The shaper prompt is rendered the
// way shape/policy.go renders it (each {{name}} replaced by the text of shape/prompts/<name>.md),
// then checked for the collection rule and against the golden that records the prompt bytes.
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const P = 'shape-collection-features-get-scaling-scenario: ';

function render(name) {
  const file = n => fs.readFileSync(`shape/prompts/${n}.md`, 'utf8');
  return file(name)
    .split('{{feasibility-guidance}}').join(file('feasibility-guidance'))
    .split('{{readiness-acknowledgment-guidance}}').join(file('readiness-acknowledgment-guidance'));
}

// containsLine: the prompt has the wanted text as one whole line.
function containsLine(prompt, want) {
  const line = want.replace(/\n$/, '');
  assert.ok(!line.includes('\n'), 'the wanted text must be a single line');
  assert.ok(prompt.split('\n').includes(line), `the rendered shaper prompt has no line equal to:\n${line}`);
}

// matchesGolden: the byte count and SHA-256 of the prompt are the golden entry's.
function matchesGolden(prompt, golden, name) {
  const entry = golden[name];
  assert.ok(entry, `shape/testdata/prompts_golden.json has no entry "${name}"`);
  const sum = crypto.createHash('sha256').update(prompt).digest('hex');
  assert.deepEqual({bytes: Buffer.byteLength(prompt), sha256: sum}, {bytes: entry.bytes, sha256: entry.sha256},
    `the rendered ${name} prompt differs from its entry in shape/testdata/prompts_golden.json`);
}

module.exports = {containsLine, matchesGolden};

if (require.main === module) {
  // Observer controls: node <this file> --control good|bad
  const rule = '- Work over a collection. Skip this only with a `- none: <reason>` line.';
  const prompt = `first line\n${rule}\nlast line\n`;
  const sum = s => ({bytes: Buffer.byteLength(s), sha256: crypto.createHash('sha256').update(s).digest('hex')});
  const kind = process.argv[3];
  const cases = {
    good: [() => containsLine(prompt, rule), () => matchesGolden(prompt, {shaper: sum(prompt)}, 'shaper')],
    bad: [() => containsLine(`first line\nlast line\n`, rule), () => containsLine(`x ${rule} y\n`, rule),
      () => matchesGolden(prompt, {shaper: sum(prompt + 'x')}, 'shaper'), () => matchesGolden(prompt, {}, 'shaper')],
  };
  assert.ok(cases[kind], 'usage: --control good|bad');
  let outcomes = 0;
  for (const c of cases[kind]) { try { c(); } catch (_) { outcomes++; } }
  if (kind === 'good') { assert.equal(outcomes, 0); console.log('accepted'); }
  else { assert.equal(outcomes, cases.bad.length); console.log('rejected'); process.exit(1); }
} else {
  const {Given, Then} = require('@cucumber/cucumber');
  let prompt = '';
  Given(P + 'the rendered shaper prompt is loaded', function () {
    prompt = render('shaper');
    assert.ok(prompt.length > 0 && !prompt.includes('{{'), 'the rendered shaper prompt is empty or has a placeholder');
  });
  Then(P + 'the rendered shaper prompt has this line:', function (text) { containsLine(prompt, text); });
  Then(P + 'the prompts golden records the rendered shaper prompt', function () {
    matchesGolden(prompt, JSON.parse(fs.readFileSync('shape/testdata/prompts_golden.json', 'utf8')), 'shaper');
  });
}
