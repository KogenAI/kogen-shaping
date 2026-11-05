package shape

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

func TestStaticCheckSelection(t *testing.T) {
	parallel(t)
	for run, want := range map[string]bool{
		"GOFLAGS=-trimpath go test ./...": true,
		"cd app && npm test":              true,
		"npm run test":                    true,
		"yarn test --ci":                  true,
		"pytest -q":                       true,
		"make && (cargo test)":            true,
		"echo `jest`":                     true,
		`test -z "$(gofmt -l .)"`:         false,
		"go build ./...":                  false,
		"go vet ./...":                    false,
		"npm run lint":                    false,
		"echo pytester":                   false,
		"go testdata":                     false,
	} {
		if got := isTestSuiteCheck(run); got != want {
			t.Errorf("%q: test suite %v, want %v", run, got, want)
		}
	}
}

func TestStaticCheckUnavailable(t *testing.T) {
	parallel(t)
	for _, tc := range []struct {
		name   string
		res    sys.Result
		reason string
	}{
		{"timeout", sys.Result{Status: 137, TimedOut: true}, "timed out after 120 s"},
		{"dash not found, exit 0", sys.Result{Output: "/bin/sh: gofmt: command not found\n"}, "gofmt not found"},
		{"dash with line, exit 1", sys.Result{Status: 1, Output: "x\n/bin/sh: 1: golint: not found\n"}, "golint not found"},
		{"bash with line", sys.Result{Status: 127, Output: "bash: line 3: vet: command not found"}, "vet not found"},
		{"exit 127", sys.Result{Status: 127, Output: "boom"}, "exit 127"},
		{"exit 126", sys.Result{Status: 126}, "exit 126"},
		{"plain failure", sys.Result{Status: 1, Output: "gofmt: not found in the output of something else"}, ""},
		{"pass", sys.Result{}, ""},
	} {
		reason, ok := staticUnavailableReason(tc.res)
		if reason != tc.reason || ok != (tc.reason != "") {
			t.Errorf("%s: got %q, %v, want %q", tc.name, reason, ok, tc.reason)
		}
	}
}

func TestStaticCheckNamedPaths(t *testing.T) {
	parallel(t)
	const p = "features/steps/a.go"
	for _, tc := range []struct {
		text  string
		named bool
		loc   string
	}{
		{p, true, ""},
		{p + ":14:3: bad", true, ":14:3"},
		{p + ":14: bad", true, ":14"},
		{p + ": bad", true, ""},
		{"./" + p + ":2:1", true, ":2:1"},
		{"/tmp/x/" + p + ":9", true, ":9"},
		{"+ test -z " + p + "\n", true, ""},
		{"xfeatures/steps/a.go", false, ""},
		{"my-" + p, false, ""},
		{"1" + p, false, ""},
		{".features/steps/a.go", false, ""},
		{p + "x", false, ""},
		{p + "_test", false, ""},
		{p + "-old", false, ""},
		{p + "/more", false, ""},
		{p + ".bak", false, ""},
		{p + ". Next", true, ""},
		{p + ".", true, ""},
		{p + ":x", true, ""},
		{p + "x " + p + ":3", true, ":3"},
	} {
		named, loc := occurrence(tc.text, p)
		if named != tc.named || loc != tc.loc {
			t.Errorf("%q: got %v %q, want %v %q", tc.text, named, loc, tc.named, tc.loc)
		}
	}
	paths := []string{"z.go", "a/b.go", "m.go"}
	got := namedPaths(paths, "z.go a/b.go:3:4", "m.go")
	if !slices.Equal(got, []string{"a/b.go", "m.go", "z.go"}) {
		t.Errorf("named paths %v", got)
	}
	entries := staticEntries([]string{"a/b.go", "m.go", "z.go"}, "a/b.go: x\nz.go:7:8: y", "a/b.go:1:1\nm.go:5\nz.go:9")
	if want := []string{"a/b.go:1:1", "m.go:5", "z.go:7:8"}; !slices.Equal(entries, want) {
		t.Errorf("entries %v, want %v", entries, want)
	}
}

func TestStaticCheckFeedback(t *testing.T) {
	parallel(t)
	one, two := 1, 2
	run := &staticRun{Outcome: staticRefused, Checks: []staticCheck{
		{Name: "layout", Run: `test -z "$(gofmt -l .)"`, Status: staticAttributable, Exit: &one, Paths: []string{"h/main.go"}},
		{Name: "pre", Run: "false", Status: staticPreexisting, Exit: &one, Paths: []string{"old.go"}, Output: "old.go\n"},
		{Name: "vet", Run: "go vet ./...", Status: staticAttributable, Exit: &two, Paths: []string{}, Output: "# pkg\nh/main.go:4:2: bad"},
		{Name: "tests", Run: "go test ./...", Status: staticNotRun, Paths: []string{}},
	}}
	headline, problems := run.refusal()
	if !strings.HasPrefix(headline, "Static checks: the shaped files fail 2 configured static check(s); nothing was published. ") ||
		!strings.HasSuffix(headline, "Resubmitting this package unchanged is refused again.") {
		t.Errorf("headline %q", headline)
	}
	want := []string{
		`- check layout: exit 1; files: h/main.go; command: test -z "$(gofmt -l .)"`,
		"  | (no output)",
		"- check vet: exit 2; files: (not named by the check); command: go vet ./...",
		"  | # pkg",
		"  | h/main.go:4:2: bad",
	}
	if !slices.Equal(problems, want) {
		t.Errorf("problems\n%s\nwant\n%s", strings.Join(problems, "\n"), strings.Join(want, "\n"))
	}
}

