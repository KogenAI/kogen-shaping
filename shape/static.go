package shape

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// The static gate: before a shaped package is published or approved, the repository's configured
// checks (commands.checks) run unchanged in a fresh copy of the exact candidate, and a failure is
// attributed to the shaped files only when it is theirs. A check that fails without them, or that
// cannot run, leaves the shaped files unestablished by it; nothing is waived or approximated.

const (
	staticPassed       = "passed"
	staticAttributable = "attributable"
	staticPreexisting  = "preexisting"
	staticUnavailable  = "unavailable"
	staticNotRun       = "not_run"

	staticChecked = "checked"
	staticUnknown = "unknown"
	staticNone    = "none"
	staticRefused = "refused"

	staticGate = "static_checks"
)

// testSuiteCheck matches a configured check that runs a test suite, which the gate never runs.
var testSuiteCheck = regexp.MustCompile(`(?:^|[\s;&|(\x60])(?:(?:go|cargo|dotnet|mix|deno|bun)\s+test|(?:npm|pnpm|yarn)\s+(?:run\s+)?test|pytest|jest|vitest|mocha|rspec)(?:$|[\s;&|)\x60])`)

// missingProgram matches the line a shell prints for a program it cannot find.
var missingProgram = regexp.MustCompile(`(?m)^(?:\S*/)?(?:sh|bash|dash|zsh|ksh): (?:line [0-9]+: |[0-9]+: )?([^\s:]+): (?:command )?not found\s*$`)

// staticCheck is the result of one configured check.
type staticCheck struct {
	Name   string   `json:"name"`
	Run    string   `json:"run"`
	Status string   `json:"status"`
	Exit   *int     `json:"exit,omitempty"`
	Paths  []string `json:"paths"`
	Reason string   `json:"reason,omitempty"`
	Output string   `json:"output"`
	Trace  string   `json:"trace"`
}

type staticFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// staticRun is the result of one run of the gate on one candidate.
type staticRun struct {
	Outcome string        `json:"outcome"`
	Dir     string        `json:"dir"`
	Head    string        `json:"head"`
	Tree    string        `json:"tree"`
	SHA256  string        `json:"sha256"`
	Files   []staticFile  `json:"files"`
	Checks  []staticCheck `json:"checks"`

	SeededPaths int   `json:"-"` // ignored paths of the checkout the copy started with
	SeededMS    int64 `json:"-"`
	attempt     int   // the submission that ran it (Shape)
}

// staticRecord is .kogen/local/static-checks/<slug>.json.
type staticRecord struct {
	Version int `json:"version"`
	staticRun
}

func staticRecordPath(root, slug string) string {
	return filepath.Join(root, ".kogen/local/static-checks", slug+".json")
}

func writeStaticRecord(root, slug string, run *staticRun) error {
	b, err := json.Marshal(staticRecord{1, *run})
	if err != nil {
		return err
	}
	path := staticRecordPath(root, slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o666)
}

// isTestSuiteCheck reports whether a configured command runs a test suite.
func isTestSuiteCheck(run string) bool { return testSuiteCheck.MatchString(run) }

// staticUnavailableReason is why a finished check says nothing about the files: it timed out, its
// shell could not find a program (whatever the exit status), or it exited 126 or 127.
func staticUnavailableReason(res sys.Result) (string, bool) {
	if res.TimedOut {
		return fmt.Sprintf("timed out after %d s", int(verify.StepTimeout.Seconds())), true
	}
	if m := missingProgram.FindStringSubmatch(res.Output); m != nil {
		return m[1] + " not found", true
	}
	if res.Status == 126 || res.Status == 127 {
		return fmt.Sprintf("exit %d", res.Status), true
	}
	return "", false
}

func isNameByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// occurrence returns whether text names path p, and the first location (":<line>[:<column>]")
// that directly follows an occurrence of it. A name is an occurrence of p whose neighbours do not
// continue a longer path.
func occurrence(text, p string) (named bool, location string) {
	for from := 0; from <= len(text)-len(p); {
		i := strings.Index(text[from:], p)
		if i < 0 {
			break
		}
		i += from
		from = i + 1
		end := i + len(p)
		if i > 0 {
			if b := text[i-1]; isNameByte(b) || b == '_' || b == '-' || b == '.' {
				continue
			}
		}
		if end < len(text) {
			b := text[end]
			if isNameByte(b) || b == '_' || b == '-' || b == '/' {
				continue
			}
			if b == '.' && end+1 < len(text) && isNameByte(text[end+1]) {
				continue
			}
		}
		named = true
		if loc := locationAt(text, end); loc != "" {
			return true, loc
		}
	}
	return named, ""
}

