package consumes

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-loop/sys"
)

// session is a Session of its own whose notices land in the returned buffer.
func session(dir string) (*sys.Session, *bytes.Buffer) {
	s := sys.NewWith(os.Environ(), dir)
	stderr := new(bytes.Buffer)
	s.Stdout, s.Stderr = io.Discard, stderr
	return s, stderr
}

func TestBuildLock(t *testing.T) {
	root := t.TempDir()
	path := git.BuildLockPath(root)
	if want := filepath.Join(root, ".kogen", "local", "build.lock"); path != want {
		t.Errorf("git.BuildLockPath = %q, want %q", path, want)
	}
	if held, err := git.NewLock(path).Held(); err != nil || held {
		t.Errorf("git.NewLock: Held of a lock nobody took = %v, %v, want false", held, err)
	}

	s, _ := session(root)
	lock, err := git.AcquireBuild(root, s)
	if err != nil {
		t.Fatalf("git.AcquireBuild: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil || !strings.HasPrefix(string(body), fmt.Sprintf("kogen-build-lock 1\npid %d\ntoken ", os.Getpid())) {
		t.Errorf("git.AcquireBuild: the lock file at %s holds %q, %v, want the header, this pid and a token", path, body, err)
	}
	// Preflight asks, without taking it, whether a running process holds it.
	if held, err := git.NewLock(path).Held(); err != nil || !held {
		t.Errorf("git.Lock.Held of a taken lock = %v, %v, want true", held, err)
	}

	// A second Build in the same repository stops: environment/build_running, exit 3, naming the owner.
	second, err := git.AcquireBuild(root, s)
	if second != nil {
		t.Errorf("git.AcquireBuild returned a second lock while the first is held")
	}
	se, ok := sys.AsError(err)
	if !ok {
		t.Fatalf("git.AcquireBuild of a held lock: error %v, want a *sys.Error", err)
	}
	if se.Class != "environment" || se.Reason != "build_running" || se.Exit != 3 || !strings.Contains(se.Detail, fmt.Sprintf("pid %d", os.Getpid())) {
		t.Errorf("git.AcquireBuild stop = %s/%s exit %d %q, want environment/build_running exit 3 naming pid %d", se.Class, se.Reason, se.Exit, se.Detail, os.Getpid())
	}
	if held, _ := git.NewLock(path).Held(); !held {
		t.Errorf("git.AcquireBuild: a refused attempt must not release the owner's lock")
	}

	lock.Release()
	if held, err := git.NewLock(path).Held(); err != nil || held {
		t.Errorf("git.Lock.Release: Held after release = %v, %v, want false", held, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("git.Lock.Release must remove the lock file, Stat gives %v", err)
	}
	lock.Release() // repeated calls do nothing

	again, err := git.AcquireBuild(root, s)
	if err != nil {
		t.Fatalf("git.AcquireBuild after Release: %v", err)
	}
	again.Release()
}

func TestBuildLockStale(t *testing.T) {
	root := t.TempDir()
	path := git.BuildLockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// A lock an earlier Kogen left behind: a plain pid file of a process that does not run.
	if err := os.WriteFile(path, []byte("2147483646\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if held, err := git.NewLock(path).Held(); err != nil || held {
		t.Errorf("git.Lock.Held of a lock whose owner does not run = %v, %v, want false", held, err)
	}
	s, stderr := session(root)
	lock, err := git.AcquireBuild(root, s)
	if err != nil {
		t.Fatalf("git.AcquireBuild over a stale lock: %v", err)
	}
	if got := stderr.String(); got != "kogen: removed the lock of a Build that is no longer running\n" {
		t.Errorf("git.AcquireBuild: notice on stderr = %q, want the removed-lock notice", got)
	}
	if held, _ := git.NewLock(path).Held(); !held {
		t.Errorf("git.AcquireBuild over a stale lock must own the lock")
	}
	lock.Release()
	if held, _ := git.NewLock(path).Held(); held {
		t.Errorf("git.Lock.Release: the lock is still held")
	}
}

func TestRepoTakePin(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "base")
	repo := git.New(dir, nil)
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")

	var first *git.Pin
	first, err := repo.TakePin()
	if err != nil || first == nil {
		t.Fatalf("git.Repo.TakePin: %v", err)
	}
	same, err := repo.TakePin()
	if err != nil {
		t.Fatal(err)
	}
	if got := first.Changes(same); len(got) != 0 {
		t.Errorf("git.Pin.Changes of two pins of an untouched checkout = %v, want none", got)
	}

	// A new hook, then its changed bytes: named by path.
	write(t, dir, ".git/hooks/pre-commit", "#!/bin/sh\nexit 0\n")
	added, err := repo.TakePin()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"hook " + hook}
	if got := first.Changes(added); !slices.Equal(got, want) {
		t.Errorf("git.Pin.Changes after a hook was added = %v, want %v", got, want)
	}
	write(t, dir, ".git/hooks/pre-commit", "#!/bin/sh\nexit 1\n")
	edited, err := repo.TakePin()
	if err != nil {
		t.Fatal(err)
	}
	if got := added.Changes(edited); !slices.Equal(got, want) {
		t.Errorf("git.Pin.Changes after a hook was edited = %v, want %v", got, want)
	}
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	removed, err := repo.TakePin()
	if err != nil {
		t.Fatal(err)
	}
	if got := edited.Changes(removed); !slices.Equal(got, want) {
		t.Errorf("git.Pin.Changes after a hook was removed = %v, want %v", got, want)
	}
	if got := first.Changes(removed); len(got) != 0 {
		t.Errorf("git.Pin.Changes after the hook is gone again = %v, want none", got)
	}

	// A configuration key that selects a program is named, never its value; other keys are not pinned.
	runGit(t, dir, "config", "core.sshCommand", "secret-ssh --flag")
	runGit(t, dir, "config", "alias.st", "status")
	runGit(t, dir, "config", "user.name", "Someone Else")
	configured, err := repo.TakePin()
	if err != nil {
		t.Fatal(err)
	}
	got := first.Changes(configured)
	if want := []string{"alias.st", "core.sshCommand"}; !slices.Equal(got, want) {
		t.Errorf("git.Pin.Changes after executable-selecting keys were set = %v, want %v", got, want)
	}
	if strings.Contains(strings.Join(got, " "), "secret-ssh") {
		t.Errorf("git.Pin.Changes must not show configuration values: %v", got)
	}
	runGit(t, dir, "config", "core.sshCommand", "other-ssh")
	changedValue, err := repo.TakePin()
	if err != nil {
		t.Fatal(err)
	}
	if got := configured.Changes(changedValue); !slices.Equal(got, []string{"core.sshCommand"}) {
		t.Errorf("git.Pin.Changes after a value changed = %v, want [core.sshCommand]", got)
	}

	if _, err := git.New(t.TempDir(), append(os.Environ(), "GIT_CEILING_DIRECTORIES="+os.TempDir())).TakePin(); err == nil {
		t.Errorf("git.Repo.TakePin outside a repository must fail, never return an unchanged pin")
	}
}
