'use strict';
// The Go source of the shape-static-check-base-run-once acceptance harness, which
// shape-static-check-base-run-once__steps.js compiles into the shape package's tests through a Go
// overlay as shape/sbro_acceptance_test.go; that path never exists in the repository. The source
// holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSBROObserve shapes a request in a project with three configured checks that fail with and
// without the shaped files: stale and twin share one command (marker file base-runs counts its
// executions without the shaped files), other has its own command (base-other-runs and
// shaped-other-runs count its executions without and with the shaped files). The scripted shaper
// is refused twice on acceptance red with different bytes, then acknowledges. SBRO_OUT receives
// the observations.
func TestSBROObserve(t *testing.T) {
	out := os.Getenv("SBRO_OUT")
	if out == "" {
		t.Fatal("shape-static-check-base-run-once: SBRO_OUT must be set")
	}
	first := shpFiles()
	second := shpFiles()
	files := second["files"].([]any)
	files[2] = map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps + "// round two\n"}
	f := newShpFx(t, shpSubmit("c1", first), shpSubmit("c2", second), shpAck("c3"))
	base := filepath.Join(f.Aux, "base-runs")
	baseOther := filepath.Join(f.Aux, "base-other-runs")
	shapedOther := filepath.Join(f.Aux, "shaped-other-runs")
	shared := "if [ -f features/steps/is-even__steps.js ]; then :; else echo base >> " + base + "; fi; echo \"stale: app.txt\"; exit 1"
	own := "if [ -f features/steps/is-even__steps.js ]; then echo shaped >> " + shapedOther + "; else echo base >> " + baseOther + "; fi; echo \"stale: app.txt\"; exit 1"
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n    - name: stale\n      run: '"+shared+"'\n    - name: twin\n      run: '"+shared+"'\n    - name: other\n      run: '"+own+"'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "stale checks")

	exit, err := f.shape("Add calc.is_even(n).")
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	lines := func(p string) int {
		b, err := os.ReadFile(p)
		if err != nil {
			return 0
		}
		return len(strings.Fields(string(b)))
	}
	seen := map[string]any{
		"exit": exit, "error": errText,
		"published": exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve "+shpSlug),
		"baseRuns": lines(base), "otherBaseRuns": lines(baseOther), "otherShapedRuns": lines(shapedOther),
		"stderr": f.Stderr.String(), "stdout": f.Stdout.String(),
	}
	b, err := json.MarshalIndent(seen, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("shape-static-check-base-run-once: writing the observations: %v", err)
	}
}
`;
