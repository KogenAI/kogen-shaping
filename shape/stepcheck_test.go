package shape

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

func TestShapeChecksConfiguredCucumberOutlineBeforeModelReview(t *testing.T) {
	parallel(t)
	const folder = "test/intents/steps"
	const stepPath = folder + "/is-even__steps.js"
	const featurePath = ".kogen/intents/is-even/acceptance.feature"
	const feature = `Feature: Inspect credential file bytes
  Scenario Outline: Keep fixture bytes exactly
    Given is-even: a home folder whose ChatGPT credential file contains "<content>"
    Then is-even: the fixture bytes are unchanged

    Examples:
      | content       |
      | not json      |
      | {"client_id": |
      |               |
`
	const steps = `const { Given, Then } = require('@cucumber/cucumber');
const assert = require('node:assert/strict');
Given('is-even: a home folder whose ChatGPT credential file contains {string}', function (raw) { this.raw = raw; });
Then('is-even: the fixture bytes are unchanged', function () {
  assert.ok(['not json', '{"client_id":', ''].includes(this.raw), JSON.stringify(this.raw));
});
`
	candidate := map[string]any{"files": []any{
		map[string]any{"path": project.IntentMDPath(shpSlug), "content": strings.ReplaceAll(shpScopedIntentMD, "features/steps", folder)},
		map[string]any{"path": featurePath, "content": feature},
		map[string]any{"path": stepPath, "content": steps},
	}}
	f := newShpFx(t, shpSubmit("c1", candidate), shpSubmit("c2", candidate), shpSubmit("c3", candidate))
	f.write(project.ConfigPath, strings.ReplaceAll(f.read(project.ConfigPath), "features/steps", folder))
	f.git("add", project.ConfigPath)
	f.git("commit", "-qm", "Use the configured non-default Cucumber folder")
	f.setFeatureReport("report", feature, "PASSED", 0)
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{
		Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "Keep malformed credential file bytes intact.",
		Client: f.Client,
	})
	shpEq(t, "exit", exit, 1)
	problem := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
	shpContains(t, "failed Outline row", strings.Join(problem.Lines, "\n"), `undefined step: Given is-even: a home folder whose ChatGPT credential file contains "{"client_id":" (Examples row line 9)`)
	shpContains(t, "repair hint", strings.Join(problem.Lines, "\n"), "Escape inner quotes inside the {string} argument")
	refused := shpNamed(f.events(), "submission_refused")
	if len(refused) != 3 || refused[2]["gate"] != "step_selfcheck" || len(f.Prov.decisionRequests()) != 0 || len(f.Prov.literalRequests()) != 0 {
		t.Fatalf("malformed Outline was not stopped before review: %v", refused)
	}

	// Run the real CLI as well: escaping the inner quotes binds the step and
	// passes the original malformed JSON bytes, not an escaped variant, to JS.
	node, cucumber := f.S.Getenv("KOGEN_NODE"), f.S.Getenv("KOGEN_CUCUMBER")
	run := func() ([]byte, error) {
		cmd := exec.Command(node, filepath.Join(cucumber, "@cucumber/cucumber/bin/cucumber.js"), "--strict", "--require", stepPath, "--format", "progress", featurePath)
		cmd.Dir = f.Root
		cmd.Env = append(slices.Clone(f.S.Env), "NODE_PATH="+cucumber)
		return cmd.CombinedOutput()
	}
	if output, err := run(); err == nil || !strings.Contains(string(output), "1 undefined") {
		t.Fatalf("Cucumber accepted the unescaped row: %v\n%s", err, output)
	}
	f.write(featurePath, strings.Replace(feature, `{"client_id":`, `{\"client_id\":`, 1))
	if output, err := run(); err != nil || !strings.Contains(string(output), "3 scenarios (3 passed)") {
		t.Fatalf("escaped row did not preserve the fixture bytes: %v\n%s", err, output)
	}
	f.write(stepPath, strings.Replace(steps, "file contains {string}", "file omits {string}", 1))
	if output, err := run(); err == nil || !strings.Contains(string(output), "3 undefined") {
		t.Fatalf("Cucumber accepted an actually unmatched definition: %v\n%s", err, output)
	}
}

func TestShapeStepsDirDefaultsForUnconfiguredAcceptance(t *testing.T) {
	for _, cfg := range []*project.Config{nil, {}, {Acceptance: &project.Acceptance{}}} {
		sh := &shaping{cfg: cfg}
		if got := sh.stepsDir(); got != "features/steps" {
			t.Fatalf("unconfigured steps directory: %q", got)
		}
	}
}

