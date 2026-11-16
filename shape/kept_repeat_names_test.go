package shape

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// The kept assertion repeat names the scenarios whose assertion is red in both
// observations, in the order of the sorted refusal lines, on the progress line
// and in the event. With none to name, both stay as before.
func TestKeptRepeatOutcomeNamesPersistentRedScenarios(t *testing.T) {
	red := "const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');\nThen('step red', function () { assert.strictEqual('observed', 'expected', 'expected=expected observed=observed'); });\n"
	firstOnly := func(marker string) string {
		return "const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict'); const fs=require('node:fs');\nThen('step red', function () { if (!fs.existsSync(" + jsQuote(marker) + ")) { fs.writeFileSync(" + jsQuote(marker) + ", 'x'); assert.strictEqual('observed', 'expected', 'expected=expected observed=observed'); } });\n"
	}
	cases := []struct {
		name, feature, steps string
		kept                 []string
		line                 string
		scenarios            []map[string]any
	}{
		{"one", "Feature: Accounts\n" + strings.Repeat("  # filler\n", 10) + "  Scenario: Admin accounts list links each account\n    Then step red\n", red, []string{"Admin accounts list links each account"},
			`persistent kept assertion red in "Admin accounts list links each account" (line 12)`, []map[string]any{{"name": "Admin accounts list links each account", "line": float64(12)}}},
		{"sorted", "Feature: Order\n\n  Scenario: Zebra\n    Then step red\n\n  # separator\n  Scenario: Ant\n    Then step red\n", red, []string{"Zebra", "Ant"},
			`persistent kept assertion red in "Ant" (line 7), "Zebra" (line 3)`, []map[string]any{{"name": "Ant", "line": float64(7)}, {"name": "Zebra", "line": float64(3)}}},
		{"first-only", "Feature: Once\n\n  Scenario: Flaky once\n    Then step red\n", "", []string{"Flaky once"}, "no persistent kept assertion red", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newShpFx(t)
			node, modules := os.Getenv("KOGEN_NODE"), os.Getenv("KOGEN_CUCUMBER")
			steps := c.steps
			if steps == "" {
				steps = firstOnly(filepath.Join(t.TempDir(), "seen"))
			}
			f.write(".kogen/intents/is-even/intent.md", shpIntentMD)
			f.write(project.FeaturePath(shpSlug), c.feature)
			f.write("features/steps/is_even_steps.js", steps)
			f.write("features/steps/calc_steps.js", "// existing steps\n")
			f.write("accept.sh", "#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\nNODE_PATH="+ShellQuote(modules)+" "+ShellQuote(node)+" "+ShellQuote(filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js"))+" --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n")
			sh := f.shaping()
			sh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S)}
			sh.started = time.Now()
			sh.role = project.Role{Provider: "chatgpt", Model: "gpt-6.1-sol", Effort: "high"}
			ctx := context.Background()
			first, err := sh.rd.Check(ctx, f.S, shpSlug)
			if err != nil {
				t.Fatal(err)
			}
			sh.ready = first
			sh.request = "keep the existing results"
			sh.lastObservationsPath = ".kogen/local/repeat-observations.txt"
			f.write(sh.lastObservationsPath, "first observation\n")
			names := []any{}
			for _, k := range c.kept {
				names = append(names, k)
			}
			prop := map[string]any{"property": "existing results", "request": sh.request, "kind": "kept", "observable": true, "scenarios": names, "decision": "", "issues": []any{}}
			review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{prop}}
			verdicts := []any{}
			for i := 1; i <= 4; i++ {
				verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "confirmed", "why": "persistent kept mismatch"})
			}
			f.Prov.contract = []shpStep{shpAnswer(shpCall("review", SubmitContractReviewTool, review)), shpAnswer(shpCall("confirm", SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts}))}
			if _, err := sh.contractCheck(ctx, first.SHA256, "first observation"); err != nil {
				t.Fatal(err)
			}
			want := "kogen: kept assertion repeat outcome: " + c.line + " (cap 2 executions)"
			found := false
			for _, l := range strings.Split(f.Stderr.String(), "\n") {
				if strings.HasPrefix(l, "kogen: kept assertion repeat outcome:") {
					found = true
					if i := strings.LastIndex(l, " ("); !strings.HasPrefix(l, want) || i < len(want)-len(" (cap 2 executions)") {
						t.Fatalf("progress line %q, want prefix %q", l, want)
					}
				}
			}
			if !found {
				t.Fatalf("no outcome line: %s", f.Stderr.String())
			}
			repeats := shpNamed(f.events(), "kept_assertion_repeat")
			if len(repeats) != 1 {
				t.Fatalf("events: %v", repeats)
			}
			got, has := repeats[0]["scenarios"]
			if c.scenarios == nil {
				if has {
					t.Fatalf("unexpected scenarios key: %v", repeats[0])
				}
				return
			}
			list, _ := got.([]any)
			if len(list) != len(c.scenarios) {
				t.Fatalf("scenarios %v, want %v", got, c.scenarios)
			}
			for i, w := range c.scenarios {
				m, _ := list[i].(map[string]any)
				if len(m) != 2 || m["name"] != w["name"] || m["line"] != w["line"] {
					t.Fatalf("scenario %d: %v, want %v", i, m, w)
				}
			}
		})
	}
}

func jsQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `\'`) + "'" }
