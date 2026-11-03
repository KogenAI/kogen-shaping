package shape

import (
	"path/filepath"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

const shpGuardConfig = "version: 1\ncommands:\n  checks: [{name: unit, run: 'true'}]\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\nprotect: [guard.txt]\n"

// withGuard protects guard.txt in the committed project configuration.
func (f *shpFx) withGuard() {
	f.t.Helper()
	f.write("guard.txt", "protected")
	f.write(".kogen/project.yaml", shpGuardConfig)
	f.git("add", "-A")
	f.git("commit", "-qm", "guard")
}

// shaping is the state a validation runs against, for a project that has no Intent yet.
func (f *shpFx) shaping() *shaping {
	f.t.Helper()
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		f.t.Fatal(err)
	}
	return &shaping{s: f.S, repo: f.Repo, root: f.Root, slug: shpSlug, cfg: cfg, ev: f.evidence(), client: f.Client,
		originals: map[string]original{}, proposed: map[string]string{}}
}

func (f *shpFx) evidence() *Evidence {
	f.t.Helper()
	ev, err := NewEvidence(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug)
	if err != nil {
		f.t.Fatal(err)
	}
	return ev
}

// newGuardedShaping is a fixture with a protected guard.txt and a committed steps folder, with
// the shaping state of its project.
func newGuardedShaping(t *testing.T) (*shpFx, *shaping) {
	t.Helper()
	f := newShpFx(t)
	f.withGuard()
	f.write("features/steps/.keep", "")
	f.git("add", "-A")
	f.git("commit", "-qm", "steps folder")
	return f, f.shaping()
}
