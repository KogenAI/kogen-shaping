'use strict';
// The Go source of the shape-refuses-steps-over-step-limit acceptance harness, which
// shape-refuses-steps-over-step-limit__steps.js compiles into the shape package's tests through a Go
// overlay as shape/shape_refuses_steps_over_step_limit_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

// The acceptance harness of shape-refuses-steps-over-step-limit: one synthetic shaping journey per
// SRSL_CASE, with a scripted shaper that submits one package once, the real Shape entry point and
// the real Node and Cucumber of the step self-check. What the shaper and the run log saw is
// written as JSON to SRSL_OUT for the acceptance steps to judge. No step body ever runs: each body
// starts with a sentinel that records the run and throws, so no test suite is ever started.

const (
	srslSlug     = "limit-proof"
	srslStepFile = "features/steps/limit-proof__steps.js"
	srslHeader   = "const { Given, When, Then } = require('@cucumber/cucumber');\n" +
		"function neverRun() { require('node:fs').appendFileSync(process.env.SRSL_SENTINEL_TRACE, 'executed\\n'); throw new Error('a step body ran'); }\n"
	srslCP = "require('node:child_process')."
)

// srslStep is one step of the submitted feature and its definition body.
type srslStep struct{ keyword, text, body string }

var srslPrepared = srslStep{"Given", "the prepared project", ""}
var srslRecorded = srslStep{"Then", "the result is recorded", ""}

func srslWhen(text, body string) []srslStep {
	return []srslStep{srslPrepared, {"When", text, srslCP + body}, srslRecorded}
}

// srslCase is the steps a case submits. The case "mixed" has a narrow When step and a whole-suite
// Then step, so only the Then step may be named.
func srslCase(name string) []srslStep {
	switch name {
	case "go-shell":
		return srslWhen("I run the Go suite", "execSync('go test ./...');")
	case "go-argv":
		return srslWhen("I run the Go suite with an argument list", "spawnSync('go', ['test', '-count=1', './...']);")
	case "concurrent":
		return srslWhen("I run two uncached full Go suites concurrently", "spawn('go', ['test', '-count=1', './...']); "+srslCP+"spawn('go', ['test', '-count=1', './...']);")
	case "mix":
		return srslWhen("I run the Elixir suite", "execSync('mix test');")
	case "npm":
		return srslWhen("I run the JavaScript suite", "spawnSync('npm', ['test']);")
	case "narrow-go-test":
		return srslWhen("I run one named Go test", "spawnSync('go', ['test', '-count=1', '-run', '^TestOne$', './shape']);")
	case "narrow-go-package":
		return srslWhen("I run one Go package", "execSync('go test ./shape');")
	case "narrow-mix":
		return srslWhen("I run one Elixir test file", "execSync('mix test test/one_test.exs');")
	case "narrow-npm":
		return srslWhen("I run one JavaScript test", "spawnSync('npm', ['test', '--', '--grep', 'one']);")
	case "mixed":
		return []srslStep{srslPrepared, {"When", "I run one named Go test", srslCP + "spawnSync('go', ['test', '-count=1', '-run', '^TestOne$', './shape']);"},
			{"Then", "I run the whole Go suite", srslCP + "execSync('go test ./...');"}}
	}
	return nil
}

func TestShapeRefusesStepsOverStepLimitAcceptance(t *testing.T) {
	name, out := os.Getenv("SRSL_CASE"), os.Getenv("SRSL_OUT")
	steps := srslCase(name)
	if steps == nil || out == "" {
		t.Fatalf("TestShapeRefusesStepsOverStepLimitAcceptance: unknown SRSL_CASE %q or no SRSL_OUT %q", name, out)
	}
	feature, code := "Feature: Bound step runtime\n  Scenario: A suite step\n", srslHeader
	for _, s := range steps {
		feature += "    " + s.keyword + " " + srslSlug + ": " + s.text + "\n"
		code += s.keyword + "('" + srslSlug + ": " + s.text + "', async function () { neverRun(); " + s.body + " });\n"
	}
	intentMD := strings.ReplaceAll(shpScopedIntentMD, "is-even", srslSlug)
	f := newShpFx(t, shpSubmit("c1", map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/" + srslSlug + "/intent.md", "content": intentMD},
		map[string]any{"path": ".kogen/intents/" + srslSlug + "/acceptance.feature", "content": feature},
		map[string]any{"path": srslStepFile, "content": code},
	}}))
	// The fixture's other step file loads, so only the candidate's own definitions are judged.
	f.write("features/steps/calc_steps.js", "const { Given } = require('@cucumber/cucumber');\nGiven('a calculator', function () {});\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "valid calc steps")
	trace := filepath.Join(f.Aux, "sentinel-trace")
	f.Env = append(f.Env, "SRSL_SENTINEL_TRACE="+trace)
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Slug: srslSlug, Request: "Add the step limit proof.", Client: f.Client})

	// The shaper's second request carries Kogen's answer to the submission.
	feedback := ""
	if reqs := f.Prov.requests(); len(reqs) > 1 {
		feedback = shpOutputs(reqs[1])["c1"]
	}
	b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping", srslSlug, "events.ndjson"))
	if rerr != nil {
		t.Fatalf("TestShapeRefusesStepsOverStepLimitAcceptance case %s: no shaping events: %v (exit %d, err %v)\nstderr: %s", name, rerr, exit, err, f.Stderr)
	}
	selfchecks, published := []any{}, 0
	for _, line := range strings.Split(string(b), "\n") {
		var ev map[string]any
		if line == "" || json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		switch ev["event"] {
		case "step_selfcheck":
			limits := []any{}
			items, _ := ev["items"].([]any)
			for _, raw := range items {
				if it, _ := raw.(map[string]any); it["check"] == "step_limit" {
					limits = append(limits, it["target"])
				}
			}
			selfchecks = append(selfchecks, map[string]any{"ok": ev["ok"], "step_limit": limits})
		case "intent_published":
			published++
		}
	}
	sentinel, _ := os.ReadFile(trace)
	_, statErr := os.Stat(filepath.Join(f.Aux, "feature-path"))
	result, _ := json.Marshal(map[string]any{
		"case":       name,
		"feedback":   feedback,
		"selfchecks": selfchecks,
		// readiness: Shape went on to its readiness check; candidate_acceptance: the acceptance
		// runner ran the candidate's feature (its baseline check of the project does not count).
		"readiness":            strings.Contains(f.Stderr.String(), "kogen: checking readiness"),
		"candidate_acceptance": statErr == nil,
		"readiness_record":     f.exists(".kogen/local/readiness/" + srslSlug + ".json"),
		"published":            published,
		"sentinel":             string(sentinel),
		// approval_limit_seconds is the constant approval stops every step at.
		"approval_limit_seconds": int(verify.StepTimeout.Seconds()),
	})
	if err := os.WriteFile(out, result, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
