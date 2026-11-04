package consumes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-verification/verify"
	"kogen-loop/sys"
)

// Signatures and field types kogen-shaping's calls rely on.
var (
	_ func(string, string) error                                                    = verify.CopyTree
	_ func(verify.Probe, context.Context, *sys.Session) (verify.ProbeResult, error) = verify.Probe.Run
	_ func(*verify.ProbeFileError) string                                           = (*verify.ProbeFileError).Error
	_ time.Duration                                                                 = verify.ProbeTimeout
	_ string                                                                        = verify.Probe{}.Root
	_ string                                                                        = verify.Probe{}.Command
	_ []verify.ProbeFile                                                            = verify.Probe{}.Files
	_ verify.Tools                                                                  = verify.Probe{}.Tools
	_ string                                                                        = verify.ProbeFile{}.Path
	_ string                                                                        = verify.ProbeFile{}.Content
	_ []string                                                                      = verify.ProbeFileError{}.Problems
	_ int                                                                           = verify.ProbeResult{}.Status
	_ bool                                                                          = verify.ProbeResult{}.TimedOut
	_ string                                                                        = verify.ProbeResult{}.Output
	_ float64                                                                       = verify.ProbeResult{}.Wall
)

// verify.CopyTree copies the working tree for a probe: every file including ignored ones,
// without .git and .kogen/local, keeping modes.
func TestCopyTree(t *testing.T) {
	root, scratch := t.TempDir(), t.TempDir()
	write(t, root, "src/main.go", "package main\n")
	write(t, root, "build/ignored.txt", "ignored but copied\n")
	write(t, root, ".git/HEAD", "ref: refs/heads/main\n")
	write(t, root, ".kogen/local/readiness/x.json", "{}\n")
	write(t, root, ".kogen/intents/demo/intent.md", "# Demo\n")
	write(t, root, ".kogen/project.yaml", "commands: {}\n")
	if err := os.WriteFile(filepath.Join(root, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := verify.CopyTree(root, scratch); err != nil {
		t.Fatalf("verify.CopyTree: %v", err)
	}
	for rel, want := range map[string]string{
		"src/main.go":                   "package main\n",
		"build/ignored.txt":             "ignored but copied\n",
		".kogen/intents/demo/intent.md": "# Demo\n",
		".kogen/project.yaml":           "commands: {}\n",
	} {
		if got := readFile(t, scratch, rel); got != want {
			t.Errorf("verify.CopyTree: %s must be copied whole, got %q", rel, got)
		}
	}
	for _, rel := range []string{".git", ".kogen/local"} {
		if exists(filepath.Join(scratch, rel)) {
			t.Errorf("verify.CopyTree: %s must not be copied", rel)
		}
	}
	if st, err := os.Stat(filepath.Join(scratch, "run.sh")); err != nil || st.Mode().Perm()&0o111 == 0 {
		t.Errorf("verify.CopyTree: the executable bit must survive, got %v %v", st, err)
	}
	if err := verify.CopyTree(filepath.Join(root, "missing"), t.TempDir()); err == nil {
		t.Error("verify.CopyTree: a missing root must fail")
	}
}

// verify.Probe.Run runs Command in a scratch copy of Root with Files laid over it: the
// checkout stays as it was, and the result carries the exit status, the combined output and
// the wall time.
func TestProbeRun(t *testing.T) {
	isolateTmp(t)
	root := t.TempDir()
	write(t, root, "data.txt", "hello\n")
	s := session(root)

	p := verify.Probe{
		Root:    root,
		Command: "cat data.txt over/new.txt; echo warning >&2; echo changed > data.txt; exit 7",
		Files:   []verify.ProbeFile{{Path: "over/new.txt", Content: "laid over\n"}},
	}
	r, err := p.Run(context.Background(), s)
	if err != nil {
		t.Fatalf("verify.Probe.Run: %v", err)
	}
	if r.Status != 7 || r.TimedOut || r.Wall < 0 || r.Wall > 60 {
		t.Errorf("verify.ProbeResult: want status 7, not timed out and a short wall time, got status %d timed out %v wall %v", r.Status, r.TimedOut, r.Wall)
	}
	for _, part := range []string{"hello\n", "laid over\n", "warning\n"} {
		if !strings.Contains(r.Output, part) {
			t.Errorf("verify.ProbeResult.Output: must hold the combined output including %q, got %q", part, r.Output)
		}
	}
	if got := readFile(t, root, "data.txt"); got != "hello\n" {
		t.Errorf("verify.Probe.Root: the probe must run in a copy, the checkout's data.txt became %q", got)
	}
	if exists(filepath.Join(root, "over")) {
		t.Error("verify.ProbeFile: files are laid over the scratch copy, not the checkout")
	}
	if left, _ := filepath.Glob(filepath.Join(os.TempDir(), "kogen-probe-*")); len(left) != 0 {
		t.Errorf("verify.Probe.Run: the scratch copy must be removed, left %v", left)
	}

	tools := verify.Tools{Node: "/tools/node", Quint: "/tools/quint", Cucumber: "/tools/cucumber"}
	r, err = verify.Probe{Root: root, Tools: tools, Command: `printf '%s|%s|%s' "$KOGEN_NODE" "$KOGEN_QUINT" "$KOGEN_CUCUMBER"`}.Run(context.Background(), s)
	if err != nil || r.Status != 0 || r.Output != "/tools/node|/tools/quint|/tools/cucumber" {
		t.Errorf("verify.Probe.Tools: the command must get the tools as KOGEN_NODE, KOGEN_QUINT and KOGEN_CUCUMBER, got %+v %v", r, err)
	}

	r, err = verify.Probe{Root: root, Command: `printf '%s' "$HOME"`}.Run(context.Background(), s)
	if err != nil || r.Output == os.Getenv("HOME") || r.Output == "" {
		t.Errorf("verify.Probe.Run: the command must run with a home of its own, not the caller's %q, got %+v %v", os.Getenv("HOME"), r, err)
	}
}

// verify.Probe.Run refuses files that leave the scratch copy or would be written through a
// symlink with a *verify.ProbeFileError before the command starts.
func TestProbeFileError(t *testing.T) {
	isolateTmp(t)
	root, outside := t.TempDir(), t.TempDir()
	write(t, root, "data.txt", "x\n")
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	p := verify.Probe{
		Root:    root,
		Command: "echo ran > ran.txt",
		Files: []verify.ProbeFile{
			{Path: "../escape.txt", Content: "x"},
			{Path: "link/inside.txt", Content: "x"},
			{Path: "fine.txt", Content: "x"},
		},
	}
	_, err := p.Run(context.Background(), session(root))
	var pe *verify.ProbeFileError
	if !errors.As(err, &pe) {
		t.Fatalf("verify.ProbeFileError: Probe.Run must refuse with a *verify.ProbeFileError, got %v", err)
	}
	want := []string{"../escape.txt is outside the scratch copy", "link/inside.txt would be written through a symlink"}
	if !equalStrings(pe.Problems, want) {
		t.Errorf("verify.ProbeFileError.Problems: want %q, got %q", want, pe.Problems)
	}
	if got, wantText := pe.Error(), "error: "+strings.Join(want, "; "); got != wantText {
		t.Errorf("verify.ProbeFileError.Error: want %q, got %q", wantText, got)
	}
	if exists(filepath.Join(outside, "inside.txt")) {
		t.Error("verify.ProbeFileError: nothing may be written through the symlink")
	}
	if got := (&verify.ProbeFileError{Problems: []string{"a", "b"}}).Error(); got != "error: a; b" {
		t.Errorf("verify.ProbeFileError.Error: want \"error: a; b\", got %q", got)
	}
}

// verify.ProbeTimeout is the five minutes a probe may run, which kogen-shaping reports in
// seconds.
func TestProbeTimeout(t *testing.T) {
	if verify.ProbeTimeout != 5*time.Minute || int(verify.ProbeTimeout.Seconds()) != 300 {
		t.Errorf("verify.ProbeTimeout: want 5 minutes (300 seconds), got %v", verify.ProbeTimeout)
	}
}
