'use strict';
// Acceptance of fix-k-shape-wrong-implementation. The Given step compiles the shape package's tests
// with the harness of fix-k-shape-wrong-implementation__harness.js (through a Go overlay), runs
// TestFixKShapeWrongImplementationAcceptance once per Cucumber process and reads the JSON
// observation the harness wrote to FKSWI_OUT. The Then steps compare one case of it with the
// feature's values. The When step runs named existing tests of the shape package with the same
// test binary. Run this file directly with --control good or --control bad to check the
// judgments on known observations.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TEST = 'TestFixKShapeWrongImplementationAcceptance';
const TARGET = 'shape/fix_k_shape_wrong_implementation_acceptance_test.go';
const FEATURE = '.kogen/intents/fix-k-shape-wrong-implementation/acceptance.feature';
const P = 'fix-k-shape-wrong-implementation: ';

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

// judge is the observation of one harness run: the run exited 0, passed the harness test exactly
// once and wrote an observation.
function judge(r) {
  const shown = `${TEST} (${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}:\n${r.output}`;
  assert.equal(r.error, null, shown);
  assert.equal(r.status, 0, shown);
  const passes = r.output.split('\n').filter(l => l.startsWith('--- PASS: ' + TEST + ' ')).length;
  assert.equal(passes, 1, `${TEST} did not pass exactly once (${passes} PASS lines): ${shown}`);
  assert.ok(r.json !== null, `${TEST} wrote no observation file: ${shown}`);
  return JSON.parse(r.json);
}

// binary builds the shape tests with the harness once per Cucumber process.
let built = null;
function binary() {
  if (built !== null) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'fkswi-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, require('./fix-k-shape-wrong-implementation__harness.js'));
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const build = ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'];
  const b = spawnSync('go', build, {env: env({}), encoding: 'utf8', timeout: 400000, maxBuffer: 32 * 1024 * 1024});
  if (b.status !== 0) {
    throw new Error(`building the harness of ${TEST}: go ${build.join(' ')} exited ${b.status}${b.signal ? ' (' + b.signal + ')' : ''}${b.error ? ' ' + b.error.message : ''}:\n${b.stdout}${b.stderr}`);
  }
  built = {dir, bin};
  return built;
}

