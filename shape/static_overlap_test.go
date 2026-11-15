package shape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-verification/verify"
)

// Exercise the configured process and scripted provider through one real submission.
// The upper bound leaves a clear margin below sequential two-second waits.
func TestStaticChecksOverlapModelChecks(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprintf("static_refused_%t", refused), func(t *testing.T) {
			parallel(t)
			f := newShpFx(t)
			marker := filepath.Join(f.Aux, "static-completed")
			command := "sleep 2; echo done >> " + marker
			if refused {
				command += "; ! grep -q BAD features/steps/is-even__steps.js"
			}
			f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n    - name: slow\n      run: '"+command+"'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
			f.git("add", "-A")
			f.git("commit", "-qm", "sleeping check")
			sh := f.shaping()
			sh.request, sh.role, sh.started, sh.submissions = orderRequest, sh.cfg.Models["shape"], time.Now(), 1
			sh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S), Evidence: sh.ev}
			for _, file := range shpFiles()["files"].([]any) {
				m := file.(map[string]any)
				text := m["content"].(string)
				if refused && strings.HasSuffix(m["path"].(string), ".js") {
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
			if err := sh.postSubmitChecks(context.Background(), shpScopedIntentMD, "", "overlap-package"); err != nil {
				t.Fatal(err)
			}
			took := time.Since(start)
			text, _, invalid := sh.settleHeld(sh.collectBatch())
			if !invalid {
				t.Fatal("expected refusal")
			}
			completed, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			wantRuns := 1
			wantEvents := []string{staticGate, "decision_check", "literal_check"}
			wantGates := []string{"decision_check", "literal_check"}
			if refused {
				wantRuns = 3 // candidate, traced candidate and base attribution
				wantEvents, wantGates = []string{staticGate}, []string{staticGate}
				wantText := "Static checks: the shaped files fail 1 configured static check(s); nothing was published. Each check ran unchanged on the final submitted bytes in a fresh copy of the candidate; repair the named files so that every check passes. Resubmitting this package unchanged is refused again.\n- check slow: exit 1; files: features/steps/is-even__steps.js; command: " + command + "\n  | (no output)"
				shpEq(t, "static feedback", text, wantText)
				if len(sh.refusedStatements) != 0 || len(sh.pendingMarks) != 0 || len(sh.notes) != 0 {
					t.Fatalf("discarded decision effects survived: %v %v %v", sh.refusedStatements, sh.pendingMarks, sh.notes)
				}
			} else {
				if took < 1900*time.Millisecond || took >= 3500*time.Millisecond {
					t.Fatalf("independent two-second waits took %v", took)
				}
				if !strings.HasPrefix(text, "Decision check:") || !strings.Contains(text, "\n\nLiteral check:") {
					t.Fatalf("combined feedback: %s", text)
				}
			}
			shpEq(t, "completed static runs", len(strings.Fields(string(completed))), wantRuns)
			shpEq(t, "decision calls", len(f.Prov.decisionRequests()), 1)
			shpEq(t, "literal calls", len(f.Prov.literalRequests()), 1)
			var events, gates []string
			for _, e := range f.events() {
				name := e["event"].(string)
				if slices.Contains([]string{staticGate, "decision_check", "literal_check", "example_check"}, name) {
					events = append(events, name)
				}
				if name == "submission_refused" {
					gates = append(gates, e["gate"].(string))
					shpEq(t, "retry total", e["total"], any(float64(1)))
				}
			}
			if !slices.Equal(events, wantEvents) || !slices.Equal(gates, wantGates) {
				t.Fatalf("events %v gates %v; want %v %v", events, gates, wantEvents, wantGates)
			}
		})
	}
}
