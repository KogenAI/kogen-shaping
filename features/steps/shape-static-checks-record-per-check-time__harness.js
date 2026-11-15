'use strict';
// The Go source of the shape-static-checks-record-per-check-time acceptance harness, which
// shape-static-checks-record-per-check-time__steps.js compiles into the shape package's tests
// through a Go overlay as shape/sct_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func sctWrite(t *testing.T, f *shpFx, exit int, err error) {
	t.Helper()
	out := os.Getenv("SCT_OUT")
	if out == "" {
		t.Fatal("shape-static-checks-record-per-check-time: SCT_OUT must be set")
	}
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	events := f.events()
	var refusals []any
	for _, e := range shpNamed(events, "submission_refused") {
		if e["gate"] == "static_checks" {
			refusals = append(refusals, e["problems"])
		}
	}
	static := []any{}
	for _, e := range shpNamed(events, "static_checks") {
		static = append(static, e)
	}
	seen := map[string]any{
		"exit": exit, "error": errText,
		"published": exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve "+shpSlug),
		"static": static, "refusals": refusals, "stderr": f.Stderr.String(),
	}
	b, merr := json.MarshalIndent(seen, "", "  ")
	if merr == nil {
		merr = os.WriteFile(out, b, 0o644)
	}
	if merr != nil {
		t.Fatalf("shape-static-checks-record-per-check-time: writing the observations: %v", merr)
	}
}

// TestSCTTiming shapes a request with the checks slow (passes), stale (fails with and without the
// shaped files) and suite (a test suite, never run). A refused submission on acceptance red is
// followed by an acknowledgment of the unchanged package.
func TestSCTTiming(t *testing.T) {
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpAck("c2"))
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n    - name: slow\n      run: 'sleep 0.05'\n    - name: stale\n      run: 'sleep 0.05; echo \"stale: app.txt\"; exit 1'\n    - name: suite\n      run: 'go test ./...'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "timed checks")
	exit, err := f.shape("Add calc.is_even(n).")
	sctWrite(t, f, exit, err)
}

// TestSCTRefusal shapes a request with one check that fails on the shaped step file while it holds
// the word BAD: the first submission is refused by it, the same package is submitted again and is
// refused with the reused result, the third is repaired and refused on acceptance red, and the
// fourth acknowledges.
func TestSCTRefusal(t *testing.T) {
	first := shpFiles()
	first["files"].([]any)[2] = map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps + "// BAD\n"}
	f := newShpFx(t, shpSubmit("c1", first), shpSubmit("c2", first), shpSubmit("c3", shpFiles()), shpAck("c4"))
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n    - name: bad\n      run: 'sleep 0.05; if grep -q BAD features/steps/is-even__steps.js; then echo \"features/steps/is-even__steps.js:3:5: bad\"; exit 1; fi'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "bad check")
	exit, err := f.shape("Add calc.is_even(n).")
	sctWrite(t, f, exit, err)
}
`;
