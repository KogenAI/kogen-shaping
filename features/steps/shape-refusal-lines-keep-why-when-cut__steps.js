'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const slug = 'shape-refusal-lines-keep-why-when-cut';
const base = 'features/steps/' + slug;
const test = 'TestRefusalCutAcceptanceBridge';

function expand(parts, input = '') {
  return parts.map(p => p.inputPrefix !== undefined
    ? Array.from(input).slice(0, p.inputPrefix).join('')
    : p.text.repeat(p.repeat === undefined ? 1 : p.repeat)).join('');
}
function lines(specs, inputs = []) {
  return specs.flatMap((s, i) => Array(s.copies === undefined ? 1 : s.copies).fill(expand(s.parts, inputs[i] || '')));
}
function expected(cases) {
  return cases.map(c => lines(c.expected, lines(c.input)));
}
function observe(actual, cases) {
  assert.equal(actual.length, cases.length, 'cutProblems must report every feature case');
  const want = expected(cases);
  cases.forEach((c, i) => assert.deepEqual(actual[i], want[i], 'cutProblems: ' + c.name));
}
function environment(extra = {}) {
  const env = {...process.env, ...extra};
  for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
  return env;
}
function run(cases) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'refusal-cut-'));
  const context = test + ' via built ./shape test binary';
  try {
    const source = path.join(dir, 'probe_test.go');
    fs.writeFileSync(source, fs.readFileSync(base + '__driver/probe_test.go', 'utf8').replace('//go:build ignore\n\n', ''));
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/refusal_cut_acceptance_test.go')]: source}}));
    const bin = path.join(dir, 'shape.test');
    const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
    const build = spawnSync('go', args, {encoding: 'utf8', env: environment(), timeout: 90000});
    assert.equal(build.status, 0, context + ': go ' + args.join(' ') + ' failed: ' + (build.error || '') + '\n' + build.stdout + build.stderr);
    const input = path.join(dir, 'input.json');
    const output = path.join(dir, 'output.json');
    fs.writeFileSync(input, JSON.stringify(cases.map(c => lines(c.input))));
    const result = spawnSync(bin, ['-test.run=^' + test + '$', '-test.count=1', '-test.timeout=10s'], {
      cwd: path.resolve('shape'), encoding: 'utf8', timeout: 15000,
      env: environment({REFUSAL_CUT_INPUT: input, REFUSAL_CUT_OUTPUT: output})
    });
    assert.equal(result.status, 0, context + ': outcome ' + result.status + ': ' + (result.error || '') + '\n' + result.stdout + result.stderr);
    assert.ok(fs.existsSync(output), context + ': absent, skipped or zero runs; no output report; outcome ' + result.status + '\n' + result.stdout + result.stderr);
    return JSON.parse(fs.readFileSync(output, 'utf8'));
  } finally {
    fs.rmSync(dir, {recursive: true, force: true});
  }
}

if (require.main === module) {
  // Targeted observer controls, not a second acceptance run or a runner-output parser.
  const feature = fs.readFileSync('.kogen/intents/' + slug + '/acceptance.feature', 'utf8');
  const cases = JSON.parse(feature.split('"""')[1]);
  const good = expected(cases);
  observe(good, cases);
  if (process.argv[2] === 'good') {
    console.log('observer accepted fixture');
  } else if (process.argv[2] === 'bad') {
    // Reach every case assertion independently, even if production currently fails earlier cases.
    for (let i = 0; i < good.length; i++) {
      const bad = JSON.parse(JSON.stringify(good));
      bad[i][0] += '!';
      let rejected = false;
      try { observe(bad, cases); } catch (_) { rejected = true; }
      if (!rejected) {
        console.log('observer accepted a corruption');
        process.exit(0);
      }
    }
    console.log('observer rejected corruptions');
    process.exitCode = 1;
  } else {
    throw new Error('expected good or bad observer control');
  }
} else {
  const {Given, When, Then, setDefaultTimeout} = require('@cucumber/cucumber');
  setDefaultTimeout(115000);
  Given(slug + ': these refusal transformation cases', function(text) { this.refusalCases = JSON.parse(text); });
  When(slug + ': the real Go cutProblems function processes the lines', function() { this.refusalActual = run(this.refusalCases); });
  Then(slug + ': every bounded line matches its expected runes', function() { observe(this.refusalActual, this.refusalCases); });
}
