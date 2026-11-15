//go:build ignore

package shape

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// A scripted checker that always answers unsupported qualifies the local gate, not a live model:
// only Kogen's own keep-existing rule can turn the answer into follows.
func TestKeepExistingAcceptance(t *testing.T) {
	var c struct {
		Statement, Citation, Expect, Why string
		Quotes                           []string
	}
	if err := json.Unmarshal([]byte(os.Getenv("KEEP_CASE")), &c); err != nil {
		t.Fatal(err)
	}
	line := "- " + c.Statement
	if c.Citation != "" {
		line += " (evidence: " + c.Citation + ")"
	}
	line += " (source: request)"
	why := c.Why
	if why == "" {
		why = "scripted: the request does not state it"
	}
	quotes := []any{}
	for _, q := range c.Quotes {
		quotes = append(quotes, q)
	}
	f := newShpFx(t)
	f.Prov.route = func(req map[string]any) *shpStep {
		if !decisionRequest(req) {
			return nil
		}
		answer := shpAnswer(shpCall("d", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": line, "request_quotes": quotes, "verdict": "unsupported", "intent_value": "", "request_value": "", "difference": "", "why": why}}}))
		return &answer
	}
	f.write("limits.go", "package limits\nconst controlsMaxRuns = 8\nconst controlsMaxFiles = 8\nconst controlsMaxBytes = 65536\nconst controlsTimeCap = 300\n")
	f.git("add", "limits.go")
	f.git("commit", "-qm", "Seed current limits")
	f.newSession()
	sh := f.shaping()
	sh.role = sh.cfg.Models["shape"]
	sh.request = "Improve controls; keep the 300 s cap."
	sh.readPaths = map[string]bool{"limits.go": true, "missing.go": true}
	intent := "## Decisions\n" + line + "\n"
	problems, err := sh.decisionCheck(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Prov.decisionRequests()) != 1 {
		t.Fatalf("TestKeepExistingAcceptance: expected one checker request, got %d", len(f.Prov.decisionRequests()))
	}
	var event map[string]any
	for _, e := range f.events() {
		if e["event"] == "decision_check" {
			event = e
		}
	}
	b, err := json.Marshal(map[string]any{"statement": line, "problems": problems, "event": event})
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
