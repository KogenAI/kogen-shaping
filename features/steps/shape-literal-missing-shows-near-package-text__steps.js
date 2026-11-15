'use strict';
// Acceptance of shape-literal-missing-shows-near-package-text. The Given step runs a Go test
// (injected into the shape package through a go build overlay, so no test of the Build is needed)
// that calls literalOutcome once per case and writes the feedback lines and event statuses as JSON
// to SLMSNPT_OUT. The Then steps compare one case with the feature's text.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-literal-missing-shows-near-package-text: ';
const TEST = 'TestSlmsnptObservation';

const HARNESS = `package shape

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

func TestSlmsnptObservation(t *testing.T) {
	request := "Register /demo_one and /demo-two."
	sources := map[string]string{"request": request}
	files := []packageFile{{path: "intent.md", content: "registers /demo_one here and /demo-one there"}}
	red := verify.NewRedactor([]string{"MY_TOKEN=tok-9f8e7d"}, "")
	mk := func(literal, packageValue, why string) literalItem {
		return literalItem{literal: literal, source: "request", quote: request, status: "required", packageValue: packageValue, why: why}
	}
	cases := map[string]literalItem{
		"missing with near value":        mk("/demo-two", "/demo_two", "w"),
		"missing without near value":     mk("/demo-two", "", "w"),
		"changed":                        mk("/demo-two", "/demo-one", "w"),
		"missing with secret near value": mk("/demo-two", "near tok-9f8e7d", "w"),
	}
	out := map[string]any{}
	for name, it := range cases {
		event, lines := literalOutcome([]literalItem{it}, sources, files, red)
		obs := map[string]any{"status": event["items"].([]any)[0].(map[string]any)["status"], "lines": lines}
		out[name] = obs
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SLMSNPT_OUT"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;

const owned = [];
process.on('exit', () => { for (const d of owned) fs.rmSync(d, {recursive: true, force: true}); });

let observation = null;

function run() {
  if (observation !== null) return observation;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'slmsnpt-'));
  owned.push(dir);
  const source = path.join(dir, 'harness_test.go');
  fs.writeFileSync(source, HARNESS);
  const target = path.join(process.cwd(), 'shape', 'slmsnpt_observation_test.go');
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[target]: source}}));
  const out = path.join(dir, 'out.json');
  const env = {};
  for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('KOGEN_ACCEPTANCE_')) env[k] = v;
  env.SLMSNPT_OUT = out;
  const args = ['test', '-count=1', '-overlay', overlay, '-run', '^' + TEST + '$', '-v', './shape'];
  const r = spawnSync('go', args, {cwd: process.cwd(), env, encoding: 'utf8', timeout: 110000});
  const shown = `go ${args.join(' ')} exited ${r.status}${r.error ? ' ' + r.error : ''}:\n${r.stdout}${r.stderr}`;
  assert.equal(r.status, 0, shown);
  assert.ok((r.stdout || '').includes('--- PASS: ' + TEST), `${TEST} did not pass: ${shown}`);
  assert.ok(fs.existsSync(out), `${TEST} wrote no observation: ${shown}`);
  observation = JSON.parse(fs.readFileSync(out, 'utf8'));
  return observation;
}

Given(P + 'the literal outcome of each case has been observed', function () {
  run();
});

function line(name) {
  const obs = run()[name];
  assert.ok(obs, `no case ${name}`);
  assert.ok(Array.isArray(obs.lines) && obs.lines.length === 2, `case ${name}: expected a headline and one problem line, got ${JSON.stringify(obs.lines)}`);
  return obs;
}

Then(P + 'the problem line of case {string} is:', function (name, doc) {
  assert.equal(line(name).lines[1], doc);
});

Then(P + 'the headline of case {string} is:', function (name, doc) {
  assert.equal(line(name).lines[0], doc);
});

Then(P + 'the status of case {string} is {string}', function (name, status) {
  assert.equal(line(name).status, status);
});
