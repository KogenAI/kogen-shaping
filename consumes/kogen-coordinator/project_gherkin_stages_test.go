package consumes

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

func TestParseGherkin(t *testing.T) {
	scenarios, err := project.ParseGherkin(featureText)
	if err != nil {
		t.Fatalf("project.ParseGherkin: %v", err)
	}
	if len(scenarios) != 3 {
		t.Fatalf("project.ParseGherkin returned %d scenarios, want 3 (one Scenario, two Examples rows)", len(scenarios))
	}
	plain, one, two := scenarios[0], scenarios[1], scenarios[2]
	eq(t, "project.Scenario.Name of a plain scenario", plain.Name, "Make a widget")
	eq(t, "project.Scenario.Line of a plain scenario", plain.Line, 4)
	if plain.Outline != nil {
		t.Errorf("project.Scenario.Outline of a plain scenario = %+v, want nil", plain.Outline)
	}
	type step struct {
		kw, text string
		line     int
	}
	steps := func(s project.Scenario) (out []step) {
		for _, st := range s.Steps {
			out = append(out, step{st.Keyword, st.Text, st.Line})
		}
		return out
	}
	// Background steps come first; the keyword keeps its trailing space.
	want := []step{{"Given ", "a widget bench", 3}, {"When ", "I make a widget", 5}, {"Then ", "the widget exists", 6}}
	if got := steps(plain); !equalSteps(got, want) {
		t.Errorf("project.Scenario.Steps = %v, want %v (project.Step.Keyword, project.Step.Text, project.Step.Line)", got, want)
	}
	for i, st := range plain.Steps {
		eq(t, "project.Step.Keyword", st.Keyword, want[i].kw)
		eq(t, "project.Step.Text", st.Text, want[i].text)
		eq(t, "project.Step.Line", st.Line, want[i].line)
	}
	eq(t, "project.Scenario.Name of the first Examples row", one.Name, "Make 1 widgets")
	eq(t, "project.Scenario.Line of the first Examples row", one.Line, 12)
	eq(t, "project.Scenario.Name of the second Examples row", two.Name, "Make 2 widgets")
	eq(t, "project.Scenario.Line of the second Examples row", two.Line, 13)
	if one.Outline == nil || one.Outline.Name != "Make <n> widgets" || two.Outline == nil || two.Outline.Name != "Make <n> widgets" {
		t.Errorf("project.Scenario.Outline = %+v and %+v, want the outline's own name with its placeholders (project.Outline.Name)", one.Outline, two.Outline)
	}
	want = []step{{"Given ", "a widget bench", 3}, {"When ", "I make 2 widgets", 8}, {"Then ", "2 widgets exist", 9}}
	if got := steps(two); !equalSteps(got, want) {
		t.Errorf("project.Scenario.Steps of an Examples row = %v, want the substituted steps %v", got, want)
	}

	_, err = project.ParseGherkin("Scenario: S\n  Then T\n")
	if err == nil || err.Error() != "line 1: expected Feature:, got: Scenario: S" {
		t.Errorf("project.ParseGherkin of a file without Feature: = %v, want the line and the problem", err)
	}
	if _, err := project.ParseGherkin("Feature: F\n  Scenario Outline: O\n    Then T\n"); err == nil || !strings.Contains(err.Error(), "has no Examples rows") {
		t.Errorf("project.ParseGherkin of an outline without rows = %v, want an error naming the missing rows", err)
	}
}

func equalSteps[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const stagesText = "# Plan\n\n## Stages\n\n```json\n" + `{"version":1,"stages":[
 {"id":"first","title":"First stage","goal":"Do the first thing","checks":["go build ./...","go test ./a"]},
 {"id":"second-one","title":"Second","goal":"Finish","checks":["go test ./..."]}
]}` + "\n```\n\n## Tests\n- x\n"

func TestParseStages(t *testing.T) {
	stages, err := project.ParseStages(stagesText)
	if err != nil {
		t.Fatalf("project.ParseStages: %v", err)
	}
	if len(stages) != 2 {
		t.Fatalf("project.ParseStages returned %d stages, want 2", len(stages))
	}
	first, second := stages[0], stages[1]
	eq(t, "project.Stage.ID", first.ID, "first")
	eq(t, "project.Stage.Title", first.Title, "First stage")
	eq(t, "project.Stage.Goal", first.Goal, "Do the first thing")
	eq(t, "project.Stage.Checks (in order)", first.Checks, []string{"go build ./...", "go test ./a"})
	eq(t, "project.Stage.ID of the second stage", second.ID, "second-one")
	eq(t, "project.Stage.Checks of the second stage", second.Checks, []string{"go test ./..."})

	// No section: the single whole-Intent route, nil without an error.
	if stages, err := project.ParseStages("# T\n\n## Tests\n- x\n"); stages != nil || err != nil {
		t.Errorf("project.ParseStages without a Stages section = %v, %v, want nil, nil", stages, err)
	}
	// A heading inside a code fence does not start a section.
	if stages, err := project.ParseStages("# T\n```\n## Stages\n```\n"); stages != nil || err != nil {
		t.Errorf("project.ParseStages of a fenced heading = %v, %v, want nil, nil", stages, err)
	}
	for name, text := range map[string]string{
		"one stage":   "## Stages\n\n```json\n" + `{"version":1,"stages":[{"id":"a","title":"A","goal":"g","checks":["c"]}]}` + "\n```\n",
		"not json":    "## Stages\n\nplain text\n",
		"extra key":   "## Stages\n\n```json\n" + `{"version":1,"stages":[{"id":"a","title":"A","goal":"g","checks":["c"],"x":1},{"id":"b","title":"B","goal":"g","checks":["c"]}]}` + "\n```\n",
		"duplicate":   stagesText + "\n## Stages\n",
		"same id":     "## Stages\n\n```json\n" + `{"version":1,"stages":[{"id":"a","title":"A","goal":"g","checks":["c"]},{"id":"a","title":"B","goal":"g","checks":["c"]}]}` + "\n```\n",
		"no checks":   "## Stages\n\n```json\n" + `{"version":1,"stages":[{"id":"a","title":"A","goal":"g","checks":[]},{"id":"b","title":"B","goal":"g","checks":["c"]}]}` + "\n```\n",
		"bad id":      "## Stages\n\n```json\n" + `{"version":1,"stages":[{"id":"A b","title":"A","goal":"g","checks":["c"]},{"id":"b","title":"B","goal":"g","checks":["c"]}]}` + "\n```\n",
		"bad version": "## Stages\n\n```json\n" + `{"version":2,"stages":[{"id":"a","title":"A","goal":"g","checks":["c"]},{"id":"b","title":"B","goal":"g","checks":["c"]}]}` + "\n```\n",
	} {
		stages, err := project.ParseStages(text)
		if err == nil || stages != nil || !strings.HasPrefix(err.Error(), "stages:") {
			t.Errorf("project.ParseStages of %s = %v, %v, want no stages and an error that starts \"stages:\"", name, stages, err)
		}
	}
}
