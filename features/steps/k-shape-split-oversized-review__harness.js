'use strict';
// The Go source of the k-shape-split-oversized-review acceptance harness, which
// k-shape-split-oversized-review__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_shape_split_oversized_review_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-executor/agent"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// The acceptance harness of k-shape-split-oversized-review: synthetic shaping runs in a small
// synthetic project with a scripted shaper. KSR_CASE names one case; what the run showed is
// written as JSON to KSR_OUT for the acceptance steps to judge.

const ksrTick = "\x60"

const ksrFeature = "Feature: Is even\n  Scenario: Even numbers\n    Given is-even: the number 4\n    Then is-even: it is even\n  Scenario: Odd numbers\n    Given is-even: the number 3\n    Then is-even: it is not even\n"

const ksrSteps = "// steps\nconst { Given, Then } = require('@cucumber/cucumber');\nGiven('is-even: the number {int}', function () {});\nThen('is-even: it is even', function () {});\nThen('is-even: it is not even', function () {});\n"

const ksrPadding = "features/steps/is-even__padding.js"

// The literal sizes of the tracked files of the synthetic project, in bytes.
const (
	ksrReadme = 3000
	ksrUtil   = 1000
	ksrCore   = 300000
	ksrTable  = 250000
)

// ksrBlob is a file of exactly n bytes that defines nothing.
func ksrBlob(n int) string { return "//" + strings.Repeat("x", n-3) + "\n" }

func ksrName(p string) string { return ksrTick + p + ksrTick }

// ksrIntent is an intent.md of the is-even Intent whose prose is body and whose extra sections
// (such as ## Split) come after the Tests; with padding it lists the padding step file.
func ksrIntent(body, extra string, padding bool) string {
	acceptance := "- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n"
	if padding {
		acceptance += "- " + ksrPadding + "\n"
	}
	return "# Add an is-even check\n\nSlug: is-even\nModel: not applicable — a pure function without state\nAcceptance:\n" + acceptance + "\n" + body + "\n\n" +
		"## Decisions\n- Non-integers raise TypeError. (source: request)\n\n## Source comparison\n- The request asks for is_even; the feature covers it.\n\n" +
		"## Tests\n- is_even returns True for even integers: the acceptance scenarios (scenario)\n\n" + extra + "## Feasibility\n- none\n"
}

// ksrSized is ksrIntent whose prose is filled so that the intent.md is exactly size bytes.
func ksrSized(body, extra string, padding bool, size int) string {
	short := ksrIntent(body+" Filler: .", extra, padding)
	return ksrIntent(body+" Filler: "+strings.Repeat("f", size-len(short))+".", extra, padding)
}

// ksrCandidate is one submitted package and the bytes of mandatory review material it holds:
// its package, the README, the testing rules and the changed files it names.
type ksrCandidate struct {
	files map[string]any
	want  int
}

// ksrCand is the candidate whose intent.md is intent, with a padding step file of fill bytes
// when fill is positive and the named bytes of changed files.
func ksrCand(intent string, fill, named int) ksrCandidate {
	files := []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": ksrFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": ksrSteps},
	}
	want := len(intent) + len(ksrFeature) + len(ksrSteps) + ksrReadme + len(agent.Prompt(agent.TestingRules)) + named
	if fill > 0 {
		files = append(files, map[string]any{"path": ksrPadding, "content": ksrBlob(fill)})
		want += fill
	}
	return ksrCandidate{map[string]any{"files": files}, want}
}

// ksrPart is the estimate of a split part that names the changed files of the given bytes.
func ksrPart(named int) int { return ksrReadme + len(agent.Prompt(agent.TestingRules)) + named }

