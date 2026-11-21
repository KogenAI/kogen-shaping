package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-verification/verify"
)

const repairDeclared = `This request repairs the failing check "Greeting is polite" on main.`

// repairJourney shapes the synthetic red-main fixture once through the real entry. first and
// second are the expected outputs of the "Greeting is polite" and "Other stays" scenarios on a
// main whose f returns "rude-" + x.
type repairResult struct {
	published          bool
	feedback, progress string
}

func repairJourney(t *testing.T, request, first, second string) repairResult {
	t.Helper()
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("testdata", "repair_red_main", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	f := newShpFx(t)
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	modules := os.Getenv("KOGEN_CUCUMBER")
	f.write("f.js", "exports.f = x => 'rude-' + x;\n")
	f.write("features/steps/calc_steps.js", "// existing\n")
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
	f.git("add", "-A")
	f.git("commit", "-qm", "red main fixture")
	feature := strings.NewReplacer("{{A}}", first, "{{B}}", second).Replace(read("acceptance.feature.tmpl"))
	intent := strings.NewReplacer("{{REQUEST}}", request).Replace(read("intent.md.tmpl"))
	pack := map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feature},
		map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": "{\"deferred\":\"synthetic fixture\"}\n"},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": read("steps.js")},
	}}
	prop := func(name, scenario string, line int) map[string]any {
		return map[string]any{"property": name, "request": request, "kind": "kept", "observable": true, "scenarios": []any{scenario}, "decision": "", "issues": []any{}, "kept_steps": []any{line}}
	}
	review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{prop("Greeting stays checked", "Greeting is polite", 3), prop("Other output stays", "Other stays", 5)}}
	verdicts := []any{}
	for i := 1; i <= 8; i++ {
		verdicts = append(verdicts, map[string]any{"finding": i, "verdict": "refuted", "why": "synthetic"})
	}
	for i := 0; i < 4; i++ {
		f.Prov.contract = append(f.Prov.contract, shpAnswer(shpCall(fmt.Sprint("review", i), SubmitContractReviewTool, review)), shpAnswer(shpCall(fmt.Sprint("confirm", i), SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts})))
	}
	f.Prov.seq = []shpStep{shpSubmit("first", pack), shpSubmit("second", pack), shpSubmit("third", pack)}
	exit, err := f.shape(request)
	var outputs []string
	for _, r := range f.Prov.requests() {
		if s, ok := shpOutputs(r)["first"]; ok {
			outputs = append(outputs, s)
		}
	}
	return repairResult{
		published: exit == 0 && err == nil && f.exists(".kogen/local/readiness/is-even.json"),
		feedback:  strings.Join(outputs, "\n") + "\n" + fmt.Sprint(err),
		progress:  f.Stderr.String(),
	}
}

