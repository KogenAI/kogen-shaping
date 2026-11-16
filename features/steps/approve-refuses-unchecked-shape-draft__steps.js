'use strict';
const {Then}=require('@cucumber/cucumber');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const cp=require('node:child_process');

const GO=String.raw`package shape

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

type ucResult struct {
	Exit   int
	Class  string
	Reason string
	Detail string
	Lines  []string
	Output string
	Marker bool
}

const ucMarker = ".kogen/local/approvals/is-even.json"
const ucLatch = ".kogen/local/shaping/is-even/not-ready.json"
const ucFeature = ".kogen/intents/is-even/acceptance.feature"
const ucIntent = ".kogen/intents/is-even/intent.md"

func ucFill(r *ucResult, err error) {
	if e, ok := sys.AsError(err); ok {
		r.Class, r.Reason, r.Detail, r.Lines = e.Class, e.Reason, e.Detail, e.Lines
		if r.Exit == 0 {
			r.Exit = e.Exit
		}
	} else if err != nil {
		r.Detail = err.Error()
	}
}

func ucApprove(f *shpFx, hash string) ucResult {
	f.newSession()
	tools := verify.Tools{Node: os.Getenv("KOGEN_NODE"), Cucumber: os.Getenv("KOGEN_CUCUMBER"), Quint: os.Getenv("KOGEN_QUINT")}
	exit, err := Approve(context.Background(), f.S, ApproveInput{Project: f.Root, Slug: shpSlug, Hash: hash, By: "Fixture", Tools: tools})
	r := ucResult{Exit: exit, Output: f.Stdout.String(), Marker: f.exists(ucMarker)}
	ucFill(&r, err)
	return r
}

func ucShape(f *shpFx, request string) ucResult {
	exit, err := f.shape(request)
	r := ucResult{Exit: exit, Output: f.Stdout.String(), Marker: f.exists(ucMarker)}
	ucFill(&r, err)
	return r
}

func ucHash(t *testing.T, f *shpFx) string {
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := project.Baseline(f.Root, shpSlug, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return project.Hash(f.Root, paths)
}

func ucRequests(f *shpFx) int {
	f.Prov.mu.Lock()
	defer f.Prov.mu.Unlock()
	return len(f.Prov.reqs)
}

func ucAppend(f *shpFx, steps ...shpStep) {
	f.Prov.mu.Lock()
	f.Prov.seq = append(f.Prov.seq, steps...)
	f.Prov.mu.Unlock()
}

func ucInterruptStep(f **shpFx) shpStep {
	return shpStep{handler: func(w http.ResponseWriter, r *http.Request) {
		(*f).S.Interrupt("SIGINT")
		<-r.Context().Done()
	}}
}

func ucInterrupted(f *shpFx, request string) (int, bool) {
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: request, Client: f.Client})
	var i *sys.Interrupted
	return exit, shpAsInterrupted(err, &i) && i.Signal == "SIGINT"
}

func TestUncheckedDraftJourney(t *testing.T) {
	out := map[string]any{}
	{
		var f *shpFx
		f = newShpFx(t, shpSubmit("c1", shpFiles()), ucInterruptStep(&f))
		f.setReport("FAILED", 1)
		exit, ok := ucInterrupted(f, "Add calc.is_even(n).")
		m := map[string]any{
			"interruptExit": exit, "interrupted": ok,
			"intentExists": f.exists(ucIntent), "featureExists": f.exists(ucFeature), "latchExists": f.exists(ucLatch),
		}
		out["s1"] = m
		m["card"] = ucApprove(f, "")
		m["hashApprove"] = ucApprove(f, ucHash(t, f))
		before := ucRequests(f)
		ucAppend(f, shpSubmit("retry", shpFiles()), shpAck("retry-ack"), shpAck("retry-ack2"))
		m["retry"] = ucShape(f, "Add calc.is_even(n).")
		m["retryRequests"] = ucRequests(f) - before
	}
	{
		f := newShpFx(t, shpSubmit("c1", shpFiles()), shpAck("c2"), shpAck("c3"))
		f.setReport("FAILED", 1)
		ready := ucShape(f, "Add calc.is_even(n).")
		m := map[string]any{"ready": ready, "featureExists": f.exists(ucFeature)}
		out["s2"] = m
		m["card"] = ucApprove(f, "")
		original := f.read(ucFeature)
		f.write(ucFeature, original+"# edited after the checks passed\n")
		m["editedHash"] = ucApprove(f, ucHash(t, f))
		f.write(ucFeature, original)
		m["restoredCard"] = ucApprove(f, "")
	}
	{
		f := newShpFx(t, shpSubmit("c1", shpFiles()), shpAck("c2"), shpAck("c3"))
		f.setReport("FAILED", 1)
		first := ucShape(f, "Add calc.is_even(n).")
		firstHash := ucHash(t, f)
		if err := os.RemoveAll(filepath.Join(f.Root, project.IntentDir(shpSlug))); err != nil {
			t.Fatal(err)
		}
		ucAppend(f, shpSubmit("d1", shpFiles()), ucInterruptStep(&f))
		exit, ok := ucInterrupted(f, "Add calc.is_even(n) for big integers.")
		m := map[string]any{"first": first, "interruptExit": exit, "interrupted": ok,
			"featureExists": f.exists(ucFeature), "latchExists": f.exists(ucLatch), "sameHash": firstHash == ucHash(t, f)}
		out["s3"] = m
		m["hashApprove"] = ucApprove(f, ucHash(t, f))
	}
	{
		f := newShpFx(t, shpSubmit("c1", shpFiles()), shpAck("c2"), shpAck("c3"))
		f.setReport("FAILED", 1)
		ready := ucShape(f, "Add calc.is_even(n).")
		m := map[string]any{"ready": ready}
		out["s4"] = m
		before := ucRequests(f)
		m["exists"] = ucShape(f, "Add calc.is_even(n).")
		m["existsRequests"] = ucRequests(f) - before
		m["card"] = ucApprove(f, "")
		m["approval"] = ucApprove(f, ucHash(t, f))
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("UNCHECKED_RESULT"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}
`;

