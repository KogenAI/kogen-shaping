'use strict';
// Steps of SHELL-QUOTE-METACHARACTER-TESTS. They copy go.mod, go.sum and the shape package into a
// scratch directory, optionally replace the body of ShellQuote in the copy with a wrong version, and
// run only the new Go tests of shape/context_test.go that call ShellQuote there. The checkout is
// never modified.
const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');
const {Given, When, Then} = require('@cucumber/cucumber');

const P = 'shell-quote-metacharacter-tests: ';

const HEADER = 'func ShellQuote(value string) string {';

// The test that exists before this change and compares the quoted text with a fixed string; it is
// left out so that only the new tests judge the wrong versions.
const EXISTING = 'TestShapeCommonHelpers';

const MUTANTS = {
  'an empty value gives no argument': `func ShellQuote(value string) string {
	if value == "" {
		return ""
	}
	return "'" + strings.ReplaceAll(value, "'", \`'\\''\`) + "'"
}
`,
  'spaces split the value': `func ShellQuote(value string) string {
	return strings.ReplaceAll("'"+strings.ReplaceAll(value, "'", \`'\\''\`)+"'", " ", "' '")
}
`,
  'an apostrophe is escaped with a backslash': `func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", \`\\'\`) + "'"
}
`,
  'an apostrophe is doubled': `func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
`,
  'double quotes leave shell syntax active': `func ShellQuote(value string) string {
	return "\\"" + strings.ReplaceAll(value, "\\"", "\\\\\\"") + "\\""
}
`,
  'the quote ends at a line break': `func ShellQuote(value string) string {
	value, _, _ = strings.Cut(value, "\\n")
	return "'" + strings.ReplaceAll(value, "'", \`'\\''\`) + "'"
}
`,
};

// shellQuoteTests lists the test functions of a Go test file, other than the pre-existing
// TestShapeCommonHelpers, whose body calls ShellQuote.
function shellQuoteTests(file) {
  const text = fs.readFileSync(file, 'utf8');
  const parts = text.split(/^func /m).slice(1);
  const names = [];
  for (const part of parts) {
    const m = /^(Test\w+)\(/.exec(part);
    if (m && m[1] !== EXISTING && /(^|[^.\w])ShellQuote\(/.test(part)) names.push(m[1]);
  }
  return names;
}

// mutate replaces the whole ShellQuote function of the scratch copy with a wrong version.
function mutate(dir, name) {
  const file = path.join(dir, 'shape', 'common.go');
  const src = fs.readFileSync(file, 'utf8');
  const start = src.indexOf(HEADER);
  assert.ok(start >= 0, 'shape/common.go has no ' + HEADER);
  const end = src.indexOf('\n}\n', start) + 3;
  const body = MUTANTS[name];
  assert.ok(body, 'unknown wrong ShellQuote ' + JSON.stringify(name));
  fs.writeFileSync(file, src.slice(0, start) + body + src.slice(end));
}

Given(P + 'a scratch copy of the shape package', function () {
  this.dir = fs.mkdtempSync(path.join(os.tmpdir(), 'shell-quote-tests-'));
  for (const f of ['go.mod', 'go.sum']) fs.copyFileSync(f, path.join(this.dir, f));
  fs.cpSync('shape', path.join(this.dir, 'shape'), {recursive: true});
});

When(P + 'the new tests of shape\\/context_test.go that call ShellQuote run against {string}', {timeout: 115 * 1000}, function (variant) {
  if (variant !== 'the unchanged ShellQuote') mutate(this.dir, variant);
  const names = shellQuoteTests(path.join(this.dir, 'shape', 'context_test.go'));
  this.names = names;
  this.variant = variant;
  if (names.length === 0) {
    this.run = {status: null, output: ''};
    return;
  }
  const r = spawnSync('go', ['test', '-count=1', '-v', './shape', '-run', '^(' + names.join('|') + ')$'], {
    cwd: this.dir, encoding: 'utf8', timeout: 100 * 1000,
  });
  this.run = {status: r.status, output: (r.stdout || '') + (r.stderr || '') + (r.error ? String(r.error) : '')};
});

Then(P + 'such tests exist and they all pass', function () {
  assert.ok(this.names.length >= 1,
    'shape/context_test.go has ' + this.names.length + ' new test(s) calling ShellQuote (' + this.names.join(', ') + '), want at least 1');
  assert.strictEqual(this.run.status, 0,
    'go test of ' + this.names.join(', ') + ' against ' + this.variant + ' exited ' + this.run.status + ':\n' + this.run.output.slice(-1500));
  for (const n of this.names) {
    assert.ok(this.run.output.includes('--- PASS: ' + n), n + ' did not run and pass:\n' + this.run.output.slice(-1500));
  }
});

Then(P + 'at least one such test fails', function () {
  assert.ok(this.names.length > 0, 'shape/context_test.go has no new test calling ShellQuote');
  assert.notStrictEqual(this.run.status, 0,
    'go test of ' + this.names.join(', ') + ' still passed against the wrong ShellQuote ' + JSON.stringify(this.variant));
  assert.ok(/--- FAIL: Test/.test(this.run.output),
    'go test against ' + this.variant + ' failed without a failing test (build error?):\n' + this.run.output.slice(-1500));
});
