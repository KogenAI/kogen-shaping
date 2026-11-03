'use strict';
// The Go source of the k-shape-step-binding-proof acceptance harness, which
// k-shape-step-binding-proof__steps.js compiles into the shape package's tests through a Go overlay
// as shape/k_shape_step_binding_proof_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The acceptance harness of k-shape-step-binding-proof: one synthetic shaping journey per
// KSBP_CASE, with a scripted shaper that submits the binding-proof package once, the real Shape
// entry point and the real Node and Cucumber of the step self-check. What a user and the run log
// see of the journey is written as JSON to KSBP_OUT for the acceptance steps to judge.

const (
	ksbpSlug     = "binding-proof"
	ksbpStepFile = "features/steps/binding-proof__steps.js"
	ksbpFeature  = "Feature: Bind every line\n" +
		"  Scenario: A numbered result\n" +
		"    Given binding-proof: input 1\n" +
		"    When binding-proof: I retain it\n" +
		"    Then binding-proof: code 12\n" +
		"  Scenario Outline: Quoted literal text\n" +
		"    Given binding-proof: input 1\n" +
		"    When binding-proof: I retain it\n" +
		"    Then binding-proof: text \"<value>\"\n" +
		"    Examples:\n" +
		"      | value      |\n" +
		"      | v2.4+[x]   |\n" +
		"      | hello.world |\n"
	// neverRun keeps the request's sentinel (it throws) and also records that it ran in
	// KSBP_SENTINEL_TRACE, so a body or hook executed anywhere in the journey is visible.
	ksbpHeader = "const { Given, When, Then, Before, After } = require('@cucumber/cucumber');\n" +
		"function neverRun() { require('node:fs').appendFileSync(process.env.KSBP_SENTINEL_TRACE, 'executed\\n'); throw new Error('binding check executed a body or hook'); }\n" +
		"Before(neverRun);\n" +
		"After(neverRun);\n" +
		"Given('binding-proof: input {int}', neverRun);\n" +
		"When('binding-proof: I retain it', neverRun);\n"
	ksbpEscaped = "Then(/^binding-proof: code \\\\d+$/, neverRun);\n"
	ksbpCode    = "Then(/^binding-proof: code \\d+$/, neverRun);\n"
	ksbpText    = "Then('binding-proof: text {string}', neverRun);\n"
	ksbpDup     = "Then('binding-proof: code {int}', neverRun);\n"
	ksbpUnused  = "Then('binding-proof: unused', neverRun);\n"
	ksbpThrow   = "throw new Error('cannot load definitions');\n"
	ksbpToken   = "Then(/^binding-proof: token [a-f0-9]{8}$/, neverRun);\n"
)

// ksbpCase is the feature and step file a case submits.
func ksbpCase(name string) (feature, steps string) {
	feature = ksbpFeature
	switch name {
	case "escaped":
		steps = ksbpHeader + ksbpEscaped + ksbpText
	case "repaired":
		steps = ksbpHeader + ksbpCode + ksbpText
	case "duplicate":
		steps = ksbpHeader + ksbpCode + ksbpText + ksbpDup
	case "text-deleted":
		steps = ksbpHeader + ksbpCode
	case "unused":
		steps = ksbpHeader + ksbpCode + ksbpText + ksbpUnused
	case "load-failure":
		steps = ksbpHeader + ksbpCode + ksbpText + ksbpThrow
	case "token":
		steps = ksbpHeader + ksbpToken + ksbpText
		feature = strings.Replace(feature, "Then binding-proof: code 12", "Then binding-proof: token a4b7c012", 1)
	}
	return feature, steps
}

func TestKShapeStepBindingProofAcceptance(t *testing.T) {
	name, out := os.Getenv("KSBP_CASE"), os.Getenv("KSBP_OUT")
	feature, steps := ksbpCase(name)
	if steps == "" || out == "" {
		t.Fatalf("TestKShapeStepBindingProofAcceptance: unknown KSBP_CASE %q or no KSBP_OUT %q", name, out)
	}
	intentMD := strings.ReplaceAll(shpScopedIntentMD, "is-even", ksbpSlug)
	f := newShpFx(t, shpSubmit("c1", map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/" + ksbpSlug + "/intent.md", "content": intentMD},
		map[string]any{"path": ".kogen/intents/" + ksbpSlug + "/acceptance.feature", "content": feature},
		map[string]any{"path": ksbpStepFile, "content": steps},
	}}))
	// The fixture's other step file loads, so only the candidate's own definitions are judged.
	f.write("features/steps/calc_steps.js", "const { Given } = require('@cucumber/cucumber');\nGiven('a calculator', function () {});\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "valid calc steps")
	trace := filepath.Join(f.Aux, "sentinel-trace")
	f.Env = append(f.Env, "KSBP_SENTINEL_TRACE="+trace)
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Slug: ksbpSlug, Request: "Add the binding proof.", Client: f.Client})

	// The shaper's second request carries Kogen's answer to the submission.
	feedback := ""
	if reqs := f.Prov.requests(); len(reqs) > 1 {
		feedback = shpOutputs(reqs[1])["c1"]
	}
	b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping", ksbpSlug, "events.ndjson"))
	if rerr != nil {
		t.Fatalf("TestKShapeStepBindingProofAcceptance case %s: no shaping events: %v (exit %d, err %v)\nstderr: %s", name, rerr, exit, err, f.Stderr)
	}
	selfchecks, published := []any{}, 0
	for _, line := range strings.Split(string(b), "\n") {
		var ev map[string]any
		if line == "" || json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		switch ev["event"] {
		case "step_selfcheck":
			bound := []any{}
			items, _ := ev["items"].([]any)
			for _, raw := range items {
				if it, _ := raw.(map[string]any); it["check"] == "steps" {
					bound = append(bound, map[string]any{"text": it["target"], "ok": it["ok"], "detail": it["detail"]})
				}
			}
			warnings, _ := ev["warnings"].([]any)
			if warnings == nil {
				warnings = []any{}
			}
			selfchecks = append(selfchecks, map[string]any{"ok": ev["ok"], "steps": bound, "warnings": warnings})
		case "intent_published":
			published++
		}
	}
	sentinel, _ := os.ReadFile(trace)
	result, _ := json.Marshal(map[string]any{
		"case":       name,
		"feedback":   feedback,
		"selfchecks": selfchecks,
		// readiness: Shape went on to its readiness check; candidate_acceptance: the acceptance
		// runner ran the candidate's feature (its baseline check of the project does not count).
		"readiness":            strings.Contains(f.Stderr.String(), "kogen: checking readiness"),
		"candidate_acceptance": ksbpExists(filepath.Join(f.Aux, "feature-path")),
		"readiness_record":     f.exists(".kogen/local/readiness/" + ksbpSlug + ".json"),
		"published":            published,
		"sentinel":             string(sentinel),
	})
	if err := os.WriteFile(out, result, 0o644); err != nil {
		t.Fatal(err)
	}
}

func ksbpExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
`;
