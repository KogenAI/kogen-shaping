'use strict';
// Acceptance of k-shape-property-coverage. The When step shapes one synthetic case of the request:
// it compiles the shape package's tests with the harness of k-shape-property-coverage__harness.js
// (through a Go overlay), runs TestKPropertyCoverageAcceptance for the case and reads what a user
// sees of each candidate. The Then steps judge that. Run this file directly with good or bad to
// check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKPropertyCoverageAcceptance';
const HEADLINE = 'Property check: ';
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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-property-coverage-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_property_coverage_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-property-coverage__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_property_coverage_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 200000});
  built = r.status === 0 ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// shapeCase runs the harness for one case and returns what a user sees of each candidate, by name.
function shapeCase(mode) {
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, mode + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=420s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KPC_MODE: mode, KPC_OUT: out}), encoding: 'utf8', timeout: 440000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KPC_MODE=${mode}): shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function candidate(seen, name) {
  const o = seen[name];
  assert.ok(o, `no candidate ${JSON.stringify(name)} was shaped; shaped: ${Object.keys(seen).join(', ')}`);
  return o;
}

function shown(name, o) {
  return `candidate ${JSON.stringify(name)}: shape ended with ${o.exit}; published=${o.published}; ` +
    `${o.shaper} shaper and ${o.contract} contract check request(s)\nfirst submission result:\n${o.feedback}\nnot-ready lines:\n${o.remaining}\nprogress:\n${o.progress}`;
}

function lineWith(text, a, b) {
  return text.split('\n').some(line => line.includes(a) && line.includes(b));
}

