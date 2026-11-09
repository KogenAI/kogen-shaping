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

// Run only after normal model-spend admission, with an existing authenticated
// HOME and KOGEN_SHAPE_LIVE_OUTPUT naming a private evidence directory:
// go test -tags shape_live ./shape -run TestDecisionContextLive -count=1 -v
// This tests the semantic checker with real inference; the ordinary suite is offline.
func TestDecisionContextLive(t *testing.T) {
	output := os.Getenv("KOGEN_SHAPE_LIVE_OUTPUT")
	if output == "" {
		t.Skip("explicit admitted live evaluation requires KOGEN_SHAPE_LIVE_OUTPUT")
	}
	cases := []struct {
		name, request, intent string
		refuses               bool
	}{
		{"repository-output", "Improve the controls feedback. Preserve existing bounds.", `# Controls
## Decisions
- controlsMaxFiles remains 8. (decision: D1, round 1) (evidence: shape/controls.go) (source: request)
## Examples
Each implementation can contain at most 8 files.
`, false},
		{"assumption-restated", "Only the owner may waive an action. Choose unspecified details and record them for approval.", `# Actions
## Decisions
- Identify the owner by the action who email. (decision: D1, round 1) (assumption: who names the owner) (source: request)
## Examples
The authenticated account email must equal who before a waiver is accepted.
`, false},
		{"authority-conflict", "Only the owner may waive an action. Never allow another member to waive it.", `# Actions
## Decisions
- Any member may waive an action. (decision: D1, round 1) (evidence: access.ex) (assumption: any member suffices) (source: request)
## Examples
Any member may waive an action, including a member who is not its owner.
`, true},
		{"narrowed-outcomes", "The hands-off view shows owner actions and all completed outcomes, including non-Build outcomes.", `# Views
## Decisions
- The hands-off view shows owner actions and completed Builds only. (decision: D1, round 1) (evidence: views.ex) (assumption: completed Builds represent completed outcomes) (source: request)
## Examples
A completed non-Build outcome is hidden from the hands-off view.
`, true},
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
	if role.Provider != "chatgpt" || role.External() {
		t.Fatal("this live control requires the repository's native ChatGPT Shape lineup")
	}
	var results []map[string]any
	for _, c := range cases {
		ev, err := NewEvidence(filepath.Join(output, c.name), c.name)
		if err != nil {
			t.Fatal(err)
		}
		sh := &shaping{s: s, repo: repo, root: repo.Dir, slug: c.name, request: c.request, role: role, ev: ev, client: provider.NewClient("kogen shape-live"), started: time.Now(), readPaths: map[string]bool{"shape/controls.go": true}}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		lines, err := sh.decisionCheck(ctx, c.intent)
		cancel()
		ev.Finish()
		refuses := len(lines) > 0
		result := map[string]any{"name": c.name, "refuses": refuses, "expected_refusal": c.refuses, "feedback": lines, "pass": err == nil && refuses == c.refuses}
		if err != nil {
			result["error"] = err.Error()
		}
		results = append(results, result)
		t.Logf("%s: refuses=%v expected=%v error=%v", c.name, refuses, c.refuses, err)
	}
	raw, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(output, "results.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result["pass"] != true {
			t.Errorf("decision context control failed: %s", result["name"])
		}
	}
}
