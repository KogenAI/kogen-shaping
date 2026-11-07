'use strict';
// Acceptance of k-shape-split-oversized-review. A When step shapes one synthetic case: it
// compiles the shape package's tests with the harness of k-shape-split-oversized-review__harness.js
// (through a Go overlay), runs TestKShapeSplitOversizedReviewAcceptance for the case with a
// scripted external shaper and literal file sizes, and reads what the run showed. Nothing in the
// checkout is written. The Then steps judge what was seen. Run this file directly with good or
// bad to check the judgments on known observations.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeSplitOversizedReviewAcceptance';
const CEILING_NOTE = '(tools.InputBudgetBytes)';
const FEATURE_BYTES = 192;
const STEPS_BYTES = 210;
let built = null;
const shaped = {};

function env(extra) {
  const e = {...process.env, ...extra};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

// build compiles the shape tests with the harness once.
function build() {
  if (built) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'k-shape-split-oversized-review-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const source = path.join(dir, 'k_shape_split_oversized_review_acceptance_test.go');
  fs.writeFileSync(source, require('./k-shape-split-oversized-review__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {'shape/k_shape_split_oversized_review_acceptance_test.go': source}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {env: env(), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
  if (r.error || r.status !== 0) throw new Error(`${TEST}: building the shape tests: go ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${r.stdout}${r.stderr}`);
  built = {bin, dir};
  return built;
}

// shapeCase runs the harness for one case once and returns what the run showed.
function shapeCase(name) {
  if (shaped[name]) return shaped[name];
  const b = build();
  const out = path.join(b.dir, name + '.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=90s'];
  const r = spawnSync(b.bin, args, {cwd: path.resolve('shape'), env: env({KSR_CASE: name, KSR_OUT: out}), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
  const text = (r.stdout || '') + (r.stderr || '');
  if (r.error || r.status !== 0 || !text.includes('--- PASS: ' + TEST + ' ') || !fs.existsSync(out)) {
    throw new Error(`${TEST} (KSR_CASE=${name}): shape.test ${args.join(' ')} exit=${r.status} error=${r.error || ''}\n${text}`);
  }
  shaped[name] = JSON.parse(fs.readFileSync(out, 'utf8'));
  return shaped[name];
}

function shown(o) {
  const feedback = o.feedback.map((text, i) => `result of submission ${i + 1}:\n${text}`).join('\n----\n');
  return `case ${o.case} (tools.InputBudgetBytes ${o.ceiling}): shape ended with exit ${o.exit}, err ${o.err}; published=${o.published}; ` +
    `${o.shaper} shaper round(s), ${o.decision} decision check(s), ${o.decisionAtSecond} before the second shaper request; refused gates ${JSON.stringify(o.gates)}; ` +
    `expected estimates ${JSON.stringify(o.expected)}\n${feedback || '(no submission results)'}\nprogress (tail):\n${String(o.progress).slice(-2500)}`;
}

function result(o, n) {
  const text = o.feedback[n - 1];
  assert.equal(typeof text, 'string', `submission ${n} got no result back\n${shown(o)}`);
  return text;
}

const ESTIMATE = /kogen: review estimate (\d+) bytes of the (\d+)-byte review ceiling \(tools\.InputBudgetBytes\)/g;

// estimates are the review estimate progress lines in order, as [estimate, ceiling].
function estimates(o) {
  return [...String(o.progress).matchAll(ESTIMATE)].map(m => [Number(m[1]), Number(m[2])]);
}

// blocks are the units of a refusal: the unit line and the item lines after it.
function blocks(text) {
  const out = [];
  for (const line of text.split('\n')) {
    if (line.startsWith('Scope check: ')) out.push({head: line, items: []});
    else if (line.startsWith('- ') && out.length) out[out.length - 1].items.push(line);
  }
  return out;
}

const checks = {
  ceiling(o, n) {
    assert.equal(o.ceiling, n, `the run used tools.InputBudgetBytes ${o.ceiling}, want ${n}\n${shown(o)}`);
  },
  refused(o, n) {
    const text = result(o, n);
    assert.ok(text.startsWith('Scope check: '), `submission ${n} was not refused by the scope check\n${shown(o)}`);
  },
  scopeRefusals(o, n) {
    const got = o.gates.filter(g => g === 'scope').length;
    assert.equal(got, n, `the scope check refused ${got} submissions, want ${n}\n${shown(o)}`);
  },
  intentLine(o, n, files, areas) {
    const want = `Scope check: the Intent is estimated at ${o.expected[n - 1]} bytes of mandatory review material (scenarios 2, acceptance harnesses 1, changed files ${files} in areas ${areas}), over the ${o.ceiling}-byte review ceiling ${CEILING_NOTE}.`;
    assert.equal(result(o, n).split('\n')[0], want, `the first line of the result of submission ${n} differs\n${shown(o)}`);
  },
  partLine(o, n, slug, files, areas) {
    const want = `Scope check: part "${slug}" is estimated at ${o.parts[slug]} bytes of mandatory review material (changed files ${files} in areas ${areas}; its own package is not counted), over the ${o.ceiling}-byte review ceiling ${CEILING_NOTE}.`;
    assert.ok(result(o, n).split('\n').includes(want), `the result of submission ${n} has no line ${JSON.stringify(want)}\n${shown(o)}`);
  },
  noIntentLine(o, n) {
    assert.ok(!result(o, n).includes('Scope check: the Intent'), `the result of submission ${n} refuses the Intent itself\n${shown(o)}`);
  },
  inputs(o, n) {
    const text = result(o, n);
    for (const line of ['- README.md: 3000 bytes (review input)', `- docs/testing-rules.md: ${o.rules} bytes (review input)`,
      `- .kogen/intents/is-even/acceptance.feature: ${FEATURE_BYTES} bytes (Intent package)`, `- features/steps/is-even__steps.js: ${STEPS_BYTES} bytes (Intent package)`]) {
      assert.ok(text.split('\n').includes(line), `the result of submission ${n} has no line ${JSON.stringify(line)}\n${shown(o)}`);
    }
  },
  itemsAddUp(o, n) {
    const found = blocks(result(o, n));
    assert.ok(found.length > 0, `the result of submission ${n} has no unit line\n${shown(o)}`);
    for (const b of found) {
      const est = Number(/ is estimated at (\d+) bytes /.exec(b.head)[1]);
      const sum = b.items.reduce((s, l) => s + Number(/: (\d+) bytes \(/.exec(l)[1]), 0);
      assert.equal(sum, est, `the item lines under ${JSON.stringify(b.head)} add up to ${sum}, not its estimate ${est}\n${shown(o)}`);
    }
  },
  contains(o, n, text) {
    assert.ok(result(o, n).includes(text), `the result of submission ${n} does not contain ${JSON.stringify(text)}\n${shown(o)}`);
  },
  lacks(o, n, text) {
    assert.ok(!result(o, n).includes(text), `the result of submission ${n} contains ${JSON.stringify(text)}\n${shown(o)}`);
  },
  guidance(o, n) {
    for (const text of ['split the work into separately reviewable Intents', 'list every remaining part in a ## Split section of intent.md',
      'A ## Stages plan does not lower the estimate', 'Do not satisfy the ceiling by deleting requested behaviour', 'Resubmitting this candidate unchanged is refused again.']) {
      assert.ok(result(o, n).includes(text), `the result of submission ${n} does not contain the guidance ${JSON.stringify(text)}\n${shown(o)}`);
    }
  },
  estimateShown(o, n) {
    const got = estimates(o)[n - 1];
    assert.ok(got, `the progress has no review estimate line for submission ${n}\n${shown(o)}`);
    assert.deepEqual(got, [o.expected[n - 1], o.ceiling], `the review estimate of submission ${n} is ${got[0]} of ${got[1]}, want the sum ${o.expected[n - 1]} of the review material and the ceiling ${o.ceiling}\n${shown(o)}`);
  },
  estimateIs(o, n, bytes) {
    const got = estimates(o)[n - 1];
    assert.ok(got, `the progress has no review estimate line for submission ${n}\n${shown(o)}`);
    assert.equal(got[0], bytes, `the review estimate of submission ${n} is ${got[0]}, want ${bytes}\n${shown(o)}`);
  },
  split(o, slugs) {
    const names = slugs.split(',');
    const want = `kogen: split: this Intent is part 1 of ${names.length + 1}; shape the other parts as their own Intents: ` +
      names.map(s => `${s} (${o.parts[s]} of ${o.ceiling} bytes)`).join(', ');
    assert.ok(String(o.progress).includes(want), `the progress has no line ${JSON.stringify(want)}\n${shown(o)}`);
  },
  noSplit(o) {
    assert.ok(!String(o.progress).includes('kogen: split:'), `the progress names a split\n${shown(o)}`);
  },
  noPart(o, slug) {
    assert.ok(!String(o.progress).includes(`${slug} (`), `the progress names the part ${slug}\n${shown(o)}`);
  },
  published(o) {
    assert.equal(o.published, true, `the Intent was not published\n${shown(o)}`);
  },
  intentHas(o, text) {
    assert.ok(o.published && o.intent.includes(text), `the published intent.md does not contain ${JSON.stringify(text)}\n${shown(o)}`);
  },
  noDecisionBeforeSecond(o) {
    assert.equal(o.decisionAtSecond, 0, `${o.decisionAtSecond} decision check(s) ran before the second shaper request, want 0\n${shown(o)}`);
  },
  decisionsAtLeast(o, n) {
    assert.ok(o.decision >= n, `the decision check ran ${o.decision} time(s), want at least ${n}\n${shown(o)}`);
  },
};

// controls runs the judgments on known observations: good must accept every known-good one, bad
// must reject every known-bad one.
function controls(kind) {
  const C = 524288;
  const rules = 15677;
  const guidance = 'Above the ceiling this cannot be approved as one Intent: split the work into separately reviewable Intents. Narrow this Intent to its first part so that its estimate fits, ' +
    'and list every remaining part in a ## Split section of intent.md, one bullet per part, `- <slug>: <what the part delivers>`, naming in backticks the files that part changes; ' +
    "a part's estimate is the review inputs plus those files, and each part must fit the ceiling too. A ## Stages plan does not lower the estimate, because the final review carries every stage. " +
    'Do not satisfy the ceiling by deleting requested behaviour: every behaviour and example the request asks for stays in this Intent or in a part. Resubmitting this candidate unchanged is refused again.';
  const inputs = ['- README.md: 3000 bytes (review input)', `- docs/testing-rules.md: ${rules} bytes (review input)`];
  const pkg = n => [`- .kogen/intents/is-even/intent.md: ${n} bytes (Intent package)`, `- .kogen/intents/is-even/acceptance.feature: ${FEATURE_BYTES} bytes (Intent package)`, `- features/steps/is-even__steps.js: ${STEPS_BYTES} bytes (Intent package)`];
  const changed = [['engine/core.js', 300000], ['engine/rules.js', 250000]];
  const changedLines = list => list.map(([p, n]) => `- ${p}: ${n} bytes (changed file, area engine)`);
  const sum = list => list.reduce((s, [, n]) => s + n, 0);
  const intentRefusal = (intentBytes, list) => {
    const est = intentBytes + FEATURE_BYTES + STEPS_BYTES + 3000 + rules + sum(list);
    return {est, text: [`Scope check: the Intent is estimated at ${est} bytes of mandatory review material (scenarios 2, acceptance harnesses 1, changed files ${list.length} in areas 1), over the ${C}-byte review ceiling ${CEILING_NOTE}.`,
      ...pkg(intentBytes), ...inputs, ...changedLines(list), guidance].join('\n')};
  };
  const partRefusal = (slug, list) => {
    const est = 3000 + rules + sum(list);
    return {est, text: [`Scope check: part "${slug}" is estimated at ${est} bytes of mandatory review material (changed files ${list.length} in areas 1; its own package is not counted), over the ${C}-byte review ceiling ${CEILING_NOTE}.`,
      ...inputs, ...changedLines(list), guidance].join('\n')};
  };
  const line = n => `kogen: review estimate ${n} bytes of the ${C}-byte review ceiling ${CEILING_NOTE} (0 s)\n`;
  const splitLine = parts => `kogen: split: this Intent is part 1 of ${parts.length + 1}; shape the other parts as their own Intents: ${parts.map(([s, n]) => `${s} (${n} of ${C} bytes)`).join(', ')} (4 s)\n`;
  const base = {ceiling: C, rules, exit: 0, err: '<nil>', class: '', reason: '', published: true, intent: '', stdout: '', gates: [], shaper: 3, decision: 2, decisionAtSecond: 0, parts: {}};
  const fitsEst = 1500 + FEATURE_BYTES + STEPS_BYTES + 3000 + rules + 1000;
  const fits = {...base, case: 'fits', expected: [fitsEst, fitsEst], progress: line(fitsEst) + line(fitsEst), feedback: ['ack'], decisionAtSecond: 1};
  const r1 = intentRefusal(2200, changed);
  const e2 = 2400 + FEATURE_BYTES + STEPS_BYTES + 3000 + rules + 300000;
  const rulesPart = 3000 + rules + 250000;
  const oversized = {...base, case: 'oversized', expected: [r1.est, e2, e2], gates: ['scope'], parts: {'engine-rules': rulesPart},
    progress: line(r1.est) + line(e2) + splitLine([['engine-rules', rulesPart]]) + line(e2) + splitLine([['engine-rules', rulesPart]]),
    feedback: [r1.text, 'ack', 'ack'], intent: 'x\n## Split\n- engine-rules: Move the table.\n'};
  const allPart = partRefusal('engine-all', changed);
  const e3 = 2500 + FEATURE_BYTES + STEPS_BYTES + 3000 + rules + 1000;
  const e4 = 2600 + FEATURE_BYTES + STEPS_BYTES + 3000 + rules + 1000;
  const corePart = 3000 + rules + 300000;
  const partOver = {...base, case: 'part-over', expected: [e3, e4, e4], gates: ['scope'], parts: {'engine-all': allPart.est, 'engine-core': corePart, 'engine-rules': rulesPart},
    progress: line(e3) + line(e4) + splitLine([['engine-core', corePart], ['engine-rules', rulesPart]]) + line(e4) + splitLine([['engine-core', corePart], ['engine-rules', rulesPart]]),
    feedback: [allPart.text, 'ack', 'ack']};
  const staged = intentRefusal(2900, changed);
  const stages = {...oversized, case: 'stages', expected: [staged.est, e2, e2], progress: line(staged.est) + line(e2) + splitLine([['engine-rules', rulesPart]]) + line(e2) + splitLine([['engine-rules', rulesPart]]), feedback: [staged.text, 'ack', 'ack']};
  const edgeText = intentRefusal(4097, [['lib/util.js', 1000]]);
  const edge = {...base, case: 'edge', expected: [C + 1, C], progress: line(C + 1) + line(C) + line(C), gates: ['scope'], feedback: [edgeText.text, 'ack'], decisionAtSecond: 0, decision: 3};
  const withEdgeEstimate = (o, n) => ({...o, expected: [n, o.expected[1]]});

  const good = [
    [fits, o => checks.ceiling(o, C)], [fits, o => checks.published(o)], [fits, o => checks.estimateShown(o, 1)], [fits, o => checks.noSplit(o)], [fits, o => checks.lacks(o, 1, 'Scope check')],
    [oversized, o => checks.refused(o, 1)], [oversized, o => checks.intentLine(o, 1, 2, 1)], [oversized, o => checks.inputs(o, 1)], [oversized, o => checks.itemsAddUp(o, 1)],
    [oversized, o => checks.contains(o, 1, '- engine/core.js: 300000 bytes (changed file, area engine)')], [oversized, o => checks.guidance(o, 1)],
    [oversized, o => checks.estimateShown(o, 1)], [oversized, o => checks.estimateShown(o, 2)], [oversized, o => checks.split(o, 'engine-rules')],
    [oversized, o => checks.noDecisionBeforeSecond(o)], [oversized, o => checks.scopeRefusals(o, 1)], [oversized, o => checks.intentHas(o, '## Split')],
    [partOver, o => checks.refused(o, 1)], [partOver, o => checks.partLine(o, 1, 'engine-all', 2, 1)], [partOver, o => checks.noIntentLine(o, 1)], [partOver, o => checks.itemsAddUp(o, 1)],
    [partOver, o => checks.noPart(o, 'engine-all')], [partOver, o => checks.split(o, 'engine-core,engine-rules')],
    [stages, o => checks.refused(o, 1)], [stages, o => checks.intentLine(o, 1, 2, 1)], [stages, o => checks.itemsAddUp(o, 1)],
    [edge, o => checks.refused(o, 1)], [edge, o => checks.scopeRefusals(o, 1)], [edge, o => checks.estimateIs(o, 1, C + 1)], [edge, o => checks.estimateIs(o, 2, C)], [edge, o => checks.decisionsAtLeast(o, 1)],
  ];
  const mutate = (o, change) => {
    const c = JSON.parse(JSON.stringify(o));
    change(c);
    return c;
  };
  // The unchanged tree's observations: nothing is counted, so nothing is shown or refused.
  const none = o => mutate(o, c => { c.progress = ''; c.gates = []; c.feedback = ['ack', 'ack']; c.decisionAtSecond = 1; c.decision = 3; });
  const bad = [
    [none(fits), o => checks.estimateShown(o, 1)],
    [none(oversized), o => checks.refused(o, 1)], [none(oversized), o => checks.split(o, 'engine-rules')], [none(oversized), o => checks.noDecisionBeforeSecond(o)], [none(oversized), o => checks.scopeRefusals(o, 1)],
    [none(partOver), o => checks.refused(o, 1)], [none(partOver), o => checks.split(o, 'engine-core,engine-rules')],
    [none(stages), o => checks.refused(o, 1)], [none(edge), o => checks.refused(o, 1)], [none(edge), o => checks.estimateIs(o, 1, C + 1)],
    [mutate(fits, o => { o.ceiling = 1048576; }), o => checks.ceiling(o, C)],
    [mutate(fits, o => { o.progress = o.progress.replace(String(fitsEst), String(fitsEst - 1000)); }), o => checks.estimateShown(o, 1)],
    [mutate(fits, o => { o.progress = o.progress.replace(String(C), '1048576'); }), o => checks.estimateShown(o, 1)],
    [mutate(fits, o => { o.progress += splitLine([['engine-rules', rulesPart]]); }), o => checks.noSplit(o)],
    [mutate(fits, o => { o.published = false; }), o => checks.published(o)],
    [mutate(oversized, o => { o.feedback[0] = 'Scope check: the Intent is estimated at 100 bytes' + o.feedback[0].slice(o.feedback[0].indexOf(' bytes of mandatory')); }), o => checks.intentLine(o, 1, 2, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('changed files 2 in areas 1', 'changed files 1 in areas 1'); }), o => checks.intentLine(o, 1, 2, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('mandatory review material', 'one review'); }), o => checks.intentLine(o, 1, 2, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace(`${C}-byte review ceiling`, '1048576-byte review ceiling'); }), o => checks.intentLine(o, 1, 2, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('- README.md: 3000 bytes (review input)\n', ''); }), o => checks.inputs(o, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace(`- docs/testing-rules.md: ${rules} bytes (review input)\n`, ''); }), o => checks.itemsAddUp(o, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('- engine/core.js: 300000 bytes', '- engine/core.js: 4096 bytes'); }), o => checks.itemsAddUp(o, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('(changed file, area engine)', '(named file, area engine)'); }), o => checks.contains(o, 1, '- engine/core.js: 300000 bytes (changed file, area engine)')],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('split the work into separately reviewable Intents', 'split the work into stages'); }), o => checks.guidance(o, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace('A ## Stages plan does not lower the estimate, because the final review carries every stage. ', ''); }), o => checks.guidance(o, 1)],
    [mutate(oversized, o => { o.feedback[0] = o.feedback[0].replace(' Resubmitting this candidate unchanged is refused again.', ''); }), o => checks.guidance(o, 1)],
    [mutate(oversized, o => { o.progress = o.progress.replace(String(r1.est), String(r1.est - 250000)); }), o => checks.estimateShown(o, 1)],
    [mutate(oversized, o => { o.progress = o.progress.replaceAll('part 1 of 2', 'part 2 of 2'); }), o => checks.split(o, 'engine-rules')],
    [mutate(oversized, o => { o.progress = o.progress.replaceAll(`engine-rules (${rulesPart} of`, `engine-rules (${rulesPart - 3000} of`); }), o => checks.split(o, 'engine-rules')],
    [mutate(oversized, o => { o.decisionAtSecond = 1; }), o => checks.noDecisionBeforeSecond(o)],
    [mutate(oversized, o => { o.gates = ['decision_check']; }), o => checks.scopeRefusals(o, 1)],
    [mutate(oversized, o => { o.intent = 'x'; }), o => checks.intentHas(o, '## Split')],
    [mutate(partOver, o => { o.feedback[0] = o.feedback[0].replace('part "engine-all"', 'part "engine-core"'); }), o => checks.partLine(o, 1, 'engine-all', 2, 1)],
    [mutate(partOver, o => { o.feedback[0] = o.feedback[0].replace('its own package is not counted', 'package counted'); }), o => checks.partLine(o, 1, 'engine-all', 2, 1)],
    [mutate(partOver, o => { o.feedback[0] = intentRefusal(2500, changed).text + '\n' + o.feedback[0]; }), o => checks.noIntentLine(o, 1)],
    [mutate(partOver, o => { o.progress += splitLine([['engine-all', allPart.est]]); }), o => checks.noPart(o, 'engine-all')],
    [mutate(stages, o => { o.feedback[0] = o.feedback[0].replace('changed files 2 in areas 1', 'changed files 0 in areas 0'); }), o => checks.intentLine(o, 1, 2, 1)],
    [mutate(edge, o => { o.progress = line(C) + line(C) + line(C); }), o => checks.estimateIs(o, 1, C + 1)],
    [mutate(edge, o => { o.progress = line(C + 1) + line(C + 1) + line(C); }), o => checks.estimateIs(o, 2, C)],
    [mutate(edge, o => { o.gates = ['scope', 'scope']; }), o => checks.scopeRefusals(o, 1)],
    [mutate(edge, o => { o.decision = 0; }), o => checks.decisionsAtLeast(o, 1)],
    [withEdgeEstimate(mutate(oversized, o => {}), r1.est + 1), o => checks.estimateShown(o, 1)],
  ];
  const passes = (o, check) => { try { check(o); return true; } catch { return false; } };
  const accepted = kind === 'good' ? good.every(([o, check]) => passes(o, check)) : bad.some(([o, check]) => passes(o, check));
  console.log(accepted ? 'accepted' : 'rejected');
  process.exitCode = accepted ? 0 : 1;
}

