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

// Opt-in real semantic controls, only after ordinary spend admission. These
// distinguish identical names in preparation procedure and product behavior.
func TestLiteralScopeLive(t *testing.T) {
	output := os.Getenv("KOGEN_SHAPE_LIVE_OUTPUT")
	if output == "" {
		t.Skip("requires explicit admitted live evidence directory")
	}
	cases := []struct {
		name, request, contract string
		refuses                 bool
	}{
		{"capture-procedure", "Add the /ready endpoint. Engineering procedure for Shape: print the complete generated controls.json from a successful probe with capture_output=true, then use propose_files from_probe to retain it. These are shaping instructions, not additional product scope.", "The /ready endpoint returns ready.", false},
		{"inspection-procedure", "Add the /ready endpoint. Before submitting the Intent, inspect cli/native.go and run npm test in a scratch probe. The requested product is the /ready endpoint; these commands guide the shaper's work.", "The /ready endpoint returns ready.", false},
		{"capture-product-missing", "Add a probe parameter named capture_output and a proposal field named from_probe. Preserve these exact public parameter names.", "The new probe uses a parameter named capture_text and proposals use a field named output_ref.", true},
		{"route-product-changed", "Add the /ready endpoint. Engineering procedure: use probe capture_output=true to prepare the proof.", "The /readiness endpoint returns ready.", true},
		{"argv-git-operation", "The check subcommand executes git status --porcelain and returns its output.", "if (command === 'check') return execFileSync('git', ['status', '--porcelain']);", false},
		{"argv-go-operation", "The verify subcommand executes go test ./... -count=1.", "if (command === 'verify') return spawnSync('go', ['test', './...', '-count=1']);", false},
		{"argv-required-flag-missing", "The check subcommand must execute git status --porcelain.", "if (command === 'check') return execFileSync('git', ['status', '--short']);", true},
		{"display-command-changed", "The help subcommand displays exactly git status --porcelain.", "if (command === 'help') return 'git status --short'; if (command === 'check') return execFileSync('git', ['status', '--porcelain']);", true},
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
		root := t.TempDir()
		slug := c.name
		dir := filepath.Join(root, ".kogen/intents", slug)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		md := "# Product\n\nSlug: " + slug + "\nModel: not applicable — stateless\nAcceptance:\n- .kogen/intents/" + slug + "/acceptance.feature\n\n" + c.contract + "\n"
		if err := os.WriteFile(filepath.Join(dir, "intent.md"), []byte(md), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "acceptance.feature"), []byte("Feature: Product\n  Scenario: Required behavior\n    Then "+c.contract+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		ev, err := NewEvidence(filepath.Join(output, c.name), slug)
		if err != nil {
			t.Fatal(err)
		}
		sh := &shaping{s: s, repo: repo, root: root, slug: slug, request: c.request, role: role, ev: ev, client: provider.NewClient("kogen literal-scope-live"), started: time.Now()}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		lines, err := sh.literalCheck(ctx)
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
	if err = os.WriteFile(filepath.Join(output, "results.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r["pass"] != true {
			t.Errorf("literal scope control failed: %s", r["name"])
		}
	}
}
