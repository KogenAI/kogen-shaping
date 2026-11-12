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

// ksrSourceRoom is the bytes the review keeps for source under the ceiling (MaterialPartBytes).
const ksrSourceRoom = 6000

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

// ksrCandidate is one submitted package and the bytes of mandatory review material (the authority)
// it holds: its package, the README and the testing rules. Changed files it names are not in it.
type ksrCandidate struct {
	files map[string]any
	want  int
}

// ksrCand is the candidate whose intent.md is intent, with a padding step file of fill bytes
// when fill is positive; the files it names are not counted.
func ksrCand(intent string, fill, named int) ksrCandidate {
	files := []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": ksrFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": ksrSteps},
	}
	want := len(intent) + len(ksrFeature) + len(ksrSteps) + ksrReadme + len(agent.Prompt(agent.TestingRules))
	if fill > 0 {
		files = append(files, map[string]any{"path": ksrPadding, "content": ksrBlob(fill)})
		want += fill
	}
	return ksrCandidate{map[string]any{"files": files}, want}
}

// ksrPart is the estimate of a split part: the review inputs alone, whatever files it names.
func ksrPart() int { return ksrReadme + len(agent.Prompt(agent.TestingRules)) }

func TestKShapeSplitOversizedReviewAcceptance(t *testing.T) {
	name, out := os.Getenv("KSR_CASE"), os.Getenv("KSR_OUT")
	ceiling := tools.InputBudgetBytes
	limit := ceiling - ksrSourceRoom
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
		cands = []ksrCandidate{ksrCand(ksrIntent(oneFile, "", false), 0, 0)}
	case "oversized":
		// An authority one byte over the limit that names two changed files, then the Intent narrowed
		// to the first with the second as a split part and a package that fits (a small padding file
		// stays declared, as a proposed file may not be dropped).
		fill := limit - 4096 - len(ksrFeature) - len(ksrSteps) - ksrReadme - len(agent.Prompt(agent.TestingRules))
		cands = []ksrCandidate{ksrCand(ksrSized(bothFiles, "", true, 4097), fill, 0), ksrCand(ksrIntent(coreOnly, splitTable, true), 100, 0)}
		if cands[0].want != limit+1 {
			t.Fatalf("TestKShapeSplitOversizedReviewAcceptance: the oversized fixture estimates %d bytes, want %d", cands[0].want, limit+1)
		}
		parts["engine-rules"] = ksrPart()
	case "many-parts":
		// An authority exactly at the limit leaves the review 6000 bytes of source per part; two
		// changed files at the 32 KiB allowance each are 11 parts, over MaxReviewParts. Then the
		// Intent narrowed to one small file with the others as split parts.
		fill := limit - 4096 - len(ksrFeature) - len(ksrSteps) - ksrReadme - len(agent.Prompt(agent.TestingRules))
		twoParts := "## Split\n- engine-core: Rework " + ksrName(core) + " alone.\n- engine-rules: Move the rule table in " + ksrName(table) + " out of the engine.\n\n"
		cands = []ksrCandidate{ksrCand(ksrSized(bothFiles, "", true, 4096), fill, 0), ksrCand(ksrIntent(oneFile, twoParts, true), 100, 0)}
		if cands[0].want != limit {
			t.Fatalf("TestKShapeSplitOversizedReviewAcceptance: the many-parts fixture estimates %d bytes, want %d", cands[0].want, limit)
		}
		parts["engine-core"], parts["engine-rules"] = ksrPart(), ksrPart()
	case "stages":
		// A staged plan with one changed file per stage does not lower the estimate of the Intent.
		fill := limit - 4096 - len(ksrFeature) - len(ksrSteps) - ksrReadme - len(agent.Prompt(agent.TestingRules))
		cands = []ksrCandidate{ksrCand(ksrSized(bothFiles, stages, true, 4097), fill, 0), ksrCand(ksrIntent(coreOnly, splitTable, true), 100, 0)}
		parts["engine-rules"] = ksrPart()
	case "edge":
		// The estimate is one byte over the limit in the first candidate and exactly the limit in
		// the second.
		fill := limit - 4096 - len(ksrFeature) - len(ksrSteps) - ksrReadme - len(agent.Prompt(agent.TestingRules))
		cands = []ksrCandidate{ksrCand(ksrSized(oneFile, "", true, 4097), fill, 0), ksrCand(ksrSized(oneFile, "", true, 4096), fill, 0)}
		if cands[0].want != limit+1 || cands[1].want != limit {
			t.Fatalf("TestKShapeSplitOversizedReviewAcceptance: the edge fixture estimates %d and %d bytes, want %d and %d", cands[0].want, cands[1].want, limit+1, limit)
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
	o := map[string]any{"case": name, "ceiling": ceiling, "limit": limit, "rules": len(agent.Prompt(agent.TestingRules)), "exit": exit, "err": fmt.Sprint(err),
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
