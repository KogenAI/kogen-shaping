'use strict';
// Acceptance of k-shape-decision-same-value. The When step shapes one synthetic request and Intent
// pair: it compiles the shape package's tests with the harness of
// k-shape-decision-same-value__harness.js (through a Go overlay), runs
// TestKShapeDecisionSameValueAcceptance for the case with a scripted shaper and a scripted
// decision-check model, and reads what the run showed. The Then steps judge that. Run this file
// directly with good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeDecisionSameValueAcceptance';
const HEADLINE = 'Decision check: ';
let built;
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-decision-same-value-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_decision_same_value_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-decision-same-value__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/k_decision_same_value_acceptance_test.go')]: source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  built = r.status === 0 ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// shapeCase runs the harness for one case once per process and returns what the run showed.
function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  if (b.error) throw new Error(b.error);
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KDSV_CASE: name, KDSV_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KDSV_CASE=${name}): shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

function shown(o) {
  return `case ${o.case} (request ${JSON.stringify(o.request)}): shape ended with exit ${o.exit}, err ${o.err}; published=${o.published}; ` +
    `${o.shaper} shaper request(s) and ${o.decision} decision check request(s)\nfeedback to the shaper:\n${o.feedback.join('\n----\n') || '(none)'}\n` +
    `decision_check events: ${JSON.stringify(o.events)}\nstderr (tail):\n${String(o.stderr).slice(-2000)}`;
}

function intentHas(o, line) {
  return o.intent.split('\n').includes(line);
}

