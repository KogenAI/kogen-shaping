package shape

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

func TestAuditPayloadKeysAbsentAreEmptyAndMalformedRefused(t *testing.T) {
	parallel(t)
	rest, audit, why := auditPayload(decodeReviewJSON(t, `{"findings":[],"environment":[],"properties":[]}`))
	if why != "" || len(audit.invented)+len(audit.guarantees)+len(audit.modes) != 0 || len(rest) != 3 {
		t.Fatalf("absent keys: %q %+v %v", why, audit, rest)
	}
	_, audit, why = auditPayload(decodeReviewJSON(t, `{"findings":[],"environment":[],"invented":[{"scenario":"s","literal":"/a","why":"w"}],"guarantees":[{"property":"p","request":"r","needs_interrupted":true}],"failure_modes":[{"property":"p","direction":"interrupted","wrong_implementation":"w","scenario":"s","assertion":"a"}]}`))
	if why != "" || len(audit.invented) != 1 || len(audit.guarantees) != 1 || len(audit.modes) != 1 {
		t.Fatalf("complete: %q %+v", why, audit)
	}
	for _, bad := range []string{
		`{"invented":[{"scenario":"s","literal":"/a"}]}`,
		`{"invented":"x"}`,
		`{"guarantees":[{"property":"p","request":"r"}]}`,
		`{"failure_modes":[{"property":"p","direction":"sideways","wrong_implementation":"w","scenario":"s","assertion":"a"}]}`,
	} {
		if _, _, why := auditPayload(decodeReviewJSON(t, bad)); why == "" {
			t.Errorf("%s: accepted", bad)
		}
	}
}

func TestAuditFailureLinesCountOnlyFailingScenariosWithThenAssertions(t *testing.T) {
	parallel(t)
	feature := "Feature: f\n  Scenario: Red\n    Given a thing\n    Then it is once\n    And it is not twice\n  Scenario: Green\n    When it runs\n    Then it is once\n"
	scenarios, err := project.ParseGherkin(feature)
	if err != nil {
		t.Fatal(err)
	}
	ev := contractEvidence{request: "Deliver it exactly once.", failures: map[string][]string{"Red": {"boom"}}}
	g := []auditGuarantee{{"once", "deliver it exactly once", false}}
	mode := func(direction, scenario, assertion string) auditFailureMode {
		return auditFailureMode{"Once", direction, "wrong", scenario, assertion}
	}
	sh := &shaping{}
	lines, fixes := sh.auditFailureLines(ev, scenarios, auditReport{guarantees: g, modes: []auditFailureMode{
		mode(directionTooMuch, "Red", "it is not twice"), mode(directionTooLittle, "Green", "it is once"),
	}}, "")
	if len(lines) != 1 || len(fixes) != 1 || !strings.Contains(lines[0], "direction: too_little") || !strings.Contains(lines[0], "Green") || !strings.Contains(lines[0], "unchanged tree") {
		t.Fatalf("lines %q", lines)
	}
	lines, _ = sh.auditFailureLines(ev, scenarios, auditReport{guarantees: g, modes: []auditFailureMode{
		mode(directionTooMuch, "Red", "it is not twice"), mode(directionTooLittle, "Red", "Given a thing"),
	}}, "")
	if len(lines) != 1 || !strings.Contains(lines[0], "not the text of a Then step") {
		t.Fatalf("given step counted: %q", lines)
	}
	lines, _ = sh.auditFailureLines(ev, scenarios, auditReport{guarantees: g, modes: []auditFailureMode{
		mode(directionTooMuch, "Red", "it is once"), mode(directionTooLittle, "Red", "it is not twice"),
	}}, "")
	if len(lines) != 0 {
		t.Fatalf("complete modes refused: %q", lines)
	}
	lines, _ = sh.auditFailureLines(ev, scenarios, auditReport{guarantees: []auditGuarantee{{"once", "deliver it twice", false}}}, "")
	if len(lines) != 0 {
		t.Fatalf("a guarantee the request does not contain counted: %q", lines)
	}
}
