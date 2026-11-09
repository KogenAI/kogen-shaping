package consumes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// Compile-time pins of the two interfaces kogen-shaping implements for verify.
var (
	_ verify.Evidence = (*memEvidence)(nil)
	_ verify.Logger   = (*memLog)(nil)
)

// memLog keeps the events verify logs.
type memLog struct {
	mu     sync.Mutex
	events []map[string]any
}

func (l *memLog) Log(event map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

func (l *memLog) named(event string) []map[string]any {
	var out []map[string]any
	for _, e := range l.events {
		if e["event"] == event {
			out = append(out, e)
		}
	}
	return out
}

// memEvidence keeps what verify tells its Evidence.
type memEvidence struct {
	counts map[string]int
	timed  []string
	probes int
	events []map[string]any
}

func newEvidence() *memEvidence { return &memEvidence{counts: map[string]int{}} }

func (e *memEvidence) Count(kind string) { e.counts[kind]++ }

func (e *memEvidence) Timed(phase string, fn func() error) error {
	e.timed = append(e.timed, phase)
	return fn()
}

func (e *memEvidence) Probe(fn func() (verify.ProbeResult, error)) (verify.ProbeResult, error) {
	e.probes++
	return fn()
}

func (e *memEvidence) Record(event map[string]any) { e.events = append(e.events, event) }

// isolateTmp makes the temporary folder of the test its own, so nothing leaks between tests.
func isolateTmp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	return dir
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(path string) bool { _, err := os.Lstat(path); return err == nil }

func sha(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func session(dir string) *sys.Session {
	s := sys.NewWith(os.Environ(), dir)
	s.Stderr = &strings.Builder{}
	return s
}

// repo is a temporary Git repository with one commit and its own identity, no signing and an
// empty hooks folder.
type repo struct {
	t    *testing.T
	Root string
	Repo *git.Repo
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	r := &repo{t: t, Root: t.TempDir()}
	r.Repo = git.New(r.Root, os.Environ())
	r.git("init", "-q")
	r.git("config", "user.name", "Consumer")
	r.git("config", "user.email", "consumer@example.invalid")
	r.git("config", "core.hooksPath", t.TempDir())
	r.git("config", "commit.gpgsign", "false")
	write(t, r.Root, ".gitignore", ".kogen/local/\n")
	write(t, r.Root, "README.md", "readme\n")
	r.commit("baseline")
	return r
}

func (r *repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Root
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func (r *repo) write(rel, content string) { write(r.t, r.Root, rel, content) }

func (r *repo) commit(msg string) {
	r.t.Helper()
	r.git("add", "-A")
	r.git("commit", "-qm", msg, "--allow-empty")
}

func (r *repo) head() string { return strings.TrimSpace(r.git("rev-parse", "HEAD")) }

// fixtureProject is a committed Intent `fixture` in a repository: its acceptance command copies a
// prepared Cucumber Messages report, the red one when app.txt holds `red`, else the green one.
type fixtureProject struct {
	*repo
	Config *project.Config
	Aux    string // a folder outside the repository: the prepared reports and traces
}

const (
	fixtureSlug = "fixture"
	// Lines: 2 Scenario First, 5 Scenario Later, 8 Scenario Outline, 13 and 14 its Examples rows.
	fixtureFeature = "Feature: Fixture\n" +
		"  Scenario: First\n" +
		"    Given a fixture\n" +
		"    Then the candidate succeeds\n" +
		"  Scenario: Later\n" +
		"    Then the candidate succeeds\n" +
		"\n" +
		"  Scenario Outline: Pair <n>\n" +
		"    Given number <n>\n" +
		"    Then number <n> is seen\n" +
		"    Examples:\n" +
		"      | n |\n" +
		"      | 1 |\n" +
		"      | 2 |\n"
)

func intentMD(feasibility string) string {
	if feasibility == "" {
		feasibility = "- none"
	}
	return "# Fixture intent\n\nSlug: " + fixtureSlug + "\nModel: not applicable — nothing to model\nAcceptance:\n- " + project.FeaturePath(fixtureSlug) +
		"\n- features/steps/fixture_steps.js\n\nThe fixture.\n\n## Decisions\n- Fixture decision. (source: request)\n\n## Tests\n- none: fixture\n\n## Feasibility\n" + feasibility + "\n"
}

// newProject commits the Intent with the given feasibility bullets (none when empty).
func newProject(t *testing.T, feasibility string) *fixtureProject {
	t.Helper()
	p := &fixtureProject{repo: newRepo(t), Aux: t.TempDir()}
	p.write("app.txt", "green\n")
	p.write(project.IntentMDPath(fixtureSlug), intentMD(feasibility))
	p.write(project.FeaturePath(fixtureSlug), fixtureFeature)
	p.write("features/steps/fixture_steps.js", "// steps\n")
	expected, err := project.ParseGherkin(fixtureFeature)
	if err != nil {
		t.Fatal(err)
	}
	green := filepath.Join(p.Aux, "green.ndjson")
	red := filepath.Join(p.Aux, "red.ndjson")
	writeReport(t, green, project.FeaturePath(fixtureSlug), expected, nil)
	writeReport(t, red, project.FeaturePath(fixtureSlug), expected, map[int][]outcome{
		0: {{"FAILED", "AssertionError: must run exactly once\n$ go test -json -run '^TestFoo$' ./pkg"}, {"SKIPPED", ""}},
		1: {{"FAILED", "Finished tests\n5 runs, 10 assertions, 0 failures, 0 errors, 0 skips\n"}},
	})
	run := `if grep -qx red app.txt; then cp '` + red + `' "$KOGEN_ACCEPTANCE_REPORT"; echo actual-case-red; exit 1; fi; cp '` + green + `' "$KOGEN_ACCEPTANCE_REPORT"`
	p.Config = &project.Config{Acceptance: &project.Acceptance{Run: run}}
	p.commit("fixture project")
	return p
}

func (p *fixtureProject) readiness() *verify.Readiness {
	p.t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		p.t.Fatal(err)
	}
	cucumber, err := filepath.Abs("../../testdata/tools/cucumber/node_modules")
	if err != nil {
		p.t.Fatal(err)
	}
	return &verify.Readiness{Root: p.Root, Repo: p.Repo, Config: p.Config, Tools: verify.Tools{Node: node, Cucumber: cucumber}}
}

// outcome is the result of one step in a prepared report.
type outcome struct{ status, message string }

// writeReport writes a Cucumber Messages report (NDJSON) of the expected scenarios: every step
// passes unless outcomes names the results of the steps of scenario i.
func writeReport(t *testing.T, path, feature string, expected []project.Scenario, outcomes map[int][]outcome) {
	t.Helper()
	type obj = map[string]any
	var children []any
	var events []obj
	for i, sc := range expected {
		id := "s" + string(rune('0'+i))
		var astSteps, pickleSteps, testSteps, finished []any
		for j, st := range sc.Steps {
			sid := id + "-" + string(rune('0'+j))
			astSteps = append(astSteps, obj{"id": sid, "location": obj{"line": st.Line}, "keyword": st.Keyword})
			pickleSteps = append(pickleSteps, obj{"id": "p" + sid, "astNodeIds": []any{sid}, "text": st.Text})
			testSteps = append(testSteps, obj{"id": "t" + sid, "pickleStepId": "p" + sid})
			result := obj{"status": "PASSED"}
			if o, ok := outcomes[i]; ok && j < len(o) {
				result = obj{"status": o[j].status}
				if o[j].message != "" {
					result["message"] = o[j].message
				}
			}
			finished = append(finished, obj{"testStepFinished": obj{"testCaseStartedId": "k" + id, "testStepId": "t" + sid, "testStepResult": result}})
		}
		children = append(children, obj{"scenario": obj{"id": id, "location": obj{"line": sc.Line}, "steps": astSteps, "examples": []any{}}})
		events = append(events,
			obj{"pickle": obj{"id": "p" + id, "uri": feature, "name": sc.Name, "astNodeIds": []any{id}, "steps": pickleSteps}},
			obj{"testCase": obj{"id": "c" + id, "pickleId": "p" + id, "testSteps": testSteps}},
			obj{"testCaseStarted": obj{"id": "k" + id, "testCaseId": "c" + id}},
		)
		for _, f := range finished {
			events = append(events, f.(obj))
		}
		events = append(events, obj{"testCaseFinished": obj{"testCaseStartedId": "k" + id, "willBeRetried": false}})
	}
	all := append([]obj{{"gherkinDocument": obj{"uri": feature, "feature": obj{"children": children}}}}, events...)
	var lines []string
	for _, e := range all {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(b))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sysError is err as the refusal verify makes of it.
func sysError(t *testing.T, err error, class, reason string) *sys.Error {
	t.Helper()
	e, ok := sys.AsError(err)
	if !ok || e.Class != class || e.Reason != reason {
		t.Fatalf("err = %v, want %s/%s", err, class, reason)
	}
	return e
}

func equalStrings(got, want []string) bool { return slices.Equal(got, want) }
