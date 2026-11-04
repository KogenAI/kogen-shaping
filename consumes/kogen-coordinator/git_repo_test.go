package consumes

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/durable"
	"github.com/KogenAI/kogen-coordinator/git"
	"kogen-loop/sys"
)

func TestGitSignatures(t *testing.T) {
	var (
		_ func(string) string                                        = git.BuildLockPath
		_ func(string, *sys.Session) (*git.Lock, error)              = git.AcquireBuild
		_ func(string) *git.Lock                                     = git.NewLock
		_ func(string, []string) *git.Repo                           = git.New
		_ func(string, []string) (*git.Repo, error)                  = git.Open
		_ func(*git.Lock) (bool, error)                              = (*git.Lock).Held
		_ func(*git.Lock)                                            = (*git.Lock).Release
		_ func(*git.Pin, *git.Pin) []string                          = (*git.Pin).Changes
		_ func(*git.Repo, string) *git.Repo                          = (*git.Repo).At
		_ func(*git.Repo, string, string) ([]string, error)          = (*git.Repo).Changed
		_ func(*git.Repo, ...string) ([]string, error)               = (*git.Repo).ChangedPaths
		_ func(*git.Repo) string                                     = (*git.Repo).Head
		_ func(*git.Repo) string                                     = (*git.Repo).Identity
		_ func(*git.Repo) (string, error)                            = (*git.Repo).IndexPath
		_ func(*git.Repo, string, string) (*git.Materialized, error) = (*git.Repo).Materialize
		_ func(*git.Repo, ...string) (string, error)                 = (*git.Repo).OK
		_ func(*git.Repo, []string, ...string) (string, error)       = (*git.Repo).OKEnv
		_ func(*git.Repo) *git.Repo                                  = (*git.Repo).Observing
		_ func(*git.Repo) (string, error)                            = (*git.Repo).OperationInProgress
		_ func(*git.Repo, ...string) (git.Result, error)             = (*git.Repo).Run
		_ func(*git.Repo, string, string) (string, bool)             = (*git.Repo).Show
		_ func(*git.Repo) (*git.Pin, error)                          = (*git.Repo).TakePin
		_ func(*git.Repo, string, []string) (string, error)          = (*git.Repo).Tree
		_ string                                                     = (&git.Repo{}).Dir
		_ int                                                        = git.Result{}.Status
		_ string                                                     = git.Result{}.Stderr
		_ string                                                     = git.Result{}.Stdout
		_ func(*os.File) error                                       = durable.Sync
		_ *git.Pin
		_ *git.Repo
	)
}

