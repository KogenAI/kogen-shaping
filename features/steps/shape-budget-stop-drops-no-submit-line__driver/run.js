'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');
const slug = 'shape-budget-stop-drops-no-submit-line';
const harness = path.join(__dirname, 'observe_test.go.txt');
const test = 'TestBudgetStopCopyObservation';

function compare(actual, expected, mode) {
  assert.deepStrictEqual(actual, expected, `${mode}: Shape refusal differs`);
}
function run(implementations = []) {
  const root = process.cwd();
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'budget-copy-'));
  try {
    const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('KOGEN_ACCEPTANCE_')));
    const replacement = {[path.join(root, 'shape/budget_copy_observation_test.go')]: harness};
    // Qualification overlays never edit the checkout, and Go compiles the final supplied harness bytes.
    for (const impl of implementations) {
      for (const [p, v] of Object.entries(impl.files)) {
        const dest = path.join(dir, p);
        fs.mkdirSync(path.dirname(dest), {recursive:true});
        if (typeof v === 'string') fs.writeFileSync(dest, v);
        else {
          if (!fs.existsSync(dest)) fs.copyFileSync(path.join(root, p), dest);
          cp.execFileSync('git', ['apply', '-'], {cwd:dir, input:v.patch, env, stdio:['pipe','pipe','pipe']});
        }
        replacement[path.join(root, p)] = dest;
      }
    }
    const overlay = path.join(dir, 'overlay.json');
    const binary = path.join(dir, 'shape.test');
    const output = path.join(dir, 'observation.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace:replacement}));
    const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', binary, './shape'];
    const built = cp.spawnSync('go', args, {env, encoding:'utf8', timeout:70000, maxBuffer:8*1024*1024});
    assert.equal(built.status, 0, `${test}: go ${args.join(' ')} failed: ${built.error || ''}\n${built.stdout || ''}${built.stderr || ''}`);
    const result = cp.spawnSync(binary, ['-test.run', '^'+test+'$', '-test.timeout=25s'], {env:{...env, BUDGET_COPY_OUT:output}, encoding:'utf8', timeout:30000, maxBuffer:8*1024*1024});
    assert.equal(result.status, 0, `${test}: ${binary} -test.run ^${test}$ failed: ${result.error || ''}\n${result.stdout || ''}${result.stderr || ''}`);
    assert.ok(fs.existsSync(output), `${test}: ${binary} -test.run ^${test}$ absent, skipped or ran zero times; no observation file: ${result.stdout || ''}${result.stderr || ''}`);
    const data = JSON.parse(fs.readFileSync(output, 'utf8'));
    assert.deepStrictEqual(Object.keys(data).sort(), ['answer','no-tool','none','submitted'], `${test}: ${binary} -test.run ^${test}$ missing scripted cases`);
    return data;
  } finally {
    fs.rmSync(dir, {recursive:true, force:true});
  }
}

function qualify(mode) {
  assert.ok(['good','bad'].includes(mode), 'qualification requires good or bad');
  const controls = JSON.parse(fs.readFileSync(`.kogen/intents/${slug}/controls.json`, 'utf8'));
  const feature = fs.readFileSync(`.kogen/intents/${slug}/acceptance.feature`, 'utf8');
  const expectations = [...feature.matchAll(/Then shape-budget-stop-drops-no-submit-line: the "([^"]+)" refusal equals\s+"""\s*([\s\S]*?)\s*"""/g)].map(m => [m[1], JSON.parse(m[2])]);
  assert.equal(expectations.length, 4, 'qualification must consume all four feature expectations');
  const data = run(mode === 'good' ? [controls.correct] : [controls.correct, controls.wrong[0]]);
  const rejected = [];
  for (const [name, expected] of expectations) {
    try { compare(data[name], expected, name); } catch (e) {
      if (mode === 'good') throw e;
      rejected.push(name);
    }
  }
  if (mode === 'bad') {
    // Reach both retained-line assertions independently; a first failure cannot mask the other.
    assert.deepStrictEqual(rejected.sort(), ['no-tool','none']);
    console.log('observer rejected both retained-line regressions');
    process.exitCode = 1;
  } else console.log('observer accepted all scripted refusals');
}
module.exports = {run, compare, harness};
if (require.main === module) qualify(process.argv[2]);