// staticFx is the approve fixture with its Intent's step file and app.txt checked by the
// commands of cfg.
func staticFx(t *testing.T, checks string) *approveFx {
	t.Helper()
	f := newApproveFx(t)
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n"+checks+"  acceptance:\n    run: sh accept.sh\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "checks")
	return f
}

func TestStaticChecksAttribution(t *testing.T) {
	parallel(t)
	const step = "features/steps/fixture_steps.js"
	f := staticFx(t, `    - name: steps
      run: '! grep -q BAD `+step+`'
    - name: stale
      run: '! grep -q BAD app.txt'
    - name: located
      run: 'if [ -f `+step+` ]; then echo "`+step+`:3:5: bad"; fi; exit 1'
    - name: suite
      run: 'go test ./...'
    - name: absent
      run: 'exit 127'
    - name: restored
      run: 'grep -q BAD `+step+`'
`)
	f.write(step, "// BAD\n")
	f.write("app.txt", "BAD\n")
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	s, _, _ := f.session()
	var progress []string
	head := f.Repo.Head()
	tree, err := f.Repo.Tree(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(l string) { progress = append(progress, l) })
	run, err := g.run(context.Background(), head, tree, "package")
	if err != nil {
		t.Fatal(err)
	}
	if run.Outcome != staticRefused || run.Head != head || run.Tree != tree || run.SHA256 != "package" {
		t.Fatalf("run %+v", run)
	}
	type got struct {
		status string
		paths  []string
		reason string
	}
	want := map[string]got{
		"steps":    {staticAttributable, []string{step}, ""},
		"stale":    {staticPreexisting, []string{"app.txt"}, ""},
		"located":  {staticAttributable, []string{step + ":3:5"}, ""},
		"suite":    {staticNotRun, []string{}, ""},
		"absent":   {staticUnavailable, []string{}, "exit 127"},
		"restored": {staticPassed, []string{}, ""},
	}
	for _, c := range run.Checks {
		w := want[c.Name]
		if c.Status != w.status || !slices.Equal(c.Paths, w.paths) || c.Reason != w.reason {
			t.Errorf("check %s: %s %v %q, want %s %v %q", c.Name, c.Status, c.Paths, c.Reason, w.status, w.paths, w.reason)
		}
		if (c.Status == staticNotRun) != (c.Exit == nil) {
			t.Errorf("check %s: exit %v", c.Name, c.Exit)
		}
	}
	if !strings.Contains(strings.Join(run.Checks[0:1][0].Paths, ""), step) || run.Checks[0].Trace == "" || run.Checks[0].Output != "" {
		t.Errorf("steps check: trace %q output %q", run.Checks[0].Trace, run.Checks[0].Output)
	}
	if rel, err := filepath.Rel(f.Root, run.Dir); err != nil || !strings.HasPrefix(rel, "..") {
		t.Errorf("the copy %s is inside the checkout", run.Dir)
	}
	if _, err := os.Stat(run.Dir); !os.IsNotExist(err) {
		t.Errorf("the copy %s was not removed: %v", run.Dir, err)
	}
	var files []string
	for _, sf := range run.Files {
		files = append(files, sf.Path)
	}
	if !slices.Contains(files, step) || !slices.Contains(files, project.IntentMDPath(approveSlug)) || slices.Contains(files, "app.txt") || !slices.IsSorted(files) {
		t.Errorf("shaped files %v", files)
	}
	if want := "kogen: running static checks on the shaped files in a fresh copy: steps, stale, located, absent, restored"; progress[0] != want {
		t.Errorf("first progress line %q", progress[0])
	}
	for _, line := range []string{
		"kogen: static check steps failed (exit 1) because of the shaped files: " + step,
		"kogen: static check stale failed (exit 1) without the shaped files too: pre-existing failure in app.txt; the shaped files are not established by this check",
		"kogen: static check suite not run: it runs the test suite",
		"kogen: static check absent unavailable: exit 127; the shaped files are not established by this check",
		"kogen: static check restored passed",
	} {
		if !slices.Contains(progress, line) {
			t.Errorf("no progress line %q in %q", line, progress)
		}
	}
	if f.read(step) != "// BAD\n" || f.read("app.txt") != "BAD\n" {
		t.Error("the checkout changed")
	}
}

// A check that deletes a shaped file in the copy and fails is judged, and the file is written
// back for the next check.
func TestStaticChecksShapedFileDeletedByCheck(t *testing.T) {
	parallel(t)
	const step = "features/steps/fixture_steps.js"
	f := staticFx(t, "    - name: clean\n      run: 'rm -f "+step+"; exit 1'\n    - name: present\n      run: 'test -f "+step+"'\n")
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
	run, err := g.run(context.Background(), head, tree, "package")
	if err != nil {
		t.Fatal(err)
	}
	if run.Checks[0].Status != staticPreexisting || run.Checks[1].Status != staticPassed || run.Outcome != staticUnknown {
		t.Fatalf("run %+v", run)
	}
}

func TestStaticChecksNoneApply(t *testing.T) {
	parallel(t)
	f := staticFx(t, "    - name: suite\n      run: 'npm test'\n")
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	s, _, _ := f.session()
	var progress []string
	g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(l string) { progress = append(progress, l) })
	run, err := g.run(context.Background(), "h", "t", "p")
	if err != nil || run.Outcome != staticNone || run.Dir != "" || progress[0] != "kogen: no configured static check applies to the shaped files" {
		t.Fatalf("run %+v, err %v, progress %q", run, err, progress)
	}
}
