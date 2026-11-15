//go:build ignore

package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

type repeatedCase struct {
	Mode    string `json:"mode"`
	Request string `json:"request"`
	Rounds  []struct {
		Statement    string `json:"statement"`
		IntentValue  string `json:"intentValue"`
		RequestValue string `json:"requestValue"`
		Verdict      string `json:"verdict"`
		Submission   int    `json:"submission"`
	} `json:"rounds"`
	Why        string `json:"why"`
	Difference string `json:"difference"`
}

// This driver reports real gate results; product expectations live in Gherkin.
func TestRepeatedStatementObservation(t *testing.T) {
	var c repeatedCase
	if err := json.Unmarshal([]byte(os.Getenv("REPEATED_CASE")), &c); err != nil {
		t.Fatal(err)
	}
	f := newShpFx(t)
	answer := func(i int) shpStep {
		r := c.Rounds[i]
		return shpAnswer(shpCall(fmt.Sprintf("d%d", i), SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{
			"statement": r.Statement, "request_quotes": []any{c.Request}, "verdict": r.Verdict,
			"intent_value": r.IntentValue, "request_value": r.RequestValue, "difference": c.Difference, "why": c.Why,
		}}}))
	}
	var reports []map[string]any
	if c.Mode == "submit" {
		for i, r := range c.Rounds {
			args := shpFilesWithDecisions(r.Statement)
			files := args["files"].([]any)
			md := files[0].(map[string]any)
			md["content"] = strings.Replace(md["content"].(string), "# Add an is-even check", fmt.Sprintf("# Add an is-even check %d", i+1), 1)
			f.Prov.seq = append(f.Prov.seq, shpSubmit(fmt.Sprintf("s%d", i+1), args))
		}
		next := 0
		f.Prov.route = func(req map[string]any) *shpStep {
			if !decisionRequest(req) {
				return nil
			}
			a := answer(next)
			next++
			return &a
		}
		// Exhausting the scripted shaper stops after the two observed submissions.
		// The request carrying each tool result is retained before that honest stop.
		_, _ = f.shape(c.Request)
		outputs := map[string]string{}
		for _, req := range f.Prov.requests() {
			for k, v := range shpOutputs(req) {
				outputs[k] = v
			}
		}
		for i := range c.Rounds {
			reports = append(reports, map[string]any{"feedback": outputs[fmt.Sprintf("s%d", i+1)]})
		}
	} else {
		sh := f.shaping()
		sh.role = sh.cfg.Models["shape"]
		sh.request = c.Request
		current := 0
		f.Prov.route = func(req map[string]any) *shpStep {
			if !decisionRequest(req) {
				return nil
			}
			a := answer(current)
			return &a
		}
		for i, r := range c.Rounds {
			current = i
			sh.submissions = r.Submission
			intent := "# Candidate\n" + strings.Repeat("\n", i) + "\n## Decisions\n" + r.Statement + "\n"
			lines, err := sh.decisionCheckWithEvidence(context.Background(), intent, "")
			if err != nil {
				t.Fatal(err)
			}
			reports = append(reports, map[string]any{"feedback": strings.Join(lines, "\n"), "intent": intent})
		}
	}
	var requests []any
	for _, req := range f.Prov.decisionRequests() {
		requests = append(requests, map[string]any{"instructions": req["instructions"], "input": req["input"]})
	}
	events := shpNamed(f.events(), "decision_check")
	out, err := json.Marshal(map[string]any{"reports": reports, "requests": requests, "events": events})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("REPEATED_OUT"), out, 0600); err != nil {
		t.Fatal(err)
	}
}
