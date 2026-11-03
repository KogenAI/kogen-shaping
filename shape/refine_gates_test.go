package shape

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

// A refined draft whose failed scenario runs a named test absent before the Build is refused with
// Shape's unwritten-tests text, counts as invalid and ends the refinement with the draft unchanged.
func TestRefineRefusesUnwrittenTestsAsShapeDoes(t *testing.T) {
	parallel(t)
	f := newRefFx(t, refOpts{},
		refSubmit("c1", refIntentPath, refTitled), refSubmit("c2", refIntentPath, refTitled), refSubmit("c3", refIntentPath, refTitled))
	f.unwrittenReport()
	before := f.pkg()
	exit, err := f.refine(f.hash0)
	e := shpWantError(t, err, "shape", "refinement_invalid", "no valid ready correction after 3 submissions; the original draft is preserved", 1)
	shpEq(t, "exit", exit, 1)
	shpEq(t, "lines", strings.Join(e.Lines, "\n"), unwrittenFeedback)
	reqs := f.Prov.requests()
	shpEq(t, "requests", len(reqs), 3)
	shpEq(t, "feedback 1", unwrittenOutput(t, reqs[1], "c1"), unwrittenFeedback)
	shpEq(t, "feedback 2", unwrittenOutput(t, reqs[2], "c2"), unwrittenFeedback)
	refEqual(t, "package", f.pkg(), before)
	shpEq(t, "hash", f.hash(), f.hash0)
	rel := strings.TrimPrefix(f.evidence(), f.Root+"/")
	shpEq(t, "diagnostic", f.read(rel+"/diagnostic-1.txt"), unwrittenFeedback)
}

// A confirmed contradiction is refused with Shape's contract text; the identical resubmission is
// refused again without another review.
func TestRefineRefusesAContractContradictionOnceAndRemembersIt(t *testing.T) {
	parallel(t)
	f := newRefFx(t, refOpts{},
		refSubmit("c1", refIntentPath, refTitled), refSubmit("c2", refIntentPath, refTitled), refSubmit("c3", refIntentPath, refTitled))
	f.Prov.contract = []shpStep{ccReview("r1", ccContradiction()), ccConfirm("v1", ccVerdict(1, "confirmed", "four is even"))}
	before := f.pkg()
	_, err := f.refine(f.hash0)
	shpWantError(t, err, "shape", "refinement_invalid", "no valid ready correction after 3 submissions; the original draft is preserved", 1)
	reqs := f.Prov.requests()
	shpEq(t, "requests", len(reqs), 3)
	shpEq(t, "feedback 1", shpOutputs(reqs[1])["c1"], ccRefusal)
	shpEq(t, "feedback 2", shpOutputs(reqs[2])["c2"], ccRefusal)
	shpEq(t, "contract requests", len(f.Prov.contractRequests()), 2)
	shpEq(t, "contract reviews", len(shpNamed(f.evidenceEvents(), "contract_review")), 1)
	refEqual(t, "package", f.pkg(), before)
	if f.exists(".kogen/local/shaping/" + shpSlug + "/contract-contradictions.json") {
		t.Fatal("a contradiction record was written")
	}
}

// A candidate that passes the gates is reviewed and publishes; the review is not a request of the
// refiner.
func TestRefineReviewsAPassingCandidateWithoutAnotherRefinerRequest(t *testing.T) {
	parallel(t)
	f := newRefFx(t, refOpts{}, refSubmit("c1", refIntentPath, refTitled))
	exit, err := f.refine(f.hash0)
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	shpEq(t, "refiner requests", len(f.Prov.requests()), 1)
	shpEq(t, "contract requests", len(f.Prov.contractRequests()), 1)
	shpEq(t, "contract reviews", len(shpNamed(f.evidenceEvents(), "contract_review")), 1)
}

// The external refiner receives the refusal text as its round feedback.
func TestRefineGivesTheExternalRefinerShapesRefusalText(t *testing.T) {
	parallel(t)
	f := newRefFx(t, refOpts{external: 6})
	f.unwrittenReport()
	files := result(map[string]any{"files": []any{map[string]any{"path": refIntentPath, "content": refTitled}}})
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){files, files, files}}
	before := f.pkg()
	_, err := f.refineWith(x, f.hash0)
	e := shpWantError(t, err, "shape", "refinement_invalid", "no valid ready correction after 3 submissions; the original draft is preserved", 1)
	shpEq(t, "lines", strings.Join(e.Lines, "\n"), unwrittenFeedback)
	shpEq(t, "rounds", len(x.inputs), 3)
	shpEq(t, "feedback 2", x.inputs[1].Feedback, unwrittenFeedback)
	shpEq(t, "feedback 3", x.inputs[2].Feedback, unwrittenFeedback)
	refEqual(t, "package", f.pkg(), before)
	noNativeRequests(t, f.shpFx)
}
