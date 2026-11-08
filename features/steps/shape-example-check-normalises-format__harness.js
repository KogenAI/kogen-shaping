'use strict';
module.exports = String.raw`package shape

import (
 "encoding/json"
 "os"
 "slices"
 "strings"
 "testing"
)

func TestShapeExampleFormatAcceptance(t *testing.T) {
 expected, derived := os.Getenv("SEF_EXPECTED"), os.Getenv("SEF_DERIVED")
 feature := "Feature: Formatting output\n  Scenario Outline: Formatted output\n    Given is-even: the derived value <derived>\n    Then is-even: the output is <expected>\n    Examples:\n      | expected | derived |\n      | " + expected + " | " + derived + " |\n"
 steps := "const {Given,Then}=require('@cucumber/cucumber');\nGiven(/^is-even: the derived value (.*)$/,function(v){this.v=v;});\nThen(/^is-even: the output is (.*)$/,function(v){require('node:assert/strict').equal(require('../../calc').format(this.v),v);});\n"
 files := shpFiles()
 list := files["files"].([]any)
 intent := list[0].(map[string]any)["content"].(string)
 intent = strings.Replace(intent, "calc.is_even(n) returns True for even integers.", "Formatting produces the JSON object with status 0 and out string 61610a0a.", 1)
 if derived == "<not-given>" {
  intent = strings.Replace(intent, "Formatting produces the JSON object with status 0 and out string 61610a0a.", "Formatting behaviour is unspecified for this example.", 1)
 }
 list[0].(map[string]any)["content"] = intent
 list[1].(map[string]any)["content"] = feature
 list[2].(map[string]any)["content"] = steps
 ack := shpSubmit("ack", map[string]any{"files": []any{map[string]any{"path":".kogen/intents/is-even/acceptance.feature","content":feature}}})
 f := newShpFx(t, shpSubmit("candidate", files), ack)
 f.setFeatureReport("report", feature, "PASSED", 0)
 f.write("calc.js", "exports.format = () => '';\n")
 f.git("add", "-A")
 f.git("commit", "-qm", "prepare formatting fixture")
 calls := 0
 f.Prov.route = func(req map[string]any) *shpStep {
  if !slices.Contains(shpToolNames(req), "submit_example_check") { return nil }
  calls++
  value := derived
  if value == "<not-given>" { value = "" }
  response := shpAnswer(shpCall("example-result", "submit_example_check", map[string]any{"items": []any{map[string]any{
   "scenario":"Formatted output", "expected":expected, "derived":value, "verdict":"contradicts", "why":"scripted raw-string mismatch",
  }}}))
  return &response
 }
 code, err := f.shape("Formatting produces status 0 and out string 61610a0a; equivalent representations follow, real differences contradict, unspecified values do not contradict.")
 verdict := ""
 refused := false
 for _, ev := range f.events() {
  if ev["event"] == "submission_refused" && ev["gate"] == "example_check" { refused = true }
  if ev["event"] == "example_check" {
   raw, _ := json.Marshal(ev["items"])
   var items []map[string]any
   _ = json.Unmarshal(raw, &items)
   for _, item := range items {
    if item["expected"] == expected { verdict, _ = item["verdict"].(string) }
   }
  }
 }
 observation := map[string]any{"calls":calls,"verdict":verdict,"refused":refused,"exit":code,"stderr":f.Stderr.String()}
 if err != nil { observation["error"] = err.Error() }
 data, e := json.Marshal(observation)
 if e != nil { t.Fatal(e) }
 if e = os.WriteFile(os.Getenv("SEF_OUT"), data, 0600); e != nil { t.Fatal(e) }
}
`;
