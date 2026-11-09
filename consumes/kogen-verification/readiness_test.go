package consumes

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// Signatures and field types kogen-shaping's calls rely on.
var (
	_ func(*verify.Readiness, context.Context, *sys.Session, string) (*verify.Ready, error) = (*verify.Readiness).Check
	_ func(*verify.Readiness, *sys.Session, string) *verify.Ready                           = (*verify.Readiness).Cached
	_ func(*verify.Readiness, *sys.Session, string, *verify.Ready) bool                     = (*verify.Readiness).Current
	_ func(*verify.Readiness, *sys.Session, string, *verify.Ready) error                    = (*verify.Readiness).RecordPass
	_ func(*verify.Readiness, context.Context, *sys.Session, string, bool) error            = (*verify.Readiness).Setup
	_ func(string, *verify.Ready, string) string                                            = verify.Observations
	_ func(string, *verify.Ready, string) string                                            = verify.Acknowledgment
	_ func(string, string, *verify.Redactor) *verify.Attempt                                = verify.NewAttempt
	_ func(*verify.Ready) bool                                                              = (*verify.Ready).Failures
)

// readinessFields pins the types of the Readiness and Ready fields kogen-shaping sets and reads.
func readinessFields(r *verify.Readiness, ready *verify.Ready, sc verify.ScenarioResult) {
	var (
		_ string                    = r.Root
		_ *git.Repo                 = r.Repo
		_ *project.Config           = r.Config
		_ verify.Tools              = r.Tools
		_ verify.Logger             = r.Log
		_ verify.Evidence           = r.Evidence
		_ func() error              = r.BeforeRecord
		_ func(*verify.Ready) error = r.Gate
		_ *verify.Attempt           = r.Attempt
		_ bool                      = r.BoundShapeAcceptance

		_ *project.Intent         = ready.Intent
		_ []verify.Obligation     = ready.Feasibility
		_ *project.ModelSpec      = ready.Model
		_ []verify.ScenarioResult = ready.Scenarios
		_ int                     = ready.AcceptanceExit
		_ []string                = ready.Baseline
		_ string                  = ready.SHA256
		_ string                  = ready.Head
		_ string                  = ready.CandidateTree
		_ []verify.StepWarning    = ready.StepWarnings

		_ string              = sc.Name
		_ int                 = sc.Line
		_ []verify.StepRecord = sc.Steps
		_ []verify.StepRecord = sc.UnexercisedSteps
		_ *string             = sc.OutlineName
		_ *int                = sc.OutlineLine
		_ *int                = sc.OutlineExamples
		_ string              = sc.Status
		_ *string             = sc.Detail
		_ []string            = sc.Unwritten
		_ string              = sc.RunnerObserver
	)
}

