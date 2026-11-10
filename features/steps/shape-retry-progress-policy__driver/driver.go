//go:build ignore

package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// This test is an observation driver, not a test of the host's test runner.
func TestShapeRetryProgressObservation(t *testing.T) {
	var out any
	if os.Getenv("SRPP_MODE") == "budget" {
		b := retryBudget{limit: 3}
		trace := []string{}
		for _, gate := range strings.Split(os.Getenv("SRPP_GATES"), ",") {
			stop := b.refuse(gate)
			trace = append(trace, fmt.Sprintf("%d:%d:%t", b.consecutive, b.total, stop))
		}
		out = strings.Join(trace, ",")
	} else {
		statement := os.Getenv("SRPP_STATEMENT")
		quotes := []string{}
		if q := os.Getenv("SRPP_QUOTE"); q != "" {
			quotes = append(quotes, q)
		}
		event, feedback := decisionOutcome([]decisionItem{{statement: statement, quotes: quotes, verdict: "unsupported", why: "not established"}}, "Initialized Kogen in <path>", os.Getenv("SRPP_INTENT"))
		out = map[string]any{"event": event, "feedback": feedback}
	}
	bytes, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SRPP_OUT"), bytes, 0600); err != nil {
		t.Fatal(err)
	}
}
