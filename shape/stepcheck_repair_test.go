package shape

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

func TestShapeGreetStepCorrection(t *testing.T) {
	const feature = "Feature: Empty name\n  Scenario: Greet an empty name\n    When is-even: I call Greet with name \"\"\n    Then is-even: the greeting is \"Hello, world\"\n"
	const bad = `const { When, Then } = require('@cucumber/cucumber');
const { spawnSync } = require('child_process');
const assert = require('assert');
When('is-even: I call Greet with name {string}', function (name) {
  const result = spawnSync('sh', ['-c', 'go run . "$1"', 'greet', name]);
  assert.strictEqual(result.status, 0);
  this.greeting = result.stdout.toString().trim();
});
Then('is-even: the greeting is {string}', function (greeting) { assert.strictEqual(this.greeting, greeting); });
`
	const fixed = `const { When, Then } = require('@cucumber/cucumber');
const { spawnSync } = require('child_process');
const assert = require('assert');
When('is-even: I call Greet with name {string}', function (name) {
  const dir = require('fs').mkdtempSync(require('path').join(require('os').tmpdir(), 'greet-'));
  const binary = require('path').join(dir, 'greet');
  const build = spawnSync('go', ['build', '-o', binary, '.']);
  assert.strictEqual(build.status, 0);
  const result = spawnSync(binary, [name]);
  require('fs').rmSync(dir, {recursive: true, force: true});
  assert.strictEqual(result.status, 0);
  this.greeting = result.stdout.toString().trim();
});
Then('is-even: the greeting is {string}', function (greeting) { assert.strictEqual(this.greeting, greeting); });
`
	for _, exhaust := range []bool{false, true} {
		name := "repair"
		if exhaust {
			name = "exhaust"
		}
		t.Run(name, func(t *testing.T) {
			f := newShpFx(t)
			f.write("go.mod", "module example.com/greet\n\ngo 1.22\n")
			f.write("greet.go", "package main\n\nimport (\"fmt\"; \"os\")\nfunc Greet(name string) string { if name == \"\" { name = \"world\" }; return \"Hello, \" + name }\nfunc main() { fmt.Println(Greet(os.Args[1])) }\n")
			f.write("greet_test.go", "package main\nimport \"testing\"\nfunc TestGreet(t *testing.T) { if Greet(\"Ada\") != \"Hello, Ada\" { t.Fatal(\"wrong greeting\") } }\n")
			f.git("add", "go.mod", "greet.go", "greet_test.go")
			f.git("commit", "-qm", "greet fixture")
			scenarios, err := project.ParseGherkin(feature)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(f.Aux, "report"), []byte(shpMessages(t, scenarios, "PASSED")), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(f.Aux, "exit"), []byte("0"), 0644); err != nil {
				t.Fatal(err)
			}
			files := shpFiles()
			for _, raw := range files["files"].([]any) {
				m := raw.(map[string]any)
				switch m["path"] {
				case ".kogen/intents/is-even/intent.md":
					m["content"] = strings.NewReplacer("Add an is-even check", "Greet empty names", "features/steps/is-even__steps.js", "features/steps/is-even__empty-name.js", "calc.is_even(n) returns True for even integers.", `Greet returns "Hello, world" for an empty name.`, "Non-integers raise TypeError.", "An empty name uses world.", "The request asks for is_even; the feature covers it.", "The request asks for an empty-name greeting; the feature covers it.", "is_even returns True for even integers: add test_is_even in tests/test_calc.py", "Greet returns Hello, world for an empty name: add a test in greet_test.go").Replace(shpScopedIntentMD)
				case ".kogen/intents/is-even/acceptance.feature":
					m["content"] = feature
				case "features/steps/is-even__steps.js":
					m["path"] = "features/steps/is-even__empty-name.js"
					m["content"] = bad
				}
			}
			corrected := map[string]any{"files": []any{map[string]any{"path": "features/steps/is-even__empty-name.js", "content": fixed}}}
			if exhaust {
				f.Prov.seq = []shpStep{shpSubmit("bad", files), shpSubmit("again", files), shpSubmit("last", files)}
			} else {
				f.Prov.seq = []shpStep{shpSubmit("bad", files), shpSubmit("fixed", corrected)}
			}
			exit, err := f.shape(`Greet should return "Hello, world" when the name is empty`)
			reqs := f.Prov.requests()
			want := 2
			if exhaust {
				want = 3
			}
			if len(reqs) != want {
				t.Fatalf("requests = %d, want %d; err=%v", len(reqs), want, err)
			}
			feedback := shpOutputs(reqs[1])["bad"]
			for _, part := range []string{"exit_code", "features/steps/is-even__empty-name.js", "go run", "go build"} {
				if !strings.Contains(feedback, part) {
					t.Errorf("feedback %q missing %q", feedback, part)
				}
			}
			if exhaust {
				e := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
				if !strings.Contains(strings.Join(e.Lines, "\n"), "exit_code features/steps/is-even__empty-name.js") {
					t.Errorf("remaining: %v", e.Lines)
				}
				if exit != 1 {
					t.Errorf("exit = %d", exit)
				}
				if len(shpNamed(f.events(), "intent_published")) != 0 || f.exists(".kogen/local/readiness/is-even.json") || f.exists(".kogen/local/provenance/is-even/current.json") {
					t.Fatal("failed package was published")
				}
			} else {
				if err != nil || exit != 0 {
					t.Fatalf("exit %d: %v", exit, err)
				}
				if got := f.read("features/steps/is-even__empty-name.js"); got != fixed {
					t.Fatalf("published step = %q", got)
				}
				checks := shpNamed(f.events(), "step_selfcheck")
				if len(checks) != 2 || checks[0]["ok"] != false || checks[1]["ok"] != true {
					t.Errorf("step checks: %v", checks)
				}
			}
		})
	}
}