func TestKShapeSplitOversizedReviewAcceptance(t *testing.T) {
	name, out := os.Getenv("KSR_CASE"), os.Getenv("KSR_OUT")
	ceiling := tools.InputBudgetBytes
	core, table, util := "engine/core.js", "engine/rules.js", "lib/util.js"
	prose := "calc.is_even(n) returns True for even integers; "
	oneFile := prose + "it uses " + ksrName(util) + "."
	bothFiles := prose + "it changes " + ksrName(core) + " and " + ksrName(table) + "."
	coreOnly := prose + "it changes " + ksrName(core) + "."
	splitTable := "## Split\n- engine-rules: Move the rule table in " + ksrName(table) + " out of the engine.\n\n"
	stages := "## Stages\n\n" + ksrTick + ksrTick + ksrTick + "json\n" +
		"{\"version\":1,\"stages\":[{\"id\":\"core\",\"title\":\"Core\",\"goal\":\"Change " + ksrName(core) + "\",\"checks\":[\"node --check engine/core.js\"]}," +
		"{\"id\":\"table\",\"title\":\"Table\",\"goal\":\"Change " + ksrName(table) + "\",\"checks\":[\"node --check engine/rules.js\"]}]}\n" +
		ksrTick + ksrTick + ksrTick + "\n\n"
	ack := result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": ksrFeature}}})
	var cands []ksrCandidate
	parts := map[string]int{}
	switch name {
	case "fits":
		cands = []ksrCandidate{ksrCand(ksrIntent(oneFile, "", false), 0, ksrUtil)}
	case "oversized":
		// Two changed files of 550000 bytes together, then the Intent narrowed to the first with
		// the second as a split part.
		cands = []ksrCandidate{ksrCand(ksrIntent(bothFiles, "", false), 0, ksrCore+ksrTable), ksrCand(ksrIntent(coreOnly, splitTable, false), 0, ksrCore)}
		parts["engine-rules"] = ksrPart(ksrTable)
	case "part-over":
		// A split part that names both large files is itself over the ceiling; then two parts that fit.
		allParts := "## Split\n- engine-all: Rework " + ksrName(core) + " and " + ksrName(table) + " together.\n\n"
		twoParts := "## Split\n- engine-core: Rework " + ksrName(core) + " alone.\n- engine-rules: Move the rule table in " + ksrName(table) + " out of the engine.\n\n"
		cands = []ksrCandidate{ksrCand(ksrIntent(oneFile, allParts, false), 0, ksrUtil), ksrCand(ksrIntent(oneFile, twoParts, false), 0, ksrUtil)}
		parts["engine-all"], parts["engine-core"], parts["engine-rules"] = ksrPart(ksrCore+ksrTable), ksrPart(ksrCore), ksrPart(ksrTable)
	case "stages":
		// A staged plan with one changed file per stage does not lower the estimate of the Intent.
		cands = []ksrCandidate{ksrCand(ksrIntent(bothFiles, stages, false), 0, ksrCore+ksrTable), ksrCand(ksrIntent(coreOnly, splitTable, false), 0, ksrCore)}
		parts["engine-rules"] = ksrPart(ksrTable)
	case "edge":
		// The estimate is one byte over the ceiling in the first candidate and exactly the ceiling
		// in the second.
		fill := ceiling - 4096 - len(ksrFeature) - len(ksrSteps) - ksrReadme - len(agent.Prompt(agent.TestingRules)) - ksrUtil
		cands = []ksrCandidate{ksrCand(ksrSized(oneFile, "", true, 4097), fill, ksrUtil), ksrCand(ksrSized(oneFile, "", true, 4096), fill, ksrUtil)}
		if cands[0].want != ceiling+1 || cands[1].want != ceiling {
			t.Fatalf("TestKShapeSplitOversizedReviewAcceptance: the edge fixture estimates %d and %d bytes, want %d and %d", cands[0].want, cands[1].want, ceiling+1, ceiling)
		}
	default:
		t.Fatalf("TestKShapeSplitOversizedReviewAcceptance: KSR_CASE %q is not a case of the harness", name)
	}
	if out == "" {
		t.Fatal("TestKShapeSplitOversizedReviewAcceptance: KSR_OUT is empty")
	}
	var rounds []func(int, string) (executor.RoundResult, error)
	expected := []int{}
	for _, c := range cands {
		rounds = append(rounds, result(c.files))
		expected = append(expected, c.want)
	}
	if name != "edge" {
		// The acknowledgment of the missing-feature red resubmits the last package unchanged.
		rounds = append(rounds, ack)
		expected = append(expected, cands[len(cands)-1].want)
	}
	f := newShpFx(t)
	f.write("README.md", "# Calc\n"+strings.Repeat("r", ksrReadme-8)+"\n")
	f.write(util, ksrBlob(ksrUtil))
	f.write(core, ksrBlob(ksrCore))
	f.write(table, ksrBlob(ksrTable))
	f.git("add", "-A")
	f.git("commit", "-qm", "scope fixture")
	f.useExternalShape()
	var x *shapeExec
	atSecond := -1
	if len(rounds) > 1 {
		inner := rounds[1]
		rounds[1] = func(n int, root string) (executor.RoundResult, error) {
			if atSecond < 0 {
				atSecond = x.decisionCalls
			}
			return inner(n, root)
		}
	}
	x = &shapeExec{t: t, rounds: rounds}
	exit, err := f.shapeWith(x, "Add calc.is_even(n).")
	o := map[string]any{"case": name, "ceiling": ceiling, "rules": len(agent.Prompt(agent.TestingRules)), "exit": exit, "err": fmt.Sprint(err),
		"class": "", "reason": "", "stdout": f.Stdout.String(), "progress": f.Stderr.String(), "expected": expected, "parts": parts}
	o["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	if e, ok := sys.AsError(err); ok {
		o["class"], o["reason"] = e.Class, e.Reason
	}
	o["intent"] = ""
	if o["published"] == true {
		o["intent"] = f.read(".kogen/intents/is-even/intent.md")
	}
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
	o["shaper"], o["decision"], o["decisionAtSecond"] = x.calls, x.decisionCalls, atSecond
	b, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("TestKShapeSplitOversizedReviewAcceptance: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
