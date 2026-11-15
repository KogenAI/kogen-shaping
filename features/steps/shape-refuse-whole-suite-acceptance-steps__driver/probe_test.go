//go:build ignore

package shape

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

// This harness supplies step SOURCE, never executes a test command, and observes
// the real Shape submission and step-selfcheck event through the local fixture.
func TestWholeSuiteContractAcceptance(t *testing.T) {
	var input struct{ Body, Sentence string }
	if err := json.Unmarshal([]byte(os.Getenv("WS_INPUT")), &input); err != nil {
		t.Fatal(err)
	}
	feature := "Feature: Inspect step source\n  Scenario: Inspect command\n    Given is-even: a prepared fixture\n    When is-even: " + input.Sentence + "\n    Then is-even: an observed result\n"
	trace := filepath.Join(t.TempDir(), "executed")
	traceJSON, _ := json.Marshal(trace)
	code := "const {Given,When,Then}=require('@cucumber/cucumber');\n" +
		"Given('is-even: a prepared fixture', function() {});\n" +
		"When(" + string(mustWSJSON("is-even: "+input.Sentence)) + ", function() { require('node:fs').writeFileSync(" + string(traceJSON) + ", 'executed'); throw new Error('must not execute'); " + input.Body + " });\n" +
		"Then('is-even: an observed result', function() {});\n"
	candidate := map[string]any{"files": []any{
		pfFile(pfIntentPath, shpScopedIntentMD), pfFile(pfFeaturePath, feature), pfFile(pfStepsPath, code),
	}}
	f := newShpFx(t, shpSubmit("candidate", candidate))
	f.write("features/steps/calc_steps.js", "const {Given}=require('@cucumber/cucumber'); Given('a calculator', function() {});\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "valid baseline steps")
	f.setFeatureReport("report", feature, "FAILED", 1)
	f.shape("Inspect the submitted command without running it.")
	requests := f.Prov.requests()
	feedback := ""
	if len(requests) > 1 {
		feedback = shpOutputs(requests[1])["candidate"]
	}
	checks := []any{}
	for _, event := range f.events() {
		if event["event"] == "step_selfcheck" {
			checks = append(checks, event)
		}
	}
	_, executed := os.Stat(trace)
	_, accepted := os.Stat(filepath.Join(f.Aux, "feature-path"))
	policy, _ := requests[0]["instructions"].(string)
	// Only the relevant paragraph is retained; matching it still observes the
	// rendered instructions actually sent to the shaper, not source-file bytes.
	paragraphs := []string{}
	for _, paragraph := range strings.Split(policy, "\n") {
		if strings.Contains(paragraph, "per-step") || strings.Contains(paragraph, "whole test suite") {
			paragraphs = append(paragraphs, paragraph)
		}
	}
	report := map[string]any{
		"feedback": feedback, "checks": checks, "executed": executed == nil,
		"readiness": accepted == nil, "seconds": int(verify.StepTimeout.Seconds()),
		"policy": strings.Join(paragraphs, "\n"),
	}
	if err := os.WriteFile(os.Getenv("WS_OUT"), mustWSJSON(report), 0600); err != nil {
		t.Fatal(err)
	}
}

func mustWSJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
