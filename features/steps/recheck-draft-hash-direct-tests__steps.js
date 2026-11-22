'use strict';
// Steps of RECHECK-DRAFT-HASH-DIRECT-TESTS. The tests of shape/recheck_test.go are found by reading the
// file, run with the Go tool against the real shape/recheck.go, and then against scratch copies of it
// (Go's -overlay, so the checkout is never written) whose RecheckDraftHash is broken in one way.
const {When, Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const cp = require('node:child_process');

const ROOT = process.cwd();
const TEST_FILE = 'shape/recheck_test.go';
const SOURCE = 'shape/recheck.go';
const HASH_RETURN = '\treturn project.Hash(root, baseline), nil\n';
const LOAD_ERR = 'cfg, err := project.LoadConfig(root)\n\tif err != nil {\n\t\treturn "", err\n\t}';

const MUTANTS = {
  empty: [HASH_RETURN, '\t_ = baseline\n\treturn "", nil\n'],
  constant: [HASH_RETURN, '\t_ = baseline\n\treturn "constant-hash", nil\n'],
  unstable: [HASH_RETURN, '\treturn fmt.Sprintf("%s%d", project.Hash(root, baseline), time.Now().UnixNano()), nil\n'],
  'ignores acceptance.feature': [HASH_RETURN,
    '\tvar kept []string\n\tfor _, p := range baseline {\n\t\tif !strings.HasSuffix(p, "acceptance.feature") {\n\t\t\tkept = append(kept, p)\n\t\t}\n\t}\n\treturn project.Hash(root, kept), nil\n'],
  'swallows the configuration error': [LOAD_ERR, 'cfg, err := project.LoadConfig(root)\n\tif err != nil {\n\t\treturn "", nil\n\t}'],
};

function testNames() {
  const file = path.join(ROOT, TEST_FILE);
  assert.ok(fs.existsSync(file), `${TEST_FILE} does not exist`);
  const text = fs.readFileSync(file, 'utf8');
  const names = [...text.matchAll(/^func (Test\w+)\(t \*testing\.T\)/gm)].map((m) => m[1]);
  assert.ok(names.length >= 1, `${TEST_FILE} declares no top-level Test function`);
  assert.ok(text.includes('RecheckDraftHash('), `${TEST_FILE} never calls RecheckDraftHash`);
  return names;
}

// run runs the tests of TEST_FILE; with a mutant, shape/recheck.go is replaced by the broken copy.
function run(names, mutant) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'recheck-hash-'));
  const args = ['test', './shape', '-count=1', '-v', '-run', `^(${names.join('|')})$`];
  if (mutant) {
    const src = fs.readFileSync(path.join(ROOT, SOURCE), 'utf8');
    const [from, to] = MUTANTS[mutant];
    assert.ok(src.includes(from), `${SOURCE} no longer contains the text the ${mutant} copy replaces`);
    fs.writeFileSync(path.join(dir, 'recheck.go'), src.replace(from, to));
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.join(ROOT, SOURCE)]: path.join(dir, 'recheck.go')}}));
    args.splice(1, 0, '-overlay', overlay);
  }
  return new Promise((resolve) => {
    const child = cp.spawn('go', args, {cwd: ROOT, env: process.env});
    let out = '';
    child.stdout.on('data', (d) => (out += d));
    child.stderr.on('data', (d) => (out += d));
    const timer = setTimeout(() => child.kill('SIGKILL'), 100000);
    child.on('close', (status) => {
      clearTimeout(timer);
      fs.rmSync(dir, {recursive: true, force: true});
      resolve({status, out, command: `go ${args.join(' ')}`});
    });
  });
}

let names = null;
let real = null;
let mutants = null;

When('recheck-draft-hash-direct-tests: I run the tests declared in the recheck test file', async function () {
  names = testNames();
  real = await run(names, null);
});

Then('recheck-draft-hash-direct-tests: the go command exits 0', function () {
  assert.equal(real.status, 0, `${real.command} exited ${real.status}:\n${real.out}`);
});

Then('recheck-draft-hash-direct-tests: every declared test is reported as passed', function () {
  for (const name of names) {
    assert.ok(real.out.includes(`--- PASS: ${name} `), `${name} was not reported as passed by ${real.command}:\n${real.out}`);
  }
});

Then('recheck-draft-hash-direct-tests: no Go file other than the recheck test file is changed', function () {
  const status = cp.spawnSync('git', ['status', '--porcelain', '-uall'], {cwd: ROOT, encoding: 'utf8'}).stdout;
  const changed = status.split('\n').filter(Boolean).map((l) => l.slice(3)).filter((p) => p.endsWith('.go') && p !== TEST_FILE);
  assert.deepEqual(changed, [], `Go files other than ${TEST_FILE} are changed`);
});

When('recheck-draft-hash-direct-tests: I run those tests against copies of RecheckDraftHash broken in five ways', async function () {
  names = testNames();
  const keys = Object.keys(MUTANTS);
  const results = await Promise.all(keys.map((k) => run(names, k)));
  mutants = Object.fromEntries(keys.map((k, i) => [k, results[i]]));
});

function rejects(kind) {
  const r = mutants[kind];
  assert.ok(r, `no run for the ${kind} copy`);
  assert.notEqual(r.status, 0, `the tests of ${TEST_FILE} still pass when the hash ${kind}; ${r.command}:\n${r.out}`);
  assert.ok(names.some((n) => r.out.includes(`--- FAIL: ${n} `)),
    `the ${kind} copy failed without a failing declared test (${names.join(', ')}); ${r.command}:\n${r.out}`);
}

Then('recheck-draft-hash-direct-tests: the recheck tests fail when the hash is empty', () => rejects('empty'));
Then('recheck-draft-hash-direct-tests: the recheck tests fail when the hash is the same for every draft', () => rejects('constant'));
Then('recheck-draft-hash-direct-tests: the recheck tests fail when the hash differs between two calls', () => rejects('unstable'));
Then('recheck-draft-hash-direct-tests: the recheck tests fail when the hash ignores acceptance.feature', () => rejects('ignores acceptance.feature'));
Then('recheck-draft-hash-direct-tests: the recheck tests fail when the project configuration error is swallowed', () => rejects('swallows the configuration error'));
