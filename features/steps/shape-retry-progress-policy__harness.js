'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');
const cp = require('child_process');
const assert = require('assert/strict');
const root = process.cwd();
const prefix = 'features/steps/shape-retry-progress-policy__driver/driver.go';
const rule = 'Mark an assumption with (assumption: <what the user must confirm>) before the required trailing tag, e.g. - <decision> (assumption: <what the user must confirm>) (source: request); every Decision line keeps its trailing (source: request) or (source: answer N).';
let directory, binary;
function build(overrides = {}) {
 directory = fs.mkdtempSync(path.join(os.tmpdir(), 'srpp-'));
 const replace = {};
 const fallback = !fs.existsSync('shape/retrybudget.go') && !overrides['shape/retrybudget.go'] ? {'shape/retrybudget.go': 'package shape\ntype retryBudget struct { limit, consecutive, total int }\nfunc (b *retryBudget) refuse(gate string) bool { return false }\n'} : {};
 const files = {...fallback, ...overrides, 'shape/shape_retry_progress_observation_test.go': fs.readFileSync(prefix, 'utf8').replace('//go:build ignore\n\n', '')};
 for (const [name, text] of Object.entries(files)) {
  const target = path.join(directory, String(Object.keys(replace).length) + '.go');
  fs.writeFileSync(target, text);
  replace[path.join(root, name)] = target;
 }
 const overlay = path.join(directory, 'overlay.json');
 fs.writeFileSync(overlay, JSON.stringify({Replace: replace}));
 binary = path.join(directory, 'observe');
 const r = cp.spawnSync('go', ['test', '-overlay', overlay, '-c', '-o', binary, './shape'], {encoding: 'utf8', timeout: 180000});
 assert.equal(r.status, 0, 'observation driver compilation failed: ' + r.stdout + r.stderr);
}
function observe(mode, values = {}) {
 if (!binary) build();
 const output = path.join(directory, 'out.json');
 fs.rmSync(output, {force:true});
 const env = {...process.env, SRPP_MODE:mode, SRPP_OUT:output, SRPP_GATES:'', SRPP_STATEMENT:'', SRPP_QUOTE:'', SRPP_INTENT:'', ...values};
 const r = cp.spawnSync(binary, ['-test.run=^TestShapeRetryProgressObservation$'], {env, encoding:'utf8', timeout:30000});
 assert.equal(r.status, 0, 'driver execution failed: ' + r.stdout + r.stderr);
 assert.ok(fs.existsSync(output), 'observation driver did not produce its result');
 return JSON.parse(fs.readFileSync(output, 'utf8'));
}
function trace(actual, expected) { assert.equal(actual, expected); }
function feedback(actual, text, presence) {
 assert.ok(actual.feedback.join('\n').includes(text), 'missing feedback: ' + text);
 assert.equal(actual.feedback.includes(rule), presence === 'present', 'assumption instruction presence');
 assert.equal(actual.event.unsupported, 1, 'unsupported verdict must remain unchanged');
}
function cleanup() { if (directory) fs.rmSync(directory, {recursive:true, force:true}); directory = binary = undefined; }
module.exports = {observe, trace, feedback, cleanup};
if (require.main === module) {
 try {
  const controls = JSON.parse(fs.readFileSync('.kogen/intents/shape-retry-progress-policy/controls.json', 'utf8'));
  const files = {...controls.correct.files};
  if (process.argv[2] === 'bad') Object.assign(files, controls.wrong.find(x => x.name === 'omit silent assumption').files);
  build(files);
  trace(observe('budget', {SRPP_GATES:'decision_record,carried_decisions,decision_check'}), '1:1:false,1:2:false,1:3:false');
  trace(observe('budget', {SRPP_GATES:'decision_record,decision_record,decision_record'}), '1:1:false,2:2:false,3:3:true');
  trace(observe('budget', {SRPP_GATES:'decision_record,carried_decisions,decision_record,carried_decisions,decision_record,carried_decisions'}), '1:1:false,1:2:false,1:3:false,1:4:false,1:5:false,1:6:true');
  const statement = 'Report the initialized directory.';
  const fix = 'the request speaks to this in the quoted passages; make the statement follow them or remove it; an assumption does not override request text.';
  feedback(observe('decision', {SRPP_STATEMENT:statement, SRPP_QUOTE:'Initialized Kogen in <path>', SRPP_INTENT:'heading\n'+statement}), 'fix for intent.md line 2: '+fix, 'absent');
  feedback(observe('decision', {SRPP_STATEMENT:statement, SRPP_QUOTE:'Initialized Kogen in <path>', SRPP_INTENT:'heading'}), 'fix for the quoted statement: '+fix, 'absent');
  feedback(observe('decision', {SRPP_STATEMENT:statement, SRPP_INTENT:'heading\n'+statement}), 'verdict: unsupported', 'present');
  console.log('observer accepted');
 } catch (e) { console.log('observer rejected'); process.exitCode = 1; }
 finally { cleanup(); }
}