func TestSelfcheckFeedbackNamesDefectsNotWarnings(t *testing.T) {
	parallel(t)
	const own = "features/steps/a_steps.js"
	var raw verify.StepRaw
	err := json.Unmarshal([]byte(`{"loads":[{"file":"`+own+`","error":""}],
	 "defs":[{"file":"`+own+`","pattern":"x","body":"","used":true},{"file":"`+own+`","pattern":"dead","body":"","used":false},
	         {"file":"features/steps/other.js","pattern":"o","body":"","used":false}],
	 "steps":[{"text":"x","count":1},{"text":"y","count":0},{"text":"z","count":2}]}`), &raw)
	if err != nil {
		t.Fatal(err)
	}
	items := verify.StepItems([]string{own}, raw)
	scenarios := []project.Scenario{{Steps: []project.Step{{Keyword: "Then ", Text: "y", Line: 3}}}}
	scenarios = append(scenarios, project.Scenario{Steps: []project.Step{{Keyword: "Then ", Text: "z", Line: 5}}})
	matches := map[string][]verify.StepDef{"z": {{File: own, Pattern: "^z$"}, {File: "features/steps/other.js", Pattern: "z {int}"}}}
	want := selfcheckHeadline + "\nacceptance.feature line 3: undefined step: Then y\nacceptance.feature line 5: ambiguous step: Then z matches 2 definitions: ^z$ in " + own + "; z {int} in features/steps/other.js"
	if got := selfcheckScenarioFeedback(items, scenarios, matches); got != want {
		t.Errorf("feedback %q, want %q", got, want)
	}
	raw.Loads[0].Error = "SyntaxError: boom"
	items = verify.StepItems([]string{own}, raw)
	if f := selfcheckScenarioFeedback(items, scenarios, nil); f != selfcheckHeadline+"\n- load "+own+": SyntaxError: boom" {
		t.Errorf("feedback %q", f)
	}
}

func TestOwnStepFilesAreTheJavaScriptStepFiles(t *testing.T) {
	parallel(t)
	intent := &project.Intent{Slug: "k", Acceptance: []string{project.FeaturePath("k"),
		"features/steps/k_steps.js", "features/steps/k_driver.go", "test/k_helper.js"}}
	if got := verify.OwnStepFiles(intent); !slices.Equal(got, []string{"features/steps/k_steps.js"}) {
		t.Errorf("own step files %v", got)
	}
}

func TestSelfcheckUndefinedLocations(t *testing.T) {
	items := []verify.StepItem{{Check: "steps", Target: "the caller doubles 4", Detail: "undefined step"}}
	scenarios := []project.Scenario{{Steps: []project.Step{{Keyword: "When ", Text: "the caller doubles 4", Line: 14}}}, {Steps: []project.Step{{Keyword: "When ", Text: "the caller doubles 4", Line: 14}}}}
	want := selfcheckHeadline + "\nacceptance.feature line 14: undefined step: When the caller doubles 4"
	if got := selfcheckScenarioFeedback(items, scenarios, nil); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSelfcheckUndefinedOutlineRows(t *testing.T) {
	items := []verify.StepItem{{Check: "steps", Target: "value 1", Detail: "undefined step"}, {Check: "steps", Target: "value 2", Detail: "ambiguous step (2 definitions)"}}
	outline := &project.Outline{Name: "o", Line: 3, Examples: 2}
	scenarios := []project.Scenario{
		{Line: 6, Outline: outline, Steps: []project.Step{{Keyword: "When ", Text: "value 1", Line: 4}, {Keyword: "Then ", Text: "value 2", Line: 5}}},
		{Line: 7, Outline: outline, Steps: []project.Step{{Keyword: "When ", Text: "value 1", Line: 4}, {Keyword: "Then ", Text: "value 2", Line: 5}}},
	}
	matches := map[string][]verify.StepDef{"value 2": {{File: "a.js", Pattern: "^value 2$"}, {File: "b.js", Pattern: "value {int}"}}}
	want := selfcheckHeadline + "\nacceptance.feature line 4: undefined step: When value 1 (Examples row line 6)" +
		"\nacceptance.feature line 4: undefined step: When value 1 (Examples row line 7)" +
		"\nacceptance.feature line 5: ambiguous step: Then value 2 (Examples row line 6) matches 2 definitions: ^value 2$ in a.js; value {int} in b.js" +
		"\nacceptance.feature line 5: ambiguous step: Then value 2 (Examples row line 7) matches 2 definitions: ^value 2$ in a.js; value {int} in b.js"
	if got := selfcheckScenarioFeedback(items, scenarios, matches); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSelfcheckStderrTail(t *testing.T) {
	for _, n := range []int{0, 5, 4095, 4096, 4097} {
		input := strings.Repeat("x", n)
		if n > 0 {
			input = "H" + input[1:]
		}
		got := selfcheckStderrTail(input)
		want := input[max(0, n-4096):]
		if got != want {
			t.Errorf("length %d: got %d bytes", n, len(got))
		}
	}
}

func TestSelfcheckStderrRedactedBeforeTail(t *testing.T) {
	redactor := verify.NewRedactor([]string{"TOKEN=boundary-secret-123456"}, "")
	input := "boundary-secret-123456" + strings.Repeat("x", 4090)
	got := selfcheckStderrTail(redactor.Redact(input))
	if strings.Contains(got, "secret-123456") || strings.Contains(got, "boundary-secret") {
		t.Fatalf("secret fragment in tail: %q", got[:40])
	}
}

func TestSelfcheckObserverDetail(t *testing.T) {
	for _, tc := range []struct{ stderr, want string }{
		{"", "exit 7, recorded exit 0"},
		{"specific failure\n", "exit 7, recorded exit 0\nstderr: specific failure\n"},
	} {
		if got := selfcheckObserverDetail(7, 0, tc.stderr); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestSelfcheckStderrTailUTF8Boundary(t *testing.T) {
	input := "界" + strings.Repeat("x", 4095)
	got := selfcheckStderrTail(input)
	if !utf8.ValidString(got) || len(got) > 4096 || got != strings.Repeat("x", 4095) {
		t.Fatalf("tail must be valid UTF-8 and at most 4096 bytes: length %d, valid %v", len(got), utf8.ValidString(got))
	}
}
