'use strict';
// The Go source of the k-shape-size-for-one-review acceptance harness, which
// k-shape-size-for-one-review__steps.js compiles into the shape package's tests through a Go overlay
// as shape/k_size_for_one_review_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// The acceptance harness of k-shape-size-for-one-review: synthetic shaping journeys in a small
// synthetic project, with a scripted shaper, scripted model checks and a real Cucumber acceptance
// runner. KSS_CASE names one case of the request; what a user and the shaper see of the run is
// written as JSON to KSS_OUT for the acceptance steps to judge.

const kssBacktick = "\x60"

var kssAreas = []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet", "kilo", "lima"}

const kssSteps = "const {Then} = require('@cucumber/cucumber');\nconst assert = require('node:assert/strict');\n" +
	"Then('is-even: {word} behaviour {int} is built', function (area, k) {\n" +
	"  assert.strictEqual(require('../../' + area + '/module.js').behaviour(k), 'built ' + k);\n});\n"

// kssCandidate is one submitted package: scenarios 1..n cycle over the first cycle areas, and
// only the scenarios of the first only areas are kept; the Intent names the files of those areas
// and, when stages is set, plans one stage per entry naming the files of its areas.
type kssCandidate struct {
	id             string
	n, cycle, only int
	stages         [][]int
}

type kssCase struct {
	request    string
	size       int // the bytes of every area's module.js
	candidates []kssCandidate
	property   string // when set, the contract review reports this requested example
}

var kssStaged = [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}, {8, 9, 10, 11}}

const kssWideRequest = "Build behaviours 1 to 40, spread over the twelve areas alpha to lima: behaviour k returns built k."

var kssCases = map[string]kssCase{
	"small": {request: "Make alpha behaviours 1 and 3 and bravo behaviour 2 return built k.", size: 2000,
		candidates: []kssCandidate{{id: "c1", n: 3, cycle: 2, only: 2}}},
	"wide": {request: kssWideRequest, size: 48000,
		candidates: []kssCandidate{{id: "c1", n: 40, cycle: 12, only: 12}, {id: "c2", n: 40, cycle: 12, only: 12, stages: kssStaged}}},
	"wide-unstaged": {request: kssWideRequest, size: 48000,
		candidates: []kssCandidate{{id: "c1", n: 40, cycle: 12, only: 12}}},
	"heavy": {request: kssWideRequest, size: 90000,
		candidates: []kssCandidate{{id: "c1", n: 40, cycle: 12, only: 12}, {id: "c2", n: 40, cycle: 12, only: 12, stages: kssStaged}}},
	"narrow": {request: "Build behaviours 1 to 6 of the areas alpha and bravo: behaviour k returns built k. No other area changes.", size: 48000,
		candidates: []kssCandidate{{id: "c1", n: 40, cycle: 12, only: 12}, {id: "c2", n: 6, cycle: 2, only: 2}}},
	"every": {request: "Build every behaviour 1 to 40 over the twelve areas alpha to lima; each one is needed. For example, lima behaviour 36 returns built 36.", size: 48000,
		property:   "lima behaviour 36 returns built 36",
		candidates: []kssCandidate{{id: "c1", n: 40, cycle: 12, only: 12}, {id: "c2", n: 40, cycle: 12, only: 4}, {id: "c3", n: 40, cycle: 12, only: 12, stages: kssStaged}}},
}

func kssPath(area int) string { return kssAreas[area] + "/module.js" }

// kssModule is a do-nothing module of exactly size bytes: no behaviour is built yet.
func kssModule(size int) string {
	head := "exports.behaviour = k => 'not built ' + k;\n"
	return head + "//" + strings.Repeat("x", size-len(head)-3) + "\n"
}

func kssNamed(areas []int) string {
	var names []string
	for _, a := range areas {
		names = append(names, kssBacktick+kssPath(a)+kssBacktick)
	}
	return strings.Join(names, ", ")
}

func kssFeature(c kssCandidate) string {
	var b strings.Builder
	b.WriteString("Feature: Synthetic behaviours\n")
	for k := 1; k <= c.n; k++ {
		if a := (k - 1) % c.cycle; a < c.only {
			fmt.Fprintf(&b, "  Scenario: %s behaviour %d\n    Then is-even: %s behaviour %d is built\n", kssAreas[a], k, kssAreas[a], k)
		}
	}
	return b.String()
}

