'use strict';
// Go source of the acceptance harness, compiled into the shape tests by overlay as
// shape/shape_audit_invented_contract_and_deferrals_acceptance_test.go (never present in the repository).
// The source holds no backquote and no dollar-brace.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const ksaidControls = ".kogen/intents/is-even/controls.json"
const ksaidStepsPath = "features/steps/is-even__steps.js"

func ksaidFiles(steps string, controls string) map[string]any {
	files := shpFiles()
	list := files["files"].([]any)
	for i, file := range list {
		m := file.(map[string]any)
		if m["path"] == ksaidStepsPath {
			list[i] = map[string]any{"path": ksaidStepsPath, "content": steps}
		}
	}
	if controls != "" {
		list = append(list, map[string]any{"path": ksaidControls, "content": controls})
	}
	files["files"] = list
	return files
}

const ksaidDeferral = "{\"deferred\":\"too large to write within the shaping budget\"}\n"

func ksaidReview(id string, extra map[string]any) shpStep {
	args := map[string]any{"findings": []any{}, "environment": []any{}, "properties": []any{}}
	for k, v := range extra {
		args[k] = v
	}
	return shpAnswer(shpCall(id, SubmitContractReviewTool, args))
}

func ksaidInvented(literal string) map[string]any {
	return map[string]any{"invented": []any{map[string]any{"scenario": "Even numbers", "literal": literal, "why": "no source states it"}}}
}

func ksaidModes(directions ...string) map[string]any {
	var modes []any
	for _, d := range directions {
		modes = append(modes, map[string]any{"property": "deliver exactly once", "direction": d, "wrong_implementation": "wrong " + d, "scenario": "Even numbers", "assertion": "is-even: it is even"})
	}
	return map[string]any{"guarantees": []any{map[string]any{"property": "deliver exactly once", "request": "deliver exactly once", "needs_interrupted": true}}, "failure_modes": modes}
}

type ksaidRun struct {
	exit    int
	feed    string
	reviews int
}

// ksaidShape runs Shape on the fixture. The fixture's unchanged tree reports every scenario
// FAILED with exit 1 (red); red false makes it report PASSED with exit 0 instead, so that a
// scenario passes on the unchanged tree. A red run needs an acknowledging submission to publish.
func ksaidShape(t *testing.T, request string, red bool, submits []shpStep, reviews ...shpStep) ksaidRun {
	f := newShpFx(t, submits...)
	if !red {
		f.setReport("PASSED", 0)
	}
	f.write("web/routes.txt", "/settings/upgrade\n")
	f.git("add", ".")
	f.git("commit", "-m", "routes")
	f.Prov.contract = reviews
	exit, _ := f.shape(request)
	feed := ""
	for _, o := range f.allToolOutputs() {
		feed += o + "\n"
	}
	n := 0
	for _, k := range ccKinds(f) {
		if k == "review" {
			n++
		}
	}
	return ksaidRun{exit: exit, feed: feed, reviews: n}
}

func (r ksaidRun) refused() bool { return strings.Contains(r.feed, "Contract check:") }

func TestShapeAuditInventedContractAcceptance(t *testing.T) {
	obs := map[string]any{}
	const upgrade = "Add an upgrade button in settings."
	const once = "Deliver exactly once."
	route := "// steps\nconst { Given, Then } = require('@cucumber/cucumber');\nGiven('is-even: the number {int}', function () { post('/account/upgrade'); });\nThen('is-even: it is even', function () {});\nThen('is-even: it is not even', function () {});\n"
	sourced := strings.ReplaceAll(route, "/account/upgrade", "/settings/upgrade")
	v2 := shpSteps + "// v2\n"
	all := ksaidModes("too_much", "too_little", "interrupted")

	r := ksaidShape(t, upgrade, true, []shpStep{shpSubmit("c1", ksaidFiles(route, "")), shpSubmit("c2", ksaidFiles(sourced, ""))}, ksaidReview("r1", ksaidInvented("/account/upgrade")), ksaidReview("r2", map[string]any{}))
	obs["invented-route"] = map[string]any{"refused": r.refused(), "names_kind": strings.Contains(r.feed, "invented_contract"),
		"names_scenario": strings.Contains(r.feed, "Even numbers"), "names_literal": strings.Contains(r.feed, "/account/upgrade")}

	r = ksaidShape(t, upgrade, true, []shpStep{shpSubmit("c1", ksaidFiles(sourced, "")), shpAck("c2")}, ksaidReview("r1", ksaidInvented("/settings/upgrade")))
	obs["sourced-literal"] = map[string]any{"refused": r.refused(), "exit": r.exit}

	r = ksaidShape(t, once, true, []shpStep{shpSubmit("c1", ksaidFiles(shpSteps, ksaidDeferral)), shpSubmit("c2", ksaidFiles(v2, ksaidDeferral))}, ksaidReview("r1", ksaidModes("too_much")), ksaidReview("r2", all))
	obs["one-direction"] = map[string]any{"refused": r.refused(), "names_kind": strings.Contains(r.feed, "uncovered_failure_mode"),
		"names_property": strings.Contains(r.feed, "deliver exactly once"), "names_missing_direction": strings.Contains(r.feed, "interrupted")}

	r = ksaidShape(t, once, true, []shpStep{shpSubmit("c1", ksaidFiles(shpSteps, ksaidDeferral)), shpAck("c2")}, ksaidReview("r1", all))
	obs["both-directions"] = map[string]any{"refused": r.refused(), "exit": r.exit}

	r = ksaidShape(t, once, false, []shpStep{shpSubmit("c1", ksaidFiles(shpSteps, ksaidDeferral)), shpSubmit("c2", ksaidFiles(v2, ksaidDeferral))}, ksaidReview("r1", all), ksaidReview("r2", all))
	obs["happy-path"] = map[string]any{"refused": r.refused(), "names_kind": strings.Contains(r.feed, "uncovered_failure_mode"), "names_scenario": strings.Contains(r.feed, "Even numbers")}

	controlsOK := "{\"correct\":{\"files\":{}},\"wrong\":[]}\n"
	r = ksaidShape(t, once, false, []shpStep{shpSubmit("c1", ksaidFiles(shpSteps, controlsOK))}, ksaidReview("r1", map[string]any{}))
	obs["controls-ran"] = map[string]any{"refused": r.refused(), "exit": r.exit,
		"audit_refusal": strings.Contains(r.feed, "uncovered_failure_mode"), "audit_requests": r.reviews - 1}

	r = ksaidShape(t, upgrade, true, []shpStep{shpSubmit("c1", ksaidFiles(route, "")), shpSubmit("c2", ksaidFiles(route, "")), shpSubmit("c3", ksaidFiles(sourced, ""))}, ksaidReview("r1", ksaidInvented("/account/upgrade")), ksaidReview("r2", map[string]any{}))
	first := strings.Count(r.feed, "invented_contract")
	obs["resubmission"] = map[string]any{"same_refusal": first >= 2, "extra_requests_for_unchanged": r.reviews - 2, "changed_audited_again": r.reviews >= 2}

	out, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("KSAID_OUT"), out, 0o666); err != nil {
		t.Fatal(err)
	}
}
`;
