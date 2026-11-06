'use strict';
// Acceptance of sh-consumes-c-models-only-in-models-yaml: kogen-shaping follows kogen-coordinator's
// c-models-only-in-models-yaml contract (model roles only in .kogen/models.yaml). The steps read
// go.mod and go.sum, run real `go test -count=1 -v` on packages of this module and judge the
// per-test `--- PASS:` lines of that output, and read the module's Go files. Nothing in the
// checkout is written.
const fs = require('node:fs');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {When, Then} = require('@cucumber/cucumber');

const P = 'sh-consumes-c-models-only-in-models-yaml: ';

// goEnv is the caller's environment for the go command, without the acceptance runner's own
// variables, offline and with the module files read-only.
function goEnv() {
  const e = {...process.env, GOWORK: 'off', GOPROXY: 'off', GOFLAGS: '-mod=readonly'};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function goTest(args) {
  const command = 'go ' + args.join(' ');
  const r = spawnSync('go', args, {env: goEnv(), encoding: 'utf8', timeout: 540000, maxBuffer: 64 * 1024 * 1024});
  return {command, status: r.status, error: r.error ? String(r.error) : '', output: (r.stdout || '') + (r.stderr || '')};
}

function tail(text) {
  const lines = text.split('\n');
  return lines.slice(Math.max(0, lines.length - 60)).join('\n');
}

function escapeRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }

// outcome is what the go test output says of the named test or subtest: passed, failed,
// skipped or did not run.
function outcome(output, name) {
  const m = output.match(new RegExp('^\\s*--- (PASS|FAIL|SKIP): ' + escapeRe(name) + ' \\(', 'm'));
  return m ? {PASS: 'passed', FAIL: 'failed', SKIP: 'skipped'}[m[1]] : 'did not run';
}

function requireLine(name) {
  const text = fs.readFileSync('go.mod', 'utf8');
  const m = text.match(new RegExp('^\\s*(?:require\\s+)?' + escapeRe(name) + '\\s+(\\S+)', 'm'));
  return m ? m[1] : null;
}

When(P + 'every Go test of package {string} is run', {timeout: 600000}, function (pkg) {
  this.run = goTest(['test', '-count=1', '-v', pkg]);
});

When(P + 'the Go tests {string} of package {string} are run', {timeout: 600000}, function (names, pkg) {
  const list = names.split(/\s+/).filter(Boolean);
  this.run = goTest(['test', '-count=1', '-v', '-run', '^(' + list.join('|') + ')$', pkg]);
});

Then(P + 'the run passed', function () {
  const r = this.run;
  assert.ok(r.status === 0 && /^ok\s/m.test(r.output) && !/^\s*--- FAIL: /m.test(r.output),
    `${r.command}: exit=${r.status} ${r.error}, want exit 0, an "ok" package line and no failed test\n${tail(r.output)}`);
});

Then(P + 'the test {string} passed', function (name) {
  const r = this.run;
  const got = outcome(r.output, name);
  assert.equal(got, 'passed', `${name}: ${got} in ${r.command} (exit=${r.status} ${r.error}), want passed\n${tail(r.output)}`);
});

Then(P + 'go.mod requires {string} at {string}', function (name, version) {
  assert.equal(requireLine(name), version, `go.mod requires ${name} at ${requireLine(name)}, want ${version}`);
});

Then(P + 'go.sum has the line {string}', function (line) {
  const lines = fs.readFileSync('go.sum', 'utf8').split('\n');
  assert.ok(lines.includes(line), `go.sum has no line ${JSON.stringify(line)}`);
});

Then(P + 'go.sum has no line for {string} at {string}', function (name, version) {
  const found = fs.readFileSync('go.sum', 'utf8').split('\n').filter(l => l.startsWith(name + ' ' + version + ' ') || l.startsWith(name + ' ' + version + '/go.mod '));
  assert.deepEqual(found, [], `go.sum still has lines for ${name} ${version}`);
});

Then(P + 'no Go file of the module mentions {string}', function (text) {
  const r = spawnSync('git', ['ls-files', '-z', '--cached', '--others', '--exclude-standard', '--', '*.go'], {encoding: 'utf8'});
  assert.equal(r.status, 0, `git ls-files: exit=${r.status}\n${r.stderr}`);
  const files = r.stdout.split('\0').filter(p => p && !p.startsWith('.kogen/') && !p.startsWith('testdata/tools/') && fs.existsSync(p));
  assert.ok(files.length > 0, 'git ls-files listed no Go file');
  const hits = files.filter(p => fs.readFileSync(p, 'utf8').includes(text));
  assert.deepEqual(hits, [], `Go files mentioning ${JSON.stringify(text)}`);
});
