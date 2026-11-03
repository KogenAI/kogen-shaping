'use strict';
// Acceptance harness of k-shape-submit-reachable: it adds k_shape_submit_reachable_test.go.txt to
// the shape package through a go overlay, builds the test binary once, runs one scripted native
// Shape conversation per scenario and judges its KSSR observation line.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');

const TEST = 'TestKShapeSubmitReachableAcceptance';
const ROOT = path.resolve(__dirname, '..', '..');
// Overlay paths are relative to ROOT, the directory go runs in.
const SOURCE = 'features/steps/k_shape_submit_reachable_test.go.txt';
const TARGET = 'shape/k_shape_submit_reachable_acceptance_test.go';
let binary = null;

// The fixture's redactor masks the values of secret-named variables, so they are not passed on.
function env(extra) {
  const out = {...process.env, ...extra};
  for (const k of Object.keys(out)) {
    if (k.startsWith('KOGEN_ACCEPTANCE_') || (/TOKEN|KEY|PASSWORD|SECRET/i.test(k) && !(k in extra))) delete out[k];
  }
  return out;
}

function build() {
  if (binary) return binary;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'kssr-'));
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[TARGET]: SOURCE}}));
  const bin = path.join(dir, 'shape.test');
  const args = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const r = spawnSync('go', args, {cwd: ROOT, env: env({PWD: ROOT}), encoding: 'utf8', timeout: 540000, maxBuffer: 16 << 20});
  if (r.status !== 0) {
    throw new Error(`${TEST}: go ${args.join(' ')} failed (exit ${r.status}, error ${r.error || 'none'}):\n${r.stdout}${r.stderr}`);
  }
  binary = bin;
  return bin;
}

function observe(wire, script) {
  const bin = build();
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=120s'];
  const r = spawnSync(bin, args, {cwd: path.join(ROOT, 'shape'), env: env({PWD: path.join(ROOT, 'shape'), KSSR_WIRE: wire, KSSR_SCRIPT: script}), encoding: 'utf8', timeout: 180000, maxBuffer: 16 << 20});
  const text = (r.stdout || '') + (r.stderr || '');
  const detail = `${TEST} (KSSR_WIRE=${wire} KSSR_SCRIPT=${script}): shape.test ${args.join(' ')}, built by go test -c -vet=off -overlay ./shape, test binary exit=${r.status} error=${r.error || 'none'}`;
  const line = text.split('\n').find((l) => l.startsWith('KSSR '));
  if (r.status !== 0 || !line || !text.includes('--- PASS: ' + TEST + ' ')) {
    throw new Error(`${detail}: the test did not run to a KSSR observation\n${text}`);
  }
  return {...JSON.parse(line.slice('KSSR '.length)), detail};
}

const check = {
  exit(o, code) { assert.equal(o.exit, code, `${o.detail}: exit code; stop text:\n${o.stop}`); },
  requests(o, n) { assert.equal(o.requests, n, `${o.detail}: shaping requests; stop text:\n${o.stop}`); },
  forced(o, list) {
    const want = list === 'none' ? [] : list.split(',').map((s) => Number(s.trim()));
    assert.deepEqual(o.forced, want, `${o.detail}: requests that required submit`);
  },
  written(o, yes) { assert.equal(o.written, yes, `${o.detail}: intent.md written`); },
  stop(o, text) { assert.equal(o.stop, text === '' ? '' : text + '\n', `${o.detail}: stop text`); },
  answers(o, rows) {
    const keys = ['chars', 'no_tool_turns', 'owner', 'request', 'text', 'truncated'];
    const got = o.answers.map((a) => {
      const has = Object.prototype.hasOwnProperty.call(a, 'text');
      assert.deepEqual(Object.keys(a).sort(), has ? keys : keys.filter((k) => k !== 'text'), `${o.detail}: tool_free_answer fields of ${JSON.stringify(a)}`);
      assert.equal(a.owner, 'shape', `${o.detail}: tool_free_answer owner`);
      const runes = has ? [...a.text] : [];
      return {
        request: String(a.request), no_tool_turns: String(a.no_tool_turns), chars: String(a.chars), truncated: String(a.truncated),
        text_runes: String(runes.length), text_start: has ? runes.slice(0, 40).join('') : '(absent)',
      };
    });
    assert.deepEqual(got, rows, `${o.detail}: tool_free_answer events`);
  },
};

const PROBLEM = 'submit takes exactly one of a non-empty questions list, a non-empty files list, or a blocked report';
const REFUSED_STOP = 'shape/invalid_output: the shaper answered without calling any tool 3 times since its last submission (no-tool limit); that submission (attempt 1) was refused at gate validation\n' +
  '  the shaper ended without calling submit\n  last refusal (attempt 1, gate validation):\n  ' + PROBLEM + '\n  last tool-free answer: Waiting for the readiness result.';