func TestDurableSync(t *testing.T) {
	var _ func(*os.File) error = durable.Sync
	f, err := os.Create(filepath.Join(t.TempDir(), "synced"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("flushed"); err != nil {
		t.Fatal(err)
	}
	if err := durable.Sync(f); err != nil {
		t.Errorf("durable.Sync: an open file was not flushed without error: %v", err)
	}
	f.Close()
	if err := durable.Sync(f); err == nil {
		t.Errorf("durable.Sync: a closed file must report the error of its flush, not nil")
	}
}

func TestOpen(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "sub/deep/file.txt", "x\n")

	repo, err := git.Open(filepath.Join(dir, "sub", "deep"), nil)
	if err != nil {
		t.Fatalf("git.Open: a folder inside a work tree must open: %v", err)
	}
	if repo.Dir != dir {
		t.Errorf("git.Repo.Dir (of git.Open) = %q, want the work tree root %q that callers lock and read from", repo.Dir, dir)
	}

	// Outside any work tree: the environment/not_a_git_repo stop, exit 3, naming the folder.
	outside := t.TempDir()
	env := append(os.Environ(), "GIT_CEILING_DIRECTORIES="+filepath.Dir(outside))
	_, err = git.Open(outside, env)
	se, ok := sys.AsError(err)
	if !ok {
		t.Fatalf("git.Open: outside a repository the error is %v, want a *sys.Error", err)
	}
	if se.Class != "environment" || se.Reason != "not_a_git_repo" || se.Exit != 3 || !strings.Contains(se.Detail, outside) {
		t.Errorf("git.Open: stop = %s/%s exit %d %q, want environment/not_a_git_repo exit 3 naming %s", se.Class, se.Reason, se.Exit, se.Detail, outside)
	}
}

func TestNew(t *testing.T) {
	// New does not check that the folder is a repository; commands run in the folder.
	dir := t.TempDir()
	repo := git.New(dir, os.Environ())
	if repo.Dir != dir {
		t.Errorf("git.Repo.Dir (of git.New) = %q, want %q", repo.Dir, dir)
	}
	if got := repo.Head(); got != "" {
		t.Errorf("git.New: Head of a plain folder = %q, want empty", got)
	}
	runGit(t, dir, "init", "-q")
	if res, err := repo.Run("rev-parse", "--is-inside-work-tree"); err != nil || res.Stdout != "true\n" {
		t.Errorf("git.New: commands must run in Repo.Dir, got %q, %v", res.Stdout, err)
	}
}

func TestRepoAt(t *testing.T) {
	a, b := newRepo(t), newRepo(t)
	write(t, a, "f", "a\n")
	write(t, b, "f", "b\n")
	headA, headB := commitAll(t, a, "a"), commitAll(t, b, "b")

	ra := git.New(a, nil)
	rb := ra.At(b)
	if rb == ra {
		t.Errorf("git.Repo.At: must return another Repo, not the receiver")
	}
	if rb.Dir != b || ra.Dir != a {
		t.Errorf("git.Repo.At: At(%q).Dir = %q and the receiver's Dir = %q, want %q and %q", b, rb.Dir, ra.Dir, b, a)
	}
	if ra.Head() != headA || rb.Head() != headB {
		t.Errorf("git.Repo.At: Head of each view = %q and %q, want %q and %q", ra.Head(), rb.Head(), headA, headB)
	}
}

func TestRepoHead(t *testing.T) {
	dir := newRepo(t)
	repo := git.New(dir, nil)
	if got := repo.Head(); got != "" {
		t.Errorf("git.Repo.Head: a repository without commits has Head %q, want empty", got)
	}
	write(t, dir, "a.txt", "1\n")
	first := commitAll(t, dir, "one")
	if got := repo.Head(); got != first || len(got) != 40 {
		t.Errorf("git.Repo.Head = %q, want the full commit id %q", got, first)
	}
	write(t, dir, "a.txt", "2\n")
	second := commitAll(t, dir, "two")
	if got := repo.Head(); got != second || got == first {
		t.Errorf("git.Repo.Head = %q after a second commit, want %q", got, second)
	}
}

func TestRepoIdentity(t *testing.T) {
	dir := newRepo(t)
	repo := git.New(dir, nil)
	if got := repo.Identity(); got != "" {
		t.Errorf("git.Repo.Identity without user.name and user.email = %q, want empty (never guessed from the host)", got)
	}
	runGit(t, dir, "config", "user.name", "Ada Lovelace")
	if got := repo.Identity(); got != "" {
		t.Errorf("git.Repo.Identity with a name but no email = %q, want empty", got)
	}
	runGit(t, dir, "config", "user.email", "ada@example.invalid")
	if got, want := repo.Identity(), "Ada Lovelace <ada@example.invalid>"; got != want {
		t.Errorf("git.Repo.Identity = %q, want %q", got, want)
	}

	// The identity is read through the caller's environment: a global configuration there counts.
	other := newRepo(t)
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, []byte("[user]\n\tname = Grace Hopper\n\temail = grace@example.invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := git.Open(other, append(os.Environ(), "GIT_CONFIG_GLOBAL="+global))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := opened.Identity(), "Grace Hopper <grace@example.invalid>"; got != want {
		t.Errorf("git.Repo.Identity through the Open environment = %q, want %q", got, want)
	}
}

func TestRepoChangedPaths(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, ".gitignore", "*.log\n")
	write(t, dir, "a.txt", "a\n")
	write(t, dir, "b.txt", "b\n")
	write(t, dir, "keep/k.txt", "k\n")
	commitAll(t, dir, "base")
	repo := git.New(dir, nil)

	got, err := repo.ChangedPaths()
	if err != nil || len(got) != 0 {
		t.Fatalf("git.Repo.ChangedPaths of a clean checkout = %v, %v, want none", got, err)
	}

	write(t, dir, "a.txt", "changed\n")       // unstaged
	write(t, dir, "staged.txt", "s\n")        // staged
	runGit(t, dir, "add", "staged.txt")       //
	write(t, dir, "new/dir/u.txt", "u\n")     // untracked, listed file by file
	write(t, dir, "ignored.log", "ignored\n") // ignored: never listed
	got, err = repo.ChangedPaths()
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"a.txt", "new/dir/u.txt", "staged.txt"}; !slices.Equal(got, want) {
		t.Errorf("git.Repo.ChangedPaths = %v, want %v", got, want)
	}

	// Limited to pathspecs.
	got, err = repo.ChangedPaths("new")
	if err != nil || !slices.Equal(got, []string{"new/dir/u.txt"}) {
		t.Errorf("git.Repo.ChangedPaths(\"new\") = %v, %v, want [new/dir/u.txt]", got, err)
	}
	got, err = repo.ChangedPaths("keep")
	if err != nil || len(got) != 0 {
		t.Errorf("git.Repo.ChangedPaths(\"keep\") = %v, %v, want none", got, err)
	}

	// A rename reports the new path, then the old one.
	runGit(t, dir, "mv", "b.txt", "moved.txt")
	got, err = repo.ChangedPaths("moved.txt", "b.txt")
	if err != nil || !slices.Equal(got, []string{"moved.txt", "b.txt"}) {
		t.Errorf("git.Repo.ChangedPaths of a rename = %v, %v, want [moved.txt b.txt]", got, err)
	}

	// Not a repository: Git's failure is the error.
	if _, err := git.New(t.TempDir(), append(os.Environ(), "GIT_CEILING_DIRECTORIES="+os.TempDir())).ChangedPaths(); err == nil {
		t.Errorf("git.Repo.ChangedPaths outside a repository must fail")
	}
}

