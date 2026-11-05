package shape

import (
	"context"
	"github.com/KogenAI/kogen-coordinator/project"
	"strings"
	"testing"
)

func TestScopedStepPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"features/steps/alpha__steps.js", true},
		{"features/steps/alpha__.js", false},
		{"features/steps/steps.js", false},
		{"features/steps/beta__steps.js", false},
		{"features/steps/nested/alpha__steps.js", false},
	} {
		if got := scopedStepPath("features/steps", "alpha", tc.path); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestScopedStepPathHelperDir(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"features/steps/alpha__driver/main.go", true},
		{"features/steps/alpha__driver/sub/dir/x.txt", true},
		{"features/steps/alpha__driver/", false},
		{"features/steps/alpha__driver", false},
		{"features/steps/alpha__/main.go", false},
		{"features/steps/alpha__driver/../x.go", false},
		{"features/steps/alpha__driver//x.go", false},
		{"features/steps/alpha__driver/./x.go", false},
		{"features/steps/beta__driver/main.go", false},
		{"features/steps/driver/main.go", false},
		{"features/steps/nested/alpha__driver/main.go", false},
		{"features/other/alpha__driver/main.go", false},
		{"alpha__driver/main.go", false},
	} {
		if got := scopedStepPath("features/steps", "alpha", tc.path); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.path, got, tc.want)
		}
	}
	if scopedStepPath("", "alpha", "alpha__driver/main.go") {
		t.Error("a helper without a configured steps folder was accepted")
	}
}

func TestSharedStepApprovalAndPreflight(t *testing.T) {
	f := newApproveFx(t)
	other := strings.ReplaceAll(approveIntentMD, "fixture", "other")
	other = strings.ReplaceAll(other, "features/steps/other_steps.js", "features/steps/fixture_steps.js")
	f.write(project.IntentMDPath("other"), other)
	f.write(project.FeaturePath("other"), approveFeature)
	f.git("add", "-A")
	f.git("commit", "-qm", "shared packages")
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"fixture", "other"} {
		paths, err := project.Baseline(f.Root, slug, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := project.WriteApproval(f.Root, project.Approval{Slug: slug, SHA256: project.Hash(f.Root, paths), Base: strings.TrimSpace(f.git("rev-parse", "HEAD")), Approver: "Fixture", ApprovedAt: "2026-10-30T17:33:00Z"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sharedStepCollision(f.Root, "fixture", cfg); err != nil {
		t.Fatalf("unchanged sharing: %v", err)
	}
	before := f.read(".kogen/local/approvals/fixture.json")
	f.write("features/steps/fixture_steps.js", "// changed\n")
	exit, _, err := f.approve(ApproveInput{Hash: f.sha(), By: "Fixture"})
	if exit == 0 || err == nil {
		t.Fatal("changed shared file approved")
	}
	for _, part := range []string{"fixture", "other", "features/steps/fixture_steps.js"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error %q missing %s", err, part)
		}
	}
	if f.read(".kogen/local/approvals/fixture.json") != before {
		t.Fatal("approval changed on refusal")
	}
	s, out, _ := f.session()
	exit, err = Preflight(context.Background(), s, PreflightInput{Project: f.Root, Slug: "fixture"})
	if err != nil || exit == 0 || !strings.Contains(out.String(), "step_ownership_collision") {
		t.Fatalf("preflight: %d %v %s", exit, err, out)
	}
}
