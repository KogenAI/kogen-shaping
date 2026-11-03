'use strict';
// Acceptance of k-shape-fixture-self-proof. The When step runs one scripted Shape journey of the
// request's synthetic repository: it compiles the shape package's tests with the harness of
// k-shape-fixture-self-proof__harness.js (through a Go overlay), runs
// TestKShapeFixtureSelfProofAcceptance with the journey's submissions and reads what the shaper and
// the run log saw. The Then steps judge that. Run this file directly with good or bad to check the
// judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeFixtureSelfProofAcceptance';

// ---- the request's synthetic repository and candidate packages ----

const COUNT = 'module.exports = function count(items) { return 0; };\n';
const FIXTURECHECK = String.raw`const fs = require('node:fs');
const value = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
if (value.kind !== 'packet') {
  process.stderr.write('missing kind=packet\n');
  process.exit(2);
}
if (!Array.isArray(value.items)) {
  process.stderr.write('items must be an array\n');
  process.exit(2);
}
process.stdout.write('fixture ok\n');
`;
const BASE = [{path: 'count.js', content: COUNT}, {path: 'fixturecheck.js', content: FIXTURECHECK}];
const REQUEST = 'Make count.js return the actual number of items.';

const INTENT_PATH = '.kogen/intents/fixture-proof/intent.md';
const FEATURE_PATH = '.kogen/intents/fixture-proof/acceptance.feature';
const STEPS_PATH = 'features/steps/fixture-proof__steps.js';
const INTENT = [
  '# Count the items of a packet', '',
  'Slug: fixture-proof',
  'Model: not applicable — a function without state',
  'Acceptance:',
  '- ' + FEATURE_PATH,
  '- ' + STEPS_PATH, '',
  'count.js returns the actual number of items.', '',
  '## Decisions', '- The output format follows the request. (source: request)', '',
  '## Source comparison', '- The request asks for this behaviour; the feature covers it.', '',
  '## Tests', '- The requested behaviour: the acceptance scenarios (scenario)', '',
  '## Feasibility', '- none', '',
].join('\n');

const FEATURE = [
  'Feature: Count items',
  '  Scenario: Count a prepared packet',
  '    Given fixture-proof: a packet',
  '    When fixture-proof: I count its items',
  '    Then fixture-proof: the count is 2', '',
].join('\n');

const HEAD = [
  "const assert = require('node:assert/strict');",
  "const fs = require('node:fs');",
  "const os = require('node:os');",
  "const path = require('node:path');",
  "const { spawnSync } = require('node:child_process');",
  "const { Given, When, Then, After } = require('@cucumber/cucumber');",
];
// builder is the request's fixture builder for one packet literal, without its enclosing step.
const builder = (packet) => [
  "  this.tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'fixture-proof-'));",
  '  this.packet = ' + packet + ';',
  "  const file = path.join(this.tmp, 'packet.json');",
  '  fs.writeFileSync(file, JSON.stringify(this.packet));',
  "  const r = spawnSync(process.execPath, ['fixturecheck.js', file], {",
  "    cwd: process.cwd(), encoding: 'utf8'",
  '  });',
  "  assert.equal(r.status, 0, `fixturecheck exit=${r.status}\\n${r.stdout}${r.stderr}`);",
];
const COUNT_CALL = "  this.count = require(path.resolve('count.js'))(this.packet.items);";
const THEN_COUNT = [
  "Then('fixture-proof: the count is {int}', function (expected) {",
  '  assert.equal(this.count, expected);',
  '});',
];
const AFTER = [
  'After(function () {',
  '  if (this.tmp) {',
  "    fs.unlinkSync(path.join(this.tmp, 'packet.json'));",
  '    fs.rmdirSync(this.tmp);',
  '  }',
  '});',
];
const lines = (...parts) => parts.flat().join('\n') + '\n';

