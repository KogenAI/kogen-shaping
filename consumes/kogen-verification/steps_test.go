package consumes

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// Signatures and field types kogen-shaping's calls rely on.
var (
	_ func(string, *project.Intent, []project.Scenario) ([]string, []string, []string)                          = verify.StepAnalysis
	_ func(string, string, *project.Intent, []project.Scenario) ([]string, []string, []string)                  = verify.StepAnalysisIn
	_ func([]string, verify.StepRaw) []verify.StepItem                                                          = verify.StepItems
	_ func([]verify.StepItem) []verify.StepWarning                                                              = verify.StepWarnings
	_ func(context.Context, string, string, string, []string, []string, []string) (verify.StepRaw, bool, error) = verify.RunStepScript
	_ func(verify.StepWarning) string                                                                           = verify.StepWarning.Line
	_ time.Duration                                                                                             = verify.StepTimeout
	_ string                                                                                                    = verify.StepDef{}.File
	_ string                                                                                                    = verify.StepDef{}.Pattern
	_ bool                                                                                                      = verify.StepDef{}.Used
	_ []verify.StepDef                                                                                          = verify.StepRaw{}.Defs
	_ string                                                                                                    = verify.StepItem{}.Check
	_ string                                                                                                    = verify.StepItem{}.Target
	_ bool                                                                                                      = verify.StepItem{}.OK
	_ string                                                                                                    = verify.StepItem{}.Detail
	_ string                                                                                                    = verify.StepItem{}.File
	_ bool                                                                                                      = verify.StepItem{}.Warning
	_ string                                                                                                    = verify.StepWarning{}.Pattern
	_ string                                                                                                    = verify.StepWarning{}.File
	_ string                                                                                                    = verify.StepWarning{}.Detail
	_ string                                                                                                    = verify.StepRecord{}.Keyword
	_ string                                                                                                    = verify.StepRecord{}.Text
	_ int                                                                                                       = verify.StepRecord{}.Line
)

// verify.StepAnalysis gives the load order of an Intent's step files and its step texts: the
// Intent's own JavaScript step files first, then the others of the steps folder in name
// order, and every distinct step text of the scenarios.
func TestStepAnalysis(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"features/steps/z_steps.js", "features/steps/a_steps.js", "features/steps/own_steps.js", "features/steps/driver_test.go.txt", "features/steps/sub/nested.js"} {
		write(t, root, rel, "// x\n")
	}
	intent := &project.Intent{Slug: "demo", Acceptance: []string{
		project.FeaturePath("demo"), "features/steps/own_steps.js", "features/steps/driver_test.go.txt", "docs/other.js",
	}}
	scenarios, err := project.ParseGherkin("Feature: F\n  Scenario: A\n    Given one\n    Then two\n  Scenario: B\n    Given one\n    Then three\n")
	if err != nil {
		t.Fatal(err)
	}
	own, files, texts := verify.StepAnalysis(root, intent, scenarios)
	if want := []string{"features/steps/own_steps.js"}; !equalStrings(own, want) {
		t.Errorf("verify.StepAnalysis: own step files are the Intent's JavaScript files of the steps folder, want %q, got %q", want, own)
	}
	if want := []string{"features/steps/own_steps.js", "features/steps/a_steps.js", "features/steps/z_steps.js"}; !equalStrings(files, want) {
		t.Errorf("verify.StepAnalysis: load order is own files then the other .js files by name, want %q, got %q", want, files)
	}
	if want := []string{"one", "two", "three"}; !equalStrings(texts, want) {
		t.Errorf("verify.StepAnalysis: texts are the distinct step texts in order, want %q, got %q", want, texts)
	}

	own, files, texts = verify.StepAnalysis(t.TempDir(), intent, nil)
	if len(own) != 1 || !equalStrings(files, own) || len(texts) != 0 {
		t.Errorf("verify.StepAnalysis: without a steps folder or scenarios only the own files remain, got %q %q %q", own, files, texts)
	}
}

// verify.StepItems turns what the Node side observed into the items of the step self-check.
func TestStepItems(t *testing.T) {
	own := []string{"features/steps/own_steps.js"}
	raw := verify.StepRaw{
		Loads: []verify.StepLoad{{File: own[0]}},
		Steps: []verify.StepText{{Text: "good", Count: 1}, {Text: "lost", Count: 0}, {Text: "twice", Count: 2}},
		Defs: []verify.StepDef{
			{File: own[0], Pattern: "good", Body: "assert.ok(true)", Used: true},
			{File: own[0], Pattern: "spare", Body: "assert.ok(true)", Used: false},
			{File: "features/steps/other_steps.js", Pattern: "foreign", Used: false},
			{File: own[0], Pattern: "exits", Body: "assert.equal(r.status, 0); spawn('go run .')", Used: true},
		},
	}
	items := verify.StepItems(own, raw)
	want := []verify.StepItem{
		{Check: "load", Target: own[0], OK: true},
		{Check: "steps", Target: "good", OK: true},
		{Check: "steps", Target: "lost", OK: false, Detail: "undefined step"},
		{Check: "steps", Target: "twice", OK: false, Detail: "ambiguous step (2 definitions)"},
		{Check: "definitions", Target: "good", OK: true, File: own[0]},
		{Check: "definitions", Target: "spare", OK: true, File: own[0], Detail: "unused definition", Warning: true},
		{Check: "definitions", Target: "exits", OK: true, File: own[0]},
		{Check: "exit_code", Target: own[0], OK: false, Detail: "step definition exits asserts an exit code but obtains it through go run, which changes exit codes; build the binary (go build) and run it instead"},
	}
	if !slices.Equal(items, want) {
		t.Errorf("verify.StepItem: items must be load, steps, definitions of own files, exit_code:\n got: %+v\nwant: %+v", items, want)
	}

	raw.Loads = []verify.StepLoad{{File: own[0], Error: "SyntaxError: Unexpected token"}}
	items = verify.StepItems(own, raw)
	if want := []verify.StepItem{{Check: "load", Target: own[0], OK: false, Detail: "SyntaxError: Unexpected token"}}; !slices.Equal(items, want) {
		t.Errorf("verify.StepItems: a file that does not load leaves the other checks out, got %+v", items)
	}
}