// A green readiness check runs setup and the acceptance command in a clean scratch copy of
// the candidate, accounts for every scenario and keeps complete evidence that stays reusable
// until the candidate changes.
func TestReadinessCheckAndKeptEvidence(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	trace := filepath.Join(p.Aux, "setup.trace")
	p.Config.Setup = []string{
		`printf '%s|%s|%s\n' "$KOGEN_NODE" "$KOGEN_QUINT" "$(test -f .kogen/intents/fixture/intent.md && echo candidate)" >> '` + trace + `'`,
		`echo second >> '` + trace + `'`,
	}
	log, ev := &memLog{}, newEvidence()
	r := p.readiness()
	toolsDir := t.TempDir()
	node := filepath.Join(toolsDir, "node")
	quint := filepath.Join(toolsDir, "quint")
	for _, path := range []string{node, quint} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	r.Tools.Node, r.Tools.Quint = node, quint
	r.Log, r.Evidence = log, ev
	s := session(p.Root)

	ready, err := r.Check(context.Background(), s, fixtureSlug)
	if err != nil {
		t.Fatalf("verify.Readiness.Check: %v", err)
	}
	if got := readFile(t, "", trace); got != node+"|"+quint+"|candidate\nsecond\n" {
		t.Errorf("verify.Readiness.Setup: setup runs in order in the candidate copy with the tools as KOGEN_NODE and KOGEN_QUINT (Tools), got %q", got)
	}
	if ev.counts["setup"] != 2 || ev.counts["acceptance"] != 1 || !slices.Equal(ev.timed, []string{"setup", "setup"}) {
		t.Errorf("verify.Readiness.Evidence: setup must be counted and timed twice and acceptance counted once, got %+v", ev)
	}
	if names := log.named("command"); len(names) != 2 || names[0]["name"] != "setup-1" || names[1]["name"] != "setup-2" {
		t.Errorf("verify.Readiness.Log: the setup commands must be logged as setup-1 and setup-2, got %+v", names)
	}
	if res := log.named("command_result"); len(res) != 2 || res[0]["status"] != 0 || res[1]["name"] != "setup-2" {
		t.Errorf("verify.Readiness.Log: command results must be logged with their status, got %+v", res)
	}

	if ready.Intent == nil || ready.Intent.Slug != fixtureSlug || ready.Intent.Title != "Fixture intent" {
		t.Errorf("verify.Ready.Intent: want the checked Intent, got %+v", ready.Intent)
	}
	if ready.AcceptanceExit != 0 || ready.Failures() {
		t.Errorf("verify.Ready.AcceptanceExit and Failures: a green run exits 0 without failures, got %d %v", ready.AcceptanceExit, ready.Failures())
	}
	if ready.Model == nil || !ready.Model.None {
		t.Errorf("verify.Ready.Model: \"not applicable\" must give a ModelSpec with None, got %+v", ready.Model)
	}
	if len(ready.Feasibility) != 0 || len(ready.StepWarnings) != 0 {
		t.Errorf("verify.Ready.Feasibility and StepWarnings: none declared and no step files loaded, got %+v %+v", ready.Feasibility, ready.StepWarnings)
	}
	wantBaseline, err := project.Baseline(p.Root, fixtureSlug, p.Config)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ready.Baseline, wantBaseline) || !slices.Contains(ready.Baseline, project.FeaturePath(fixtureSlug)) {
		t.Errorf("verify.Ready.Baseline: the baseline files of the Intent, want %q, got %q", wantBaseline, ready.Baseline)
	}
	if ready.SHA256 != project.Hash(p.Root, ready.Baseline) {
		t.Errorf("verify.Ready.SHA256: want the hash of the baseline, got %s", ready.SHA256)
	}
	tree, err := p.Repo.Tree(p.head(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Head != p.head() || ready.CandidateTree != tree {
		t.Errorf("verify.Ready.Head and CandidateTree: want %s and %s, got %s and %s", p.head(), tree, ready.Head, ready.CandidateTree)
	}

	want := []string{"First", "Later", "Pair 1", "Pair 2"}
	lines := []int{2, 5, 13, 14}
	if len(ready.Scenarios) != 4 {
		t.Fatalf("verify.Ready.Scenarios: want one result per scenario and Examples row, got %+v", ready.Scenarios)
	}
	for i, sc := range ready.Scenarios {
		if sc.Status != "passed" || sc.Detail != nil || sc.Name != want[i] || sc.Line != lines[i] || len(sc.UnexercisedSteps) != 0 {
			t.Errorf("verify.ScenarioResult: scenario %d must be passed with its name %q and line %d, got %+v", i, want[i], lines[i], sc)
		}
	}
	first := ready.Scenarios[0]
	if !slices.Equal(first.Steps, []verify.StepRecord{{Keyword: "Given ", Text: "a fixture", Line: 3}, {Keyword: "Then ", Text: "the candidate succeeds", Line: 4}}) {
		t.Errorf("verify.StepRecord: the steps carry keyword, text and line, got %+v", first.Steps)
	}
	row := ready.Scenarios[3]
	if row.OutlineName == nil || *row.OutlineName != "Pair <n>" || row.OutlineLine == nil || *row.OutlineLine != 8 || row.OutlineExamples == nil || *row.OutlineExamples != 2 {
		t.Errorf("verify.ScenarioResult: an Examples row names its outline (Pair <n>, line 8, 2 examples), got %+v", row)
	}
	if first.OutlineName != nil || first.OutlineLine != nil || first.OutlineExamples != nil {
		t.Errorf("verify.ScenarioResult: a plain scenario has no outline, got %+v", first)
	}

	if rec := readFile(t, p.Root, ".kogen/local/readiness/fixture.json"); !strings.Contains(rec, `"outcome":"ready"`) || !strings.Contains(rec, `"head":"`+p.head()+`"`) {
		t.Errorf("verify.Readiness.RecordPass: the check must keep a ready record for the HEAD, got %s", rec)
	}
	if !exists(filepath.Join(p.Root, ".kogen/local/packages/fixture.json")) {
		t.Error("verify.Readiness.RecordPass: the package facts are refreshed after the record")
	}
	if status := p.git("status", "--porcelain"); status != "" {
		t.Errorf("verify.Readiness.Check: the checkout must stay untouched, got status %q", status)
	}

	cached := r.Cached(s, fixtureSlug)
	if cached == nil || cached.CandidateTree != ready.CandidateTree || cached.SHA256 != ready.SHA256 || cached.Head != ready.Head ||
		cached.AcceptanceExit != 0 || len(cached.Scenarios) != 4 || cached.Intent.Title != "Fixture intent" {
		t.Fatalf("verify.Readiness.Cached: the kept evidence of an unchanged candidate must be reused, got %+v", cached)
	}
	if !r.Current(s, fixtureSlug, cached) || !r.Current(s, fixtureSlug, ready) || r.Current(s, fixtureSlug, nil) {
		t.Error("verify.Readiness.Current: the evidence of the unchanged candidate is current, nil is not")
	}

	p.write("extra.txt", "x\n")
	p.commit("move HEAD")
	if r.Cached(s, fixtureSlug) != nil || r.Current(s, fixtureSlug, ready) {
		t.Error("verify.Readiness.Cached and Current: a moved HEAD makes the evidence stale")
	}
}

// Legitimate red is valid readiness: failed scenarios are accounted with their detail, the
// skipped steps after the failure, the named tests that did not run and the runner
// observation, and the observations and acknowledgment report it.
func TestReadinessRedObservations(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	p.write("app.txt", "red\n")
	p.commit("red")
	r := p.readiness()
	s := session(p.Root)

	ready, err := r.Check(context.Background(), s, fixtureSlug)
	if err != nil {
		t.Fatalf("verify.Readiness.Check: red that the dry run accounts for is valid readiness, got %v", err)
	}
	if ready.AcceptanceExit != 1 || !ready.Failures() {
		t.Fatalf("verify.Ready.AcceptanceExit and Failures: the red run exits 1 with failed scenarios, got %d %v", ready.AcceptanceExit, ready.Failures())
	}
	first, later, row := ready.Scenarios[0], ready.Scenarios[1], ready.Scenarios[2]
	if first.Status != "failed" || first.Detail == nil || !strings.HasPrefix(*first.Detail, "acceptance.feature line 3: Given a fixture\nAssertionError: must run exactly once") {
		t.Errorf("verify.ScenarioResult.Detail: a failed scenario names the failing step and its message, got %+v", first)
	}
	if want := []verify.StepRecord{{Keyword: "Then ", Text: "the candidate succeeds", Line: 4}}; !slices.Equal(first.UnexercisedSteps, want) {
		t.Errorf("verify.ScenarioResult.UnexercisedSteps: the skipped step after the failure, want %+v, got %+v", want, first.UnexercisedSteps)
	}
	if want := []string{"TestFoo: did not run (the failure shows no run of it)"}; !slices.Equal(first.Unwritten, want) {
		t.Errorf("verify.ScenarioResult.Unwritten: the named test that did not run, want %q, got %q", want, first.Unwritten)
	}
	if want := "the step failed although the test runner reported a passing run: 5 runs, 10 assertions, 0 failures, 0 errors, 0 skips"; later.Status != "failed" || later.RunnerObserver != want {
		t.Errorf("verify.ScenarioResult.RunnerObserver: want %q, got %q (status %s)", want, later.RunnerObserver, later.Status)
	}
	if row.Status != "passed" || row.RunnerObserver != "" || len(row.Unwritten) != 0 {
		t.Errorf("verify.ScenarioResult: a passed row has no observations, got %+v", row)
	}

	obs := verify.Observations(fixtureSlug, ready, "")
	for _, part := range []string{
		"Readiness observations\nChecked baseline: " + ready.SHA256 + "\nHEAD: " + p.head() + "\nAcceptance exit: 1\nEvidence: .kogen/local/readiness/fixture.json\n",
		"Unexercised acceptance steps: 1\n",
		"Scenario: First (acceptance.feature line 2)\nFailure phase: assertion\nacceptance.feature line 3: Given a fixture\nAssertionError: must run exactly once",
		"Scenario: Later (acceptance.feature line 5)",
		"Scenario: First (acceptance.feature line 2)\nacceptance.feature line 4: Then the candidate succeeds",
	} {
		if !strings.Contains(obs, part) {
			t.Errorf("verify.Observations: the observations must hold %q, got:\n%s", part, obs)
		}
	}
	if got := verify.Observations(fixtureSlug, ready, "custom/evidence.json"); !strings.Contains(got, "Evidence: custom/evidence.json\n") {
		t.Errorf("verify.Observations: the reference names the evidence file, got:\n%s", got)
	}
	if got := verify.Observations(fixtureSlug, &verify.Ready{}, ""); got != "" {
		t.Errorf("verify.Observations: without failures there are none, got %q", got)
	}

	ack := verify.Acknowledgment(fixtureSlug, ready, "")
	if !strings.HasPrefix(ack, "Inspect whether fixtures, inputs and commands reached the intended behavior") ||
		!strings.Contains(ack, "The files array must be nonempty") || !strings.HasSuffix(ack, "\n\n"+obs) {
		t.Errorf("verify.Acknowledgment: want the inspection prompt, the acknowledgment guidance and the observations, got:\n%s", ack)
	}

	cached := r.Cached(s, fixtureSlug)
	if cached == nil || !cached.Failures() || cached.Scenarios[0].Detail == nil || *cached.Scenarios[0].Detail != *first.Detail || len(cached.Scenarios[0].UnexercisedSteps) != 1 {
		t.Errorf("verify.Readiness.Cached: unchanged red is reused with its details, got %+v", cached)
	}
}

// Gate judges the complete checked result before anything is recorded and before
// BeforeRecord; an error from either stops the check with nothing recorded.
func TestReadinessGateAndBeforeRecord(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	s := session(p.Root)
	record := filepath.Join(p.Root, ".kogen/local/readiness/fixture.json")

	var order []string
	var gated *verify.Ready
	r := p.readiness()
	r.Gate = func(ready *verify.Ready) error {
		order = append(order, "gate")
		gated = ready
		if exists(record) {
			t.Error("verify.Readiness.Gate: runs before anything is recorded")
		}
		return nil
	}
	r.BeforeRecord = func() error {
		order = append(order, "before")
		if exists(record) {
			t.Error("verify.Readiness.BeforeRecord: runs right before the record")
		}
		return nil
	}
	ready, err := r.Check(context.Background(), s, fixtureSlug)
	if err != nil || !slices.Equal(order, []string{"gate", "before"}) || gated != ready || !exists(record) {
		t.Fatalf("verify.Readiness.Gate and BeforeRecord: Gate gets the checked result, then BeforeRecord, then the record is kept; got %v %v gated=%v", order, err, gated == ready)
	}

	r = p.readiness()
	stop := sys.Fail("intent", "not_ready", "gate stop", 1)
	r.Gate = func(*verify.Ready) error { return stop }
	r.BeforeRecord = func() error { t.Error("verify.Readiness.BeforeRecord: must not run after a Gate error"); return nil }
	if _, err := r.Check(context.Background(), s, fixtureSlug); err != error(stop) || exists(record) {
		t.Errorf("verify.Readiness.Gate: its error stops the check with nothing recorded, got %v (record kept: %v)", err, exists(record))
	}

	r = p.readiness()
	stop = sys.Fail("intent", "not_ready", "custody stop", 1)
	r.BeforeRecord = func() error { return stop }
	if _, err := r.Check(context.Background(), s, fixtureSlug); err != error(stop) || exists(record) {
		t.Errorf("verify.Readiness.BeforeRecord: its error stops the check with nothing recorded, got %v (record kept: %v)", err, exists(record))
	}
}

// RecordPass keeps the evidence only while the inputs it checked still hold, then refreshes
// the package facts; the step warnings of the result are kept with it.
func TestReadinessRecordPass(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	s := session(p.Root)
	r := p.readiness()
	ready, err := r.Check(context.Background(), s, fixtureSlug)
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(p.Root, ".kogen/local/readiness/fixture.json")
	facts := filepath.Join(p.Root, ".kogen/local/packages/fixture.json")
	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	os.Remove(facts)

	ready.StepWarnings = []verify.StepWarning{{Pattern: "spare", File: "features/steps/fixture_steps.js", Detail: "unused definition"}}
	if err := r.RecordPass(s, fixtureSlug, ready); err != nil {
		t.Fatalf("verify.Readiness.RecordPass: %v", err)
	}
	if !exists(record) || !exists(facts) {
		t.Errorf("verify.Readiness.RecordPass: the record and the package facts must be written, got record %v facts %v", exists(record), exists(facts))
	}
	cached := r.Cached(s, fixtureSlug)
	if cached == nil || !slices.Equal(cached.StepWarnings, ready.StepWarnings) {
		t.Errorf("verify.Ready.StepWarnings: the step warnings are kept with the evidence, got %+v", cached)
	}

	p.commit("move HEAD")
	os.Remove(record)
	err = r.RecordPass(s, fixtureSlug, ready)
	e := sysError(t, err, "intent", "not_ready")
	if e.Detail != "the Intent is not ready to build" || e.Exit != 1 || !equalStrings(e.Lines, []string{"readiness inputs changed during checking or publication"}) {
		t.Errorf("verify.Readiness.RecordPass: a result whose inputs changed is refused, got exit %d %q %q", e.Exit, e.Detail, e.Lines)
	}
	if exists(record) {
		t.Error("verify.Readiness.RecordPass: nothing may be recorded for stale inputs")
	}
}

// Setup runs commands.setup in order and refuses with environment/setup_failed (exit 3) at
// the first failure; with reuse it is skipped while the checkout state is unchanged.
func TestReadinessSetup(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	trace := filepath.Join(p.Aux, "setup.trace")
	p.Config.Setup = []string{`echo one >> '` + trace + `'`, `echo two >> '` + trace + `'`}
	r := p.readiness()
	s := session(p.Root)
	ctx := context.Background()

	if err := r.Setup(ctx, s, p.Root, true); err != nil {
		t.Fatalf("verify.Readiness.Setup: %v", err)
	}
	if err := r.Setup(ctx, s, p.Root, true); err != nil {
		t.Fatalf("verify.Readiness.Setup: %v", err)
	}
	if got := readFile(t, "", trace); got != "one\ntwo\n" {
		t.Errorf("verify.Readiness.Setup: with reuse the second call is skipped for an unchanged state, got %q", got)
	}
	if err := r.Setup(ctx, s, p.Root, false); err != nil || readFile(t, "", trace) != "one\ntwo\none\ntwo\n" {
		t.Errorf("verify.Readiness.Setup: without reuse it always runs, got %v", err)
	}
	if !strings.Contains(s.Stderr.(*strings.Builder).String(), "kogen: setup already ran for this checkout state") {
		t.Errorf("verify.Readiness.Setup: the skip is reported as progress, got %q", s.Stderr.(*strings.Builder).String())
	}

	p.Config.Setup = []string{"echo ok", "for i in $(seq 1 30); do echo line $i; done; exit 3", "echo never >> '" + trace + "'"}
	err := r.Setup(ctx, session(p.Root), p.Root, false)
	e := sysError(t, err, "environment", "setup_failed")
	if e.Exit != 3 || e.Detail != "Setup setup-2 failed (status=3)." || len(e.Lines) != 22 || e.Lines[19] != "line 30" || e.Lines[0] != "line 11" ||
		!strings.HasPrefix(e.Lines[20], "commands.setup[1]: ") || !strings.HasPrefix(e.Lines[21], "next: fix commands.setup") {
		t.Errorf("verify.Readiness.Setup: the first failure refuses with exit 3 and the last 20 output lines, got exit %d %q %d lines", e.Exit, e.Detail, len(e.Lines))
	}
	if strings.Contains(readFile(t, "", trace), "never") {
		t.Error("verify.Readiness.Setup: commands after the failing one must not run")
	}
}

// An approval Attempt bounds and records the steps, masks secrets with its Redactor and,
// when the check is refused, keeps private evidence and names the steps that ran.
func TestReadinessAttemptRefusal(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	p.Config.Setup = []string{"echo leaked tok-0a1b2c3d4e >&2; echo password=hunter2; exit 3"}
	red := verify.NewRedactor([]string{"MY_TOKEN=tok-0a1b2c3d4e"}, "")
	r := p.readiness()
	r.Attempt = verify.NewAttempt(p.Root, fixtureSlug, red)

	_, err := r.Check(context.Background(), session(p.Root), fixtureSlug)
	e := sysError(t, err, "environment", "setup_failed")
	text := strings.Join(e.Lines, "\n")
	if strings.Contains(text, "tok-0a1b2c3d4e") || strings.Contains(text, "hunter2") || !strings.Contains(text, "leaked [REDACTED]") || !strings.Contains(text, "password=[REDACTED]") {
		t.Errorf("verify.NewAttempt: secrets in the refusal's lines must be masked by the Redactor and labels, got:\n%s", text)
	}
	for _, part := range []string{"Steps that ran, with wall time (each step has its own 120s limit):", "  setup-1: failed with exit 3 after ", "Not run: acceptance"} {
		if !strings.Contains(text, part) {
			t.Errorf("verify.Readiness.Attempt: the refusal must hold %q, got:\n%s", part, text)
		}
	}
	evidence := e.Lines[len(e.Lines)-1]
	path, ok := strings.CutPrefix(evidence, "Evidence: ")
	if !ok || !strings.HasPrefix(path, filepath.Join(p.Root, ".kogen/local/readiness/attempts")) {
		t.Fatalf("verify.Readiness.Attempt: the last line must name the retained evidence, got %q", evidence)
	}
	var rec struct {
		Kind, Outcome, Slug string
		Failure             struct{ Step, Reason string }
	}
	if err := json.Unmarshal([]byte(readFile(t, "", path)), &rec); err != nil {
		t.Fatalf("verify.Readiness.Attempt: the evidence is not JSON: %v", err)
	}
	if rec.Kind != "approval_attempt" || rec.Outcome != "refused" || rec.Slug != fixtureSlug || rec.Failure.Step != "setup-1" || rec.Failure.Reason != "setup_failed" {
		t.Errorf("verify.Readiness.Attempt: the retained evidence must name the refused attempt and its failing step, got %+v", rec)
	}
	if strings.Contains(readFile(t, "", path), "tok-0a1b2c3d4e") {
		t.Error("verify.NewAttempt: the retained evidence must not keep the secret")
	}
	if exists(filepath.Join(p.Root, ".kogen/local/readiness/fixture.json")) {
		t.Error("verify.Readiness.Attempt: a refused attempt is never readiness evidence")
	}

	// The same project passes with an Attempt once setup works: nothing is refused.
	p.Config.Setup = []string{"true"}
	r = p.readiness()
	r.Attempt = verify.NewAttempt(p.Root, fixtureSlug, red)
	if ready, err := r.Check(context.Background(), session(p.Root), fixtureSlug); err != nil || ready == nil {
		t.Errorf("verify.Readiness.Attempt: a passing check under an Attempt is not refused, got %v", err)
	}
}

// Shape's acceptance is bounded by the approval step limit: a check that finishes in time
// gives the same result.
func TestReadinessBoundShapeAcceptance(t *testing.T) {
	isolateTmp(t)
	p := newProject(t, "")
	r := p.readiness()
	r.BoundShapeAcceptance = true
	ready, err := r.Check(context.Background(), session(p.Root), fixtureSlug)
	if err != nil || ready.AcceptanceExit != 0 || len(ready.Scenarios) != 4 || ready.Failures() {
		t.Errorf("verify.Readiness.BoundShapeAcceptance: a bounded acceptance that finishes in time must give the same green result, got %v %+v", err, ready)
	}
}

// Declared feasibility evidence runs in probes of the candidate, is bound to the bytes of its
// inputs and is kept with the readiness record as obligations; a mismatch refuses.
func TestReadinessFeasibilityObligations(t *testing.T) {
	isolateTmp(t)
	bullet := `- evidence: {"id":"route","kind":"capability","inputs":["evidence/in.txt"],"observers":[],"checks":[{"name":"support","command":"cat evidence/in.txt; exit 23","exit":23,"output":"hello\n"}]}`
	p := newProject(t, bullet)
	p.write("evidence/in.txt", "hello\n")
	p.commit("evidence input")
	ev := newEvidence()
	r := p.readiness()
	r.Evidence = ev
	s := session(p.Root)

	ready, err := r.Check(context.Background(), s, fixtureSlug)
	if err != nil {
		t.Fatalf("verify.Readiness.Check: %v", err)
	}
	if len(ready.Feasibility) != 1 {
		t.Fatalf("verify.Obligation: one obligation per declaration, got %+v", ready.Feasibility)
	}
	ob := ready.Feasibility[0]
	if ob.ID != "route" || ob.Kind != "capability" || ob.Inputs["evidence/in.txt"] != sha("hello\n") || len(ob.Observers) != 0 ||
		len(ob.Checks) != 1 || ob.Checks[0].Name != "support" || ob.Checks[0].Exit != 23 || ob.Checks[0].Output != "hello\n" || ob.Checks[0].TimedOut {
		t.Errorf("verify.Obligation: want the proof of the declared check bound to the input's SHA-256, got %+v", ob)
	}
	if ev.probes != 1 || len(ev.events) != 1 || ev.events[0]["event"] != "feasibility_check" {
		t.Errorf("verify.Readiness.Evidence: the check runs as one probe and is recorded as a feasibility_check event, got %d probes %+v", ev.probes, ev.events)
	}
	if got := readFile(t, p.Root, ".kogen/local/readiness/fixture-feasibility.ndjson"); !strings.Contains(got, `"id":"route"`) {
		t.Errorf("verify.Obligation: the actual results are retained in the feasibility ndjson, got %q", got)
	}
	cached := r.Cached(s, fixtureSlug)
	if cached == nil || len(cached.Feasibility) != 1 || cached.Feasibility[0].Inputs["evidence/in.txt"] != sha("hello\n") {
		t.Errorf("verify.Readiness.Cached: the obligations are kept with the record, got %+v", cached)
	}

	p.write("evidence/in.txt", "changed\n")
	p.commit("input changed")
	_, err = r.Check(context.Background(), s, fixtureSlug)
	e := sysError(t, err, "intent", "not_ready")
	if e.Detail != "declared feasibility evidence failed" || !slices.ContainsFunc(e.Lines, func(line string) bool {
		return strings.HasPrefix(line, "route/support: expected ") && strings.Contains(line, `"output":"changed\n"`)
	}) {
		t.Errorf("verify.Readiness.Check: a declared output that the actual check does not give must refuse, got %q %q", e.Detail, e.Lines)
	}
}