// steps is the request's candidate step file with the given packet literal in its Given.
const steps = (packet) => lines(HEAD,
  "Given('fixture-proof: a packet', function () {", builder(packet), '});',
  "When('fixture-proof: I count its items', function () {", COUNT_CALL, '});',
  THEN_COUNT, AFTER);
const BROKEN = steps("{ items: ['a', 'b'] }");
const REPAIRED = steps("{ kind: 'packet', items: ['a', 'b'] }");
const ITEMS_ZERO = steps("{ kind: 'packet', items: 0 }");
// MOVED puts the same failing builder at the start of the When body, followed by the count call.
const MOVED = lines(HEAD,
  "Given('fixture-proof: a packet', function () {", '});',
  "When('fixture-proof: I count its items', function () {", builder("{ items: ['a', 'b'] }"), COUNT_CALL, '});',
  THEN_COUNT, AFTER);

// The negative-input control: the malformed packet is written successfully and its rejection is
// the asserted outcome.
const NEGATIVE_FEATURE = [
  'Feature: Reject a malformed packet',
  '  Scenario: A packet whose items are not an array is rejected',
  '    Given fixture-proof: a packet whose items are 0',
  '    When fixture-proof: I check the packet',
  '    Then fixture-proof: the check exits 2 with "items must be an array"', '',
].join('\n');
const NEGATIVE = lines(HEAD,
  "Given('fixture-proof: a packet whose items are 0', function () {",
  "  this.tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'fixture-proof-'));",
  "  fs.writeFileSync(path.join(this.tmp, 'packet.json'), JSON.stringify({ kind: 'packet', items: 0 }));",
  '});',
  "When('fixture-proof: I check the packet', function () {",
  "  this.check = spawnSync(process.execPath, ['fixturecheck.js', path.join(this.tmp, 'packet.json')], {",
  "    cwd: process.cwd(), encoding: 'utf8'",
  '  });',
  '});',
  "Then('fixture-proof: the check exits {int} with {string}', function (status, message) {",
  '  assert.equal(this.check.status, status);',
  "  assert.equal(this.check.stdout + this.check.stderr, message + '\\n');",
  '});',
  AFTER);

// The skipped-builder package: the repaired scenario with a broken second builder after its Then.
// The builder records in FIXTURE_PROOF_TRACE that its body ran.
const SKIPPED_FEATURE = FEATURE + '    And fixture-proof: a second packet without its kind\n';
const SKIPPED = REPAIRED + lines(
  "Then('fixture-proof: a second packet without its kind', function () {",
  "  fs.appendFileSync(process.env.FIXTURE_PROOF_TRACE, 'second packet built\\n');",
  "  const file = path.join(this.tmp, 'second.json');",
  "  fs.writeFileSync(file, JSON.stringify({ items: ['c'] }));",
  "  const r = spawnSync(process.execPath, ['fixturecheck.js', file], {",
  "    cwd: process.cwd(), encoding: 'utf8'",
  '  });',
  '  fs.unlinkSync(file);',
  "  assert.equal(r.status, 0, `fixturecheck exit=${r.status}\\n${r.stdout}${r.stderr}`);",
  '});');

const file = (p, content) => ({path: p, content});
const whole = (feature, stepFile) => [file(INTENT_PATH, INTENT), file(FEATURE_PATH, feature), file(STEPS_PATH, stepFile)];
const JOURNEYS = {
  'missing-kind': [whole(FEATURE, BROKEN)],
  'repair': [whole(FEATURE, BROKEN), [file(FEATURE_PATH, FEATURE)], [file(STEPS_PATH, REPAIRED)], [file(STEPS_PATH, REPAIRED)]],
  'items-not-array': [whole(FEATURE, ITEMS_ZERO)],
  'negative-input': [whole(NEGATIVE_FEATURE, NEGATIVE)],
  'moved-builder': [whole(FEATURE, BROKEN), [file(STEPS_PATH, MOVED)]],
  'skipped-builder': [whole(SKIPPED_FEATURE, SKIPPED), [file(FEATURE_PATH, SKIPPED_FEATURE)]],
};

