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
		if run.Outcome != staticChecked {
			t.Fatalf("run %d: outcome %s", i, run.Outcome)
		}
	}
	if b, _ := os.ReadFile(counter); len(b) != 0 {
		t.Errorf("setup compiled from scratch %d times, want 0", strings.Count(string(b), "\n"))
	}
}

func TestSeededTreeKeepsTrackedContentAndSkipsKogen(t *testing.T) {
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

	if err := verify.SeedIgnored(root, dir); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string { b, _ := os.ReadFile(filepath.Join(dir, p)); return string(b) }
	if got := read("ignored.txt"); got != "tracked in the tree" {
		t.Errorf("tracked file overwritten: %q", got)
	}
	if got := read("deps/a/lib.bin"); got != "dependency" {
		t.Errorf("deps not seeded: %q", got)
	}
	for _, p := range []string{".kogen", "untracked.txt"} {
		if _, err := os.Lstat(filepath.Join(dir, p)); err == nil {
			t.Errorf("%s was copied", p)
		}
	}
}

// The static-check copy is seeded with the dependency build outputs a readiness scratch tree of
// the same lockfile and toolchain left in the run's cache; a copy of a different lockfile gets
// none and compiles from scratch.
func TestStaticChecksStartFromTheReadinessBuildOutputsOfTheSameLockfile(t *testing.T) {
	parallel(t)
	f := newApproveFx(t)
	counter := filepath.Join(f.Aux, "compiles")
	f.write(".gitignore", ".kogen/local/\nbuild_cache/\n")
	f.write("go.sum", "lock one\n")
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  setup:\n    - '[ -d build_cache ] || { mkdir build_cache; echo scratch >> "+counter+"; }; echo ok > build_cache/unit'\n  checks:\n    - name: lint\n      run: 'test -f build_cache/unit'\n  acceptance:\n    run: sh accept.sh\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "setup with a cache")
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	s, _, _ := f.session()
	cache := verify.NewBuildCache()
	defer cache.Close()
	rd := &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg, Cache: cache}

	// What readiness leaves behind: a scratch tree of the candidate whose setup compiled.
	head := f.Repo.Head()
	tree, err := f.Repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(t.TempDir(), "readiness")
	mat, err := f.Repo.Materialize(tree, scratch)
	if err != nil {
		t.Fatal(err)
	}
	key := rd.SeedBuild(s, scratch, approveSlug, mat.Paths)
	if err := os.MkdirAll(filepath.Join(scratch, "build_cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(scratch, "build_cache/unit"), []byte("ok\n"), 0o644)
	rd.KeepBuild(key, scratch)

	run := func(tree string) {
		t.Helper()
		g := newStaticGateRunner(s, f.Repo, rd, cfg, f.Root, approveSlug, func(string) {})
		r, err := g.run(context.Background(), f.Repo.Head(), tree, "package")
		if err != nil {
			t.Fatal(err)
		}
		if r.Outcome != staticChecked {
			t.Fatalf("outcome %s", r.Outcome)
		}
	}
	compiles := func() int { b, _ := os.ReadFile(counter); return strings.Count(string(b), "\n") }

	run(tree)
	if n := compiles(); n != 0 {
		t.Errorf("the same lockfile compiled from scratch %d times, want 0", n)
	}

	f.write("go.sum", "lock two\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "another lockfile")
	other, err := f.Repo.Tree(f.Repo.Head(), nil)
	if err != nil {
		t.Fatal(err)
	}
	run(other)
	if n := compiles(); n != 1 {
		t.Errorf("a different lockfile compiled from scratch %d times, want 1", n)
	}
}
