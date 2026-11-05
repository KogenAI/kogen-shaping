'use strict';
// Harness of SH-CONSUMES-TOOLS-REQUIRE-EXECUTABLE. It runs the two consumer tests of
// consumes/kogen-verification with `go test -json` in the checkout, against the
// kogen-verification that go.mod pins or against a scratch copy of that pinned module with a
// named variant's patches applied (through a scratch -modfile that replaces the module), and
// returns each test's outcome. It needs no Cucumber, so the control runs it too.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const MODULE = 'github.com/KogenAI/kogen-verification';
const VERSION = 'v0.0.0-20261105003813-ac4f6a9a4877';
const PACKAGE = './consumes/kogen-verification/';
const TESTS = ['TestReadinessCheckAndKeptEvidence', 'TestToolsFromInstall'];

// Stand-in for the unpublished producer Build. It is not that Build's source. It reproduces the
// two reported failures on the unchanged tests:
// - Tools.Require refuses a Node that is missing or not an executable regular file. A missing
//   Node uses the reported detail "node is not available: node does not exist; reinstall Kogen".
// - Readiness.Check calls Tools.Require first when the tools are not the zero value, which is
//   the only Check path among these tests that sets tools and the only one the report shows failing.
// - Without a node-path file, ToolsFromInstall gives an empty Node, so tools_test.go:58's exact
//   file-path comparison fails. The readable node-path text is unchanged.
// Each patch is [file under verify/, exact text occurring once, replacement].
const NEW_REQUIRE = [
  ['env.go',
    '\tif !isRegular(t.Node) {\n\t\treturn sys.Fail("environment", "tool_missing", "node is not available at "+t.Node+"; reinstall Kogen", 3)\n',
    '\tif st, err := os.Stat(t.Node); err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {\n\t\twhy := "node does not exist"\n\t\tif err == nil {\n\t\t\twhy = t.Node + " is not executable"\n\t\t}\n\t\treturn sys.Fail("environment", "tool_missing", "node is not available: "+why+"; reinstall Kogen", 3)\n'],
  ['env.go',
    '\tnode := filepath.Join(installRoot, "node-path")\n\tif b, err := os.ReadFile(node); err == nil {\n',
    '\tnode := ""\n\tif b, err := os.ReadFile(filepath.Join(installRoot, "node-path")); err == nil {\n'],
  ['readiness.go',
    '\tos.Remove(readinessPath(r.Root, slug))\n\tvar in inputs\n',
    '\tif r.Tools != (Tools{}) {\n\t\tif err := r.Tools.Require(); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\tos.Remove(readinessPath(r.Root, slug))\n\tvar in inputs\n'],
];

const ALTER_NODE_PATH = ['env.go', '\t\tnode = strings.TrimSpace(string(b))\n', '\t\tnode = strings.TrimSpace(string(b)) + "-altered"\n'];

const VARIANTS = {
  'pinned': [],
  'pinned with the node-path text altered': [ALTER_NODE_PATH],
  'new Require': NEW_REQUIRE,
  'new Require with the node-path text altered': [...NEW_REQUIRE, ALTER_NODE_PATH],
  'new Require without KOGEN_NODE in commands': [...NEW_REQUIRE, ['env.go', '"KOGEN_NODE": t.Node,', '"KOGEN_NODE": "",']],
  'new Require resolving a missing node-path to /bin/sh': [...NEW_REQUIRE, ['env.go', '\tnode := ""\n', '\tnode := "/bin/sh"\n']],
};

function go(root, args) {
  const r = spawnSync('go', args, {cwd: root, encoding: 'utf8', maxBuffer: 256 * 1024 * 1024, timeout: 5 * 60 * 1000});
  if (r.error) throw new Error(`go ${args.join(' ')}: ${r.error.message}`);
  return {status: r.status, signal: r.signal, stdout: r.stdout || '', stderr: r.stderr || ''};
}

// pinnedDir is the module cache directory of the kogen-verification go.mod pins.
function pinnedDir(root) {
  const args = ['mod', 'download', '-json', MODULE];
  const r = go(root, args);
  if (r.status !== 0) throw new Error(`go ${args.join(' ')} exited ${r.status}:\n${r.stdout}${r.stderr}`);
  const info = JSON.parse(r.stdout);
  if (info.Version !== VERSION || !info.Dir || !fs.existsSync(path.join(info.Dir, 'verify', 'env.go'))) {
    throw new Error(`go ${args.join(' ')} gave no ${MODULE}@${VERSION} directory with verify/env.go: ${r.stdout}`);
  }
  return info.Dir;
}