// ---- running a journey ----

let built;

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once per process.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-fixture-self-proof-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_shape_fixture_self_proof_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-fixture-self-proof__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_shape_fixture_self_proof_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env(), encoding: 'utf8', timeout: 280000});
  built = r.status === 0 && !r.error ? {bin, dir} : {error: `${TEST}: go test -c -vet=off -overlay ./shape failed: exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`};
  return built;
}

// journey runs the harness for one journey and returns its observation.
function journey(name) {
  const submissions = JOURNEYS[name];
  if (!submissions) throw new Error(`${TEST}: unknown journey ${name}`);
  const b = build();
  if (b.error) throw new Error(b.error);
  const input = path.join(b.dir, name + '-journey.json');
  const out = path.join(b.dir, name + '.json');
  fs.writeFileSync(input, JSON.stringify({request: REQUEST, base: BASE, submissions}));
  fs.rmSync(out, {force: true});
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=400s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KFSP_JOURNEY: input, KFSP_OUT: out}), encoding: 'utf8', timeout: 420000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  const ran = `${b.bin} ${args.join(' ')} (journey ${name})`;
  if (r.error || r.status !== 0) throw new Error(`${TEST} journey ${name} failed: ${ran} exit=${r.status} error=${r.error || ''}\n${text}`);
  if (!text.includes('--- PASS: ' + TEST)) throw new Error(`${TEST} journey ${name} did not run (absent, skipped or zero runs): ${ran} exit=${r.status}\n${text}`);
  if (!fs.existsSync(out)) throw new Error(`${TEST} journey ${name} wrote no observation to ${out}: ${ran}\n${text}`);
  return {journey: name, ...JSON.parse(fs.readFileSync(out, 'utf8'))};
}

// ---- the judgments of the Then steps ----

const OBSERVATIONS = 'Inspect whether fixtures, inputs and commands reached the intended behavior';

function where(obs) { return `${TEST} journey ${obs.journey}: observation ${JSON.stringify(obs)}`; }

function feedbackOf(obs, n) {
  const text = obs.feedback && obs.feedback['c' + n];
  assert.equal(typeof text, 'string', `${where(obs)}: Kogen returned no answer to submission ${n}`);
  return text;
}

function refusalOf(obs, n) {
  return (obs.refusals || []).filter(r => r.attempt === n);
}

function scenarioOf(obs, name) {
  assert.ok(Array.isArray(obs.record), `${where(obs)}: no readiness record was kept`);
  const found = obs.record.filter(s => s.name === name);
  assert.equal(found.length, 1, `${where(obs)}: the readiness record must hold scenario ${JSON.stringify(name)} once`);
  return found[0];
}

