'use strict';
// Acceptance of k-shape-request-literal-coverage. The When step shapes one synthetic request and
// candidate: it compiles the shape package's tests with the harness of
// k-shape-request-literal-coverage__harness.js (through a Go overlay), runs
// TestKShapeRequestLiteralCoverageAcceptance for the case with a scripted shaper and a scripted
// literal-check model, and reads what the run showed. The Then steps judge that. Run this file
// directly with good or bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeRequestLiteralCoverageAcceptance';
const HEADLINE = 'Literal check: ';
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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-request-literal-coverage-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_request_literal_coverage_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-request-literal-coverage__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve('shape/k_request_literal_coverage_acceptance_test.go')]: source}}));
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
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KRLC_CASE: name, KRLC_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KRLC_CASE=${name}): shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape) exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

function shown(o) {
  return `case ${o.case}: shape ended with exit ${o.exit}, err ${o.err}; published=${o.published}; ` +
    `${o.shaper} shaper request(s) in the last run and ${o.literal} literal check request(s); refused gates ${JSON.stringify(o.gates)}\n` +
    `feedback to the shaper:\n${o.feedback.join('\n----\n') || '(none)'}\n` +
    `literal_check events: ${JSON.stringify(o.events)}\nstderr (tail):\n${String(o.stderr).slice(-2000)}`;
}

function intentHas(o, line) {
  return o.intent.split('\n').includes(line);
}

function compared(o) {
  const e = o.events.find(ev => ev.status === 'compared');
  assert.ok(e, `no literal_check event with status "compared" was recorded\n${shown(o)}`);
  return e;
}