func TestRepoRun(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "base")
	repo := git.New(dir, nil)

	res, err := repo.Run("rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("git.Repo.Run: %v", err)
	}
	eq(t, "git.Result.Status of a command that succeeded", res.Status, 0)
	eq(t, "git.Result.Stdout of a command that succeeded", res.Stdout, "true\n")
	eq(t, "git.Result.Stderr of a command that succeeded", res.Stderr, "")

	// A command that ran and failed is a Result, not an error; stdout and stderr stay apart.
	res, err = repo.Run("rev-parse", "--verify", "refs/heads/nope")
	if err != nil {
		t.Errorf("git.Repo.Run of a command that ran and failed returned the error %v, want it in git.Result.Status", err)
	}
	eq(t, "git.Result.Status of a command that failed", res.Status, 128)
	eq(t, "git.Result.Stdout of a command that failed", res.Stdout, "")
	if res.Stderr == "" {
		t.Errorf("git.Result.Stderr of a command that failed is empty, want Git's message apart from stdout")
	}

	// The calls of Shape: an unset key is status 1 without output; a set path is printed.
	res, err = repo.Run("config", "--path", "--get", "core.excludesFile")
	if err != nil || res.Status != 1 || res.Stdout != "" {
		t.Errorf("git.Repo.Run of an unset key = %+v, %v, want status 1 and no output", res, err)
	}
	excludes := filepath.Join(t.TempDir(), "excludes")
	runGit(t, dir, "config", "core.excludesFile", excludes)
	res, err = repo.Run("config", "--path", "--get", "core.excludesFile")
	if err != nil || res.Status != 0 || res.Stdout != excludes+"\n" {
		t.Errorf("git.Repo.Run of a set key = %+v, %v, want status 0 and %q", res, err, excludes+"\n")
	}

	// The porcelain status of one untracked path is exactly "?? <path>" and a NUL.
	write(t, dir, "docs/request.txt", "ask\n")
	res, err = repo.Run("status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ":(literal)docs/request.txt")
	if err != nil || res.Status != 0 || res.Stdout != "?? docs/request.txt\x00" {
		t.Errorf("git.Repo.Run status = %+v, %v, want stdout %q", res, err, "?? docs/request.txt\x00")
	}

	// Commands that rewrite history or move HEAD never run: that is an error, not a Result.
	if _, err := repo.Run("reset", "--hard"); err == nil {
		t.Errorf("git.Repo.Run: \"reset\" must be refused with an error")
	}
	if _, err := repo.Run("push"); err == nil {
		t.Errorf("git.Repo.Run: \"push\" must be refused with an error")
	}
}

