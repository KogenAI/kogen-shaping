package shape

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

func frtEnv(path string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_") || (path != "" && strings.HasPrefix(kv, "PATH=")) {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	if path != "" {
		env = append(env, "PATH="+path)
	}
	return env
}

func frtGit(t *testing.T, root string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func frtWrite(t *testing.T, root, rel, content string, mode os.FileMode) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
}

func frtRead(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// frtFixture is a committed project: README.md, shaped.txt, an executable bin/run.sh and a
// link lnk to README.md.
func frtFixture(t *testing.T) (string, []string) {
	t.Helper()
	root, env := t.TempDir(), frtEnv("")
	frtGit(t, root, env, "init", "-q")
	frtGit(t, root, env, "config", "user.name", "Fixture")
	frtGit(t, root, env, "config", "user.email", "fixture@example.invalid")
	frtGit(t, root, env, "config", "commit.gpgsign", "false")
	frtGit(t, root, env, "config", "core.hooksPath", t.TempDir())
	frtWrite(t, root, "README.md", "# Calc\n\nA tiny calculator.\n", 0o644)
	frtWrite(t, root, "shaped.txt", "shaped\n", 0o644)
	frtWrite(t, root, "bin/run.sh", "#!/bin/sh\necho hi\n", 0o755)
	if err := os.Symlink("README.md", filepath.Join(root, "lnk")); err != nil {
		t.Fatal(err)
	}
	frtGit(t, root, env, "add", "-A")
	frtGit(t, root, env, "commit", "-qm", "fixture")
	return root, env
}

// frtPass runs the fixer pass with one fix command; the shaped paths are kept as they are.
func frtPass(root string, env []string, fix string, shaped ...string) error {
	sh := &shaping{s: sys.NewWith(env, root), repo: git.New(root, env), root: root,
		cfg: &project.Config{Fix: []string{fix}}, ev: &Evidence{}, started: time.Now()}
	return sh.fixAttempt(context.Background(), shaped)
}

func frtOnlyShapedChanged(t *testing.T, root string, env []string) {
	t.Helper()
	if status := frtGit(t, root, env, "status", "--porcelain"); status != " M shaped.txt\n" {
		t.Fatalf("status = %q, want only the shaped file", status)
	}
}

func TestFixerRevertRestoresATrackedFile(t *testing.T) {
	root, env := frtFixture(t)
	head := frtRead(t, root, "README.md")
	if err := frtPass(root, env, "echo more >> README.md; git add README.md; echo formatted >> shaped.txt", "shaped.txt"); err != nil {
		t.Fatalf("the pass returned an error: %v", err)
	}
	if got := frtRead(t, root, "README.md"); got != head {
		t.Fatalf("README.md = %q, want the HEAD content %q", got, head)
	}
	if staged := frtGit(t, root, env, "diff", "--cached", "--name-only"); staged != "" {
		t.Fatalf("staged paths = %q, want none", staged)
	}
	frtOnlyShapedChanged(t, root, env)
	if got := frtRead(t, root, "shaped.txt"); got != "shaped\nformatted\n" {
		t.Fatalf("shaped.txt = %q, want its formatting kept", got)
	}
}

func TestFixerRevertKeepsModeAndLinks(t *testing.T) {
	root, env := frtFixture(t)
	head := frtRead(t, root, "bin/run.sh")
	fix := "echo x >> bin/run.sh; chmod 644 bin/run.sh; rm lnk; echo plain > lnk; echo formatted >> shaped.txt"
	if err := frtPass(root, env, fix, "shaped.txt"); err != nil {
		t.Fatalf("the pass returned an error: %v", err)
	}
	if got := frtRead(t, root, "bin/run.sh"); got != head {
		t.Fatalf("bin/run.sh = %q, want %q", got, head)
	}
	if st, err := os.Lstat(filepath.Join(root, "bin/run.sh")); err != nil || st.Mode()&0o100 == 0 {
		t.Fatalf("bin/run.sh lost its executable bit: %v %v", st, err)
	}
	if target, err := os.Readlink(filepath.Join(root, "lnk")); err != nil || target != "README.md" {
		t.Fatalf("lnk is not a link to README.md: target %q, err %v", target, err)
	}
	frtOnlyShapedChanged(t, root, env)
	if idx := frtGit(t, root, env, "ls-files", "-s", "--", "bin/run.sh", "lnk"); !strings.HasPrefix(idx, "100755 ") || !strings.Contains(idx, "\n120000 ") {
		t.Fatalf("index entries = %q, want 100755 and 120000", idx)
	}
}

func TestFixerRevertSendsNoForbiddenGitCommand(t *testing.T) {
	root, env := frtFixture(t)
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "git.log")
	shim := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	frtWrite(t, bin, "git", shim, 0o755)
	shimEnv := frtEnv(bin + ":" + os.Getenv("PATH"))

	if err := frtPass(root, shimEnv, "echo more >> README.md; echo x >> bin/run.sh; echo formatted >> shaped.txt", "shaped.txt"); err != nil {
		t.Fatalf("the pass returned an error: %v", err)
	}
	logged, err := os.ReadFile(log)
	if err != nil || len(logged) == 0 {
		t.Fatalf("the git shim saw no command: %v", err)
	}
	forbidden := map[string]bool{"restore": true, "reset": true, "checkout": true, "clean": true, "symbolic-ref": true, "worktree": true, "remote": true}
	for _, line := range strings.Split(strings.TrimSpace(string(logged)), "\n") {
		args := strings.Fields(line)
		for i := 0; i < len(args); i++ {
			a := args[i]
			if a == "-c" || a == "-C" || a == "--git-dir" || a == "--work-tree" {
				i++
				continue
			}
			if strings.HasPrefix(a, "-") {
				continue
			}
			if forbidden[a] {
				t.Fatalf("the pass ran a forbidden git command: %q", line)
			}
			break
		}
	}
	frtOnlyShapedChanged(t, root, env)

	// The pinned guard is the one that ships: it refuses restore.
	if _, err := git.New(root, env).Run("restore", "--", "README.md"); !errors.Is(err, git.ErrForbidden) {
		t.Fatalf("restore through the guarded runner: err = %v, want git.ErrForbidden", err)
	}
}

func TestFixerRevertKeepsPriorBehaviourForSnapshotted(t *testing.T) {
	root, env := frtFixture(t)
	mine := frtRead(t, root, "README.md") + "mine\n"
	frtWrite(t, root, "README.md", mine, 0o644)
	if err := frtPass(root, env, "echo fixer >> README.md; echo junk > junk.txt; echo formatted >> shaped.txt", "shaped.txt"); err != nil {
		t.Fatalf("the pass returned an error: %v", err)
	}
	if got := frtRead(t, root, "README.md"); got != mine {
		t.Fatalf("README.md = %q, want the pre-fixer text %q", got, mine)
	}
	if _, err := os.Lstat(filepath.Join(root, "junk.txt")); err == nil {
		t.Fatal("junk.txt, which did not exist before the fixers, was kept")
	}
}
