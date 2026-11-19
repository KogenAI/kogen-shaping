'use strict';
// Acceptance of shape-static-gate-skips-full-validators. The steps compile the shape package's tests with
// a small harness test file (through a Go overlay) holding two tests: TestSgfvSelection asks
// isTestSuiteCheck about command lines, TestSgfvGate runs the static gate with a recording command seam.
// Each test writes one JSON observation, which the Then steps compare with the feature. The binary is
// built once and reused.
const {When, Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const P = 'shape-static-gate-skips-full-validators: ';
const TARGET = 'shape/shape_static_gate_skips_full_validators_acceptance_test.go';

const GO = `package shape

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

func TestSgfvSelection(t *testing.T) {
	var cmds []string
	if err := json.Unmarshal([]byte(os.Getenv("SGFV_COMMANDS")), &cmds); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range cmds {
		got[c] = isTestSuiteCheck(c)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SGFV_OUT"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSgfvGate(t *testing.T) {
	const step = "features/steps/fixture_steps.js"
	const validator = "python3 reproduce/validate_repo.py"
	privacy := \`if [ -f \` + step + \` ]; then echo "\` + step + \`:3:5: bad"; fi; exit 1\`
	f := staticFx(t, "    - name: validate-repo\\n      run: '"+validator+"'\\n    - name: privacy\\n      run: '"+privacy+"'\\n")
	f.write(step, "// BAD\\n")
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	commands := []string{}
	real := staticRunCommand
	staticRunCommand = func(ctx context.Context, s *sys.Session, command, dir string, opts sys.Options) (sys.Result, error) {
		mu.Lock()
		commands = append(commands, strings.TrimPrefix(command, "set -x\\n"))
		mu.Unlock()
		if strings.Contains(command, "validate_repo.py") {
			return sys.Result{}, nil
		}
		return real(ctx, s, command, dir, opts)
	}
	t.Cleanup(func() { staticRunCommand = real })
	s, _, _ := f.session()
	progress := []string{}
	head := f.Repo.Head()
	tree, err := f.Repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(l string) { progress = append(progress, l) })
	run, err := g.run(context.Background(), head, tree, "package")
	if err != nil {
		t.Fatal(err)
	}
	type chk struct {
		Status string   \`json:"status"\`
		Paths  []string \`json:"paths"\`
	}
	checks := map[string]chk{}
	for _, c := range run.Checks {
		checks[c.Name] = chk{c.Status, c.Paths}
	}
	b, err := json.Marshal(map[string]any{"commands": commands, "checks": checks, "outcome": run.Outcome, "progress": progress, "privacy": privacy, "step": step})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SGFV_OUT"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}
`;

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let built = null;

function binary() {
  if (built !== null) return built;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sgfv-'));
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, GO);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
  const bin = path.join(dir, 'shape.test');
  const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env({}), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
  assert.equal(b.status, 0, `building the shape test binary with the harness exited ${b.status}:\n${b.stdout}${b.stderr}`);
  built = {dir, bin};
  process.on('exit', () => fs.rmSync(dir, {recursive: true, force: true}));
  return built;
}

function runTest(name, extra) {
  const {dir, bin} = binary();
  const out = path.join(dir, name + '.json');
  fs.rmSync(out, {force: true});
  const r = spawnSync(bin, ['-test.run', '^' + name + '$', '-test.count=1', '-test.v', '-test.timeout=60s'], {cwd: path.resolve('shape'), env: env({...extra, SGFV_OUT: out}), encoding: 'utf8', timeout: 70000, maxBuffer: 32 * 1024 * 1024});
  const shown = (r.stdout || '') + (r.stderr || '');
  assert.equal(r.status, 0, `${name} exited ${r.status}:\n${shown}`);
  assert.ok(fs.existsSync(out), `${name} wrote no observation:\n${shown}`);
  return JSON.parse(fs.readFileSync(out, 'utf8'));
}

function answers(doc) {
  const cmds = doc.split('\n').map((l) => l.trim()).filter((l) => l !== '');
  assert.ok(cmds.length > 0, 'the step lists no command');
  return {cmds, got: runTest('TestSgfvSelection', {SGFV_COMMANDS: JSON.stringify(cmds)})};
}

Then(P + 'each of these commands is a test suite check:', function (doc) {
  const {cmds, got} = answers(doc);
  const bad = cmds.filter((c) => got[c] !== true);
  assert.deepEqual(bad, [], `not recognised as a test suite check (the gate would run them): ${JSON.stringify(bad)}`);
});

Then(P + 'none of these commands is a test suite check:', function (doc) {
  const {cmds, got} = answers(doc);
  const bad = cmds.filter((c) => got[c] !== false);
  assert.deepEqual(bad, [], `wrongly recognised as a test suite check (the gate would skip them): ${JSON.stringify(bad)}`);
});

When(P + 'the static gate runs the checks {string} and {string} with a recording command seam', function (a, b) {
  assert.deepEqual([a, b], ['validate-repo', 'privacy'], 'the harness configures exactly the checks validate-repo and privacy');
  this.sgfv = runTest('TestSgfvGate', {});
});

Then(P + 'the only command given to the command seam is the {string} check command', function (name) {
  assert.equal(name, 'privacy');
  const o = this.sgfv;
  const distinct = [...new Set(o.commands)];
  assert.deepEqual(distinct, [o.privacy], `commands given to the command seam: ${JSON.stringify(o.commands)}`);
});

Then(P + 'the check {string} has the status {string}', function (name, status) {
  const c = this.sgfv.checks[name];
  assert.ok(c, `no check ${name} in ${JSON.stringify(this.sgfv.checks)}`);
  assert.equal(c.status, status, `check ${name}: ${JSON.stringify(c)}`);
});

Then(P + 'the check {string} names the shaped file', function (name) {
  const c = this.sgfv.checks[name];
  assert.ok(c, `no check ${name} in ${JSON.stringify(this.sgfv.checks)}`);
  assert.ok(c.paths.some((p) => p.startsWith(this.sgfv.step)), `check ${name} paths ${JSON.stringify(c.paths)} do not name ${this.sgfv.step}`);
});

Then(P + 'the progress shows the line {string}', function (line) {
  assert.ok(this.sgfv.progress.includes(line), `no progress line ${JSON.stringify(line)} in ${JSON.stringify(this.sgfv.progress)}`);
});

Then(P + 'the gate outcome is {string}', function (outcome) {
  assert.equal(this.sgfv.outcome, outcome, `outcome ${this.sgfv.outcome}`);
});