func TestRepoOK(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	write(t, dir, "sub/b.txt", "b\n")
	head := commitAll(t, dir, "base")
	repo := git.New(dir, nil)

	out, err := repo.OK("ls-tree", "-r", "-z", "--name-only", "HEAD")
	if err != nil || out != "a.txt\x00sub/b.txt\x00" {
		t.Errorf("git.Repo.OK ls-tree = %q, %v, want the NUL-separated tracked paths", out, err)
	}
	out, err = repo.OK("cat-file", "-t", "HEAD:a.txt")
	if err != nil || strings.TrimSpace(out) != "blob" {
		t.Errorf("git.Repo.OK cat-file = %q, %v, want blob", out, err)
	}
	out, err = repo.OK("ls-tree", "-r", "-l", "-z", "HEAD")
	if err != nil || !strings.Contains(out, "\t") || !strings.Contains(out, "a.txt\x00") {
		t.Errorf("git.Repo.OK ls-tree -l = %q, %v, want the sized listing", out, err)
	}

	// A command that must succeed and fails is an error naming the command.
	if out, err := repo.OK("rev-parse", "--verify", "refs/heads/nope"); err == nil || out != "" || !strings.Contains(err.Error(), "refs/heads/nope") {
		t.Errorf("git.Repo.OK of a failing command = %q, %v, want an error naming it", out, err)
	}

	// The absolute path of the exclude file and a shared clone without checkout.
	out, err = repo.OK("rev-parse", "--path-format=absolute", "--git-path", "info/exclude")
	if want := filepath.Join(dir, ".git", "info", "exclude"); err != nil || !samePath(strings.TrimSpace(out), want) {
		t.Errorf("git.Repo.OK git-path = %q, %v, want %q", out, err, want)
	}
	scratch := filepath.Join(t.TempDir(), "scratch")
	if _, err := repo.OK("clone", "--quiet", "--shared", "--no-checkout", "--", dir, scratch); err != nil {
		t.Fatalf("git.Repo.OK clone: %v", err)
	}
	if got := git.New(scratch, nil).Head(); got != head {
		t.Errorf("git.Repo.OK clone: the clone's Head = %q, want %q", got, head)
	}

	// Restoring one literal path from HEAD, staged and in the work tree.
	write(t, dir, "a.txt", "dirty\n")
	runGit(t, dir, "add", "a.txt")
	if _, err := repo.OK("restore", "--source=HEAD", "--staged", "--worktree", "--", ":(literal)a.txt"); err != nil {
		t.Fatalf("git.Repo.OK restore: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.txt")); string(b) != "a\n" {
		t.Errorf("git.Repo.OK restore left a.txt = %q, want the committed bytes", b)
	}
	if got, err := repo.ChangedPaths(); err != nil || len(got) != 0 {
		t.Errorf("git.Repo.OK restore left changes %v, %v", got, err)
	}
}

