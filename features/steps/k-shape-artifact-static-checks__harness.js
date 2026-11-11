'use strict';
// The Go source of the k-shape-artifact-static-checks acceptance harness, which
// k-shape-artifact-static-checks__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_shape_artifact_static_checks_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

// The acceptance harness of k-shape-artifact-static-checks: one native Shape per case of a small
// Go repository (module example.test/staticshape, Count returning 0), with a scripted shaper, the
// repository's own .kogen/project.yaml and the real Cucumber acceptance command. KSSC_CASES names
// the cases (they run in parallel), and what a user and the shaper see of each run is written as
// JSON to KSSC_OUT/<case>.json for the acceptance steps to judge.

const (
	ksscSlug     = "static-proof"
	ksscRequest  = "make Count return the slice length"
	ksscDriver   = "features/steps/static-proof__driver/main.go"
	ksscCounter  = "features/steps/static-proof__counter/main.go"
	ksscFeatureP = ".kogen/intents/static-proof/acceptance.feature"
	ksscStepsP   = "features/steps/static-proof__steps.js"

	ksscGoMod     = "module example.test/staticshape\n\ngo 1.27\n"
	ksscCount     = "package count\n\nfunc Count(items []string) int { return 0 }\n"
	ksscUnrelated = "package count;var Existing=1\n"
	ksscLayout    = "test -z \"$(gofmt -l count features/steps)\""
	ksscAnalysis  = "go vet ./..."
	ksscAccept    = "NODE_PATH=\"$KOGEN_CUCUMBER\" \"$KOGEN_NODE\" \"$KOGEN_CUCUMBER/@cucumber/cucumber/bin/cucumber.js\" --strict --require \"features/steps/*.js\" --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\""

	// The helper as the request gives it: unformatted, its gofmt-clean form, the uncalled function
	// with the vet defect and its repair, and both defects in one file.
	ksscUnformatted = "package main\nimport(\"fmt\";\"example.test/staticshape/count\")\nfunc main(){fmt.Println(count.Count([]string{\"a\",\"b\"}))}\n"
	ksscFormatted   = "package main\n\nimport (\n\t\"fmt\"\n\n\t\"example.test/staticshape/count\"\n)\n\nfunc main() {\n\tfmt.Println(count.Count([]string{\"a\", \"b\"}))\n}\n"
	ksscDiagnostic  = "\nfunc diagnostic() {\n\tfmt.Printf(\"%d\", \"not-a-number\")\n}\n"
	ksscRepaired    = "\nfunc diagnostic() {\n\tfmt.Printf(\"%s\", \"not-a-number\")\n}\n"
	ksscBoth        = ksscUnformatted + "func diagnostic(){fmt.Printf(\"%d\",\"not-a-number\")}\n"

	ksscFeature = "Feature: Count items\n  Scenario: Observe the count\n    When static-proof: I count two items\n    Then static-proof: the count is 2\n"
)

func ksscConfig(fix, layout, analysis string) string {
	return "version: 1\ncommands:\n  setup: []\n  fix: " + fix + "\n  checks:\n    - name: " + layout + "\n      run: '" + ksscLayout +
		"'\n    - name: " + analysis + "\n      run: '" + ksscAnalysis + "'\n  acceptance:\n    run: '" + ksscAccept +
		"'\n    steps_dir: features/steps\nprotect: []\n"
}

func ksscIntent(helper string) string {
	return "# Make Count return the slice length\n\nSlug: static-proof\nModel: not applicable — Count is a pure function without state\nAcceptance:\n" +
		"- " + ksscFeatureP + "\n- " + ksscStepsP + "\n- " + helper + "\n\n" +
		"Count returns the number of items in its slice. The step module runs the helper, which prints the count of two items.\n\n" +
		"## Decisions\n- Count returns the length of its slice. (source: request)\n\n" +
		"## Source comparison\n- The request asks Count to return the slice length; the scenario counts two items.\n\n" +
		"## Tests\n- Count returns the slice length: the acceptance scenario counts two items (scenario)\n\n## Feasibility\n- none\n"
}

