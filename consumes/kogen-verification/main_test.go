package consumes

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMain isolates the suite from the caller: no inherited GIT_* or KOGEN_* variable, no
// EMAIL, a HOME of its own that is removed afterwards and a Git that reads no configuration.
func TestMain(m *testing.M) {
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "GIT_") || strings.HasPrefix(name, "KOGEN_") || name == "EMAIL" {
			os.Unsetenv(name)
		}
	}
	home, err := os.MkdirTemp("", "consumes-verification-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for k, v := range map[string]string{
		"HOME":                home,
		"GIT_CONFIG_GLOBAL":   "/dev/null",
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		os.Setenv(k, v)
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
