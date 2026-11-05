package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

const (
	slug         = "is-even"
	intentPath   = ".kogen/intents/is-even/intent.md"
	featurePath  = ".kogen/intents/is-even/acceptance.feature"
	stepFilePath = "features/steps/is-even__steps.js"
)

const feature = "Feature: Is even\n  Scenario: Even numbers\n    Given is-even: the number 4\n    Then is-even: it is even\n  Scenario: Odd numbers\n    Given is-even: the number 3\n    Then is-even: it is not even\n"

const steps = "// steps\nconst { Given, Then } = require('@cucumber/cucumber');\nGiven('is-even: the number {int}', function () {});\nThen('is-even: it is even', function () {});\nThen('is-even: it is not even', function () {});\n"

const intentMD = "# Add an is-even check\n\nSlug: is-even\nModel: not applicable — a pure function without state\nAcceptance:\n- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\ncalc.is_even(n) returns True for even integers.\n\n## Decisions\n- Non-integers raise TypeError. (source: request)\n\n## Source comparison\n- The request asks for is_even; the feature covers it.\n\n## Tests\n- is_even returns True for even integers: add test_is_even in tests/test_calc.py (unit)\n\n## Feasibility\n- none\n"

// filesSubmission is a valid submission of the whole package.
func filesSubmission() map[string]any {
	return map[string]any{"files": []any{
		map[string]any{"path": intentPath, "content": intentMD},
		map[string]any{"path": featurePath, "content": feature},
		map[string]any{"path": stepFilePath, "content": steps},
	}}
}

// providerURL is a loopback origin that is never dialed: the client's transport answers in process.
const providerURL = "http://127.0.0.1:9"

// processEnv is the process environment without GIT_* plus a Git that reads no configuration and
// never prompts.
func processEnv(extra ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") {
			env = append(env, kv)
		}
	}
	env = append(env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	return append(env, extra...)
}

// fx is a committed project whose acceptance runner replays a prepared report, a signed-in home
// and a scripted provider.
type fx struct {
	t      *testing.T
	Root   string
	Home   string
	Aux    string
	Env    []string
	Repo   *git.Repo
	Prov   *scripted
	Clock  *clock
	Client *provider.Client
	Stdout *strings.Builder
	Stderr *strings.Builder
	S      *sys.Session
}

