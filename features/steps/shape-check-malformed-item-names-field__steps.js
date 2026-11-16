'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');
const assert = require('node:assert/strict');
const slug = 'shape-check-malformed-item-names-field';
const base = `.kogen/intents/${slug}/`;
const helper = `features/steps/${slug}__driver/diagnostics_test.go.txt`;
const packageName = 'github.com/KogenAI/kogen-shaping/shape';
function environment(extra = {}) {
  const env = {...process.env, ...extra};
  for (const key of Object.keys(env)) if (key.startsWith('KOGEN_ACCEPTANCE_')) delete env[key];
  return env;
}
function run(parser, cases, replacements = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'malformed-items-'));
  try {
    const test = parser === 'literal' ? 'TestMalformedLiteralItemDiagnostics' : 'TestMalformedDecisionItemDiagnostics';
    const existing = parser === 'literal' ? 'TestParseLiteralCheck' : 'TestParseDecisionCheck';
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {...replacements, [path.resolve('shape/zz_malformed_item_acceptance_test.go')]: path.resolve(helper)}}));
    const args = ['test', '-json', '-vet=off', '-overlay', overlay, '-count=1', '-timeout=30s', '-run', `^(${test}|${existing})$`, './shape'];
    const result = cp.spawnSync('go', args, {env: environment({MALFORMED_DIAGNOSTIC_CASES: JSON.stringify({[parser]: cases})}), encoding: 'utf8', timeout: 110000, maxBuffer: 8 * 1024 * 1024});
    const diagnostic = `${test} and ${existing}; go ${args.join(' ')}; outcome ${result.status}; ${result.error || ''}\n${result.stdout || ''}${result.stderr || ''}`;
    const events = (result.stdout || '').trim().split('\n').filter(Boolean).map(line => JSON.parse(line));
    // Distinguish compilation/setup failures from a known-bad observer control.
    assert.ok(events.some(e => e.Package === packageName && e.Test === test && e.Action === 'run'), `${diagnostic}\n${test} absent or ran zero times`);
    return {result, events, diagnostic, names: [test, existing, ...cases.map(c => `${test}/${c.name}`)]};
  } finally { fs.rmSync(dir, {recursive: true, force: true}); }
}
function observe(record) {
  const {result, events, diagnostic, names} = record;
  for (const name of names) {
    for (const action of ['run', 'pass']) {
      assert.equal(events.filter(e => e.Package === packageName && e.Test === name && e.Action === action).length, 1, `${name}: ${action} absent, skipped or ran other than once; ${diagnostic}`);
    }
    assert.equal(events.filter(e => e.Test === name && ['skip', 'fail'].includes(e.Action)).length, 0, `${name}: skipped or failed; ${diagnostic}`);
  }
  assert.ok(events.some(e => e.Package === packageName && !e.Test && e.Action === 'pass'), `${names.join(', ')}: package did not pass; ${diagnostic}`);
  assert.equal(result.status, 0, `${names.join(', ')}: runner unsuccessful; ${diagnostic}`);
}
function qualify(mode) {
  assert.ok(['good', 'bad'].includes(mode));
  const controls = JSON.parse(fs.readFileSync(base + 'controls.json', 'utf8'));
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'malformed-control-'));
  try {
    for (const file of Object.keys(controls.correct.files)) {
      fs.mkdirSync(path.dirname(path.join(dir, file)), {recursive: true});
      fs.copyFileSync(file, path.join(dir, file));
    }
    function apply(files) {
      for (const change of Object.values(files)) {
        const r = cp.spawnSync('git', ['apply', '-'], {cwd: dir, input: change.patch, encoding: 'utf8'});
        assert.equal(r.status, 0, `control fixture patch failed: ${r.stdout}${r.stderr}`);
      }
    }
    apply(controls.correct.files);
    if (mode === 'bad') {
      apply(controls.wrong.find(w => w.name === 'allowed statuses omitted').files);
      apply(controls.wrong.find(w => w.name === 'valid item why discarded').files);
    }
    const replacements = Object.fromEntries(Object.keys(controls.correct.files).map(file => [path.resolve(file), path.join(dir, file)]));
    const feature = fs.readFileSync(base + 'acceptance.feature', 'utf8');
    const records = [];
    for (const parser of ['literal', 'decision']) {
      const expression = new RegExp('the ' + parser + ' parser cases\\s+"""\\s*([\\s\\S]*?)\\s*"""');
      const supplied = JSON.parse(feature.match(expression)[1]);
      // Reach enum diagnostics and valid-value preservation independently of the
      // earlier missing-field failures in acceptance.
      const names = parser === 'literal' ? ['wrong_status_at_item_2', 'all_allowed_statuses'] : ['wrong_verdict_at_item_2', 'all_allowed_verdicts'];
      const cases = supplied.filter(c => names.includes(c.name));
      assert.equal(cases.length, 2);
      records.push(run(parser, cases, replacements));
    }
    let rejected = false;
    for (const record of records) {
      try { observe(record); }
      catch (error) { if (error.code !== 'ERR_ASSERTION') throw error; rejected = true; }
    }
    console.log(rejected ? 'observer rejected' : 'observer accepted');
    if (rejected) process.exitCode = 1;
  } finally { fs.rmSync(dir, {recursive: true, force: true}); }
}
if (require.main === module) {
  qualify(process.argv[2]);
} else {
  const {Given, Then} = require('@cucumber/cucumber');
  const prefix = slug + ': ';
  Given(prefix + 'the {word} parser cases', function(parser, text) {
    assert.ok(['literal', 'decision'].includes(parser));
    this.malformedParser = parser;
    this.malformedCases = JSON.parse(text);
    assert.ok(this.malformedCases.length > 0);
  });
  Then(prefix + 'named test {string} passes every supplied case', {timeout: 120000}, function(test) {
    const expected = this.malformedParser === 'literal' ? 'TestMalformedLiteralItemDiagnostics' : 'TestMalformedDecisionItemDiagnostics';
    assert.equal(test, expected, `${test}: wrong named shape test`);
    observe(run(this.malformedParser, this.malformedCases));
  });
}