const checks = {
  publishedFirst(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.ok(intentHas(o, o.first), `the published intent.md does not carry the first submission's Decisions line ${JSON.stringify(o.first)}\n${shown(o)}`);
    assert.equal(o.shaper, 1, `the Intent was not published from the first submission: ${o.shaper} shaper requests, want 1\n${shown(o)}`);
  },
  publishedCorrected(o) {
    assert.ok(o.second, `case ${o.case} has no corrected submission`);
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.ok(intentHas(o, o.second), `the published intent.md does not carry the corrected Decisions line ${JSON.stringify(o.second)}\n${shown(o)}`);
    assert.equal(o.shaper, 2, `the Intent was not published from the corrected (second) submission: ${o.shaper} shaper requests, want 2\n${shown(o)}`);
  },
  literalRequests(o, n) {
    assert.equal(o.literal, n, `${o.literal} literal check request(s) (requests offering submit_literal_check), want ${n}\n${shown(o)}`);
  },
  refused(o, a, b, c) {
    const lines = o.first_feedback.split('\n');
    assert.ok(lines.some(l => l.startsWith(HEADLINE)), `the first submission was not refused by the literal check (no line starting ${JSON.stringify(HEADLINE)})\n${shown(o)}`);
    assert.ok(o.gates.includes('literal_check'), `no submission_refused event has gate literal_check\n${shown(o)}`);
    assert.ok(lines.some(l => l.includes(a) && l.includes(b) && l.includes(c)),
      `no line of the first submission's feedback contains ${JSON.stringify(a)}, ${JSON.stringify(b)} and ${JSON.stringify(c)}\n${shown(o)}`);
  },
  refusedWithout(o, text) {
    const lines = o.first_feedback.split('\n');
    assert.ok(lines.some(l => l.startsWith(HEADLINE)), `the first submission was not refused by the literal check (no line starting ${JSON.stringify(HEADLINE)})\n${shown(o)}`);
    assert.ok(!lines.some(l => l.includes(text)), `a line of the first submission's feedback contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  noFeedback(o, text) {
    assert.ok(!o.feedback.some(f => f.includes(text)), `feedback to the shaper contains ${JSON.stringify(text)}\n${shown(o)}`);
    assert.ok(!o.gates.includes('literal_check'), `a submission was refused with gate literal_check\n${shown(o)}`);
  },
  intentContains(o, text) {
    assert.ok(o.intent.includes(text), `the published intent.md does not contain ${JSON.stringify(text)}:\n${o.intent}`);
  },
  intentAsSubmitted(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
    assert.equal(o.intent, o.first_intent, `the published intent.md is not the submitted one byte for byte\n${shown(o)}`);
    assert.equal(o.approved, false, `an approval exists after shaping (.kogen/local/approvals/is-even.json)\n${shown(o)}`);
    assert.ok(o.stdout.includes('Next: kogen intent approve is-even'), `stdout does not offer the normal approval step "Next: kogen intent approve is-even":\n${o.stdout}`);
  },
  recorded(o, literal, status) {
    const e = compared(o);
    const item = (e.items || []).find(it => it.literal === literal);
    assert.ok(item, `the first compared literal_check event has no item for ${JSON.stringify(literal)}\n${shown(o)}`);
    assert.equal(item.status, status, `the first compared literal_check event recorded ${JSON.stringify(item.status)} for ${JSON.stringify(literal)}, want ${JSON.stringify(status)}\n${shown(o)}`);
  },
  secretCount(o, n) {
    const e = compared(o);
    assert.equal(e.secret, n, `the first compared literal_check event counts ${JSON.stringify(e.secret)} secret item(s), want ${n}\n${shown(o)}`);
  },
  notExposed(o, text) {
    const places = {feedback: o.feedback.join('\n'), stdout: o.stdout, stderr: o.stderr, 'literal_check events': JSON.stringify(o.events), 'published intent.md': o.intent};
    for (const [where, value] of Object.entries(places)) {
      assert.ok(!String(value).includes(text), `the ${where} contain(s) the secret ${JSON.stringify(text)}\n${shown(o)}`);
    }
  },
  unknown(o) {
    const e = o.events.find(ev => ev.status === 'unknown');
    assert.ok(e, `no literal_check event with status "unknown" was recorded\n${shown(o)}`);
    assert.equal(e.reason, 'the original request is not available', `the unknown literal_check event gives reason ${JSON.stringify(e.reason)}\n${shown(o)}`);
    assert.ok(!o.events.some(ev => ev.status === 'compared'), `a literal_check event with status "compared" was recorded although the original request is not available\n${shown(o)}`);
  },
  stderrHas(o, text) {
    assert.ok(o.stderr.includes(text), `stderr does not contain ${JSON.stringify(text)}:\n${o.stderr}`);
  },
  carries(o) {
    assert.ok(o.literal >= 1, `no literal check request was made\n${shown(o)}`);
    for (const want of ['# Request\n\n' + o.request + '\n', '# Answers\n\n', '# Candidate package\n\n',
      '## .kogen/intents/is-even/intent.md\n\n', '## .kogen/intents/is-even/acceptance.feature\n\n', '## features/steps/is-even__steps.js\n\n', o.first]) {
      assert.ok(o.input.includes(want), `the first literal check request does not carry ${JSON.stringify(want)}:\n${o.input}`);
    }
  },
  carriesText(o, text) {
    assert.ok(o.literal >= 1, `no literal check request was made\n${shown(o)}`);
    assert.ok(o.input.includes(text), `the first literal check request does not carry ${JSON.stringify(text)}:\n${o.input}`);
  },
  instructions(o, text) {
    assert.ok(o.literal >= 1, `no literal check request was made\n${shown(o)}`);
    assert.ok(o.body.includes(text), `the first literal check request does not include ${JSON.stringify(text)}:\n${o.body}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const S1 = 'A required literal is a value that the request or a settled answer explicitly requires the implementation or an example to use exactly, such as a path, command, name, identifier, number or example input; descriptive text that may be paraphrased is not a literal.';
  const S2 = 'Give status superseded for a value that the request or a later answer explicitly replaces, and status secret for secret material such as a password, token, key or credential; for secret material leave literal, source_quote, package_value and why empty.';
  const S3 = 'Copy each literal exactly as its source writes it, with source_quote copied exactly from that source and containing the literal; Kogen itself compares each literal with the package byte for byte.';
  const S4 = 'Give status informal for a value that appears only in informal background, an aside or an example the source marks optional; Kogen never demands an informal value.';
  const INFORMAL = 'For background only, an old prototype once used /legacy-demo; it is not part of this change.';
  const R = 'Add a registration example to the README.\nThe example must register exactly these two repositories: /demo_one and /demo-two.\nAn earlier draft used /demo_old; it is superseded by /demo_one.\nKeep the wording friendly.';
  const Q = 'The example must register exactly these two repositories: /demo_one and /demo-two.';
  const SECRET = 'hunter2-demo-secret';
  const NONE = '- The README example registers both repositories. (source: request)';
  const BOTH = '- The README example registers /demo_one and /demo-two. (source: request)';
  const SUB = '- The README example registers /demo-one and /demo-two. (source: request)';
  const ONE = '- The README example registers /demo_one. (source: request)';
  const FRIENDLY = '- The README example registers /demo_one and /demo-two in a welcoming tone. (source: request)';
  const PNONE = '- The README example shows the chosen port. (source: answer 1)';
  const PORT = '- The README example shows port 7443. (source: answer 1)';
  const PANSWER = 'The example uses port 7443';
  const intent = line => `# Add an is-even check\n\nSlug: is-even\n\ncalc.is_even(n) returns True for even integers.\n\n## Decisions\n${line}\n\n## Tests\n- x (unit)\n`;
  const input = (request, answers, line) => `Slug: is-even\n\n# Request\n\n${request}\n\n# Answers\n\n${answers}\n\n# Candidate package\n\n## .kogen/intents/is-even/intent.md\n\n${intent(line)}\n\n## .kogen/intents/is-even/acceptance.feature\n\nFeature: Is even\n\n## features/steps/is-even__steps.js\n\n// steps\n`;
  const head = 'Literal check: 2 required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:';
  const miss = (lit, src, quote) => `literal: ${lit} | source: ${src} | source_quote: "${quote}" | status: missing | why: synthetic scripted item`;
  const changedLine = `literal: /demo_one | source: request | source_quote: "${Q}" | status: changed | package_value: /demo-one | why: synthetic scripted item`;
  const item = (literal, status, source = 'request') => ({literal, source, source_quote: Q, status, package_value: '', why: 'synthetic scripted item'});
  const event = (items, extra = {}) => ({event: 'literal_check', status: 'compared', present: 0, missing: 0, changed: 0, superseded: 0, informal: 0, secret: 0, unverified: 0, items, ...extra});
  const stdout = 'Intent: /x/.kogen/intents/is-even/intent.md\nNext: kogen intent approve is-even\n';
  const base = {answer: '', exit: 0, err: '<nil>', published: true, approved: false, stdout, stderr: 'kogen: checking the required literals (0 s)\n', body: '', input: ''};
  const known = {
    omitted: {...base, case: 'omitted', request: R, first: NONE, second: BOTH, first_intent: intent(NONE), intent: intent(BOTH), shaper: 2, literal: 2,
      feedback: [[head, miss('/demo_one', 'request', Q), miss('/demo-two', 'request', Q)].join('\n')],
      first_feedback: [head, miss('/demo_one', 'request', Q), miss('/demo-two', 'request', Q)].join('\n'), gates: ['literal_check'],
      events: [event([item('/demo_one', 'missing'), item('/demo-two', 'missing'), item('/demo_old', 'superseded')]), event([item('/demo_one', 'present'), item('/demo-two', 'present'), item('/demo_old', 'superseded')])]},
    substituted: {...base, case: 'substituted', request: R, first: SUB, second: BOTH, first_intent: intent(SUB), intent: intent(BOTH), shaper: 2, literal: 2,
      feedback: [head.replace(': 2 ', ': 1 ') + '\n' + changedLine], first_feedback: head.replace(': 2 ', ': 1 ') + '\n' + changedLine, gates: ['literal_check'],
      events: [event([{...item('/demo_one', 'changed'), package_value: '/demo-one'}, item('/demo-two', 'present'), item('/demo_old', 'superseded')])]},
    complete: {...base, case: 'complete', request: R, first: FRIENDLY, second: '', first_intent: intent(FRIENDLY), intent: intent(FRIENDLY), shaper: 1, literal: 1,
      feedback: [], first_feedback: '', gates: [], body: `You check that a candidate Intent carries the required literal values.\n${S1}\n${S2}\n${S3}\n${S4}\n${input(R, '(none)', FRIENDLY)}`, input: input(R, '(none)', FRIENDLY),
      events: [event([item('/demo_one', 'present'), item('/demo-two', 'present'), item('/demo_old', 'superseded'), item('/demo_three', 'unverified')], {present: 2, superseded: 1, unverified: 1})]},
    informal: {...base, case: 'informal', request: R + '\n' + INFORMAL, first: BOTH, second: '', first_intent: intent(BOTH), intent: intent(BOTH), shaper: 1, literal: 1,
      feedback: [], first_feedback: '', gates: [],
      events: [event([item('/demo_one', 'present'), item('/demo-two', 'present'), item('/demo_old', 'superseded'), {...item('/legacy-demo', 'informal'), source_quote: INFORMAL}], {present: 2, superseded: 1, informal: 1})]},
    answer: {...base, case: 'answer', request: 'Add a port example to the README.', answer: '1: ' + PANSWER, first: PNONE, second: PORT, first_intent: intent(PNONE), intent: intent(PORT), shaper: 2, literal: 2,
      feedback: [head.replace(': 2 ', ': 1 ') + '\n' + miss('7443', 'answer 1', PANSWER)], first_feedback: head.replace(': 2 ', ': 1 ') + '\n' + miss('7443', 'answer 1', PANSWER), gates: ['literal_check'],
      input: input('Add a port example to the README.', 'answer 1: ' + PANSWER, PNONE),
      events: [event([item('7443', 'missing', 'answer 1')]), event([item('7443', 'present', 'answer 1')])]},
    unknown: {...base, case: 'unknown', request: R, answer: '1: ' + PANSWER, first: NONE, second: BOTH, first_intent: intent(NONE), intent: intent(NONE), shaper: 1, literal: 0,
      feedback: [], first_feedback: '', gates: [], stderr: 'kogen: request literal check unknown: the original request is not available; required literals were not compared (0 s)\n',
      events: [{event: 'literal_check', status: 'unknown', reason: 'the original request is not available'}]},
    secret: {...base, case: 'secret', request: R + '\nSign in with password: ' + SECRET + '.', first: ONE, second: BOTH, first_intent: intent(ONE), intent: intent(BOTH), shaper: 2, literal: 2,
      feedback: [head.replace(': 2 ', ': 1 ') + '\n' + miss('/demo-two', 'request', Q)], first_feedback: head.replace(': 2 ', ': 1 ') + '\n' + miss('/demo-two', 'request', Q), gates: ['literal_check'],
      events: [event([item('/demo_one', 'present'), item('/demo-two', 'missing'), item('/demo_old', 'superseded'), {source: 'request', status: 'secret'}], {secret: 1}),
        event([item('/demo_one', 'present'), item('/demo-two', 'present'), item('/demo_old', 'superseded'), {source: 'request', status: 'secret'}], {secret: 1})]},
  };
  const good = [
    s => checks.refused(s.omitted, 'literal: /demo_one', 'source: request', 'status: missing'),
    s => checks.refused(s.omitted, 'literal: /demo-two', 'source: request', 'status: missing'),
    s => checks.publishedCorrected(s.omitted),
    s => checks.intentContains(s.omitted, '/demo_one'),
    s => checks.intentContains(s.omitted, '/demo-two'),
    s => checks.literalRequests(s.omitted, 2),
    s => checks.refused(s.substituted, 'literal: /demo_one', 'status: changed', 'package_value: /demo-one'),
    s => checks.refusedWithout(s.substituted, 'literal: /demo-two'),
    s => checks.publishedCorrected(s.substituted),
    s => checks.publishedFirst(s.complete),
    s => checks.literalRequests(s.complete, 1),
    s => checks.noFeedback(s.complete, HEADLINE),
    s => checks.recorded(s.complete, '/demo_one', 'present'),
    s => checks.recorded(s.complete, '/demo-two', 'present'),
    s => checks.recorded(s.complete, '/demo_old', 'superseded'),
    s => checks.recorded(s.complete, '/demo_three', 'unverified'),
    s => checks.intentAsSubmitted(s.complete),
    s => checks.carries(s.complete),
    s => checks.instructions(s.complete, S1),
    s => checks.instructions(s.complete, S2),
    s => checks.instructions(s.complete, S3),
    s => checks.instructions(s.complete, S4),
    s => checks.recorded(s.informal, '/legacy-demo', 'informal'),
    s => checks.noFeedback(s.informal, HEADLINE),
    s => checks.publishedFirst(s.informal),
    s => checks.refused(s.answer, 'literal: 7443', 'source: answer 1', 'status: missing'),
    s => checks.carriesText(s.answer, 'answer 1: ' + PANSWER),
    s => checks.publishedCorrected(s.answer),
    s => checks.literalRequests(s.unknown, 0),
    s => checks.unknown(s.unknown),
    s => checks.stderrHas(s.unknown, 'kogen: request literal check unknown: the original request is not available; required literals were not compared'),
    s => checks.noFeedback(s.unknown, HEADLINE),
    s => checks.publishedFirst(s.unknown),
    s => checks.refused(s.secret, 'literal: /demo-two', 'source: request', 'status: missing'),
    s => checks.notExposed(s.secret, SECRET),
    s => checks.secretCount(s.secret, 1),
    s => checks.publishedCorrected(s.secret),
  ];
  const changed = (name, change) => {
    const s = JSON.parse(JSON.stringify(known));
    change(s[name]);
    return s;
  };
  // unchangedTree is what the tree without a literal check shows: the first submission is
  // published with no literal check request, event or feedback.
  const unchangedTree = o => {
    o.literal = 0; o.events = []; o.body = ''; o.input = ''; o.feedback = []; o.first_feedback = ''; o.gates = []; o.shaper = 1;
    o.intent = intent(o.first); o.stderr = 'kogen: checking the stated decisions (0 s)\n';
  };
  const bad = [
    [changed('omitted', unchangedTree), s => checks.refused(s.omitted, 'literal: /demo_one', 'source: request', 'status: missing')],
    [changed('omitted', unchangedTree), s => checks.publishedCorrected(s.omitted)],
    [changed('omitted', unchangedTree), s => checks.intentContains(s.omitted, '/demo_one')],
    [changed('omitted', unchangedTree), s => checks.literalRequests(s.omitted, 2)],
    [changed('omitted', o => { o.first_feedback = o.first_feedback.split('\n').filter(l => !l.includes('/demo-two')).join('\n'); }), s => checks.refused(s.omitted, 'literal: /demo-two', 'source: request', 'status: missing')],
    [changed('omitted', o => { o.first_feedback = o.first_feedback.replace(HEADLINE, 'Readiness: '); }), s => checks.refused(s.omitted, 'literal: /demo_one', 'source: request', 'status: missing')],
    [changed('omitted', o => { o.gates = ['decision_check']; }), s => checks.refused(s.omitted, 'literal: /demo_one', 'source: request', 'status: missing')],
    [changed('omitted', o => { o.first_feedback = o.first_feedback.replace('| source: request | source_quote: "' + Q + '" | status: missing | why: synthetic scripted item\nliteral: /demo-two', '\nliteral: /demo-two'); }), s => checks.refused(s.omitted, 'literal: /demo_one', 'source: request', 'status: missing')],
    [changed('omitted', o => { o.shaper = 3; }), s => checks.publishedCorrected(s.omitted)],
    [changed('omitted', o => { o.published = false; }), s => checks.publishedCorrected(s.omitted)],
    [changed('substituted', unchangedTree), s => checks.refused(s.substituted, 'literal: /demo_one', 'status: changed', 'package_value: /demo-one')],
    [changed('substituted', unchangedTree), s => checks.refusedWithout(s.substituted, 'literal: /demo-two')],
    [changed('substituted', o => { o.first_feedback = o.first_feedback.replace('status: changed | package_value: /demo-one', 'status: missing'); }), s => checks.refused(s.substituted, 'literal: /demo_one', 'status: changed', 'package_value: /demo-one')],
    [changed('substituted', o => { o.first_feedback += '\n' + miss('/demo-two', 'request', Q); }), s => checks.refusedWithout(s.substituted, 'literal: /demo-two')],
    [changed('complete', o => { o.shaper = 2; o.gates = ['literal_check']; o.feedback = [head]; }), s => checks.publishedFirst(s.complete)],
    [changed('complete', o => { o.gates = ['literal_check']; o.feedback = [head + '\n' + miss('/demo_old', 'request', Q)]; }), s => checks.noFeedback(s.complete, HEADLINE)],
    [changed('complete', o => { o.literal = 2; }), s => checks.literalRequests(s.complete, 1)],
    [changed('complete', unchangedTree), s => checks.literalRequests(s.complete, 1)],
    [changed('complete', unchangedTree), s => checks.recorded(s.complete, '/demo_one', 'present')],
    [changed('complete', unchangedTree), s => checks.carries(s.complete)],
    [changed('complete', unchangedTree), s => checks.instructions(s.complete, S1)],
    [changed('complete', o => { o.events[0].items[2].status = 'missing'; }), s => checks.recorded(s.complete, '/demo_old', 'superseded')],
    [changed('complete', o => { o.events[0].items[3].status = 'missing'; }), s => checks.recorded(s.complete, '/demo_three', 'unverified')],
    [changed('complete', o => { o.events[0].status = 'unknown'; }), s => checks.recorded(s.complete, '/demo_one', 'present')],
    [changed('complete', o => { o.intent = o.intent.replace(' in a welcoming tone', ''); }), s => checks.intentAsSubmitted(s.complete)],
    [changed('complete', o => { o.approved = true; }), s => checks.intentAsSubmitted(s.complete)],
    [changed('complete', o => { o.input = o.input.replace('# Answers', '# Rounds'); }), s => checks.carries(s.complete)],
    [changed('complete', o => { o.input = o.input.replace('## features/steps/is-even__steps.js', '## steps'); }), s => checks.carries(s.complete)],
    [changed('complete', o => { o.input = o.input.replace(R + '\n', 'another request\n'); }), s => checks.carries(s.complete)],
    [changed('complete', o => { o.body = o.body.replace(S2, ''); }), s => checks.instructions(s.complete, S2)],
    [changed('complete', o => { o.body = o.body.replace(S3, ''); }), s => checks.instructions(s.complete, S3)],
    [changed('complete', o => { o.body = o.body.replace(S4, ''); }), s => checks.instructions(s.complete, S4)],
    [changed('informal', unchangedTree), s => checks.recorded(s.informal, '/legacy-demo', 'informal')],
    [changed('informal', o => { o.events[0].items[3].status = 'missing'; }), s => checks.recorded(s.informal, '/legacy-demo', 'informal')],
    [changed('informal', o => { o.gates = ['literal_check']; o.feedback = [head + '\n' + miss('/legacy-demo', 'request', INFORMAL)]; }), s => checks.noFeedback(s.informal, HEADLINE)],
    [changed('informal', o => { o.shaper = 2; }), s => checks.publishedFirst(s.informal)],
    [changed('answer', unchangedTree), s => checks.refused(s.answer, 'literal: 7443', 'source: answer 1', 'status: missing')],
    [changed('answer', unchangedTree), s => checks.carriesText(s.answer, 'answer 1: ' + PANSWER)],
    [changed('answer', unchangedTree), s => checks.publishedCorrected(s.answer)],
    [changed('answer', o => { o.first_feedback = o.first_feedback.replace('source: answer 1', 'source: request'); }), s => checks.refused(s.answer, 'literal: 7443', 'source: answer 1', 'status: missing')],
    [changed('answer', o => { o.input = o.input.replace('answer 1: ' + PANSWER, '(none)'); }), s => checks.carriesText(s.answer, 'answer 1: ' + PANSWER)],
    [changed('unknown', unchangedTree), s => checks.unknown(s.unknown)],
    [changed('unknown', unchangedTree), s => checks.stderrHas(s.unknown, 'kogen: request literal check unknown: the original request is not available; required literals were not compared')],
    [changed('unknown', o => { o.literal = 1; }), s => checks.literalRequests(s.unknown, 0)],
    [changed('unknown', o => { o.events.push(event([])); }), s => checks.unknown(s.unknown)],
    [changed('unknown', o => { o.events = [event([])]; }), s => checks.unknown(s.unknown)],
    [changed('unknown', o => { o.events[0].reason = 'the request is empty'; }), s => checks.unknown(s.unknown)],
    [changed('unknown', o => { o.gates = ['literal_check']; o.feedback = [head]; }), s => checks.noFeedback(s.unknown, HEADLINE)],
    [changed('unknown', o => { o.published = false; }), s => checks.publishedFirst(s.unknown)],
    [changed('secret', unchangedTree), s => checks.refused(s.secret, 'literal: /demo-two', 'source: request', 'status: missing')],
    [changed('secret', unchangedTree), s => checks.secretCount(s.secret, 1)],
    [changed('secret', unchangedTree), s => checks.publishedCorrected(s.secret)],
    [changed('secret', o => { o.feedback[0] += '\n' + miss(SECRET, 'request', 'password: ' + SECRET); }), s => checks.notExposed(s.secret, SECRET)],
    [changed('secret', o => { o.events[0].items[3] = {...item(SECRET, 'secret'), source_quote: 'password: ' + SECRET}; }), s => checks.notExposed(s.secret, SECRET)],
    [changed('secret', o => { o.stderr += 'literal ' + SECRET + '\n'; }), s => checks.notExposed(s.secret, SECRET)],
    [changed('secret', o => { o.intent += SECRET + '\n'; }), s => checks.notExposed(s.secret, SECRET)],
    [changed('secret', o => { o.events[0].secret = 0; }), s => checks.secretCount(s.secret, 1)],
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
  const P = 'k-shape-request-literal-coverage: ';
  When(P + 'the {string} case is shaped', {timeout: 540000}, function (name) { this.krlc = shapeCase(name); });
  Then(P + 'the Intent is published from the first submission', function () { checks.publishedFirst(this.krlc); });
  Then(P + 'the Intent is published from the corrected submission', function () { checks.publishedCorrected(this.krlc); });
  Then(P + 'exactly {int} literal check request(s) is/are made', function (n) { checks.literalRequests(this.krlc, n); });
  Then(P + 'the first submission is refused by the literal check with a line containing {string}, {string} and {string}', function (a, b, c) { checks.refused(this.krlc, a, b, c); });
  Then(P + 'the first submission is refused by the literal check with no line containing {string}', function (text) { checks.refusedWithout(this.krlc, text); });
  Then(P + 'no feedback to the shaper contains {string}', function (text) { checks.noFeedback(this.krlc, text); });
  Then(P + 'the published intent.md contains {string}', function (text) { checks.intentContains(this.krlc, text); });
  Then(P + 'the published intent.md is the submitted one and the Intent is not approved', function () { checks.intentAsSubmitted(this.krlc); });
  Then(P + 'the literal check recorded {string} as {string}', function (literal, status) { checks.recorded(this.krlc, literal, status); });
  Then(P + 'the literal check recorded {int} secret item(s)', function (n) { checks.secretCount(this.krlc, n); });
  Then(P + 'nothing Shape showed or recorded contains {string}', function (text) { checks.notExposed(this.krlc, text); });
  Then(P + 'the literal check recorded the original request as unknown', function () { checks.unknown(this.krlc); });
  Then(P + 'stderr contains {string}', function (text) { checks.stderrHas(this.krlc, text); });
  Then(P + 'the literal check request carries the request, the answers and the candidate package', function () { checks.carries(this.krlc); });
  Then(P + 'the literal check request carries {string}', function (text) { checks.carriesText(this.krlc, text); });
  Then(P + 'the literal check request includes:', function (text) { checks.instructions(this.krlc, text); });
}
