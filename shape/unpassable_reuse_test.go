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

// Exercise the real readiness gate and kept-repeat route with local control launches.
func TestUnpassableCompletedReuse(t *testing.T) {
	for _, mode := range []string{"repeat", "package", "tree", "skipped", "refusal", "error"} {
		t.Run(mode, func(t *testing.T) {
			f, sh, _, _ := unpassableFixture(t, "", []controlProbe{{name: "kept", body: "assert.equal(require('../../calc').value, 'expected');"}}, nil)
			ctx := context.Background()
			node, modules := os.Getenv("KOGEN_NODE"), os.Getenv("KOGEN_CUCUMBER")
			trace, controls := filepath.Join(f.Aux, "readiness"), filepath.Join(f.Aux, "controls")
			wrapper := filepath.Join(f.Aux, "node-wrapper")
			script := "#!/bin/sh\nif [ \"$1\" = -r ]; then\n echo run >> " + ShellQuote(controls) + "\n"
			if mode == "skipped" {
				script += " if [ ! -f " + ShellQuote(filepath.Join(f.Aux, "once")) + " ]; then touch " + ShellQuote(filepath.Join(f.Aux, "once")) + "; exit 3; fi\n"
			}
			script += "fi\nexec " + ShellQuote(node) + " \"$@\"\n"
			if err := os.WriteFile(wrapper, []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			f.write("features/steps/calc_steps.js", "// existing\n")
			f.write("accept.sh", "#!/bin/sh\necho run >> "+ShellQuote(trace)+"\nNODE_PATH="+ShellQuote(modules)+" "+ShellQuote(node)+" "+ShellQuote(filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js"))+" --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n")
			sh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S)}
			sh.rd.Tools.Node = wrapper
			sh.started = time.Now()
			sh.role = project.Role{Provider: "chatgpt", Model: "gpt-6.1-sol", Effort: "high"}
			sh.rd.Gate = func(r *verify.Ready) error {
				if err := readinessGate(r); err != nil {
					return err
				}
				if err := sh.preparationGate(r); err != nil {
					return err
				}
				return sh.unpassableGate(ctx, r)
			}
			var first, second *verify.Ready
			var e1, e2 error
			refused := []string{}
			switch mode {
			case "repeat", "skipped":
				first, e1 = sh.rd.Check(ctx, f.S, shpSlug)
				if e1 != nil {
					t.Fatalf("first Check: %v", e1)
				}
				sh.ready = first
				sh.request = "keep the calculation unchanged"
				sh.lastObservationsPath = ".kogen/local/repeat-observations.txt"
				f.write(sh.lastObservationsPath, "first observation\n")
				prop := map[string]any{"property": "calculation unchanged", "request": sh.request, "kind": "kept", "observable": true, "scenarios": []any{"kept"}, "decision": "", "issues": []any{}, "kept_steps": []any{3}}
				review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{prop}}
				f.Prov.contract = []shpStep{shpAnswer(shpCall("review", SubmitContractReviewTool, review)), shpAnswer(shpCall("confirm", SubmitContractConfirmationTool, map[string]any{"verdicts": []any{map[string]any{"finding": 1, "verdict": "confirmed", "why": "persistent kept mismatch"}}}))}
				result, err := sh.contractCheck(ctx, first.SHA256, "first observation")
				if err != nil {
					t.Fatalf("contractCheck: %v", err)
				}
				refused = result.all()
				second = sh.ready
			case "package", "tree":
				first, e1 = sh.rd.Check(ctx, f.S, shpSlug)
				if e1 != nil {
					t.Fatalf("first Check: %v", e1)
				}
				if mode == "package" {
					f.write(project.FeaturePath(shpSlug), f.read(project.FeaturePath(shpSlug))+"# changed package bytes\n")
				} else {
					f.write("calc.js", "exports.value = 'still wrong';\n")
				}
				second, e2 = sh.rd.Check(ctx, f.S, shpSlug)
				if e2 != nil {
					t.Fatalf("second Check: %v", e2)
				}
			case "refusal", "error":
				intent, _, err := project.CheckIntent(f.Root, shpSlug, nil)
				if err != nil {
					t.Fatal(err)
				}
				tree, err := f.Repo.Tree(f.Repo.Head(), nil)
				if err != nil {
					t.Fatal(err)
				}
				baseline, err := project.Baseline(f.Root, shpSlug, sh.cfg)
				if err != nil {
					t.Fatal(err)
				}
				detail := "acceptance.feature line 3: Then step kept\nError: stub"
				if mode == "refusal" {
					f.write("features/steps/is_even_steps.js", "const {Then}=require('@cucumber/cucumber');\nThen('step kept',function(){throw new Error('stub');});\n")
					tree, err = f.Repo.Tree(f.Repo.Head(), nil)
					if err != nil {
						t.Fatal(err)
					}
				}
				first = &verify.Ready{Intent: intent, CandidateTree: tree, SHA256: project.Hash(f.Root, baseline), Scenarios: []verify.ScenarioResult{{Name: "kept", Line: 2, Status: "failed", Detail: &detail, Steps: []verify.StepRecord{{Line: 3, Keyword: "Then ", Text: "step kept"}}}}}
				if mode == "error" {
					canceled, cancel := context.WithCancel(ctx)
					cancel()
					e1 = sh.unpassableGate(canceled, first)
				} else {
					e1 = sh.unpassableGate(ctx, first)
				}
				e2 = sh.unpassableGate(ctx, first)
				second = first
			default:
				t.Fatalf("unknown mode %q", mode)
			}
			count := func(p string) int { b, _ := os.ReadFile(p); return strings.Count(string(b), "run\n") }
			checks := shpNamed(f.events(), "unpassable_check")
			repeats := shpNamed(f.events(), "kept_assertion_repeat")
			if repeats == nil {
				repeats = []map[string]any{}
			}
			if checks == nil {
				checks = []map[string]any{}
			}
			errorText := func(e error) string {
				if e == nil {
					return ""
				}
				if u, ok := e.(*unpassableError); ok {
					return u.text()
				}
				return e.Error()
			}

			wantControls, wantEvents, wantReadiness := 1, 1, 2
			if mode == "package" || mode == "tree" || mode == "skipped" {
				wantControls, wantEvents = 2, 2
			}
			if mode == "refusal" || mode == "error" {
				wantReadiness = 0
			}
			if count(controls) != wantControls || len(checks) != wantEvents || count(trace) != wantReadiness {
				t.Fatalf("launches=%d events=%v readiness=%d", count(controls), checks, count(trace))
			}
			if mode == "repeat" || mode == "skipped" {
				if len(repeats) != 1 || repeats[0]["outcome"] != "persistent kept assertion red" || repeats[0]["execution_cap"] != float64(2) || len(refused) == 0 {
					t.Fatalf("repeat=%v refusals=%v", repeats, refused)
				}
				for _, text := range []string{"kogen: repeating kept assertion observation (execution 2 of 2 maximum)", `kogen: kept assertion repeat outcome: persistent kept assertion red in "kept" (line 2) (cap 2 executions)`} {
					if !strings.Contains(f.Stderr.String(), text) {
						t.Fatalf("missing progress %q: %s", text, f.Stderr.String())
					}
				}
			}
			if wantReadiness > 0 {
				if first.Scenarios[0].Status != "failed" || second.Scenarios[0].Status != "failed" {
					t.Fatal("readiness did not stay red")
				}
				if (first.SHA256 == second.SHA256) != (mode != "package") || (first.CandidateTree == second.CandidateTree) != (mode != "package" && mode != "tree") {
					t.Fatal("unexpected candidate identity")
				}
			}
			if mode != "refusal" {
				completed := checks[len(checks)-1]
				steps := completed["steps"].([]any)
				if len(steps) != 1 || steps[0].(map[string]any)["verdict"] != verdictStandInUsed || completed["ok"] != true {
					t.Fatalf("completed judgment: %v", completed)
				}
			}
			if mode == "skipped" && checks[0]["skipped"] == nil {
				t.Fatalf("first control not skipped: %v", checks)
			}
			if mode == "error" && (e1 == nil || e2 != nil) {
				t.Fatalf("error retry: %v, %v", e1, e2)
			}
			if mode == "refusal" {
				const want = "Acceptance has steps that no implementation can pass: 1 step definition(s) fail the same way whatever the repository contains, so no pass is reachable; nothing was published. Each step must be able to pass once the requested behaviour exists: run the program, call the code or inspect its output, and compare what it observed with an assertion that shows the actual value (for example assert.strictEqual). Rewrite these steps; resubmitting them unchanged is refused again.\n- acceptance.feature line 3: Then step kept (step definition \"step kept\", features/steps/is_even_steps.js line 2): no pass is reachable: it fails with \"Error: stub\" whatever the repository contains"
				if errorText(e1) != want || errorText(e2) != want {
					t.Fatalf("refusal changed: %q, %q", errorText(e1), errorText(e2))
				}
			}
		})
	}
}