function makeWritable(p) {
  const st = fs.lstatSync(p);
  if (st.isSymbolicLink()) return;
  fs.chmodSync(p, st.mode | 0o200);
  if (st.isDirectory()) for (const e of fs.readdirSync(p)) makeWritable(path.join(p, e));
}

// patchedModfile copies the pinned module into work/kogen-verification, applies patches and
// writes work/go.mod (the checkout's go.mod replacing the module with the copy) and work/go.sum.
function patchedModfile(root, work, patches) {
  const copy = path.join(work, 'kogen-verification');
  fs.cpSync(pinnedDir(root), copy, {recursive: true});
  makeWritable(copy);
  for (const [file, from, to] of patches) {
    const p = path.join(copy, 'verify', file);
    const text = fs.readFileSync(p, 'utf8');
    const n = text.split(from).length - 1;
    if (n !== 1) throw new Error(`the patch of verify/${file} expects ${JSON.stringify(from)} once in ${MODULE}@${VERSION}, found it ${n} times`);
    fs.writeFileSync(p, text.replace(from, () => to));
  }
  const modfile = path.join(work, 'go.mod');
  fs.writeFileSync(modfile, fs.readFileSync(path.join(root, 'go.mod'), 'utf8') + `\nreplace ${MODULE} => ${copy}\n`);
  fs.copyFileSync(path.join(root, 'go.sum'), path.join(work, 'go.sum'));
  return modfile;
}

// runConsumerTests runs TESTS against the named variant and returns the command, its exit
// status, each test's final action (pass, fail or skip; absent when it did not run), each
// test's output, the raw `go test -json` run and outcome rows of TESTS, and the output of
// everything else (build errors, package lines).
function runConsumerTests(root, variant) {
  const patches = VARIANTS[variant];
  if (!patches) throw new Error(`unknown kogen-verification variant ${JSON.stringify(variant)}; known: ${Object.keys(VARIANTS).join(', ')}`);
  const args = ['test', '-count=1', '-json', '-run', `^(${TESTS.join('|')})$`];
  const work = patches.length ? fs.mkdtempSync(path.join(os.tmpdir(), 'sh-consumes-tools-')) : null;
  try {
    if (work) args.push(`-modfile=${patchedModfile(root, work, patches)}`);
    args.push(PACKAGE);
    const r = go(root, args);
    const outcomes = {}, output = {}, rows = [], other = [];
    for (const line of r.stdout.split('\n')) {
      if (!line.trim()) continue;
      let ev;
      try { ev = JSON.parse(line); } catch { other.push(line + '\n'); continue; }
      if (ev.Test && TESTS.includes(ev.Test)) {
        if (ev.Action === 'output') output[ev.Test] = (output[ev.Test] || '') + ev.Output;
        if (['run', 'pass', 'fail', 'skip'].includes(ev.Action)) rows.push(line.trim());
        if (['pass', 'fail', 'skip'].includes(ev.Action)) outcomes[ev.Test] = ev.Action;
      } else if (ev.Output) other.push(ev.Output);
    }
    return {variant, command: `go ${args.join(' ')}`, status: r.status, signal: r.signal, outcomes, output, rows, other: other.join('') + r.stderr};
  } finally {
    if (work) fs.rmSync(work, {recursive: true, force: true});
  }
}

function tail(text) {
  return text.length > 6000 ? '...' + text.slice(-6000) : text;
}

function describe(result, test) {
  const outcome = result.outcomes[test];
  return `${test} ${outcome ? 'ended with ' + outcome : 'did not run'} against the ${JSON.stringify(result.variant)} kogen-verification ` +
    `(${result.command} exited ${result.status}${result.signal ? ' (' + result.signal + ')' : ''})\n` +
    tail((result.output[test] || '') + (outcome ? '' : result.other));
}

// withRows ends a nonempty problem list with the raw go test -json rows of TESTS, so a
// failure shows which named tests ran and how they ended.
function withRows(result, problems) {
  if (!problems.length) return problems;
  return [...problems, `go test -json rows of ${TESTS.join(' and ')} (${result.rows.length}):\n${result.rows.join('\n')}`];
}

// passProblems is empty when every one of TESTS passed.
function passProblems(result) {
  return withRows(result, TESTS.filter(test => result.outcomes[test] !== 'pass').map(test => `want pass: ${describe(result, test)}`));
}

// failProblems is empty when test ran and failed.
function failProblems(result, test) {
  if (!TESTS.includes(test)) throw new Error(`${test} is not one of ${TESTS.join(', ')}`);
  return withRows(result, result.outcomes[test] === 'fail' ? [] : [`want fail: ${describe(result, test)}`]);
}

module.exports = {MODULE, VERSION, TESTS, VARIANTS, runConsumerTests, passProblems, failProblems};
