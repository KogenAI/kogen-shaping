//go:build shape_live

package shape

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
)

// Explicit opt-in paired semantic controls after ordinary spend admission.
func TestDecisionEvidenceLive(t *testing.T) {
	output := os.Getenv("KOGEN_SHAPE_LIVE_OUTPUT")
	if output == "" {
		t.Skip("requires admitted live evidence directory")
	}
	cases := []struct {
		name, request, line string
		refuses             bool
	}{
		{"keep-current-limits", "Improve controls feedback and keep the existing bounds.", "- controlsMaxRuns stays 8, controlsMaxFiles stays 8, and controlsMaxBytes stays 65536. (evidence: shape/controls.go) (source: request)", false},
		{"unsupported-new-limit", "Improve controls feedback.", "- Set controlsMaxFiles to 9. (source: request)", true},
		{"wrong-cited-limit", "Improve controls feedback and keep the existing bounds.", "- controlsMaxFiles stays 9. (evidence: shape/controls.go) (source: request)", true},
		{"request-overrides-baseline", "Change controlsMaxFiles to 7.", "- controlsMaxFiles stays 8. (evidence: shape/controls.go) (source: request)", true},
	}
	s := sys.New()
	repo, err := git.Open(s.Cwd, s.Env)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := project.LoadConfig(repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	role := cfg.Models["shape"]
	role.Provider = "chatgpt"
	role.Model = "gpt-6.1-terra"
	role.Effort = "high"
	var results []map[string]any
	for _, c := range cases {
		ev, err := NewEvidence(filepath.Join(output, c.name), c.name)
		if err != nil {
			t.Fatal(err)
		}
		sh := &shaping{s: s, repo: repo, root: repo.Dir, slug: c.name, request: c.request, role: role, ev: ev, client: provider.NewClient("kogen decision-evidence-live"), started: time.Now(), readPaths: map[string]bool{"shape/controls.go": true}}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		lines, err := sh.decisionCheck(ctx, "# Controls\n\n## Decisions\n"+c.line+"\n")
		cancel()
		ev.Finish()
		refuses := len(lines) > 0
		row := map[string]any{"name": c.name, "refuses": refuses, "expected_refusal": c.refuses, "feedback": lines, "pass": err == nil && refuses == c.refuses}
		if err != nil {
			row["error"] = err.Error()
		}
		results = append(results, row)
		t.Logf("%s: refuses=%v expected=%v error=%v", c.name, refuses, c.refuses, err)
	}
	raw, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(output, "evidence-results.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r["pass"] != true {
			t.Errorf("evidence control failed: %s", r["name"])
		}
	}
}
