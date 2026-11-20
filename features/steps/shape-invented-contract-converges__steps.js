'use strict';
// Acceptance of shape-invented-contract-converges. A Go harness is compiled into the shape tests through an overlay.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
const {Given, Then} = require('@cucumber/cucumber');

const P = 'shape-invented-contract-converges: ';
const TARGET = 'shape/shape_invented_contract_converges_acceptance_test.go';
const GO = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func siccRun(t *testing.T, literal1, literal2 string, mixed bool) map[string]any {
	files := func(lit string) map[string]any {
		m := shpFiles()
		l := m["files"].([]any)
		for _, raw := range l {
			e := raw.(map[string]any)
			if e["path"] == ccStepsPath {
				e["content"] = e["content"].(string) + "\n// " + lit + "\n"
			}
		}
		if mixed {
			m["files"] = append(l, map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": "{\"deferred\":\"the correct implementation is too large to write here\"}"})
		}
		return m
	}
	review := func(id, lit string) shpStep {
		inv := []any{}
		if lit != "" {
			inv = append(inv, map[string]any{"scenario": "Even number", "literal": lit, "why": "no source states it"})
		}
		args := map[string]any{"findings": []any{}, "environment": []any{}, "invented": inv}
		if mixed {
			args["guarantees"] = []any{map[string]any{"property": "Delivery", "request": "exactly once", "needs_interrupted": false}}
			args["failure_modes"] = []any{}
		}
		return shpAnswer(shpCall(id, SubmitContractReviewTool, args))
	}
	f := newShpFx(t, shpSubmit("c1", files(literal1)), shpSubmit("c2", files(literal2)), shpSubmit("c3", files(literal2)))
	f.Prov.contract = []shpStep{review("r1", literal1), review("r2", literal2), review("r3", literal2)}
	request := "Show whether a number is even."
	if mixed {
		request = "Deliver exactly once."
	}
	exit, err := f.shape(request)
	events := f.events()
	out := f.Stdout.String() + f.Stderr.String()
	invented, uncovered := 0, 0
	for _, e := range ccRefused(f) {
		p := fmt.Sprint(e["problems"])
		if strings.Contains(p, "invented_contract") {
			invented++
		}
		if strings.Contains(p, "uncovered_failure_mode") {
			uncovered++
		}
	}
	return map[string]any{"exit": exit, "error": err != nil, "dropped": shpNamed(events, "invented_literal_dropped"),
		"heading": strings.Contains(out, "Literals chosen by the Shaper"), "output": out, "invented_refusals": invented, "uncovered_refusals": uncovered}
}

func siccDecision(statement, verdict, intentValue string) map[string]any {
	const request = "Deliver exactly once."
	it := decisionItem{statement: statement, quotes: []string{request}, verdict: verdict, intentValue: intentValue, requestValue: "Deliver exactly once.", difference: "d", why: "w"}
	_, refusal, _, _ := decisionResolveNotes([]decisionItem{it}, request, "## Decisions\n"+statement+"\n", nil)
	return map[string]any{"refusal": strings.Join(refusal, "\n"), "count": len(refusal)}
}

