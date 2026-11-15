package shape

import (
	"strings"
	"testing"
)

func TestContractReviewInstructionsNameSelfContradictions(t *testing.T) {
	low := strings.ToLower(contractReviewInstructions)
	for _, phrase := range []string{
		"removed or replaced earlier in the same test",
		"uniqueness rule",
		"different request or tenant context",
		"a scenario that fails only because the requested feature is missing is not a finding",
		"when unsure, report nothing",
	} {
		if !strings.Contains(low, phrase) {
			t.Errorf("review instructions lack %q", phrase)
		}
	}
}

func TestContractEvidenceEarlierStatementOfSameStepFile(t *testing.T) {
	const steps = "const card = cards(:unfinished_thoughts);\ncard.destroy;\nassert.equal(cards(:unfinished_thoughts).title, 'x');\n"
	const feature = "Feature: Cards\n  Scenario: Out of cards\n    Then is-even: the drafted card is shown\n"
	e := contractEvidence{
		scenarios:    []string{"Out of cards"},
		expectations: map[string]string{"acceptance.feature": feature, "steps.js": steps},
		intentPath:   "intent.md", intent: "Show cards.", request: "Show cards.",
		tracked: func(string) (string, bool) { return "", false },
	}
	for _, tc := range []struct {
		name, quote string
		want        bool
	}{
		{"exact earlier statement", "card.destroy;", true},
		{"invented earlier statement", "card.destroy_all;", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := contractFinding{scenario: "Out of cards", expectation: contractQuote{"steps.js", "assert.equal(cards(:unfinished_thoughts).title, 'x');"}, contract: contractQuote{"steps.js", tc.quote}, why: "the card was destroyed earlier in the same test"}
			if got := e.evidenced(f); got != tc.want {
				t.Fatalf("evidenced=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestContractCheckRefusesSelfContradictingStepFile(t *testing.T) {
	parallel(t)
	const feat = "Feature: Update docs\n  Scenario: Drafted card\n    Then is-even: the drafted card is shown\n"
	const stp = `const {Then} = require('@cucumber/cucumber');
Then('is-even: the drafted card is shown', function () {
  card.destroy();
  assert.equal(cards('unfinished_thoughts').title, 'x');
});
`
	files := shpFiles()
	for _, raw := range files["files"].([]any) {
		m := raw.(map[string]any)
		switch m["path"] {
		case ccFeaturePath:
			m["content"] = feat
		case ccStepsPath:
			m["content"] = stp
		}
	}
	f := newShpFx(t, shpSubmit("c1", files), shpSubmit("c2", files), shpSubmit("c3", files))
	f.setFeatureReport("report", feat, "FAILED", 1)
	fa := ccFindingArgs("Drafted card", "Then is-even: the drafted card is shown", "")
	fa["expectation"] = map[string]any{"path": ccStepsPath, "quote": "assert.equal(cards('unfinished_thoughts').title, 'x');"}
	fa["contract"] = map[string]any{"path": ccStepsPath, "quote": "card.destroy();"}
	fa["why"] = "the card was destroyed earlier in the same test"
	f.Prov.contract = []shpStep{ccReview("r1", fa), ccConfirm("v1", ccVerdict(1, "confirmed", "destroyed then read"))}
	exit, err := f.shape("Show the drafted card.")
	reviews := ccReviews(f)
	if len(reviews) != 1 || reviews[0]["evidenced"] != float64(1) || err == nil || exit == 0 || reviews[0]["result"] != "refused" {
		t.Fatalf("exit %d: %v; reviews %v", exit, err, reviews)
	}
}
