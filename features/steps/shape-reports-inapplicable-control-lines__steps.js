'use strict';
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const P = 'shape-reports-inapplicable-control-lines: ';
const slug = 'shape-reports-inapplicable-control-lines';
const helper = path.resolve('features/steps/' + slug + '__driver/probe_test.go');
const test = 'TestInapplicableControlLinesAcceptance';
let dir, bin;
function applyPatches(original, patches) {
 const work = fs.mkdtempSync(path.join(os.tmpdir(), 'iclp-'));
 fs.mkdirSync(path.join(work, 'shape'));
 fs.writeFileSync(path.join(work, 'shape/controls.go'), original);
 for (const patch of patches) {
  const r = spawnSync('git', ['apply', '-'], {cwd: work, input: patch, encoding: 'utf8'});
  assert.equal(r.status, 0, 'git apply of the controls.json patch: ' + r.stdout + r.stderr);
 }
 const out = fs.readFileSync(path.join(work, 'shape/controls.go'), 'utf8');
 fs.rmSync(work, {recursive: true, force: true});
 return out;
}
function build(impl) {
 dir = fs.mkdtempSync(path.join(os.tmpdir(), 'icl-'));
 const source = path.join(dir, 'probe_test.go');
 fs.writeFileSync(source, fs.readFileSync(helper, 'utf8').replace('//go:build ignore', ''));
 const replace = {[path.resolve('shape/icl_acceptance_test.go')]: source};
 let n = 0;
 for (const [p, c] of Object.entries(impl || {})) {
  const f = path.join(dir, 'file' + (n++) + '.go');
  fs.writeFileSync(f, typeof c === 'string' ? c : applyPatches(fs.readFileSync(path.resolve(p), 'utf8'), c.patches));
  replace[path.resolve(p)] = f;
 }
 const overlay = path.join(dir, 'overlay.json');
 fs.writeFileSync(overlay, JSON.stringify({Replace: replace}));
 bin = path.join(dir, 'shape.test');
 const r = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {encoding: 'utf8', timeout: 180000});
 if (r.status !== 0) throw new Error('go test -c ./shape for ' + test + ': ' + r.stdout + r.stderr);
}
function observe(input) {
 if (!bin) build();
 const inFile = path.join(dir, 'in.json'), out = path.join(dir, 'out.json');
 fs.rmSync(out, {force: true});
 fs.writeFileSync(inFile, JSON.stringify(input));
 const r = spawnSync(bin, ['-test.run', '^' + test + '$', '-test.count=1', '-test.v'], {encoding: 'utf8', timeout: 100000, env: {...process.env, ICL_IN: inFile, ICL_OUT: out}});
 assert.equal(r.status, 0, test + ' built binary exited ' + r.status + ': ' + r.stdout + r.stderr);
 assert.ok(fs.existsSync(out), test + ' ran zero times, was skipped or wrote no result: ' + r.stdout + r.stderr);
 return JSON.parse(fs.readFileSync(out, 'utf8')).lines;
}
function refusal(lines) {
 assert.equal(lines.length, 1, 'expected exactly one refusal line, got: ' + JSON.stringify(lines));
 assert.ok(!lines[0].includes('\n'), 'the refusal is one line: ' + lines[0]);
 return lines[0];
}
function lineTexts(content) { return content.split('\n').filter((l, i, a) => i < a.length - 1 || l !== ''); }
function esc(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }
function rest(world) {
 const m = world.iclLine.indexOf(world.iclHeader || '@@');
 assert.ok(m >= 0, 'the hunk header is named in: ' + world.iclLine);
 return world.iclLine.slice(m + (world.iclHeader || '').length);
}
function judgeExcerpt(world, from, to) {
 const texts = lineTexts(world.iclContent);
 let tail = rest(world);
 for (let n = from; n <= to; n++) {
  const m = new RegExp('(^|[^\\w])' + n + '[^\\w]{1,4}' + esc(texts[n - 1]) + '(?![\\w])').exec(tail);
  assert.ok(m, 'line ' + n + ' (' + texts[n - 1] + ') is shown with its number, in order, in: ' + world.iclLine);
  const before = tail.slice(0, m.index);
  if (n === from) {
   for (let k = 1; k < from; k++) assert.ok(!new RegExp('(?<![\\w-])' + esc(texts[k - 1]) + '(?![\\w])').test(before + m[0]), 'line ' + k + ' is before the hunk start in: ' + world.iclLine);
  }
  tail = tail.slice(m.index + m[0].length);
 }
 assert.ok(!/(^|[^\w])\d+[^\w]{1,4}[a-z]/.test(tail), 'no numbered line follows line ' + to + ': ' + world.iclLine);
 assert.ok(!tail.includes('padding line') && !world.iclLine.includes('padding line'), 'no line of another file is shown: ' + world.iclLine);
 if (to < texts.length) assert.ok(!tail.includes(texts[to]), 'line ' + (to + 1) + ' is not shown: ' + world.iclLine);
 assert.ok(to - from + 1 <= 8, 'at most eight lines');
}
function cleanup() { if (dir) fs.rmSync(dir, {recursive: true, force: true}); }
process.on('exit', cleanup);
if (require.main === module) {
 const mode = process.argv[2];
 const cf = JSON.parse(fs.readFileSync('.kogen/intents/' + slug + '/controls.json', 'utf8'));
 const patches = [cf.correct.files['shape/controls.go'].patch];
 const wrongIndex = {good: -1, bad: 0, 'bad-wrong-only': 1, 'bad-missing': 2, 'bad-pad': 3, 'bad-clean': 4}[mode];
 if (wrongIndex >= 0) patches.push(cf.wrong[wrongIndex].files['shape/controls.go'].patch);
 build({'shape/controls.go': {patches}});
 const twenty = 'apple\nbirch\ncedar\ndune\nelm\nfern\ngrove\nheath\niris\njuniper\nkelp\nlarch\nmaple\nnettle\noak\npine\nquince\nreed\nspruce\nthistle\n';
 const patch = '--- a/notes.txt\n+++ b/notes.txt\n@@ -10,5 +10,5 @@\n juniper\n kelp\n DIFFERENT\n-maple\n+MAPLE\n nettle\n';
 const exercises = [
  () => { const w = {iclContent: twenty, iclHeader: '@@ -10,5 +10,5 @@'}; w.iclLine = refusal(observe({mode: 'correct', target: 'notes.txt', content: twenty, patch})); assert.ok(w.iclLine.startsWith('- controls.json: the correct implementation: notes.txt: the patch does not apply to the candidate tree: error: patch failed: notes.txt:10')); judgeExcerpt(w, 10, 17); },
  () => { const w = {iclContent: twenty, iclHeader: '@@ -10,5 +10,5 @@'}; w.iclLine = refusal(observe({mode: 'wrong', target: 'notes.txt', content: twenty, patch})); assert.ok(w.iclLine.startsWith('- controls.json: wrong implementation "add removed": notes.txt: the patch does not apply to the candidate tree with the correct implementation applied: error: patch failed: notes.txt:10')); judgeExcerpt(w, 10, 17); },
  () => { const l = refusal(observe({mode: 'correct', target: 'missing.txt', content: null, patch: '--- a/missing.txt\n+++ b/missing.txt\n@@ -2,3 +2,3 @@\n birch\n-cedar\n+CEDAR\n dune\n'})); assert.ok(l.includes('@@ -2,3 +2,3 @@') && l.includes('does not exist in the candidate tree'), l); },
  () => { const five = 'apple\nbirch\ncedar\ndune\nelm\n'; const w = {iclContent: five, iclHeader: '@@ -3,3 +3,3 @@'}; w.iclLine = refusal(observe({mode: 'correct', target: 'short.txt', content: five, patch: '--- a/short.txt\n+++ b/short.txt\n@@ -3,3 +3,3 @@\n cedar\n DIFFERENT\n-elm\n+ELM\n'})); judgeExcerpt(w, 3, 5); },
  () => { assert.deepEqual(observe({mode: 'correct', target: 'notes.txt', content: 'apple\nbirch\ncedar\ndune\nelm\n', patch: '--- a/notes.txt\n+++ b/notes.txt\n@@ -2,3 +2,3 @@\n birch\n-cedar\n+CEDAR\n dune\n'}), []); },
 ];
 try {
  exercises.forEach(f => f());
  console.log('observer accepted');
 } catch (e) { if (e.code !== 'ERR_ASSERTION') throw e; console.log('observer rejected'); process.exitCode = 1; }
} else {
 const {Given, When, Then} = require('@cucumber/cucumber');
 Given(P + 'the candidate tree has the file {string} with these lines:', function (file, doc) { this.iclFile = file; this.iclContent = doc + '\n'; });
 Given(P + 'the candidate tree has no file {string}', function (file) { this.iclFile = file; this.iclContent = null; });
 function patches(world, mode, file, doc) {
  assert.equal(file, world.iclFile, 'the patch targets the file of the candidate tree');
  world.iclLines = observe({mode, target: file, content: world.iclContent, patch: doc + '\n'});
  if (world.iclLines.length > 0) world.iclLine = refusal(world.iclLines);
 }
 When(P + 'the correct implementation patches {string} with:', function (file, doc) { patches(this, 'correct', file, doc); });
 When(P + 'a wrong implementation patches {string} with:', function (file, doc) { patches(this, 'wrong', file, doc); });
 Then(P + 'the refusal line starts with:', function (doc) { assert.ok(this.iclLine && this.iclLine.startsWith(doc), 'refusal line should start with ' + JSON.stringify(doc) + ' but is ' + JSON.stringify(this.iclLine)); });
 Then(P + 'the refusal line then names the hunk header {string}', function (header) { this.iclHeader = header; assert.ok(this.iclLine.includes(header), 'the line names the hunk header ' + header + ': ' + this.iclLine); });
 Then(P + 'the refusal line then shows numbered lines {int} to {int} of the file in order and no others', function (from, to) { judgeExcerpt(this, from, to); });
 Then(P + 'the refusal line then says the file does not exist in the candidate tree and shows no numbered lines', function () {
  assert.ok(this.iclLine.includes('does not exist in the candidate tree'), 'the line says the file does not exist in the candidate tree: ' + this.iclLine);
  assert.ok(!/(^|[^\w])\d+[^\w]{1,4}[a-z]/.test(this.iclLine.slice(this.iclLine.indexOf(this.iclHeader) + this.iclHeader.length)), 'no numbered lines: ' + this.iclLine);
 });
 Then(P + 'no refusal line is reported', function () { assert.deepEqual(this.iclLines, []); });
}
