package shape

import (
	"strings"
	"testing"
)

func TestRequestGroundedUnsupportedFeedback(t *testing.T) {
	for _, tc := range []struct{ draft, where string }{{"heading\nReport the initialized directory.", "intent.md line 2"}, {"heading", "the quoted statement"}} {
		_, lines := decisionOutcome([]decisionItem{{statement: "Report the initialized directory.", verdict: "unsupported", quotes: []string{"Initialized Kogen in <path>"}}}, "Initialized Kogen in <path>", tc.draft)
		text := strings.Join(lines, "\n")
		fix := "fix for " + tc.where + ": the request speaks to this in the quoted passages; make the statement follow them or remove it; an assumption does not override request text."
		if !strings.Contains(text, fix) || strings.Contains(text, assumptionRule) {
			t.Fatalf("feedback: %s", text)
		}
	}
	_, lines := decisionOutcome([]decisionItem{{statement: "silent", verdict: "unsupported"}, {statement: "grounded", verdict: "unsupported", quotes: []string{"Initialized Kogen in <path>"}}}, "Initialized Kogen in <path>", "silent\ngrounded")
	text := strings.Join(lines, "\n")
	if strings.Contains(text, assumptionRule) || !strings.Contains(text, "fix for intent.md line 2:") {
		t.Fatal(text)
	}
}

func TestRetryBudgetProgress(t *testing.T) {
	for _, tc := range []struct {
		gates []string
		want  []int
		stop  int
	}{
		{[]string{"a", "b", "c"}, []int{1, 1, 1}, -1},
		{[]string{"a", "a", "a"}, []int{1, 2, 3}, 2},
		{[]string{"a", "b", "a", "b", "a", "b"}, []int{1, 1, 1, 1, 1, 1}, 5},
	} {
		b := retryBudget{limit: 3}
		for i, g := range tc.gates {
			stop := b.refuse(g)
			if b.consecutive != tc.want[i] || b.total != i+1 || stop != (i == tc.stop) {
				t.Fatalf("%v step %d: %+v stop %v", tc.gates, i, b, stop)
			}
		}
	}
}
