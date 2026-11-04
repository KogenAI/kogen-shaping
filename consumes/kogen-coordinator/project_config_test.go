package consumes

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"kogen-loop/sys"
)

func TestProjectPaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "checkout")
	for name, c := range map[string]struct{ got, want string }{
		"project.ConfigPath":   {project.ConfigPath, ".kogen/project.yaml"},
		"project.IntentDir":    {project.IntentDir("my-slug"), ".kogen/intents/my-slug"},
		"project.IntentMDPath": {project.IntentMDPath("my-slug"), ".kogen/intents/my-slug/intent.md"},
		"project.FeaturePath":  {project.FeaturePath("my-slug"), ".kogen/intents/my-slug/acceptance.feature"},
		"project.ModelPath":    {project.ModelPath("my-slug"), ".kogen/intents/my-slug/model.qnt"},
		"project.ApprovalPath": {project.ApprovalPath(root, "my-slug"), filepath.Join(root, ".kogen/local/approvals/my-slug.json")},
		"project.SourcePath":   {project.SourcePath(root, "my-slug"), filepath.Join(root, ".kogen/local/provenance/my-slug/current.json")},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, project.ConfigPath, projectYAML)
	cfg, err := project.LoadConfig(dir)
	if err != nil {
		t.Fatalf("project.LoadConfig: %v", err)
	}
	if !slices.Equal(cfg.Setup, []string{"npm ci"}) {
		t.Errorf("project.Config.Setup = %q, want [npm ci]", cfg.Setup)
	}
	if !slices.Equal(cfg.Fix, []string{"gofmt -w ."}) {
		t.Errorf("project.Config.Fix = %q, want [gofmt -w .]", cfg.Fix)
	}
	checks := cfg.Checks
	if len(checks) != 2 || checks[0].Name != "format" || checks[0].Run != `test -z "$(gofmt -l .)"` || checks[1].Name != "go-test" || checks[1].Run != "go test ./..." {
		t.Errorf("project.Config.Checks = %+v, want format and go-test in file order with their commands (project.Check.Name, project.Check.Run)", checks)
	}
	if cfg.Acceptance == nil || cfg.Acceptance.Run != "cucumber-js" || cfg.Acceptance.StepsDir != "features/steps" {
		t.Errorf("project.Config.Acceptance = %+v, want run cucumber-js and steps_dir features/steps (project.Acceptance.Run, project.Acceptance.StepsDir; trailing slash dropped)", cfg.Acceptance)
	}
	if !cfg.AcceptanceConfigured() {
		t.Errorf("project.Config.AcceptanceConfigured = false, want true when run and steps_dir are set")
	}
	if !slices.Equal(cfg.Protect, []string{".golangci.yml", "testdata/**"}) {
		t.Errorf("project.Config.Protect = %q, want the two protect entries", cfg.Protect)
	}
}

func TestConfigModels(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, project.ConfigPath, projectYAML)
	cfg, err := project.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Models) != 3 {
		t.Fatalf("project.Config.Models has %d roles, want shape, build and review", len(cfg.Models))
	}
	shape, build, review := cfg.Models["shape"], cfg.Models["build"], cfg.Models["review"]
	eq(t, "project.Role.Provider of shape", shape.Provider, "anthropic")
	eq(t, "project.Role.Model of shape", shape.Model, "m-shape")
	eq(t, "project.Role.Effort of shape", shape.Effort, "high")
	eq(t, "project.Role.BaseURL of shape", shape.BaseURL, "")
	eq(t, "project.Role.Executor of shape", shape.Executor, "")
	eq(t, "project.Role.Executor of build", build.Executor, "claude-code")
	eq(t, "project.Role.Model of build", build.Model, "m-build")
	eq(t, "project.Role.Effort of build", build.Effort, "medium")
	eq(t, "project.Role.Provider of build", build.Provider, "")
	eq(t, "project.Role.Provider of review", review.Provider, "openai-compatible")
	eq(t, "project.Role.BaseURL of review (trailing slash dropped)", review.BaseURL, "http://localhost:11434/v1")
	eq(t, "project.Role.Model of review", review.Model, "m-review")
	eq(t, "project.Role.Effort of review", review.Effort, "low")
	if shape.External() || !build.External() || review.External() {
		t.Errorf("project.Role.External: shape %v, build %v, review %v, want only the claude-code role external", shape.External(), build.External(), review.External())
	}
	if (project.Role{Executor: "native"}).External() || (project.Role{}).External() {
		t.Errorf("project.Role.External must be false for the native executor and for none")
	}

	// A file that names no model gets the defaults of each role.
	bare := t.TempDir()
	write(t, bare, project.ConfigPath, "version: 1\n")
	cfg, err = project.LoadConfig(bare)
	if err != nil {
		t.Fatal(err)
	}
	for role, want := range map[string][2]string{"shape": {"gpt-6.1-sol", "high"}, "build": {"gpt-6.1-sol", "medium"}, "review": {"gpt-6.1-terra", "medium"}} {
		got := cfg.Models[role]
		if got.Model != want[0] || got.Effort != want[1] || got.Provider != "" || got.Executor != "" || got.External() {
			t.Errorf("project.Config.Models[%s] = %+v, want the default model %s with effort %s on no provider and the native executor", role, got, want[0], want[1])
		}
	}
	if len(cfg.Setup)+len(cfg.Fix)+len(cfg.Checks)+len(cfg.Protect) != 0 || cfg.Acceptance != nil || cfg.AcceptanceConfigured() {
		t.Errorf("project.Config of a bare file = %+v, want no commands, no protect and no acceptance", cfg)
	}
}

