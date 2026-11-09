package shape

import (
	"context"
	"strings"
	"testing"
)

// Before a first submission the accepted candidate lives in private proposals.
// A proof must exercise those bytes, not the stale retained draft on disk.
func TestShapeProbeUsesAcceptedProposalBytes(t *testing.T) {
	parallel(t)
	for _, fresh := range []bool{false, true} {
		t.Run(map[bool]string{false: "replacement", true: "new artifact"}[fresh], func(t *testing.T) {
			parallel(t)
			f, sh := newGuardedShaping(t)
			p := ".kogen/intents/is-even/probe-candidate.txt"
			if !fresh {
				f.write(p, "original")
			}
			text, err := sh.proposeFiles(map[string]any{"files": []any{map[string]any{"path": p, "content": "candidate"}}})
			if err != nil || !strings.HasPrefix(text, "retained:") {
				t.Fatalf("proposal: %s %v", text, err)
			}
			out, err := sh.probe(context.Background(), "cat "+p, nil)
			if err != nil || !strings.HasSuffix(out, "\ncandidate") {
				t.Fatalf("probe did not use accepted candidate: %q %v", out, err)
			}
			out, err = sh.probe(context.Background(), "cat "+p, []any{map[string]any{"path": p, "content": "probe-only"}})
			if err != nil || !strings.HasSuffix(out, "\nprobe-only") {
				t.Fatalf("explicit scratch override lost: %q %v", out, err)
			}
			if sh.proposed[p] != "candidate" {
				t.Fatal("scratch override changed the proposal")
			}
			if fresh {
				if f.exists(p) {
					t.Fatal("probe published a new file")
				}
			} else {
				shpEq(t, "project remains unchanged", f.read(p), "original")
			}
		})
	}
}
