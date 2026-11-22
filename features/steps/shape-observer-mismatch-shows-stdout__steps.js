'use strict';
const fs = require('node:fs'), os = require('node:os'), path = require('node:path');
const { spawnSync } = require('node:child_process');
const assert = require('node:assert/strict');
const { Given, When, Then } = require('@cucumber/cucumber');

const P = 'shape-observer-mismatch-shows-stdout: ';
let compiled = null;

function build() {
  if (compiled) return compiled;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sos-'));
  const root = process.cwd();
  const overlay = path.join(dir, 'overlay.json');
  const bin = path.join(dir, 'shape.test');
  fs.writeFileSync(overlay, JSON.stringify({ Replace: { [path.join(root, 'shape/sos_acceptance_test.go')]: path.join(root, 'features/steps/shape-observer-mismatch-shows-stdout__driver/driver_test.go.txt') } }));
  const env = { ...process.env };
  for (const k of Object.keys(env)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete env[k];
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], { env, encoding: 'utf8', timeout: 100000 });
  assert.ok(b.status === 0 && !b.error, `go test -c ./shape failed: exit=${b.status} ${b.error || ''}\n${b.stdout}${b.stderr}`);
  compiled = { dir, bin, env };
  return compiled;
}

Given(P + 'the good control is {string}', function (cmd) { this.good = cmd; });
Given(P + 'the bad control is {string}', function (cmd) { this.bad = cmd; });
Given(P + 'the probe deadline is {int} milliseconds', function (ms) { this.deadline = String(ms); });

When(P + 'the observer controls are re-run', { timeout: 115000 }, function () {
  const c = build();
  const out = path.join(c.dir, 'out-' + Date.now() + '-' + Math.random().toString(16).slice(2) + '.json');
  const controls = JSON.stringify([{ command: this.good, exit: 0 }, { command: this.bad, exit: 1 }]);
  const env = { ...c.env, SOS_CONTROLS: controls, SOS_OUT: out };
  if (this.deadline) env.SOS_DEADLINE_MS = this.deadline;
  const r = spawnSync(c.bin, ['-test.run', '^TestShapeObserverStdoutAcceptance$', '-test.count=1', '-test.timeout=100s'], {
    cwd: path.join(process.cwd(), 'shape'), env, encoding: 'utf8', timeout: 105000,
  });
  assert.ok(r.status === 0 && fs.existsSync(out), `driver test exit=${r.status}: ${r.error || ''}\n${r.stdout}${r.stderr}`);
  this.items = JSON.parse(fs.readFileSync(out, 'utf8'));
});

function item(world, which) {
  const it = world.items.find((i) => i.target.endsWith(' ' + which));
  assert.ok(it, `no ${which} observer item in ${JSON.stringify(world.items)}`);
  return it;
}
const trimEnd = (s) => s.replace(/\n+$/, '');

Then(P + 'the {word} item is not ok with detail:', function (which, doc) {
  const it = item(this, which);
  assert.equal(it.ok, false, `item is ok: ${JSON.stringify(it)}`);
  assert.deepEqual(trimEnd(it.detail).split('\n'), trimEnd(doc).split('\n'), `detail ${JSON.stringify(it.detail)}`);
});
Then(P + 'the {word} item is ok with empty detail', function (which) {
  const it = item(this, which);
  assert.ok(it.ok && it.detail === '', JSON.stringify(it));
});
Then(P + 'the {word} item detail starts with {string}', function (which, prefix) {
  const it = item(this, which);
  assert.ok(!it.ok && it.detail.startsWith(prefix.replace(/\\n/g, '\n')), JSON.stringify(it.detail.slice(0, 200)));
});
Then(P + 'the {word} item stdout line is valid text of at most 4096 bytes ending with {string}', function (which, end) {
  const it = item(this, which);
  const lines = it.detail.split('\n').filter((l) => l.startsWith('stdout: '));
  assert.equal(lines.length, 1, `stdout lines in ${JSON.stringify(it.detail.slice(0, 200))}`);
  const text = lines[0].slice('stdout: '.length);
  assert.ok(!text.includes('�'), 'invalid text');
  assert.ok(Buffer.byteLength(text) <= 4096, `stdout text is ${Buffer.byteLength(text)} bytes`);
  assert.ok(text.endsWith(end), `tail ${JSON.stringify(text.slice(-40))}`);
});
Then(P + 'the {word} item detail holds none of {string} and {string}', function (which, a, b) {
  const it = item(this, which);
  for (const s of [a, b]) assert.ok(!it.detail.includes(s), `detail holds ${s}`);
});
Then(P + 'the {word} item detail holds {string} before {string}', function (which, first, second) {
  const it = item(this, which);
  assert.equal(it.ok, false, `item is ok: ${JSON.stringify(it)}`);
  const a = it.detail.indexOf(first), b = it.detail.indexOf(second);
  assert.ok(a >= 0, `detail lacks ${first}: ${JSON.stringify(it.detail)}`);
  assert.ok(b >= 0, `detail lacks ${second}: ${JSON.stringify(it.detail)}`);
  assert.ok(a < b, `${first} does not come before ${second}: ${JSON.stringify(it.detail)}`);
});