// harness runs the harness test once.
function harness() {
  const {dir, bin} = binary();
  const out = path.join(dir, 'observation.json');
  const args = ['-test.run', '^' + TEST + '$', '-test.count=1', '-test.v', '-test.timeout=200s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({FKSWI_OUT: out}), encoding: 'utf8', timeout: 220000, maxBuffer: 32 * 1024 * 1024});
  return judge({status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    json: fs.existsSync(out) ? fs.readFileSync(out, 'utf8') : null,
    command: `FKSWI_OUT=${out} shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape)`});
}

// existing runs the named existing tests of the shape package once with the same test binary.
function existing(names) {
  const {bin} = binary();
  const args = ['-test.run', '^(' + names.join('|') + ')$', '-test.count=1', '-test.v', '-test.timeout=280s'];
  const r = spawnSync(bin, args, {cwd: path.resolve('shape'), env: env({}), encoding: 'utf8', timeout: 300000, maxBuffer: 64 * 1024 * 1024});
  return {status: r.status, signal: r.signal, error: r.error ? r.error.message : null, output: (r.stdout || '') + (r.stderr || ''),
    command: `shape.test ${args.join(' ')} (built with go test -c -vet=off -overlay ./shape), in shape/`};
}

// kept checks that the run exited 0, that each named test passed exactly once, and that no test
// or subtest failed or was skipped.
function kept(r, names) {
  const shown = `(${r.command}) exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}${r.error ? ' ' + r.error : ''}`;
  const lines = r.output.split('\n');
  for (const name of names) {
    const passes = lines.filter(l => l.startsWith('--- PASS: ' + name + ' (')).length;
    const other = lines.filter(l => l.startsWith('--- FAIL: ' + name + ' (') || l.startsWith('--- SKIP: ' + name + ' (')).map(l => l.split(' ')[1]);
    assert.equal(passes, 1, `existing test ${name} was not reported as passed exactly once by ${shown}: ${passes} PASS lines, outcome ${other.length ? other.join(', ') : passes ? 'passed more than once' : 'absent (did not run)'}:\n${r.output}`);
  }
  const bad = lines.filter(l => /^\s*--- (FAIL|SKIP): /.test(l));
  assert.deepEqual(bad, [], `tests or subtests failed or were skipped in ${shown}:\n${r.output}`);
  assert.equal(r.error, null, `the existing tests did not finish: ${shown}:\n${r.output}`);
  assert.equal(r.status, 0, `the existing tests did not pass: ${shown}:\n${r.output}`);
}

// observed is one case of the observation.
function observed(o, name) {
  assert.ok(o && Object.hasOwn(o, name), `the harness observed no case "${name}":\n${JSON.stringify(o, null, 1)}`);
  return o[name];
}

// include checks that each listed key of a case has exactly the listed value.
function include(o, name, want) {
  const c = observed(o, name);
  const summary = JSON.stringify(c, null, 1);
  for (const [key, value] of Object.entries(want)) {
    assert.ok(Object.hasOwn(c, key), `case "${name}" has no ${key}:\n${summary}`);
    assert.deepEqual(c[key], value, `case "${name}": ${key} is ${JSON.stringify(c[key])}, want ${JSON.stringify(value)}\nobservation:\n${summary}`);
  }
}

function errorNotEmpty(o, name) {
  const c = observed(o, name);
  assert.ok(typeof c.error === 'string' && c.error !== '', `case "${name}": the check returned no error:\n${JSON.stringify(c, null, 1)}`);
}

function errorContains(o, name, text) {
  const c = observed(o, name);
  assert.ok(typeof c.error === 'string' && c.error.includes(text), `case "${name}": the error ${JSON.stringify(c.error)} does not contain ${JSON.stringify(text)}:\n${JSON.stringify(c, null, 1)}`);
}

// ---- observer controls: node features/steps/fix-k-shape-wrong-implementation__steps.js --control good|bad ----

// checks are the feature's Then steps, in order, each as a judgment of an observation.
function checks() {
  const lines = fs.readFileSync(path.join(__dirname, '../..', FEATURE), 'utf8').split('\n');
  const out = [];
  for (let i = 0; i < lines.length; i++) {
    const step = lines[i].trim().replace(/^(Then|And) /, '');
    let m;
    if ((m = step.match(/^fix-k-shape-wrong-implementation: the observation of "([^"]+)" includes:$/))) {
      assert.equal(lines[i + 1].trim(), '"""', 'an include step has a doc string');
      const end = lines.findIndex((l, j) => j > i + 1 && l.trim() === '"""');
      const want = JSON.parse(lines.slice(i + 2, end).join('\n'));
      const name = m[1];
      out.push({name, kind: 'include', want, run: o => include(o, name, want)});
      i = end;
    } else if (step === 'When fix-k-shape-wrong-implementation: I run these existing shape tests:') {
      const end = lines.findIndex((l, j) => j > i + 1 && l.trim() === '"""');
      const names = lines.slice(i + 2, end).map(l => l.trim()).filter(Boolean);
      out.push({name: 'existing', kind: 'kept', names, run: () => {}});
      i = end;
    } else if ((m = step.match(/^fix-k-shape-wrong-implementation: the error of "([^"]+)" is not empty$/))) {
      const name = m[1];
      out.push({name, kind: 'error', run: o => errorNotEmpty(o, name)});
    } else if ((m = step.match(/^fix-k-shape-wrong-implementation: the error of "([^"]+)" contains '([^']*)'$/))) {
      const [name, text] = [m[1], m[2]];
      out.push({name, kind: 'error', text, run: o => errorContains(o, name, text)});
    }
  }
  assert.ok(out.length > 0, 'the feature has judgments');
  return out;
}

// RUN is a real harness run's -test.v output on the unchanged tree.
const RUN = `=== RUN   ${TEST}\n=== PAUSE ${TEST}\n=== CONT  ${TEST}\n--- PASS: ${TEST} (1.70s)\nPASS\n`;

// KEPT is the result lines, in order, of a real run of the existing tests on the unchanged tree.
const KEPT = "--- PASS: TestParseControls (0.00s)\n" +
  "    --- PASS: TestParseControls/valid (0.00s)\n" +
  "    --- PASS: TestParseControls/an_outline's_example_step_with_the_outline's_name (0.00s)\n" +
  "    --- PASS: TestParseControls/the_correct_implementation_alone (0.00s)\n" +
  "    --- PASS: TestParseControls/not_json (0.00s)\n" +
  "    --- PASS: TestParseControls/not_an_object (0.00s)\n" +
  "    --- PASS: TestParseControls/an_extra_field (0.00s)\n" +
  "    --- PASS: TestParseControls/wrong_is_not_a_list (0.00s)\n" +
  "    --- PASS: TestParseControls/correct_with_another_field (0.00s)\n" +
  "    --- PASS: TestParseControls/a_wrong_implementation_without_a_field (0.00s)\n" +
  "    --- PASS: TestParseControls/a_blank_and_a_repeated_name (0.00s)\n" +
  "    --- PASS: TestParseControls/no_files (0.00s)\n" +
  "    --- PASS: TestParseControls/paths (0.00s)\n" +
  "    --- PASS: TestParseControls/machinery (0.00s)\n" +
  "    --- PASS: TestParseControls/too_many_files (0.00s)\n" +
  "    --- PASS: TestParseControls/too_many_bytes (0.00s)\n" +
  "    --- PASS: TestParseControls/the_byte_limit_itself (0.00s)\n" +
  "    --- PASS: TestParseControls/unknown_scenario_and_step (0.00s)\n" +
  "--- PASS: TestControlsJudge (0.00s)\n" +
  "    --- PASS: TestControlsJudge/an_assertion_in_a_step_file (0.00s)\n" +
  "    --- PASS: TestControlsJudge/the_resolved_copy_path (0.00s)\n" +
  "    --- PASS: TestControlsJudge/a_file_URL (0.00s)\n" +
  "    --- PASS: TestControlsJudge/a_frame_without_a_function (0.00s)\n" +
  "    --- PASS: TestControlsJudge/a_passing_scenario (0.00s)\n" +
  "    --- PASS: TestControlsJudge/an_earlier_step (0.00s)\n" +
  "    --- PASS: TestControlsJudge/an_Error (0.00s)\n" +
  "    --- PASS: TestControlsJudge/a_step_timeout (0.00s)\n" +
  "    --- PASS: TestControlsJudge/a_missing_module (0.00s)\n" +
  "    --- PASS: TestControlsJudge/an_assertion_thrown_by_implementation_code (0.00s)\n" +
  "    --- PASS: TestControlsJudge/an_assertion_thrown_outside_the_copy (0.00s)\n" +
  "    --- PASS: TestControlsJudge/an_assertion_without_a_stack (0.00s)\n" +
  "--- PASS: TestControlsJudgeAll (0.00s)\n" +
  "    --- PASS: TestControlsJudgeAll/rule_6_names_the_first_scenario_of_another_property_that_fails (0.00s)\n" +
  "    --- PASS: TestControlsJudgeAll/an_incomplete_wrong_run_and_a_property_no_wrong_implementation_names (0.00s)\n" +
  "    --- PASS: TestControlsJudgeAll/one_example_is_enough_for_an_outline (0.00s)\n" +
  "--- PASS: TestControlsUnreachable (0.00s)\n" +
  "--- PASS: TestControlsRefusalJoinsContractRefusal (0.00s)\n" +
  "--- PASS: TestControlsPreloadFreezesAssert (0.07s)\n" +
  "--- PASS: TestWrongImplementationRealRuns (0.00s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/a_failure_at_an_earlier_unrelated_step (1.69s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/acceptance_that_fails_on_each_removal (1.76s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/a_wrong_implementation_that_overwrites_an_assertion (1.76s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/a_wrong_implementation_that_throws_its_own_AssertionError (1.76s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/no_wrong_implementation (0.61s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/a_correct_implementation_that_disables_assertions (0.61s)\n" +
  "    --- PASS: TestWrongImplementationRealRuns/an_add_step_that_passes_whatever_calc_returns (1.28s)\n" +
  "--- PASS: TestWrongImplementationCaps (0.00s)\n" +
  "    --- PASS: TestWrongImplementationCaps/more_runs_than_the_cap_are_refused_before_anything_runs (0.19s)\n" +
  "    --- PASS: TestWrongImplementationCaps/a_run_that_would_start_with_no_time_left_does_not_start (0.19s)\n" +
  "    --- PASS: TestWrongImplementationCaps/the_time_cap_ends_the_check_at_the_run_it_cuts_off (2.21s)\n" +
  "--- PASS: TestWrongImplementationCheckRecordsAndSkips (0.00s)\n" +
  "    --- PASS: TestWrongImplementationCheckRecordsAndSkips/nothing_is_read,_run_or_recorded_without_a_checked_property (0.22s)\n" +
  "    --- PASS: TestWrongImplementationCheckRecordsAndSkips/a_missing_controls.json_is_refused_with_the_form (0.22s)\n" +
  "    --- PASS: TestWrongImplementationCheckRecordsAndSkips/a_clean_check_records_one_event (1.22s)\n" +
  "--- PASS: TestContractEvidenceRule (0.00s)\n" +
  "--- PASS: TestContractPayloadValidation (0.00s)\n" +
  "--- PASS: TestContractCheckLastRefusalByAnotherGateWritesNoRecord (2.45s)\n" +
  "--- PASS: TestContractVerdictCounting (0.00s)\n" +
  "--- PASS: TestContractFormats (0.00s)\n" +
  "--- PASS: TestContractEnvironmentEvidenceRule (0.00s)\n" +
  "--- PASS: TestContractCheckNumbersPropertyProblemsAfterContradictions (4.70s)\n" +
  "--- PASS: TestContractCheckRefusesAConfirmedContradictionAndPublishesItsCorrection (4.70s)\n" +
  "--- PASS: TestContractCheckKeptContradictionEndsNotReadyAndApprovalRefusesIt (7.13s)\n" +
  "--- PASS: TestContractCheckExternalShape (0.00s)\n" +
  "    --- PASS: TestContractCheckExternalShape/a_session_that_ends_without_a_result_is_incomplete_and_the_candidate_goes_on (4.03s)\n" +
  "    --- PASS: TestContractCheckExternalShape/one_read-only_session_per_review_and_confirmation;_the_refusal_is_the_next_round's_feedback (6.73s)\n" +
  "--- PASS: TestContractCheckEnvironmentFindings (0.00s)\n" +
  "    --- PASS: TestContractCheckEnvironmentFindings/a_confirmed_environment_finding_refuses_and_its_correction_is_published (4.60s)\n" +
  "    --- PASS: TestContractCheckEnvironmentFindings/a_not-ready_end_writes_the_record_and_approval_refuses_the_hash (2.42s)\n" +
  "    --- PASS: TestContractCheckEnvironmentFindings/environment_findings_are_numbered_after_contradictions_and_both_headlines_are_shown (2.35s)\n" +
  "    --- PASS: TestContractCheckEnvironmentFindings/unevidenced,_refuted_and_uncounted_environment_findings_do_not_refuse (2.84s)\n" +
  "--- PASS: TestContractCheckUnevidencedAndRefutedFindingsNeverRefuse (0.00s)\n" +
  "    --- PASS: TestContractCheckUnevidencedAndRefutedFindingsNeverRefuse/unevidenced_findings_are_recorded_only (2.75s)\n" +
  "    --- PASS: TestContractCheckUnevidencedAndRefutedFindingsNeverRefuse/a_refuted_finding_and_a_finding_without_a_counted_verdict_do_not_refuse (2.83s)\n" +
  "--- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews (0.00s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/a_request_above_the_ceiling_ends_the_review_incomplete_without_a_request (0.16s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/a_text-only_answer (2.81s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/the_request_bound (2.80s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/read-only_tool_rounds_continue_the_same_conversation (2.84s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/an_invalid_confirmation (2.95s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/the_tool_call_bound (3.00s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/an_extra_payload_field (3.07s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/the_submit_call_beside_another_call (2.94s)\n" +
  "    --- PASS: TestContractCheckReadOnlyToolRoundsAndIncompleteReviews/an_invalid_payload (2.86s)\n" +
  "PASS\n";

// NOPASS is the unchanged tree's refusal: the run did not report every scenario.
const NOPASS = ['add', 'sub'].map(p => `- ${p} keeps working (kept) | request: ${p} | problem: no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed`);

// RED is the unchanged tree's observation (HEAD b93f323) of the cases this change changes; its
// "overlay" case, which stays, is the same as the feature's.
const RED = {
  kit: {error: '', events: ['refused'], refusal: NOPASS, runs: 1},
  materialize: {error: '', events: ['refused'], refusal: NOPASS, runs: 1},
  setup: {error: '', events: ['refused'], refusal: NOPASS, runs: 1},
  stale: {contract_error: true, error: 'intent/parse: the Intent cannot be read', events: [], refusal: [], review_properties: 2, runs: 0},
};

// good is an observation that every judgment of the feature accepts.
function good(all) {
  const o = {};
  for (const c of all) {
    o[c.name] = o[c.name] || {};
    if (c.kind === 'kept') continue;
    if (c.kind === 'include') Object.assign(o[c.name], c.want);
    else o[c.name].error = (c.text || 'open /tmp/x/kogen-wrong-implementation-0/kit: not a directory') + (o[c.name].error ? '; ' + o[c.name].error : '');
  }
  return o;
}

function rejects(fn, label) {
  let threw = false;
  try { fn(); } catch (_) { threw = true; }
  assert.ok(threw, 'the observer accepted ' + label);
}

function control(which) {
  const all = checks();
  const want = good(all);
  const run = extra => ({status: 0, signal: null, error: null, output: RUN, json: JSON.stringify(want), command: 'recorded', ...extra});
  const every = o => { for (const c of all) c.run(o); };
  const names = all.filter(c => c.kind === 'kept').flatMap(c => c.names);
  assert.ok(names.length > 0, 'the feature lists existing tests');
  const ran = extra => ({status: 0, signal: null, error: null, output: KEPT, command: 'recorded', ...extra});
  if (which === 'good') {
    every(judge(run()));
    kept(ran(), names);
    return 'accepted';
  }
  rejects(() => kept(ran({status: 1}), names), 'a failed run of the existing tests');
  rejects(() => kept(ran({error: 'spawnSync ETIMEDOUT'}), names), 'a run of the existing tests that did not finish');
  rejects(() => kept(ran({output: KEPT.replace('--- PASS: ' + names[0] + ' ', '--- FAIL: ' + names[0] + ' ')}), names), 'a failed existing test');
  rejects(() => kept(ran({output: KEPT.replace(/^--- PASS: (\S+) .*\n/m, '')}), names), 'an existing test that did not run');
  rejects(() => kept(ran({output: KEPT.replace(/^(\s+)--- PASS: /m, '$1--- FAIL: ')}), names), 'a failed subtest');
  rejects(() => kept(ran({output: KEPT.replace(/^(\s+)--- PASS: /m, '$1--- SKIP: ')}), names), 'a skipped subtest');
  rejects(() => kept(ran({output: KEPT + KEPT}), names), 'existing tests reported twice');
  rejects(() => kept(ran({output: 'testing: warning: no tests to run\nPASS\n'}), names), 'a run that ran no test');
  rejects(() => judge(run({status: 1})), 'a failed run');
  rejects(() => judge(run({error: 'spawnSync ETIMEDOUT'})), 'a run that did not finish');
  rejects(() => judge(run({output: RUN.replace('--- PASS', '--- FAIL')})), 'a failed harness test');
  rejects(() => judge(run({output: RUN.replace('--- PASS', '--- SKIP')})), 'a skipped harness test');
  rejects(() => judge(run({output: RUN + RUN})), 'two harness runs');
  rejects(() => judge(run({output: 'PASS\n'})), 'a run that ran no harness test');
  rejects(() => judge(run({json: null})), 'a run without an observation');
  const obs = judge(run());
  for (const name of Object.keys(RED)) {
    rejects(() => { for (const c of all.filter(c => c.name === name)) c.run({...obs, [name]: RED[name]}); }, `the unchanged tree's case "${name}"`);
    const missing = {...obs}; delete missing[name];
    rejects(() => every(missing), `a missing case "${name}"`);
  }
  for (const [name, c] of Object.entries(obs)) {
    for (const [key, value] of Object.entries(c)) {
      const wrong = typeof value === 'boolean' ? !value : typeof value === 'number' ? value + 1 :
        typeof value === 'string' ? (value === '' ? 'x' : '') : value.length ? value.slice(1) : ['x'];
      rejects(() => every({...obs, [name]: {...c, [key]: wrong}}), `a wrong ${key} of "${name}"`);
      const without = {...c}; delete without[key];
      rejects(() => every({...obs, [name]: without}), `a missing ${key} of "${name}"`);
    }
  }
  rejects(() => every({...obs, setup: {...obs.setup, error: 'setup failed'}}), 'a setup error without the command and its status');
  rejects(() => every({...obs, overlay: {error: 'mkdir calc.js: not a directory', events: [], refusal: [], runs: 1}}), 'an overlay failure returned as an error');
  return 'rejected';
}

if (require.main === module) {
  const which = process.argv[2] === '--control' ? process.argv[3] : '';
  assert.ok(which === 'good' || which === 'bad', 'usage: --control good|bad');
  const outcome = control(which);
  console.log(outcome);
  process.exitCode = outcome === 'accepted' ? 0 : 1;
} else {
  const {Given, When, Then} = require('@cucumber/cucumber');
  let cached = null;
  Given(P + 'the wrong-implementation check harness has run', {timeout: 660000}, function () {
    if (cached === null) cached = harness();
    this.fkswi = cached;
  });
  When(P + 'I run these existing shape tests:', {timeout: 720000}, function (text) {
    this.fkswiNames = text.split('\n').map(l => l.trim()).filter(Boolean);
    this.fkswiKept = existing(this.fkswiNames);
  });
  Then(P + 'each of them is reported as passed exactly once, and no test or subtest failed or was skipped', function () {
    kept(this.fkswiKept, this.fkswiNames);
  });
  Then(P + 'the error of {string} is not empty', function (name) {
    errorNotEmpty(this.fkswi, name);
  });
  Then(P + 'the error of {string} contains {string}', function (name, text) {
    errorContains(this.fkswi, name, text);
  });
  Then(P + 'the observation of {string} includes:', function (name, text) {
    include(this.fkswi, name, JSON.parse(text));
  });
}
