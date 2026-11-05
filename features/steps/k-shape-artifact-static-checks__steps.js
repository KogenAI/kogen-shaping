'use strict';
// Acceptance of k-shape-artifact-static-checks. The first When step compiles the shape package's
// tests with the harness of k-shape-artifact-static-checks__harness.js (through a Go overlay) and
// runs TestKShapeArtifactStaticChecksAcceptance once for every case, in parallel; each case shapes a
// small Go repository with a scripted shaper and the real Cucumber acceptance command. The Then
// steps judge what the runs showed. Run this file directly with good or bad to check the
// judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeArtifactStaticChecksAcceptance';
const CASES = ['repair', 'vet', 'fixer', 'unrelated', 'absent', 'renamed', 'valid'];
const HEADLINE = 'Static checks: ';
const CONFIG = ['.kogen/project.yaml', '.kogen/models.yaml'];
let ran;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// runAll compiles the shape tests with the harness and runs every case, once per process.
function runAll() {
  if (ran) return ran;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-artifact-static-checks-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_shape_artifact_static_checks_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-artifact-static-checks__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/k_shape_artifact_static_checks_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  if (b.status !== 0) {
    ran = {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${b.status} error=${b.error || ''}\n${b.stdout}${b.stderr}`};
    return ran;
  }
  const out = path.join(dir, 'out');
  fs.mkdirSync(out);
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.parallel=4', '-test.timeout=540s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({KSSC_CASES: CASES.join(','), KSSC_OUT: out}), encoding: 'utf8', timeout: 570000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ')) {
    ran = {error: `${TEST} (KSSC_CASES=${CASES.join(',')}): shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`};
    return ran;
  }
  ran = {seen: {}};
  for (const name of CASES) ran.seen[name] = JSON.parse(fs.readFileSync(path.join(out, name + '.json'), 'utf8'));
  return ran;
}

function shapeCase(name) {
  const r = runAll();
  if (r.error) throw new Error(r.error);
  return r.seen[name];
}

function shown(o) {
  const events = o.events.map(e => ({attempt: e.attempt, outcome: e.outcome, reused: e.reused, checks: (e.checks || []).map(c => [c.name, c.status, c.exit, c.paths, c.reason])}));
  const feedback = Object.keys(o.feedback).sort().map(k => `${k}:\n${o.feedback[k]}`).join('\n----\n') || '(none)';
  return `case ${o.case}: shape ended with exit ${o.exit}, error ${JSON.stringify(o.error)}; published=${o.published}; ${o.shaper} shaper request(s); ` +
    `refusals ${JSON.stringify(o.gates.map(g => [g.attempt, g.gate]))}\nstatic_checks events: ${JSON.stringify(events)}\nfeedback to the shaper:\n${feedback}\n` +
    `stderr (tail):\n${String(o.stderr).slice(-3000)}`;
}

function runOf(o, name) {
  const c = o.checks.find(c => c.name === name);
  assert.ok(c, `case ${o.case} configures no check named ${JSON.stringify(name)}`);
  return c.run;
}

function feedbackOf(o, n) {
  return String(o.feedback['c' + n] || '');
}

function gateOf(o, n) {
  const g = o.gates.find(g => g.attempt === n);
  return g ? g.gate : '';
}

// checkLines finds "- check <name>: ..." in lines and the "  | " output lines that follow it.
function checkLines(lines, name) {
  const i = lines.findIndex(l => l.startsWith(`- check ${name}: `));
  if (i < 0) return null;
  const output = [];
  for (let j = i + 1; j < lines.length && lines[j].startsWith('  | '); j++) output.push(lines[j].slice(4));
  return {line: lines[i], output};
}

function entryMatches(entry, file) {
  return entry === file || entry.startsWith(file + ':');
}

// problemLine judges one "- check <name>: exit <status>; files: <files>; command: <run>" line.
function problemLine(o, lines, name, exit, file, where) {
  const c = checkLines(lines, name);
  assert.ok(c, `${where} has no line starting ${JSON.stringify('- check ' + name + ': ')}\n${shown(o)}`);
  const m = /^- check (.+?): exit (-?\d+); files: (.*); command: (.*)$/.exec(c.line);
  assert.ok(m, `${where}: ${JSON.stringify(c.line)} is not "- check <name>: exit <status>; files: <files>; command: <run>"\n${shown(o)}`);
  assert.equal(Number(m[2]), exit, `${where}: the static check ${name} shows exit ${m[2]}, want ${exit}\n${shown(o)}`);
  assert.ok(m[3].split(', ').some(e => entryMatches(e, file)), `${where}: the static check ${name} names the files ${JSON.stringify(m[3])}, not ${JSON.stringify(file)}\n${shown(o)}`);
  assert.equal(m[4], runOf(o, name), `${where}: the static check ${name} shows the command ${JSON.stringify(m[4])}, not the configured ${JSON.stringify(runOf(o, name))}\n${shown(o)}`);
  assert.ok(c.output.length > 0, `${where}: no "  | " output line follows the line of the static check ${name}\n${shown(o)}`);
  return c;
}

function eventOf(o, n) {
  const e = o.events.find(e => e.attempt === n);
  assert.ok(e, `no static_checks event was recorded for submission ${n}\n${shown(o)}`);
  return e;
}

function checkOf(o, n, name) {
  const c = (eventOf(o, n).checks || []).find(c => c.name === name);
  assert.ok(c, `the static_checks event of submission ${n} has no check named ${JSON.stringify(name)}\n${shown(o)}`);
  return c;
}

function lastFresh(o) {
  const fresh = o.events.filter(e => e.reused === false);
  assert.ok(fresh.length > 0, `no static_checks event with reused false was recorded\n${shown(o)}`);
  return fresh[fresh.length - 1];
}

const checks = {
  refused(o, n, name, exit, file) {
    assert.equal(gateOf(o, n), 'static_checks', `submission ${n} was not refused with gate static_checks\n${shown(o)}`);
    const lines = feedbackOf(o, n).split('\n');
    assert.ok(lines.some(l => l.startsWith(HEADLINE)), `the feedback to submission ${n} has no line starting ${JSON.stringify(HEADLINE)}\n${shown(o)}`);
    problemLine(o, lines, name, exit, file, `the feedback to submission ${n}`);
  },
  output(o, n, name, text) {
    const c = checkLines(feedbackOf(o, n).split('\n'), name);
    assert.ok(c, `the feedback to submission ${n} has no line of the static check ${name}\n${shown(o)}`);
    assert.ok(c.output.join('\n').includes(text), `the output lines of the static check ${name} in the feedback to submission ${n} do not contain ${JSON.stringify(text)}: ${JSON.stringify(c.output)}\n${shown(o)}`);
  },
  notNamed(o, n, name) {
    assert.ok(!checkLines(feedbackOf(o, n).split('\n'), name), `the feedback to submission ${n} names the static check ${name}\n${shown(o)}`);
  },
  unchangedSince(o, n, m) {
    const want = `Unchanged: this package is identical to the one refused at attempt ${m}`;
    assert.ok(feedbackOf(o, n).includes(want), `the feedback to submission ${n} does not say ${JSON.stringify(want)}\n${shown(o)}`);
  },
  passes(o, n, a, b, outcome) {
    assert.notEqual(gateOf(o, n), 'static_checks', `submission ${n} was refused with gate static_checks\n${shown(o)}`);
    const e = eventOf(o, n);
    assert.equal(e.outcome, outcome, `the static_checks event of submission ${n} has outcome ${JSON.stringify(e.outcome)}, want ${JSON.stringify(outcome)}\n${shown(o)}`);
    for (const name of [a, b]) {
      assert.equal(checkOf(o, n, name).status, 'passed', `the static check ${name} of submission ${n} is ${JSON.stringify(checkOf(o, n, name).status)}, want "passed"\n${shown(o)}`);
    }
  },
  observations(o, n, text) {
    assert.equal(gateOf(o, n), '', `submission ${n} was refused with gate ${gateOf(o, n)}\n${shown(o)}`);
    const f = feedbackOf(o, n);
    assert.ok(f.includes('Readiness observations') && f.includes(text), `the feedback to submission ${n} is not readiness observations containing ${JSON.stringify(text)}\n${shown(o)}`);
  },
  publishedAfter(o, n) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.equal(o.shaper, n, `the Intent was published after ${o.shaper} shaper request(s), want ${n} submissions\n${shown(o)}`);
  },
  noRefusal(o) {
    assert.equal(o.gates.length, 0, `submissions were refused: ${JSON.stringify(o.gates.map(g => [g.attempt, g.gate]))}\n${shown(o)}`);
  },
  notRefused(o, n) {
    eventOf(o, n);
    assert.notEqual(gateOf(o, n), 'static_checks', `submission ${n} was refused with gate static_checks\n${shown(o)}`);
  },
  status(o, n, name, status) {
    const c = checkOf(o, n, name);
    assert.equal(c.status, status, `the static check ${name} of submission ${n} is ${JSON.stringify(c.status)}, want ${JSON.stringify(status)}\n${shown(o)}`);
  },
  naming(o, n, name, status, file) {
    checks.status(o, n, name, status);
    const c = checkOf(o, n, name);
    assert.ok((c.paths || []).some(p => entryMatches(p, file)), `the static check ${name} of submission ${n} names ${JSON.stringify(c.paths)}, not ${JSON.stringify(file)}\n${shown(o)}`);
  },
  notNaming(o, n, name, file) {
    const c = checkOf(o, n, name);
    assert.ok(!(c.paths || []).some(p => entryMatches(p, file)), `the static check ${name} of submission ${n} names ${JSON.stringify(file)}\n${shown(o)}`);
  },
  configured(o, n, name) {
    assert.equal(checkOf(o, n, name).run, runOf(o, name), `the static check ${name} of submission ${n} recorded the command ${JSON.stringify(checkOf(o, n, name).run)}, not the configured one\n${shown(o)}`);
  },
  reason(o, n, name, status, reason) {
    checks.status(o, n, name, status);
    const c = checkOf(o, n, name);
    assert.ok(String(c.reason || '').includes(reason), `the static check ${name} of submission ${n} gives the reason ${JSON.stringify(c.reason)}, not ${JSON.stringify(reason)}\n${shown(o)}`);
  },
  outcome(o, n, outcome) {
    assert.equal(eventOf(o, n).outcome, outcome, `the static_checks event of submission ${n} has outcome ${JSON.stringify(eventOf(o, n).outcome)}, want ${JSON.stringify(outcome)}\n${shown(o)}`);
  },
  stderrHas(o, text) {
    assert.ok(String(o.stderr).includes(text), `stderr does not contain ${JSON.stringify(text)}:\n${o.stderr}`);
  },
  publishedFile(o, p, text) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.equal(o.after[p], text.replaceAll('<TAB>', '\t') + '\n', `the published ${p} is not the expected bytes\n${shown(o)}`);
  },
  unchanged(o, p) {
    assert.ok(typeof o.before[p] === 'string', `${p} did not exist before shaping`);
    assert.equal(o.after[p], o.before[p], `${p} changed during shaping\n${shown(o)}`);
  },
  reads(o, p, text) {
    assert.equal(o.after[p], text + '\n', `${p} does not read ${JSON.stringify(text)}: ${JSON.stringify(o.after[p])}`);
  },
  freshCopy(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    const e = lastFresh(o);
    const root = String(o.root).replace(/\/+$/, '');
    assert.ok(typeof e.dir === 'string' && path.isAbsolute(e.dir), `the last fresh static checks recorded no absolute directory: ${JSON.stringify(e.dir)}`);
    assert.ok(e.dir !== root && !e.dir.startsWith(root + '/'), `the last fresh static checks ran in ${e.dir}, inside the shaping checkout ${root}`);
    const want = Object.fromEntries(Object.entries(o.package.files).filter(([p]) => !CONFIG.includes(p)));
    const got = Object.fromEntries((e.files || []).map(f => [f.path, f.sha256]));
    assert.deepEqual(got, want, `the last fresh static checks did not record the published package files' bytes\n${shown(o)}`);
    assert.equal(e.sha256, o.package.sha256, `the last fresh static checks recorded the package hash ${e.sha256}, not the published ${o.package.sha256}`);
  },
  record(o, outcome) {
    const r = o.record;
    assert.ok(r && typeof r === 'object', `no static check record .kogen/local/static-checks/static-proof.json was written\n${shown(o)}`);
    assert.equal(r.outcome, outcome, `the static check record has outcome ${JSON.stringify(r.outcome)}, want ${JSON.stringify(outcome)}`);
    assert.equal(r.sha256, o.package.sha256, `the static check record names the package hash ${r.sha256}, not the published ${o.package.sha256}`);
    assert.equal(r.tree, lastFresh(o).tree, `the static check record names the tree ${r.tree}, not the last checked ${lastFresh(o).tree}`);
  },
  approved(o) {
    const a = o.approvals[0];
    assert.ok(a, `the published package was not approved: no approval ran\n${shown(o)}`);
    assert.ok(a.exit === 0 && a.approved === true, `approving the published package ended with exit ${a.exit}, error ${JSON.stringify(a.error)}, approved=${a.approved}\n${a.stderr}`);
  },
  oldHashRefused(o) {
    const a = o.approvals[0];
    assert.ok(a, `no approval of the published hash after the change ran\n${shown(o)}`);
    assert.ok(a.exit !== 0 && a.error && a.error.reason === 'hash_mismatch' && a.approved === false,
      `approving the published hash after the helper changed ended with exit ${a.exit}, error ${JSON.stringify(a.error)}, approved=${a.approved}`);
  },
  changedRefused(o, name, file) {
    const a = o.approvals[1];
    assert.ok(a, `no approval of the changed package ran\n${shown(o)}`);
    assert.ok(a.exit !== 0 && a.approved === false, `approving the changed package ended with exit ${a.exit}, approved=${a.approved}, error ${JSON.stringify(a.error)}\n${a.stderr}`);
    assert.ok(a.error && a.error.class === 'intent' && a.error.reason === 'static_checks', `approving the changed package was refused with ${JSON.stringify(a.error)}, not intent/static_checks`);
    problemLine(o, a.error.lines || [], name, 1, file, 'the refusal of the changed package');
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const D = 'features/steps/static-proof__driver/main.go';
  const C = 'features/steps/static-proof__counter/main.go';
  const LAYOUT = 'test -z "$(gofmt -l count features/steps)"';
  const VET = 'go vet ./...';
  const UNFORMATTED = 'package main\nimport("fmt";"example.test/staticshape/count")\nfunc main(){fmt.Println(count.Count([]string{"a","b"}))}\n';
  const FORMATTED = 'package main\n\nimport (\n\t"fmt"\n\n\t"example.test/staticshape/count"\n)\n\nfunc main() {\n\tfmt.Println(count.Count([]string{"a", "b"}))\n}\n';
  const GOFMT = 'package main\n\nimport (\n\t"example.test/staticshape/count"\n\t"fmt"\n)\n\nfunc main() { fmt.Println(count.Count([]string{"a", "b"})) }\n';
  const FORMATTED_DOC = 'package main\n\nimport (\n<TAB>"fmt"\n\n<TAB>"example.test/staticshape/count"\n)\n\nfunc main() {\n<TAB>fmt.Println(count.Count([]string{"a", "b"}))\n}';
  const GOFMT_DOC = 'package main\n\nimport (\n<TAB>"example.test/staticshape/count"\n<TAB>"fmt"\n)\n\nfunc main() { fmt.Println(count.Count([]string{"a", "b"})) }';
  const UNRELATED = 'package count;var Existing=1\n';
  const VETMSG = 'fmt.Printf format %d has arg "not-a-number" of wrong type string';
  const RED = "'0' !== '2'";
  const head = n => `Static checks: the shaped files fail ${n} configured static check(s); nothing was published. Each check ran unchanged on the final submitted bytes in a fresh copy of the candidate; repair the named files so that every check passes. Resubmitting this package unchanged is refused again.`;
  const line = (name, exit, files, run) => `- check ${name}: exit ${exit}; files: ${files}; command: ${run}`;
  const unchangedText = '\nUnchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.';
  const obs = 'Inspect whether fixtures, inputs and commands reached the intended behavior.\n\nReadiness observations\nChecked baseline: p\nScenario: Observe the count (acceptance.feature line 2)\nAssertionError [ERR_ASSERTION]: Expected values to be strictly equal:\n\n' + RED + '\n';
  const root = '/work/t/TestKShape/001';
  const files = helper => ({'.kogen/intents/static-proof/intent.md': 'f1', '.kogen/intents/static-proof/acceptance.feature': 'f2', 'features/steps/static-proof__steps.js': 'f3', [helper]: 'f4', '.kogen/project.yaml': 'f5'});
  const eventFiles = helper => Object.entries(files(helper)).filter(([p]) => !CONFIG.includes(p)).map(([path, sha256]) => ({path, sha256}));
  const chk = (name, run, status, extra = {}) => ({name, run, status, exit: status === 'passed' ? 0 : 1, paths: [], output: '', trace: '', ...extra});
  const ev = (attempt, outcome, list, helper, extra = {}) => ({event: 'static_checks', attempt, outcome, reused: false, dir: '/work/t/kogen-static-1', head: 'h', tree: 't' + attempt, sha256: 'pkg', files: eventFiles(helper), checks: list, ...extra});
  const reused = (e, attempt) => ({...e, attempt, reused: true});
  const before = {'go.mod': 'module example.test/staticshape\n\ngo 1.27\n', 'count/count.go': 'package count\n\nfunc Count(items []string) int { return 0 }\n', 'count/unrelated.go': null, '.kogen/project.yaml': 'version: 1\n'};
  const base = (name, helper, cfg) => ({case: name, root, helper, exit: 0, error: null, stdout: 'Intent: x\nNext: kogen intent approve static-proof\n', stderr: '', published: true,
    checks: [{name: cfg[0], run: LAYOUT}, {name: cfg[1], run: VET}], before, after: {...before}, package: {sha256: 'pkg', files: files(helper)}, approvals: []});
  const refusal = (n, lines) => head(n) + '\n' + lines.join('\n');
  const layoutLines = (name, file) => [line(name, 1, file, LAYOUT), '  | (no output)'];
  const vetLines = (name, at) => [line(name, 1, at, VET), '  | ' + at + ': ' + VETMSG];

  const e1 = ev(1, 'refused', [chk('layout', LAYOUT, 'attributable', {paths: [D]}), chk('analysis', VET, 'passed')], D,
    {sha256: 'pkg0', files: eventFiles(D).map(f => (f.path === D ? {...f, sha256: 'f0'} : f))});
  const e4 = ev(4, 'checked', [chk('layout', LAYOUT, 'passed'), chk('analysis', VET, 'passed')], D);
  const repair = {...base('repair', D, ['layout', 'analysis']), shaper: 5,
    feedback: {c1: refusal(1, layoutLines('layout', D)), c2: refusal(1, layoutLines('layout', D)) + unchangedText, c3: refusal(1, layoutLines('layout', D)) + unchangedText, c4: obs},
    gates: [1, 2, 3].map(attempt => ({attempt, gate: 'static_checks', problems: layoutLines('layout', D)})),
    events: [e1, reused(e1, 2), reused(e1, 3), e4, reused(e4, 5)], record: {version: 1, outcome: 'checked', sha256: 'pkg', tree: 't4'},
    after: {...before, [D]: FORMATTED}, approvals: [{hash: 'pkg', exit: 0, error: null, stdout: 'approved static-proof pkg\n', stderr: '', approved: true}]};
  const v1 = ev(1, 'refused', [chk('layout', LAYOUT, 'passed'), chk('analysis', VET, 'attributable', {paths: [D + ':14:14']})], D);
  const v2 = ev(2, 'checked', [chk('layout', LAYOUT, 'passed'), chk('analysis', VET, 'passed')], D);
  const vet = {...base('vet', D, ['layout', 'analysis']), shaper: 3, feedback: {c1: refusal(1, vetLines('analysis', D + ':14:14')), c2: obs},
    gates: [{attempt: 1, gate: 'static_checks', problems: vetLines('analysis', D + ':14:14')}], events: [v1, v2, reused(v2, 3)], record: {version: 1, outcome: 'checked', sha256: 'pkg', tree: 't2'},
    after: {...before, [D]: FORMATTED + '\nfunc diagnostic() {\n\tfmt.Printf("%s", "not-a-number")\n}\n'}};
  const f1 = ev(1, 'checked', [chk('layout', LAYOUT, 'passed'), chk('analysis', VET, 'passed')], D);
  const fixer = {...base('fixer', D, ['layout', 'analysis']), shaper: 2, feedback: {c1: obs}, gates: [], events: [f1, reused(f1, 2)], record: {version: 1, outcome: 'checked', sha256: 'pkg', tree: 't1'},
    after: {...before, [D]: GOFMT},
    approvals: [{hash: 'pkg', exit: 1, error: {class: 'intent', reason: 'hash_mismatch', detail: 'x', lines: null}, approved: false, stderr: ''},
      {hash: 'new', exit: 1, error: {class: 'intent', reason: 'static_checks', detail: "the Intent's files fail configured static checks", lines: vetLines('analysis', D + ':11:14')}, approved: false, stderr: ''}]};
  const u1 = ev(1, 'unknown', [chk('layout', LAYOUT, 'preexisting', {paths: ['count/unrelated.go']}), chk('analysis', VET, 'passed')], D);
  const unrelatedBefore = {...before, 'count/unrelated.go': UNRELATED};
  const unrelated = {...base('unrelated', D, ['layout', 'analysis']), shaper: 2, feedback: {c1: obs}, gates: [], events: [u1, reused(u1, 2)], record: {version: 1, outcome: 'unknown', sha256: 'pkg', tree: 't1'},
    before: unrelatedBefore, after: {...unrelatedBefore, [D]: FORMATTED},
    stderr: 'kogen: running static checks on the shaped files in a fresh copy: layout, analysis (0 s)\nkogen: static check layout failed (exit 1) without the shaped files too: pre-existing failure in count/unrelated.go; the shaped files are not established by this check (1 s)\nkogen: static check analysis passed (1 s)\n'};
  const a1 = ev(1, 'unknown', [chk('layout', LAYOUT, 'unavailable', {exit: 0, reason: 'gofmt not found', output: '/bin/sh: gofmt: command not found'}), chk('analysis', VET, 'passed')], D);
  const absent = {...base('absent', D, ['layout', 'analysis']), shaper: 2, feedback: {c1: obs}, gates: [], events: [a1, reused(a1, 2)], record: {version: 1, outcome: 'unknown', sha256: 'pkg', tree: 't1'},
    before: unrelatedBefore, after: {...unrelatedBefore, [D]: FORMATTED},
    stderr: 'kogen: static check layout unavailable: gofmt not found; the shaped files are not established by this check (0 s)\nkogen: static check analysis passed (1 s)\n'};
  const r1 = ev(1, 'refused', [chk('tidy', LAYOUT, 'attributable', {paths: [C]}), chk('inspect', VET, 'attributable', {paths: [C + ':4:31']})], C);
  const r2 = ev(2, 'checked', [chk('tidy', LAYOUT, 'passed'), chk('inspect', VET, 'passed')], C);
  const renamedLines = [...layoutLines('tidy', C), ...vetLines('inspect', C + ':4:31')];
  const renamed = {...base('renamed', C, ['tidy', 'inspect']), shaper: 3, feedback: {c1: refusal(2, renamedLines), c2: obs}, gates: [{attempt: 1, gate: 'static_checks', problems: renamedLines}],
    events: [r1, r2, reused(r2, 3)], record: {version: 1, outcome: 'checked', sha256: 'pkg', tree: 't2'}, after: {...before, [C]: FORMATTED + '\nfunc diagnostic() {\n\tfmt.Printf("%s", "not-a-number")\n}\n'}};
  const w1 = ev(1, 'checked', [chk('tidy', LAYOUT, 'passed'), chk('inspect', VET, 'passed')], C);
  const valid = {...base('valid', C, ['tidy', 'inspect']), shaper: 2, feedback: {c1: obs}, gates: [], events: [w1, reused(w1, 2)], record: {version: 1, outcome: 'checked', sha256: 'pkg', tree: 't1'},
    after: {...before, [C]: FORMATTED}};
  const known = {repair, vet, fixer, unrelated, absent, renamed, valid};

  const good = [
    s => checks.refused(s.repair, 1, 'layout', 1, D),
    s => checks.output(s.repair, 1, 'layout', '(no output)'),
    s => checks.notNamed(s.repair, 1, 'analysis'),
    s => checks.refused(s.repair, 2, 'layout', 1, D),
    s => checks.unchangedSince(s.repair, 2, 1),
    s => checks.refused(s.repair, 3, 'layout', 1, D),
    s => checks.passes(s.repair, 4, 'layout', 'analysis', 'checked'),
    s => checks.observations(s.repair, 4, RED),
    s => checks.publishedAfter(s.repair, 5),
    s => checks.publishedFile(s.repair, D, FORMATTED_DOC),
    s => checks.unchanged(s.repair, 'count/count.go'),
    s => checks.freshCopy(s.repair),
    s => checks.record(s.repair, 'checked'),
    s => checks.approved(s.repair),
    s => checks.refused(s.vet, 1, 'analysis', 1, D + ':14'),
    s => checks.output(s.vet, 1, 'analysis', VETMSG),
    s => checks.notNamed(s.vet, 1, 'layout'),
    s => checks.passes(s.vet, 2, 'layout', 'analysis', 'checked'),
    s => checks.observations(s.vet, 2, RED),
    s => checks.publishedAfter(s.vet, 3),
    s => checks.passes(s.fixer, 1, 'layout', 'analysis', 'checked'),
    s => checks.publishedAfter(s.fixer, 2),
    s => checks.publishedFile(s.fixer, D, GOFMT_DOC),
    s => checks.unchanged(s.fixer, 'go.mod'),
    s => checks.unchanged(s.fixer, 'count/count.go'),
    s => checks.unchanged(s.fixer, '.kogen/project.yaml'),
    s => checks.freshCopy(s.fixer),
    s => checks.record(s.fixer, 'checked'),
    s => checks.oldHashRefused(s.fixer),
    s => checks.changedRefused(s.fixer, 'analysis', D + ':11'),
    s => checks.notRefused(s.unrelated, 1),
    s => checks.naming(s.unrelated, 1, 'layout', 'preexisting', 'count/unrelated.go'),
    s => checks.notNaming(s.unrelated, 1, 'layout', D),
    s => checks.configured(s.unrelated, 1, 'layout'),
    s => checks.status(s.unrelated, 1, 'analysis', 'passed'),
    s => checks.outcome(s.unrelated, 1, 'unknown'),
    s => checks.stderrHas(s.unrelated, 'kogen: static check layout failed (exit 1) without the shaped files too: pre-existing failure in count/unrelated.go; the shaped files are not established by this check'),
    s => checks.publishedAfter(s.unrelated, 2),
    s => checks.unchanged(s.unrelated, 'count/unrelated.go'),
    s => checks.reads(s.unrelated, 'count/unrelated.go', 'package count;var Existing=1'),
    s => checks.reason(s.absent, 1, 'layout', 'unavailable', 'gofmt not found'),
    s => checks.status(s.absent, 1, 'analysis', 'passed'),
    s => checks.outcome(s.absent, 1, 'unknown'),
    s => checks.stderrHas(s.absent, 'kogen: static check layout unavailable: gofmt not found; the shaped files are not established by this check'),
    s => checks.notRefused(s.absent, 1),
    s => checks.publishedAfter(s.absent, 2),
    s => checks.refused(s.renamed, 1, 'tidy', 1, C),
    s => checks.refused(s.renamed, 1, 'inspect', 1, C + ':4'),
    s => checks.passes(s.renamed, 2, 'tidy', 'inspect', 'checked'),
    s => checks.publishedAfter(s.renamed, 3),
    s => checks.passes(s.valid, 1, 'tidy', 'inspect', 'checked'),
    s => checks.observations(s.valid, 1, RED),
    s => checks.noRefusal(s.valid),
    s => checks.publishedAfter(s.valid, 2),
  ];
  const changed = (name, change) => {
    const s = JSON.parse(JSON.stringify(known));
    change(s[name]);
    return s;
  };
  // unchangedTree is what the tree without the change shows: the helper path is refused by the
  // submission validation, nothing is published and no static check runs.
  const unchangedTree = o => {
    const refusal = `invalid submission:\n${o.helper}: new step files must be named features/steps/static-proof__<suffix>.js (nonempty suffix); prefix step sentences and definitions with static-proof: \nNo proposals from this submission were retained; earlier proposals are unchanged.`;
    o.exit = 4; o.error = {class: 'shape', reason: 'provider_failed', detail: 'shaping stopped on a provider error', lines: null}; o.published = false; o.stdout = '';
    o.shaper = 3; o.feedback = {c1: refusal, c2: 'invalid submission:\nintent.md does not exist'}; o.gates = [{attempt: 1, gate: 'validation', problems: []}, {attempt: 2, gate: 'intent', problems: []}];
    o.events = []; o.record = null; o.approvals = []; o.after = {...o.before, [o.helper]: null};
    o.stderr = 'kogen: checking the acceptance runner\nkogen: shaping (request 1, 0 s)\n';
  };
  // fixerToday is the fixer case without the static gate: published as today, and the changed
  // package approved.
  const fixerToday = o => {
    o.events = []; o.record = null;
    o.approvals[1] = {hash: 'new', exit: 0, error: null, stdout: 'approved static-proof new\n', stderr: '', approved: true};
  };
  const bad = [];
  for (const [name, check] of [
    ['repair', s => checks.refused(s.repair, 1, 'layout', 1, D)], ['repair', s => checks.passes(s.repair, 4, 'layout', 'analysis', 'checked')],
    ['repair', s => checks.publishedAfter(s.repair, 5)], ['repair', s => checks.freshCopy(s.repair)], ['repair', s => checks.approved(s.repair)],
    ['vet', s => checks.refused(s.vet, 1, 'analysis', 1, D + ':14')], ['vet', s => checks.publishedAfter(s.vet, 3)],
    ['fixer', s => checks.passes(s.fixer, 1, 'layout', 'analysis', 'checked')], ['fixer', s => checks.changedRefused(s.fixer, 'analysis', D + ':11')],
    ['fixer', s => checks.record(s.fixer, 'checked')], ['fixer', s => checks.publishedFile(s.fixer, D, GOFMT_DOC)],
    ['unrelated', s => checks.naming(s.unrelated, 1, 'layout', 'preexisting', 'count/unrelated.go')], ['unrelated', s => checks.outcome(s.unrelated, 1, 'unknown')],
    ['unrelated', s => checks.notRefused(s.unrelated, 1)], ['unrelated', s => checks.stderrHas(s.unrelated, 'kogen: static check layout failed (exit 1) without the shaped files too: pre-existing failure in count/unrelated.go')],
    ['absent', s => checks.reason(s.absent, 1, 'layout', 'unavailable', 'gofmt not found')], ['absent', s => checks.outcome(s.absent, 1, 'unknown')],
    ['renamed', s => checks.refused(s.renamed, 1, 'tidy', 1, C)], ['renamed', s => checks.refused(s.renamed, 1, 'inspect', 1, C + ':4')],
    ['valid', s => checks.passes(s.valid, 1, 'tidy', 'inspect', 'checked')], ['valid', s => checks.publishedAfter(s.valid, 2)],
  ]) bad.push([changed(name, unchangedTree), check]);
  const corrupt = (name, change, check) => bad.push([changed(name, change), check]);
  const setLine = (o, c, from, to) => { o.feedback[c] = o.feedback[c].replace(from, to); };
  corrupt('fixer', fixerToday, s => checks.changedRefused(s.fixer, 'analysis', D + ':11'));
  corrupt('repair', o => setLine(o, 'c1', '- check layout: exit 1;', '- check layout: exit 0;'), s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => setLine(o, 'c1', 'files: ' + D, 'files: features/steps/static-proof__steps.js'), s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => setLine(o, 'c1', 'files: ' + D, 'files: (not named by the check)'), s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => setLine(o, 'c1', 'command: ' + LAYOUT, 'command: gofmt -l ' + D), s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => setLine(o, 'c1', '\n  | (no output)', ''), s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => setLine(o, 'c1', '\n  | (no output)', '\n  | layout failed'), s => checks.output(s.repair, 1, 'layout', '(no output)'));
  corrupt('repair', o => setLine(o, 'c1', HEADLINE, 'Readiness: '), s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => { o.gates[0].gate = 'readiness'; }, s => checks.refused(s.repair, 1, 'layout', 1, D));
  corrupt('repair', o => { o.feedback.c1 += '\n' + line('analysis', 1, D, VET); }, s => checks.notNamed(s.repair, 1, 'analysis'));
  corrupt('repair', o => setLine(o, 'c2', 'Unchanged: ', 'Changed: '), s => checks.unchangedSince(s.repair, 2, 1));
  corrupt('repair', o => { o.gates[2].gate = 'readiness'; o.feedback.c3 = obs; }, s => checks.refused(s.repair, 3, 'layout', 1, D));
  corrupt('repair', o => { o.events[3].outcome = 'unknown'; }, s => checks.passes(s.repair, 4, 'layout', 'analysis', 'checked'));
  corrupt('repair', o => { o.events[3].checks[0].status = 'unavailable'; }, s => checks.passes(s.repair, 4, 'layout', 'analysis', 'checked'));
  corrupt('repair', o => { o.events.splice(3, 1); }, s => checks.passes(s.repair, 4, 'layout', 'analysis', 'checked'));
  corrupt('repair', o => { o.feedback.c4 = o.feedback.c4.replace(RED, "'2' !== '2'"); }, s => checks.observations(s.repair, 4, RED));
  corrupt('repair', o => { o.gates.push({attempt: 4, gate: 'static_checks', problems: []}); }, s => checks.observations(s.repair, 4, RED));
  corrupt('repair', o => { o.shaper = 6; }, s => checks.publishedAfter(s.repair, 5));
  corrupt('repair', o => { o.published = false; }, s => checks.publishedAfter(s.repair, 5));
  corrupt('repair', o => { o.after[D] = UNFORMATTED; }, s => checks.publishedFile(s.repair, D, FORMATTED_DOC));
  corrupt('repair', o => { o.after['count/count.go'] = 'package count\n\nfunc Count(items []string) int { return len(items) }\n'; }, s => checks.unchanged(s.repair, 'count/count.go'));
  corrupt('repair', o => { o.events[3].dir = root + '/.kogen/local/static'; }, s => checks.freshCopy(s.repair));
  corrupt('repair', o => { o.events[3].dir = root; }, s => checks.freshCopy(s.repair));
  corrupt('repair', o => { o.events[3].files[3].sha256 = 'other'; }, s => checks.freshCopy(s.repair));
  corrupt('repair', o => { o.events[3].files.pop(); }, s => checks.freshCopy(s.repair));
  corrupt('repair', o => { o.events[3].sha256 = 'other'; }, s => checks.freshCopy(s.repair));
  corrupt('repair', o => { o.events[3].reused = true; o.events[4].reused = true; }, s => checks.freshCopy(s.repair));
  corrupt('repair', o => { o.record.sha256 = 'other'; }, s => checks.record(s.repair, 'checked'));
  corrupt('repair', o => { o.record.outcome = 'unknown'; }, s => checks.record(s.repair, 'checked'));
  corrupt('repair', o => { o.record.tree = 't1'; }, s => checks.record(s.repair, 'checked'));
  corrupt('repair', o => { o.record = null; }, s => checks.record(s.repair, 'checked'));
  corrupt('repair', o => { o.approvals[0] = {...o.approvals[0], exit: 1, approved: false, error: {class: 'intent', reason: 'static_checks', lines: []}}; }, s => checks.approved(s.repair));
  corrupt('vet', o => setLine(o, 'c1', 'files: ' + D + ':14:14', 'files: ' + D), s => checks.refused(s.vet, 1, 'analysis', 1, D + ':14'));
  corrupt('vet', o => setLine(o, 'c1', VETMSG, 'vet failed'), s => checks.output(s.vet, 1, 'analysis', VETMSG));
  corrupt('vet', o => { o.feedback.c1 += '\n' + line('layout', 1, D, LAYOUT); }, s => checks.notNamed(s.vet, 1, 'layout'));
  corrupt('vet', o => { o.events[1].checks[1].status = 'attributable'; o.events[1].outcome = 'refused'; }, s => checks.passes(s.vet, 2, 'layout', 'analysis', 'checked'));
  corrupt('fixer', o => { o.after[D] = UNFORMATTED; }, s => checks.publishedFile(s.fixer, D, GOFMT_DOC));
  corrupt('fixer', o => { o.after['.kogen/project.yaml'] = 'version: 1\n# changed\n'; }, s => checks.unchanged(s.fixer, '.kogen/project.yaml'));
  corrupt('fixer', o => { o.after['go.mod'] = 'module example.test/staticshape\n\ngo 1.27.2\n'; }, s => checks.unchanged(s.fixer, 'go.mod'));
  corrupt('fixer', o => { o.approvals[0] = {...o.approvals[0], exit: 0, error: null, approved: true}; }, s => checks.oldHashRefused(s.fixer));
  corrupt('fixer', o => { o.approvals[1].error.reason = 'not_ready'; }, s => checks.changedRefused(s.fixer, 'analysis', D + ':11'));
  corrupt('fixer', o => { o.approvals[1].error.lines = [line('analysis', 1, D, VET), '  | x']; }, s => checks.changedRefused(s.fixer, 'analysis', D + ':11'));
  corrupt('fixer', o => { o.approvals[1].approved = true; }, s => checks.changedRefused(s.fixer, 'analysis', D + ':11'));
  corrupt('fixer', o => { o.approvals.pop(); }, s => checks.changedRefused(s.fixer, 'analysis', D + ':11'));
  corrupt('fixer', o => { o.events[0].outcome = 'refused'; o.gates = [{attempt: 1, gate: 'static_checks', problems: []}]; }, s => checks.passes(s.fixer, 1, 'layout', 'analysis', 'checked'));
  corrupt('unrelated', o => { o.events[0].checks[0].status = 'attributable'; o.events[0].checks[0].paths = [D]; }, s => checks.naming(s.unrelated, 1, 'layout', 'preexisting', 'count/unrelated.go'));
  corrupt('unrelated', o => { o.events[0].checks[0].paths = []; }, s => checks.naming(s.unrelated, 1, 'layout', 'preexisting', 'count/unrelated.go'));
  corrupt('unrelated', o => { o.events[0].checks[0].paths.push(D); }, s => checks.notNaming(s.unrelated, 1, 'layout', D));
  corrupt('unrelated', o => { o.events[0].checks[0].run = 'test -z "$(gofmt -l features/steps)"'; }, s => checks.configured(s.unrelated, 1, 'layout'));
  corrupt('unrelated', o => { o.events[0].checks[0].status = 'passed'; o.events[0].outcome = 'checked'; }, s => checks.naming(s.unrelated, 1, 'layout', 'preexisting', 'count/unrelated.go'));
  corrupt('unrelated', o => { o.events[0].outcome = 'checked'; }, s => checks.outcome(s.unrelated, 1, 'unknown'));
  corrupt('unrelated', o => { o.gates = [{attempt: 1, gate: 'static_checks', problems: []}]; }, s => checks.notRefused(s.unrelated, 1));
  corrupt('unrelated', o => { o.stderr = ''; }, s => checks.stderrHas(s.unrelated, 'kogen: static check layout failed (exit 1) without the shaped files too: pre-existing failure in count/unrelated.go'));
  corrupt('unrelated', o => { o.after['count/unrelated.go'] = 'package count\n\nvar Existing = 1\n'; }, s => checks.unchanged(s.unrelated, 'count/unrelated.go'));
  corrupt('unrelated', o => { o.after['count/unrelated.go'] = 'package count\n\nvar Existing = 1\n'; }, s => checks.reads(s.unrelated, 'count/unrelated.go', 'package count;var Existing=1'));
  corrupt('absent', o => { o.events[0].checks[0].status = 'passed'; delete o.events[0].checks[0].reason; o.events[0].outcome = 'checked'; }, s => checks.reason(s.absent, 1, 'layout', 'unavailable', 'gofmt not found'));
  corrupt('absent', o => { o.events[0].checks[0].reason = 'exit 0'; }, s => checks.reason(s.absent, 1, 'layout', 'unavailable', 'gofmt not found'));
  corrupt('absent', o => { o.events[0].outcome = 'checked'; }, s => checks.outcome(s.absent, 1, 'unknown'));
  corrupt('absent', o => { o.stderr = 'kogen: static check layout passed (0 s)\n'; }, s => checks.stderrHas(s.absent, 'kogen: static check layout unavailable: gofmt not found; the shaped files are not established by this check'));
  corrupt('absent', o => { o.events = []; }, s => checks.notRefused(s.absent, 1));
  corrupt('renamed', o => setLine(o, 'c1', '- check tidy:', '- check layout:'), s => checks.refused(s.renamed, 1, 'tidy', 1, C));
  corrupt('renamed', o => setLine(o, 'c1', 'files: ' + C + ';', 'files: ' + D + ';'), s => checks.refused(s.renamed, 1, 'tidy', 1, C));
  corrupt('renamed', o => { o.feedback.c1 = refusal(1, layoutLines('tidy', C)); }, s => checks.refused(s.renamed, 1, 'inspect', 1, C + ':4'));
  corrupt('renamed', o => { o.events[1].checks[0].status = 'not_run'; }, s => checks.passes(s.renamed, 2, 'tidy', 'inspect', 'checked'));
  corrupt('valid', o => { o.gates = [{attempt: 1, gate: 'static_checks', problems: []}]; o.shaper = 3; }, s => checks.noRefusal(s.valid));
  corrupt('valid', o => { o.gates = [{attempt: 1, gate: 'static_checks', problems: []}]; o.shaper = 3; }, s => checks.publishedAfter(s.valid, 2));
  corrupt('valid', o => { o.events[0].outcome = 'refused'; }, s => checks.passes(s.valid, 1, 'tidy', 'inspect', 'checked'));
  corrupt('valid', o => { o.feedback.c1 = 'accepted: the Intent is ready'; }, s => checks.observations(s.valid, 1, RED));

  const passes = (seen, check) => { try { check(seen); return true; } catch { return false; } };
  const accepted = kind === 'good' ? good.every(check => passes(known, check)) : bad.some(([seen, check]) => passes(seen, check));
  if (process.env.KSSC_CONTROLS_DEBUG) {
    if (kind === 'good') good.forEach((check, i) => { try { check(known); } catch (e) { console.error(`good ${i}: ${e.message.split('\n')[0]}`); } });
    else bad.forEach(([seen, check], i) => { if (passes(seen, check)) console.error(`bad ${i} was accepted: ${check}`); });
  }
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'k-shape-artifact-static-checks: ';
  When(P + 'the {string} case is shaped', {timeout: 900000}, function (name) { this.kssc = shapeCase(name); });
  Then(P + 'submission {int} is refused by the static check {string} with exit {int} naming {string}', function (n, name, exit, file) { checks.refused(this.kssc, n, name, exit, file); });
  Then(P + 'submission {int} shows the output of the static check {string} containing {string}', function (n, name, text) { checks.output(this.kssc, n, name, text); });
  Then(P + 'submission {int} names no static check {string}', function (n, name) { checks.notNamed(this.kssc, n, name); });
  Then(P + 'submission {int} is told the package is unchanged since submission {int}', function (n, m) { checks.unchangedSince(this.kssc, n, m); });
  Then(P + 'submission {int} passes the static checks {string} and {string} with outcome {string}', function (n, a, b, outcome) { checks.passes(this.kssc, n, a, b, outcome); });
  Then(P + 'submission {int} receives readiness observations containing {string}', function (n, text) { checks.observations(this.kssc, n, text); });
  Then(P + 'the Intent is published after {int} submissions', function (n) { checks.publishedAfter(this.kssc, n); });
  Then(P + 'no submission is refused', function () { checks.noRefusal(this.kssc); });
  Then(P + 'submission {int} is not refused by the static checks', function (n) { checks.notRefused(this.kssc, n); });
  Then(P + 'submission {int} records the static check {string} as {string}', function (n, name, status) { checks.status(this.kssc, n, name, status); });
  Then(P + 'submission {int} records the static check {string} as {string} naming {string}', function (n, name, status, file) { checks.naming(this.kssc, n, name, status, file); });
  Then(P + 'submission {int} records the static check {string} without naming {string}', function (n, name, file) { checks.notNaming(this.kssc, n, name, file); });
  Then(P + 'submission {int} records the static check {string} with its configured command', function (n, name) { checks.configured(this.kssc, n, name); });
  Then(P + 'submission {int} records the static check {string} as {string} with the reason {string}', function (n, name, status, reason) { checks.reason(this.kssc, n, name, status, reason); });
  Then(P + 'submission {int} records the static checks outcome {string}', function (n, outcome) { checks.outcome(this.kssc, n, outcome); });
  Then(P + 'stderr contains {string}', function (text) { checks.stderrHas(this.kssc, text); });
  Then(P + 'the published file {string} is the following, each "<TAB>" standing for a tab:', function (p, text) { checks.publishedFile(this.kssc, p, text); });
  Then(P + 'the file {string} is unchanged', function (p) { checks.unchanged(this.kssc, p); });
  Then(P + 'the file {string} reads {string}', function (p, text) { checks.reads(this.kssc, p, text); });
  Then(P + 'the last fresh static checks ran outside the checkout on the published bytes', function () { checks.freshCopy(this.kssc); });
  Then(P + 'the static check record identifies the published package with outcome {string}', function (outcome) { checks.record(this.kssc, outcome); });
  Then(P + 'approving the published package succeeds', function () { checks.approved(this.kssc); });
  Then(P + 'approving the published hash after the helper changed is refused', function () { checks.oldHashRefused(this.kssc); });
  Then(P + "approving the changed package's hash is refused by the static check {string} naming {string}", function (name, file) { checks.changedRefused(this.kssc, name, file); });
}
