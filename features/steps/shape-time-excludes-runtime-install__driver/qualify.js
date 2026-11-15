'use strict';
// Final-byte observer controls, independent of missing-feature failure order.
// The Go harness itself has been exercised on the unchanged tree. These fixtures
// qualify each output assertion, not an imagined passing test-runner transcript.
const fs = require('node:fs');
const path = require('node:path');
process.env.STI_OBSERVER_ONLY = '1';
const check = require('../shape-time-excludes-runtime-install__steps.js');
const feature = fs.readFileSync(path.join(process.cwd(), '.kogen/intents/shape-time-excludes-runtime-install/acceptance.feature'), 'utf8');
const lines = [...feature.matchAll(/      kogen: shape time [^\n]+/g)].map(m => m[0].trim());
const cold = {exit:0, rounds:2, error:'', accounts:[{elapsed_ms:908023,shaper_ms:280916,checks_ms:511168,install_ms:115939,gap_ms:0}],time_lines:[lines[0]]};
const warm = {exit:0, rounds:2, error:'', accounts:[{elapsed_ms:794084,shaper_ms:280916,checks_ms:511168,install_ms:0,gap_ms:2000}],time_lines:[lines[1]]};
const budget = 'shaping budget: 900 s (15 min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)';
const attempt = 'attempt 1: gate validation, model request 900 s, Kogen checks 0 s, 0 s remaining';
function refused(ms) { return {class:'shape',reason:'invalid_output', detail:'the shaper used its 15-minute shaping budget (15 min elapsed, 1 requests) without a submission that passed; shaping stopped', lines:[budget+(ms?`; install_ms=${ms}`:''),attempt+(ms?`; install_ms=${ms}`:'')]}; }
function judge(o, want, text) { check.result(o,0,2); check.account(o,want); check.line(o,text); }
const copy = o => JSON.parse(JSON.stringify(o));
const rejects = fn => { try { fn(); return false; } catch (_) { return true; } };
let accepted;
if (process.argv[2] === 'good') {
  judge(cold,cold.accounts[0],lines[0]); judge(warm,warm.accounts[0],lines[1]);
  check.refusal(refused(115939),115939); check.refusal(refused(0),0);
  accepted = true;
} else {
  const bad = [];
  for (const fixture of [cold,warm]) {
    for (const field of ['elapsed_ms','shaper_ms','checks_ms','install_ms','gap_ms']) {
      const o = copy(fixture); o.accounts[0][field]++;
      bad.push(() => judge(o,fixture.accounts[0],fixture.time_lines[0]));
    }
    const missing = copy(fixture); missing.accounts = [];
    bad.push(() => judge(missing,fixture.accounts[0],fixture.time_lines[0]));
    const omitted = copy(fixture); delete omitted.accounts[0].install_ms;
    bad.push(() => judge(omitted,fixture.accounts[0],fixture.time_lines[0]));
    const wrongLine = copy(fixture); wrongLine.time_lines[0] += ', runtime install 0.0 s';
    bad.push(() => check.line(wrongLine,fixture.time_lines[0]));
  }
  const overlap = copy(cold); overlap.accounts[0].gap_ms = 115939;
  bad.push(() => check.account(overlap,{}));
  const negative = copy(cold); negative.accounts[0].gap_ms = -1;
  bad.push(() => check.account(negative,{}));
  const noNext = copy(cold); noNext.exit=1; noNext.rounds=1;
  bad.push(() => check.result(noNext,0,2));
  for (const ms of [115939,0]) {
    const o = refused(ms); o.lines[1] = attempt;
    if (!ms) o.lines[1] += '; install_ms=0';
    bad.push(() => check.refusal(o,ms));
    const changed = refused(ms); changed.detail += ' changed';
    bad.push(() => check.refusal(changed,ms));
  }
  accepted = !bad.every(fn => rejects(fn));
}
console.log(accepted ? 'observer accepted' : 'observer rejected');
process.exitCode = accepted ? 0 : 1;