let result;
function journey(){
 if(result)return result;
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'unchecked-draft-'));
 try {
  const tree=path.join(root,'tree');
  fs.cpSync(process.cwd(),tree,{recursive:true,filter:p=>!['.git','node_modules'].includes(path.basename(p))});
  fs.writeFileSync(path.join(tree,'shape','unchecked_draft_acceptance_test.go'),GO);
  const binary=path.join(root,'unchecked.test');
  const build=cp.spawnSync('go',['test','-c','-o',binary,'./shape'],{cwd:tree,encoding:'utf8',timeout:100000});
  assert.equal(build.status,0,'compile the injected TestUncheckedDraftJourney: '+build.stdout+build.stderr);
  const report=path.join(root,'result.json');
  const run=cp.spawnSync(binary,['-test.run=^TestUncheckedDraftJourney$','-test.v','-test.timeout=100s'],{cwd:path.join(tree,'shape'),env:{...process.env,UNCHECKED_RESULT:report},encoding:'utf8',timeout:110000});
  assert.equal(run.status,0,'TestUncheckedDraftJourney via built test binary: '+run.stdout+run.stderr);
  assert.ok(fs.existsSync(report),'TestUncheckedDraftJourney must write its observed report');
  result=JSON.parse(fs.readFileSync(report,'utf8'));return result;
 } finally { fs.rmSync(root,{recursive:true,force:true}); }
}
function text(r){return [r.Detail,...(r.Lines||[]),r.Output].join('\n');}
function refuses(label,x){
 assert.equal(x.Class,'intent',label+': '+text(x));
 assert.equal(x.Reason,'not_ready',label+': '+text(x));
 assert.equal(x.Exit,1,label+': '+text(x));
 assert.match(x.Detail,/no passed Shape checks/i,label+': '+text(x));
 assert.ok(text(x).includes('Retry Shape on the retained draft, or delete .kogen/intents/is-even/ and shape again.'),label+': '+text(x));
 assert.ok(text(x).includes('Next: kogen intent shape is-even <request-file>'),label+': '+text(x));
 assert.ok(!x.Output.includes('Approve with:'),label+' printed a card: '+x.Output);
 assert.equal(x.Marker,false,label+' wrote an approval marker');
}
Then('approve-refuses-unchecked-shape-draft: an interrupted Shape leaves a not-ready draft that approve refuses and Shape retries',function(){
 const r=journey().s1;
 assert.equal(r.interrupted,true,'Shape returns the SIGINT interrupt');
 assert.equal(r.interruptExit,130);
 assert.equal(r.intentExists,true,'precondition: intent.md is on disk after the interrupt');
 assert.equal(r.featureExists,true,'precondition: acceptance.feature is on disk after the interrupt');
 assert.equal(r.latchExists,false,'an interrupted Shape writes no not-ready.json');
 refuses('approve without a hash',r.card);
 refuses('approve with the current package hash',r.hashApprove);
 assert.notEqual(r.retry.Reason,'exists','a second Shape must not answer intent/exists: '+text(r.retry));
 assert.ok(r.retryRequests>=1,'a second Shape sends at least one shaper request on the retained draft, sent '+r.retryRequests);
});
Then('approve-refuses-unchecked-shape-draft: a draft edited after its checks passed is not approvable',function(){
 const r=journey().s2;
 assert.equal(r.ready.Exit,0,'precondition: Shape reaches ready: '+text(r.ready));
 assert.equal(r.featureExists,true,'precondition: acceptance.feature is on disk');
 assert.equal(r.card.Exit,5,'the unedited ready draft gets its card: '+text(r.card));
 refuses('approve with the edited package hash',r.editedHash);
 assert.equal(r.restoredCard.Exit,5,'restoring the file restores the card: '+text(r.restoredCard));
 assert.equal(r.restoredCard.Reason,'');
});
Then('approve-refuses-unchecked-shape-draft: checks recorded for a different request do not count',function(){
 const r=journey().s3;
 assert.equal(r.first.Exit,0,'precondition: the first request reaches ready: '+text(r.first));
 assert.equal(r.interrupted,true,'the second Shape is interrupted');
 assert.equal(r.featureExists,true,'precondition: the interrupted draft is on disk');
 assert.equal(r.latchExists,false);
 assert.equal(r.sameHash,true,'precondition: the draft package hash equals the earlier passed one');
 refuses('approve with the identical package hash',r.hashApprove);
});
Then('approve-refuses-unchecked-shape-draft: a ready draft keeps its card, approval and exists answer',function(){
 const r=journey().s4;
 assert.equal(r.ready.Exit,0,'precondition: Shape reaches ready: '+text(r.ready));
 assert.equal(r.exists.Class,'intent');assert.equal(r.exists.Reason,'exists');assert.equal(r.exists.Exit,5);
 assert.ok(r.exists.Detail.includes('is-even is already shaped'),text(r.exists));
 assert.equal(r.existsRequests,0,'the exists answer sends no model request');
 assert.equal(r.card.Exit,5,text(r.card));assert.equal(r.card.Reason,'');
 assert.ok(r.card.Output.includes('Approve with:'),r.card.Output);
 assert.equal(r.card.Marker,false);
 assert.equal(r.approval.Exit,0,text(r.approval));assert.equal(r.approval.Marker,true);
});