const TEXT_ROWS = [
  {request: '1', no_tool_turns: '1', chars: '39', truncated: 'false', text_runes: '39', text_start: 'Plan: token=[REDACTED] and then submit.'},
  {request: '2', no_tool_turns: '2', chars: '0', truncated: 'false', text_runes: '0', text_start: '(absent)'},
  {request: '4', no_tool_turns: '3', chars: '2513', truncated: 'true', text_runes: '2000', text_start: 'Long answer: abcdefghijabcdefghijabcdefg'},
];
const LONG = 'Long answer: ' + 'abcdefghij'.repeat(250);
const goodAnswers = () => [
  {owner: 'shape', request: 1, no_tool_turns: 1, chars: 39, truncated: false, text: 'Plan: token=[REDACTED] and then submit.'},
  {owner: 'shape', request: 2, no_tool_turns: 2, chars: 0, truncated: false},
  {owner: 'shape', request: 4, no_tool_turns: 3, chars: 2513, truncated: true, text: [...LONG].slice(0, 2000).join('')},
];

// Observer controls: good runs the live harness on a conversation this Intent does not change
// and accepts known-good observations; bad must reject every known-bad observation.
function controls(mode) {
  const accepts = (fn) => { try { fn(); return true; } catch (e) { return false; } };
  if (mode === 'good') {
    const live = observe('responses', 'control');
    check.exit(live, 1);
    check.requests(live, 4);
    check.forced(live, '2');
    check.written(live, false);
    check.stop(live, "shape/invalid_output: the shaper's output was not a valid Intent after 3 attempts\n  " + PROBLEM);
    const cycle = {detail: 'good cycle', exit: 0, requests: 6, forced: [2, 4, 6], written: true, stop: '', answers: []};
    check.exit(cycle, 0); check.requests(cycle, 6); check.forced(cycle, '2, 4, 6'); check.written(cycle, true); check.stop(cycle, '');
    check.stop({detail: 'good refusal stop', stop: REFUSED_STOP + '\n'}, REFUSED_STOP);
    check.answers({detail: 'good answers', answers: goodAnswers()}, TEXT_ROWS);
    return true;
  }
  const old = {detail: 'old cycle', exit: 1, requests: 5, forced: [2, 4], written: false, answers: [],
    stop: 'shape/invalid_output: the shaper answered without calling any tool 3 times (no-tool limit); no submission was made\n  the shaper ended without calling submit\n'};
  const unredacted = goodAnswers(); unredacted[0].text = 'Plan: token=abc123secret and then submit.'; unredacted[0].chars = 41;
  const untruncated = goodAnswers(); untruncated[2].text = LONG; untruncated[2].truncated = false;
  const emptyText = goodAnswers(); emptyText[1].text = '';
  const bad = [
    () => check.exit(old, 0),
    () => check.requests(old, 6),
    () => check.forced(old, '2, 4, 6'),
    () => check.stop(old, REFUSED_STOP),
    () => check.answers({detail: 'missing', answers: []}, TEXT_ROWS),
    () => check.answers({detail: 'unredacted', answers: unredacted}, TEXT_ROWS),
    () => check.answers({detail: 'untruncated', answers: untruncated}, TEXT_ROWS),
    () => check.answers({detail: 'empty text', answers: emptyText}, TEXT_ROWS),
  ];
  return bad.some(accepts);
}

if (require.main === module) {
  let ok;
  try { ok = controls(process.argv[2]); } catch (e) { ok = false; }
  console.log(ok ? 'accepted' : 'rejected');
  process.exitCode = ok ? 0 : 1;
} else {
  const {When, Then} = require('@cucumber/cucumber');
  When('Shape runs the scripted {string} conversation on the {string} route', {timeout: 720000}, function (script, wire) {
    this.kssr = observe(wire, script);
  });
  Then('Shape exits with code {int}', function (code) { check.exit(this.kssr, code); });
  Then('the provider received {int} shaping requests', function (n) { check.requests(this.kssr, n); });
  Then('the requests that required submit are {string}', function (list) { check.forced(this.kssr, list); });
  Then('the Intent file intent.md is written', function () { check.written(this.kssr, true); });
  Then('no Intent file intent.md is written', function () { check.written(this.kssr, false); });
  Then('there is no stop text', function () { check.stop(this.kssr, ''); });
  Then('the stop text is:', function (text) { check.stop(this.kssr, text); });
  Then('the tool-free answer events are:', function (table) { check.answers(this.kssr, table.hashes()); });
}