func TestConfigAcceptanceConfigured(t *testing.T) {
	for name, tc := range map[string]struct {
		yaml string
		want bool
	}{
		"both":       {"version: 1\ncommands:\n  acceptance:\n    run: 'x'\n    steps_dir: 'steps'\n", true},
		"run only":   {"version: 1\ncommands:\n  acceptance:\n    run: 'x'\n    steps_dir: null\n", false},
		"dir only":   {"version: 1\ncommands:\n  acceptance:\n    run: null\n    steps_dir: 'steps'\n", false},
		"neither":    {"version: 1\ncommands:\n  acceptance:\n    run: null\n    steps_dir: null\n", false},
		"no section": {"version: 1\n", false},
	} {
		dir := t.TempDir()
		write(t, dir, project.ConfigPath, tc.yaml)
		cfg, err := project.LoadConfig(dir)
		if err != nil {
			t.Fatalf("%s: project.LoadConfig: %v", name, err)
		}
		if got := cfg.AcceptanceConfigured(); got != tc.want {
			t.Errorf("project.Config.AcceptanceConfigured with %s = %v, want %v (acceptance %+v)", name, got, tc.want, cfg.Acceptance)
		}
	}
}

func TestLoadConfigStops(t *testing.T) {
	stop := func(t *testing.T, err error) *sys.Error {
		t.Helper()
		se, ok := sys.AsError(err)
		if !ok {
			t.Fatalf("project.LoadConfig error = %v, want a *sys.Error", err)
		}
		return se
	}

	empty := t.TempDir()
	_, err := project.LoadConfig(empty)
	if se := stop(t, err); se.Class != "environment" || se.Reason != "not_initialized" || se.Exit != 3 {
		t.Errorf("project.LoadConfig without a file = %s/%s exit %d, want environment/not_initialized exit 3", se.Class, se.Reason, se.Exit)
	}

	old := t.TempDir()
	write(t, old, project.ConfigPath, "commands:\n  setup: []\n")
	_, err = project.LoadConfig(old)
	if se := stop(t, err); se.Class != "environment" || se.Reason != "old_kogen_config" || se.Exit != 3 {
		t.Errorf("project.LoadConfig of a file without version = %s/%s exit %d, want environment/old_kogen_config exit 3", se.Class, se.Reason, se.Exit)
	}

	bad := t.TempDir()
	write(t, bad, project.ConfigPath, "version: 1\ncommands:\n  bogus: 1\nmodels:\n  shape:\n    model: m\n    effort: ultra\n")
	_, err = project.LoadConfig(bad)
	se := stop(t, err)
	if se.Class != "environment" || se.Reason != "project_config_invalid" || se.Exit != 3 || se.Detail != filepath.Join(bad, project.ConfigPath) {
		t.Errorf("project.LoadConfig of an invalid file = %s/%s exit %d %q, want environment/project_config_invalid exit 3 naming the file", se.Class, se.Reason, se.Exit, se.Detail)
	}
	wantLines := []string{"line 3: unknown key bogus", `line 7: models.shape.effort "ultra" is not supported; accepted: any effort except ultra`}
	if !slices.Equal(se.Lines, wantLines) {
		t.Errorf("project.LoadConfig problems = %q, want one line per problem: %q", se.Lines, wantLines)
	}
}
