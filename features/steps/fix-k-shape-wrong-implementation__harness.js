'use strict';
// The Go source of the fix-k-shape-wrong-implementation acceptance harness, which
// fix-k-shape-wrong-implementation__steps.js compiles into the shape package's tests through a Go
// overlay as shape/fix_k_shape_wrong_implementation_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps
// it byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of fix-k-shape-wrong-implementation. Each case builds the real-run
// fixture of the wrong-implementation check (controlsReal), arranges one failure and calls
// wrongImplementationCheck as checkSubmission does. What each case showed is written as JSON to
// FKSWI_OUT for the acceptance steps to judge. The harness judges nothing itself.

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

// fkswiEvents are the results of the wrong_implementation_check events recorded so far.
func fkswiEvents(f *shpFx) []string {
	out := []string{}
	file, err := os.Open(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug, "events.ndjson"))
	if err != nil {
		return out
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var e map[string]any
		if json.Unmarshal(sc.Bytes(), &e) == nil && e["event"] == "wrong_implementation_check" {
			r, _ := e["result"].(string)
			out = append(out, r)
		}
	}
	return out
}

// fkswiCheck runs the check and observes it.
func fkswiCheck(f *shpFx, sh *shaping, hash string, around func(func())) map[string]any {
	var confirmed contractConfirmed
	var err error
	call := func() { confirmed, err = sh.wrongImplementationCheck(context.Background(), hash) }
	if around != nil {
		around(call)
	} else {
		call()
	}
	obs := map[string]any{"error": "", "refusal": append([]string{}, confirmed.wrong...),
		"runs": strings.Count(f.Stderr.String(), "kogen: wrong-implementation check: run "), "events": fkswiEvents(f)}
	if err != nil {
		obs["error"] = err.Error()
	}
	return obs
}

func TestFixKShapeWrongImplementationAcceptance(t *testing.T) {
	out := os.Getenv("FKSWI_OUT")
	if out == "" {
		t.Fatal("FKSWI_OUT must be set")
	}
	exact := "Then is-even: add of 2 and 3 is 5"
	candidate := func() controlsCandidate {
		return controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
			correct: map[string]string{"calc.js": controlsCalc("add", "sub")}, wrong: controlsWrongs(exact, controlsCalc("sub"))}
	}
	obs := map[string]any{}

	f, sh := controlsReal(t, candidate())
	sh.ready.CandidateTree = "0123456789abcdef0123456789abcdef01234567"
	obs["materialize"] = fkswiCheck(f, sh, "h-materialize", nil)

	f, sh = controlsReal(t, candidate())
	notDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(notDir, []byte("a file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	obs["kit"] = fkswiCheck(f, sh, "h-kit", func(call func()) {
		old, had := os.LookupEnv("TMPDIR")
		os.Setenv("TMPDIR", notDir)
		defer func() {
			if had {
				os.Setenv("TMPDIR", old)
			} else {
				os.Unsetenv("TMPDIR")
			}
		}()
		call()
	})

	f, sh = controlsReal(t, candidate())
	sh.cfg.Setup = []string{"exit 3"}
	obs["setup"] = fkswiCheck(f, sh, "h-setup", nil)

	c := candidate()
	c.correct = map[string]string{"calc.js/inner.js": controlsCalc("add", "sub")}
	f, sh = controlsReal(t, c)
	obs["overlay"] = fkswiCheck(f, sh, "h-overlay", nil)

	f, sh = controlsReal(t, candidate())
	if err := os.Remove(filepath.Join(f.Root, project.IntentMDPath(shpSlug))); err != nil {
		t.Fatal(err)
	}
	_, cerr := sh.contractCheck(context.Background(), "h-stale", "")
	stale := map[string]any{"contract_error": cerr != nil, "review_properties": len(sh.reviewProps)}
	for k, v := range fkswiCheck(f, sh, "h-stale", nil) {
		stale[k] = v
	}
	obs["stale"] = stale

	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
