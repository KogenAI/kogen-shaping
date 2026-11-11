package shape

import (
	"strings"
	"testing"
)

// An unmarked decision the request is silent on is an assumption: Kogen marks it itself.
func TestDecisionResolveMarksUnmarkedAssumption(t *testing.T) {
	parallel(t)
	const line = "- Acceptance builds the real program from `cli` and runs `provider list` with a private HOME (decision: D5, round 1) (evidence: cli/native.go; cli/main.go) (source: request)"
	intent := "## Decisions\n" + line + "\n"
	item := decisionItem{statement: line, verdict: "unsupported",
		why: "The user must confirm how acceptance runs (build the binary with a private HOME)."}
	event, lines, marks := decisionResolve([]decisionItem{item}, "Fix the unreadable login message.", intent, nil)
	if len(lines) != 0 || len(marks) != 1 {
		t.Fatalf("want a rewrite and no refusal, got lines %q marks %v", lines, marks)
	}
	want := "- Acceptance builds the real program from `cli` and runs `provider list` with a private HOME (decision: D5, round 1) (evidence: cli/native.go; cli/main.go) (assumption: The user must confirm how acceptance runs [build the binary with a private HOME].) (source: request)"
	if marks[0].marked != want {
		t.Fatalf("marked line = %q", marks[0].marked)
	}
	if event["items"].([]any)[0].(map[string]any)["marked_assumption"] == nil {
		t.Fatalf("the rewrite is not recorded: %v", event)
	}
	// The rewritten line passes the check as a marked assumption.
	if _, lines := decisionOutcome([]decisionItem{{statement: want, verdict: "unsupported", why: item.why}}, "Fix the unreadable login message.", strings.Replace(intent, line, want, 1)); len(lines) != 0 {
		t.Fatalf("marked line refused: %q", lines)
	}
}

func TestShapeRewritesUnmarkedAssumptionOnDisk(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	sh := f.shaping()
	const line = "- Use a heading named Fix. (source: request)"
	f.write(".kogen/intents/is-even/intent.md", "# T\n\n## Decisions\n"+line+"\n")
	sh.pendingMarks = []assumptionMark{{line: line, marked: "- Use a heading named Fix. (assumption: confirm the heading) (source: request)", assumption: "confirm the heading"}}
	if err := sh.applyAssumptionMarks(); err != nil {
		t.Fatal(err)
	}
	shpContains(t, "intent", f.read(".kogen/intents/is-even/intent.md"), "(assumption: confirm the heading) (source: request)")
	if len(shpNamed(f.events(), "decision_marked")) != 1 {
		t.Fatalf("no decision_marked event: %v", f.events())
	}
}

func TestDecisionResolveStillRefusesContradiction(t *testing.T) {
	parallel(t)
	const request = "The saved login file may be unreadable; print `saved login cannot be read` and exit 4."
	const line = "- An unreadable login exits 0. (source: request)"
	item := decisionItem{statement: line, quotes: []string{"exit 4"}, verdict: "contradicts",
		intentValue: "exits 0", requestValue: "exit 4", difference: "exit code", why: "request says 4"}
	_, lines, marks := decisionResolve([]decisionItem{item}, request, "## Decisions\n"+line+"\n", nil)
	if len(marks) != 0 || !strings.Contains(strings.Join(lines, "\n"), "verdict: contradicts") {
		t.Fatalf("contradiction not refused: %q %v", lines, marks)
	}
	// An unsupported line whose quotes speak to it is not rewritten either.
	item = decisionItem{statement: line, quotes: []string{"exit 4"}, verdict: "unsupported", why: "x"}
	_, lines, marks = decisionResolve([]decisionItem{item}, request, "## Decisions\n"+line+"\n", nil)
	if len(marks) != 0 || len(lines) == 0 {
		t.Fatalf("grounded unsupported line rewritten: %q %v", lines, marks)
	}
}

// Labels the shaper invented for its own new decisions are not refused.
func TestCarriedDecisionsGateAcceptsInventedLabels(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	sh := f.shaping()
	sh.ledger = &decisionLedger{}
	draft := "## Decisions\n- Invalid JSON exits 4 (decision: D1, round 1) (evidence: cli/native.go) (source: request)\n" +
		"- Other providers keep their wording (decision: D2, round 1) (source: request)\n"
	if feedback, _, refused := sh.carriedDecisionsGate(draft); refused {
		t.Fatalf("invented labels refused: %q", feedback)
	}
}

// A decision claiming a carried-over record that no longer stands, or one that revises a
// record that does not exist, is still refused.
func TestCarriedClaimsStillRefused(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	sh := f.shaping()
	old := recordedDecision{ID: "D1", Round: 1, Decision: "Keep the old message", SupersededBy: "D2"}
	current := recordedDecision{ID: "D2", Round: 2, Decision: "Use the new message"}
	sh.ledger = &decisionLedger{Decisions: []recordedDecision{old, current}}
	draft := "## Decisions\n" + current.intentLine() + "\n" + old.intentLine() + "\n"
	if feedback, _, refused := sh.carriedDecisionsGate(draft); !refused || !strings.Contains(feedback, "D1 is not a recorded decision that stands") {
		t.Fatalf("superseded claim accepted: %q", feedback)
	}
	problems := []string{}
	_, _, refused, err := sh.decisionRecordGate([]sourcedDecision{{decision: "Revised", uncertainty: "u", revises: "D9", why: "w"}})
	if err != nil || !refused {
		t.Fatalf("revising a nonexistent decision accepted: %v %v", err, problems)
	}
}
