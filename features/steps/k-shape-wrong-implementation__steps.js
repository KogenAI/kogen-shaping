'use strict';
// Acceptance of k-shape-wrong-implementation. The When step shapes one synthetic case of the
// request: it compiles the shape package's tests with the harness of
// k-shape-wrong-implementation__harness.js (through a Go overlay), runs
// TestKWrongImplementationAcceptance for the case and reads what a user sees of each journey.
// The Then steps judge that, each about the submission it names. Run this file directly with
// good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKWrongImplementationAcceptance';
const HEADLINE = 'Wrong-implementation check: ';
const INCOMPLETE = 'contract check incomplete';
let built;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-wrong-implementation-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_wrong_implementation_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-wrong-implementation__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_wrong_implementation_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 300000});
  built = r.status === 0 ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// shapeCase runs the harness for one case and returns what a user sees of each journey, by name.
function shapeCase(mode) {
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, mode + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=420s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KWI_MODE: mode, KWI_OUT: out}), encoding: 'utf8', timeout: 440000, maxBuffer: 64 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KWI_MODE=${mode}): shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function journey(seen, name) {
  const o = seen[name];
  assert.ok(o, `no journey ${JSON.stringify(name)} was shaped; shaped: ${Object.keys(seen).join(', ')}`);
  return o;
}

function shown(name, o) {
  const results = Object.entries(o.results || {}).map(([id, text]) => `result of submission ${id}:\n${text}`).join('\n');
  return `journey ${JSON.stringify(name)}: shape ended with ${o.exit}; published=${o.published}; files on disk=${o.files}; ` +
    `${o.shaper} shaper and ${o.contract} contract check request(s); ${o.runs} acceptance run(s); time cap field=${o.time_cap_field}\n` +
    `${results}\nnot-ready lines:\n${o.remaining}\nprogress:\n${o.progress}`;
}

function result(name, o, sub) {
  const text = (o.results || {})[sub];
  assert.equal(typeof text, 'string', `submission ${JSON.stringify(sub)} got no result\n${shown(name, o)}`);
  return text;
}

function lineWith(text, a, b) {
  return text.split('\n').some(line => line.includes(a) && line.includes(b));
}

