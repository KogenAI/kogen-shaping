package shape

import (
	"strings"
	"testing"
)

// TestDecisionRecordProblemsQuoteTheDecision: every problem line of a refused sourced decision
// names it by position and quotes its first 120 characters, shrunk to keep the line within 300.
func TestDecisionRecordProblemsQuoteTheDecision(t *testing.T) {
	parallel(t)
	const unsourced = "no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under"
	long := "Use the documented rounding rule for every halfway value so that 2.5 and 3.5 are rounded in the same direction, and keep it stable across releases."
	whole := long[:120]
	lines := func(submitted ...sourcedDecision) []string {
		sh := newShpFx(t).shaping()
		sh.stateDir = t.TempDir()
		sh.ledger = &decisionLedger{}
		text, _, refused, err := sh.decisionRecordGate(submitted)
		if err != nil || !refused {
			t.Fatalf("refused %v, err %v", refused, err)
		}
		return strings.Split(text, "\n")[1:]
	}
	eq := func(what string, got []string, want ...string) {
		t.Helper()
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s:\n%q\nwant\n%q", what, got, want)
		}
	}

	eq("second unsourced", lines(sourcedDecision{decision: "Keep it", uncertainty: "u"}, sourcedDecision{decision: "Round halves up"}),
		`- sourced decision 2 ("Round halves up"): `+unsourced)
	eq("long", lines(sourcedDecision{decision: long}), `- sourced decision 1 ("`+whole+`"): `+unsourced)
	eq("exactly 120", lines(sourcedDecision{decision: whole}), `- sourced decision 1 ("`+whole+`"): `+unsourced)
	eq("whitespace", lines(sourcedDecision{decision: "A  b\n\tc "}), `- sourced decision 1 ("A b c"): `+unsourced)

	path := strings.Repeat("d/", 60) + "x.rb"
	crowded := lines(sourcedDecision{decision: long, uncertainty: "u", evidence: []string{path}})
	if len(crowded) != 1 || len([]rune(crowded[0])) != 300 || !strings.HasSuffix(crowded[0], path+" is not a file tracked at HEAD") ||
		!strings.HasPrefix(crowded[0], `- sourced decision 1 ("Use the documented rounding rule for eve`) {
		t.Fatalf("crowded line: %q", crowded)
	}
	huge := strings.Repeat("d/", 150) + "x.rb"
	full := lines(sourcedDecision{decision: long, uncertainty: "u", evidence: []string{huge}})
	if !strings.HasPrefix(full[0], `- sourced decision 1 (""): `) {
		t.Fatalf("no room: %q", full)
	}
}
