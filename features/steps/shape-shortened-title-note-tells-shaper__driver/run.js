'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');
const slug = 'shape-shortened-title-note-tells-shaper';
const harness = path.join(__dirname, 'observe_test.go.txt');
const test = 'TestShortenedTitleNoteObservation';

function run(implementations = []) {
  const root = process.cwd();
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'title-note-'));
  try {
    const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('KOGEN_ACCEPTANCE_')));
    const replacement = {[path.join(root, 'shape/title_note_observation_test.go')]: harness};
    // Qualification overlays never edit the checkout, and Go compiles the final supplied harness bytes.
    for (const impl of implementations) {
      for (const [p, v] of Object.entries(impl.files)) {
        const dest = path.join(dir, p);
        fs.mkdirSync(path.dirname(dest), {recursive: true});
        if (typeof v === 'string') fs.writeFileSync(dest, v);
        else {
          if (!fs.existsSync(dest)) fs.copyFileSync(path.join(root, p), dest);
          cp.execFileSync('git', ['apply', '-'], {cwd: dir, input: v.patch, env, stdio: ['pipe', 'pipe', 'pipe']});
        }
        replacement[path.join(root, p)] = dest;
      }
    }
    const overlay = path.join(dir, 'overlay.json');
    const binary = path.join(dir, 'shape.test');
    const output = path.join(dir, 'observation.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: replacement}));
    const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', binary, './shape'];
    const built = cp.spawnSync('go', args, {env, encoding: 'utf8', timeout: 70000, maxBuffer: 8 * 1024 * 1024});
    assert.equal(built.status, 0, `${test}: go ${args.join(' ')} failed: ${built.error || ''}\n${built.stdout || ''}${built.stderr || ''}`);
    const result = cp.spawnSync(binary, ['-test.run', '^' + test + '$', '-test.timeout=25s'], {env: {...env, SHORTEN_NOTE_OUT: output}, encoding: 'utf8', timeout: 30000, maxBuffer: 8 * 1024 * 1024});
    assert.equal(result.status, 0, `${test}: ${binary} -test.run ^${test}$ failed: ${result.error || ''}\n${result.stdout || ''}${result.stderr || ''}`);
    assert.ok(fs.existsSync(output), `${test}: ${binary} -test.run ^${test}$ absent, skipped or ran zero times; no observation file: ${result.stdout || ''}${result.stderr || ''}`);
    const data = JSON.parse(fs.readFileSync(output, 'utf8'));
    assert.deepStrictEqual(Object.keys(data).sort(), ['fifty', 'lint-only', 'refused'], `${test}: ${binary} -test.run ^${test}$ missing scripted cases`);
    return data;
  } finally {
    fs.rmSync(dir, {recursive: true, force: true});
  }
}

const checks = {
  startsWith(data, name, text) {
    assert.ok(data[name].output.startsWith(text), `${name}: the output for c1 does not start with ${JSON.stringify(text)}: ${JSON.stringify(data[name].output.slice(0, 300))}`);
  },
  lacks(data, name, text) {
    assert.ok(data[name].output !== '' && !data[name].output.includes(text), `${name}: the output for c1 is empty or contains ${JSON.stringify(text)}: ${JSON.stringify(data[name].output.slice(0, 300))}`);
  },
  counts(data, name, events, progress, exit) {
    const d = data[name];
    assert.deepStrictEqual([d.events, d.progress, d.exit], [events, progress, exit], `${name}: title_shortened events, progress lines and exit differ`);
  },
  event(data, name, original, title) {
    assert.deepStrictEqual([data[name].original, data[name].title], [original, title], `${name}: title_shortened event differs`);
  },
};

function qualify(mode) {
  assert.ok(['good', 'bad'].includes(mode), 'qualification requires good or bad');
  const controls = JSON.parse(fs.readFileSync(`.kogen/intents/${slug}/controls.json`, 'utf8'));
  const feature = fs.readFileSync(`.kogen/intents/${slug}/acceptance.feature`, 'utf8');
  const P = slug + ': ';
  const list = [];
  for (const m of feature.matchAll(new RegExp('Then ' + P + 'the "([^"]+)" output for c1 starts with\\s+"""\\n([\\s\\S]*?)\\n\\s*"""', 'g'))) {
    list.push(['starts:' + m[1], d => checks.startsWith(d, m[1], m[2].split('\n').map(l => l.trim()).join('\n'))]);
  }
  for (const m of feature.matchAll(new RegExp('Then ' + P + 'the "([^"]+)" output for c1 is not empty and lacks "([^"]+)"', 'g'))) {
    list.push(['lacks:' + m[1], d => checks.lacks(d, m[1], m[2])]);
  }
  for (const m of feature.matchAll(new RegExp('(?:Then|And) ' + P + 'the "([^"]+)" case recorded (\\d+) title_shortened event\\(s\\), (\\d+) progress line\\(s\\) and exit (\\d+)', 'g'))) {
    list.push(['counts:' + m[1], d => checks.counts(d, m[1], +m[2], +m[3], +m[4])]);
  }
  assert.deepStrictEqual(list.map(x => x[0]).sort(), ['counts:fifty', 'counts:lint-only', 'lacks:fifty', 'starts:refused'], 'qualification must consume every feature expectation');
  const rejects = data => list.filter(([, f]) => { try { f(data); return false; } catch (e) { return true; } }).map(x => x[0]);
  if (mode === 'good') {
    const data = run([controls.correct]);
    assert.deepStrictEqual(rejects(data), [], 'the correct implementation must satisfy every expectation');
    console.log('observer accepted the correct implementation');
    return;
  }
  const expected = {'No note': 'starts:refused', 'Progress twice': 'counts:lint-only', 'Note on every submission': 'lacks:fifty'};
  assert.equal(controls.wrong.length, 3);
  for (const w of controls.wrong) {
    const got = rejects(run([controls.correct, w]));
    assert.ok(got.includes(expected[w.name]), `${w.name}: expected rejection by ${expected[w.name]}, got ${JSON.stringify(got)}`);
  }
  console.log('observer rejected all three wrong implementations');
  process.exitCode = 1;
}
module.exports = {run, checks, harness};
if (require.main === module) qualify(process.argv[2]);
