//go:build ignore

package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// Compiled through a Go overlay: the regression input exists before Build.
func TestRepeatControlObservation(t *testing.T) {
	mode := os.Getenv("REPEAT_CONTROL_MODE")
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
	out := map[string]any{"mode": mode, "controls": count(controls), "readiness": count(trace), "checks": checks, "repeats": repeats, "progress": f.Stderr.String(), "refusals": refused, "first_error": errorText(e1), "second_error": errorText(e2), "same_package": first.SHA256 == second.SHA256, "same_tree": first.CandidateTree == second.CandidateTree, "statuses": []string{first.Scenarios[0].Status, second.Scenarios[0].Status}}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(os.Getenv("REPEAT_CONTROL_OUT"), b, 0600); err != nil {
		t.Fatal(fmt.Errorf("write observation: %w", err))
	}
}
