//go:build ignore

package shape

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-verification/verify"
)

// The overlay makes this a real shape-package test, with the existing local
// scripted provider. It observes one post-submit journey, without live models.
func TestStaticModelOverlapObservation(t *testing.T) {
	mode := os.Getenv("OVERLAP_MODE")
	f := newShpFx(t)
	marker := filepath.Join(f.Aux, "slept")
	command := "sleep 2; echo done >> " + marker
	if mode == "refused" {
		command += "; ! grep -q BAD features/steps/is-even__steps.js"
	}
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n    - name: slow\n      run: '"+command+"'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "slow static check")
	sh := f.shaping()
	sh.request, sh.role, sh.started, sh.submissions = orderRequest, sh.cfg.Models["shape"], time.Now(), 1
	sh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S), Evidence: sh.ev}
	for _, file := range shpFiles()["files"].([]any) {
		m := file.(map[string]any)
		text := m["content"].(string)
		if mode == "refused" && strings.HasSuffix(m["path"].(string), ".js") {
			text += "\n// BAD\n"
		}
		f.write(m["path"].(string), text)
	}
	decision, literal := orderDecisionRefusal(), orderLiteralRefusal()
	f.Prov.route = func(req map[string]any) *shpStep {
		if decisionRequest(req) {
			time.Sleep(2 * time.Second)
			return &decision
		}
		if literalRequest(req) {
			return &literal
		}
		return nil
	}
	sh.startBatch()
	start := time.Now()
	err := sh.postSubmitChecks(context.Background(), shpScopedIntentMD, "", "overlap-package")
	took := time.Since(start).Milliseconds()
	if err != nil {
		t.Fatal(err)
	}
	text, _, _ := sh.settleHeld(sh.collectBatch())
	var gates, semantic []string
	var totals []any
	for _, e := range f.events() {
		name := e["event"].(string)
		if name == "submission_refused" {
			gates = append(gates, e["gate"].(string))
			totals = append(totals, e["total"])
		}
		if name == "static_checks" || name == "decision_check" || name == "literal_check" || name == "example_check" {
			semantic = append(semantic, name)
		}
	}
	slept, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"mode": mode, "elapsed_ms": took, "sleep_runs": len(strings.Fields(string(slept))), "decision_calls": len(f.Prov.decisionRequests()), "literal_calls": len(f.Prov.literalRequests()), "gates": gates, "events": semantic, "totals": totals, "feedback": strings.ReplaceAll(text, marker, "<marker>")}
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(os.Getenv("OVERLAP_OUT"), b, 0600); err != nil {
		t.Fatal(err)
	}
}
