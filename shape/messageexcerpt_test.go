package shape

import (
	"fmt"
	"strings"
	"testing"
)

func TestFailedStepDetailKeepsFailureAtEnd(t *testing.T) {
	var b []string
	b = append(b, "AssertionError: compiled regression:", "=== RUN   TestWidget")
	for i := 0; i < 300; i++ {
		b = append(b, fmt.Sprintf("    log line %d", i))
	}
	b = append(b, "    widget_test.go:77: want 1 got 2", "--- FAIL: TestWidget (0.00s)")
	got := failureDetail("acceptance.feature line 3: Then it works", strings.Join(b, "\n"))
	for _, want := range []string{"AssertionError", "widget_test.go:77", "--- FAIL: TestWidget", "lines omitted"} {
		if !strings.Contains(got, want) {
			t.Errorf("detail lost %q", want)
		}
	}
	if n := len(strings.Split(got, "\n")); n > 204 {
		t.Errorf("detail has %d lines, want a bounded excerpt", n)
	}
}