func TestShapeRepairRedMainTarget(t *testing.T) {
	r := repairJourney(t, "Make the greeting polite. "+repairDeclared, "polite-A", "rude-B")
	if !r.published {
		t.Fatalf("red main with a declared repair was refused: %s", r.feedback)
	}
	var lines []string
	for _, l := range strings.Split(r.progress, "\n") {
		if strings.Contains(l, "repair target") {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 || !strings.Contains(lines[0], `"Greeting is polite"`) || !strings.Contains(lines[0], "red before") || !strings.Contains(lines[0], "green after") {
		t.Fatalf("repair target progress lines: %q", lines)
	}
}

func TestShapeRepairKeepsOtherChecks(t *testing.T) {
	r := repairJourney(t, "Make the greeting polite. "+repairDeclared, "polite-A", "wrong-B")
	if r.published {
		t.Fatal("a kept check that is red on the unchanged tree was published")
	}
	if !strings.Contains(r.feedback, "Other stays") || !strings.Contains(r.feedback, "red on the unchanged tree") {
		t.Fatalf("the broken kept check is not refused: %s", r.feedback)
	}
	if strings.Contains(r.feedback, "Greeting is polite (kept)") {
		t.Fatalf("the repair target was treated as kept: %s", r.feedback)
	}
}

func TestShapeRepairNamedCheckGreen(t *testing.T) {
	green := &verify.Ready{Scenarios: []verify.ScenarioResult{{Name: "Greeting is polite", Status: "passed"}, {Name: "Other", Status: "failed"}}}
	got := observeRepair(repairNames("Fix it. "+repairDeclared), green)
	if len(got.targets) != 0 || len(got.refusals) != 1 || !strings.Contains(got.refusals[0], "not failing") || !strings.Contains(got.refusals[0], "Greeting is polite") {
		t.Fatalf("green named check: %+v", got)
	}
	red := &verify.Ready{Scenarios: []verify.ScenarioResult{{Name: "Greeting is polite", Status: "failed"}}}
	if got := observeRepair(repairNames(repairDeclared), red); !reflect.DeepEqual(got.targets, []string{"Greeting is polite"}) || len(got.refusals) != 0 {
		t.Fatalf("red named check: %+v", got)
	}
	if got := observeRepair(repairNames(repairDeclared), nil); len(got.targets)+len(got.refusals) != 0 {
		t.Fatalf("no observation: %+v", got)
	}
	r := repairJourney(t, "Make the greeting polite. "+repairDeclared, "rude-A", "rude-B")
	if r.published || !strings.Contains(r.feedback, "Greeting is polite") || !strings.Contains(r.feedback, "not failing") {
		t.Fatalf("a named check that is green was not refused (published=%v): %s", r.published, r.feedback)
	}
}

func TestShapeRepairDeclarationForms(t *testing.T) {
	for request, want := range map[string][]string{
		repairDeclared: {"Greeting is polite"},
		`It repairs the failing checks "A", "B" and "C  d" on main.`: {"A", "B", "C d"},
		`This repairs the failing check "A" on the main branch.`:     {"A"},
		`Make it polite; the check "Greeting is polite" is red.`:     nil,
		`Repair the failing check "A" in the shaper.`:                nil,
		`Make the greeting polite.`:                                  nil,
	} {
		if got := repairNames(request); !reflect.DeepEqual(got, want) {
			t.Errorf("repairNames(%q) = %v, want %v", request, got, want)
		}
	}
}

func TestShapeRedMainWithoutRepairRefused(t *testing.T) {
	r := repairJourney(t, "Make the greeting polite.", "polite-A", "rude-B")
	if r.published {
		t.Fatal("a red main without a repair declaration was published")
	}
	if !strings.Contains(r.feedback, "Greeting is polite") || !strings.Contains(r.feedback, "red on the unchanged tree") {
		t.Fatalf("red main is not refused as before: %s", r.feedback)
	}
	if strings.Contains(r.progress, "repair target") {
		t.Fatalf("a repair target was recorded without a declaration: %s", r.progress)
	}
}

func TestShapeReportsOwnTimeAndTimeoutApart(t *testing.T) {
	_, ev := stcShape(t, []string{"slow", "ok"}, map[string]stcSpec{"slow": {120 * time.Second, 137, true}, "ok": {time.Second, 0, false}})
	seconds, ok := ev["own check seconds"].(float64)
	if !ok || seconds < 120 {
		t.Fatalf("own check seconds = %v", ev["own check seconds"])
	}
	if got := ev["acceptance timeout"]; !reflect.DeepEqual(got, []any{"slow"}) {
		t.Fatalf("acceptance timeout = %v, want [slow]", got)
	}
	_, ev = stcShape(t, []string{"ok"}, map[string]stcSpec{"ok": {time.Second, 0, false}})
	if got, ok := ev["acceptance timeout"].([]any); !ok || len(got) != 0 {
		t.Fatalf("acceptance timeout without a timeout = %v, want an empty array", ev["acceptance timeout"])
	}
}
