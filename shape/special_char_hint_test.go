package shape

import (
	"context"
	"github.com/KogenAI/kogen-coordinator/project"
	"strings"
	"testing"
)

func TestSpecialCharHintRegression(t *testing.T) {
	for _, tc := range []struct {
		text string
		hint bool
	}{
		{"the refusal in setup/init_route.go is provider/login", true},
		{"the refusal in build/unit_test.go is provider/login", true},
		{"the refusal (local) is literal", true},
		{"the refusal {mystery} is literal", true},
		{"the refusal is ordinary", false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			text := "is-even: " + tc.text
			pattern := text
			if !tc.hint {
				pattern = "is-even: a different ordinary step"
			}
			feature := "Feature: Hint\n  Scenario: Undefined\n    Then " + text + "\n"
			steps := "const {Then}=require('@cucumber/cucumber');\nThen('" + pattern + "', function() {});\n"
			candidate := map[string]any{"files": []any{
				map[string]any{"path": project.IntentMDPath(shpSlug), "content": shpScopedIntentMD},
				map[string]any{"path": project.FeaturePath(shpSlug), "content": feature},
				map[string]any{"path": "features/steps/is-even__steps.js", "content": steps},
			}}
			f := newShpFx(t, shpSubmit("c1", candidate), shpSubmit("c2", candidate), shpSubmit("c3", candidate))
			f.newSession()
			_, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "Explain undefined literal steps.", Client: f.Client})
			problem := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
			feedback := strings.Join(problem.Lines, "\n")
			if !strings.Contains(feedback, "undefined step:") {
				t.Fatalf("did not reach undefined step: %s", feedback)
			}
			got := strings.Contains(feedback, "Cucumber Expressions read")
			if got != tc.hint {
				t.Fatalf("special character hint = %v, want %v: %s", got, tc.hint, feedback)
			}
			if tc.hint {
				for _, part := range []string{"alternation", "optional text", "parameters", `\\/`, `\\(`, `\\)`, "rephrase"} {
					if !strings.Contains(feedback, part) {
						t.Fatalf("missing hint guidance %q: %s", part, feedback)
					}
				}
			}
		})
	}
}

func TestLiteralExpressionDiagnostics(t *testing.T) {
	f := newShpFx(t)
	node, cucumber := f.S.Getenv("KOGEN_NODE"), f.S.Getenv("KOGEN_CUCUMBER")
	for _, tc := range []struct {
		name, definition, text string
		want                   bool
	}{
		{"slash", `Then('is-even: file a/b', function() {});`, "is-even: file a/b", true},
		{"parentheses-full", `Then('is-even: the refusal (local) is literal', function() {});`, "is-even: the refusal (local) is literal", true},
		{"parentheses", `Then('is-even: (local)', function() {});`, "is-even: (local)", true},
		{"unknown", `Then('is-even: {mystery}', function() {});`, "is-even: {mystery}", true},
		{"parameter", `Then('is-even: {int} a/b', function() {});`, "is-even: 42 a/b", true},
		{"mismatch", `Then('is-even: different a/b', function() {});`, "is-even: file a/b", false},
		{"escaped", `Then('is-even: file a\\/b', function() {});`, "is-even: file a/b", false},
		{"regex", `Then(/^is-even: file a\/b$/, function() {});`, "is-even: file a/b", false},
		{"parameter not literal", `Then('is-even: {int} a/b', function() {});`, "is-even: {int} a/b", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := "features/steps/is-even__steps.js"
			f.write(file, "const {Then} = require('@cucumber/cucumber');\n"+tc.definition)
			got := literalExpressionHint(context.Background(), node, cucumber, f.Root, f.S.Env, []string{file}, []string{file}, []string{tc.text})
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestSpecialCharHintOnce(t *testing.T) {
	const feature = "Feature: Hint\n  Scenario: Undefined\n    Then is-even: first a/b\n    Then is-even: second c/d\n"
	const steps = "const {Then}=require('@cucumber/cucumber');\nThen('is-even: first a/b', function() {});\nThen('is-even: second c/d', function() {});\n"
	candidate := map[string]any{"files": []any{
		map[string]any{"path": project.IntentMDPath(shpSlug), "content": shpScopedIntentMD},
		map[string]any{"path": project.FeaturePath(shpSlug), "content": feature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": steps},
	}}
	f := newShpFx(t, shpSubmit("c1", candidate), shpSubmit("c2", candidate), shpSubmit("c3", candidate))
	f.newSession()
	_, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug, Request: "Explain undefined literal steps.", Client: f.Client})
	problem := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
	feedback := strings.Join(problem.Lines, "\n")
	if strings.Count(feedback, "Cucumber Expressions read") != 1 {
		t.Fatalf("expected one hint: %s", feedback)
	}
}
