package contract

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-shaping/shape"
)

// TestApproveExitCodes holds the exit codes and files kogen relies on from shape.Approve.
func TestApproveExitCodes(t *testing.T) {
	t.Parallel()
	t.Run("the review card is exit 5 and the full hash approves with exit 0", func(t *testing.T) {
		t.Parallel()
		f := newFx(t)
		f.committedDraft()
		hash := f.hash()

		f.newSession()
		exit, err := shape.Approve(context.Background(), f.S, shape.ApproveInput{Project: f.Root, Slug: slug})
		if err != nil || exit != 5 {
			t.Fatalf("card: exit %d, err %v\n%s", exit, err, f.Stdout)
		}

		f.newSession()
		by := "Contract Approver <approver@example.invalid>"
		exit, err = shape.Approve(context.Background(), f.S, shape.ApproveInput{Project: f.Root, Slug: slug, Hash: hash, By: by})
		if err != nil || exit != 0 {
			t.Fatalf("approve: exit %d, err %v\n%s", exit, err, f.Stdout)
		}
		eq(t, "stdout", f.Stdout.String(), "approved is-even "+hash[:8]+"\nNext: kogen intent build is-even\n")
		var a map[string]any
		if err := json.Unmarshal([]byte(f.read(".kogen/local/approvals/is-even.json")), &a); err != nil {
			t.Fatal(err)
		}
		eq(t, "slug", a["slug"], any("is-even"))
		eq(t, "sha256", a["sha256"], any(hash))
		eq(t, "approver", a["approver"], any(by))
		eq(t, "base", a["base"], any(f.head()))
	})

	t.Run("a directory that is not a Git repository is exit 3", func(t *testing.T) {
		t.Parallel()
		f := newFx(t)
		exit, err := shape.Approve(context.Background(), f.S, shape.ApproveInput{Project: t.TempDir(), Slug: slug})
		e, ok := sys.AsError(err)
		if !ok || e.Class != "environment" || e.Reason != "not_a_git_repo" || e.Exit != 3 || exit != 3 {
			t.Fatalf("exit %d err %v", exit, err)
		}
		if strings.TrimSpace(f.Stdout.String()) != "" {
			t.Fatalf("stdout %q", f.Stdout)
		}
	})
}
