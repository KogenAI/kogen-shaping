'use strict';
// Acceptance of shape-static-gate-skips-rails-suite. The Then steps compile the shape package's tests
// with a small harness test (through a Go overlay), run it with the listed commands and compare what
// isTestSuiteCheck answers for each line. The binary is built once and reused by every step.
const {Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TARGET = 'shape/shape_static_gate_skips_rails_suite_acceptance_test.go';

const GO = `package shape

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSgsrsAcceptance(t *testing.T) {
	var cmds []string
	if err := json.Unmarshal([]byte(os.Getenv("SGSRS_COMMANDS")), &cmds); err != nil {
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
	if err := os.WriteFile(os.Getenv("SGSRS_OUT"), b, 0o600); err != nil {
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
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sgsrs-'));
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

function answers(doc) {
  const cmds = doc.split('\n').map((l) => l.trim()).filter((l) => l !== '');
  assert.ok(cmds.length > 0, 'the step lists no command');
  const {dir, bin} = binary();
  const out = path.join(dir, 'out.json');
  fs.rmSync(out, {force: true});
  const r = spawnSync(bin, ['-test.run', '^TestSgsrsAcceptance$', '-test.count=1', '-test.v', '-test.timeout=60s'], {cwd: path.resolve('shape'), env: env({SGSRS_COMMANDS: JSON.stringify(cmds), SGSRS_OUT: out}), encoding: 'utf8', timeout: 70000, maxBuffer: 32 * 1024 * 1024});
  const shown = (r.stdout || '') + (r.stderr || '');
  assert.equal(r.status, 0, `TestSgsrsAcceptance exited ${r.status}:\n${shown}`);
  assert.ok(fs.existsSync(out), `the harness wrote no answers:\n${shown}`);
  return {cmds, got: JSON.parse(fs.readFileSync(out, 'utf8'))};
}

Then('shape-static-gate-skips-rails-suite: each of these commands is a test suite check:', function (doc) {
  const {cmds, got} = answers(doc);
  const bad = cmds.filter((c) => got[c] !== true);
  assert.deepEqual(bad, [], `not recognised as a test suite check (the gate would run them): ${JSON.stringify(bad)}`);
});

Then('shape-static-gate-skips-rails-suite: none of these commands is a test suite check:', function (doc) {
  const {cmds, got} = answers(doc);
  const bad = cmds.filter((c) => got[c] !== false);
  assert.deepEqual(bad, [], `wrongly recognised as a test suite check (the gate would skip them): ${JSON.stringify(bad)}`);
});
