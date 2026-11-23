'use strict';
// Steps of EXIT-CODE-WRAPPED-ERRORS-TESTED. They copy go.mod, go.sum and the shape package into a
// scratch directory, optionally replace the body of ExitCode in the copy with a wrong version, and
// run only the Go tests of shape/context_test.go that call ExitCode there. The checkout is never
// modified.
const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');
const {Given, When, Then} = require('@cucumber/cucumber');

const P = 'exit-code-wrapped-errors-tested: ';

const HEADER = 'func ExitCode(err error) int {';

const MUTANTS = {
  'direct type assertion for a Kogen error': `func ExitCode(err error) int {
	if e, ok := err.(*sys.Error); ok {
		return e.Exit
	}
	var i *sys.Interrupted
	if errors.As(err, &i) {
		return i.ExitCode()
	}
	return 70
}
`,
  'no interruption handling': `func ExitCode(err error) int {
	var e *sys.Error
	if errors.As(err, &e) {
		return e.Exit
	}
	return 70
}
`,
  'direct type assertion for an interruption': `func ExitCode(err error) int {
	var e *sys.Error
	if errors.As(err, &e) {
		return e.Exit
	}
	if i, ok := err.(*sys.Interrupted); ok {
		return i.ExitCode()
	}
	return 70
}
`,
  'one fixed code for every interruption': `func ExitCode(err error) int {
	var e *sys.Error
	var i *sys.Interrupted
	switch {
	case errors.As(err, &e):
		return e.Exit
	case errors.As(err, &i):
		return 130
	}
	return 70
}
`,
  'any wrapped error is a Kogen error': `func ExitCode(err error) int {
	var e *sys.Error
	var i *sys.Interrupted
	switch {
	case errors.As(err, &e):
		return e.Exit
	case errors.As(err, &i):
		return i.ExitCode()
	case errors.Unwrap(err) != nil:
		return 1
	}
	return 70
}
`,
};

// exitCodeTests lists the test functions of a Go test file whose body calls ExitCode.
function exitCodeTests(file) {
  const text = fs.readFileSync(file, 'utf8');
  const parts = text.split(/^func /m).slice(1);
  const names = [];
  for (const part of parts) {
    const m = /^(Test\w+)\(/.exec(part);
    if (m && /(^|[^.\w])ExitCode\(/.test(part)) names.push(m[1]);
  }
  return names;
}

// mutate replaces the whole ExitCode function of the scratch copy with a wrong version.
function mutate(dir, name) {
  const file = path.join(dir, 'shape', 'common.go');
  const src = fs.readFileSync(file, 'utf8');
  const start = src.indexOf(HEADER);
  assert.ok(start >= 0, 'shape/common.go has no ' + HEADER);
  const end = src.indexOf('\n}\n', start) + 3;
  const body = MUTANTS[name];
  assert.ok(body, 'unknown wrong ExitCode ' + JSON.stringify(name));
  fs.writeFileSync(file, src.slice(0, start) + body + src.slice(end));
}

Given(P + 'a scratch copy of the shape package', function () {
  this.dir = fs.mkdtempSync(path.join(os.tmpdir(), 'exit-code-wrapped-'));
  for (const f of ['go.mod', 'go.sum']) fs.copyFileSync(f, path.join(this.dir, f));
  fs.cpSync('shape', path.join(this.dir, 'shape'), {recursive: true});
});

When(P + 'the tests of shape\\/context_test.go that call ExitCode run against {string}', {timeout: 115 * 1000}, function (variant) {
  if (variant !== 'the unchanged ExitCode') mutate(this.dir, variant);
  const names = exitCodeTests(path.join(this.dir, 'shape', 'context_test.go'));
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
    'shape/context_test.go has ' + this.names.length + ' test(s) calling ExitCode (' + this.names.join(', ') + '), want at least 1');
  assert.strictEqual(this.run.status, 0,
    'go test of ' + this.names.join(', ') + ' against ' + this.variant + ' exited ' + this.run.status + ':\n' + this.run.output.slice(-1500));
  for (const n of this.names) {
    assert.ok(this.run.output.includes('--- PASS: ' + n), n + ' did not run and pass:\n' + this.run.output.slice(-1500));
  }
});

Then(P + 'at least one such test fails', function () {
  assert.ok(this.names.length > 0, 'shape/context_test.go has no test calling ExitCode');
  assert.notStrictEqual(this.run.status, 0,
    'go test of ' + this.names.join(', ') + ' still passed against the wrong ExitCode ' + JSON.stringify(this.variant));
  assert.ok(/--- FAIL: Test/.test(this.run.output),
    'go test against ' + this.variant + ' failed without a failing test (build error?):\n' + this.run.output.slice(-1500));
});
