//go:build shape_live

package shape

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
)

// Explicitly admitted semantic controls, not fabricated readiness executions.
func TestContractBuildChecksLive(t *testing.T) {
	output := os.Getenv("KOGEN_SHAPE_LIVE_OUTPUT")
	if output == "" {
		t.Skip("requires explicit admitted live evidence directory")
	}
	cases := []struct {
		name, command, condition string
		refuses                  bool
	}{
		{"whole-go-suite", "GOFLAGS=-trimpath go test ./...", "The existing whole Go suite must remain green. Engineering procedure for Shape: inspect calc.js, keep acceptance offline and focused, and submit complete files.", false},
		{"whole-npm-suite", "npm test", "The existing npm test suite must remain green. Engineering procedure for Shape: use a scratch probe to prepare bounded controls.", false},
		{"partial-go-suite", "go test ./shape -run '^TestOne$'", "The existing whole Go suite must remain green.", true},
		{"unrelated-check", "true", "The existing whole Go suite must remain green.", true},
		{"missing-offline-product", "go test ./...", "The product must also provide an offline report command that never makes network requests.", true},
		{"missing-kept-bound", "go test ./...", "Keep the product's existing request cache limit of 8 unchanged.", true},
	}
	s := sys.New()
	repo, err := git.Open(s.Cwd, s.Env)
	if err != nil {
		t.Fatal(err)
	}
	var results []map[string]any
	for _, c := range cases {
		cfg, err := project.LoadConfig(repo.Dir)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Checks = []project.Check{{Name: "go-test", Run: c.command}}
		role := cfg.Models["shape"]
		role.Provider, role.Model, role.Effort = "chatgpt", "gpt-6.1-terra", "high"
		root := t.TempDir()
		step := "features/steps/is-even__steps.js"
		body := `const {Given,Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict'); const calc=require('../../calc.js');
Given('is-even: the number {int}',function(n){this.n=n});
Then('is-even: it is even',function(){assert.equal(calc.is_even(this.n),true)});
Then('is-even: it is not even',function(){assert.equal(calc.is_even(this.n),false)});
`
		files := map[string]string{project.IntentMDPath(shpSlug): shpScopedIntentMD, project.FeaturePath(shpSlug): shpFeature, step: body, "calc.js": "exports.is_even=n=>n%2===0;\n"}
		for name, data := range files {
			p := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
		}
		ev, err := NewEvidence(filepath.Join(output, c.name), shpSlug)
		if err != nil {
			t.Fatal(err)
		}
		sh := &shaping{s: s, repo: repo, root: root, slug: shpSlug, cfg: cfg, role: role, request: "Add calc.is_even(n): true for even integers and false for odd integers. " + c.condition, ev: ev, client: provider.NewClient("kogen contract-build-checks-live"), proposed: map[string]string{step: body}, started: time.Now()}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		confirmed, err := sh.contractCheck(ctx, "semantic-screen", "Readiness was not executed in this isolated semantic screen. No readiness result is claimed.")
		cancel()
		ev.Finish()
		result := ""
		data, readErr := os.ReadFile(filepath.Join(output, c.name, "events.ndjson"))
		for _, line := range strings.Split(string(data), "\n") {
			var event map[string]any
			if json.Unmarshal([]byte(line), &event) == nil && event["event"] == "contract_review" {
				result, _ = event["result"].(string)
			}
		}
		refuses := len(confirmed.all()) > 0
		row := map[string]any{"name": c.name, "refuses": refuses, "expected_refusal": c.refuses, "feedback": confirmed.all(), "result": result, "pass": err == nil && readErr == nil && (result == "clean" || result == "refused") && refuses == c.refuses}
		if err != nil {
			row["error"] = err.Error()
		}
		results = append(results, row)
		t.Logf("%s: refuses=%v expected=%v result=%s error=%v", c.name, refuses, c.refuses, result, err)
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
			t.Errorf("contract coverage control failed: %s", r["name"])
		}
	}
}
