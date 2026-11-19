'use strict';
// Acceptance of shape-time-excludes-timed-out-static-checks. The Given step compiles the shape
// package's tests with harness tests (through a Go overlay), runs them once and reads the JSON each
// wrote. The Then steps compare one case. Each case is its own top-level test because the shape
// fixtures run their test in parallel.
const {Given, Then} = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

const TESTS = ['TestSteTimeoutAcceptance', 'TestSteFailedAcceptance', 'TestSteBoundedAcceptance', 'TestSteRefusalAcceptance'];
const CASES = ['timeout', 'failed', 'bounded', 'refusal'];
const TARGET = 'shape/shape_time_excludes_timed_out_static_checks_acceptance_test.go';

const GO = `package shape

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

type steSpec struct {
	wall     time.Duration
	status   int
	timedOut bool
}

func steWrite(t *testing.T, name string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("STESC_OUT"), name+".json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func steFns(clock *budgetClock, specs map[string]steSpec) map[string]stmCheckFn {
	fns := map[string]stmCheckFn{}
	for n, sp := range specs {
		sp := sp
		fns[n] = func(int) sys.Result {
			clock.Advance(sp.wall)
			stmWork()
			return sys.Result{Status: sp.status, TimedOut: sp.timedOut}
		}
	}
	return fns
}

func steCase(t *testing.T, name string, names []string, specs map[string]steSpec) {
	t.Helper()
	clock := newBudgetClock()
	s := stmNew(t, names, steFns(clock, specs), "c1")
	s.Now = clock.Now
	s.shapeOnce(t)
	accounts := shpNamed(s.events(), "shape_time")
	if len(accounts) != 1 {
		t.Fatalf("shape_time events: %d", len(accounts))
	}
	steWrite(t, name, accounts[0])
}

func TestSteTimeoutAcceptance(t *testing.T) {
	steCase(t, "timeout", []string{"slow"}, map[string]steSpec{"slow": {120 * time.Second, 137, true}})
}

func TestSteFailedAcceptance(t *testing.T) {
	steCase(t, "failed", []string{"slow"}, map[string]steSpec{"slow": {120 * time.Second, 1, false}})
}

func TestSteBoundedAcceptance(t *testing.T) {
	steCase(t, "bounded", []string{"a", "b", "ok"}, map[string]steSpec{
		"a":  {150 * time.Second, 137, true},
		"b":  {120 * time.Second, 137, true},
		"ok": {30 * time.Second, 0, false},
	})
}

func TestSteRefusalAcceptance(t *testing.T) {
	clock := newBudgetClock()
	s := stmNew(t, []string{"slow", "ok"}, steFns(clock, map[string]steSpec{
		"slow": {120 * time.Second, 137, true},
		"ok":   {14*time.Minute + 30*time.Second, 0, false},
	}), "c1", "c2")
	s.Now = clock.Now
	_, err := s.shape("Add calc.is_even(n).")
	e, ok := sys.AsError(err)
	if !ok {
		t.Fatalf("expected the budget refusal, got %v", err)
	}
	steWrite(t, "refusal", map[string]any{"detail": e.Detail, "lines": e.Lines})
}
`;

function env(extra) {
  const out = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) out[k] = v;
  return {...out, ...extra};
}

let observation = null;

function run() {
  if (observation !== null) return observation;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stesc-'));
  try {
    const source = path.join(dir, 'harness_test.go');
    fs.writeFileSync(source, GO);
    const overlay = path.join(dir, 'overlay.json');
    fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: source}}));
    const bin = path.join(dir, 'shape.test');
    const b = spawnSync('go', ['test', '-c', '-vet=off', '-overlay', overlay, '-o', bin, './shape'], {env: env({}), encoding: 'utf8', timeout: 100000, maxBuffer: 32 * 1024 * 1024});
    assert.equal(b.status, 0, `building the shape test binary with the harness exited ${b.status}:\n${b.stdout}${b.stderr}`);
    const out = path.join(dir, 'out');
    fs.mkdirSync(out);
    const r = spawnSync(bin, ['-test.run', '^(' + TESTS.join('|') + ')$', '-test.count=1', '-test.v', '-test.timeout=100s'], {cwd: path.resolve('shape'), env: env({STESC_OUT: out}), encoding: 'utf8', timeout: 110000, maxBuffer: 32 * 1024 * 1024});
    const shown = (r.stdout || '') + (r.stderr || '');
    assert.equal(r.status, 0, `${TESTS.join(', ')} exited ${r.status}:\n${shown}`);
    observation = {};
    for (const name of CASES) {
      const f = path.join(out, name + '.json');
      assert.ok(fs.existsSync(f), `the harness wrote no observation for case ${name}:\n${shown}`);
      observation[name] = JSON.parse(fs.readFileSync(f, 'utf8'));
    }
  } finally {
    fs.rmSync(dir, {recursive: true, force: true});
  }
  return observation;
}

Given('shape-time-excludes-timed-out-static-checks: the shaping harness has run', function () {
  run();
});

Then('shape-time-excludes-timed-out-static-checks: the shape_time of case {string} includes:', function (name, doc) {
  const o = run();
  assert.ok(Object.hasOwn(o, name), `no case "${name}" in ${JSON.stringify(o)}`);
  const want = JSON.parse(doc);
  const got = o[name];
  for (const [k, v] of Object.entries(want)) {
    assert.ok(Object.hasOwn(got, k), `case "${name}": shape_time has no ${k}: ${JSON.stringify(got)}`);
    assert.deepEqual(got[k], v, `case "${name}": ${k} is ${JSON.stringify(got[k])}, want ${JSON.stringify(v)}: ${JSON.stringify(got)}`);
  }
});

Then('shape-time-excludes-timed-out-static-checks: the budget refusal reports 29 min elapsed and names static_uncharged_ms 120000 on its figure and attempt lines', function () {
  const r = run().refusal;
  const shown = JSON.stringify(r, null, 1);
  const suffix = ', static_uncharged_ms=120000';
  assert.ok(r.detail.startsWith('the shaper used its 15-minute shaping budget (29 min elapsed,'), `detail: ${shown}`);
  const figure = r.lines.filter(l => l.startsWith('shaping budget: 900 s'));
  assert.equal(figure.length, 1, `figure line: ${shown}`);
  assert.ok(figure[0].endsWith(suffix), `the figure line does not end with ${suffix}: ${shown}`);
  const attempts = r.lines.filter(l => l.startsWith('attempt '));
  assert.ok(attempts.length >= 1, `no attempt line: ${shown}`);
  for (const l of attempts) assert.ok(l.endsWith(suffix), `attempt line does not end with ${suffix}: ${l}\n${shown}`);
});
