'use strict';
// Observer control of the shape-batch-independent-tool-calls driver: `good` runs every case of
// run.js and cases.go against the correct control of controls.json, `bad` runs the overlap case
// against its "reads run one at a time" wrong control. Prints one verdict line.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');

const slug = 'shape-batch-independent-tool-calls';
const intentDir = path.join('.kogen', 'intents', slug);

function literals() {
  const feature = fs.readFileSync(path.join(intentDir, 'acceptance.feature'), 'utf8');
  const note = /README\.md answers "([^"]*)"/.exec(feature);
  const sentence = /probe descriptions contain "([^"]*)"/.exec(feature);
  if (!note || !sentence) throw new Error('acceptance.feature lost its literals');
  return { note: note[1], sentence: sentence[1] };
}

function main(mode) {
  const controls = JSON.parse(fs.readFileSync(path.join(intentDir, 'controls.json'), 'utf8'));
  const { note, sentence } = literals();
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-batch-observer-'));
  try {
    const listed = cp.execFileSync('git', ['ls-files', '-z', '--cached', '--others', '--exclude-standard'], { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 });
    for (const p of listed.split('\0')) {
      if (!p || p.startsWith('.kogen/local/') || !fs.existsSync(p) || !fs.lstatSync(p).isFile()) continue;
      fs.mkdirSync(path.join(dir, path.dirname(p)), { recursive: true });
      fs.copyFileSync(p, path.join(dir, p));
    }
    const overlay = { ...controls.correct.files };
    if (mode === 'bad') {
      const w = controls.wrong.find((x) => x.name === 'reads run one at a time');
      Object.assign(overlay, w.files);
    }
    for (const [p, s] of Object.entries(overlay)) fs.writeFileSync(path.join(dir, p), s);
    cp.execFileSync('git', ['init', '-q'], { cwd: dir });
    cp.execFileSync('git', ['add', '-A'], { cwd: dir });
    const script = [
      `const { runCase } = require(${JSON.stringify(path.join(dir, 'features/steps', slug + '__driver', 'run.js'))});`,
      mode === 'bad'
        ? `runCase('overlap', {});`
        : `runCase('overlap', {}); runCase('barriers', {}); runCase('repeat', { ACCEPT_BATCH_README_NOTE: ${JSON.stringify(note)} }); runCase('policy', { ACCEPT_BATCH_SENTENCE: ${JSON.stringify(sentence)} });`,
    ].join('\n');
    const r = cp.spawnSync(process.execPath, ['-e', script], { cwd: dir, encoding: 'utf8', timeout: 290000, maxBuffer: 64 * 1024 * 1024 });
    return r.status === 0 && !r.error;
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

if (require.main === module) {
  if (main(process.argv[2])) {
    process.stdout.write('observer accepted\n');
  } else {
    process.stdout.write('observer rejected\n');
    process.exitCode = 1;
  }
}
