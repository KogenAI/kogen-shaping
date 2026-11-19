package shape

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

func TestStaticCheckSelection(t *testing.T) {
	parallel(t)
	for run, want := range map[string]bool{
		"GOFLAGS=-trimpath go test ./...":     true,
		"cd app && npm test":                  true,
		"npm run test":                        true,
		"yarn test --ci":                      true,
		"pytest -q":                           true,
		"make && (cargo test)":                true,
		"echo `jest`":                         true,
		"bin/rails test":                      true,
		"rails test test/models":              true,
		"bundle exec rake test":               true,
		"RAILS_ENV=test bin/rake test:system": true,
		"cd app && bin/rails test":            true,
		"bin/rails routes":                    false,
		"bin/rubocop":                         false,
		"rake testdata":                       false,
		"echo rails testing":                  false,
		`test -z "$(gofmt -l .)"`:             false,
		"go build ./...":                      false,
		"go vet ./...":                        false,
		"npm run lint":                        false,
		"echo pytester":                       false,
		"go testdata":                         false,

		"make spec":              true,
		"make test":              true,
		"make -j4 check":         true,
		"cd app && make verify":  true,
		"MIX_ENV=test make spec": true,
		"gmake ci":               true,
		"make -C app tests":      true,
		"make lint":              false,
		"make fmt":               false,
		"make specimen":          false,
		"make test-unit":         false,
		"make -C spec lint":      false,

		"python3 reproduce/validate_repo.py":             true,
		"python3 reproduce/validate_release.py --strict": true,
		"cd reproduce && python validate_all.py":         true,
		"CI=1 bash ./validate_repo.py":                   true,
		"python3 reproduce/validate_round.py":            false,
		"python3 reproduce/validate_spec_snapshot.py":    false,
		"python3 reproduce/check_shaped_files.py":        false,
		"python3 reproduce/validate_repo.py.bak":         false,
		"python3 reproduce/validate_repo_helpers.py":     false,
	} {
		if got := isTestSuiteCheck(run); got != want {
			t.Errorf("%q: test suite %v, want %v", run, got, want)
		}
	}
}

// TestStaticChecksSkipFullValidator: the gate does not run a whole-repository validator script,
// reports it not_run, and still attributes and refuses on the other check.
func TestStaticChecksSkipFullValidator(t *testing.T) {
	parallel(t)
	const step = "features/steps/fixture_steps.js"
	var mu sync.Mutex
	var ran []string
	var validator string
	record := func(command string, res sys.Result) string {
		fakeStaticChecks.Store(command, func(context.Context) (sys.Result, error) {
			mu.Lock()
			ran = append(ran, command)
			first := len(ran) == 1 || command == validator
			mu.Unlock()
			if first {
				return res, nil
			}
			return sys.Result{}, nil
		})
		t.Cleanup(func() { fakeStaticChecks.Delete(command) })
		return command
	}
	tag := nonWord.ReplaceAllString(t.Name(), "-")
	validator = record("KOGEN_FAKE="+tag+" python3 reproduce/validate_repo.py", sys.Result{})
	privacy := record("true kogen-fake-check-"+tag, sys.Result{Status: 1, Output: step + ":2:1: private\n"})
	f := staticFx(t, "    - name: validate-repo\n      run: '"+validator+"'\n    - name: privacy\n      run: '"+privacy+"'\n")
	f.write(step, "// BAD\n")
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
	mu.Lock()
	defer mu.Unlock()
	if len(ran) == 0 || slices.ContainsFunc(ran, func(c string) bool { return c != privacy }) {
		t.Errorf("commands given to the seam %q, want only %q", ran, privacy)
	}
	if len(run.Checks) != 2 || run.Checks[0].Status != staticNotRun || run.Checks[0].Exit != nil {
		t.Fatalf("checks %+v", run.Checks)
	}
	if c := run.Checks[1]; c.Status != staticAttributable || !slices.Contains(c.Paths, step+":2:1") {
		t.Errorf("privacy check %+v", c)
	}
	if run.Outcome != staticRefused {
		t.Errorf("outcome %s", run.Outcome)
	}
	if !slices.Contains(progress, "kogen: static check validate-repo not run: it runs the test suite") {
		t.Errorf("progress %q", progress)
	}
}

