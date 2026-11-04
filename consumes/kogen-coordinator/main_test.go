package consumes

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestMain isolates every test from the caller: no inherited GIT_* or KOGEN_* variable, no
// EMAIL, a HOME of its own, and no user or system Git configuration.
func TestMain(m *testing.M) {
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(k, "GIT_") || strings.HasPrefix(k, "KOGEN_") || k == "EMAIL" {
			os.Unsetenv(k)
		}
	}
	home, err := os.MkdirTemp("", "consumes-kogen-coordinator-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// runGit runs the git binary in dir with a fixed author and no signing and returns its stdout.
// It arranges the repositories the tests observe; it is not one of the uses under test.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{
		"-c", "user.name=Test Author", "-c", "user.email=author@example.invalid",
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// gitStatus runs git and returns its combined output and exit status without failing.
func gitStatus(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Test Author", "-c", "user.email=author@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	return "", -1
}

// newRepo makes an empty repository in a fresh temporary directory.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	return dir
}

// write writes a file (and its folders) under dir.
func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commitAll stages and commits everything and returns the new HEAD.
func commitAll(t *testing.T, dir, message string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", message)
	return strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))
}

// treeOf is the tree id of a revision.
func treeOf(t *testing.T, dir, rev string) string {
	t.Helper()
	return strings.TrimSpace(runGit(t, dir, "rev-parse", rev+"^{tree}"))
}

// time_ is January 1st of year, for stamping a file with an old modification time.
func time_(year int) time.Time { return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC) }

// eq reports a difference between the value of the named inventory item and what its users rely on.
func eq(t *testing.T, item string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %#v, want %#v", item, got, want)
	}
}
