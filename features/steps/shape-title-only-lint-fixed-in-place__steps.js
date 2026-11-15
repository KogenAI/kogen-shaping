'use strict';
// Acceptance of shape-title-only-lint-fixed-in-place. The When step shapes one synthetic Intent with
// a scripted shaper: it compiles the shape package's tests with the harness of
// shape-title-only-lint-fixed-in-place__harness.js (through a Go overlay), runs
// TestShapeTitleOnlyLintAcceptance for the case and reads what the run showed. The Then steps judge
// that. Run this file directly with good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestShapeTitleOnlyLintAcceptance';
let built;
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shape-title-only-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_title_only_acceptance_test.go');
  fs.writeFileSync(source, require('./shape-title-only-lint-fixed-in-place__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/k_title_only_acceptance_test.go')]: source}}));
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
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({STO_CASE: name, STO_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (STO_CASE=${name}): shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

function shown(o) {
  return `case ${o.case}: exit ${o.exit}, err ${o.err}, published=${o.published}; refused=${JSON.stringify(o.refused)}; ` +
    `title_shortened=${JSON.stringify(o.shortened)}\nfirst line of intent.md: ${JSON.stringify(String(o.intent).split('\n')[0])}\n` +
    `feedback: ${JSON.stringify(o.feedback)}\nstderr (tail):\n${String(o.stderr).slice(-2000)}`;
}

function lines(text) {
  return String(text || '').split('\n');
}

const checks = {
  published(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
  },
  notPublished(o) {
    assert.equal(o.published, false, `the Intent was published\n${shown(o)}`);
    assert.notEqual(o.err, '<nil>', `shape ended without an error\n${shown(o)}`);
  },
  noRefusal(o) {
    assert.deepEqual(o.refused, [], `submissions were refused: ${JSON.stringify(o.refused)}\n${shown(o)}`);
  },
  refused(o, list) {
    assert.equal(o.refused.join(' '), list, `refused submissions (attempt:gate) are ${JSON.stringify(o.refused.join(' '))}, want ${JSON.stringify(list)}\n${shown(o)}`);
  },
  title(o, title) {
    assert.equal(lines(o.intent)[0], '# ' + title, `the stored intent.md title line is ${JSON.stringify(lines(o.intent)[0])}, want ${JSON.stringify('# ' + title)}\n${shown(o)}`);
  },
  shortened(o, n, original, title) {
    assert.equal(o.shortened.length, n, `${o.shortened.length} title_shortened event(s), want ${n}\n${shown(o)}`);
    if (n === 1 && original !== undefined) {
      assert.equal(o.shortened[0].original, original, `title_shortened original ${JSON.stringify(o.shortened[0].original)}, want ${JSON.stringify(original)}\n${shown(o)}`);
      assert.equal(o.shortened[0].title, title, `title_shortened title ${JSON.stringify(o.shortened[0].title)}, want ${JSON.stringify(title)}\n${shown(o)}`);
    }
  },
  progress(o, line, n) {
    const got = lines(o.stderr).filter(l => l === line).length;
    assert.equal(got, n, `progress printed the line ${JSON.stringify(line)} ${got} time(s), want ${n}\n${shown(o)}`);
  },
  feedbackLine(o, id, prefix) {
    assert.ok(lines(o.feedback[id]).some(l => l.startsWith(prefix)), `the feedback to submission ${id} has no line starting ${JSON.stringify(prefix)}\n${shown(o)}`);
  },
};

function controls(kind) {
  const LONG = 'Add an is-even check for integers and also reject all';
  const SHORT = 'Add an is-even check for integers and also reject';
  const FIFTY = 'Add an is-even check for integers and also rejects';
  const PL = 'kogen: shortened the title to 50 characters: ' + SHORT;
  const tl = `title: the title must be at most 50 characters, not 53`;
  const dl = 'decisions: a Decisions line lacks its source tag';
  const base = {exit: 0, err: '<nil>', published: true, stderr: '', refused: [], shortened: [], feedback: {}, intent: ''};
  const known = {
    long: {...base, case: 'long-title', intent: `# ${SHORT}\n\nSlug: is-even\n`, shortened: [{original: LONG, title: SHORT}], stderr: PL + '\nkogen: other\n'},
    other: {...base, case: 'long-title-other', exit: 1, err: 'shape: invalid_output', published: false, refused: ['1:intent', '2:intent', '3:intent'],
      shortened: [{original: LONG, title: SHORT}], feedback: {c1: tl + '\n' + dl + '\nEarlier', c2: tl + '\n' + dl + '\nEarlier'}},
    fifty: {...base, case: 'fifty-char-title', intent: `# ${FIFTY}\n\nSlug: is-even\n`},
  };
  const good = [
    s => checks.published(s.long), s => checks.noRefusal(s.long), s => checks.title(s.long, SHORT),
    s => checks.shortened(s.long, 1, LONG, SHORT), s => checks.progress(s.long, PL, 1),
    s => checks.notPublished(s.other), s => checks.refused(s.other, '1:intent 2:intent 3:intent'),
    s => checks.feedbackLine(s.other, 'c1', 'title:'), s => checks.feedbackLine(s.other, 'c1', 'decisions:'),
    s => checks.feedbackLine(s.other, 'c2', 'title:'), s => checks.feedbackLine(s.other, 'c2', 'decisions:'),
    s => checks.shortened(s.other, 1),
    s => checks.published(s.fifty), s => checks.noRefusal(s.fifty), s => checks.title(s.fifty, FIFTY),
    s => checks.shortened(s.fifty, 0), s => checks.progress(s.fifty, PL, 0),
  ];
  const changed = (name, change) => {
    const s = JSON.parse(JSON.stringify(known));
    change(s[name]);
    return s;
  };
  const bad = [
    [changed('long', o => { o.refused = ['1:intent']; }), s => checks.noRefusal(s.long)],
    [changed('long', o => { o.published = false; }), s => checks.published(s.long)],
    [changed('long', o => { o.intent = `# ${LONG}\n`; }), s => checks.title(s.long, SHORT)],
    [changed('long', o => { o.shortened = []; }), s => checks.shortened(s.long, 1, LONG, SHORT)],
    [changed('long', o => { o.shortened.push(o.shortened[0]); }), s => checks.shortened(s.long, 1, LONG, SHORT)],
    [changed('long', o => { o.stderr = ''; }), s => checks.progress(s.long, PL, 1)],
    [changed('long', o => { o.stderr = PL + ' (3 s)\n'; }), s => checks.progress(s.long, PL, 1)],
    [changed('other', o => { o.feedback.c1 = dl; }), s => checks.feedbackLine(s.other, 'c1', 'title:')],
    [changed('other', o => { o.feedback.c2 = tl; }), s => checks.feedbackLine(s.other, 'c2', 'decisions:')],
    [changed('other', o => { o.refused = ['1:intent', '2:intent']; }), s => checks.refused(s.other, '1:intent 2:intent 3:intent')],
    [changed('other', o => { o.published = true; }), s => checks.notPublished(s.other)],
    [changed('fifty', o => { o.shortened = [{original: FIFTY, title: FIFTY}]; }), s => checks.shortened(s.fifty, 0)],
    [changed('fifty', o => { o.stderr = PL; }), s => checks.progress(s.fifty, PL, 0)],
    [changed('fifty', o => { o.intent = `# ${SHORT}\n`; }), s => checks.title(s.fifty, FIFTY)],
  ];
  const passes = (seen, check) => { try { check(seen); return true; } catch { return false; } };
  const accepted = kind === 'good' ? good.every(check => passes(known, check)) : bad.some(([seen, check]) => passes(seen, check));
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'shape-title-only-lint-fixed-in-place: ';
  When(P + 'the {string} case is shaped', {timeout: 540000}, function (name) { this.sto = shapeCase(name); });
  Then(P + 'the Intent is published', function () { checks.published(this.sto); });
  Then(P + 'the Intent is not published', function () { checks.notPublished(this.sto); });
  Then(P + 'no submission is refused', function () { checks.noRefusal(this.sto); });
  Then(P + 'the refused submissions are {string}', function (list) { checks.refused(this.sto, list); });
  Then(P + 'the stored intent.md title is {string}', function (title) { checks.title(this.sto, title); });
  Then(P + '{int} title_shortened event(s) is/are recorded', function (n) { checks.shortened(this.sto, n); });
  Then(P + '1 title_shortened event is recorded with original {string} and title {string}', function (original, title) { checks.shortened(this.sto, 1, original, title); });
  Then(P + 'progress printed the line {string} {int} time(s)', function (line, n) { checks.progress(this.sto, line, n); });
  Then(P + 'the feedback to submission {string} has a line starting {string}', function (id, prefix) { checks.feedbackLine(this.sto, id, prefix); });
}