const checks = {
  refused(seen, name, property, reason) {
    const o = candidate(seen, name);
    assert.equal(o.published, false, `the candidate was published\n${shown(name, o)}`);
    assert.ok(o.feedback.includes(HEADLINE), `the first submission was not refused by the property check (${JSON.stringify(HEADLINE)})\n${shown(name, o)}`);
    assert.ok(lineWith(o.feedback, property, reason), `no line of the refusal names ${JSON.stringify(property)} with ${JSON.stringify(reason)}\n${shown(name, o)}`);
    assert.ok(lineWith(o.remaining, property, reason), `no line of the not-ready report names ${JSON.stringify(property)} with ${JSON.stringify(reason)}\n${shown(name, o)}`);
  },
  publishes(seen, name) {
    const o = candidate(seen, name);
    assert.ok(!o.progress.includes(INCOMPLETE), `the contract check did not take the review's answer\n${shown(name, o)}`);
    assert.ok(!o.feedback.includes(HEADLINE), `the property check refused the candidate\n${shown(name, o)}`);
    assert.equal(o.published, true, `the candidate was not published\n${shown(name, o)}`);
  },
  requests(seen, name, shaper, contract) {
    const o = candidate(seen, name);
    assert.deepEqual({shaper: o.shaper, contract: o.contract}, {shaper, contract}, `model requests differ\n${shown(name, o)}`);
  },
  contains(seen, name, text) {
    const o = candidate(seen, name);
    assert.ok(o.feedback.includes(text), `the first submission result does not contain ${JSON.stringify(text)}\n${shown(name, o)}`);
  },
  lacks(seen, name, text) {
    const o = candidate(seen, name);
    assert.ok(!o.feedback.includes(text), `the first submission result contains ${JSON.stringify(text)}\n${shown(name, o)}`);
  },
  review(seen, name, text) {
    const o = candidate(seen, name);
    assert.ok(o.review.includes(text), `the contract review input does not contain ${JSON.stringify(text)}:\n${o.review}\n${shown(name, o)}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const line = '- add keeps working (kept) | request: Keep add(a, b) | problem: no scenario exercises it | confirmed: synthetic confirmation';
  const known = {
    refused: {published: false, shaper: 3, contract: 2, exit: 'exit 1, err intent/not_ready', progress: 'kogen: checking the contract (1 s)\n',
      feedback: 'Property check: 1 problem(s) with how acceptance covers the properties the request asks for; nothing was published.\n' + line,
      remaining: line, review: 'Slug: is-even\n\n# Request\n\nKeep add(a, b) and sub(a, b) working exactly as they do now.\n\n# Questions and answers\n\nnone'},
    published: {published: true, shaper: 2, contract: 1, exit: 'exit 0, err <nil>', progress: 'kogen: checking the contract (1 s)\n',
      feedback: 'Readiness observations\nScenario: Mul multiplies (acceptance.feature line 6)\nTypeError: mul is not a function', remaining: '', review: ''},
  };
  const good = [
    s => checks.refused(s, 'refused', 'add keeps working', 'no scenario exercises it'),
    s => checks.requests(s, 'refused', 3, 2),
    s => checks.review(s, 'refused', '# Request\n\nKeep add(a, b)'),
    s => checks.publishes(s, 'published'),
    s => checks.requests(s, 'published', 2, 1),
    s => checks.contains(s, 'published', 'Scenario: Mul multiplies'),
    s => checks.lacks(s, 'published', 'Scenario: Add keeps working ('),
  ];
  const changed = (name, change) => {
    const s = JSON.parse(JSON.stringify(known));
    change(s[name]);
    return s;
  };
  const bad = [
    [changed('refused', o => { o.feedback = o.feedback.replace('add keeps', 'sub keeps'); o.remaining = o.remaining.replace('add keeps', 'sub keeps'); }),
      s => checks.refused(s, 'refused', 'add keeps working', 'no scenario exercises it')],
    [changed('refused', o => { o.remaining = ''; }), s => checks.refused(s, 'refused', 'add keeps working', 'no scenario exercises it')],
    [changed('refused', o => { o.published = true; }), s => checks.refused(s, 'refused', 'add keeps working', 'no scenario exercises it')],
    [changed('refused', o => { o.feedback = 'Readiness observations\n' + line; }), s => checks.refused(s, 'refused', 'add keeps working', 'no scenario exercises it')],
    [changed('refused', o => { o.review = 'Slug: is-even\n\n# Candidate intent.md'; }), s => checks.review(s, 'refused', '# Request\n\nKeep add(a, b)')],
    [changed('published', o => { o.progress += 'kogen: contract check incomplete (invalid submit_contract_review payload); the candidate goes on (1 s)\n'; }), s => checks.publishes(s, 'published')],
    [changed('published', o => { o.feedback = 'Property check: 1 problem(s)\n' + line; }), s => checks.publishes(s, 'published')],
    [changed('published', o => { o.published = false; }), s => checks.publishes(s, 'published')],
    [changed('published', o => { o.shaper = 3; }), s => checks.requests(s, 'published', 2, 1)],
    [changed('published', o => { o.contract = 2; }), s => checks.requests(s, 'published', 2, 1)],
    [changed('published', o => { o.feedback = 'Readiness observations'; }), s => checks.contains(s, 'published', 'Scenario: Mul multiplies')],
    [changed('published', o => { o.feedback += '\nScenario: Add keeps working (acceptance.feature line 2)'; }), s => checks.lacks(s, 'published', 'Scenario: Add keeps working (')],
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
  When('k-shape-property-coverage: the {string} case is shaped', {timeout: 660000}, function (mode) { this.kpc = shapeCase(mode); });
  Then('k-shape-property-coverage: candidate {string} is refused naming {string} with {string}', function (name, property, reason) { checks.refused(this.kpc, name, property, reason); });
  Then('k-shape-property-coverage: candidate {string} publishes after a complete contract check', function (name) { checks.publishes(this.kpc, name); });
  Then('k-shape-property-coverage: candidate {string} made {int} shaper request(s) and {int} contract check request(s)', function (name, shaper, contract) { checks.requests(this.kpc, name, shaper, contract); });
  Then('k-shape-property-coverage: the first result for candidate {string} contains {string}', function (name, text) { checks.contains(this.kpc, name, text); });
  Then('k-shape-property-coverage: the first result for candidate {string} does not contain {string}', function (name, text) { checks.lacks(this.kpc, name, text); });
  Then('k-shape-property-coverage: the contract review for candidate {string} includes:', function (name, text) { checks.review(this.kpc, name, text); });
}
