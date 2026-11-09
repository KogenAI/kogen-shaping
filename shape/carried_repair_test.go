package shape

import (
	"fmt"
	"strings"
	"testing"
)

func TestRepairCarriedDecisionDraft(t *testing.T) {
	draft := "# Docs\n\n## Decisions\n- Keep task-first documentation. (source: request)\n\n## Tests\n- none: documentation\n"
	var standing []recordedDecision
	var lines []string
	for i := 9; i <= 16; i++ {
		line := fmt.Sprintf("- Synthetic docs choice %d (decision: D%d, round 1) (source: request)", i, i)
		lines = append(lines, line)
		standing = append(standing, recordedDecision{ID: fmt.Sprintf("D%d", i), Round: 1, Decision: fmt.Sprintf("Synthetic docs choice %d", i)})
	}
	want := strings.Replace(draft, "\n\n## Tests", "\n"+strings.Join(lines, "\n")+"\n\n## Tests", 1)
	got := repairCarriedDecisionDraft(draft, standing)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if twice := repairCarriedDecisionDraft(got, standing); twice != want {
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
	standing := []recordedDecision{
		{ID: "D8", Round: 1, Decision: "Keep concise pages.", Constraints: []string{"task first"}, Evidence: []receipt{{Item: "README.md"}}},
		{ID: "D9", Round: 1, Decision: "Keep examples.", Constraints: []string{"retain commands"}, Evidence: []receipt{{Item: "README.md"}}},
	}
	want := strings.Replace(draft, existing, existing+"\n"+missing, 1)
	if got := repairCarriedDecisionDraft(draft, standing); got != want {
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

func TestCarriedRepairKeepsOneAuthorityPerDecision(t *testing.T) {
	const canonical = "- The --open flag is parsed from mix todo.list arguments; task file format stays id/status/base64 title and the app stays dependency-free plain Elixir (decision: D4, round 1) (evidence: lib/tiny_tasks/store.ex; README.md; mix.exs) (source: request)"
	const authored = "- The `--open` flag is parsed from the arguments of `mix todo.list` (e.g. with `OptionParser`); the task file format stays `<id>\\t<status>\\t<base64 title>` and the dependency-free plain-Elixir convention is kept"
	const marker = " (decision: D4, round 2)"
	const evidence = " (evidence: lib/tiny_tasks/store.ex; README.md; mix.exs) (source: request)"
	const otherSection = "\n\n## Tests\n" + authored + marker + evidence + "\n"
	draft := "# Tiny Tasks\n\n## Decisions\n" + authored + marker + evidence + otherSection
	standing := []recordedDecision{{
		ID: "D4", Round: 1,
		Decision: "The --open flag is parsed from mix todo.list arguments; task file format stays id/status/base64 title and the app stays dependency-free plain Elixir",
		Evidence: []receipt{{Item: "lib/tiny_tasks/store.ex"}, {Item: "README.md"}, {Item: "mix.exs"}},
	}}
	got := repairCarriedDecisionDraft(draft, standing)
	want := "# Tiny Tasks\n\n## Decisions\n" + authored + evidence + "\n" + canonical + otherSection
	if got != want || repairCarriedDecisionDraft(got, standing) != want {
		t.Fatalf("repair did not retain one ledger identity and all draft content:\n%s", got)
	}
	request := "Add a dependency for command-line parsing."
	_, problems := decisionOutcome([]decisionItem{{
		statement: authored + evidence, quotes: []string{request}, verdict: "contradicts",
		intentValue: "dependency-free", requestValue: "Add a dependency",
	}}, request, got)
	if !strings.Contains(strings.Join(problems, "\n"), "verdict: contradicts") {
		t.Fatalf("repair hid a real contradiction: %v", problems)
	}
}