const judge = {
  refused(obs, n, headline) {
    const text = feedbackOf(obs, n);
    assert.deepEqual(refusalOf(obs, n).map(r => r.gate), ['preparation'], `${where(obs)}: submission ${n} must be refused once, at the preparation gate`);
    assert.equal(text.split('\n')[0], headline, `${where(obs)}: the first line of the answer to submission ${n}`);
  },
  linesInOrder(obs, n, want) {
    const got = feedbackOf(obs, n).split('\n');
    let at = 0;
    for (const line of want) {
      const i = got.indexOf(line, at);
      assert.ok(i >= 0, `${where(obs)}: the answer to submission ${n} lacks, after its line ${at + 1}, the line ${JSON.stringify(line)}`);
      at = i + 1;
    }
  },
  excludes(obs, n, text) {
    assert.ok(!feedbackOf(obs, n).includes(text), `${where(obs)}: the answer to submission ${n} must not contain ${JSON.stringify(text)}`);
  },
  endsWith(obs, n, text) {
    assert.ok(feedbackOf(obs, n).endsWith(text), `${where(obs)}: the answer to submission ${n} must end with ${JSON.stringify(text)}`);
  },
  observations(obs, n) {
    const text = feedbackOf(obs, n);
    assert.deepEqual(refusalOf(obs, n), [], `${where(obs)}: submission ${n} must not be refused`);
    assert.ok(text.startsWith(OBSERVATIONS), `${where(obs)}: the answer to submission ${n} must return accounted red observations`);
  },
  refusals(obs, spec) {
    const want = spec === 'none' ? [] : spec.split(' ');
    assert.deepEqual((obs.refusals || []).map(r => `${r.attempt}:${r.gate}`), want, `${where(obs)}: the refused submissions and their gates`);
  },
  published(obs) {
    assert.equal(obs.exit, 0, `${where(obs)}: Shape must exit 0`);
    assert.equal(obs.stop, '', `${where(obs)}: Shape must not stop with an error`);
    assert.ok(Array.isArray(obs.record), `${where(obs)}: publication keeps the readiness record`);
  },
  unpublished(obs) {
    assert.ok(obs.exit !== 0 || obs.stop !== '', `${where(obs)}: Shape must not succeed`);
    assert.equal(obs.record, null, `${where(obs)}: no readiness record may be written`);
  },
  status(obs, name, status) {
    assert.equal(scenarioOf(obs, name).status, status, `${where(obs)}: the recorded status of ${JSON.stringify(name)}`);
  },
  unexercised(obs, name, want) {
    assert.deepEqual(scenarioOf(obs, name).unexercised, want, `${where(obs)}: the recorded unexercised steps of ${JSON.stringify(name)}`);
  },
  noTrace(obs) {
    assert.equal(obs.trace, '', `${where(obs)}: a skipped step body ran`);
  },
};

