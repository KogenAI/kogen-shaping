'use strict';
// The Go source of the acceptance harness of shape-controls-scenario-keyword, which
// shape-controls-scenario-keyword__steps.js compiles into the shape package's tests through a Go
// overlay as shape/shape_controls_scenario_keyword_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

// Each case parses a controls.json with one wrong implementation whose scenario field is given,
// against a fixture feature, and writes the problem lines, the stored scenario name and the witness
// indices as JSON to SCSK_OUT for the acceptance steps to judge. The harness judges nothing itself.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

const scskFeature = "Feature: Calc\n  Scenario: Add keeps working\n    Then k: add of 2 and 3 is 5\n  Scenario Outline: Mul <a> times <b>\n    Then k: mul of <a> and <b> is <c>\n    Examples:\n      | a | b | c |\n      | 2 | 3 | 6 |\n      | 3 | 3 | 9 |\n  Scenario: Scenario: odd\n    Then k: odd\n  Scenario: odd\n    Then k: odd\n"

func TestShapeControlsScenarioKeywordAcceptance(t *testing.T) {
	out := os.Getenv("SCSK_OUT")
	if out == "" {
		t.Fatal("SCSK_OUT must be set")
	}
	scenarios, err := project.ParseGherkin(scskFeature)
	if err != nil {
		t.Fatal(err)
	}
	feature := strings.Split(scskFeature, "\n")
	observe := func(field, step string) map[string]any {
		wrong := map[string]any{"name": "n1", "scenario": field, "step": step, "files": map[string]any{"calc.js": "x"}}
		raw, err := json.Marshal(map[string]any{"correct": map[string]any{"files": map[string]any{"calc.js": "y"}}, "wrong": []any{wrong}})
		if err != nil {
			t.Fatal(err)
		}
		cf, problems := parseControls(raw, scenarios, feature, controlsRules{steps: "features/steps"})
		if problems == nil {
			problems = []string{}
		}
		witness := []int{}
		scenario := ""
		if cf != nil && len(cf.wrong) == 1 {
			scenario = cf.wrong[0].scenario
			if w := controlsWitness(scenarios, cf.wrong[0], feature); w != nil {
				witness = w
			}
		}
		return map[string]any{"problems": problems, "scenario": scenario, "witness": witness}
	}
	add := "Then k: add of 2 and 3 is 5"
	mul := "Then k: mul of <a> and <b> is <c>"
	odd := "Then k: odd"
	obs := map[string]map[string]any{
		"plain scenario":                      observe("Add keeps working", add),
		"keyword scenario":                    observe("Scenario: Add keeps working", add),
		"plain outline":                       observe("Mul <a> times <b>", mul),
		"keyword outline":                     observe("Scenario Outline: Mul <a> times <b>", mul),
		"keyword template":                    observe("Scenario Template: Mul <a> times <b>", mul),
		"unknown plain":                       observe("Missing", add),
		"unknown keyword":                     observe("Scenario: Missing", add),
		"odd with keyword text":               observe("Scenario: odd", odd),
		"odd plain":                           observe("odd", odd),
		"keyword twice":                       observe("Scenario: Scenario: Add keeps working", add),
		"keyword twice before keyword-named":  observe("Scenario: Scenario: odd", odd),
	}
	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
