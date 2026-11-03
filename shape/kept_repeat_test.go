package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

func TestPersistentKeptAssertionRepeatCap(t *testing.T) {
	f := newShpFx(t)
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	modules := os.Getenv("KOGEN_CUCUMBER")
	trace := filepath.Join(f.Aux, "runs")
	f.write("f.js", "exports.f = x => 'observed-A';\n")
	f.write("features/steps/calc_steps.js", "// existing\n")
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\necho run >> '%s'\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", trace, modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
	f.git("add", "-A")
	f.git("commit", "-qm", "synthetic kept fixture")
	request := "keep behaviour B and add C"
	steps := "const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');\nThen('is-even: f of A is {string}', function(want) { const actual=require('../../f').f('A'); assert.strictEqual(actual,want,'expected='+want+' observed='+actual); });\nThen('is-even: C returns ready',function(){ assert.strictEqual(require('../../f').c,'ready','expected=ready observed=undefined'); });\n"
	feature := "Feature: Pin output\n  Scenario: B pins A\n    Then is-even: f of A is \"wrong-A\"\n  Scenario: C is new\n    Then is-even: C returns ready\n"
	intent := "# Pin the current output\n\nSlug: is-even\nModel: not applicable — input/output only\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" + request + "\n\n## Decisions\n- Preserve the current output. (source: request)\n\n## Source comparison\n- The request is exercised by the feature.\n\n## Tests\n- Output: acceptance (scenario)\n\n## Feasibility\n- none\n"
	pack := func(feat string) map[string]any {
		return map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feat},
			map[string]any{"path": "features/steps/is-even__steps.js", "content": steps},
		}}
	}
	prop := func(name, quote, k, scenario string) map[string]any {
		return map[string]any{"property": name, "request": quote, "kind": k, "observable": true, "scenarios": []any{scenario}, "decision": "", "issues": []any{}}
	}
	props := []any{prop("B stays unchanged", request, "kept", "B pins A"), prop("C is added", "add C", "new", "C is new")}
	review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": props}
	verdicts := []any{}
	for i := 1; i <= 8; i++ {
		verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "confirmed", "why": "synthetic confirmation of observed mismatch"})
	}
	for i := 0; i < 4; i++ {
		f.Prov.contract = append(f.Prov.contract, shpAnswer(shpCall(fmt.Sprint("review", i), SubmitContractReviewTool, review)), shpAnswer(shpCall(fmt.Sprint("confirm", i), SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts})))
	}
	first := pack(feature)
	f.Prov.seq = []shpStep{shpSubmit("first", first), shpSubmit("second", first), shpSubmit("third", first)}
	exit, err := f.shape(request)
	published := exit == 0 && err == nil && f.exists(".kogen/local/readiness/is-even.json")
	outputs := []string{}
	for _, r := range f.Prov.requests() {
		if s, ok := shpOutputs(r)["first"]; ok {
			outputs = append(outputs, s)
		}
	}
	count := 0
	if b, e := os.ReadFile(trace); e == nil {
		count = strings.Count(string(b), "run\n")
	}
	if published {
		t.Fatal("persistent kept contradiction was published")
	}
	if count != 2 {
		t.Fatalf("readiness executions = %d, want 2", count)
	}
	feedback := strings.Join(outputs, "\n")
	for _, text := range []string{"B pins A", "wrong-A", "observed-A", "fixture"} {
		if !strings.Contains(feedback, text) {
			t.Fatalf("feedback lacks %q: %s", text, feedback)
		}
	}
	if !strings.Contains(f.Stderr.String(), "persistent kept assertion red (cap 2 executions)") {
		t.Fatalf("missing repeat evidence: %s", f.Stderr.String())
	}
}

