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

func TestKShapeDecideNotAskRecommendedAcceptance(t *testing.T) {
 name := os.Getenv("KDN_CASE")
 out := map[string]any{}
 request := "Add calc.is_even(n), which returns true for even integers.\n"
 q := func(rec bool) map[string]any {
  return map[string]any{"question":"How should non-integers be handled?", "options":[]any{
   map[string]any{"label":"Raise TypeError", "recommended":rec},
   map[string]any{"label":"Return false", "recommended":false},
  }}
 }
 ask := func(rec bool) map[string]any { return map[string]any{"questions":[]any{q(rec)}} }
 switch name {
 case "recommended", "unrecommended":
  recommended := name == "recommended"
  line := "- Non-integers raise TypeError. (assumption: use Raise TypeError for non-integers) (source: request)"
  f := newShpFx(t, shpSubmit("q1", ask(recommended)), shpSubmit("f1", shpFilesWithDecisions(line)))
  f.setReport("PASSED", 0)
  check := shpUI(1)
  f.Prov.route = func(req map[string]any) *shpStep {
   for _, tool := range shpToolNames(req) { if tool == SubmitQuestionCheckTool { return &check } }
   return nil
  }
  exit, err := f.shape(request)
  if err != nil { out["error"] = err.Error() }
  out["exit"] = exit
  out["asked"] = strings.Count(f.Stdout.String(), "1. How should non-integers be handled?")
  b, _ := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md"))
  out["assumption"] = strings.Contains(string(b), line)
  out["pending"] = f.exists(".kogen/local/shaping/is-even/questions.json")
  out["ready"] = f.exists(".kogen/local/readiness/is-even.json")
  out["diagnostic"] = f.Stdout.String()+f.Stderr.String()
 case "stale":
  f := newShpFx(t)
  second, seeded, observed := false, false, false
  stale := ".kogen/local/shaping/is-even/questions.json"
  f.Prov.route = func(req map[string]any) *shpStep {
   if contractKind(req) != "" || decisionRequest(req) || literalRequest(req) { return nil }
   if !second && !seeded {
    // Model the reported leftover artifact in a session that ends unsuccessfully.
    f.write(stale, "{\"version\":1,\"data\":{\"questions\":[{\"number\":1,\"question\":\"stale\",\"options\":[]}]}}\n")
    seeded = true
   }
   if second && !observed { out["carried"] = f.exists(stale); observed = true }
   step := shpAnswer()
   return &step
  }
  exit, err := f.shape(request)
  out["failed"] = err != nil || exit != 0
  f.newSession()
  second = true
  _, _ = f.shape(request)
  out["second_started"] = observed
 case "answer", "assumption":
  line := "- Non-integers raise TypeError. (source: answer 1)"
  if name == "assumption" { line = "- Non-integers raise TypeError. (assumption: use Raise TypeError for non-integers) (source: request)" }
  intent := "# Add parity\n\n## Decisions\n"+line+"\n"
  item := decisionItem{statement:line, quotes:[]string{}, verdict:"unsupported", why:"not in the initial request"}
  _, problems := decisionOutcome([]decisionItem{item}, request, intent)
  out["accepted"] = len(problems) == 0
  out["diagnostic"] = strings.Join(problems,"\n")
 default: t.Fatalf("unknown KDN_CASE %q", name)
 }
 b, err := json.Marshal(out)
 if err != nil { t.Fatal(err) }
 if err := os.WriteFile(os.Getenv("KDN_OUT"), b, 0600); err != nil { t.Fatal(err) }
 fmt.Println("KDN observation written")
}
`;
