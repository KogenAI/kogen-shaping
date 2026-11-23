'use strict';
// The Go source of the acceptance harness of shape-escalate-repeated-guarantee-conflict, which
// shape-escalate-repeated-guarantee-conflict__steps.js compiles into the shape package's tests
// through a Go overlay as shape/kseg_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace.
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

const ksegTool = "submit_decision_check"

const (
	ksegRequest = "Deliver each webhook event to the receiver over HTTP exactly once."
	ksegQuote   = "Deliver each webhook event to the receiver over HTTP exactly once."
	ksegFirst   = "- After a crash during the HTTP request the sender does not retry, so delivery is at most once (source: request)"
	ksegSecond  = "- After a crash during the HTTP request the delivery is never resent, so delivery is at most once (source: request)"
	ksegThird   = "- After a crash during the HTTP request the delivery is retried, so delivery is at least once (source: request)"
	ksegFixed   = "- A crash during the HTTP request is covered by an idempotency key, so each event is neither lost nor repeated (source: request)"
)

func ksegItem(statement, verdict, intentValue, requestValue string) map[string]any {
	return map[string]any{"statement": statement, "request_quotes": []any{ksegQuote}, "verdict": verdict,
		"intent_value": intentValue, "request_value": requestValue, "difference": "the draft weakens the guarantee", "why": "a crash leaves the delivery unconfirmed"}
}

// ksegCheckCall says whether a request is one of Kogen's own check calls (decision, literal,
// example and the like), which offer a submit_<name>_check tool and are not shaper requests.
func ksegCheckCall(r map[string]any) bool {
	for _, n := range shpToolNames(r) {
		if strings.HasPrefix(n, "submit_") && strings.HasSuffix(n, "_check") {
			return true
		}
	}
	return false
}

type ksegCase struct {
	lines  []string
	checks [][]map[string]any
}

var ksegCases = map[string]ksegCase{
	"webhook-twice": {lines: []string{ksegFirst, ksegSecond, ksegThird},
		checks: [][]map[string]any{
			{ksegItem(ksegFirst, "contradicts", "at most once", "exactly once")},
			{ksegItem(ksegSecond, "contradicts", "at most once", "exactly once")},
			{ksegItem(ksegThird, "contradicts", "at least once", "exactly once")}}},
	"webhook-resolved": {lines: []string{ksegFirst, ksegFixed},
		checks: [][]map[string]any{
			{ksegItem(ksegFirst, "contradicts", "at most once", "exactly once")},
			{ksegItem(ksegFixed, "follows", "", "")}}},
}

func TestKSegAcceptance(t *testing.T) {
	name, out := os.Getenv("KSEG_CASE"), os.Getenv("KSEG_OUT")
	c, ok := ksegCases[name]
	if !ok || out == "" {
		t.Fatalf("shape-escalate-repeated-guarantee-conflict: KSEG_CASE %q is not a case of the harness, or KSEG_OUT is empty", name)
	}
	files := func(line string) map[string]any {
		intent := strings.Replace(shpScopedIntentMD, "- Non-integers raise TypeError. (source: request)", line, 1)
		return map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
			map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
		}}
	}
	var steps []shpStep
	for i, line := range c.lines {
		steps = append(steps, shpSubmit(fmt.Sprintf("c%d", i+1), files(line)))
	}
	steps = append(steps, shpAck("ack"))
	f := newShpFx(t, steps...)
	f.setReport("PASSED", 0)
	var mu sync.Mutex
	answered := 0
	f.Prov.route = func(req map[string]any) *shpStep {
		if !slices.Contains(shpToolNames(req), ksegTool) {
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
		st := shpAnswer(shpCall(fmt.Sprintf("decision-%d", n+1), ksegTool, map[string]any{"items": items}))
		return &st
	}
	exit, err := f.shape(ksegRequest)
	seen := map[string]any{"case": name, "exit": exit, "err": fmt.Sprint(err), "stdout": f.Stdout.String(), "stderr": f.Stderr.String()}
	seen["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	shaper, decision := 0, 0
	for _, r := range f.Prov.allRequests() {
		switch {
		case slices.Contains(shpToolNames(r), ksegTool):
			decision++
		case ksegCheckCall(r):
		case contractKind(r) != "":
		default:
			shaper++
		}
	}
	seen["shaper"], seen["decision"] = shaper, decision
	seen["intent_exists"] = f.exists(".kogen/intents/is-even/intent.md")
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
		t.Fatalf("shape-escalate-repeated-guarantee-conflict: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