const checks = {
  refused(seen, name, sub, a, b) {
    const o = journey(seen, name);
    const text = result(name, o, sub);
    assert.equal(o.published, false, `the journey published\n${shown(name, o)}`);
    assert.ok(text.includes(HEADLINE), `the result of submission ${JSON.stringify(sub)} is not a refusal by the wrong-implementation check (${JSON.stringify(HEADLINE)})\n${shown(name, o)}`);
    assert.ok(lineWith(text, a, b), `no line of the result of submission ${JSON.stringify(sub)} names ${JSON.stringify(a)} with ${JSON.stringify(b)}\n${shown(name, o)}`);
    assert.ok(lineWith(o.remaining, a, b), `no line of the not-ready report names ${JSON.stringify(a)} with ${JSON.stringify(b)}\n${shown(name, o)}`);
  },
  refusedBefore(seen, name, a, b) {
    const o = journey(seen, name);
    const text = result(name, o, 'cand');
    assert.equal(o.published, false, `the journey published\n${shown(name, o)}`);
    assert.ok(!text.includes(HEADLINE), `the wrong-implementation check refused the candidate\n${shown(name, o)}`);
    assert.ok(lineWith(text, a, b), `no line of the result of submission "cand" names ${JSON.stringify(a)} with ${JSON.stringify(b)}\n${shown(name, o)}`);
    assert.ok(lineWith(o.remaining, a, b), `no line of the not-ready report names ${JSON.stringify(a)} with ${JSON.stringify(b)}\n${shown(name, o)}`);
  },
  refined(seen, name) {
    const o = journey(seen, name);
    const why = `\nrefinement: ${o.refine_exit}; outcome=${JSON.stringify(o.refine_outcome)}; written=${o.refined}; ${o.refine_runs} acceptance run(s)\n${o.refine_progress}\n${shown(name, o)}`;
    assert.equal(o.refine_exit, 'exit 0, err <nil>', `the refinement did not succeed${why}`);
    assert.equal(o.refine_outcome, 'ready', `the refinement is not ready${why}`);
    assert.equal(o.refined, true, `the refined intent.md was not written${why}`);
    assert.ok(!o.refine_progress.toLowerCase().includes('wrong-implementation check'), `the refinement ran the wrong-implementation check${why}`);
  },
  refineRuns(seen, name, runs) {
    const o = journey(seen, name);
    assert.equal(o.refine_runs, runs, `the refinement ran acceptance ${o.refine_runs} time(s), not ${runs}\n${o.refine_progress}\n${shown(name, o)}`);
  },
  publishes(seen, name) {
    const o = journey(seen, name);
    assert.ok(!o.progress.includes(INCOMPLETE), `the contract check did not take the review's answer\n${shown(name, o)}`);
    assert.equal(o.published, true, `the journey did not publish\n${shown(name, o)}`);
    assert.equal(o.files, true, `the published Intent files are not the final candidate's\n${shown(name, o)}`);
  },
  requests(seen, name, shaper, contract) {
    const o = journey(seen, name);
    assert.deepEqual({shaper: o.shaper, contract: o.contract}, {shaper, contract}, `model requests differ\n${shown(name, o)}`);
  },
  runs(seen, name, runs) {
    const o = journey(seen, name);
    assert.equal(o.runs, runs, `acceptance ran ${o.runs} time(s), not ${runs}\n${shown(name, o)}`);
  },
  contains(seen, name, sub, text) {
    const o = journey(seen, name);
    assert.ok(result(name, o, sub).includes(text), `the result of submission ${JSON.stringify(sub)} does not contain ${JSON.stringify(text)}\n${shown(name, o)}`);
  },
  lacks(seen, name, sub, text) {
    const o = journey(seen, name);
    assert.ok(!result(name, o, sub).includes(text), `the result of submission ${JSON.stringify(sub)} contains ${JSON.stringify(text)}\n${shown(name, o)}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const line = '- add keeps working (kept) | request: answering add | problem: wrong implementation "add removed" passes scenario "Add keeps working"';
  const refusal = 'Wrong-implementation check: 1 problem(s) with whether acceptance rejects a wrong implementation of each property the request asks for; nothing was published.\n' + line;
  const known = {
    refused: {published: false, files: false, shaper: 3, contract: 1, runs: 5, exit: 'exit 1, err intent/not_ready', time_cap_field: true,
      results: {cand: refusal, again: refusal + '\nUnchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.'},
      remaining: line, progress: 'kogen: checking the contract (1 s)\n'},
    changed: {published: false, files: false, shaper: 4, contract: 2, runs: 10, exit: 'exit 1, err intent/not_ready', time_cap_field: true,
      results: {cand: 'Readiness observations\nScenario: Mul multiplies (acceptance.feature line 6)', change: refusal},
      remaining: line, progress: 'kogen: checking the contract (1 s)\n'},
    published: {published: true, files: true, shaper: 2, contract: 1, runs: 5, exit: 'exit 0, err <nil>', time_cap_field: true,
      results: {cand: 'Readiness observations\nScenario: Mul multiplies (acceptance.feature line 6)\nAssertionError [ERR_ASSERTION]'},
      remaining: '', progress: 'kogen: checking the contract (1 s)\n'},
    existing: {published: false, files: false, shaper: 3, contract: 2, runs: 2, exit: 'exit 1, err intent/not_ready', time_cap_field: true,
      results: {cand: 'Property check: 1 problem(s) with how acceptance covers the properties the request asks for; nothing was published.\n- sub keeps working (kept) | request: sub exactly as it does now | problem: no scenario exercises it | confirmed: synthetic confirmation'},
      remaining: '- sub keeps working (kept) | request: sub exactly as it does now | problem: no scenario exercises it | confirmed: synthetic confirmation', progress: ''},
    refining: {published: true, files: true, shaper: 4, contract: 2, runs: 3, exit: 'exit 0, err <nil>', time_cap_field: true, results: {},
      remaining: '', progress: '', refine_exit: 'exit 0, err <nil>', refine_outcome: 'ready', refined: true, refine_runs: 1, refine_progress: 'kogen: checking the contract (1 s)\n'},
  };
  const good = [
    s => checks.refused(s, 'refused', 'cand', 'add keeps working', 'passes scenario "Add keeps working"'),
    s => checks.refused(s, 'changed', 'change', 'add keeps working', 'passes scenario "Add keeps working"'),
    s => checks.lacks(s, 'changed', 'cand', HEADLINE),
    s => checks.publishes(s, 'published'),
    s => checks.requests(s, 'published', 2, 1),
    s => checks.runs(s, 'published', 5),
    s => checks.contains(s, 'published', 'cand', 'Scenario: Mul multiplies'),
    s => checks.lacks(s, 'published', 'cand', HEADLINE),
    s => checks.refusedBefore(s, 'existing', 'sub keeps working', 'no scenario exercises it'),
    s => checks.refined(s, 'refining'),
    s => checks.refineRuns(s, 'refining', 1),
  ];
  const changed = (name, change) => {
    const s = JSON.parse(JSON.stringify(known));
    change(s[name]);
    return s;
  };
  const refusedAdd = s => checks.refused(s, 'refused', 'cand', 'add keeps working', 'passes scenario "Add keeps working"');
  const bad = [
    [changed('refused', o => { o.results.cand = o.results.cand.replace('add keeps', 'sub keeps'); }), refusedAdd],
    [changed('refused', o => { o.remaining = ''; }), refusedAdd],
    [changed('refused', o => { o.published = true; }), refusedAdd],
    [changed('refused', o => { o.results.cand = 'Readiness observations\n' + line; }), refusedAdd],
    [changed('refused', o => { delete o.results.cand; }), refusedAdd],
    [changed('changed', o => { o.results.change = o.results.cand; }), s => checks.refused(s, 'changed', 'change', 'add keeps working', 'passes scenario "Add keeps working"')],
    [changed('changed', o => { o.results.cand = refusal; }), s => checks.lacks(s, 'changed', 'cand', HEADLINE)],
    [changed('published', o => { o.progress += 'kogen: contract check incomplete (invalid submit_contract_review payload); the candidate goes on (1 s)\n'; }), s => checks.publishes(s, 'published')],
    [changed('published', o => { o.published = false; }), s => checks.publishes(s, 'published')],
    [changed('published', o => { o.files = false; }), s => checks.publishes(s, 'published')],
    [changed('published', o => { o.shaper = 3; }), s => checks.requests(s, 'published', 2, 1)],
    [changed('published', o => { o.contract = 2; }), s => checks.requests(s, 'published', 2, 1)],
    [changed('published', o => { o.runs = 4; }), s => checks.runs(s, 'published', 5)],
    [changed('published', o => { o.results.cand = 'Readiness observations'; }), s => checks.contains(s, 'published', 'cand', 'Scenario: Mul multiplies')],
    [changed('published', o => { o.results.cand += '\n' + refusal; }), s => checks.lacks(s, 'published', 'cand', HEADLINE)],
    [changed('existing', o => { o.results.cand += '\n' + refusal; }), s => checks.refusedBefore(s, 'existing', 'sub keeps working', 'no scenario exercises it')],
    [changed('existing', o => { o.published = true; }), s => checks.refusedBefore(s, 'existing', 'sub keeps working', 'no scenario exercises it')],
    [changed('existing', o => { o.remaining = ''; }), s => checks.refusedBefore(s, 'existing', 'sub keeps working', 'no scenario exercises it')],
    [changed('refining', o => { o.refine_exit = 'exit 1, err intent/not_ready'; }), s => checks.refined(s, 'refining')],
    [changed('refining', o => { o.refine_outcome = 'invalid'; }), s => checks.refined(s, 'refining')],
    [changed('refining', o => { o.refined = false; }), s => checks.refined(s, 'refining')],
    [changed('refining', o => { o.refine_progress += 'kogen: wrong-implementation check: run 1 of 2 (correct implementation)\n'; }), s => checks.refined(s, 'refining')],
    [changed('refining', o => { o.refine_runs = 3; }), s => checks.refineRuns(s, 'refining', 1)],
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
  const P = 'k-shape-wrong-implementation: ';
  When(P + 'the {string} case is shaped', {timeout: 600000}, function (mode) { this.kwi = shapeCase(mode); });
  Then(P + 'candidate {string} is refused with a line naming {string} and {string}', function (name, a, b) { checks.refused(this.kwi, name, 'cand', a, b); });
  Then(P + 'submission {string} of candidate {string} is refused with a line naming {string} and {string}', function (sub, name, a, b) { checks.refused(this.kwi, name, sub, a, b); });
  Then(P + 'candidate {string} publishes', function (name) { checks.publishes(this.kwi, name); });
  Then(P + 'candidate {string} is refused by an existing check with a line naming {string} and {string}', function (name, a, b) { checks.refusedBefore(this.kwi, name, a, b); });
  Then(P + 'the refinement of candidate {string} is ready and written', function (name) { checks.refined(this.kwi, name); });
  Then(P + 'the refinement of candidate {string} ran acceptance {int} time(s)', function (name, runs) { checks.refineRuns(this.kwi, name, runs); });
  Then(P + 'candidate {string} made {int} shaper request(s) and {int} contract check request(s)', function (name, shaper, contract) { checks.requests(this.kwi, name, shaper, contract); });
  Then(P + 'candidate {string} ran acceptance {int} time(s)', function (name, runs) { checks.runs(this.kwi, name, runs); });
  Then(P + 'the result of submission {string} of candidate {string} contains {string}', function (sub, name, text) { checks.contains(this.kwi, name, sub, text); });
  Then(P + 'the result of submission {string} of candidate {string} does not contain {string}', function (sub, name, text) { checks.lacks(this.kwi, name, sub, text); });
}
