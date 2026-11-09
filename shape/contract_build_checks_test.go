package shape

import (
	"strings"
	"testing"
)

func TestContractReviewAndConfirmationReceiveEffectiveChecks(t *testing.T) {
	parallel(t)
	for _, command := range []string{"GOFLAGS=-trimpath go test ./...", "go test ./shape -run '^TestOne$'", "true"} {
		t.Run(command, func(t *testing.T) {
			f := newShpFx(t, shpSubmit("c1", shpFiles()), ccCorrected("c2"), ccIntentOnly("c3"))
			f.write(".kogen/project.yaml", strings.Replace(f.read(".kogen/project.yaml"), "run: 'true'", "run: \""+command+"\"", 1))
			f.setFeatureReport("corrected-report", ccCorrectedFeature, "FAILED", 1)
			f.Prov.contract = []shpStep{ccReview("r1", ccContradiction()), ccConfirm("v1", ccVerdict(1, "confirmed", "four is even"))}
			if exit, err := f.shape("Add calc.is_even(n). Existing tests must stay green."); err != nil || exit != 0 {
				t.Fatalf("exit %d: %v\n%s", exit, err, f.Stderr)
			}
			requests := f.Prov.contractRequests()
			if len(requests) < 2 {
				t.Fatalf("wanted independent review and confirmation, got %d", len(requests))
			}
			for _, request := range requests[:2] {
				input := ccInput(request)
				if !strings.Contains(input, "# Configured Build checks") || !strings.Contains(input, command) {
					t.Fatalf("checker lacks the actual configured command %q", command)
				}
			}
		})
	}
}
