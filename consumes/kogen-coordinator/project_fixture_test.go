package consumes

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

const slug = "widget"

const projectYAML = `version: 1
commands:
  setup:
    - 'npm ci'
  fix:
    - 'gofmt -w .'
  checks:
    - name: format
      run: 'test -z "$(gofmt -l .)"'
    - name: go-test
      run: 'go test ./...'
  acceptance:
    run: 'cucumber-js'
    steps_dir: 'features/steps/'
protect:
  - '.golangci.yml'
  - 'testdata/**'
models:
  shape:
    provider: anthropic
    model: m-shape
    effort: high
  build:
    executor: claude-code
    model: m-build
    effort: medium
  review:
    provider: openai-compatible
    base_url: http://localhost:11434/v1/
    model: m-review
    effort: low
`

// modelsYAML is the team lineup, which the baseline holds right after the configuration.
const modelsYAML = `models:
  shape:
    model: m-shape
    effort: high
`

const featureText = `Feature: Widget
  Background:
    Given a widget bench
  Scenario: Make a widget
    When I make a widget
    Then the widget exists
  Scenario Outline: Make <n> widgets
    When I make <n> widgets
    Then <n> widgets exist
    Examples:
      | n |
      | 1 |
      | 2 |
`

// intentText is a valid Intent package text for slug "widget".
func intentText(comparison string) string {
	return `# Add the widget

Slug: widget
Model: not applicable — the widget has no state
Acceptance:
- .kogen/intents/widget/acceptance.feature
- features/steps/widget__steps.js

Widgets are made on request.

## Decisions
- Make widgets on request (source: request)

## Feasibility
- none

## Tests
- Widgets are made: the widget scenario (scenario)

## Source comparison
` + comparison + "\n"
}

// newProject is a committed checkout holding .kogen/project.yaml, the widget Intent package
// and its step file.
func newProject(t *testing.T) (string, *project.Config) {
	t.Helper()
	dir := newRepo(t)
	write(t, dir, project.ConfigPath, projectYAML)
	write(t, dir, ".kogen/models.yaml", modelsYAML)
	write(t, dir, ".gitignore", "*.log\n")
	write(t, dir, project.IntentMDPath(slug), intentText("The request maps to the widget."))
	write(t, dir, project.FeaturePath(slug), featureText)
	write(t, dir, project.IntentDir(slug)+"/notes.md", "notes\n")
	write(t, dir, "features/steps/widget__steps.js", "// steps\n")
	write(t, dir, ".golangci.yml", "run: {}\n")
	write(t, dir, "testdata/a/golden.txt", "golden\n")
	write(t, dir, "README.md", "readme\n")
	commitAll(t, dir, "project")
	cfg, err := project.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, cfg
}

func lines(s string) []string { return strings.Split(strings.TrimSuffix(s, "\n"), "\n") }