if (require.main === module) {
  // Known observations: what a correct implementation shows for each kind of journey, and wrong
  // variants each judgment must reject.
  const H = 'Acceptance preparation failed on the unchanged tree: 1 scenario(s) fail before they reach the behaviour under test, so this is not missing-feature red; nothing was published.';
  const head = H + ' A failed Given or Background step or Before hook, or a When or Then step that fails exactly as a refused preparation did, is a defect of the acceptance package: repair the preparation so that it succeeds on the unchanged tree and the consumer of its fixture accepts it. Keep the scenario and its outcome assertions, and do not change production code to accept a malformed fixture. Resubmitting or acknowledging this package unchanged is refused again.';
  const item = '- Count a prepared packet (acceptance.feature line 2): acceptance.feature line 3: Given fixture-proof: a packet failed (a Given step); acceptance exit 1';
  const output = ['  AssertionError [ERR_ASSERTION]: fixturecheck exit=2', '  missing kind=packet', '', '', '  2 !== 0', '', '      at World.<anonymous> (/tmp/r/features/steps/fixture-proof__steps.js:15:10)'];
  const notRun = ['  not run: acceptance.feature line 4: When fixture-proof: I count its items', '  not run: acceptance.feature line 5: Then fixture-proof: the count is 2'];
  const refusal = [head, item, ...output, ...notRun].join('\n');
  const unchanged = '\nUnchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.';
  const observed = OBSERVATIONS + '. Repair demonstrated contract defects.\n\nReadiness observations\nScenario: Count a prepared packet (acceptance.feature line 2)\nacceptance.feature line 5: Then fixture-proof: the count is 2\nAssertionError [ERR_ASSERTION]: Expected values to be strictly equal:\n\n0 !== 2\n';
  const refused = {journey: 'missing-kind', exit: 4, stop: 'kogen: shape/provider_failed', feedback: {c1: refusal}, refusals: [{attempt: 1, gate: 'preparation'}], record: null, trace: ''};
  const repair = {journey: 'repair', exit: 0, stop: '', feedback: {c1: refusal, c2: refusal + unchanged, c3: observed},
    refusals: [{attempt: 1, gate: 'preparation'}, {attempt: 2, gate: 'preparation'}],
    record: [{name: 'Count a prepared packet', status: 'failed', unexercised: []}], trace: ''};
  const skipped = {journey: 'skipped-builder', exit: 0, stop: '', feedback: {c1: observed + '\nacceptance.feature line 6: And fixture-proof: a second packet without its kind'}, refusals: [],
    record: [{name: 'Count a prepared packet', status: 'failed', unexercised: ['acceptance.feature line 6: And fixture-proof: a second packet without its kind']}], trace: ''};
  const runs = [
    o => judge.refused(o.refused, 1, head), o => judge.linesInOrder(o.refused, 1, [item, ...output.slice(0, 2), ...notRun]),
    o => judge.excludes(o.refused, 1, 'items must be an array'), o => judge.refusals(o.refused, '1:preparation'), o => judge.unpublished(o.refused),
    o => judge.refused(o.repair, 2, head), o => judge.endsWith(o.repair, 2, unchanged.slice(1)), o => judge.observations(o.repair, 3),
    o => judge.linesInOrder(o.repair, 3, ['acceptance.feature line 5: Then fixture-proof: the count is 2', '0 !== 2']),
    o => judge.refusals(o.repair, '1:preparation 2:preparation'), o => judge.published(o.repair), o => judge.status(o.repair, 'Count a prepared packet', 'failed'),
    o => judge.observations(o.skipped, 1), o => judge.refusals(o.skipped, 'none'), o => judge.published(o.skipped), o => judge.status(o.skipped, 'Count a prepared packet', 'failed'),
    o => judge.unexercised(o.skipped, 'Count a prepared packet', ['acceptance.feature line 6: And fixture-proof: a second packet without its kind']), o => judge.noTrace(o.skipped),
  ];
  const all = o => { for (const run of runs) run(o); };
  const good = {refused, repair, skipped};
  const with_ = (key, change) => ({...good, [key]: {...good[key], ...change}});
  const answer = (key, n, text) => with_(key, {feedback: {...good[key].feedback, ['c' + n]: text}});
  const bad = {
    'refused at another gate': with_('refused', {refusals: [{attempt: 1, gate: 'readiness'}]}),
    'refused twice': with_('refused', {refusals: [{attempt: 1, gate: 'preparation'}, {attempt: 1, gate: 'contract_review'}]}),
    'not refused': with_('refused', {refusals: []}),
    'another headline': answer('refused', 1, refusal.replace('so this is not missing-feature red', 'so this is red')),
    'accounted red instead of a refusal': answer('refused', 1, observed),
    'the failed step not named': answer('refused', 1, refusal.replace('acceptance.feature line 3: Given fixture-proof: a packet failed', 'a step failed')),
    'the When reported as the failed step': answer('refused', 1, refusal.replace('line 3: Given fixture-proof: a packet', 'line 4: When fixture-proof: I count its items')),
    'no acceptance exit': answer('refused', 1, refusal.replace('; acceptance exit 1', '')),
    'no exit 2': answer('refused', 1, refusal.replace('fixturecheck exit=2', 'fixturecheck failed')),
    'no consumer message': answer('refused', 1, refusal.replace('  missing kind=packet\n', '')),
    'skipped steps not listed as not run': answer('refused', 1, [head, item, ...output].join('\n')),
    'skipped steps reported as passed': answer('refused', 1, [head, item, ...output, ...notRun.map(l => l.replace('not run', 'passed'))].join('\n')),
    'not-run lines before the output': answer('refused', 1, [head, item, ...notRun, ...output].join('\n')),
    'the other schema message too': answer('refused', 1, refusal + '\n  items must be an array'),
    'a readiness record of the refused package': with_('refused', {record: []}),
    'a refused package that Shape published': with_('refused', {exit: 0, stop: ''}),
    'an unchanged acknowledgment accepted': answer('repair', 2, observed),
    'an unchanged acknowledgment without the unchanged line': answer('repair', 2, refusal),
    'the repaired package refused': with_('repair', {refusals: [{attempt: 1, gate: 'preparation'}, {attempt: 2, gate: 'preparation'}, {attempt: 3, gate: 'preparation'}]}),
    'the repaired package answered with a refusal': answer('repair', 3, refusal),
    'repaired red without its actual value': answer('repair', 3, observed.replace('0 !== 2', '')),
    'repaired red at another step': answer('repair', 3, observed.replace('line 5: Then fixture-proof: the count is 2', 'line 3: Given fixture-proof: a packet')),
    'the repaired package not published': with_('repair', {exit: 1, stop: 'kogen: intent/not_ready'}),
    'published without a readiness record': with_('repair', {record: null}),
    'the legitimate red recorded as passed': with_('repair', {record: [{name: 'Count a prepared packet', status: 'passed', unexercised: []}]}),
    'the skipped builder refused': with_('skipped', {refusals: [{attempt: 1, gate: 'preparation'}]}),
    'the skipped builder executed': with_('skipped', {trace: 'second packet built\n'}),
    'the skipped builder recorded as exercised': with_('skipped', {record: [{name: 'Count a prepared packet', status: 'failed', unexercised: []}]}),
    'the skipped builder scenario recorded as passed': with_('skipped', {record: [{name: 'Count a prepared packet', status: 'passed', unexercised: ['acceptance.feature line 6: And fixture-proof: a second packet without its kind']}]}),
  };
  let accepted = '';
  try { all(good); } catch (e) { accepted = 'good: ' + e.message; }
  const mode = process.argv[2];
  if (mode === 'good') {
    if (process.argv.includes('--diagnostic') && accepted) console.error(accepted);
    console.log(accepted ? 'rejected' : 'accepted');
    process.exitCode = accepted ? 1 : 0;
  } else if (mode === 'bad') {
    const wrongly = Object.keys(bad).filter(name => { try { all(bad[name]); return true; } catch { return false; } });
    if (process.argv.includes('--diagnostic') && wrongly.length) console.error('accepted: ' + wrongly.join(', '));
    console.log(wrongly.length || accepted ? 'accepted' : 'rejected');
    process.exitCode = wrongly.length || accepted ? 0 : 1;
  } else {
    console.error('usage: node features/steps/k-shape-fixture-self-proof__steps.js good|bad');
    process.exitCode = 2;
  }
} else {
  const {Given, When, Then} = require('@cucumber/cucumber');
  const P = 'k-shape-fixture-self-proof: ';
  Given(P + 'the preparation refusal begins with this line:', function (line) {
    this.kfspHeadline = line;
  });
  When(P + 'the scripted shaper runs the {string} journey', {timeout: 720000}, function (name) {
    this.kfsp = journey(name);
  });
  Then(P + 'submission {int} is refused at preparation', function (n) {
    judge.refused(this.kfsp, n, this.kfspHeadline);
  });
  Then(P + 'the answer to submission {int} has these lines in order:', function (n, text) {
    judge.linesInOrder(this.kfsp, n, text.split('\n'));
  });
  Then(P + 'the answer to submission {int} does not contain {string}', function (n, text) {
    judge.excludes(this.kfsp, n, text);
  });
  Then(P + 'the answer to submission {int} ends with {string}', function (n, text) {
    judge.endsWith(this.kfsp, n, text);
  });
  Then(P + 'submission {int} returns accounted red observations', function (n) {
    judge.observations(this.kfsp, n);
  });
  Then(P + 'the refused submissions and their gates are {string}', function (spec) {
    judge.refusals(this.kfsp, spec);
  });
  Then(P + 'the Intent is published', function () {
    judge.published(this.kfsp);
  });
  Then(P + 'nothing is published', function () {
    judge.unpublished(this.kfsp);
  });
  Then(P + 'the readiness record keeps scenario {string} {word}', function (name, status) {
    judge.status(this.kfsp, name, status);
  });
  Then(P + 'the readiness record lists these unexercised steps of scenario {string}:', function (name, text) {
    judge.unexercised(this.kfsp, name, text.split('\n'));
  });
  Then(P + 'no skipped step body ran', function () {
    judge.noTrace(this.kfsp);
  });
}
