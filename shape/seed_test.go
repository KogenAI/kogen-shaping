package shape

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// Two static check runs of a project whose setup compiles into a git-ignored cache only when
// the cache is absent compile nothing when the checkout already holds the cache.
func TestStaticChecksStartFromCheckoutBuildOutputs(t *testing.T) {
	parallel(t)
	f := newApproveFx(t)
	counter := filepath.Join(f.Aux, "compiles")
	f.write(".gitignore", ".kogen/local/\nbuild_cache/\n")
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  setup:\n    - '[ -d build_cache ] || { mkdir build_cache; echo scratch >> "+counter+"; }; echo ok > build_cache/unit'\n  checks:\n    - name: lint\n      run: 'test -f build_cache/unit'\n  acceptance:\n    run: sh accept.sh\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "setup with a cache")
	// the shaping checkout has been set up once
	if err := os.MkdirAll(filepath.Join(f.Root, "build_cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(f.Root, "build_cache/unit"), []byte("ok\n"), 0o644)
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	s, _, _ := f.session()
	head := f.Repo.Head()
	tree, err := f.Repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(string) {})
	for i := 0; i < 2; i++ {
		run, err := g.run(context.Background(), head, tree, "package")
		if err != nil {
			t.Fatal(err)
		}
		if run.Outcome != staticChecked || run.SeededPaths != 1 {
			t.Fatalf("run %d: outcome %s, seeded %d", i, run.Outcome, run.SeededPaths)
		}
	}
	if b, _ := os.ReadFile(counter); len(b) != 0 {
		t.Errorf("setup compiled from scratch %d times, want 0", strings.Count(string(b), "\n"))
	}
}

func TestSeedIgnoredKeepsTrackedContentAndSkipsKogen(t *testing.T) {
	parallel(t)
	root, dir := t.TempDir(), t.TempDir()
	put := func(base, p, text string) {
		t.Helper()
		full := filepath.Join(base, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	put(root, ".gitignore", "ignored.txt\ndeps/\n.kogen/local/\n")
	put(root, "ignored.txt", "from the checkout")
	put(root, "deps/a/lib.bin", "dependency")
	put(root, ".kogen/local/secret.json", "local")
	put(root, "untracked.txt", "not ignored")
	put(dir, "ignored.txt", "tracked in the tree")

	n, _ := seedIgnored(root, dir)
	read := func(p string) string { b, _ := os.ReadFile(filepath.Join(dir, p)); return string(b) }
	if got := read("ignored.txt"); got != "tracked in the tree" {
		t.Errorf("tracked file overwritten: %q", got)
	}
	if got := read("deps/a/lib.bin"); got != "dependency" || n != 1 {
		t.Errorf("deps not seeded: %q, %d paths", got, n)
	}
	for _, p := range []string{".kogen", "untracked.txt"} {
		if _, err := os.Lstat(filepath.Join(dir, p)); err == nil {
			t.Errorf("%s was copied", p)
		}
	}
}
