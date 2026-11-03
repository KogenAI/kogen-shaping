package shape

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"kogen-loop/sys"
)

// lineupShapeConfig loads a temporary project whose lineup is roles and whose other settings are
// rest, with the lineup where the coordinator in use reads it.
func lineupShapeConfig(t *testing.T, roles, rest string) *project.Config {
	t.Helper()
	root := t.TempDir()
	for path, content := range lineupFiles("version: 1\n"+rest, roles) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := project.LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// TestClaudeLineupNativeCompatibility: an omitted and an explicit native shape role are the same
// provider role and route natively; an external shaper is refused before inference (never shaped
// natively instead).
func TestClaudeLineupNativeCompatibility(t *testing.T) {
	parallel(t)
	omitted := lineupShapeConfig(t, "shape: {model: s, effort: high}\n", "")
	explicit := lineupShapeConfig(t, "shape: {executor: native, model: s, effort: high}\n", "")
	if ProviderRole(omitted.Models["shape"]) != ProviderRole(explicit.Models["shape"]) || omitted.Models["shape"] != explicit.Models["shape"] {
		t.Fatalf("explicit native differs: %+v %+v", omitted.Models["shape"], explicit.Models["shape"])
	}
	for _, cfg := range []*project.Config{omitted, explicit} {
		if roles := ShapeRoles(cfg); len(roles) != 1 || roles[0].Model != "s" {
			t.Fatalf("native roles %+v", roles)
		}
	}
	// An external shaper needs no native login and is selected only through the program's
	// executor; none available stops executor/unavailable (never shaped natively instead).
	ext := lineupShapeConfig(t, "shape: {executor: claude-code, model: s, effort: high}\n", "")
	if roles := ShapeRoles(ext); len(roles) != 0 {
		t.Fatalf("external shaper: %+v", roles)
	}
	_, _, err := selectExternal(nil, ext.Models["shape"])
	if se, ok := sys.AsError(err); !ok || se.Class != "executor" || se.Reason != executor.ReasonUnavailable {
		t.Fatalf("no executor: %v", err)
	}
}
