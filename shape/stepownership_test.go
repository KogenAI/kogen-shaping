package shape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
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

// bindingFx is an approve fixture whose Intent `fixture` is checked against other Intents
// with the real Node behind a wrapper that logs one line per invocation.
type bindingFx struct {
	*approveFx
	log string
}

func newBindingFx(t *testing.T) *bindingFx {
	t.Helper()
	cucumber := os.Getenv("KOGEN_CUCUMBER")
	if cucumber == "" {
		t.Skip("run npm ci in testdata/tools/cucumber for the real Cucumber")
	}
	f := &bindingFx{approveFx: newApproveFx(t)}
	f.log = filepath.Join(f.Aux, "node.log")
	wrapper := filepath.Join(f.Aux, "node-wrapper.sh")
	script := fmt.Sprintf("#!/bin/sh\necho run >> '%s'\nexec '%s' \"$@\"\n", f.log, os.Getenv("KOGEN_NODE"))
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	f.Env = append(f.Env, "KOGEN_NODE="+wrapper, "KOGEN_CUCUMBER="+cucumber)
	f.addIntent("fixture", "Given('own: step 1', () => {});\n")
	return f
}

// addIntent writes an Intent with one scenario of one step `<slug>: step 1` and a step file
// of the given content.
func (f *bindingFx) addIntent(slug, steps string) {
	step := fmt.Sprintf("features/steps/%s_steps.js", slug)
	if slug == "fixture" {
		step = "features/steps/fixture_steps.js"
	}
	md := strings.ReplaceAll(approveIntentMD, "fixture", slug)
	md = strings.ReplaceAll(md, "features/steps/"+slug+"_steps.js", step)
	text := slug + ": step 1"
	if slug == "fixture" {
		text = "own: step 1"
	}
	f.write(project.IntentMDPath(slug), md)
	f.write(project.FeaturePath(slug), "Feature: "+slug+"\n  Scenario: First\n    Then "+text+"\n")
	f.write(step, "const { Given } = require('@cucumber/cucumber');\n"+steps)
}

func (f *bindingFx) loaders() int {
	b, _ := os.ReadFile(f.log)
	return strings.Count(string(b), "run\n")
}

func (f *bindingFx) check() error {
	s, _, _ := f.session()
	return bindingCollision(context.Background(), f.Root, "fixture", s, verify.Tools{})
}

func TestBindingCollisionStartsOneLoaderPerStepFilePair(t *testing.T) {
	f := newBindingFx(t)
	// Each file defines only its own Intent's text: no text is matched on both sides.
	f.addIntent("oa", "Given('oa: step 1', () => {});\n")
	f.addIntent("ob", "Given('ob: step 1', () => {});\n")
	f.addIntent("oc", "Given('oc: step 1', () => {});\n")
	if err := f.check(); err != nil {
		t.Fatalf("no collision: %v", err)
	}
	if got := f.loaders(); got != 3 {
		t.Fatalf("loaders = %d, want 3 (one per step-file pair)", got)
	}
}

func TestBindingCollisionRefusesTheFirstCollidingPairInIntentOrder(t *testing.T) {
	f := newBindingFx(t)
	f.addIntent("oa", "Given('oa: step 1', () => {});\n")
	f.addIntent("ob", "Given('own: step 1', () => {});\n")
	f.addIntent("oc", "Given('own: step 1', () => {});\n")
	err := f.check()
	e := approveWantError(t, err, "intent", "step_ownership_collision",
		"Intents fixture and ob have overlapping step bindings in features/steps/fixture_steps.js and features/steps/ob_steps.js", 1)
	if got := strings.Join(e.Lines, "|"); got != "features/steps/fixture_steps.js|features/steps/ob_steps.js" {
		t.Fatalf("lines = %q", got)
	}
	if got := f.loaders(); got != 2 {
		t.Fatalf("loaders = %d, want 2 (oa, then the colliding ob)", got)
	}
}

func TestBindingCollisionNeedsOneTextMatchedOnBothSides(t *testing.T) {
	f := newBindingFx(t)
	// Each file's definition matches a different text, so no single text is matched twice.
	f.addIntent("fixture", "Given('own: step 1', () => {});\n")
	f.addIntent("oa", "Given('oa: step 1', () => {});\n")
	if err := f.check(); err != nil {
		t.Fatalf("definitions matching different texts: %v", err)
	}
	// The same text matched by a definition in each file collides.
	f.addIntent("oa", "Given(/^(own|oa): step 1$/, () => {});\n")
	approveWantError(t, f.check(), "intent", "step_ownership_collision", "", 1)
}

func TestBindingCollisionLoadFailuresAndMissingResultsDoNotRefuse(t *testing.T) {
	f := newBindingFx(t)
	f.addIntent("oa", "throw new Error('cannot load');\nGiven('own: step 1', () => {});\n")
	if err := f.check(); err != nil {
		t.Fatalf("file throwing before any definition: %v", err)
	}
	if f.loaders() != 1 {
		t.Fatalf("loaders = %d, want 1", f.loaders())
	}
	// A file keeps the definitions registered before it throws.
	f.addIntent("oa", "Given('own: step 1', () => {});\nthrow new Error('late');\n")
	approveWantError(t, f.check(), "intent", "step_ownership_collision", "", 1)
	// A Node that exits 1 without output yields no result: not a collision, no error.
	failing := filepath.Join(f.Aux, "node-fails.sh")
	if err := os.WriteFile(failing, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.Env = append(f.Env, "KOGEN_NODE="+failing)
	if err := f.check(); err != nil {
		t.Fatalf("loader without a result: %v", err)
	}
}
