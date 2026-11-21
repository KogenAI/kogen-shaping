package shape

import (
	"strings"
	"testing"
)

// A first reference failure is no repeated failure and never ends the shaping, whatever was
// refused before; a second one ends it.
func TestRetryBudgetReferenceFailureGetsOneExtraRound(t *testing.T) {
	b := retryBudget{limit: submissionRetryLimit}
	set := []gateProblems{{gate: contractGate, problems: []string{"a"}}}
	for i := 0; i < 2*submissionRetryLimit-1; i++ {
		b.refuse("readiness")
	}
	if b.refuseReference(set, false) {
		t.Fatal("the first reference failure ended the shaping")
	}
	if b.consecutive != 1 {
		t.Errorf("consecutive = %d, want 1: a first reference failure is not a repeat", b.consecutive)
	}
	if !b.refuseReference(set, true) {
		t.Error("the second reference failure did not end the shaping")
	}
}

func TestReferenceFailureLinesNameCommandExitCodeAndAssertion(t *testing.T) {
	scenarios, feature := controlsParseFixture(t)
	c := controlsFailed(4, "AssertionError", "new range is not implemented\n+ actual - expected", "")
	props := []controlsProp{{report: propertyReport{property: "add", kind: "kept", request: "add"}, scenarios: []int{0}}}
	run := controlsRunResult{complete: true, cases: map[int]*controlsCase{scenarios[0].Line: c}, accept: "sh accept.sh", acceptStatus: 1}
	got := strings.ToLower(strings.Join(referenceFailureLines(props, scenarios, feature, run), "\n"))
	for _, want := range []string{"`sh accept.sh`", "exit code 1", "new range is not implemented", "reference", "at fault"} {
		if !strings.Contains(got, want) {
			t.Errorf("lines lack %q: %s", want, got)
		}
	}
	incomplete := controlsRunResult{accept: "sh accept.sh", acceptStatus: 2, acceptOutput: "SyntaxError: boom\n"}
	got = strings.Join(referenceFailureLines(props, scenarios, feature, incomplete), "\n")
	if !strings.Contains(got, "exit code 2") || !strings.Contains(got, "SyntaxError: boom") {
		t.Errorf("incomplete run lines = %s", got)
	}
	passed := controlsRunResult{complete: true, cases: map[int]*controlsCase{scenarios[0].Line: &controlsCase{steps: []controlsStep{{line: 3, status: "PASSED"}}}}, accept: "sh accept.sh"}
	if lines := referenceFailureLines(props, scenarios, feature, passed); lines != nil {
		t.Errorf("a passing reference got lines %q", lines)
	}
}
