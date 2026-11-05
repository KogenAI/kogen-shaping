package consumes

import (
	"slices"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

func TestLoadIntent(t *testing.T) {
	dir, _ := newProject(t)
	in, err := project.LoadIntent(dir, slug)
	if err != nil {
		t.Fatalf("project.LoadIntent: %v", err)
	}
	eq(t, "project.Intent.Root", in.Root, dir)
	eq(t, "project.Intent.Slug", in.Slug, slug)
	if in.Text != intentText("The request maps to the widget.") {
		t.Errorf("project.Intent.Text is not the exact bytes of intent.md: %q", in.Text)
	}
	if in.Title != "Add the widget" {
		t.Errorf("project.Intent.Title = %q, want the # heading without its marker", in.Title)
	}
	if in.SlugLine != " widget" {
		t.Errorf("project.Intent.SlugLine = %q, want the Slug: line's text as written (with its space)", in.SlugLine)
	}
	if in.ModelLine != "not applicable — the widget has no state" || in.Model == nil || !in.Model.None {
		t.Errorf("project.Intent.ModelLine, Model = %q, %+v, want the line and a ModelSpec with None (project.ModelSpec.None)", in.ModelLine, in.Model)
	}
	if want := []string{".kogen/intents/widget/acceptance.feature", "features/steps/widget__steps.js"}; !slices.Equal(in.Acceptance, want) {
		t.Errorf("project.Intent.Acceptance = %q, want %q", in.Acceptance, want)
	}
	if want := []string{"- Make widgets on request (source: request)"}; !slices.Equal(in.Decisions, want) {
		t.Errorf("project.Intent.Decisions = %q, want the bullet lines of the section: %q", in.Decisions, want)
	}
	if want := []string{"- none"}; !slices.Equal(in.Feasibility, want) {
		t.Errorf("project.Intent.Feasibility = %q, want %q", in.Feasibility, want)
	}
	if want := []string{"- Widgets are made: the widget scenario (scenario)"}; !slices.Equal(in.Tests, want) {
		t.Errorf("project.Intent.Tests = %q, want %q", in.Tests, want)
	}
	if want := []string{"features/steps/widget__steps.js"}; !slices.Equal(in.StepFiles(), want) {
		t.Errorf("project.Intent.StepFiles = %q, want the Acceptance list without the feature: %q", in.StepFiles(), want)
	}

	_, err = project.LoadIntent(dir, "absent")
	se, ok := sys.AsError(err)
	if !ok || se.Class != "intent" || se.Reason != "parse" || se.Exit != 1 || !slices.Equal(se.Lines, []string{"intent.md does not exist"}) {
		t.Errorf("project.LoadIntent of a missing Intent = %v, want intent/parse exit 1 with the line \"intent.md does not exist\"", err)
	}
}

func TestReadIntent(t *testing.T) {
	dir := t.TempDir() // nothing on disk: the overlay holds the file Shape has proposed
	text := `# Draft

Slug: widget
Model: .kogen/intents/widget/model.qnt invariants=Safe,Live seed=7 samples=20 steps=5
Acceptance:

- .kogen/intents/widget/acceptance.feature
- features/a.js
- features/b.js

Not a bullet.
- features/after-break.js

## Decisions
- first (source: request)

not a bullet
- second (source: answer 1)
  wrapped
## Feasibility
## Tests
- none: no behaviour
`
	overlay := map[string][]byte{project.IntentMDPath(slug): []byte(text)}
	in, err := project.ReadIntent(dir, slug, overlay)
	if err != nil {
		t.Fatalf("project.ReadIntent: %v", err)
	}
	m := in.Model
	if m == nil {
		t.Fatalf("project.Intent.Model is nil for a readable Model: line")
	}
	eq(t, "project.ModelSpec.None", m.None, false)
	eq(t, "project.ModelSpec.Path", m.Path, ".kogen/intents/widget/model.qnt")
	eq(t, "project.ModelSpec.Invariants", m.Invariants, []string{"Safe", "Live"})
	eq(t, "project.ModelSpec.Seed", m.Seed, "7")
	eq(t, "project.ModelSpec.Samples", m.Samples, "20")
	eq(t, "project.ModelSpec.Steps", m.Steps, "5")
	if want := []string{".kogen/intents/widget/acceptance.feature", "features/a.js", "features/b.js"}; !slices.Equal(in.Acceptance, want) {
		t.Errorf("project.Intent.Acceptance = %q, want the bullets after the header up to the first blank line (a wrapped line continues its bullet): %q", in.Acceptance, want)
	}
	if want := []string{"- first (source: request)", "- second (source: answer 1) wrapped"}; !slices.Equal(in.Decisions, want) {
		t.Errorf("project.Intent.Decisions = %q, want %q", in.Decisions, want)
	}
	if len(in.Feasibility) != 0 {
		t.Errorf("project.Intent.Feasibility = %q, want none for an empty section", in.Feasibility)
	}
	if want := []string{"- none: no behaviour"}; !slices.Equal(in.Tests, want) {
		t.Errorf("project.Intent.Tests = %q, want %q", in.Tests, want)
	}
	if want := []string{"features/a.js", "features/b.js"}; !slices.Equal(in.StepFiles(), want) {
		t.Errorf("project.Intent.StepFiles = %q, want %q", in.StepFiles(), want)
	}

	// A Model: line that cannot be read leaves the line and no spec.
	unreadable := map[string][]byte{project.IntentMDPath(slug): []byte("# T\n\nModel: whatever\n")}
	in, err = project.ReadIntent(dir, slug, unreadable)
	if err != nil || in.Model != nil || in.ModelLine != "whatever" {
		t.Errorf("project.ReadIntent of an unreadable Model line = %+v, %v, want ModelLine \"whatever\" and no Model", in, err)
	}
	// Fields whose line is missing are empty.
	in, err = project.ReadIntent(dir, slug, map[string][]byte{project.IntentMDPath(slug): []byte("no heading\n")})
	if err != nil || in.Title != "" || in.SlugLine != "" || in.ModelLine != "" || in.Model != nil || len(in.Acceptance)+len(in.Decisions)+len(in.Feasibility)+len(in.Tests) != 0 {
		t.Errorf("project.ReadIntent of a bare text = %+v, %v, want empty fields", in, err)
	}
}

func TestIntentExistsAndDir(t *testing.T) {
	dir, _ := newProject(t)
	if !project.IntentExists(dir, slug) || project.IntentExists(dir, "absent") {
		t.Errorf("project.IntentExists = %v for the Intent and %v for an absent one, want true and false", project.IntentExists(dir, slug), project.IntentExists(dir, "absent"))
	}
	write(t, dir, ".kogen/intents/folder-only/acceptance.feature", featureText)
	if project.IntentExists(dir, "folder-only") {
		t.Errorf("project.IntentExists of a folder without intent.md must be false")
	}
	if err := project.RequireIntentDir(dir, "folder-only"); err != nil {
		t.Errorf("project.RequireIntentDir of an existing folder: %v", err)
	}
	if err := project.RequireIntentDir(dir, slug); err != nil {
		t.Errorf("project.RequireIntentDir of the Intent: %v", err)
	}
	se, ok := sys.AsError(project.RequireIntentDir(dir, "absent"))
	if !ok || se.Class != "intent" || se.Reason != "not_found" || se.Exit != 2 || se.Detail != "Intent does not exist" {
		t.Errorf("project.RequireIntentDir of a missing folder = %+v, want intent/not_found exit 2 \"Intent does not exist\"", se)
	}
	// A file where the folder should be is no folder.
	write(t, dir, ".kogen/intents/a-file", "x")
	if _, ok := sys.AsError(project.RequireIntentDir(dir, "a-file")); !ok {
		t.Errorf("project.RequireIntentDir of a regular file must fail")
	}
}

func TestCheckIntent(t *testing.T) {
	dir, _ := newProject(t)
	in, scenarios, err := project.CheckIntent(dir, slug, nil)
	if err != nil {
		t.Fatalf("project.CheckIntent of a valid Intent: %v", err)
	}
	if in.Slug != slug || in.Title != "Add the widget" {
		t.Errorf("project.CheckIntent returned the Intent %q %q, want widget \"Add the widget\"", in.Slug, in.Title)
	}
	var names []string
	for _, s := range scenarios {
		names = append(names, s.Name)
	}
	if want := []string{"Make a widget", "Make 1 widgets", "Make 2 widgets"}; !slices.Equal(names, want) {
		t.Errorf("project.CheckIntent scenarios = %q, want one per Scenario and per Examples row: %q", names, want)
	}

	// Files not yet written come from the overlay.
	overlay := map[string][]byte{project.IntentMDPath("fresh"): []byte(`# Fresh

Slug: fresh
Model: not applicable — none
Acceptance:
- .kogen/intents/fresh/acceptance.feature
- features/steps/fresh__steps.js

## Feasibility
- none

## Tests
- none: nothing to test
`), project.FeaturePath("fresh"): []byte("Feature: F\n  Scenario: S\n    Then T\n"), "features/steps/fresh__steps.js": []byte("//\n")}
	if _, scenarios, err := project.CheckIntent(dir, "fresh", overlay); err != nil || len(scenarios) != 1 || scenarios[0].Name != "S" {
		t.Errorf("project.CheckIntent with an overlay = %v, %v, want the overlay's scenario S", scenarios, err)
	}
}

func TestCheckIntentStops(t *testing.T) {
	dir, _ := newProject(t)
	feature := []byte("Feature: F\n  Scenario: S\n    When W\n")
	broken := []byte(`# A title that ends with a period.

Slug: other
Model: nope
Acceptance:
- .kogen/intents/widget/acceptance.feature
- features/steps/missing.js

## Decisions
- no source here

## Tests
- not a test line
`)
	overlay := map[string][]byte{project.IntentMDPath(slug): broken, project.FeaturePath(slug): feature}
	in, scenarios, err := project.CheckIntent(dir, slug, overlay)
	se, ok := sys.AsError(err)
	if !ok || in != nil || scenarios != nil {
		t.Fatalf("project.CheckIntent of a broken Intent = %v, %v, %v, want only an error", in, scenarios, err)
	}
	if se.Class != "intent" || se.Reason != "lint" || se.Exit != 1 || se.Detail != "the Intent needs changes" {
		t.Errorf("project.CheckIntent stop = %s/%s exit %d %q, want intent/lint exit 1 \"the Intent needs changes\"", se.Class, se.Reason, se.Exit, se.Detail)
	}
	want := []string{
		"title: the title must not end with a period",
		"slug: the Slug: line must be widget",
		"model: the Model: line must be `not applicable — <reason>` or `.kogen/intents/widget/model.qnt invariants=<Name> seed=<n> samples=<n> steps=<n>`",
		"tests: - not a test line must be `- <rule>: <test to add or update> (<layer>)` or `- none: <reason>`",
		"acceptance: features/steps/missing.js does not exist",
		"decisions: - no source here must end with `(source: request)` or `(source: answer <n>)`",
		"feasibility: intent.md needs a ## Feasibility section listing each probe, or - none",
		"acceptance: scenario S has no Then step",
	}
	if !slices.Equal(se.Lines, want) {
		t.Errorf("project.CheckIntent problems =\n%q\nwant one line per problem, in this order:\n%q", se.Lines, want)
	}

	// An Intent that cannot be read at all is intent/parse.
	for _, c := range []struct {
		name, slug string
		overlay    map[string][]byte
		line       string
	}{
		{"no intent.md", "absent", map[string][]byte{}, "intent.md does not exist"},
		{"no feature", "nofeature", map[string][]byte{project.IntentMDPath("nofeature"): []byte("# T\n")}, "acceptance.feature does not exist"},
		{"broken feature", "badfeature", map[string][]byte{project.IntentMDPath("badfeature"): []byte("# T\n"), project.FeaturePath("badfeature"): []byte("Scenario: S\n")}, "acceptance.feature line 1: expected Feature:, got: Scenario: S"},
	} {
		_, _, err := project.CheckIntent(dir, c.slug, c.overlay)
		se, ok := sys.AsError(err)
		if !ok || se.Class != "intent" || se.Reason != "parse" || se.Exit != 1 || se.Detail != "the Intent cannot be read" || len(se.Lines) == 0 || se.Lines[0] != c.line {
			t.Errorf("project.CheckIntent with %s = %v, want intent/parse exit 1 whose first line is %q (a broken feature also lists the lint problems after it)", c.name, err, c.line)
		}
	}
}
