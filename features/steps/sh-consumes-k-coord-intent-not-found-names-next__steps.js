'use strict';
// Acceptance of sh-consumes-k-coord-intent-not-found-names-next: kogen-shaping follows
// kogen-coordinator's k-coord-intent-not-found-names-next-command contract (project.RequireIntentDir
// names the slug, the folder it looked in and the next command). The steps read go.mod and go.sum
// and run real `go test -count=1 -v` on packages of this module, either with the module files as
// checked out or with a scratch -modfile that requires another kogen-coordinator version, and judge
// the per-test `--- PASS:` / `--- FAIL:` lines of that output. Nothing in the checkout is written.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const assert = require('node:assert/strict');
const {When, Then} = require('@cucumber/cucumber');

const P = 'sh-consumes-k-coord-intent-not-found-names-next: ';
const COORDINATOR = 'github.com/KogenAI/kogen-coordinator';

// goEnv is the caller's environment for the go command, without the acceptance runner's own
// variables and offline. A scratch -modfile run may add go.sum lines to its own scratch copy.
function goEnv(flags) {
  const e = {...process.env, GOWORK: 'off', GOPROXY: 'off', GOSUMDB: 'off', GOFLAGS: flags};
  for (const k of Object.keys(e)) if (k.startsWith('KOGEN_ACCEPTANCE_')) delete e[k];
  return e;
}

function goTest(args, flags) {
  const command = 'go ' + args.join(' ');
  const r = spawnSync('go', args, {env: goEnv(flags), encoding: 'utf8', timeout: 540000, maxBuffer: 64 * 1024 * 1024});
  return {command, status: r.status, error: r.error ? String(r.error) : '', output: (r.stdout || '') + (r.stderr || '')};
}

function tail(text) {
  const lines = text.split('\n');
  return lines.slice(Math.max(0, lines.length - 60)).join('\n');
}

function escapeRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }

function runPattern(names) {
  const list = names.split(/\s+/).filter(Boolean);
  return '^(' + list.join('|') + ')$';
}

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

When(P + 'I run the Go tests {string} of package {string} with the module files as checked out', {timeout: 600000}, function (names, pkg) {
  this.run = goTest(['test', '-count=1', '-v', '-run', runPattern(names), pkg], '-mod=readonly');
});

// A scratch copy of go.mod (and go.sum) in its own directory requires the given kogen-coordinator
// version; the checkout's files are not touched.
When(P + 'I run the Go tests {string} of package {string} against kogen-coordinator {string}', {timeout: 600000}, function (names, pkg, version) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sh-consumes-modfile-'));
  try {
    const mod = fs.readFileSync('go.mod', 'utf8');
    const re = new RegExp('^(\\s*(?:require\\s+)?' + escapeRe(COORDINATOR) + '\\s+)\\S+', 'm');
    assert.ok(re.test(mod), 'go.mod has no requirement of ' + COORDINATOR);
    fs.writeFileSync(path.join(dir, 'alt.mod'), mod.replace(re, '$1' + version));
    fs.writeFileSync(path.join(dir, 'alt.sum'), fs.readFileSync('go.sum', 'utf8'));
    this.run = goTest(['test', '-count=1', '-v', '-modfile=' + path.join(dir, 'alt.mod'), '-run', runPattern(names), pkg], '-mod=mod');
  } finally {
    fs.rmSync(dir, {recursive: true, force: true});
  }
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

Then(P + 'the test {string} failed', function (name) {
  const r = this.run;
  const got = outcome(r.output, name);
  assert.equal(got, 'failed', `${name}: ${got} in ${r.command} (exit=${r.status} ${r.error}), want failed on an assertion\n${tail(r.output)}`);
});

Then(P + 'go.mod requires {string} at {string}', function (name, version) {
  assert.equal(requireLine(name), version, `go.mod requires ${name} at ${requireLine(name)}, want ${version}`);
});

Then(P + 'go.mod has no replace directive', function () {
  assert.ok(!/^\s*replace\b/m.test(fs.readFileSync('go.mod', 'utf8')), 'go.mod has a replace directive');
});

Then(P + 'go.sum has the line {string}', function (line) {
  const lines = fs.readFileSync('go.sum', 'utf8').split('\n');
  assert.ok(lines.includes(line), `go.sum has no line ${JSON.stringify(line)}`);
});

Then(P + 'go.sum lists {string} only at {string}', function (name, version) {
  const found = fs.readFileSync('go.sum', 'utf8').split('\n').filter(l => l.startsWith(name + ' '));
  assert.equal(found.length, 2, `go.sum has ${found.length} lines for ${name}, want the module and go.mod hash lines: ${JSON.stringify(found)}`);
  for (const l of found) {
    assert.ok(l.startsWith(name + ' ' + version + ' ') || l.startsWith(name + ' ' + version + '/go.mod '),
      `go.sum line ${JSON.stringify(l)} is not at ${version}`);
  }
});
