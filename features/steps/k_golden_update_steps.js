'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
function passed(r, name) {
  const message = `${name}: ${r.command}; exit ${r.status}\n${r.stdout || ''}${r.stderr || ''}`;
  assert.equal(r.status, 0, message);
  assert.equal((r.stdout || '').split('\n').filter(x => x.startsWith(`--- PASS: ${name} (`)).length, 1, `Required ${name} absent, skipped or not run exactly once: ${message}`);
}
const recorded = '=== RUN   TestInitMixVerificationMatchesElixir\n=== PAUSE TestInitMixVerificationMatchesElixir\n=== CONT  TestInitMixVerificationMatchesElixir\n--- PASS: TestInitMixVerificationMatchesElixir (0.00s)\nPASS\nok  \tgithub.com/KogenAI/kogen-shaping/shape\t0.013s\n';
function changed(a, b) { assert.notEqual(b, a, 'update must replace corrupted expected values'); }
function stable(a, b) { assert.equal(b, a, 'ordinary mode and repeated update must preserve bytes'); }
if (process.argv[2] === '--control') {
  try {
    passed({status: 0, command: 'recorded go test ./shape -run ^TestInitMixVerificationMatchesElixir$ -count=1 -v', stdout: recorded}, 'TestInitMixVerificationMatchesElixir');
    changed('corrupt', 'actual'); stable('actual', 'actual');
    if (process.argv[3] === 'bad') changed('corrupt', 'corrupt');
    console.log('accepted');
  } catch (_) { console.log('rejected'); process.exitCode = 1; }
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('I exercise golden regeneration for {string} using {string}', {timeout: 240000}, function(file, name) {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'golden-cycle-'));
    try {
      fs.cpSync(process.cwd(), dir, {recursive: true, filter: src => !['.git', '.kogen', 'node_modules'].includes(path.basename(src))});
      const target = path.join(dir, file);
      const original = JSON.parse(fs.readFileSync(target, 'utf8'));
      let transcript = '';
      const run = update => {
        const args = ['test', './shape', '-run', `^${name}$`, '-count=1', '-v'];
        if (update) args.push('-update');
        const r = spawnSync('go', args, {cwd: dir, env: process.env, encoding: 'utf8', maxBuffer: 16 * 1024 * 1024});
        return {...r, command: 'go ' + args.join(' ') + '\nPrior ordinary execution:\n' + transcript};
      };
      const baseline = run(false);
      passed(baseline, name);
      transcript = baseline.stdout + baseline.stderr;
      const data = structuredClone(original);
      const mix = Array.isArray(data);
      if (mix) data[0].eligible = !data[0].eligible;
      else data[Object.keys(data)[0]][0].body_sha256 = '0'.repeat(64);
      fs.writeFileSync(target, JSON.stringify(data, null, 2) + '\n');
      const corrupt = fs.readFileSync(target, 'utf8');
      const drift = run(false);
      assert.notEqual(drift.status, 0, `${name}: ${drift.command} must reject drift\n${drift.stdout}${drift.stderr}`);
      assert.ok(drift.stdout.includes(`--- FAIL: ${name} (`), `${name}: ${drift.command} must execute the golden test, not fail compilation\n${drift.stdout}${drift.stderr}`);
      stable(corrupt, fs.readFileSync(target, 'utf8'));
      passed(run(true), name);
      const updated = fs.readFileSync(target, 'utf8');
      changed(corrupt, updated);
      const result = JSON.parse(updated);
      if (mix) assert.deepEqual(result.map(({eligible, ...input}) => input), original.map(({eligible, ...input}) => input));
      else assert.deepEqual(Object.keys(result).sort(), Object.keys(original).sort());
      passed(run(false), name);
      stable(updated, fs.readFileSync(target, 'utf8'));
      passed(run(true), name);
      stable(updated, fs.readFileSync(target, 'utf8'));
      this.goldenCycle = true;
    } finally { fs.rmSync(dir, {recursive: true, force: true}); }
  });
  Then('the golden regeneration cycle succeeds', function() { assert.equal(this.goldenCycle, true); });
}
