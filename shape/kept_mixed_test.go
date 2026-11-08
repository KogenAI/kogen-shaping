package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A kept bound and a newly requested result share one scenario. A failure of
// the new result says nothing about whether the existing bound regressed.
func TestKeptBoundDoesNotOwnEveryAssertionInItsScenario(t *testing.T) {
	for _, genuine := range []bool{false, true} {
		t.Run(fmt.Sprint("genuine-kept-failure-", genuine), func(t *testing.T) {
			f := newShpFx(t)
			node, modules := os.Getenv("KOGEN_NODE"), os.Getenv("KOGEN_CUCUMBER")
			trace := filepath.Join(f.Aux, "runs")
			f.write("reader.js", "exports.read = () => "+fmt.Sprintf("%q", "first page")+";\n")
			if genuine {
				f.write("reader.js", "exports.read = () => 'x'.repeat(20000);\n")
			}
			f.write("features/steps/calc_steps.js", "// existing\n")
			f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\necho run >> '%s'\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", trace, modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
			f.git("add", "-A")
			f.git("commit", "-qm", "mixed kept and new fixture")
			feature := "Feature: Read ranges\n  Scenario: A bounded later range\n    Then is-even: the later range fits the bound\n"
			steps := `const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');
Then('is-even: the later range fits the bound',function(){
 const result=require('../../reader').read(80000);
 assert.ok(result.length<=16384,'kept bound exceeded');
 assert.strictEqual(result,'later page','new range is not implemented');
});
`
			intent := strings.ReplaceAll(shpScopedIntentMD, "calc.is_even(n) returns True for even integers.", "Read a later range while preserving the existing bound.")
			controls, _ := json.Marshal(map[string]any{"correct": map[string]any{"files": map[string]any{"reader.js": "exports.read = () => 'later page';\n"}}, "wrong": []any{map[string]any{"name": "too-large", "scenario": "A bounded later range", "step": "Then is-even: the later range fits the bound", "files": map[string]any{"reader.js": "exports.read = () => 'x'.repeat(20000);\n"}}}})
			pack := map[string]any{"files": []any{map[string]any{"path": ccIntentPath, "content": intent}, map[string]any{"path": ccFeaturePath, "content": feature}, map[string]any{"path": ccStepsPath, "content": steps}, map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": string(controls)}}}
			f.Prov.seq = []shpStep{shpSubmit("first", pack), shpSubmit("second", pack), shpSubmit("third", pack)}
			request := "Read later bytes; keep the existing result-size bound."
			prop := map[string]any{"property": "result bound", "request": "keep the existing result-size bound", "kind": "kept", "observable": true, "scenarios": []any{"A bounded later range"}, "decision": "", "issues": []any{}}
			verdict := "refuted"
			if genuine {
				verdict = "confirmed"
			}
			f.Prov.contract = []shpStep{shpAnswer(shpCall("review", SubmitContractReviewTool, map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{prop}})), ccConfirm("confirm", ccVerdict(1, verdict, "judge the assertion that actually failed"))}
			exit, err := f.shape(request)
			if genuine {
				if err == nil || exit == 0 {
					t.Fatalf("a real kept violation passed: %d %v", exit, err)
				}
			} else if err != nil || exit != 0 {
				t.Fatalf("a new assertion was treated as a kept failure: %d %v\nfeedback: %v", exit, err, shpOutputs(f.Prov.requests()[1]))
			}
			// The accepted candidate also executes its correct/wrong controls.
			wantRuns := 2
			if !genuine {
				wantRuns += 2
			}
			if b, e := os.ReadFile(trace); e != nil || strings.Count(string(b), "run\n") != wantRuns {
				t.Fatalf("unexpected readiness/control executions: %q %v; want %d", b, e, wantRuns)
			}
			repeats := shpNamed(f.events(), "kept_assertion_repeat")
			if len(repeats) != 1 || repeats[0]["execution_cap"] != float64(2) {
				t.Fatalf("repeat cap: %v", repeats)
			}
			reqs := f.Prov.contractRequests()
			if len(reqs) != 2 || contractKind(reqs[1]) != SubmitContractConfirmationTool {
				t.Fatalf("persistent observation was not independently confirmed: %v", ccKinds(f))
			}
			if !strings.Contains(ccInput(reqs[1]), "A bounded later range") {
				t.Fatal("confirmation lost the scenario")
			}
		})
	}
}
