package shape

import (
	"strings"
	"testing"
)

// A new file expectation can contradict another new expectation. Neither is
// present at HEAD, so looking only at the Intent and unchanged files misses it.
func TestContractEvidenceBetweenCandidateExpectations(t *testing.T) {
	const feature = "Then docs: llms.txt changes\nThen docs: llms.txt stays unchanged\n"
	const steps = "assert.notEqual(after, before);\nassert.equal(after, before);\n"
	e := contractEvidence{
		scenarios:    []string{"Update docs"},
		expectations: map[string]string{"acceptance.feature": feature, "steps.js": steps},
		intentPath:   "intent.md", intent: "Update documentation.", request: "Update llms.txt with a get-started link.",
		tracked: func(string) (string, bool) { return "", false },
	}
	for _, tc := range []struct {
		name, path, quote string
		want              bool
	}{
		{"opposite scenario", "acceptance.feature", "Then docs: llms.txt stays unchanged", true},
		{"opposite step assertion", "steps.js", "assert.equal(after, before);", true},
		{"invented quote", "steps.js", "assert.equal(after, invented);", false},
		{"request conflict", "<request>", "Update llms.txt with a get-started link.", true},
		{"invented request quote", "<request>", "Keep llms.txt unchanged.", false},
		{"outside package", "other.js", "assert.equal(after, before);", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := contractFinding{scenario: "Update docs", expectation: contractQuote{"acceptance.feature", "Then docs: llms.txt changes"}, contract: contractQuote{tc.path, tc.quote}, why: "the same finished file cannot both change and stay unchanged"}
			if got := e.evidenced(f); got != tc.want {
				t.Fatalf("evidenced=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestContractCheckRefusesConflictingCandidateScenarios(t *testing.T) {
	parallel(t)
	const feature = "Feature: Update docs\n  Scenario: Link from llms.txt\n    Then is-even: llms.txt links to get-started\n  Scenario: Keep llms.txt unchanged\n    Then is-even: llms.txt stays unchanged\n"
	const steps = `const {Then} = require('@cucumber/cucumber');
Then('is-even: llms.txt links to get-started', function () {});
Then('is-even: llms.txt stays unchanged', function () {});
`
	files := shpFiles()
	for _, raw := range files["files"].([]any) {
		m := raw.(map[string]any)
		switch m["path"] {
		case ccFeaturePath:
			m["content"] = feature
		case ccStepsPath:
			m["content"] = steps
		}
	}
	for _, verdict := range []string{"confirmed", "refuted"} {
		t.Run(verdict, func(t *testing.T) {
			f := newShpFx(t, shpSubmit("c1", files), shpSubmit("c2", files), shpSubmit("c3", files))
			f.setFeatureReport("report", feature, "FAILED", 1)
			finding := ccFindingArgs("Link from llms.txt", "Then is-even: llms.txt links to get-started", "")
			finding["contract"] = map[string]any{"path": ccFeaturePath, "quote": "Scenario: Keep llms.txt unchanged\n    Then is-even: llms.txt stays unchanged"}
			finding["why"] = "Link from llms.txt requires new content while Keep llms.txt unchanged requires the same finished file to retain its old content"
			f.Prov.contract = []shpStep{ccReview("r1", finding), ccConfirm("v1", ccVerdict(1, verdict, "compare the two finished-file requirements"))}
			exit, err := f.shape("Update llms.txt to link to https://get.kogen.dev/get-started.")
			reviews := ccReviews(f)
			if len(reviews) != 1 || reviews[0]["evidenced"] != float64(1) {
				t.Fatalf("reviews: %v", reviews)
			}
			if verdict == "confirmed" {
				if err == nil || exit == 0 || reviews[0]["result"] != "refused" {
					t.Fatalf("exit %d: %v; reviews %v", exit, err, reviews)
				}
				feedback := shpOutputs(f.Prov.requests()[1])["c1"]
				for _, name := range []string{"Link from llms.txt", "Keep llms.txt unchanged"} {
					if !strings.Contains(feedback, name) {
						t.Fatalf("feedback omits %s: %s", name, feedback)
					}
				}
			} else if err != nil || exit != 0 || reviews[0]["result"] != "clean" {
				t.Fatalf("refuted: exit %d: %v; reviews %v", exit, err, reviews)
			}
		})
	}
}