// verify.StepWarnings are the unused definitions among the items, each one line of output.
func TestStepWarnings(t *testing.T) {
	items := []verify.StepItem{
		{Check: "steps", Target: "good", OK: true},
		{Check: "definitions", Target: "spare", OK: true, File: "features/steps/own_steps.js", Detail: "unused definition", Warning: true},
		{Check: "ownership", Target: "foreign", File: "features/steps/other.js", Detail: "outside"},
	}
	got := verify.StepWarnings(items)
	want := []verify.StepWarning{{Pattern: "spare", File: "features/steps/own_steps.js", Detail: "unused definition"}}
	if !slices.Equal(got, want) {
		t.Fatalf("verify.StepWarnings: only the Warning items count, want %+v, got %+v", want, got)
	}
	if line, wantLine := got[0].Line(), "warning: step definition spare in features/steps/own_steps.js: unused definition"; line != wantLine {
		t.Errorf("verify.StepWarning.Line: want %q, got %q", wantLine, line)
	}
	if got := verify.StepWarnings(items[:1]); len(got) != 0 {
		t.Errorf("verify.StepWarnings: no warning items give none, got %+v", got)
	}
}

// verify.RunStepScript loads the step files with the Node it is given and reads the result
// from the last line it prints; a Node that cannot run leaves the failure as the load error
// of every file.
func TestRunStepScript(t *testing.T) {
	root := t.TempDir()
	stdin := filepath.Join(t.TempDir(), "stdin.json")
	node := filepath.Join(t.TempDir(), "node")
	script := "#!/bin/sh\ncat > '" + stdin + "'\n" +
		`printf 'noise\nKOGEN_SELFCHECK_RESULT:{"loads":[{"file":"a.js","error":""}],"defs":[{"file":"%s","pattern":"p","body":"b","used":true}],"steps":[{"text":"t","count":1}]}\n' "$NODE_PATH|$KOGEN_MARK|$(pwd -P)"` + "\n"
	if err := os.WriteFile(node, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "KOGEN_MARK=marked")
	raw, loaded, err := verify.RunStepScript(context.Background(), node, "/cucumber/node_modules", root, env, []string{"a.js"}, []string{"t"})
	if err != nil || !loaded {
		t.Fatalf("verify.RunStepScript: a Node that prints the result must load, got loaded %v err %v", loaded, err)
	}
	resolved, _ := filepath.EvalSymlinks(root)
	if want := []verify.StepDef{{File: "/cucumber/node_modules|marked|" + resolved, Pattern: "p", Body: "b", Used: true}}; !slices.Equal(raw.Defs, want) {
		t.Errorf("verify.StepRaw.Defs: the definitions come from the result line (NODE_PATH = the Cucumber folder, env passed, run in root), want %+v, got %+v", want, raw.Defs)
	}
	if len(raw.Loads) != 1 || raw.Loads[0].File != "a.js" || len(raw.Steps) != 1 || raw.Steps[0].Count != 1 {
		t.Errorf("verify.RunStepScript: loads and steps must come from the result, got %+v", raw)
	}
	if got := readFile(t, filepath.Dir(stdin), "stdin.json"); got != `{"files":["a.js"],"texts":["t"]}` {
		t.Errorf("verify.RunStepScript: the files and texts go to Node as JSON on stdin, got %q", got)
	}

	raw, loaded, err = verify.RunStepScript(context.Background(), filepath.Join(root, "no-such-node"), "/c", root, os.Environ(), []string{"a.js", "b.js"}, nil)
	if err != nil || loaded || len(raw.Loads) != 2 || raw.Loads[0].File != "a.js" || raw.Loads[1].File != "b.js" ||
		raw.Loads[0].Error == "" || raw.Loads[0].Error != raw.Loads[1].Error || len(raw.Defs) != 0 {
		t.Errorf("verify.RunStepScript: a Node that cannot run must give every file the same load error, not loaded, got loaded %v err %v %+v", loaded, err, raw)
	}
	if !strings.Contains(raw.Loads[0].Error, "no-such-node") {
		t.Errorf("verify.RunStepScript: the load error must name what failed, got %q", raw.Loads[0].Error)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, loaded, err := verify.RunStepScript(cancelled, node, "/c", root, os.Environ(), []string{"a.js"}, nil); loaded || err != context.Canceled {
		t.Errorf("verify.RunStepScript: a cancelled context must return its error, got loaded %v err %v", loaded, err)
	}
}

// verify.StepTimeout is the 120 seconds each approval step may take; kogen-shaping reuses it
// as the bound of its unpassable-step control.
func TestStepTimeout(t *testing.T) {
	if verify.StepTimeout != 120*time.Second {
		t.Errorf("verify.StepTimeout: want 120s, got %v", verify.StepTimeout)
	}
}
