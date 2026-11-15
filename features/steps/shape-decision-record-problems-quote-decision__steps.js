'use strict';
// Acceptance of shape-decision-record-problems-quote-decision. The When step compiles the shape
// package's tests with the harness of shape-decision-record-problems-quote-decision__harness.js
// (through a Go overlay), runs TestKShapeDecisionRecordQuoteAcceptance for one case and reads the
// refusal feedback the scripted shaper received. The Then steps judge that feedback.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeDecisionRecordQuoteAcceptance';
const HEADLINE = 'Sourced decisions: ';
let built;
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-decision-record-quote-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'kdrq_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-decision-record-problems-quote-decision__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/kdrq_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  built = r.status === 0 ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KDRQ_CASE: name, KDRQ_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KDRQ_CASE=${name}): shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

// problemLines are the lines of the refused submission's feedback after its headline.
function problemLines(o) {
  const lines = o.feedback.split('\n');
  assert.ok(lines[0].startsWith(HEADLINE), `case ${o.case}: the first submission was not refused by the decision record gate (first line ${JSON.stringify(lines[0])}); shape ended with exit ${o.exit}, err ${o.err}\nfeedback:\n${o.feedback}`);
  return lines.slice(1);
}

const checks = {
  exactly(o, text) {
    assert.deepEqual(problemLines(o), text.split('\n'), `case ${o.case}: the problem lines of the decision record refusal differ`);
  },
  shrunk(o, start, end) {
    const lines = problemLines(o);
    assert.equal(lines.length, 1, `case ${o.case}: want one problem line, got ${lines.length}:\n${lines.join('\n')}`);
    const line = lines[0];
    assert.equal([...line].length, 300, `case ${o.case}: the problem line has ${[...line].length} characters, want exactly 300:\n${line}`);
    assert.ok(line.startsWith(start), `case ${o.case}: the problem line does not start with ${JSON.stringify(start)}:\n${line}`);
    assert.ok(line.endsWith(end), `case ${o.case}: the problem line does not end with the whole reason ${JSON.stringify(end)}:\n${line}`);
  },
};

function controls(kind) {
  const head = 'Sourced decisions: 1 problem(s) with the decisions this submission records; nothing was recorded, asked or published.';
  const reason = 'no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under';
  const want = `- sourced decision 2 ("Round halves up"): ${reason}`;
  const seen = feedback => ({case: 'control', exit: 5, err: '', feedback});
  const crowdedReason = 'n'.repeat(200) + '.rb is not a file tracked at HEAD';
  const crowded = `- sourced decision 1 ("${'Q'.repeat(40)}"): ${crowdedReason}`;
  const good = [
    () => checks.exactly(seen(`${head}\n${want}`), want),
    () => checks.shrunk(seen(`${head}\n${crowded}`), '- sourced decision 1 ("QQQ', crowdedReason),
  ];
  const bad = [
    () => checks.exactly(seen(`${head}\n- sourced decision 2: ${reason}`), want),
    () => checks.exactly(seen(`${head}\n- sourced decision 1 ("Round halves up"): ${reason}`), want),
    () => checks.exactly(seen(`${head}\n${want.replace('under', 'under.')}`), want),
    () => checks.exactly(seen(`${want}`), want),
    () => checks.shrunk(seen(`${head}\n${crowded.slice(0, 299)}`), '- sourced decision 1 ("QQQ', crowdedReason),
    () => checks.shrunk(seen(`${head}\n${crowded.replace(crowdedReason, crowdedReason.slice(0, -2))}QQ`), '- sourced decision 1 ("QQQ', crowdedReason),
  ];
  const passes = check => { try { check(); return true; } catch { return false; } };
  const accepted = kind === 'good' ? good.every(passes) : bad.some(passes);
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'shape-decision-record-problems-quote-decision: ';
  When(P + 'the {string} case is shaped', {timeout: 540000}, function (name) { this.kdrq = shapeCase(name); });
  Then(P + 'the refusal problem lines are exactly:', function (text) { checks.exactly(this.kdrq, text); });
  Then(P + 'the only problem line has 300 characters, starts with {string} and ends with {string}', function (start, end) { checks.shrunk(this.kdrq, start, end); });
}