func TestPersistentKeptOutlineRepeatCap(t *testing.T) {
	f := newShpFx(t)
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	modules := os.Getenv("KOGEN_CUCUMBER")
	trace := filepath.Join(f.Aux, "runs")
	f.write("f.js", "exports.f = x => 'observed-A';\n")
	f.write("features/steps/calc_steps.js", "// existing\n")
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\necho run >> '%s'\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", trace, modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
	f.git("add", "-A")
	f.git("commit", "-qm", "synthetic kept fixture")
	request := "keep behaviour B and add C"
	steps := "const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');\nThen('is-even: f of A is {string}', function(want) { const actual=require('../../f').f('A'); assert.strictEqual(actual,want,'expected='+want+' observed='+actual); });\nThen('is-even: C returns ready',function(){ assert.strictEqual(require('../../f').c,'ready','expected=ready observed=undefined'); });\n"
	feature := "Feature: Pin output\n  Scenario Outline: B pins <input>\n    Then is-even: f of A is \"wrong-A\"\n    Examples:\n      | input |\n      | A |\n  Scenario: C is new\n    Then is-even: C returns ready\n"
	intent := "# Pin the current output\n\nSlug: is-even\nModel: not applicable — input/output only\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" + request + "\n\n## Decisions\n- Preserve the current output. (source: request)\n\n## Source comparison\n- The request is exercised by the feature.\n\n## Tests\n- Output: acceptance (scenario)\n\n## Feasibility\n- none\n"
	pack := func(feat string) map[string]any {
		return map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feat},
			map[string]any{"path": "features/steps/is-even__steps.js", "content": steps},
		}}
	}
	prop := func(name, quote, k, scenario string) map[string]any {
		return map[string]any{"property": name, "request": quote, "kind": k, "observable": true, "scenarios": []any{scenario}, "decision": "", "issues": []any{}}
	}
	props := []any{prop("B stays unchanged", request, "kept", "B pins <input>"), prop("C is added", "add C", "new", "C is new")}
	review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": props}
	verdicts := []any{}
	for i := 1; i <= 8; i++ {
		verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "confirmed", "why": "synthetic confirmation of observed mismatch"})
	}
	for i := 0; i < 4; i++ {
		f.Prov.contract = append(f.Prov.contract, shpAnswer(shpCall(fmt.Sprint("review", i), SubmitContractReviewTool, review)), shpAnswer(shpCall(fmt.Sprint("confirm", i), SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts})))
	}
	first := pack(feature)
	f.Prov.seq = []shpStep{shpSubmit("first", first), shpSubmit("second", first), shpSubmit("third", first)}
	exit, err := f.shape(request)
	published := exit == 0 && err == nil && f.exists(".kogen/local/readiness/is-even.json")
	outputs := []string{}
	for _, r := range f.Prov.requests() {
		if s, ok := shpOutputs(r)["first"]; ok {
			outputs = append(outputs, s)
		}
	}
	count := 0
	if b, e := os.ReadFile(trace); e == nil {
		count = strings.Count(string(b), "run\n")
	}
	if published {
		t.Fatal("persistent kept contradiction was published")
	}
	if count != 2 {
		t.Fatalf("readiness executions = %d, want 2", count)
	}
	feedback := strings.Join(outputs, "\n")
	for _, text := range []string{"B pins A", "wrong-A", "observed-A", "fixture"} {
		if !strings.Contains(feedback, text) {
			t.Fatalf("feedback lacks %q: %s", text, feedback)
		}
	}
	if !strings.Contains(f.Stderr.String(), "persistent kept assertion red (cap 2 executions)") {
		t.Fatalf("missing repeat evidence: %s", f.Stderr.String())
	}
}

// Distinct example rows can share a name; a red moving between rows is not persistent.
func TestKeptRepeatDistinguishesRows(t *testing.T) {
	detail := "AssertionError: expected wrong observed right"
	first := &verify.Ready{Scenarios: []verify.ScenarioResult{{Name: "same", Line: 7, Status: "failed", Detail: &detail}}}
	second := &verify.Ready{Scenarios: []verify.ScenarioResult{{Name: "same", Line: 8, Status: "failed", Detail: &detail}}}
	a, b := keptAssertionFailures(first, map[string]bool{"same": true}), keptAssertionFailures(second, map[string]bool{"same": true})
	for key := range a {
		if _, ok := b[key]; ok {
			t.Fatal("different example rows treated as persistent")
		}
	}
}

func TestKeptAssertionFailureText(t *testing.T) {
	detail := "assertion failed: expected wrong-A observed observed-A"
	ready := &verify.Ready{Scenarios: []verify.ScenarioResult{{Name: "pin", Line: 2, Status: "failed", Detail: &detail}}}
	if got := keptAssertionFailures(ready, map[string]bool{"pin": true}); len(got) != 1 {
		t.Fatalf("assertion failure not selected: %v", got)
	}
	missing := "Error: ENOENT absent-fixture.txt"
	ready.Scenarios[0].Detail = &missing
	if got := keptAssertionFailures(ready, map[string]bool{"pin": true}); len(got) != 0 {
		t.Fatalf("missing fixture treated as assertion: %v", got)
	}
}
