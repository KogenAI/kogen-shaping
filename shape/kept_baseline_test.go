package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

func TestKeptStepRedOnBaselineIsRefusedDespiteRefutation(t *testing.T) {
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
	prop := func(name, quote, k, scenario string, steps []any) map[string]any {
		return map[string]any{"property": name, "request": quote, "kind": k, "observable": true, "scenarios": []any{scenario}, "decision": "", "issues": []any{}, "kept_steps": steps}
	}
	props := []any{prop("B stays unchanged", request, "kept", "B pins A", []any{3}), prop("C is added", "add C", "new", "C is new", []any{})}
	review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": props}
	verdicts := []any{}
	for i := 1; i <= 8; i++ {
		verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "refuted", "why": "the failure is new behaviour"})
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
		t.Fatal("kept assertion red on the baseline was published after a reviewer refutation")
	}
	if count != 2 {
		t.Fatalf("readiness executions = %d, want 2", count)
	}
	feedback := strings.Join(outputs, "\n")
	for _, text := range []string{"B pins A", "line 3", "red on the unchanged tree", "wrong-A", "observed-A", "mark the behaviour as changed"} {
		if !strings.Contains(feedback, text) {
			t.Fatalf("feedback lacks %q: %s", text, feedback)
		}
	}
	if strings.Contains(feedback, "C is new") {
		t.Fatalf("new behaviour was refused: %s", feedback)
	}
}

func keptTestReady(status, detail string, unexercised ...int) *verify.Ready {
	d := detail
	sc := verify.ScenarioResult{Name: "S", Line: 2, Status: status, Detail: &d,
		Steps: []verify.StepRecord{{Line: 3}, {Line: 4}, {Line: 5}}}
	for _, l := range unexercised {
		sc.UnexercisedSteps = append(sc.UnexercisedSteps, verify.StepRecord{Line: l})
	}
	return &verify.Ready{Scenarios: []verify.ScenarioResult{sc}}
}

func keptTestRefusals(r *verify.Ready, kind string, lines ...int) []string {
	props := []propertyReport{{property: "P", kind: kind, keptSteps: lines}}
	st := keptStepStates(r, map[string][]int{"S": lines})
	return keptBaselineRefusals(st, st, props)
}

func TestKeptBaselineRefusalCases(t *testing.T) {
	red := keptTestReady("failed", "AssertionError: expected=a observed=b", 5)
	if got := keptTestRefusals(red, "kept", 4); len(got) != 1 || !strings.Contains(got[0], "observed=b") || !strings.Contains(got[0], "line 4") {
		t.Fatalf("red kept step: %v", got)
	}
	// The kept step ran and passed; a later step is red: not a claim about the baseline.
	if got := keptTestRefusals(keptTestReady("failed", "AssertionError: new"), "kept", 3); len(got) != 0 {
		t.Fatalf("passing kept step refused: %v", got)
	}
	if got := keptTestRefusals(keptTestReady("passed", ""), "kept", 4); len(got) != 0 {
		t.Fatalf("green scenario refused: %v", got)
	}
	// A kept step that never ran because an earlier step failed is reported as unobserved.
	if got := keptTestRefusals(keptTestReady("failed", "AssertionError: Unknown tool", 5), "kept", 5); len(got) != 1 || !strings.Contains(got[0], "never runs on the unchanged tree") {
		t.Fatalf("blocked kept step: %v", got)
	}
	// Without named kept steps (new behaviour) nothing is refused.
	if got := keptTestRefusals(red, "new"); len(got) != 0 {
		t.Fatalf("new behaviour refused: %v", got)
	}
	// Red in one observation only (flaky) is not persistent.
	a := keptStepStates(red, map[string][]int{"S": {4}})
	if got := keptBaselineRefusals(a, keptStepStates(keptTestReady("passed", ""), map[string][]int{"S": {4}}), nil); len(got) != 0 {
		t.Fatalf("transient red refused: %v", got)
	}
}

func TestKeptBaselineBlockedStepsShareOneRefusal(t *testing.T) {
	steps := func(ls ...int) []verify.StepRecord {
		var out []verify.StepRecord
		for _, l := range ls {
			out = append(out, verify.StepRecord{Line: l})
		}
		return out
	}
	detail := "AssertionError: protected output lacks repaired text"
	ready := func(name string, all []int, blocked ...int) verify.ScenarioResult {
		d := detail
		return verify.ScenarioResult{Name: name, Line: 1, Status: "failed", Detail: &d, Steps: steps(all...), UnexercisedSteps: steps(blocked...)}
	}
	// Property P names 6,5,4,3 of S: line 3 is red, 4 and 5 are blocked in both, 6 is not seen in the second.
	kept := map[string][]int{"S": {6, 5, 4, 3}}
	props := []propertyReport{{property: "P", kind: "kept", keptSteps: []int{6, 5, 4, 3}}}
	a := keptStepStates(&verify.Ready{Scenarios: []verify.ScenarioResult{ready("S", []int{3, 4, 5, 6}, 4, 5, 6)}}, kept)
	b := keptStepStates(&verify.Ready{Scenarios: []verify.ScenarioResult{ready("S", []int{3, 4, 5}, 4, 5)}}, kept)
	got := keptBaselineRefusals(a, b, props)
	if len(got) != 2 {
		t.Fatalf("want 2 refusals: %v", got)
	}
	if !strings.Contains(got[0], "line 3 asserts unchanged behaviour but is red") || strings.Contains(got[0], "lines") {
		t.Fatalf("red step: %s", got[0])
	}
	if !strings.Contains(got[1], "scenario \"S\" lines 4 and 5 is meant to assert unchanged behaviour") || strings.Contains(got[1], "6") {
		t.Fatalf("blocked group: %s", got[1])
	}

	// Three blocked steps give one line with the failure text once; separate properties and scenarios stay apart.
	kept = map[string][]int{"S": {3, 4, 5, 6}, "T": {13, 14}}
	props = []propertyReport{
		{property: "Alpha", kind: "kept", keptSteps: []int{3, 4, 5}},
		{property: "Beta", kind: "kept", keptSteps: []int{6}},
		{property: "Gamma", kind: "kept", keptSteps: []int{13, 14}},
	}
	obs := keptStepStates(&verify.Ready{Scenarios: []verify.ScenarioResult{ready("S", []int{2, 3, 4, 5, 6}, 3, 4, 5, 6), ready("T", []int{12, 13, 14}, 13, 14)}}, kept)
	got = keptBaselineRefusals(obs, obs, props)
	if len(got) != 3 {
		t.Fatalf("want 3 refusals: %v", got)
	}
	for i, want := range []string{"- Alpha (kept)", "- Beta (kept)", "- Gamma (kept)"} {
		if !strings.Contains(got[i], want) || strings.Count(got[i], detail) != 1 {
			t.Fatalf("refusal %d: %s", i, got[i])
		}
	}
	if !strings.Contains(got[0], "lines 3, 4 and 5 is meant") || !strings.Contains(got[1], "line 6 is meant") || !strings.Contains(got[2], "scenario \"T\" lines 13 and 14 is meant") {
		t.Fatalf("lines: %v", got)
	}
}
