'use strict';
// The Go source of the k-shape-fixture-self-proof acceptance harness, which
// k-shape-fixture-self-proof__steps.js compiles into the shape package's tests through a Go overlay
// as shape/k_shape_fixture_self_proof_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"kogen-loop/sys"
)

// The acceptance harness of k-shape-fixture-self-proof: one scripted native Shape journey per run
// against the loopback provider (no model), in a synthetic repository whose acceptance command
// runs the real Node and Cucumber. KFSP_JOURNEY names a JSON file with the request, the committed
// base files and the files of each submission c1, c2, ...; the scripted shaper submits them in
// order and every contract review answers no finding. What the shaper and the run log saw is
// written as JSON to KFSP_OUT for the acceptance steps to judge.

const kfspSlug = "fixture-proof"

const kfspRun = "NODE_PATH=\"$KOGEN_CUCUMBER\" \"$KOGEN_NODE\" \"$KOGEN_CUCUMBER/@cucumber/cucumber/bin/cucumber.js\" --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\""

var kfspConfig = "version: 1\ncommands:\n  setup: []\n  checks: [{name: unit, run: 'true'}]\n  acceptance:\n    run: '" + strings.ReplaceAll(kfspRun, "'", "''") + "'\n    steps_dir: features/steps\n"

var kfspCallID = regexp.MustCompile("^c[0-9]+$")

// kfspFiles are the files of a journey list: objects with a path and a content.
func kfspFiles(t *testing.T, what string, v any) []map[string]any {
	list, ok := v.([]any)
	if !ok {
		t.Fatalf("TestKShapeFixtureSelfProofAcceptance: %s is not a list of files", what)
	}
	var out []map[string]any
	for _, e := range list {
		m, ok := e.(map[string]any)
		_, p := m["path"].(string)
		_, c := m["content"].(string)
		if !ok || !p || !c {
			t.Fatalf("TestKShapeFixtureSelfProofAcceptance: %s holds an entry without a path and a content", what)
		}
		out = append(out, m)
	}
	return out
}

func TestKShapeFixtureSelfProofAcceptance(t *testing.T) {
	in, out := os.Getenv("KFSP_JOURNEY"), os.Getenv("KFSP_OUT")
	raw, err := os.ReadFile(in)
	var journey map[string]any
	if err != nil || out == "" || json.Unmarshal(raw, &journey) != nil {
		t.Fatalf("TestKShapeFixtureSelfProofAcceptance: unreadable KFSP_JOURNEY %q or no KFSP_OUT %q: %v", in, out, err)
	}
	request, _ := journey["request"].(string)
	submissions, _ := journey["submissions"].([]any)
	if request == "" || len(submissions) == 0 {
		t.Fatalf("TestKShapeFixtureSelfProofAcceptance: journey %s has no request or no submission", in)
	}
	var steps []shpStep
	for i, s := range submissions {
		files := []any{}
		for _, f := range kfspFiles(t, fmt.Sprintf("submission %d", i+1), s) {
			files = append(files, f)
		}
		steps = append(steps, shpSubmit(fmt.Sprintf("c%d", i+1), map[string]any{"files": files}))
	}
	f := newShpFxRun(t, "'true'", steps...)
	f.write(".kogen/project.yaml", kfspConfig)
	f.write("features/steps/calc_steps.js", "'use strict';\n")
	for _, b := range kfspFiles(t, "base", journey["base"]) {
		f.write(b["path"].(string), b["content"].(string))
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "fixture-proof base")
	trace := filepath.Join(f.Aux, "fixture-proof-trace")
	f.Env = append(f.Env, "FIXTURE_PROOF_TRACE="+trace)
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Slug: kfspSlug, Request: request, Client: f.Client})
	stop := ""
	if err != nil {
		if e, ok := sys.AsError(err); ok {
			exit, stop = e.Exit, e.Render()
		} else {
			exit, stop = -1, err.Error()
		}
	}

	// Kogen's answer to submission c<n>, as the shaper's next request carries it.
	feedback := map[string]string{}
	for _, req := range f.Prov.requests() {
		for id, text := range shpOutputs(req) {
			if kfspCallID.MatchString(id) {
				feedback[id] = text
			}
		}
	}
	refusals := []any{}
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping", kfspSlug, "events.ndjson")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if line == "" || json.Unmarshal([]byte(line), &ev) != nil || ev["event"] != "submission_refused" {
				continue
			}
			refusals = append(refusals, map[string]any{"attempt": ev["attempt"], "gate": ev["gate"]})
		}
	}
	// The readiness record that publication keeps: each scenario's status and unexercised steps.
	var record any
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/readiness", kfspSlug+".json")); err == nil {
		var rec map[string]any
		if json.Unmarshal(b, &rec) == nil {
			scenarios := []any{}
			list, _ := rec["scenarios"].([]any)
			for _, s := range list {
				sm, _ := s.(map[string]any)
				unexercised := []any{}
				skipped, _ := sm["unexercised_steps"].([]any)
				for _, st := range skipped {
					m, _ := st.(map[string]any)
					unexercised = append(unexercised, fmt.Sprintf("acceptance.feature line %v: %s %v", m["line"], strings.TrimSpace(fmt.Sprint(m["keyword"])), m["text"]))
				}
				scenarios = append(scenarios, map[string]any{"name": sm["name"], "status": sm["status"], "unexercised": unexercised})
			}
			record = scenarios
		}
	}
	executed, _ := os.ReadFile(trace)
	result, _ := json.Marshal(map[string]any{
		"exit": exit, "stop": stop, "feedback": feedback, "refusals": refusals, "record": record, "trace": string(executed),
	})
	if err := os.WriteFile(out, result, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
