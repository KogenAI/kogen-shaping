//go:build ignore

package shape

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// A scripted checker qualifies the local gate and its evidence transport, not
// the reasoning of a live provider. Acceptance separately checks the policy
// delivered to that checker.
func TestKeepUnchangedAcceptance(t *testing.T) {
	var c struct {
		Statement, Citation, Evidence, Verdict string
		Probe                                  bool
	}
	if err := json.Unmarshal([]byte(os.Getenv("KEEP_CASE")), &c); err != nil {
		t.Fatal(err)
	}
	line := "- " + c.Statement
	if c.Citation != "" {
		line += " (evidence: " + c.Citation + ")"
	}
	line += " (source: request)"
	f := newShpFx(t)
	f.Prov.route = func(req map[string]any) *shpStep {
		if !decisionRequest(req) {
			return nil
		}
		answer := shpAnswer(shpCall("d", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": line, "request_quotes": []any{}, "verdict": c.Verdict, "intent_value": "", "request_value": "", "difference": "", "why": "scripted policy result"}}}))
		return &answer
	}

	f.write("limits.go", "package limits\nconst controlsMaxRuns = 8\nconst controlsMaxFiles = 8\nconst controlsMaxBytes = 65536\nconst controlsTimeCap = 300\n")
	f.git("add", "limits.go")
	f.git("commit", "-qm", "Seed current limits")
	f.newSession()
	sh := f.shaping()
	sh.role = sh.cfg.Models["shape"]
	sh.request = "Improve controls; keep the 300 s cap."
	sh.readPaths = map[string]bool{"limits.go": true}
	if c.Probe {
		sh.probed = map[string]bool{"observe helpers": true}
		sh.probeTexts = map[string]string{"observe helpers": "helpers directory: features/steps\n"}
	}
	evidenceProblems, err := sh.evidenceProblems("## Decisions\n" + line + "\n")
	if err != nil {
		t.Fatal(err)
	}
	problems, err := sh.decisionCheck(context.Background(), "## Decisions\n"+line+"\n")
	if err != nil {
		t.Fatal(err)
	}
	requests := f.Prov.decisionRequests()
	if len(requests) != 1 {
		t.Fatalf("TestKeepUnchangedAcceptance: expected one checker request, got %d", len(requests))
	}
	var instructions string
	instructions, _ = requests[0]["instructions"].(string)
	// qcText includes the input and instructions and preserves the actual request.
	input := qcText(requests[0])
	var event map[string]any
	for _, e := range f.events() {
		if e["event"] == "decision_check" {
			event = e
		}
	}
	report := map[string]any{"statement": line, "instructions": instructions, "input": input, "problems": problems, "evidenceProblems": evidenceProblems, "event": event, "request": sh.request}
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(os.Getenv("KEEP_OUT")) == "" {
		t.Fatal("KEEP_OUT missing")
	}
	if err := os.WriteFile(os.Getenv("KEEP_OUT"), b, 0600); err != nil {
		t.Fatal(err)
	}
}