func ksscSteps(dir string) string {
	return "const assert = require('node:assert/strict');\nconst { execFileSync } = require('node:child_process');\nconst { When, Then } = require('@cucumber/cucumber');\n" +
		"When('static-proof: I count two items', function () {\n  this.value = execFileSync('go', ['run', './" + dir + "'], {\n    cwd: process.cwd(), encoding: 'utf8'\n  }).trim();\n});\n" +
		"Then('static-proof: the count is {int}', function (expected) {\n  assert.equal(this.value, String(expected));\n});\n"
}

func ksscFile(path, content string) map[string]any {
	return map[string]any{"path": path, "content": content}
}

// ksscFull is a whole package with helper (at its path) holding body.
func ksscFull(helper, body string) []any {
	return []any{ksscFile(project.IntentMDPath(ksscSlug), ksscIntent(helper)), ksscFile(ksscFeatureP, ksscFeature),
		ksscFile(ksscStepsP, ksscSteps(filepath.Dir(helper))), ksscFile(helper, body)}
}

type ksscCase struct {
	helper    string
	fix       bool // fix: ['gofmt -w features/steps'] instead of fix: []
	unrelated bool // count/unrelated.go, unformatted, is committed in the baseline
	absent    bool // gofmt is on no directory of PATH
	renamed   bool // the checks are named tidy and inspect instead of layout and analysis
	approve   bool // approve the published package
	change    bool // after publication, add the vet defect to the helper and approve the old and the new hash
	subs      [][]any
}

func ksscCases() map[string]ksscCase {
	ack := []any{ksscFile(ksscFeatureP, ksscFeature)}
	return map[string]ksscCase{
		"repair": {helper: ksscDriver, approve: true, subs: [][]any{ksscFull(ksscDriver, ksscUnformatted), ksscFull(ksscDriver, ksscUnformatted), ack,
			{ksscFile(ksscDriver, ksscFormatted)}, ack}},
		"vet":       {helper: ksscDriver, subs: [][]any{ksscFull(ksscDriver, ksscFormatted+ksscDiagnostic), {ksscFile(ksscDriver, ksscFormatted+ksscRepaired)}, ack}},
		"fixer":     {helper: ksscDriver, fix: true, change: true, subs: [][]any{ksscFull(ksscDriver, ksscUnformatted), ack}},
		"unrelated": {helper: ksscDriver, unrelated: true, subs: [][]any{ksscFull(ksscDriver, ksscFormatted), ack}},
		"absent":    {helper: ksscDriver, unrelated: true, absent: true, subs: [][]any{ksscFull(ksscDriver, ksscFormatted), ack}},
		"renamed":   {helper: ksscCounter, renamed: true, subs: [][]any{ksscFull(ksscCounter, ksscBoth), {ksscFile(ksscCounter, ksscFormatted+ksscRepaired)}, ack}},
		"valid":     {helper: ksscCounter, renamed: true, subs: [][]any{ksscFull(ksscCounter, ksscFormatted), ack}},
	}
}

func TestKShapeArtifactStaticChecksAcceptance(t *testing.T) {
	names, out := os.Getenv("KSSC_CASES"), os.Getenv("KSSC_OUT")
	if names == "" || out == "" {
		t.Fatalf("k-shape-artifact-static-checks: KSSC_CASES (%q) and KSSC_OUT (%q) must both be set", names, out)
	}
	cases := ksscCases()
	gocache := ksscGo(t, "env", "GOCACHE")
	for _, name := range strings.Split(names, ",") {
		c, ok := cases[name]
		if !ok {
			t.Fatalf("k-shape-artifact-static-checks: %q is not a case of the harness", name)
		}
		t.Run(name, func(t *testing.T) { ksscShape(t, name, c, gocache, out) })
	}
}

