'use strict';
// The Go source of the shape-carried-decisions-problems-show-lines acceptance harness, which
// shape-carried-decisions-problems-show-lines__steps.js compiles into the shape package's tests through a Go overlay as
// shape/kcdp_acceptance_test.go; that path never exists in the repository. The source holds no
// backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"os"
	"testing"
)

// The acceptance harness of shape-carried-decisions-problems-show-lines: it runs the carried
// decisions gate on a ledger with a superseded D1 and a standing D2 and a draft carrying D1's line
// and a D2 line with changed text, and writes the refusal text as JSON to KCDP_OUT.
func TestKShapeCarriedDecisionsLinesAcceptance(t *testing.T) {
	out := os.Getenv("KCDP_OUT")
	if out == "" {
		t.Fatal("shape-carried-decisions-problems-show-lines: KCDP_OUT is empty")
	}
	f := newShpFx(t)
	sh := f.shaping()
	old := recordedDecision{ID: "D1", Round: 1, Decision: "Keep the old message", SupersededBy: "D2"}
	current := recordedDecision{ID: "D2", Round: 2, Decision: "Use the new message"}
	sh.ledger = &decisionLedger{Decisions: []recordedDecision{old, current}}
	changed := "- Use the newer message (decision: D2, round 2) (source: request)"
	draft := "## Decisions\n" + old.intentLine() + "\n" + changed + "\n"
	feedback, _, refused := sh.carriedDecisionsGate(draft)
	b, err := json.Marshal(map[string]any{"refused": refused, "feedback": feedback})
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("shape-carried-decisions-problems-show-lines: writing the observations to %s: %v", out, err)
	}
}
`;
