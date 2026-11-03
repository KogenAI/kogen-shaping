'use strict';
module.exports = String.raw`package shape

import (
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestKeptGreenJourney(t *testing.T) {
 mode := os.Getenv("KG_MODE")
 f := newShpFx(t)
 node := os.Getenv("KOGEN_NODE")
 if node == "" { node = "node" }
 modules := os.Getenv("KOGEN_CUCUMBER")
 trace := filepath.Join(f.Aux, "runs")
 f.write("f.js", "exports.f = x => 'observed-A';\n")
 f.write("features/steps/calc_steps.js", "// existing\n")
 f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\necho run >> '%s'\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", trace, modules, node, filepath.Join(modules,"@cucumber/cucumber/bin/cucumber.js")))
 f.git("add", "-A"); f.git("commit", "-qm", "synthetic kept fixture")
 request := "pin the current output of function f for input A; no behaviour change"
 kind := "kept"
 want := "wrong-A"
 if mode == "green" { want = "observed-A" }
 if mode == "plain" { request = "Make f return wrong-A for input A"; kind = "new" }
 if mode == "mixed" { request = "keep behaviour B and add C" }
 steps := "const {Then}=require('@cucumber/cucumber'); const assert=require('node:assert/strict'); const fs=require('node:fs');\nThen('is-even: f of A is {string}', function(want) { let actual=require('../../f').f('A');"
 if mode == "missing" { steps += "actual=fs.readFileSync('absent-fixture.txt','utf8');" }
 if mode == "flaky" { steps += fmt.Sprintf("const p=%q; if(fs.existsSync(p)) actual=want; else fs.writeFileSync(p,'first');", filepath.Join(f.Aux,"flaky")) }
 steps += "assert.strictEqual(actual,want,'expected='+want+' observed='+actual); });\n"
 feature := "Feature: Pin output\n  Scenario: B pins A\n    Then is-even: f of A is \""+want+"\"\n"
 if mode == "mixed" { feature += "  Scenario: C is new\n    Then is-even: C returns ready\n"; steps += "Then('is-even: C returns ready',function(){ assert.strictEqual(require('../../f').c,'ready','expected=ready observed=undefined'); });\n" }
 intent := "# Pin the current output\n\nSlug: is-even\nModel: not applicable — input/output only\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n"+request+"\n\n## Decisions\n- Preserve the current output. (source: request)\n\n## Source comparison\n- The request is exercised by the feature.\n\n## Tests\n- Output: acceptance (scenario)\n\n## Feasibility\n- none\n"
 pack := func(feat string) map[string]any { return map[string]any{"files": []any{
 map[string]any{"path":".kogen/intents/is-even/intent.md","content":intent},
 map[string]any{"path":".kogen/intents/is-even/acceptance.feature","content":feat},
 map[string]any{"path":"features/steps/is-even__steps.js","content":steps},
 }} }
 prop := func(name, quote, k, scenario string) map[string]any { return map[string]any{"property":name,"request":quote,"kind":k,"observable":true,"scenarios":[]any{scenario},"decision":"","issues":[]any{}} }
 props := []any{}
 if mode != "plain" { props = append(props, prop("B stays unchanged",request,kind,"B pins A")) }
 if mode == "mixed" { props = append(props,prop("C is added","add C","new","C is new")) }
 review := map[string]any{"findings":[]any{},"environment":[]any{},"properties":props}
 verdicts := []any{}
 for i:=1;i<=8;i++ { verdicts=append(verdicts,map[string]any{"finding":i,"verdict":"confirmed","why":"synthetic confirmation of observed mismatch"}) }
 for i:=0;i<4;i++ { f.Prov.contract=append(f.Prov.contract,shpAnswer(shpCall(fmt.Sprint("review",i),SubmitContractReviewTool,review)),shpAnswer(shpCall(fmt.Sprint("confirm",i),SubmitContractConfirmationTool,map[string]any{"verdicts":verdicts}))) }
 first := pack(feature)
 second := first
 expectedFeature := feature
 if mode == "correct" { expectedFeature=strings.ReplaceAll(feature,"wrong-A","observed-A"); second=pack(expectedFeature) }
 f.Prov.seq=[]shpStep{shpSubmit("first",first),shpSubmit("second",second),shpSubmit("third",second)}
 exit,err:=f.shape(request)
 published:=exit==0 && err==nil && f.exists(".kogen/local/readiness/is-even.json")
 if published {
  actualFeature:=f.read(".kogen/intents/is-even/acceptance.feature")
  // Only the scripted second submission may correct the expectation. Shape must not
  // rewrite or delete any scenario or its steps while publishing either submission.
  expectedPublished:=feature
  if mode=="correct" && len(f.Prov.requests())>1 {expectedPublished=expectedFeature}
  if actualFeature!=expectedPublished {t.Fatalf("TestKeptGreenJourney: Shape changed or dropped scenarios: expected %q observed %q",expectedPublished,actualFeature)}
  if actualSteps:=f.read("features/steps/is-even__steps.js");actualSteps!=steps {t.Fatalf("TestKeptGreenJourney: Shape rewrote scenario bindings: expected %q observed %q",steps,actualSteps)}
 }
 outputs:=[]string{}
 for _,r:=range f.Prov.requests(){ if s,ok:=shpOutputs(r)["first"];ok { outputs=append(outputs,s) } }
 count:=0
 if b,e:=os.ReadFile(trace);e==nil { count=strings.Count(string(b),"run\n") }
 b,e:=json.Marshal(map[string]any{"published":published,"feedback":strings.Join(outputs,"\n"),"requests":len(f.Prov.requests()),"contracts":len(f.Prov.contractRequests()),"runs":count,"progress":f.Stderr.String(),"outcome":fmt.Sprintf("exit=%d err=%v",exit,err)})
 if e!=nil {t.Fatal(e)}
 if e=os.WriteFile(os.Getenv("KG_OUT"),b,0600);e!=nil {t.Fatal(e)}
}
`;
