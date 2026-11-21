'use strict';
// The Go source of the acceptance harness of shape-controls-patch-applies-with-recount, which
// shape-controls-patch-applies-with-recount__steps.js compiles into the shape package's tests through a Go
// overlay as shape/shape_controls_patch_recount_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

// Each case builds a candidate whose correct implementation patches big_a.js, big_b.js and
// notes.txt, checks it through controlsLargeLines and writes the refusal lines as JSON to
// SCPAR_OUT for the acceptance steps to judge. The harness judges nothing itself.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestShapeControlsPatchRecountAcceptance(t *testing.T) {
	out := os.Getenv("SCPAR_OUT")
	if out == "" {
		t.Fatal("SCPAR_OUT must be set")
	}
	notes := ""
	for _, w := range strings.Fields("apple birch cedar dune elm fern grove heath iris juniper kelp larch") {
		notes += w + "\n"
	}
	// The hunk header says 3 lines on each side; the hunk has 4 on each side.
	patchNotes := func(last string) map[string]any {
		return map[string]any{"patch": "--- a/notes.txt\n+++ b/notes.txt\n@@ -1,3 +1,3 @@\n apple\n birch\n-cedar\n+CEDAR\n " + last + "\n"}
	}
	lines := func(last string) []string {
		correct := map[string]any{
			"big_a.js": map[string]any{"patch": controlsOffsetPatch("big_a.js", 1, 0)},
			"big_b.js": map[string]any{"patch": controlsOffsetPatch("big_b.js", 1, 0)},
			"notes.txt": patchNotes(last),
		}
		wrongAdd := map[string]any{"big_a.js": map[string]any{"patch": controlsOffsetPatch("big_a.js", 0, 7)}}
		wrongSub := map[string]any{"big_b.js": map[string]any{"patch": controlsOffsetPatch("big_b.js", 0, 7)}}
		c, _ := controlsLargeCandidate(t, correct, wrongAdd, wrongSub)
		c.base["notes.txt"] = notes
		got := controlsLargeLines(t, c)
		if got == nil {
			got = []string{}
		}
		return got
	}
	obs := map[string][]string{
		"miscounted":      lines("dune"),
		"changed-context": lines("DUNEX"),
	}
	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