// ksscGo runs go with the test's own environment and returns its trimmed output.
func ksscGo(t *testing.T, args ...string) string {
	t.Helper()
	b, err := exec.Command("go", args...).Output()
	if err != nil || strings.TrimSpace(string(b)) == "" {
		t.Fatalf("k-shape-artifact-static-checks: go %s: %v %q", strings.Join(args, " "), err, b)
	}
	return strings.TrimSpace(string(b))
}

// ksscSetenv replaces or adds each KEY=value of kv in env.
func ksscSetenv(env []string, kv ...string) []string {
	out := slices.Clone(env)
	for _, pair := range kv {
		key := pair[:strings.Index(pair, "=")+1]
		out = slices.DeleteFunc(out, func(e string) bool { return strings.HasPrefix(e, key) })
		out = append(out, pair)
	}
	return out
}

func ksscHas(dir, name string) bool {
	fi, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}

// ksscPathWithout is PATH without the directories holding tool, with go still reachable.
func ksscPathWithout(t *testing.T, tool string) string {
	t.Helper()
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d != "" && !ksscHas(d, tool) {
			dirs = append(dirs, d)
		}
	}
	if !slices.ContainsFunc(dirs, func(d string) bool { return ksscHas(d, "go") }) {
		bin := t.TempDir()
		if err := os.Symlink(filepath.Join(ksscGo(t, "env", "GOROOT"), "bin", "go"), filepath.Join(bin, "go")); err != nil {
			t.Fatalf("k-shape-artifact-static-checks: linking go for a PATH without %s: %v", tool, err)
		}
		dirs = append([]string{bin}, dirs...)
	}
	path := strings.Join(dirs, string(os.PathListSeparator))
	cmd := exec.Command("/bin/sh", "-c", "if command -v "+tool+" >/dev/null 2>&1; then echo "+tool+" is still reachable; exit 1; fi; go version")
	cmd.Env = ksscSetenv(os.Environ(), "PATH="+path)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("k-shape-artifact-static-checks: the PATH without %s is not usable (%v): %s\nPATH=%s", tool, err, b, path)
	}
	return path
}

func ksscError(err error) any {
	if err == nil {
		return nil
	}
	if e, ok := sys.AsError(err); ok {
		return map[string]any{"class": e.Class, "reason": e.Reason, "detail": e.Detail, "lines": e.Lines}
	}
	return map[string]any{"text": err.Error()}
}

// ksscRead is the content of each path, nil when it does not exist.
func ksscRead(root string, paths []string) map[string]any {
	seen := map[string]any{}
	for _, p := range paths {
		if b, err := os.ReadFile(filepath.Join(root, p)); err == nil {
			seen[p] = string(b)
		} else {
			seen[p] = nil
		}
	}
	return seen
}

// ksscPackage is the approval hash of the Intent's baseline and the SHA-256 of each of its files.
func ksscPackage(t *testing.T, root string) map[string]any {
	t.Helper()
	cfg, err := project.LoadConfig(root)
	if err != nil {
		t.Fatalf("k-shape-artifact-static-checks: reading the configuration of %s: %v", root, err)
	}
	baseline, err := project.Baseline(root, ksscSlug, cfg)
	if err != nil {
		t.Fatalf("k-shape-artifact-static-checks: the baseline of %s: %v", ksscSlug, err)
	}
	files := map[string]any{}
	for _, p := range baseline {
		if b, err := os.ReadFile(filepath.Join(root, p)); err == nil {
			sum := sha256.Sum256(b)
			files[p] = hex.EncodeToString(sum[:])
		}
	}
	return map[string]any{"sha256": project.Hash(root, baseline), "files": files}
}

func ksscApprove(f *shpFx, hash string) map[string]any {
	f.newSession()
	exit, err := Approve(context.Background(), f.S, ApproveInput{Project: f.Root, Slug: ksscSlug, Hash: hash, By: "Fixture"})
	_, statErr := os.Stat(project.ApprovalPath(f.Root, ksscSlug))
	return map[string]any{"hash": hash, "exit": exit, "error": ksscError(err), "stdout": f.Stdout.String(), "stderr": f.Stderr.String(), "approved": statErr == nil}
}