if (require.main === module) {
  controls(process.argv[2]);
} else {
  const {When, Then} = require('@cucumber/cucumber');
  const P = 'k-shape-split-oversized-review: ';
  When(P + 'the {string} case is shaped', {timeout: 120000}, function (name) { this.ksr = shapeCase(name); });
  Then(P + 'the run used a review ceiling of {int} bytes', function (n) { checks.ceiling(this.ksr, n); });
  Then(P + 'submission {int} is refused by the scope check', function (n) { checks.refused(this.ksr, n); });
  Then(P + 'the scope check refused {int} submission(s)', function (n) { checks.scopeRefusals(this.ksr, n); });
  Then(P + 'the first line of the result of submission {int} is the Intent line for {int} changed files in {int} areas', function (n, files, areas) { checks.intentLine(this.ksr, n, files, areas); });
  Then(P + 'the result of submission {int} has the part line for {string} with {int} changed files in {int} areas', function (n, slug, files, areas) { checks.partLine(this.ksr, n, slug, files, areas); });
  Then(P + 'the result of submission {int} does not refuse the Intent itself', function (n) { checks.noIntentLine(this.ksr, n); });
  Then(P + 'the result of submission {int} lists the review inputs and the Intent package', function (n) { checks.inputs(this.ksr, n); });
  Then(P + 'the item lines of the result of submission {int} add up to the estimate of each unit', function (n) { checks.itemsAddUp(this.ksr, n); });
  Then(P + 'the result of submission {int} contains {string}', function (n, text) { checks.contains(this.ksr, n, text); });
  Then(P + 'the result of submission {int} does not contain {string}', function (n, text) { checks.lacks(this.ksr, n, text); });
  Then(P + 'the result of submission {int} tells the shaper to split the work into separately reviewable Intents', function (n) { checks.guidance(this.ksr, n); });
  Then(P + 'the progress shows the review estimate of submission {int} against the ceiling', function (n) { checks.estimateShown(this.ksr, n); });
  Then(P + 'the progress shows the review estimate of submission {int} as {int} bytes', function (n, bytes) { checks.estimateIs(this.ksr, n, bytes); });
  Then(P + 'the progress shows the split into the parts {string}', function (slugs) { checks.split(this.ksr, slugs); });
  Then(P + 'the progress shows no split', function () { checks.noSplit(this.ksr); });
  Then(P + 'the progress does not name the part {string}', function (slug) { checks.noPart(this.ksr, slug); });
  Then(P + 'the Intent is published', function () { checks.published(this.ksr); });
  Then(P + 'the published intent.md contains {string}', function (text) { checks.intentHas(this.ksr, text); });
  Then(P + 'no decision check ran before the second shaper request', function () { checks.noDecisionBeforeSecond(this.ksr); });
  Then(P + 'the decision check ran at least {int} time(s)', function (n) { checks.decisionsAtLeast(this.ksr, n); });
}
