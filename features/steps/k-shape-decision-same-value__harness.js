'use strict';
// The Go source of the k-shape-decision-same-value acceptance harness, which
// k-shape-decision-same-value__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_decision_same_value_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// The acceptance harness of k-shape-decision-same-value: one native Shape per synthetic request
// and Intent pair, with a scripted shaper and a scripted decision-check model. KDSV_CASE names the
// case; what a user and the shaper see of the run is written as JSON to KDSV_OUT for the
// acceptance steps to judge. On a tree without a decision check no request offers
// submit_decision_check and the script is never used.

const kdsvTool = "submit_decision_check"

type kdsvCase struct {
	request string
	first   string // the Decisions line of the first submission
	second  string // the Decisions line of the corrected submission; "" when there is none
	checks  [][]map[string]any
}

func kdsvItem(statement, quote, verdict, intentValue, requestValue, difference string) map[string]any {
	return map[string]any{"statement": statement, "request_quotes": []any{quote}, "verdict": verdict,
		"intent_value": intentValue, "request_value": requestValue, "difference": difference, "why": "synthetic scripted item"}
}

const (
	kdsvUploads     = "limit uploads to 1,000 rows"
	kdsvUploads1000 = "- Upload limit is 1000 rows (source: request)"
	kdsvUploads1k   = "- Upload limit is 1,000 rows (source: request)"
	kdsvRetry       = "- Retry delay is 120 seconds (source: request)"
	kdsvDisplay     = "- Display is 0.5 as a fraction (source: request)"
	kdsvUploads10k  = "- Upload limit is 10,000 rows (source: request)"
	kdsvOldest      = "- Sort oldest first (source: request)"
	kdsvNewest      = "- Sort newest first (source: request)"
)

var kdsvCases = map[string]kdsvCase{
	"uploads-1000": {request: kdsvUploads, first: kdsvUploads1000,
		checks: [][]map[string]any{{kdsvItem(kdsvUploads1000, kdsvUploads, "follows", "", "", "")}}},
	"retry-120-seconds": {request: "retry after 2 minutes", first: kdsvRetry,
		checks: [][]map[string]any{{kdsvItem(kdsvRetry, "retry after 2 minutes", "follows", "", "", "")}}},
	"display-fraction": {request: "show 50%", first: kdsvDisplay,
		checks: [][]map[string]any{{kdsvItem(kdsvDisplay, "show 50%", "follows", "", "", "")}}},
	"uploads-10000": {request: kdsvUploads, first: kdsvUploads10k, second: kdsvUploads1k,
		checks: [][]map[string]any{
			{kdsvItem(kdsvUploads10k, kdsvUploads, "contradicts", "10,000", "1,000", "10,000 is ten times 1,000")},
			{kdsvItem(kdsvUploads1k, kdsvUploads, "follows", "", "", "")}}},
	"sort-oldest-first": {request: "sort newest first", first: kdsvOldest, second: kdsvNewest,
		checks: [][]map[string]any{
			{kdsvItem(kdsvOldest, "sort newest first", "contradicts", "oldest first", "newest first", "the order is reversed and the order matters")},
			{kdsvItem(kdsvNewest, "sort newest first", "follows", "", "", "")}}},
	"uploads-1000-scripted-contradicts": {request: kdsvUploads, first: kdsvUploads1000, second: kdsvUploads1k,
		checks: [][]map[string]any{
			{kdsvItem(kdsvUploads1000, kdsvUploads, "contradicts", "1000", "1,000", "only the digit grouping differs")},
			{kdsvItem(kdsvUploads1k, kdsvUploads, "follows", "", "", "")}}},
	"uploads-1000-contradicts-without-values": {request: kdsvUploads, first: kdsvUploads1000, second: kdsvUploads1k,
		checks: [][]map[string]any{
			{kdsvItem(kdsvUploads1000, kdsvUploads, "contradicts", "", "", "")},
			{kdsvItem(kdsvUploads1k, kdsvUploads, "follows", "", "", "")}}},
}

func TestKShapeDecisionSameValueAcceptance(t *testing.T) {
	name, out := os.Getenv("KDSV_CASE"), os.Getenv("KDSV_OUT")
	c, ok := kdsvCases[name]
	if !ok || out == "" {
		t.Fatalf("k-shape-decision-same-value: KDSV_CASE %q is not a case of the harness, or KDSV_OUT is empty", name)
	}
	files := func(line string) map[string]any {
		intent := strings.Replace(shpScopedIntentMD, "- Non-integers raise TypeError. (source: request)", line, 1)
		return map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
			map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
		}}
	}
	steps := []shpStep{shpSubmit("c1", files(c.first))}
	if c.second != "" {
		steps = append(steps, shpSubmit("c2", files(c.second)))
	}
	f := newShpFx(t, steps...)
	f.setReport("PASSED", 0)
	var mu sync.Mutex
	answered := 0
	f.Prov.route = func(req map[string]any) *shpStep {
		if !slices.Contains(shpToolNames(req), kdsvTool) {
			return nil
		}
		mu.Lock()
		n := answered
		answered++
		mu.Unlock()
		items := []any{}
		if n < len(c.checks) {
			for _, it := range c.checks[n] {
				items = append(items, it)
			}
		}
		st := shpAnswer(shpCall(fmt.Sprintf("decision-%d", n+1), kdsvTool, map[string]any{"items": items}))
		return &st
	}
	exit, err := f.shape(c.request)
	seen := map[string]any{"case": name, "request": c.request, "first": c.first, "second": c.second,
		"exit": exit, "err": fmt.Sprint(err), "stdout": f.Stdout.String(), "stderr": f.Stderr.String()}
	seen["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	seen["intent"] = ""
	if f.exists(".kogen/intents/is-even/intent.md") {
		seen["intent"] = f.read(".kogen/intents/is-even/intent.md")
	}
	shaper, decision := 0, 0
	feedback := []any{}
	fed := map[string]bool{}
	firstFeedback, body, input := "", "", ""
	for _, r := range f.Prov.allRequests() {
		switch {
		case slices.Contains(shpToolNames(r), kdsvTool):
			if decision == 0 {
				body = shpBody(r)
				input = shpBody(map[string]any{"input": r["input"]})
			}
			decision++
		case contractKind(r) != "":
		default:
			shaper++
			outs := shpOutputs(r)
			for _, id := range []string{"c1", "c2"} {
				if o, ok := outs[id]; ok && !fed[id] {
					fed[id] = true
					feedback = append(feedback, o)
					if id == "c1" {
						firstFeedback = o
					}
				}
			}
		}
	}
	seen["shaper"], seen["decision"] = shaper, decision
	seen["feedback"], seen["first_feedback"] = feedback, firstFeedback
	seen["body"], seen["input"] = body, input
	events := []any{}
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping/is-even/events.ndjson")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if json.Unmarshal([]byte(line), &ev) == nil && ev["event"] == "decision_check" {
				events = append(events, ev)
			}
		}
	}
	seen["events"] = events
	b, err := json.MarshalIndent(seen, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("k-shape-decision-same-value: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
