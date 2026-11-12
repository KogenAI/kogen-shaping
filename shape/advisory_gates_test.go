package shape

import (
	"strings"
	"testing"
)

const advisoryRequest = "Add `mix todo.list --open`."

// Unsupported but carrying evidence: advisory, recorded and passed on as a note.
func TestDecisionResolveEvidencedUnsupportedIsAdvisory(t *testing.T) {
	parallel(t)
	cases := []struct {
		name, line string
		quotes     []string
		locked     bool
		why        string
	}{
		{"attempt 4 request-quoted", "- `--open` is parsed with a strict boolean switch on the existing `Mix.Tasks.Todo.List`; no other options are added. (evidence: lib/mix/tasks/todo/list.ex) (source: request)",
			[]string{advisoryRequest}, false, "Request names the --open flag only; the strict-switch parsing and 'no other options' are new choices."},
		{"attempt 3 locked, request silent", "- Acceptance uses its own complete-task-prefixed steps with a per-scenario temporary TODO_FILE through the real Mix CLI. (decision: D6, round 1) (evidence: features/steps/todo_steps.js; README.md) (source: request)",
			nil, true, "The request is silent on acceptance step structure."},
	}
	for _, c := range cases {
		intent := "## Decisions\n" + c.line + "\n"
		locked := map[string]bool{}
		if c.locked {
			locked[decisionLineKey(c.line)] = true
		}
		item := decisionItem{statement: c.line, quotes: c.quotes, verdict: "unsupported", why: c.why}
		event, lines, marks, notes := decisionResolveNotes([]decisionItem{item}, advisoryRequest, intent, locked)
		if len(lines) != 0 || len(marks) != 0 || len(notes) != 1 || !strings.Contains(notes[0], c.why) {
			t.Fatalf("%s: want one advisory note, got lines %q marks %v notes %q", c.name, lines, marks, notes)
		}
		if event["items"].([]any)[0].(map[string]any)["advisory"] != true {
			t.Fatalf("%s: advisory not recorded: %v", c.name, event)
		}
		if _, lines := decisionOutcome([]decisionItem{item}, advisoryRequest, intent); len(lines) != 0 {
			t.Fatalf("%s: refused: %q", c.name, lines)
		}
	}
}

func TestDecisionResolveStillRefusesWithoutEvidenceOrAssumption(t *testing.T) {
	parallel(t)
	const line = "- `--open` is parsed with a strict boolean switch. (source: request)"
	item := decisionItem{statement: line, quotes: []string{advisoryRequest}, verdict: "unsupported", why: "new choice"}
	_, lines, _, notes := decisionResolveNotes([]decisionItem{item}, advisoryRequest, "## Decisions\n"+line+"\n", nil)
	if len(lines) == 0 || len(notes) != 0 {
		t.Fatalf("bare unsupported line accepted: %q %q", lines, notes)
	}
}

func TestDecisionResolveEvidencedContradictionStillRefused(t *testing.T) {
	parallel(t)
	const request = "The saved login file may be unreadable; print `saved login cannot be read` and exit 4."
	const line = "- An unreadable login exits 0. (evidence: cli/main.go) (source: request)"
	item := decisionItem{statement: line, quotes: []string{"exit 4"}, verdict: "contradicts",
		intentValue: "exits 0", requestValue: "exit 4", difference: "exit code", why: "request says 4"}
	_, lines, _, notes := decisionResolveNotes([]decisionItem{item}, request, "## Decisions\n"+line+"\n", nil)
	if len(notes) != 0 || !strings.Contains(strings.Join(lines, "\n"), "verdict: contradicts") {
		t.Fatalf("contradiction not refused: %q %q", lines, notes)
	}
}

// A recognisable assumption in a wrong place or wording is rewritten into the canonical form.
func TestNormalizeAssumptionForm(t *testing.T) {
	parallel(t)
	for in, want := range map[string]string{
		"- Use X. (source: request) (assumption: confirm X)":                 "- Use X. (assumption: confirm X) (source: request)",
		"- Use X. (Assumption: confirm X) (source: request)":                 "- Use X. (assumption: confirm X) (source: request)",
		"- Use X. (assumed: confirm X) (source: request)":                    "- Use X. (assumption: confirm X) (source: request)",
		"- Use X. [assumption: confirm X] (source: answer 2)":                "- Use X. (assumption: confirm X) (source: answer 2)",
		"- Use X. (assumption - confirm X) (source: request)":                "- Use X. (assumption: confirm X) (source: request)",
		"- Use X. (assumption: confirm X) (source: request)":                 "",
		"- Use X. (evidence: a.go) (source: request)":                        "",
		"- Use X. (assumption: confirm X) (assumption: y) (source: request)": "",
	} {
		got, ok := normalizeAssumptionForm(in)
		if (want == "") == ok || got != want && ok {
			t.Fatalf("%q: got %q %v, want %q", in, got, ok, want)
		}
	}
}

func TestShapeNormalizesAssumptionOnDisk(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	sh := f.shaping()
	const line = "- Use X. (source: request) (assumption: confirm X)"
	intent := "# T\n\n## Decisions\n" + line + "\n"
	f.write(".kogen/intents/is-even/intent.md", intent)
	got, marks := sh.normalizedAssumptions(intent)
	if !strings.Contains(got, "- Use X. (assumption: confirm X) (source: request)") || len(marks) != 1 {
		t.Fatalf("not normalised: %q %v", got, marks)
	}
	sh.pendingMarks = marks
	if err := sh.applyAssumptionMarks(); err != nil {
		t.Fatal(err)
	}
	shpContains(t, "intent", f.read(".kogen/intents/is-even/intent.md"), "- Use X. (assumption: confirm X) (source: request)")
}

// Attempt 5: a revision of D3 without why or evidence keeps D3 and is a note, not a refusal.
func TestDecisionRecordRevisionWithoutWhyKeepsEarlier(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	sh := f.shaping()
	d3 := recordedDecision{ID: "D3", Round: 1, Decision: "`--open` parsed as a strict boolean switch on Mix.Tasks.Todo.List", Uncertainty: "u"}
	sh.stateDir = t.TempDir()
	sh.ledger = &decisionLedger{Decisions: []recordedDecision{d3}}
	text, _, refused, err := sh.decisionRecordGate([]sourcedDecision{{decision: "`--open` is a strict boolean switch", uncertainty: "u", revises: "D3"}})
	if err != nil || refused {
		t.Fatalf("refused: %v %v %q", refused, err, text)
	}
	if len(sh.ledger.Decisions) != 1 || !sh.ledger.Decisions[0].stands() {
		t.Fatalf("D3 does not stand: %+v", sh.ledger.Decisions)
	}
	if len(sh.notes) != 1 || !strings.Contains(sh.notes[0], "D3 stands") {
		t.Fatalf("no note: %q", sh.notes)
	}
	if len(shpNamed(f.events(), "decision_note")) != 1 {
		t.Fatalf("note not recorded: %v", f.events())
	}
	// A revision of a decision that is not recorded is still refused.
	_, _, refused, err = sh.decisionRecordGate([]sourcedDecision{{decision: "x", uncertainty: "u", revises: "D9", why: "w"}})
	if err != nil || !refused {
		t.Fatalf("revising D9 accepted: %v %v", refused, err)
	}
}