func ksscShape(t *testing.T, name string, c ksscCase, gocache, out string) {
	steps := make([]shpStep, len(c.subs))
	for i, files := range c.subs {
		steps[i] = shpSubmit(fmt.Sprintf("c%d", i+1), map[string]any{"files": files})
	}
	f := newShpFx(t, steps...)
	f.git("rm", "-q", "accept.sh", "calc.js", "app.txt", "features/steps/calc_steps.js")
	f.write("README.md", "# staticshape\n\nCounts items.\n")
	f.write("go.mod", ksscGoMod)
	f.write("count/count.go", ksscCount)
	if c.unrelated {
		f.write("count/unrelated.go", ksscUnrelated)
	}
	layout, analysis, fix := "layout", "analysis", "[]"
	if c.renamed {
		layout, analysis = "tidy", "inspect"
	}
	if c.fix {
		fix = "['gofmt -w features/steps']"
	}
	f.write(".kogen/project.yaml", ksscConfig(fix, layout, analysis))
	f.git("add", "-A")
	f.git("commit", "-qm", "small Go repository")
	env := []string{"GOCACHE=" + gocache, "GOFLAGS=", "GOTOOLCHAIN=local"}
	if c.absent {
		env = append(env, "PATH="+ksscPathWithout(t, "gofmt"))
	}
	f.Env = ksscSetenv(f.Env, env...)
	watched := []string{"go.mod", "count/count.go", "count/unrelated.go", ".kogen/project.yaml"}
	before := ksscRead(f.Root, watched)

	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Slug: ksscSlug, Request: ksscRequest, Client: f.Client})
	seen := map[string]any{"case": name, "root": f.Root, "helper": c.helper, "exit": exit, "error": ksscError(err),
		"stdout": f.Stdout.String(), "stderr": f.Stderr.String(), "before": before,
		"checks": []any{map[string]any{"name": layout, "run": ksscLayout}, map[string]any{"name": analysis, "run": ksscAnalysis}}}
	published := exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve "+ksscSlug)
	seen["published"] = published
	seen["after"] = ksscRead(f.Root, append(slices.Clone(watched), c.helper))

	feedback, shaper := map[string]any{}, 0
	for _, r := range f.Prov.allRequests() {
		if contractKind(r) != "" || decisionRequest(r) || literalRequest(r) || slices.Contains(shpToolNames(r), SubmitQuestionCheckTool) {
			continue
		}
		shaper++
		for id, o := range shpOutputs(r) {
			if _, ok := feedback[id]; !ok && !strings.HasSuffix(id, "-inspect") {
				feedback[id] = o
			}
		}
	}
	seen["shaper"], seen["feedback"] = shaper, feedback

	gates, events := []any{}, []any{}
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping", ksscSlug, "events.ndjson")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			switch ev["event"] {
			case "submission_refused":
				gates = append(gates, map[string]any{"attempt": ev["attempt"], "gate": ev["gate"], "problems": ev["problems"]})
			case "static_checks":
				events = append(events, ev)
			}
		}
	}
	seen["gates"], seen["events"] = gates, events
	var record any
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/static-checks", ksscSlug+".json")); err == nil {
		json.Unmarshal(b, &record)
	}
	seen["record"] = record

	pkg := ksscPackage(t, f.Root)
	seen["package"] = pkg
	approvals := []any{}
	if published && c.approve {
		approvals = append(approvals, ksscApprove(f, pkg["sha256"].(string)))
	}
	if published && c.change {
		f.write(c.helper, f.read(c.helper)+ksscDiagnostic)
		changed := ksscPackage(t, f.Root)
		approvals = append(approvals, ksscApprove(f, pkg["sha256"].(string)), ksscApprove(f, changed["sha256"].(string)))
	}
	seen["approvals"] = approvals

	b, err := json.MarshalIndent(seen, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(out, name+".json"), b, 0o644)
	}
	if err != nil {
		t.Fatalf("k-shape-artifact-static-checks: writing the observations of case %s: %v", name, err)
	}
}
`;
