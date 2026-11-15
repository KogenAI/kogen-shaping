package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are local synthetic document and summary fixtures, not Studio or ExUnit runs.
func TestMaskedKeptStepBaseline(t *testing.T) {
	for _, mode := range []string{"error", "summary", "assertion", "new-only", "kept-first"} {
		t.Run(mode, func(t *testing.T) {
			f := newShpFx(t)
			node, modules := os.Getenv("KOGEN_NODE"), os.Getenv("KOGEN_CUCUMBER")
			trace := filepath.Join(f.Aux, "runs")
			f.write("document.js", "exports.document = {latest:{body:'hello'}}; exports.summary = '1 passed, 6 excluded';\n")
			f.write("features/steps/calc_steps.js", "// existing\n")
			f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\necho run >> '%s'\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", trace, modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
			f.git("add", "-A")
			f.git("commit", "-qm", "masked kept fixture")
			steps := `const {When,Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');
When('is-even: studio_records is available',function(){const m=require('../../document'); if(typeof m.studio_records!=='function') throw Error('Unknown tool studio_records'); assert.equal(m.studio_records(),'ready');});
When('is-even: studio_records returns ready',function(){assert.equal(require('../../document').studio_records?.(),'ready','new studio_records result');});
Then('is-even: document.body is hello',function(){assert.equal(require('../../document').document.body,'hello','document.body is nested under latest');});
Then('is-even: latest document body is hello',function(){assert.equal(require('../../document').document.latest.body,'hello');});
Then('is-even: the summary says ran exactly once',function(){assert.equal(require('../../document').summary,'ran exactly once');});
`
			first := "    When is-even: studio_records is available\n"
			kept := "    Then is-even: document.body is hello\n"
			if mode == "assertion" {
				first = "    When is-even: studio_records returns ready\n"
			}
			if mode == "summary" {
				kept = "    Then is-even: the summary says ran exactly once\n"
			}
			keptLine := 4
			if mode == "kept-first" {
				kept = "    Then is-even: latest document body is hello\n"
				keptLine = 3
			}
			feature := "Feature: Preserve document output\n  Scenario: Mixed document\n" + first + kept
			if mode == "new-only" {
				feature = "Feature: Preserve document output\n  Scenario: Mixed document\n" + strings.Replace(first, "When", "Then", 1)
				keptLine = 0
			}
			if mode == "kept-first" {
				feature = "Feature: Preserve document output\n  Scenario: Mixed document\n" + kept + first
			}
			request := "Add studio_records; keep the document envelope and test summary unchanged."
			intent := "# Preserve document output\n\nSlug: is-even\nModel: not applicable — stateless output validation\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" + request + "\n\n## Decisions\n- Preserve the current output. (source: request)\n\n## Source comparison\n- Addressed: the request by the feature.\n\n## Tests\n- Output: acceptance (scenario)\n\n## Feasibility\n- none\n"
			correct := "exports.document = {body:'hello',latest:{body:'hello'}}; exports.summary = 'ran exactly once'; exports.studio_records=()=> 'ready';\n"
			wrongNew := strings.Replace(correct, "()=> 'ready'", "()=> 'wrong'", 1)
			newStep := "When is-even: studio_records is available"
			if mode == "new-only" {
				newStep = "Then is-even: studio_records is available"
			}
			if mode == "assertion" {
				newStep = "When is-even: studio_records returns ready"
			}
			wrong := []any{map[string]any{"name": "new output wrong", "scenario": "Mixed document", "step": newStep, "files": map[string]any{"document.js": wrongNew}}}
			if keptLine != 0 {
				wrongKept := strings.ReplaceAll(correct, "hello", "wrong")
				if mode == "summary" {
					wrongKept = strings.Replace(correct, "ran exactly once", "1 passed, 6 excluded", 1)
				}
				wrong = append(wrong, map[string]any{"name": "kept output wrong", "scenario": "Mixed document", "step": strings.TrimSpace(kept), "files": map[string]any{"document.js": wrongKept}})
			}
			controls, e := json.Marshal(map[string]any{"correct": map[string]any{"files": map[string]any{"document.js": correct}}, "wrong": wrong})
			if e != nil {
				t.Fatal(e)
			}
			pack := map[string]any{"files": []any{
				map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
				map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feature},
				map[string]any{"path": "features/steps/is-even__steps.js", "content": steps},
				map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": string(controls)},
			}}
			prop := func(kind string, lines []any) map[string]any {
				return map[string]any{"property": kind + " output", "request": request, "kind": kind, "observable": true, "scenarios": []any{"Mixed document"}, "decision": "", "issues": []any{}, "kept_steps": lines}
			}
			props := []any{prop("new", []any{})}
			if keptLine != 0 {
				props = append(props, prop("kept", []any{keptLine}))
			}
			review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": props}
			verdicts := []any{}
			for i := 1; i <= 8; i++ {
				verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "refuted", "why": "new behaviour is not implemented"})
			}
			for i := 0; i < 4; i++ {
				f.Prov.contract = append(f.Prov.contract, shpAnswer(shpCall(fmt.Sprint("review", i), SubmitContractReviewTool, review)), shpAnswer(shpCall(fmt.Sprint("confirm", i), SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts})))
			}
			f.Prov.seq = []shpStep{shpSubmit("first", pack), shpSubmit("second", pack), shpSubmit("third", pack)}
			exit, err := f.shape(request)
			feedback := ""
			for _, r := range f.Prov.requests() {
				if s, ok := shpOutputs(r)["first"]; ok {
					feedback = s
					break
				}
			}
			runs := 0
			if b, e := os.ReadFile(trace); e == nil {
				runs = strings.Count(string(b), "run\n")
			}

			published := exit == 0 && err == nil && f.exists(".kogen/local/readiness/is-even.json")
			wantPublished := mode == "new-only" || mode == "kept-first"
			if published != wantPublished {
				t.Fatalf("published=%v, want %v; exit=%d err=%v; feedback=%s", published, wantPublished, exit, err, feedback)
			}
			wantRuns := 2
			if mode == "new-only" {
				wantRuns = 3
			}
			if mode == "kept-first" {
				wantRuns = 4
			}
			if runs != wantRuns {
				t.Fatalf("runs=%d, want %d; feedback=%s", runs, wantRuns, feedback)
			}
			if !wantPublished {
				for _, text := range []string{"Mixed document", "line 4", "never runs on the unchanged tree", "studio_records", "own scenario"} {
					if !strings.Contains(feedback, text) {
						t.Errorf("feedback lacks %q: %s", text, feedback)
					}
				}
			}
			if f.read(".kogen/intents/is-even/acceptance.feature") != feature || f.read("features/steps/is-even__steps.js") != steps {
				t.Fatal("Shape rewrote the draft scenarios or bindings")
			}
		})
	}
}