// locationAt is ":<line>" or ":<line>:<column>" at text[at:], "" when there is none.
func locationAt(text string, at int) string {
	digits := func(from int) int {
		n := from
		for n < len(text) && text[n] >= '0' && text[n] <= '9' {
			n++
		}
		return n
	}
	if at >= len(text) || text[at] != ':' {
		return ""
	}
	line := digits(at + 1)
	if line == at+1 {
		return ""
	}
	if line < len(text) && text[line] == ':' {
		if col := digits(line + 1); col > line+1 {
			return text[at:col]
		}
	}
	return text[at:line]
}

// namedPaths are the paths of candidates that text names, sorted.
func namedPaths(candidates []string, texts ...string) []string {
	var out []string
	for _, p := range candidates {
		for _, text := range texts {
			if named, _ := occurrence(text, p); named {
				out = append(out, p)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// staticEntries are the entries of the paths: p:<line>[:<column>] from the first located
// occurrence in the output, else in the trace, else p alone.
func staticEntries(paths []string, output, trace string) []string {
	entries := make([]string, 0, len(paths))
	for _, p := range paths {
		entry := p
		for _, text := range []string{output, trace} {
			if _, loc := occurrence(text, p); loc != "" {
				entry = p + loc
				break
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// staticGateRunner runs the gate for one session and checkout.
type staticGateRunner struct {
	s        *sys.Session
	repo     *git.Repo
	rd       *verify.Readiness
	cfg      *project.Config
	root     string
	slug     string
	progress func(string)
	redact   *verify.Redactor
}

func newStaticGateRunner(s *sys.Session, repo *git.Repo, rd *verify.Readiness, cfg *project.Config, root, slug string, progress func(string)) *staticGateRunner {
	home := s.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return &staticGateRunner{s: s, repo: repo, rd: rd, cfg: cfg, root: root, slug: slug, progress: progress, redact: verify.NewRedactor(s.Env, home)}
}

// bounded is command output as feedback and evidence carry it: redacted, then cut.
func (g *staticGateRunner) bounded(text string) string {
	return verify.Diagnostic(g.redact.Redact(text))
}

// shapedFiles are the regular files under the Intent's folder and its acceptance paths, sorted.
func (g *staticGateRunner) shapedFiles() ([]string, error) {
	var files []string
	dir := filepath.Join(g.root, project.IntentDir(g.slug))
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == dir && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.Type().IsRegular() {
			if rel, err := filepath.Rel(g.root, path); err == nil {
				files = append(files, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if intent, err := project.LoadIntent(g.root, g.slug); err == nil {
		for _, p := range intent.Acceptance {
			if !filepath.IsLocal(p) {
				continue
			}
			if fi, err := os.Lstat(filepath.Join(g.root, p)); err == nil && fi.Mode().IsRegular() {
				files = append(files, p)
			}
		}
	}
	sort.Strings(files)
	return slices.Compact(files), nil
}

// shapedCopy is the shaped files as the copy holds them.
type shapedCopy struct {
	path    string
	content []byte
	mode    fs.FileMode
	sum     string
}

func readShaped(dir string, paths []string) ([]shapedCopy, error) {
	var out []shapedCopy
	for _, p := range paths {
		file := filepath.Join(dir, p)
		fi, err := os.Lstat(file)
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		out = append(out, shapedCopy{p, b, fi.Mode().Perm(), hex.EncodeToString(sum[:])})
	}
	return out, nil
}

// without runs fn with the shaped files removed from dir (empty folders stay), then writes them
// back with their bytes and mode and verifies their SHA-256.
func without(dir string, shaped []shapedCopy, fn func() error) error {
	for _, f := range shaped {
		if err := os.Remove(filepath.Join(dir, f.path)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	runErr := fn()
	var errs []error
	errs = append(errs, runErr)
	for _, f := range shaped {
		file := filepath.Join(dir, f.path)
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.WriteFile(file, f.content, f.mode); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.Chmod(file, f.mode); err != nil {
			errs = append(errs, err)
			continue
		}
		b, err := os.ReadFile(file)
		sum := sha256.Sum256(b)
		if err != nil || hex.EncodeToString(sum[:]) != f.sum {
			errs = append(errs, fmt.Errorf("static checks: %s was not restored in the copy", f.path))
		}
	}
	return errors.Join(errs...)
}

// run checks the candidate tree: sha is the hash of the package it holds.
func (g *staticGateRunner) run(ctx context.Context, head, tree, sha string) (*staticRun, error) {
	run := &staticRun{Head: head, Tree: tree, SHA256: sha, Files: []staticFile{}, Checks: []staticCheck{}}
	var selected []string
	for _, c := range g.cfg.Checks {
		sc := staticCheck{Name: c.Name, Run: c.Run, Paths: []string{}}
		if isTestSuiteCheck(c.Run) {
			sc.Status = staticNotRun
		} else {
			selected = append(selected, c.Name)
		}
		run.Checks = append(run.Checks, sc)
	}
	paths, err := g.shapedFiles()
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		g.progress("kogen: no configured static check applies to the shaped files")
		shaped, err := readShaped(g.root, paths)
		if err != nil {
			return nil, err
		}
		for _, f := range shaped {
			run.Files = append(run.Files, staticFile{f.path, f.sum})
		}
		for _, c := range run.Checks {
			g.progress(fmt.Sprintf("kogen: static check %s not run: it runs the test suite", c.Name))
		}
		run.Outcome = staticNone
		return run, nil
	}
	g.progress("kogen: running static checks on the shaped files in a fresh copy: " + strings.Join(selected, ", "))

	dir := sys.TmpPath("kogen-static")
	run.Dir = dir
	defer os.RemoveAll(dir)
	mat, err := g.repo.Materialize(tree, dir)
	if err != nil {
		return nil, err
	}
	seeded, took := seedIgnored(g.root, dir)
	run.SeededPaths, run.SeededMS = seeded, took.Milliseconds()
	if err := g.rd.Setup(ctx, g.s, dir, false); err != nil {
		return nil, err
	}
	shaped, err := readShaped(dir, paths)
	if err != nil {
		return nil, err
	}
	shapedPaths := make([]string, len(shaped))
	for i, f := range shaped {
		shapedPaths[i] = f.path
		run.Files = append(run.Files, staticFile{f.path, f.sum})
	}
	var others []string
	for _, p := range mat.Paths {
		if !slices.Contains(shapedPaths, p) {
			others = append(others, p)
		}
	}
	opts := sys.Options{Timeout: verify.StepTimeout, KillGroupOnExit: true}

	for i := range run.Checks {
		c := &run.Checks[i]
		if c.Status == staticNotRun {
			g.progress(fmt.Sprintf("kogen: static check %s not run: it runs the test suite", c.Name))
			continue
		}
		res, err := g.s.Run(c.Run, dir, opts)
		if err != nil {
			return nil, err
		}
		exit := res.Status
		c.Exit = &exit
		c.Output = g.bounded(res.Output)
		if reason, unavailable := staticUnavailableReason(res); unavailable {
			c.Status, c.Reason = staticUnavailable, reason
			g.progress(fmt.Sprintf("kogen: static check %s unavailable: %s; the shaped files are not established by this check", c.Name, reason))
			continue
		}
		if res.Status == 0 {
			c.Status = staticPassed
			g.progress(fmt.Sprintf("kogen: static check %s passed", c.Name))
			continue
		}
		traced, err := g.s.Run("set -x\n"+c.Run, dir, opts)
		if err != nil {
			return nil, err
		}
		c.Trace = g.bounded(traced.Output)
		var baseline sys.Result
		if err := without(dir, shaped, func() (err error) {
			baseline, err = g.s.Run(c.Run, dir, opts)
			return err
		}); err != nil {
			return nil, err
		}
		_, baselineUnavailable := staticUnavailableReason(baseline)
		named := namedPaths(shapedPaths, res.Output)
		baselineNamed := namedPaths(shapedPaths, baseline.Output)
		attributable := baseline.Status == 0 && !baselineUnavailable
		for _, p := range named {
			if !slices.Contains(baselineNamed, p) {
				attributable = true
			}
		}
		if attributable {
			if len(named) == 0 {
				named = namedPaths(shapedPaths, traced.Output)
			}
			c.Status = staticAttributable
			c.Paths = staticEntries(named, res.Output, traced.Output)
			g.progress(fmt.Sprintf("kogen: static check %s failed (exit %d) because of the shaped files: %s", c.Name, res.Status, listOr(c.Paths, "(not named by the check)")))
			continue
		}
		c.Status = staticPreexisting
		c.Paths = staticEntries(namedPaths(others, res.Output, traced.Output), res.Output, traced.Output)
		g.progress(fmt.Sprintf("kogen: static check %s failed (exit %d) without the shaped files too: pre-existing failure in %s; the shaped files are not established by this check", c.Name, res.Status, listOr(c.Paths, "(no path named)")))
	}

	run.Outcome = staticChecked
	for _, c := range run.Checks {
		switch c.Status {
		case staticAttributable:
			run.Outcome = staticRefused
		case staticPreexisting, staticUnavailable:
			if run.Outcome != staticRefused {
				run.Outcome = staticUnknown
			}
		}
	}
	return run, nil
}

func listOr(items []string, none string) string {
	if len(items) == 0 {
		return none
	}
	return strings.Join(items, ", ")
}

// refusal is the headline of a refused run and its problem lines.
func (r *staticRun) refusal() (string, []string) {
	var problems []string
	failing := 0
	for _, c := range r.Checks {
		if c.Status != staticAttributable {
			continue
		}
		failing++
		problems = append(problems, fmt.Sprintf("- check %s: exit %d; files: %s; command: %s", c.Name, *c.Exit, listOr(c.Paths, "(not named by the check)"), c.Run))
		if strings.TrimSpace(c.Output) == "" {
			problems = append(problems, "  | (no output)")
			continue
		}
		for _, line := range strings.Split(c.Output, "\n") {
			problems = append(problems, "  | "+line)
		}
	}
	return fmt.Sprintf("Static checks: the shaped files fail %d configured static check(s); nothing was published. "+
		"Each check ran unchanged on the final submitted bytes in a fresh copy of the candidate; repair the named files so that every check passes. "+
		"Resubmitting this package unchanged is refused again.", failing), problems
}

// staticChecks is the gate in a files submission: it runs the checks on the candidate or reuses
// the result for the identical one, and refuses an attributable defect.
func (sh *shaping) staticChecks(ctx context.Context, hash string) (string, outcome, error) {
	head := sh.repo.Head()
	tree, err := sh.repo.Tree(head, nil)
	if err != nil {
		return "", outcome{}, err
	}
	run, reused := sh.staticLast, false
	if run != nil && run.Head == head && run.Tree == tree {
		reused = true
		sh.progress(fmt.Sprintf("kogen: static checks: the candidate is unchanged since submission %d; their result stands", run.attempt))
	} else {
		g := newStaticGateRunner(sh.s, sh.repo, sh.rd, sh.cfg, sh.root, sh.slug, sh.progress)
		if run, err = g.run(ctx, head, tree, hash); err != nil {
			return "", outcome{}, err
		}
		run.attempt = sh.submissions
		sh.staticLast = run
		if err := writeStaticRecord(sh.root, sh.slug, run); err != nil {
			return "", outcome{}, err
		}
	}
	sh.ev.Record(map[string]any{"event": "static_checks", "attempt": sh.submissions, "outcome": run.Outcome, "reused": reused, "dir": run.Dir,
		"head": run.Head, "tree": run.Tree, "sha256": run.SHA256, "files": run.Files, "checks": run.Checks,
		"seeded_paths": run.SeededPaths, "seeded_ms": run.SeededMS})
	if run.Outcome != staticRefused {
		return "", outcome{}, nil
	}
	headline, problems := run.refusal()
	sh.refused(staticGate, problems)
	return headline + "\n" + strings.Join(problems, "\n"), outcome{kind: outInvalid, problems: problems}, nil
}

// approveStaticChecks is the gate in `kogen intent approve`: it reuses the result Shape recorded
// for the identical package and otherwise runs the checks on the candidate tree of ready.
func approveStaticChecks(ctx context.Context, s *sys.Session, in ApproveInput, repo *git.Repo, cfg *project.Config, ready *verify.Ready, sha string) error {
	root, slug := repo.Dir, in.Slug
	if b, err := os.ReadFile(staticRecordPath(root, slug)); err == nil {
		var rec staticRecord
		if json.Unmarshal(b, &rec) == nil && rec.Version == 1 && rec.Head == ready.Head && rec.Tree == ready.CandidateTree && rec.SHA256 == sha &&
			slices.Contains([]string{staticChecked, staticUnknown, staticNone}, rec.Outcome) {
			s.Progress("kogen: static checks: reusing the result recorded for this package (" + rec.Outcome + ")")
			return nil
		}
	}
	rd := &verify.Readiness{Root: root, Repo: repo, Config: cfg, Tools: in.Tools, Log: in.Log, Evidence: in.Evidence}
	run, err := newStaticGateRunner(s, repo, rd, cfg, root, slug, s.Progress).run(ctx, ready.Head, ready.CandidateTree, sha)
	if err != nil {
		return err
	}
	if err := writeStaticRecord(root, slug, run); err != nil {
		return err
	}
	if run.Outcome != staticRefused {
		return nil
	}
	_, problems := run.refusal()
	e := sys.Fail("intent", staticGate, "the Intent's files fail configured static checks", 1)
	e.Lines = problems
	return e
}