const checks = {
  publishedFirst(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.ok(intentHas(o, o.first), `the published intent.md does not carry the first submission's Decisions line ${JSON.stringify(o.first)}\n${shown(o)}`);
    assert.equal(o.shaper, 2, `the first submission should require one readiness acknowledgment: ${o.shaper} shaper requests, want 2\n${shown(o)}`);
  },
  publishedCorrected(o) {
    assert.ok(o.second, `case ${o.case} has no corrected submission`);
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.ok(intentHas(o, o.second), `the published intent.md does not carry the corrected Decisions line ${JSON.stringify(o.second)}\n${shown(o)}`);
    assert.equal(o.shaper, 3, `the corrected submission should require one readiness acknowledgment: ${o.shaper} shaper requests, want 3\n${shown(o)}`);
  },
  decisionRequests(o, n) {
    assert.equal(o.decision, n, `${o.decision} decision check request(s) (requests offering submit_decision_check), want ${n}\n${shown(o)}`);
  },
  recorded(o, verdict) {
    assert.ok(o.events.length >= 1, `no decision_check event was recorded\n${shown(o)}`);
    const item = (o.events[0].items || []).find(it => it.statement === o.first);
    assert.ok(item, `the first decision_check event has no item for ${JSON.stringify(o.first)}\n${shown(o)}`);
    assert.equal(item.verdict, verdict, `the first decision_check event recorded ${JSON.stringify(item.verdict)} for ${JSON.stringify(o.first)}, want ${JSON.stringify(verdict)}\n${shown(o)}`);
  },
  noFeedback(o, text) {
    assert.ok(!o.feedback.some(f => f.includes(text)), `feedback to the shaper contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  refused(o, a, b, c) {
    assert.ok(o.first_feedback.split('\n').some(l => l.startsWith(HEADLINE)), `the first submission was not refused by the decision check (no line starting ${JSON.stringify(HEADLINE)})\n${shown(o)}`);
    assert.ok(o.first_feedback.split('\n').some(l => l.includes(a) && l.includes(b) && l.includes(c)),
      `no line of the first submission's feedback contains ${JSON.stringify(a)}, ${JSON.stringify(b)} and ${JSON.stringify(c)}\n${shown(o)}`);
  },
  firstLacks(o, text) {
    assert.ok(o.first_feedback !== '', `the first submission got no feedback\n${shown(o)}`);
    assert.ok(!o.first_feedback.includes(text), `the first submission's feedback contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  carries(o) {
    assert.ok(o.decision >= 1, `no decision check request was made\n${shown(o)}`);
    assert.ok(o.input.includes('# Request\n\n' + o.request + '\n'), `the first decision check request does not carry "# Request" followed by the request ${JSON.stringify(o.request)}:\n${o.input}`);
    assert.ok(o.input.includes('# Candidate intent.md\n\n'), `the first decision check request does not carry "# Candidate intent.md":\n${o.input}`);
    assert.ok(o.input.includes(o.first), `the first decision check request does not carry the candidate's Decisions line ${JSON.stringify(o.first)}:\n${o.input}`);
  },
  instructions(o, text) {
    assert.ok(o.decision >= 1, `no decision check request was made\n${shown(o)}`);
    assert.ok(o.body.includes(text), `the first decision check request does not include ${JSON.stringify(text)}:\n${o.body}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const S1 = 'Compare values, not notation: before comparing, normalise digit grouping (1,000 and 1000), percentages and fractions (50% and 0.5), units of the same quantity (2 minutes and 120 seconds), date layouts, the order of a list whose order does not matter, and letter case where case does not matter.';
  const S2 = "A statement that restates the request's value in another notation follows the request.";
  const S3 = 'Give contradicts only when the values still differ after that normalisation, such as a different number, a wrongly converted unit or a different order where order matters; then give intent_value (the value as the statement writes it), request_value (the value as a request quote writes it) and difference (what makes them differ). For follows and unsupported give these three as empty strings.';
  const U = 'limit uploads to 1,000 rows', U1000 = '- Upload limit is 1000 rows (source: request)', U1k = '- Upload limit is 1,000 rows (source: request)', U10k = '- Upload limit is 10,000 rows (source: request)';
  const intent = line => `# Add an is-even check\n\nSlug: is-even\n\ncalc.is_even(n) returns True for even integers.\n\n## Decisions\n${line}\n\n## Tests\n- x (unit)\n`;
  const input = line => `Slug: is-even\n\n# Request\n\n${U}\n\n# Candidate intent.md\n\n${intent(line)}\n`;
  const head = 'Decision check: 1 stated decisions are not supported by the request; follow the request, cite the repository evidence that decides it with (evidence: ...), or mark the decision as an assumption the user must confirm:\nMark an assumption with (assumption: <what the user must confirm>) before the required trailing tag.\n';
  const item = (statement, verdict) => ({statement, request_quotes: [U], verdict, intent_value: '', request_value: '', difference: '', why: 'synthetic scripted item'});
  const event = (...items) => ({event: 'decision_check', items});
  const diffLine = `statement: ${U10k} | request_quotes: "${U}" | verdict: contradicts | intent_value: 10,000 | request_value: 1,000 | difference: 10,000 is ten times 1,000 | why: synthetic scripted item`;
  const unsLine = `statement: ${U1000} | request_quotes: "${U}" | verdict: unsupported | why: synthetic scripted item`;
  const known = {
    same: {case: 'uploads-1000', request: U, first: U1000, second: '', exit: 0, err: '<nil>', published: true, intent: intent(U1000),
      shaper: 2, decision: 1, feedback: [], first_feedback: '', body: `You check the stated decisions.\n${S1}\n${S2}\n${S3}\n${input(U1000)}`, input: input(U1000),
      events: [event(item(U1000, 'follows'))], stderr: ''},
    different: {case: 'uploads-10000', request: U, first: U10k, second: U1k, exit: 0, err: '<nil>', published: true, intent: intent(U1k),
      shaper: 3, decision: 2, feedback: [head + diffLine], first_feedback: head + diffLine, body: '', input: input(U10k),
      events: [event({...item(U10k, 'contradicts'), intent_value: '10,000', request_value: '1,000'}), event(item(U1k, 'follows'))], stderr: ''},
    unnamed: {case: 'uploads-1000-contradicts-without-values', request: U, first: U1000, second: U1k, exit: 0, err: '<nil>', published: true, intent: intent(U1k),
      shaper: 3, decision: 2, feedback: [head + unsLine], first_feedback: head + unsLine, body: '', input: input(U1000),
      events: [event(item(U1000, 'unsupported')), event(item(U1k, 'follows'))], stderr: ''},
  };
  const good = [
    s => checks.publishedFirst(s.same),
    s => checks.decisionRequests(s.same, 1),
    s => checks.recorded(s.same, 'follows'),
    s => checks.noFeedback(s.same, 'Decision check:'),
    s => checks.carries(s.same),
    s => checks.instructions(s.same, S1),
    s => checks.instructions(s.same, S2),
    s => checks.instructions(s.same, S3),
    s => checks.refused(s.different, 'verdict: contradicts', 'intent_value: 10,000', 'request_value: 1,000'),
    s => checks.recorded(s.different, 'contradicts'),
    s => checks.publishedCorrected(s.different),
    s => checks.refused(s.unnamed, 'verdict: unsupported', `statement: ${U1000}`, `request_quotes: "${U}"`),
    s => checks.firstLacks(s.unnamed, 'contradicts'),
    s => checks.recorded(s.unnamed, 'unsupported'),
  ];
  const changed = (name, change) => {
    const s = JSON.parse(JSON.stringify(known));
    change(s[name]);
    return s;
  };
  const unchangedTree = o => { o.decision = 0; o.events = []; o.body = ''; o.input = ''; o.feedback = []; o.first_feedback = ''; o.shaper = 1; o.intent = intent(o.first); };
  const bad = [
    [changed('same', unchangedTree), s => checks.decisionRequests(s.same, 1)],
    [changed('same', unchangedTree), s => checks.recorded(s.same, 'follows')],
    [changed('same', unchangedTree), s => checks.carries(s.same)],
    [changed('same', unchangedTree), s => checks.instructions(s.same, S1)],
    [changed('same', o => { o.shaper = 3; }), s => checks.publishedFirst(s.same)],
    [changed('same', o => { o.published = false; }), s => checks.publishedFirst(s.same)],
    [changed('same', o => { o.intent = intent(U1k); }), s => checks.publishedFirst(s.same)],
    [changed('same', o => { o.decision = 2; }), s => checks.decisionRequests(s.same, 1)],
    [changed('same', o => { o.feedback = [head + unsLine]; }), s => checks.noFeedback(s.same, 'Decision check:')],
    [changed('same', o => { o.events[0].items[0].verdict = 'contradicts'; }), s => checks.recorded(s.same, 'follows')],
    [changed('same', o => { o.events[0].items[0].statement = U1k; }), s => checks.recorded(s.same, 'follows')],
    [changed('same', o => { o.body = o.body.replace(S2, ''); }), s => checks.instructions(s.same, S2)],
    [changed('same', o => { o.input = o.input.replace(U + '\n', 'another request\n'); }), s => checks.carries(s.same)],
    [changed('same', o => { o.input = o.input.replace('# Candidate intent.md', '# Intent'); }), s => checks.carries(s.same)],
    [changed('different', unchangedTree), s => checks.refused(s.different, 'verdict: contradicts', 'intent_value: 10,000', 'request_value: 1,000')],
    [changed('different', unchangedTree), s => checks.publishedCorrected(s.different)],
    [changed('different', o => { o.first_feedback = o.first_feedback.replace(' | request_value: 1,000', ''); }), s => checks.refused(s.different, 'verdict: contradicts', 'intent_value: 10,000', 'request_value: 1,000')],
    [changed('different', o => { o.first_feedback = o.first_feedback.replace(' | request_value: 1,000', '\nrequest_value: 1,000'); }), s => checks.refused(s.different, 'verdict: contradicts', 'intent_value: 10,000', 'request_value: 1,000')],
    [changed('different', o => { o.first_feedback = o.first_feedback.replace(HEADLINE, 'Readiness: '); }), s => checks.refused(s.different, 'verdict: contradicts', 'intent_value: 10,000', 'request_value: 1,000')],
    [changed('different', o => { o.events[0].items[0].verdict = 'unsupported'; }), s => checks.recorded(s.different, 'contradicts')],
    [changed('different', o => { o.shaper = 4; }), s => checks.publishedCorrected(s.different)],
    [changed('unnamed', o => { o.first_feedback = o.first_feedback.replace('verdict: unsupported', 'verdict: contradicts'); }), s => checks.firstLacks(s.unnamed, 'contradicts')],
    [changed('unnamed', o => { o.first_feedback = o.first_feedback.replace('verdict: unsupported', 'verdict: contradicts'); }), s => checks.refused(s.unnamed, 'verdict: unsupported', `statement: ${U1000}`, `request_quotes: "${U}"`)],
    [changed('unnamed', unchangedTree), s => checks.firstLacks(s.unnamed, 'contradicts')],
    [changed('unnamed', o => { o.events[0].items[0].verdict = 'contradicts'; }), s => checks.recorded(s.unnamed, 'unsupported')],
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
  When('k-shape-decision-same-value: the {string} case is shaped', {timeout: 540000}, function (name) { this.kdsv = shapeCase(name); });
  Then('k-shape-decision-same-value: the Intent is published from the first submission', function () { checks.publishedFirst(this.kdsv); });
  Then('k-shape-decision-same-value: the Intent is published from the corrected submission', function () { checks.publishedCorrected(this.kdsv); });
  Then('k-shape-decision-same-value: exactly {int} decision check request(s) is/are made', function (n) { checks.decisionRequests(this.kdsv, n); });
  Then('k-shape-decision-same-value: the decision check recorded {string} for the first submission', function (verdict) { checks.recorded(this.kdsv, verdict); });
  Then('k-shape-decision-same-value: no feedback to the shaper contains {string}', function (text) { checks.noFeedback(this.kdsv, text); });
  Then('k-shape-decision-same-value: the first submission is refused by the decision check with a line containing {string}, {string} and {string}', function (a, b, c) { checks.refused(this.kdsv, a, b, c); });
  Then('k-shape-decision-same-value: the first submission\'s feedback does not contain {string}', function (text) { checks.firstLacks(this.kdsv, text); });
  Then('k-shape-decision-same-value: the decision check request carries the request and the candidate intent.md', function () { checks.carries(this.kdsv); });
  Then('k-shape-decision-same-value: the decision check request includes:', function (text) { checks.instructions(this.kdsv, text); });
}
