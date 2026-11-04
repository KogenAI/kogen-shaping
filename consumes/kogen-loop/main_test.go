package consumes

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMain isolates the suite from the caller: no inherited Git or Kogen variable, a HOME of its
// own, no Git configuration and no Git prompt.
func TestMain(m *testing.M) {
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "GIT_") || strings.HasPrefix(name, "KOGEN_") || name == "EMAIL" {
			os.Unsetenv(name)
		}
	}
	home, err := os.MkdirTemp("", "consumes-kogen-loop-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "consumes: HOME:", err)
		os.Exit(1)
	}
	for name, value := range map[string]string{
		"HOME":                home,
		"GIT_CONFIG_GLOBAL":   "/dev/null",
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		os.Setenv(name, value)
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// jsonOf is the canonical JSON text of v (map keys sorted), to compare values structurally.
func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("encode %T: %v", v, err)
	}
	return string(b)
}
