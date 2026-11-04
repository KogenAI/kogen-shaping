package consumes

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMain isolates the suite from the caller: no inherited GIT_* or KOGEN_* variable, no
// EMAIL, a HOME of its own that is removed afterwards and no Git configuration or prompt.
func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "GIT_") || strings.HasPrefix(name, "KOGEN_") || name == "EMAIL" {
			os.Unsetenv(name)
		}
	}
	home, err := os.MkdirTemp("", "consumes-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "consumes:", err)
		return 1
	}
	defer os.RemoveAll(home)
	for name, value := range map[string]string{
		"HOME":                home,
		"GIT_CONFIG_GLOBAL":   os.DevNull,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(name, value); err != nil {
			fmt.Fprintln(os.Stderr, "consumes:", err)
			return 1
		}
	}
	return m.Run()
}