func TestSiccAcceptance(t *testing.T) {
	obs := map[string]any{}
	obs["rewritten"] = siccRun(t, "assert_operator counts.last, :<=, counts.first + 2", "", false)
	obs["repeated"] = siccRun(t, "bound = 40", "bound = 40", false)
	obs["fresh"] = siccRun(t, "bound = 40", "account_exports_url", false)
	obs["mixed"] = siccRun(t, "bound = 40", "bound = 40", true)
	obs["weak"] = siccDecision("- at-most-once is accepted for a crash mid-send (assumption: crash handling) (source: request)", "contradicts", "at-most-once")
	obs["full"] = siccDecision("- exactly once, including across a crash and a restart; the receiver limit is a separate open limit (assumption: receiver dedupe) (source: request)", "follows", "")
	b, _ := json.Marshal(obs)
	if err := os.WriteFile(os.Getenv("SICC_OUT"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;

let cache = null;
function observe() {
  if (cache) return cache;
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sicc-'));
  const src = path.join(dir, 'h_test.go');
  fs.writeFileSync(src, GO);
  const overlay = path.join(dir, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({Replace: {[path.resolve(TARGET)]: src}}));
  const out = path.join(dir, 'obs.json');
  const r = spawnSync('go', ['test', '-count=1', '-vet=off', '-overlay', overlay, '-run', '^TestSiccAcceptance$', './shape'], {env: {...process.env, SICC_OUT: out}, encoding: 'utf8', timeout: 110000, maxBuffer: 1 << 26});
  assert.equal(r.status, 0, 'go test TestSiccAcceptance exited ' + r.status + ':\n' + r.stdout + r.stderr);
  cache = JSON.parse(fs.readFileSync(out, 'utf8'));
  return cache;
}

Given(P + 'the harness has run', function () { observe(); });

Then(P + 'case {string} had first submission refused with {string}, {int} dropped events and no card heading', function (name, kind, n) {
  const c = observe()[name];
  assert.equal(kind, 'invented_contract');
  assert.equal(c.invented_refusals, 1, name + ': expected one invented_contract refusal event, got ' + c.invented_refusals);
  assert.equal((c.dropped || []).length, n, name + ' dropped events');
  assert.equal(c.heading, false, name + ' card heading');
});

Then(P + 'case {string} dropped {int} literals including {string} in the event and in the card', function (name, n, lit) {
  const c = observe()[name];
  assert.equal((c.dropped || []).length, n, name + ' dropped events: ' + JSON.stringify(c.dropped));
  assert.ok(JSON.stringify(c.dropped).includes(lit), name + ' event lacks ' + lit);
  assert.equal(c.heading, true, name + ' output lacks the heading');
  assert.ok(c.output.includes(lit), name + ' card lacks ' + lit);
});

Then(P + 'case {string} was refused once, dropped {int} literal {string} and used {int} submissions', function (name, n, lit, subs) {
  const c = observe()[name];
  assert.equal(c.invented_refusals, 1, name + ' invented_contract refusals');
  assert.equal((c.dropped || []).length, n, name + ' dropped events: ' + JSON.stringify(c.dropped));
  assert.ok(JSON.stringify(c.dropped).includes(lit), name + ' event lacks ' + lit);
  assert.equal(c.dropped[0].submission, subs, name + ' submission number');
});

Then(P + 'case {string} dropped {string} yet still refused {string} on the second submission', function (name, lit, kind) {
  const c = observe()[name];
  assert.equal(kind, 'uncovered_failure_mode');
  assert.ok(JSON.stringify(c.dropped || []).includes(lit), name + ' did not drop ' + lit + ': ' + JSON.stringify(c.dropped));
  assert.ok(c.uncovered_refusals >= 1, name + ' did not refuse ' + kind + ' beside the dropped literal');
  assert.equal(c.invented_refusals, 1, name + ' refused invented_contract more than once');
});

Then(P + 'shaper.md contains {string} and {string} and {string}', function (a, b, c) {
  const text = fs.readFileSync('shape/prompts/shaper.md', 'utf8');
  for (const s of [a, b, c]) assert.ok(text.includes(s), 'shape/prompts/shaper.md lacks "' + s + '"');
});

Then(P + 'the weakened Decision is refused with a fix line quoting {string} and the full Decision is accepted', function (quote) {
  const o = observe();
  assert.ok(o.weak.count > 0, 'the at-most-once Decision was not refused');
  assert.ok(o.weak.refusal.includes(quote), 'refusal does not quote "' + quote + '": ' + o.weak.refusal);
  assert.ok(o.weak.refusal.toLowerCase().includes('restate'), 'refusal does not tell the Shaper to restate: ' + o.weak.refusal);
  assert.equal(o.full.count, 0, 'the full-guarantee Decision with a separate open limit was refused: ' + o.full.refusal);
});