func kssIntent(c kssCandidate) string {
	var areas []int
	for a := 0; a < c.only; a++ {
		areas = append(areas, a)
	}
	text := "# Build the synthetic behaviours\n\nSlug: is-even\nModel: not applicable — synthetic behaviours without state\nAcceptance:\n" +
		"- .kogen/intents/is-even/acceptance.feature\n- features/steps/is-even__steps.js\n\n" +
		"Behaviour k of an area returns built k. The change is in " + kssNamed(areas) + ".\n\n" +
		"## Decisions\n- Behaviour k returns built k. (source: request)\n\n## Source comparison\n- The request is covered by the feature.\n\n" +
		"## Tests\n- Each behaviour returns built k: the acceptance scenarios (scenario)\n\n"
	if c.stages != nil {
		ids := []string{"first", "second", "third"}
		var stages []any
		for i, st := range c.stages {
			stages = append(stages, map[string]any{"id": ids[i], "title": "Stage " + ids[i], "goal": "Build the behaviours of " + kssNamed(st), "checks": []any{"sh accept.sh"}})
		}
		plan, _ := json.Marshal(map[string]any{"version": 1, "stages": stages})
		fence := strings.Repeat(kssBacktick, 3)
		text += "## Stages\n" + fence + "json\n" + string(plan) + "\n" + fence + "\n\n"
	}
	return text + "## Feasibility\n- none\n"
}

func kssFiles(c kssCandidate) map[string]any {
	return map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": kssIntent(c)},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": kssFeature(c)},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": kssSteps},
	}}
}

func TestKShapeSizeForOneReviewAcceptance(t *testing.T) {
	name, out := os.Getenv("KSS_CASE"), os.Getenv("KSS_OUT")
	c, ok := kssCases[name]
	if !ok || out == "" {
		t.Fatalf("TestKShapeSizeForOneReviewAcceptance: KSS_CASE %q is not a case of the harness, or KSS_OUT is empty", name)
	}
	var steps []shpStep
	for _, cand := range c.candidates {
		steps = append(steps, shpSubmit(cand.id, kssFiles(cand)))
	}
	last := c.candidates[len(c.candidates)-1]
	steps = append(steps, shpSubmit("ack", map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": kssFeature(last)},
	}}))
	f := newShpFx(t, steps...)
	modules := os.Getenv("KOGEN_CUCUMBER")
	node := os.Getenv("KOGEN_NODE")
	if node == "" {
		node = "node"
	}
	runner := filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")
	f.write("features/steps/calc_steps.js", "// existing steps\n")
	for a := range kssAreas {
		f.write(kssPath(a), kssModule(c.size))
	}
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\necho run >> '%s/trace'\ncase \"$KOGEN_ACCEPTANCE_FEATURES\" in */check.feature) : > \"$KOGEN_ACCEPTANCE_REPORT\"; exit 0;; esac\n"+
		"NODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n", f.Aux, modules, node, runner))
	f.git("add", "-A")
	f.git("commit", "-qm", "size fixture")
	if c.property != "" {
		review := map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{map[string]any{
			"property": c.property, "request": c.property, "kind": "new", "observable": true,
			"scenarios": []any{"lima behaviour 36"}, "decision": "", "issues": []any{}}}}
		verdicts := []any{}
		for n := 1; n <= 4; n++ {
			verdicts = append(verdicts, map[string]any{"finding": n, "verdict": "confirmed", "why": "synthetic confirmation"})
		}
		f.Prov.route = func(req map[string]any) *shpStep {
			var st shpStep
			switch contractKind(req) {
			case SubmitContractReviewTool:
				st = shpAnswer(shpCall("review", SubmitContractReviewTool, review))
			case SubmitContractConfirmationTool:
				st = shpAnswer(shpCall("confirm", SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts}))
			default:
				return nil
			}
			return &st
		}
	}
	exit, err := f.shape(c.request)
	o := map[string]any{"case": name, "request": c.request, "bound": tools.InputBudgetBytes, "exit": exit, "err": fmt.Sprint(err),
		"stdout": f.Stdout.String(), "progress": f.Stderr.String(), "remaining": "", "intent": "", "feature": ""}
	o["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	if e, ok := sys.AsError(err); ok {
		o["remaining"] = strings.Join(e.Lines, "\n")
	}
	if f.exists(".kogen/intents/is-even/intent.md") {
		o["intent"] = f.read(".kogen/intents/is-even/intent.md")
	}
	if f.exists(".kogen/intents/is-even/acceptance.feature") {
		o["feature"] = f.read(".kogen/intents/is-even/acceptance.feature")
	}
	results := map[string]any{}
	for _, r := range f.Prov.requests() {
		for id, text := range shpOutputs(r) {
			if _, seen := results[id]; !seen && !strings.HasSuffix(id, "-inspect") {
				results[id] = text
			}
		}
	}
	o["results"] = results
	o["shaper"], o["decision"], o["contract"] = len(f.Prov.requests()), len(f.Prov.decisionRequests()), len(f.Prov.contractRequests())
	b, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("TestKShapeSizeForOneReviewAcceptance: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
