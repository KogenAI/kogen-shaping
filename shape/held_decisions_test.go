package shape

import (
	"strings"
	"testing"
)

// The retained context-spend draft used its third submission attempt as
// the round, while Kogen recorded the new decision in conversation round 1.
func TestCarryDecisionNormalizesAttemptAsRound(t *testing.T) {
	d := recordedDecision{ID: "D10", Round: 1, Decision: "An item's kind is its type field, or message for an item without one, such as the opening and reset messages, and request items and the breakdown use the same derivation",
		Constraints: []string{"request items and the window breakdown derive kinds the same way"}, Uncertainty: "The request names bytes by item kind but not how a kind is derived; the user must confirm that the item type, with message for typeless items, is the intended kind",
		Evidence: []receipt{{Item: "build/context.go"}, {Item: "build/executor_native.go"}},
	}
	submitted := "- An item's kind is its type field, or message for an item without one, such as the opening and reset messages, and request items and the breakdown use the same derivation (decision: D10, round 3) (constraint: request items and the window breakdown derive kinds the same way) (evidence: build/context.go; build/executor_native.go) (assumption: The request names bytes by item kind but not how a kind is derived; the user must confirm that the item type, with message for typeless items, is the intended kind) (source: request)"
	sh := &shaping{ledger: &decisionLedger{Decisions: []recordedDecision{d}}}
	for _, tc := range []struct {
		name, line string
		repaired   bool
	}{
		{"attempt number", submitted, true},
		{"correct round", d.intentLine(), true},
		{"changed constraint", strings.Replace(submitted, "derive kinds the same way", "derive kinds differently", 1), false},
		{"changed decision", strings.Replace(submitted, "or message", "or unknown", 1), false},
		{"changed uncertainty", strings.Replace(submitted, "must confirm", "need not confirm", 1), false},
		{"different id", strings.Replace(submitted, "D10", "D11", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			draft := "# Record context\n\n## Decisions\n" + tc.line + "\n\n## Tests\n- Keep existing reset tests.\n"
			got := sh.normalizeDecisionRounds(draft)
			want := draft
			if tc.repaired {
				want = strings.Replace(draft, tc.line, d.intentLine(), 1)
			}
			if got != want {
				t.Fatalf("round repair: got %q, want %q", got, want)
			}
			if strings.Contains(got, "round 3") && tc.repaired {
				t.Fatal("attempt remains in round metadata")
			}
			if sh.normalizeDecisionRounds(got) != got {
				t.Fatal("round repair is not idempotent")
			}
		})
	}
	// A verbatim copy outside Decisions is not ledger metadata to edit.
	draft := "# Examples\n" + submitted + "\n\n## Decisions\n" + submitted + "\n\n## Tests\n" + submitted + "\n"
	want := "# Examples\n" + submitted + "\n\n## Decisions\n" + d.intentLine() + "\n\n## Tests\n" + submitted + "\n"
	if got := sh.normalizeDecisionRounds(draft); got != want {
		t.Fatalf("round repair changed text outside Decisions: %q", got)
	}
}

// A checker can paraphrase the body while copying the exact decision identity
// and annotations. The owner's D5/D9 refusals both lost their marker exemption.
func TestDecisionCheckRecognizesMarkedDecisionReference(t *testing.T) {
	const d5 = "- An action closes only by a passing check run on the server host or an explicit waiver; any member may request the check, only the person the action is for may waive. (decision: D5, round 1) (evidence: lib/studio/access.ex) (assumption: The owner of an action is taken to be the person named by who.) (source: request)"
	paraphrase := strings.Replace(d5, "may waive.", "may waive it.", 1)
	intent := "## Decisions\n" + d5 + "\n"
	it := decisionItem{statement: paraphrase, verdict: "unsupported", why: "The owner/ who mapping is a marked assumption."}
	if _, lines := decisionOutcome([]decisionItem{it}, "Close only on a passing check or owner waiver.", intent); lines != nil {
		t.Fatalf("marked decision paraphrase refused: %v", lines)
	}
	// Evidence or assumptions never override an explicit request conflict.
	it.verdict, it.intentValue, it.requestValue = "contradicts", "any member", "only administrators"
	it.quotes = []string{"only administrators may request the check"}
	if _, lines := decisionOutcome([]decisionItem{it}, it.quotes[0], intent); len(lines) == 0 {
		t.Fatal("explicit authority conflict was missed")
	}
	for _, bad := range []string{
		strings.Replace(paraphrase, "D5", "D6", 1),
		strings.Replace(paraphrase, "round 1", "round 2", 1),
		strings.Replace(paraphrase, "lib/studio/access.ex", "invented.ex", 1),
	} {
		if _, lines := decisionOutcome([]decisionItem{{statement: bad, verdict: "unsupported"}}, "", intent); len(lines) == 0 {
			t.Fatalf("invented decision reference accepted: %s", bad)
		}
	}
}

func TestShapeCarriesNewDecisionWithAttemptRound(t *testing.T) {
	parallel(t)
	files := shpFilesWithDecisions("- Safe diagnostics (decision: D1, round 3) (constraint: no bytes) (evidence: calc.js) (source: request)")
	files["decisions"] = []any{map[string]any{"decision": "Safe diagnostics", "evidence": []any{"calc.js"}, "constraints": []any{"no bytes"}}}
	f := newShpFx(t, shpAnswer(dlRead("calc.js")), shpSubmit("files", files), shpAck("ack"))
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("new decision refused for submission metadata: exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	published := f.read(".kogen/intents/is-even/intent.md")
	if !strings.Contains(published, "(decision: D1, round 1)") || strings.Contains(published, "(decision: D1, round 3)") {
		t.Fatalf("published incorrect ledger metadata: %s", published)
	}
	if refused := shpNamed(f.events(), "submission_refused"); len(refused) != 0 {
		t.Fatalf("metadata repair consumed a refusal: %v", refused)
	}
}
