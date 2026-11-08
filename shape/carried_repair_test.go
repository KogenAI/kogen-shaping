package shape

import (
	"fmt"
	"strings"
	"testing"
)

func TestRepairCarriedDecisionDraft(t *testing.T) {
	draft := "# Docs\n\n## Decisions\n- Keep task-first documentation. (source: request)\n\n## Tests\n- none: documentation\n"
	var problems, lines []string
	for i := 9; i <= 16; i++ {
		line := fmt.Sprintf("- Synthetic docs choice %d (decision: D%d, round 1) (source: request)", i, i)
		lines = append(lines, line)
		problems = append(problems, fmt.Sprintf("- D%d (round 1) is not in Decisions; keep it as: %s", i, line))
	}
	want := strings.Replace(draft, "\n\n## Tests", "\n"+strings.Join(lines, "\n")+"\n\n## Tests", 1)
	got := repairCarriedDecisionDraft(draft, problems)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if twice := repairCarriedDecisionDraft(got, problems); twice != want {
		t.Fatalf("second repair: %q", twice)
	}
}

func TestShapeRepairsRepeatedCarriedOmission(t *testing.T) {
	parallel(t)
	d := map[string]any{"decision": "Safe diagnostics", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}}
	line := "- Safe diagnostics (decision: D1, round 1) (constraint: no bytes) (evidence: calc.js) (source: request)"
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js")), shpSubmit("a1", dlAsk("Which wording?", d)), shpUI(1),
		shpSubmit("b1", shpFilesWithDecisions("- Safe diagnostics (source: request)")),
		shpSubmit("b2", shpFilesWithDecisions("- Safe diagnostics (source: request)")), shpAck("b3"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("first: exit %d, err %v", exit, err)
	}
	if exit, err := f.shape("1: a"); err != nil || exit != 0 {
		t.Fatalf("revision: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	published := f.read(".kogen/intents/is-even/intent.md")
	if !strings.Contains(published, line) || !strings.Contains(published, "- Safe diagnostics (source: request)") {
		t.Fatalf("published draft lost decisions: %s", published)
	}
	refused := shpNamed(f.events(), "submission_refused")
	if len(refused) != 1 || refused[0]["gate"] != "carried_decisions" {
		t.Fatalf("refusals: %v", refused)
	}
}

func TestRepairCarriedDecisionDraftPreservesMetadata(t *testing.T) {
	existing := "- Keep concise pages. (decision: D8, round 1) (constraint: task first) (evidence: README.md) (source: request)"
	missing := "- Keep examples. (decision: D9, round 1) (constraint: retain commands) (evidence: README.md) (source: request)"
	draft := "# Docs\n\n## Decisions\n" + existing + "\n\n## Tests\n- none: documentation\n"
	problems := []string{"- D8 (round 1) is not in Decisions; keep it as: " + existing, "- D9 (round 1) is not in Decisions; keep it as: " + missing}
	want := strings.Replace(draft, existing, existing+"\n"+missing, 1)
	if got := repairCarriedDecisionDraft(draft, problems); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCarriedRepairIncludesNewDecisions(t *testing.T) {
	parallel(t)
	d := map[string]any{"decision": "Safe diagnostics", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}}
	line := "- Safe diagnostics (decision: D1, round 1) (constraint: no bytes) (evidence: calc.js) (source: request)"
	next := shpFilesWithDecisions("- Safe diagnostics (source: request)")
	next["decisions"] = []any{map[string]any{"decision": "Keep token routes", "uncertainty": "retain existing routes", "constraints": []any{"stdin only"}}}
	f := newShpFx(t,
		shpAnswer(dlRead("calc.js")), shpSubmit("a1", dlAsk("Which wording?", d)), shpUI(1),
		shpSubmit("b1", shpFilesWithDecisions("- Safe diagnostics (source: request)")),
		shpSubmit("b2", next), shpAck("b3"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 5 {
		t.Fatalf("first: exit %d, err %v", exit, err)
	}
	if exit, err := f.shape("1: a"); err != nil || exit != 0 {
		t.Fatalf("revision: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	published := f.read(".kogen/intents/is-even/intent.md")
	if !strings.Contains(published, line) || !strings.Contains(published, "- Safe diagnostics (source: request)") {
		t.Fatalf("published draft lost decisions: %s", published)
	}
	if !strings.Contains(published, "- Keep token routes (decision: D2, round 2) (constraint: stdin only) (assumption: retain existing routes) (source: request)") {
		t.Fatalf("new recorded decision lost: %s", published)
	}
	refused := shpNamed(f.events(), "submission_refused")
	if len(refused) != 1 || refused[0]["gate"] != "carried_decisions" {
		t.Fatalf("refusals: %v", refused)
	}
}