func TestRepoOKEnv(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	write(t, dir, ".kogen/local/state.json", "{}\n")
	write(t, dir, "keep/k.txt", "k\n")
	commitAll(t, dir, "base")
	repo := git.New(dir, nil)
	tree := treeOf(t, dir, "HEAD")
	userIndex, err := os.ReadFile(filepath.Join(dir, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}

	// The revision computation of Shape: a private index through GIT_INDEX_FILE, without .kogen/local.
	index := filepath.Join(t.TempDir(), "private-index")
	env := []string{"GIT_INDEX_FILE=" + index}
	if _, err := repo.OKEnv(env, "read-tree", tree); err != nil {
		t.Fatalf("git.Repo.OKEnv read-tree: %v", err)
	}
	if _, err := repo.OKEnv(env, "rm", "--cached", "-r", "--ignore-unmatch", "--", ".kogen/local"); err != nil {
		t.Fatalf("git.Repo.OKEnv rm: %v", err)
	}
	out, err := repo.OKEnv(env, "write-tree")
	if err != nil {
		t.Fatalf("git.Repo.OKEnv write-tree: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if len(trimmed) != 40 || out != trimmed+"\n" || trimmed == tree {
		t.Errorf("git.Repo.OKEnv write-tree = %q, want a new tree id and a newline (not %s)", out, tree)
	}
	listed := runGit(t, dir, "ls-tree", "-r", "--name-only", trimmed)
	if listed != "a.txt\nkeep/k.txt\n" {
		t.Errorf("git.Repo.OKEnv: the private tree lists %q, want a.txt and keep/k.txt without .kogen/local", listed)
	}
	if now, _ := os.ReadFile(filepath.Join(dir, ".git", "index")); !bytes.Equal(now, userIndex) {
		t.Errorf("git.Repo.OKEnv: the user's index changed although the commands used GIT_INDEX_FILE")
	}
	if _, err := os.Stat(index); err != nil {
		t.Errorf("git.Repo.OKEnv: the private index %s was not written: %v", index, err)
	}
	if _, err := repo.OKEnv(env, "rev-parse", "--verify", "refs/heads/nope"); err == nil {
		t.Errorf("git.Repo.OKEnv of a failing command must be an error")
	}
}

func TestRepoShow(t *testing.T) {
	dir := newRepo(t)
	repo := git.New(dir, nil)
	if got, ok := repo.Show("HEAD", "a.txt"); ok || got != "" {
		t.Errorf("git.Repo.Show without commits = %q, %v, want empty and false", got, ok)
	}
	write(t, dir, "a.txt", "committed\n")
	write(t, dir, "sub/b.txt", "nested\n")
	commitAll(t, dir, "base")
	write(t, dir, "a.txt", "working copy\n")

	if got, ok := repo.Show("HEAD", "a.txt"); !ok || got != "committed\n" {
		t.Errorf("git.Repo.Show(HEAD, a.txt) = %q, %v, want the committed bytes and true", got, ok)
	}
	if got, ok := repo.Show("HEAD", "sub/b.txt"); !ok || got != "nested\n" {
		t.Errorf("git.Repo.Show(HEAD, sub/b.txt) = %q, %v, want nested\\n and true", got, ok)
	}
	if got, ok := repo.Show("HEAD", "missing.txt"); ok || got != "" {
		t.Errorf("git.Repo.Show of a path HEAD lacks = %q, %v, want empty and false", got, ok)
	}
	if _, ok := repo.Show("-x", "a.txt"); ok {
		t.Errorf("git.Repo.Show of a revision that looks like an option must be false")
	}
	if _, ok := repo.Show("", "a.txt"); ok {
		t.Errorf("git.Repo.Show of an empty revision must be false")
	}
}

func TestRepoTree(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, ".gitignore", "*.log\nbuild/\n")
	write(t, dir, "a.txt", "a\n")
	write(t, dir, "b.txt", "b\n")
	write(t, dir, "run.sh", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(dir, "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	head := commitAll(t, dir, "base")
	repo := git.New(dir, nil)

	clean, err := repo.Tree(head, nil)
	if err != nil || clean != treeOf(t, dir, "HEAD") {
		t.Errorf("git.Repo.Tree of a clean checkout = %q, %v, want HEAD's tree %q", clean, err, treeOf(t, dir, "HEAD"))
	}

	write(t, dir, "a.txt", "edited\n")              // tracked, edited
	runGit(t, dir, "rm", "-q", "--cached", "b.txt") // untracked now, still on disk
	write(t, dir, "new/u.txt", "untracked\n")       // untracked, not ignored
	write(t, dir, "x.log", "ignored\n")             // ignored
	write(t, dir, "build/out.o", "ignored\n")       // ignored folder
	write(t, dir, "staged.txt", "staged\n")
	runGit(t, dir, "add", "staged.txt")
	if err := os.Remove(filepath.Join(dir, "run.sh")); err != nil {
		t.Fatal(err)
	}
	indexBefore, _ := os.ReadFile(filepath.Join(dir, ".git", "index"))

	tree, err := repo.Tree(head, nil)
	if err != nil {
		t.Fatalf("git.Repo.Tree: %v", err)
	}
	got := strings.Fields(runGit(t, dir, "ls-tree", "-r", "--name-only", tree))
	if want := []string{".gitignore", "a.txt", "b.txt", "new/u.txt", "staged.txt"}; !slices.Equal(got, want) {
		t.Errorf("git.Repo.Tree lists %v, want tracked and untracked-not-ignored files, without ignored or deleted ones: %v", got, want)
	}
	if body := runGit(t, dir, "show", tree+":a.txt"); body != "edited\n" {
		t.Errorf("git.Repo.Tree: a.txt = %q, want the working copy", body)
	}
	if indexAfter, _ := os.ReadFile(filepath.Join(dir, ".git", "index")); !bytes.Equal(indexAfter, indexBefore) {
		t.Errorf("git.Repo.Tree changed the user's index; it must build in a private index")
	}

	// File modes are part of the tree.
	write(t, dir, "run.sh", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(dir, "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	tree, err = repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	if listing := runGit(t, dir, "ls-tree", tree, "run.sh"); !strings.HasPrefix(listing, "100755 blob ") {
		t.Errorf("git.Repo.Tree: run.sh listed as %q, want mode 100755", listing)
	}

	if _, err := repo.Tree("-x", nil); err == nil {
		t.Errorf("git.Repo.Tree of a revision that looks like an option must fail")
	}
}

func TestRepoChanged(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	write(t, dir, "b.txt", "b\n")
	write(t, dir, "sub/c.txt", "c\n")
	head := commitAll(t, dir, "base")
	repo := git.New(dir, nil)
	before := treeOf(t, dir, "HEAD")

	if got, err := repo.Changed(before, before); err != nil || len(got) != 0 {
		t.Errorf("git.Repo.Changed of equal trees = %v, %v, want none", got, err)
	}

	write(t, dir, "a.txt", "edited\n")                             // modified
	if err := os.Remove(filepath.Join(dir, "b.txt")); err != nil { // deleted ...
		t.Fatal(err)
	}
	write(t, dir, "e.txt", "b\n")     // ... and added with the same bytes: no rename detection
	write(t, dir, "sub/n/d.txt", "d") // added in a new folder
	after, err := repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Changed(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.txt", "b.txt", "e.txt", "sub/n/d.txt"}; !slices.Equal(got, want) {
		t.Errorf("git.Repo.Changed = %v, want %v", got, want)
	}
	if got, err := repo.Changed("-x", after); err == nil {
		t.Errorf("git.Repo.Changed with an option-like revision = %v, want an error", got)
	}
}

func TestRepoIndexPath(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "base")
	repo := git.New(dir, nil)

	path, err := repo.IndexPath()
	if err != nil {
		t.Fatal(err)
	}
	gitDir := strings.TrimSpace(runGit(t, dir, "rev-parse", "--absolute-git-dir"))
	if path != filepath.Join(gitDir, "index") || !filepath.IsAbs(path) {
		t.Errorf("git.Repo.IndexPath = %q, want the absolute %q", path, filepath.Join(gitDir, "index"))
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("git.Repo.IndexPath: %s is not the index file: %v", path, err)
	}
	write(t, dir, "b.txt", "b\n")
	runGit(t, dir, "add", "b.txt")
	after, err := os.ReadFile(path)
	if err != nil || bytes.Equal(before, after) {
		t.Errorf("git.Repo.IndexPath: staging must change the bytes at %s", path)
	}

	// A linked worktree has its own index.
	linked := filepath.Join(t.TempDir(), "linked")
	runGit(t, dir, "worktree", "add", "-q", "-b", "other", linked)
	lp, err := git.New(linked, nil).IndexPath()
	if err != nil || lp == path || !strings.HasSuffix(lp, filepath.Join("worktrees", "linked", "index")) {
		t.Errorf("git.Repo.IndexPath of a linked worktree = %q, %v, want its own index", lp, err)
	}
	if _, err := git.New(t.TempDir(), append(os.Environ(), "GIT_CEILING_DIRECTORIES="+os.TempDir())).IndexPath(); err == nil {
		t.Errorf("git.Repo.IndexPath outside a repository must fail")
	}
}

func TestRepoObserving(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "base")
	repo := git.New(dir, nil)
	indexPath := filepath.Join(dir, ".git", "index")

	view := repo.Observing()
	if view == repo || view.Dir != dir {
		t.Errorf("git.Repo.Observing must return a view of the same folder, not the receiver")
	}

	// Make the stat cache stale without changing content: a plain status refreshes the
	// index on disk, an observing one must leave it as it is.
	touch := func() {
		old := time_(2001)
		if err := os.Chtimes(filepath.Join(dir, "a.txt"), old, old); err != nil {
			t.Fatal(err)
		}
	}
	touch()
	before, _ := os.ReadFile(indexPath)
	if got, err := view.ChangedPaths(); err != nil || len(got) != 0 {
		t.Fatalf("git.Repo.Observing: ChangedPaths = %v, %v, want none", got, err)
	}
	if after, _ := os.ReadFile(indexPath); !bytes.Equal(before, after) {
		t.Errorf("git.Repo.Observing: a read-only observation rewrote the index")
	}
	if got, err := view.At(dir).ChangedPaths(); err != nil || len(got) != 0 {
		t.Fatalf("git.Repo.Observing: At on a view: %v, %v", got, err)
	}
	if after, _ := os.ReadFile(indexPath); !bytes.Equal(before, after) {
		t.Errorf("git.Repo.Observing: a view made by At must stay read-only, but the index was rewritten")
	}
	if got, err := repo.ChangedPaths(); err != nil || len(got) != 0 {
		t.Fatalf("git.Repo.Observing: plain ChangedPaths = %v, %v", got, err)
	}
	if after, _ := os.ReadFile(indexPath); bytes.Equal(before, after) {
		t.Errorf("git.Repo.Observing: the plain Repo refreshes the index, so this test cannot tell a read-only view from it")
	}
}

func TestRepoMaterialize(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, ".gitignore", "*.log\n")
	write(t, dir, "a.txt", "alpha\n")
	write(t, dir, "bin/run.sh", "#!/bin/sh\necho run\n")
	if err := os.Chmod(filepath.Join(dir, "bin", "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.txt", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	head := commitAll(t, dir, "base")
	write(t, dir, "new/u.txt", "untracked\n")
	write(t, dir, "x.log", "ignored\n")
	repo := git.New(dir, nil)
	tree, err := repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "verification")
	if _, err := repo.Materialize(tree, target); err != nil {
		t.Fatalf("git.Repo.Materialize: %v", err)
	}
	for rel, want := range map[string]string{"a.txt": "alpha\n", "bin/run.sh": "#!/bin/sh\necho run\n", "new/u.txt": "untracked\n", ".gitignore": "*.log\n"} {
		if b, err := os.ReadFile(filepath.Join(target, rel)); err != nil || string(b) != want {
			t.Errorf("git.Repo.Materialize: %s = %q, %v, want %q", rel, b, err, want)
		}
	}
	if st, err := os.Stat(filepath.Join(target, "bin", "run.sh")); err != nil || st.Mode().Perm()&0o111 == 0 {
		t.Errorf("git.Repo.Materialize: bin/run.sh lost its executable bit: %v", err)
	}
	if to, err := os.Readlink(filepath.Join(target, "link")); err != nil || to != "a.txt" {
		t.Errorf("git.Repo.Materialize: link -> %q, %v, want a symlink to a.txt", to, err)
	}
	if _, err := os.Lstat(filepath.Join(target, "x.log")); err == nil {
		t.Errorf("git.Repo.Materialize wrote the ignored x.log, which is not part of the tree")
	}
	// The directory is its own repository whose HEAD and index hold exactly the tree.
	if got := treeOf(t, target, "HEAD"); got != tree {
		t.Errorf("git.Repo.Materialize: HEAD's tree in the new repository = %s, want %s", got, tree)
	}
	if out := runGit(t, target, "status", "--porcelain"); out != "" {
		t.Errorf("git.Repo.Materialize: the new repository is not clean: %q", out)
	}
	if got := git.New(target, nil).Head(); got == "" || got == head {
		t.Errorf("git.Repo.Materialize: the new repository has Head %q, want a commit of its own (not %s)", got, head)
	}

	if _, err := repo.Materialize("-x", filepath.Join(t.TempDir(), "never")); err == nil {
		t.Errorf("git.Repo.Materialize of an option-like revision must fail")
	}
}

// conflict leaves dir in the middle of the named operation, which Git stopped on a conflict.
func conflict(t *testing.T, op string) string {
	t.Helper()
	dir := newRepo(t)
	write(t, dir, "a.txt", "base\n")
	commitAll(t, dir, "base")
	runGit(t, dir, "checkout", "-q", "-b", "other")
	write(t, dir, "a.txt", "other\n")
	other := commitAll(t, dir, "other")
	runGit(t, dir, "checkout", "-q", "main")
	write(t, dir, "a.txt", "main\n")
	mainHead := commitAll(t, dir, "main")
	var args []string
	switch op {
	case "merge":
		args = []string{"merge", "other"}
	case "cherry-pick":
		args = []string{"cherry-pick", other}
	case "rebase":
		args = []string{"rebase", "other"}
	case "revert":
		write(t, dir, "a.txt", "later\n")
		commitAll(t, dir, "later")
		args = []string{"revert", "--no-edit", mainHead}
	}
	if out, status := gitStatus(t, dir, args...); status == 0 {
		t.Fatalf("git %v did not stop on a conflict:\n%s", args, out)
	}
	return dir
}

func TestRepoOperationInProgress(t *testing.T) {
	clean := newRepo(t)
	write(t, clean, "a.txt", "a\n")
	commitAll(t, clean, "base")
	if got, err := git.New(clean, nil).OperationInProgress(); err != nil || got != "" {
		t.Errorf("git.Repo.OperationInProgress of a quiet repository = %q, %v, want empty", got, err)
	}
	for _, op := range []string{"merge", "cherry-pick", "rebase", "revert"} {
		repo := git.New(conflict(t, op), nil)
		if got, err := repo.OperationInProgress(); err != nil || got != op {
			t.Errorf("git.Repo.OperationInProgress during %s = %q, %v, want %q", op, got, err, op)
		}
		// The read-only view of the preflight sees it too.
		if got, err := repo.Observing().OperationInProgress(); err != nil || got != op {
			t.Errorf("git.Repo.OperationInProgress through Observing during %s = %q, %v, want %q", op, got, err, op)
		}
	}
	if _, err := git.New(t.TempDir(), append(os.Environ(), "GIT_CEILING_DIRECTORIES="+os.TempDir())).OperationInProgress(); err == nil {
		t.Errorf("git.Repo.OperationInProgress outside a repository must fail")
	}
}

func TestRepoUnmergedListing(t *testing.T) {
	// Approve refuses a checkout with conflicting paths and names them from this listing.
	repo := git.New(conflict(t, "merge"), nil)
	out, err := repo.Observing().OK("ls-files", "--unmerged", "--format=%(path)", "-z")
	if err != nil {
		t.Fatalf("git.Repo.OK ls-files --unmerged: %v", err)
	}
	for _, p := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		if p != "a.txt" {
			t.Errorf("git.Repo.OK ls-files --unmerged lists %q, want only a.txt NUL-separated: %q", p, out)
		}
	}
	if out == "" {
		t.Errorf("git.Repo.OK ls-files --unmerged is empty during a conflicted merge")
	}
	clean := newRepo(t)
	if out, err := git.New(clean, nil).Observing().OK("ls-files", "--unmerged", "--format=%(path)", "-z"); err != nil || out != "" {
		t.Errorf("git.Repo.OK ls-files --unmerged of a clean repository = %q, %v, want empty", out, err)
	}
}

func TestScratchCheckout(t *testing.T) {
	// Refine works in a shared clone: it reads the frozen tree into the clone's index, lets
	// files change there and compares the trees.
	src := newRepo(t)
	write(t, src, "a.txt", "a\n")
	write(t, src, "b.txt", "b\n")
	head := commitAll(t, src, "base")
	repo := git.New(src, nil)
	tree, err := repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(t.TempDir(), "scratch")
	if _, err := repo.OK("clone", "--quiet", "--shared", "--no-checkout", "--", src, scratch); err != nil {
		t.Fatal(err)
	}
	scratchR := repo.At(scratch)
	if _, err := scratchR.OK("read-tree", tree); err != nil {
		t.Fatalf("git.Repo.At: read-tree in the clone: %v", err)
	}
	write(t, scratch, "a.txt", "a\n")
	write(t, scratch, "b.txt", "changed\n")
	write(t, scratch, "c.txt", "new\n")
	after, err := scratchR.Tree(head, nil)
	if err != nil {
		t.Fatalf("git.Repo.At: Tree in the clone: %v", err)
	}
	changed, err := scratchR.Changed(tree, after)
	if err != nil || !slices.Equal(changed, []string{"b.txt", "c.txt"}) {
		t.Errorf("git.Repo.At: Changed in the clone = %v, %v, want [b.txt c.txt]", changed, err)
	}
	excludes := filepath.Join(t.TempDir(), "excludes")
	if _, err := scratchR.OK("config", "core.excludesFile", excludes); err != nil {
		t.Fatal(err)
	}
	if res, err := scratchR.Run("config", "--path", "--get", "core.excludesFile"); err != nil || res.Stdout != excludes+"\n" {
		t.Errorf("git.Repo.At: the clone's core.excludesFile = %q, %v, want %q", res.Stdout, err, excludes)
	}
	if _, err := scratchR.OK("read-tree", "--reset", "-u", tree); err != nil {
		t.Fatalf("git.Repo.At: read-tree --reset -u: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(scratch, "b.txt")); string(b) != "b\n" {
		t.Errorf("git.Repo.At: read-tree --reset -u left b.txt = %q, want the tree's bytes", b)
	}
	if got := repo.Head(); got != head {
		t.Errorf("git.Repo.At: the source Head = %q, want %q unchanged", got, head)
	}
}

func samePath(a, b string) bool {
	ra, ea := filepath.EvalSymlinks(filepath.Dir(a))
	rb, eb := filepath.EvalSymlinks(filepath.Dir(b))
	return errors.Join(ea, eb) == nil && filepath.Join(ra, filepath.Base(a)) == filepath.Join(rb, filepath.Base(b))
}