// TestStaticChecksSkipMakeSuite: the gate does not run a whole-project make target, reports it
// not_run, and still attributes and refuses on the other check.
func TestStaticChecksSkipMakeSuite(t *testing.T) {
	parallel(t)
	const step = "features/steps/fixture_steps.js"
	var mu sync.Mutex
	var ran []string
	record := func(command string, res sys.Result) string {
		fakeStaticChecks.Store(command, func(context.Context) (sys.Result, error) {
			mu.Lock()
			ran = append(ran, command)
			first := len(ran) == 1
			mu.Unlock()
			if first {
				return res, nil
			}
			return sys.Result{}, nil
		})
		t.Cleanup(func() { fakeStaticChecks.Delete(command) })
		return command
	}
	tag := nonWord.ReplaceAllString(t.Name(), "-")
	suite := record("make spec # kogen-fake-"+tag, sys.Result{})
	vet := record("true kogen-fake-check-"+tag, sys.Result{Status: 1, Output: step + ":2:1: bad\n"})
	f := staticFx(t, "    - name: spec\n      run: '"+suite+"'\n    - name: vet\n      run: '"+vet+"'\n")
	f.write(step, "// BAD\n")
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
	mu.Lock()
	defer mu.Unlock()
	if len(ran) == 0 || slices.ContainsFunc(ran, func(c string) bool { return c != vet }) {
		t.Errorf("commands given to the seam %q, want only %q", ran, vet)
	}
	if len(run.Checks) != 2 || run.Checks[0].Status != staticNotRun || run.Checks[0].Exit != nil {
		t.Fatalf("checks %+v", run.Checks)
	}
	if c := run.Checks[1]; c.Status != staticAttributable || !slices.Contains(c.Paths, step+":2:1") {
		t.Errorf("vet check %+v", c)
	}
	if run.Outcome != staticRefused {
		t.Errorf("outcome %s", run.Outcome)
	}
	if !slices.Contains(progress, "kogen: static check spec not run: it runs the test suite") {
		t.Errorf("progress %q", progress)
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
      run: 'if grep -q BAD app.txt; then echo "app.txt"; exit 1; fi'
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

// TestStaticBaseRunReused: the base run of a failing check is computed once per check name,
// command and base tree, and a later gate runner (the next round, or Approve) reuses it.
func TestStaticBaseRunReused(t *testing.T) {
	parallel(t)
	f := staticFx(t, `    - name: stale
      run: '! grep -q BAD app.txt'
`)
	f.write("app.txt", "BAD\n")
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
	round := func(cfg *project.Config) (*staticRun, string) {
		var progress []string
		g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(l string) { progress = append(progress, l) })
		run, err := g.run(context.Background(), head, tree, "package")
		if err != nil {
			t.Fatal(err)
		}
		return run, strings.Join(progress, "\n")
	}
	const mark = "without the shaped files too (base result reused): pre-existing failure"
	first, out := round(cfg)
	if first.Checks[0].Status != staticPreexisting || strings.Contains(out, "reused") {
		t.Fatalf("round 1: %s\n%s", first.Checks[0].Status, out)
	}
	second, out := round(cfg)
	if second.Checks[0].Status != staticPreexisting || !strings.Contains(out, mark) {
		t.Fatalf("round 2: %s\n%s", second.Checks[0].Status, out)
	}
	renamed := &project.Config{Checks: []project.Check{{Name: "renamed", Run: cfg.Checks[0].Run}}}
	if _, out := round(renamed); strings.Contains(out, "reused") {
		t.Errorf("a changed name reused the base result:\n%s", out)
	}
	changed := &project.Config{Checks: []project.Check{{Name: "stale", Run: cfg.Checks[0].Run + " && true"}}}
	if _, out := round(changed); strings.Contains(out, "reused") {
		t.Errorf("a changed command reused the base result:\n%s", out)
	}
}

// TestStaticChecksPreexistingSkipsTracedRerun: a check that fails identically without the shaped
// files runs twice in the first round and once in the next (the base result is reused), never
// under set -x, and is recorded pre-existing with the paths of its own output and no trace; a check
// that passes without the shaped files and fails with them still runs the traced rerun.
func TestStaticChecksPreexistingSkipsTracedRerun(t *testing.T) {
	parallel(t)
	const step = "features/steps/fixture_steps.js"
	marker := filepath.Join(t.TempDir(), "old-marker")
	// The traced rerun prefixes the check text with set -x, which shows in $-.
	f := staticFx(t, `    - name: old
      run: 'case "$-" in *x*) echo traced >> `+marker+`;; *) echo plain >> `+marker+`;; esac; echo "app.txt:3:1: old"; exit 1'
    - name: new
      run: 'if [ -f `+step+` ]; then echo "`+step+`:2:1: bad"; exit 1; fi'
`)
	f.write(step, "// BAD\n")
	f.write("app.txt", "BAD\n")
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
	round := func() *staticRun {
		g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(string) {})
		run, err := g.run(context.Background(), head, tree, "package")
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	runs := func() []string {
		b, err := os.ReadFile(marker)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Fields(string(b))
	}
	first := round()
	if got := runs(); !slices.Equal(got, []string{"plain", "plain"}) {
		t.Errorf("round 1 runs of the pre-existing check %v, want the shaped and the base run, both plain", got)
	}
	old, added := first.Checks[0], first.Checks[1]
	if old.Status != staticPreexisting || !slices.Equal(old.Paths, []string{"app.txt:3:1"}) || old.Trace != "" {
		t.Errorf("round 1 old: %s %v trace %q", old.Status, old.Paths, old.Trace)
	}
	if added.Status != staticAttributable || added.Trace == "" || first.Outcome != staticRefused {
		t.Errorf("round 1 new: %s trace %q, outcome %s", added.Status, added.Trace, first.Outcome)
	}
	second := round()
	if got := runs(); !slices.Equal(got, []string{"plain", "plain", "plain"}) {
		t.Errorf("after round 2 runs %v, want one more plain run (the base result is reused)", got)
	}
	if old := second.Checks[0]; old.Status != staticPreexisting || !slices.Equal(old.Paths, []string{"app.txt:3:1"}) || old.Trace != "" {
		t.Errorf("round 2 old: %s %v trace %q", old.Status, old.Paths, old.Trace)
	}
}

// TestStaticChecksRecordTimes: each check records the wall time of its run on the candidate and of
// the base run that executed in this gate run, the run records the time of the copy, a check that
// was not run or whose base run did not happen records 0, a later round that reuses the base result
// records 0 for it, and the recorded result file carries the same fields.
func TestStaticChecksRecordTimes(t *testing.T) {
	parallel(t)
	f := staticFx(t, `    - name: slow
      run: 'sleep 0.05'
    - name: stale
      run: 'sleep 0.05; echo "stale: app.txt"; exit 1'
    - name: bad
      run: 'sleep 0.05; echo "app.txt:1:1: bad"; exit 1'
    - name: suite
      run: 'go test ./...'
`)
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
	round := func() *staticRun {
		g := newStaticGateRunner(s, f.Repo, &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}, cfg, f.Root, approveSlug, func(string) {})
		run, err := g.run(context.Background(), head, tree, "package")
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	first := round()
	if first.CopyMS <= 0 {
		t.Errorf("copy_ms %d, want > 0", first.CopyMS)
	}
	for i, want := range []struct{ wall, base bool }{{true, false}, {true, true}, {true, true}, {false, false}} {
		c := first.Checks[i]
		if (c.WallMS > 0) != want.wall || (c.BaseWallMS > 0) != want.base {
			t.Errorf("round 1 check %s: wall_ms %d, base_wall_ms %d; want wall>0 %v, base>0 %v", c.Name, c.WallMS, c.BaseWallMS, want.wall, want.base)
		}
	}
	// A pre-existing check does not run the traced re-run, so its wall_ms is the plain run only.
	if w := first.Checks[1].WallMS; w < 50 {
		t.Errorf("stale wall_ms %d, want at least the plain run (50)", w)
	}
	second := round()
	for _, c := range second.Checks[1:3] {
		if c.WallMS <= 0 || c.BaseWallMS != 0 {
			t.Errorf("round 2 check %s: wall_ms %d, base_wall_ms %d; the reused base result has no base time", c.Name, c.WallMS, c.BaseWallMS)
		}
	}

	if err := writeStaticRecord(f.Root, approveSlug, first); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(staticRecordPath(f.Root, approveSlug))
	if err != nil {
		t.Fatal(err)
	}
	var rec struct {
		CopyMS int64 `json:"copy_ms"`
		Checks []struct {
			WallMS     int64 `json:"wall_ms"`
			BaseWallMS int64 `json:"base_wall_ms"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.CopyMS != first.CopyMS || len(rec.Checks) != 4 || rec.Checks[1].WallMS != first.Checks[1].WallMS || rec.Checks[1].BaseWallMS != first.Checks[1].BaseWallMS {
		t.Errorf("the recorded result %s does not carry the timings of the run", b)
	}
}

// A check whose plain run and traced re-run fail differently without naming a shaped file is flaky and
// leaves the outcome unknown; a failure that repeats, or names a shaped file, is still refused.
func TestStaticChecksFlakyNotAttributable(t *testing.T) {
	parallel(t)
	const step = "features/steps/fixture_steps.js"
	gate := func(checks func(count string) string) (*staticRun, []string) {
		f := staticFx(t, checks(filepath.Join(t.TempDir(), "count")))
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
		return run, progress
	}
	// Each execution in the shaped copy fails with another message (digits differ too); the base run (no shaped files, so no features/steps) passes without counting.
	run, progress := gate(func(count string) string {
		return `    - name: unsteady
      run: 'if test -z "$(ls features/steps)"; then exit 0; fi; n=$(cat ` + count + ` 2>/dev/null || echo 0); n=$((n+1)); echo $n > ` + count + `; case $n in 1) echo "  Failure: alpha_test"; echo "3 errors"; exit 1;; 2) echo "Error: beta_test"; echo "7 errors"; exit 1;; esac; exit 0'
`
	})
	c := run.Checks[0]
	if c.Status != staticFlaky || run.Outcome != staticUnknown || c.Exit == nil || *c.Exit != 1 || len(c.Paths) != 0 {
		t.Fatalf("unsteady: %+v outcome %s", c, run.Outcome)
	}
	if want := []string{"# errors", "Failure: alpha_test"}; !slices.Equal(c.Failing, want) {
		t.Errorf("failing %q, want %q", c.Failing, want)
	}
	if want := []string{"# errors", "Error: beta_test"}; !slices.Equal(c.TracedFailing, want) {
		t.Errorf("traced failing %q, want %q", c.TracedFailing, want)
	}
	if want := "kogen: static check unsteady failed (exit 1) with different failures on two runs of the same shaped files: not attributable; the shaped files are not established by this check"; !slices.Contains(progress, want) {
		t.Errorf("no progress line %q in %q", want, progress)
	}
	if b, _ := json.Marshal(run.Checks[0]); !strings.Contains(string(b), `"failing":["# errors","Failure: alpha_test"]`) || !strings.Contains(string(b), `"traced_failing"`) {
		t.Errorf("check json %s", b)
	}

	run, _ = gate(func(count string) string {
		return `    - name: identical
      run: 'if test -n "$(ls features/steps)"; then echo "Failure: same_test"; exit 1; fi'
    - name: located
      run: 'if [ -f ` + step + ` ]; then echo "` + step + `:3:5: bad"; fi; exit 1'
    - name: tracedpath
      run: 'if test -z "$(ls features/steps)"; then exit 0; fi; n=$(cat ` + count + ` 2>/dev/null || echo 0); n=$((n+1)); echo $n > ` + count + `; case $n in 1) echo "Failure: alpha_test"; exit 1;; 2) echo "` + step + `:3:5: bad"; exit 1;; esac; exit 0'
    - name: unsteady
      run: 'if test -z "$(ls features/steps)"; then exit 0; fi; n=$(cat ` + count + `2 2>/dev/null || echo 0); n=$((n+1)); echo $n > ` + count + `2; case $n in 1) echo "Failure: alpha_test"; exit 1;; 2) echo "Error: beta_test"; exit 1;; esac; exit 0'
`
	})
	for i, w := range []string{staticAttributable, staticAttributable, staticAttributable, staticFlaky} {
		c := run.Checks[i]
		if c.Status != w {
			t.Errorf("check %s: %s, want %s", c.Name, c.Status, w)
		}
		if w != staticFlaky && (c.Failing != nil || c.TracedFailing != nil) {
			t.Errorf("check %s records failing sets %q %q", c.Name, c.Failing, c.TracedFailing)
		}
	}
	if run.Outcome != staticRefused {
		t.Errorf("outcome %s, want refused", run.Outcome)
	}
}
