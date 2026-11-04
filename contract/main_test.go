package contract

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/durable"
)

// TestMain prepares the process the way shape's own TestMain does: a git wrapper first on PATH
// on macOS (the xcode-select shim costs about twice what the real git does on every call), one
// absolute Node as KOGEN_NODE and the repository's Cucumber as KOGEN_CUCUMBER, which Shape's step
// check runs on the submitted step file.
func TestMain(m *testing.M) {
	var shim string
	if dir := realGitDir(); dir != "" {
		if d, err := os.MkdirTemp("", "kogen-git-"); err == nil {
			if writeGitWrapper(d, filepath.Join(dir, "git")) == nil {
				shim = d
				os.Setenv("PATH", d+string(os.PathListSeparator)+os.Getenv("PATH"))
			} else {
				os.RemoveAll(d)
			}
		}
	}
	if err := resolveTestTools(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	durable.SkipSyncInTests()
	code := m.Run()
	if shim != "" {
		os.RemoveAll(shim)
	}
	os.Exit(code)
}

// writeGitWrapper writes a script that execs the real git; a symlink would break git's runtime
// prefix under the Command Line Tools.
func writeGitWrapper(dir, git string) error {
	quoted := "'" + strings.ReplaceAll(git, "'", `'\''`) + "'"
	return os.WriteFile(filepath.Join(dir, "git"), []byte("#!/bin/sh\nexec "+quoted+" \"$@\"\n"), 0o700)
}

func realGitDir() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	if p, err := exec.LookPath("git"); err != nil || p != "/usr/bin/git" {
		return ""
	}
	out, err := exec.Command("xcode-select", "-p").Output()
	if err != nil {
		return ""
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "usr", "bin")
	if st, err := os.Stat(filepath.Join(dir, "git")); err != nil || st.IsDir() {
		return ""
	}
	return dir
}

// resolveTestTools exports one absolute Node as KOGEN_NODE, and the checkout's Cucumber as
// KOGEN_CUCUMBER when the caller sets none.
func resolveTestTools() error {
	node := os.Getenv("KOGEN_NODE")
	if st, err := os.Stat(node); node == "" || err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {
		path, err := exec.LookPath("node")
		if err != nil {
			return fmt.Errorf("contract tests need Node: no executable node found (KOGEN_NODE unset or invalid, none on PATH)")
		}
		out, err := exec.Command(path, "-p", "process.execPath").Output()
		node = strings.TrimSpace(string(out))
		if err != nil || !filepath.IsAbs(node) {
			return fmt.Errorf("contract tests need Node: %s did not report its absolute path: %v", path, err)
		}
	}
	os.Setenv("KOGEN_NODE", node)
	if os.Getenv("KOGEN_CUCUMBER") == "" {
		if dir, err := filepath.Abs(filepath.Join("..", "testdata", "tools", "cucumber", "node_modules")); err == nil {
			if st, err := os.Stat(dir); err == nil && st.IsDir() {
				os.Setenv("KOGEN_CUCUMBER", dir)
			}
		}
	}
	return nil
}
