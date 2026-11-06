'use strict';
// The Go source of the k-shape-scope-gate-counts-package acceptance harness, which
// k-shape-scope-gate-counts-package__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_scope_gate_counts_package_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// The acceptance harness of k-shape-scope-gate-counts-package: synthetic shaping runs in a small
// synthetic project with a scripted shaper. KGP_CASE names one case; what the run showed is
// written as JSON to KGP_OUT for the acceptance steps to judge.

const kgpBacktick = "\x60"

const kgpFeature = "Feature: Is even\n  Scenario: Even numbers\n    Given is-even: the number 4\n    Then is-even: it is even\n  Scenario: Odd numbers\n    Given is-even: the number 3\n    Then is-even: it is not even\n"

const kgpOutline = "  Scenario Outline: Even <n>\n    Given is-even: the number <n>\n    Then is-even: it is even\n    Examples:\n      | n |\n      | 2 |\n      | 6 |\n      | 8 |\n"

const kgpSteps = "// steps\nconst { Given, Then } = require('@cucumber/cucumber');\nGiven('is-even: the number {int}', function () {});\nThen('is-even: it is even', function () {});\nThen('is-even: it is not even', function () {});\n"

const kgpModel = "module is_even {\n var count: int\n action init = count' = 0\n action step = count' = count + 1\n val Safety = count >= 0\n}\n"

const kgpPadding = "features/steps/is-even__padding.js"

// kgpIntent is an intent.md of the is-even Intent whose prose is body; with model it declares
// .kogen/intents/is-even/model.qnt, and with padding it lists the padding step file.
func kgpIntent(body string, model, padding bool) string {
	modelLine := "Model: not applicable — a pure function without state\n"
	if model {
		modelLine = "Model: .kogen/intents/is-even/model.qnt invariants=Safety seed=7 samples=2 steps=2\n"
	}
	acceptance := "- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n"
	if padding {
		acceptance += "- " + kgpPadding + "\n"
	}
	return "# Add an is-even check\n\nSlug: is-even\n" + modelLine + "Acceptance:\n" + acceptance + "\n" + body + "\n\n" +
		"## Decisions\n- Non-integers raise TypeError. (source: request)\n\n## Source comparison\n- The request asks for is_even; the feature covers it.\n\n" +
		"## Tests\n- is_even returns True for even integers: the acceptance scenarios (scenario)\n\n## Feasibility\n- none\n"
}

// kgpSized is kgpIntent whose prose is filled so that the intent.md is exactly size bytes.
func kgpSized(body string, model, padding bool, size int) string {
	short := kgpIntent(body+" Filler: .", model, padding)
	return kgpIntent(body+" Filler: "+strings.Repeat("f", size-len(short))+".", model, padding)
}

func kgpName(p string) string { return kgpBacktick + p + kgpBacktick }

func kgpFiles(intent, feature string, model bool, fill int) map[string]any {
	files := []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": feature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": kgpSteps},
	}
	if fill > 0 {
		files = append(files, map[string]any{"path": kgpPadding, "content": kgpFill(fill)})
	}
	if model {
		files = append(files, map[string]any{"path": ".kogen/intents/is-even/model.qnt", "content": kgpModel})
	}
	return map[string]any{"files": files}
}

// kgpFill is a step file of exactly size bytes that defines nothing.
func kgpFill(size int) string { return "//" + strings.Repeat("x", size-3) + "\n" }

func TestKShapeScopeGateCountsPackageAcceptance(t *testing.T) {
	name, out := os.Getenv("KGP_CASE"), os.Getenv("KGP_OUT")
	bound := tools.InputBudgetBytes
	big := "data/big.json"
	named := "calc.is_even(n) returns True for even integers; its table is " + kgpName(big) + "."
	var rounds []func(int, string) (executor.RoundResult, error)
	notes := false
	switch name {
	case "named":
		// A small package that names a tracked file larger than the bound, then the acknowledgment
		// of its missing-feature red.
		rounds = append(rounds, result(kgpFiles(kgpIntent(named, false, false), kgpFeature, false, 0)),
			result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": kgpFeature}}}))
	case "edge":
		// The package's own bytes (intent.md, the feature with an outline, two step files and the
		// model) are one byte over the bound in c1 and exactly the bound in c2. The Intent names a
		// tracked file larger than the bound, a path not at HEAD and a tracked path under .kogen/.
		body := named + " It adds " + kgpName("missing/new.js") + " and keeps " + kgpName(".kogen/notes.txt") + "."
		feature := kgpFeature + kgpOutline
		fill := bound - 4096 - len(feature) - len(kgpSteps) - len(kgpModel)
		notes = true
		rounds = append(rounds, result(kgpFiles(kgpSized(body, true, true, 4097), feature, true, fill)),
			result(kgpFiles(kgpSized(body, true, true, 4096), feature, true, fill)))
	case "over":
		// A package whose own bytes are over the bound, submitted unchanged until shaping ends.
		files := kgpFiles(kgpIntent("calc.is_even(n) returns True for even integers.", false, true), kgpFeature, false, bound+1000)
		rounds = append(rounds, result(files))
	default:
		t.Fatalf("TestKShapeScopeGateCountsPackageAcceptance: KGP_CASE %q is not a case of the harness", name)
	}
	if out == "" {
		t.Fatal("TestKShapeScopeGateCountsPackageAcceptance: KGP_OUT is empty")
	}
	f := newShpFx(t)
	f.write(big, "["+strings.Repeat("0,", (bound+4096-3)/2)+"]\n")
	if notes {
		f.write(".kogen/notes.txt", "notes\n")
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "scope fixture")
	f.useExternalShape()
	x := &shapeExec{t: t, rounds: rounds}
	exit, err := f.shapeWith(x, "Add calc.is_even(n).")
	o := map[string]any{"case": name, "bound": bound, "exit": exit, "err": fmt.Sprint(err), "class": "", "reason": "",
		"stdout": f.Stdout.String(), "progress": f.Stderr.String()}
	o["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	if e, ok := sys.AsError(err); ok {
		o["class"], o["reason"] = e.Class, e.Reason
	}
	o["big"] = len(f.read(big))
	feedback := []string{}
	for _, in := range x.inputs[1:] {
		text, _, _ := strings.Cut(in.Feedback, "\n\nCumulative proposal manifest: ")
		feedback = append(feedback, text)
	}
	o["feedback"] = feedback
	gates := []string{}
	for _, ev := range shpNamed(f.events(), "submission_refused") {
		gates = append(gates, fmt.Sprint(ev["gate"]))
	}
	o["gates"] = gates
	o["shaper"], o["decision"] = x.calls, x.decisionCalls
	b, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("TestKShapeScopeGateCountsPackageAcceptance: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
