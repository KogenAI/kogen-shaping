'use strict';
// The Go source of the shape-decision-record-problems-quote-decision acceptance harness, which
// shape-decision-record-problems-quote-decision__steps.js compiles into the shape package's tests through a Go overlay as
// shape/kdrq_acceptance_test.go; that path never exists in the repository. The source holds no
// backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The acceptance harness of shape-decision-record-problems-quote-decision: one native Shape per
// case, with a scripted shaper that first submits decisions the decision record gate refuses and
// then a good submission. KDRQ_CASE names the case; the feedback of the refused submission is
// written as JSON to KDRQ_OUT for the acceptance steps to judge.

const kdrqLong = "Use the documented rounding rule for every halfway value so that 2.5 and 3.5 are rounded in the same direction, and keep that rule in one place for all callers of the calculator module."

func kdrqDecisions(name string) []any {
	switch name {
	case "second-unsourced":
		return []any{
			map[string]any{"decision": "Keep the first wording", "uncertainty": "no test"},
			map[string]any{"decision": "Round halves up"},
		}
	case "long-text":
		return []any{map[string]any{"decision": kdrqLong}}
	case "exact-120":
		return []any{map[string]any{"decision": kdrqLong[:120]}}
	case "crowded-line":
		return []any{map[string]any{"decision": kdrqLong, "evidence": []any{strings.Repeat("n", 200) + ".rb"}}}
	}
	return nil
}

func TestKShapeDecisionRecordQuoteAcceptance(t *testing.T) {
	name, out := os.Getenv("KDRQ_CASE"), os.Getenv("KDRQ_OUT")
	decisions := kdrqDecisions(name)
	if decisions == nil || out == "" {
		t.Fatalf("shape-decision-record-problems-quote-decision: KDRQ_CASE %q is not a case of the harness, or KDRQ_OUT is empty", name)
	}
	ask := func(ds []any) map[string]any {
		return map[string]any{"questions": []any{shpQuestion("Which wording?", "keep", "shorter")}, "decisions": ds}
	}
	good := []any{map[string]any{"decision": "B", "uncertainty": "no test"}}
	f := newShpFx(t, shpSubmit("s1", ask(decisions)), shpSubmit("s2", ask(good)), shpUI(1))
	exit, err := f.shape("Add calc.is_even(n).")
	feedback := ""
	for _, r := range shpShaperRequests(f.Prov.requests()) {
		if o, ok := shpOutputs(r)["s1"]; ok {
			feedback = o
		}
	}
	b, merr := json.Marshal(map[string]any{"case": name, "exit": exit, "err": errString(err), "feedback": feedback, "stderr": f.Stderr.String()})
	if merr == nil {
		merr = os.WriteFile(out, b, 0o644)
	}
	if merr != nil {
		t.Fatalf("shape-decision-record-problems-quote-decision: writing the observations of case %s to %s: %v", name, out, merr)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
`;
