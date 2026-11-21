'use strict';
module.exports = String.raw`package shape

import (
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func TestRepairRedMainJourney(t *testing.T) {
 mode := os.Getenv("KG_MODE")
 out := map[string]any{}
 if mode == "timing" {
  _, ev := stcShape(t, []string{"slow"}, map[string]stcSpec{"slow": {120 * time.Second, 137, true}})
  b, _ := json.Marshal(ev)
  out["event"] = string(b)
  b, e := json.Marshal(out)
  if e != nil { t.Fatal(e) }
  if e = os.WriteFile(os.Getenv("KG_OUT"), b, 0600); e != nil { t.Fatal(e) }
  return
 }
 f := newShpFx(t)
 node := os.Getenv("KOGEN_NODE")
 if node == "" { node = "node" }
 modules := os.Getenv("KOGEN_CUCUMBER")
 f.write("f.js", "exports.f = x => 'rude-' + x;\n")
 f.write("features/steps/calc_steps.js", "// existing\n")
 f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
 f.git("add", "-A"); f.git("commit", "-qm", "red main fixture")
 request := "Make the greeting polite."
 if mode != "norepair" { request += " This request repairs the failing check \"Greeting is polite\" on main." }
 first := "polite-A"
 if mode == "green" { first = "rude-A" }
 second := "rude-B"
 if mode == "keeps" { second = "wrong-B" }
 steps := "const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict');\nThen('is-even: f of {word} is {string}', function(x, want) { const actual=require('../../f').f(x); assert.strictEqual(actual,want,'expected='+want+' observed='+actual); });\n"
 feature := "Feature: Greeting\n  Scenario: Greeting is polite\n    Then is-even: f of A is \"" + first + "\"\n  Scenario: Other stays\n    Then is-even: f of B is \"" + second + "\"\n"
 intent := "# Make the greeting polite\n\nSlug: is-even\nModel: not applicable — input/output only\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" + request + "\n\n## Decisions\n- Keep other output. (source: request)\n\n## Source comparison\n- The request is exercised by the feature.\n\n## Tests\n- Output: acceptance (scenario)\n\n## Feasibility\n- none\n"
 pack := map[string]any{"files": []any{
  map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
  map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feature},
  map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": "{\"deferred\":\"synthetic fixture; the Build writes the implementation\"}\n"},
  map[string]any{"path": "features/steps/is-even__steps.js", "content": steps},
 }}
 prop := func(name, scenario string, line int) map[string]any {
  return map[string]any{"property": name, "request": request, "kind": "kept", "observable": true, "scenarios": []any{scenario}, "decision": "", "issues": []any{}, "kept_steps": []any{line}}
 }
 props := []any{prop("Greeting stays checked", "Greeting is polite", 3), prop("Other output stays", "Other stays", 5)}
 review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": props}
 verdicts := []any{}
 for i := 1; i <= 8; i++ { verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "refuted", "why": "synthetic"}) }
 for i := 0; i < 4; i++ { f.Prov.contract = append(f.Prov.contract, shpAnswer(shpCall(fmt.Sprint("review", i), SubmitContractReviewTool, review)), shpAnswer(shpCall(fmt.Sprint("confirm", i), SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts}))) }
 f.Prov.seq = []shpStep{shpSubmit("first", pack), shpSubmit("second", pack), shpSubmit("third", pack)}
 exit, err := f.shape(request)
 published := exit == 0 && err == nil && f.exists(".kogen/local/readiness/is-even.json")
 outputs := []string{}
 for _, r := range f.Prov.requests() { if s, ok := shpOutputs(r)["first"]; ok { outputs = append(outputs, s) } }
 out["published"] = published
 out["feedback"] = strings.Join(outputs, "\n") + "\n" + fmt.Sprint(err)
 out["progress"] = f.Stderr.String()
 b, e := json.Marshal(out)
 if e != nil { t.Fatal(e) }
 if e = os.WriteFile(os.Getenv("KG_OUT"), b, 0600); e != nil { t.Fatal(e) }
}
`;
