'use strict';
module.exports = String.raw`package shape

import (
 "context"
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "github.com/KogenAI/kogen-coordinator/project"
 "github.com/KogenAI/kogen-loop/sys"
)

type draftResult struct { Exit int; Class string; Reason string; Detail string; Lines []string; Output string; Marker bool }
func draftError(exit int, err error, output string, marker bool) draftResult {
 r:=draftResult{Exit:exit,Output:output,Marker:marker}
 if e,ok:=sys.AsError(err);ok { r.Class=e.Class;r.Reason=e.Reason;r.Detail=e.Detail;r.Lines=e.Lines } else if err!=nil {r.Detail=err.Error()}
 return r
}
func TestDraftApprovalJourney(t *testing.T) {
 f:=newShpFx(t,shpSubmit("c1",shpFiles()),shpAck("c2"),shpAck("c3"))
 marker:=".kogen/local/approvals/is-even.json"
 approve:=func(hash string) draftResult {
  f.newSession()
  exit,err:=Approve(context.Background(),f.S,ApproveInput{Project:f.Root,Slug:shpSlug,Hash:hash,By:"Fixture",Tools:verifyToolsForDraft()})
  return draftError(exit,err,f.Stdout.String(),f.exists(marker))
 }
 sha:=func() string {
  cfg,err:=project.LoadConfig(f.Root);if err!=nil {t.Fatal(err)}
  paths,err:=project.Baseline(f.Root,shpSlug,cfg);if err!=nil {t.Fatal(err)}
  return project.Hash(f.Root,paths)
 }
 out:=map[string]any{}
 // A never-shaped package uses the existing fixture bytes, not an invented ready receipt.
 for _,v:=range shpFiles()["files"].([]any) {m:=v.(map[string]any);f.write(m["path"].(string),m["content"].(string))}
 out["legacyCard"]=approve("")
 stale:=sha()
 out["legacyApproval"]=approve(stale)
 if err:=os.RemoveAll(filepath.Join(f.Root,".kogen/intents/is-even"));err!=nil {t.Fatal(err)}
 if err:=os.Remove(filepath.Join(f.Root,marker));err!=nil && !os.IsNotExist(err) {t.Fatal(err)}
 // Remove local publication made by legacy approval before starting the fresh Shape draft.
 for _,p:=range []string{".kogen/local/readiness/is-even.json",".kogen/local/packages/is-even.json"} {os.Remove(filepath.Join(f.Root,p))}
 f.setReport("UNDEFINED",1)
 exit,err:=f.shapeFrom("Add calc.is_even(n).",filepath.Join(f.Aux,"request.md"))
 out["notReady"]=draftError(exit,err,f.Stdout.String(),f.exists(marker))
 // Make readiness pass on its own. The refusal must still be sticky.
 f.setReport("FAILED",1)
 out["blockedCard"]=approve("")
 out["blockedHash"]=approve(sha())
 out["staleHash"]=approve(stale)
 f.write(".kogen/intents/is-even/intent.md",f.read(".kogen/intents/is-even/intent.md")+"\nHand edit.\n")
 out["handEdit"]=approve(sha())
 f.Prov.mu.Lock(); f.Prov.seq=append(f.Prov.seq,shpSubmit("retry",shpFiles()),shpAck("retry-ack"));f.Prov.mu.Unlock()
 exit,err=f.shapeFrom("Add calc.is_even(n).",filepath.Join(f.Aux,"request.md"))
 out["retry"]=draftError(exit,err,f.Stdout.String(),f.exists(marker))
 out["readyCard"]=approve("")
 out["readyApproval"]=approve(sha())
 exit,err=f.shapeFrom("Add calc.is_even(n).",filepath.Join(f.Aux,"request.md"))
 out["exists"]=draftError(exit,err,f.Stdout.String(),f.exists(marker))
 paths:=[]string{}
 filepath.WalkDir(filepath.Join(f.Root,".kogen/intents/is-even"),func(p string,d os.DirEntry,e error)error{if e!=nil{return e};if !d.IsDir(){rel,_:=filepath.Rel(f.Root,p);paths=append(paths,filepath.ToSlash(rel))};return nil})
 paths=append(paths,"features/steps/is-even__steps.js")
 out["files"]=paths
 // Preserve exact diagnostics for acceptance, including gate names from Shape.
 b,e:=json.Marshal(out);if e!=nil{t.Fatal(e)}
 if e=os.WriteFile(os.Getenv("DRAFT_RESULT"),b,0600);e!=nil{t.Fatal(e)}
 fmt.Println("draft journey recorded")
 _=strings.Contains
}
`;