// newFx builds the fixture project in t.TempDir(): a Git repository with a local identity, no
// hooks and no signing, and the files below committed.
func newFx(t *testing.T, script ...turn) *fx {
	t.Helper()
	f := &fx{t: t, Root: t.TempDir(), Home: t.TempDir(), Aux: t.TempDir(), Clock: newClock()}
	f.Prov = newScripted(f.Clock, script...)
	f.Env = processEnv("HOME="+f.Home, "KOGEN_PROVIDER_URL="+providerURL)
	f.Repo = git.New(f.Root, f.Env)
	f.git("init", "-q")
	cfg := "[user]\n\tname = Fixture\n\temail = fixture@example.invalid\n[core]\n\thooksPath = " + t.TempDir() + "\n[commit]\n\tgpgsign = false\n"
	conf, err := os.OpenFile(filepath.Join(f.Root, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		_, err = conf.WriteString(cfg)
		if cerr := conf.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	f.write(".gitignore", ".kogen/local/\n")
	f.write("README.md", "# Calc\n\nA tiny calculator.\n")
	f.write("calc.js", "function add(a, b) { return a + b }\n")
	f.write("features/steps/calc_steps.js", "Given('a calculator', function () {});\n")
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks: [{name: unit, run: 'true'}]\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	script0 := fmt.Sprintf(`#!/bin/sh
case "$KOGEN_ACCEPTANCE_FEATURES" in */check.feature) : > "$KOGEN_ACCEPTANCE_REPORT"; exit 0;; esac
cp '%[1]s/report' "$KOGEN_ACCEPTANCE_REPORT"
exit $(cat '%[1]s/exit')
`, f.Aux)
	f.write("accept.sh", script0)
	if err := os.Chmod(filepath.Join(f.Root, "accept.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.setReport("FAILED", 1)
	f.git("add", "-A")
	f.git("commit", "-qm", "fixture project")
	f.signIn()
	f.newSession()
	f.Client = &provider.Client{
		Cache: provider.NewCache(nil),
		Sleep: func(ctx context.Context, d time.Duration) error { return nil },
		HTTP:  &http.Client{Transport: f.Prov},
		Now:   f.Clock.Now,
	}
	return f
}

// newSession gives the next run its own session with fresh output buffers.
func (f *fx) newSession() {
	f.Stdout, f.Stderr = &strings.Builder{}, &strings.Builder{}
	f.S = sys.NewWith(f.Env, f.Root)
	f.S.Stdout, f.S.Stderr = f.Stdout, f.Stderr
}

// setReport prepares the Cucumber message report the acceptance runner replays and its exit code:
// red fails the last step of each scenario (exit 1), green passes every step (exit 0).
func (f *fx) setReport(status string, code int) {
	f.t.Helper()
	scenarios, err := project.ParseGherkin(feature)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.Aux, "report"), []byte(messages(f.t, scenarios, status)), 0o644); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.Aux, "exit"), []byte(fmt.Sprint(code)), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fx) signIn() {
	dir := filepath.Join(f.Home, ".kogen", "credentials")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	creds, _ := json.Marshal(map[string]any{
		"access_token": "access", "refresh_token": "refresh", "client_id": "oaiapp_fixture",
		"expires_at": time.Now().Add(24 * time.Hour).Unix(),
	})
	if err := os.WriteFile(filepath.Join(dir, "chatgpt.json"), creds, 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// messages is a Cucumber message report of the scenarios: "PASSED" passes every step, "FAILED"
// fails the last step of each scenario.
func messages(t *testing.T, scenarios []project.Scenario, status string) string {
	t.Helper()
	feature := project.FeaturePath(slug)
	type msg = map[string]any
	var children []any
	for i, sc := range scenarios {
		n := i + 1
		var astSteps []any
		for j, st := range sc.Steps {
			astSteps = append(astSteps, msg{"id": fmt.Sprintf("ast-step-%d-%d", n, j+1), "location": msg{"line": st.Line}, "keyword": st.Keyword})
		}
		children = append(children, msg{"scenario": msg{"id": fmt.Sprintf("ast-sc-%d", n), "location": msg{"line": sc.Line}, "steps": astSteps, "examples": []any{}}})
	}
	events := []msg{{"gherkinDocument": msg{"uri": feature, "feature": msg{"children": children}}}}
	for i, sc := range scenarios {
		n := i + 1
		var pickleSteps, testSteps, finished []any
		for j, st := range sc.Steps {
			pid, tid := fmt.Sprintf("p-%d-%d", n, j+1), fmt.Sprintf("t-%d-%d", n, j+1)
			pickleSteps = append(pickleSteps, msg{"id": pid, "astNodeIds": []any{fmt.Sprintf("ast-step-%d-%d", n, j+1)}, "text": st.Text})
			testSteps = append(testSteps, msg{"id": tid, "pickleStepId": pid})
			result := msg{"status": status}
			if status == "FAILED" {
				if j < len(sc.Steps)-1 {
					result["status"] = "PASSED"
				} else {
					result["message"] = "assertion failed in scenario " + sc.Name
				}
			}
			finished = append(finished, msg{"testStepFinished": msg{"testCaseStartedId": fmt.Sprintf("s-%d", n), "testStepId": tid, "testStepResult": result}})
		}
		events = append(events,
			msg{"pickle": msg{"id": fmt.Sprintf("pk-%d", n), "uri": feature, "name": sc.Name, "astNodeIds": []any{fmt.Sprintf("ast-sc-%d", n)}, "steps": pickleSteps}},
			msg{"testCase": msg{"id": fmt.Sprintf("c-%d", n), "pickleId": fmt.Sprintf("pk-%d", n), "testSteps": testSteps}},
			msg{"testCaseStarted": msg{"id": fmt.Sprintf("s-%d", n), "testCaseId": fmt.Sprintf("c-%d", n)}})
		for _, fin := range finished {
			events = append(events, fin.(msg))
		}
		events = append(events, msg{"testCaseFinished": msg{"testCaseStartedId": fmt.Sprintf("s-%d", n), "willBeRetried": false}})
	}
	var b strings.Builder
	for _, e := range events {
		line, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func (f *fx) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = f.Root
	cmd.Env = f.Env
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func (f *fx) write(rel, content string) {
	f.t.Helper()
	path := filepath.Join(f.Root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fx) read(rel string) string {
	f.t.Helper()
	b, err := os.ReadFile(filepath.Join(f.Root, rel))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

func (f *fx) head() string { return strings.TrimSpace(f.git("rev-parse", "HEAD")) }

// draft writes the is-even package and records its readiness with verify.Readiness.Check, against
// the report currently prepared.
func (f *fx) draft() {
	f.t.Helper()
	f.write(intentPath, intentMD)
	f.write(featurePath, feature)
	f.write(stepFilePath, steps)
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		f.t.Fatal(err)
	}
	f.newSession()
	rd := &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: cfg}
	if _, err := rd.Check(context.Background(), f.S, slug); err != nil {
		f.t.Fatal(err)
	}
}

// committedDraft is a draft committed with a green readiness: the state Approve and Preflight
// start from.
func (f *fx) committedDraft() {
	f.t.Helper()
	f.setReport("PASSED", 0)
	f.write(intentPath, intentMD)
	f.write(featurePath, feature)
	f.write(stepFilePath, steps)
	f.git("add", "-A")
	f.git("commit", "-qm", "is-even draft")
	f.draft()
}

// hash is the baseline hash of the is-even package.
func (f *fx) hash() string {
	f.t.Helper()
	cfg, err := project.LoadConfig(f.Root)
	if err != nil {
		f.t.Fatal(err)
	}
	paths, err := project.Baseline(f.Root, slug, cfg)
	if err != nil {
		f.t.Fatal(err)
	}
	return project.Hash(f.Root, paths)
}

// events are the events of an evidence file under the project.
func (f *fx) events(rel string) []map[string]any {
	f.t.Helper()
	b, err := os.ReadFile(filepath.Join(f.Root, rel))
	if err != nil {
		f.t.Fatal(err)
	}
	var out []map[string]any
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			f.t.Fatalf("event %q: %v", line, err)
		}
		out = append(out, ev)
	}
	return out
}

// shapeEvents are the events of the Shape run log.
func (f *fx) shapeEvents() []map[string]any {
	return f.events(".kogen/local/shaping/" + slug + "/events.ndjson")
}

// refineEvents are the events of the one refinement evidence folder.
func (f *fx) refineEvents() []map[string]any {
	f.t.Helper()
	files, _ := filepath.Glob(filepath.Join(f.Root, ".kogen/local/refinements/*/events.ndjson"))
	if len(files) != 1 {
		f.t.Fatalf("refinement evidence files: %v", files)
	}
	rel, err := filepath.Rel(f.Root, files[0])
	if err != nil {
		f.t.Fatal(err)
	}
	return f.events(rel)
}
