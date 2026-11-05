'use strict';
// The Go source of the fix-k-shape-environment-stop-evidence acceptance harness, which
// fix-k-shape-environment-stop-evidence__steps.js compiles into the shape package's tests through
// a Go overlay as shape/fix_k_shape_environment_stop_evidence_acceptance_test.go; that path never
// exists in the repository. The source holds no backquote and no dollar-brace, so this raw
// template keeps it byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of fix-k-shape-environment-stop-evidence: one external shaping run in the
// shape tests' is-even project, with lib/weather.js committed and lib/secret.js committed as a
// symlink to a file outside the checkout. The scripted session asks one question citing both
// files, and the question check answers ui_ux. What the run showed is written as JSON to
// FKSESE_OUT for the acceptance steps to judge. The harness judges nothing itself.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

const fkseseSecret = "OUTSIDE-SECRET-7f3a: this file lies outside the checkout\n"

func TestFixKShapeEnvironmentStopEvidenceAcceptance(t *testing.T) {
	out := os.Getenv("FKSESE_OUT")
	if out == "" {
		t.Fatal("FKSESE_OUT must be set")
	}
	outside := filepath.Join(t.TempDir(), "outside-secret.txt")
	if err := os.WriteFile(outside, []byte(fkseseSecret), 0o644); err != nil {
		t.Fatal(err)
	}
	f := newShpFx(t)
	f.useExternalShape()
	qcCommit(f)
	if err := os.Symlink(outside, filepath.Join(f.Root, "lib", "secret.js")); err != nil {
		t.Fatal(err)
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "Add a symlink to a file outside the checkout")
	ask := map[string]any{"questions": []any{shpQuestion("Needed? (evidence: lib/weather.js; lib/secret.js)", "yes", "no")}}
	check := result(map[string]any{"items": []any{map[string]any{"number": float64(1), "class": "ui_ux", "why": "w"}}})
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(ask), check}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n).")
	obs := map[string]any{"exit": exit, "error": "", "sessions": len(x.starts), "cited_evidence": "",
		"tree_entry_mode": strings.Fields(f.git("ls-tree", "HEAD", "--", "lib/secret.js") + " -")[0],
		"secret_sent":     strings.Contains(fmt.Sprintf("%+v %+v", x.starts, x.inputs), "OUTSIDE-SECRET-7f3a")}
	if err != nil {
		obs["error"] = err.Error()
	}
	if len(x.starts) > 1 {
		stable := x.starts[1].Brief.Stable
		if i := strings.Index(stable, "\n\n# Cited evidence\n\n"); i >= 0 {
			obs["cited_evidence"] = stable[i+len("\n\n# Cited evidence\n\n"):]
		}
	}
	b, jerr := json.MarshalIndent(obs, "", " ")
	if jerr != nil {
		t.Fatal(jerr)
	}
	if werr := os.WriteFile(out, b, 0o644); werr != nil {
		t.Fatal(werr)
	}
}
`;
