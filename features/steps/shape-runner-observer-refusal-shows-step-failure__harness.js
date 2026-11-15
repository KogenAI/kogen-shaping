'use strict';
// The Go source of the acceptance harness of shape-runner-observer-refusal-shows-step-failure,
// which the steps file compiles into the shape package's tests through a Go overlay as
// shape/shape_runner_observer_refusal_acceptance_test.go; that path never exists in the
// repository. It holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of shape-runner-observer-refusal-shows-step-failure. Each case builds a
// verify.Ready with failed scenarios carrying a runner-observer result and a detail, calls
// runnerObserverGate and writes the refusal items and text as JSON to SROR_OUT. It judges nothing.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

const srorPassing = "the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips"
const srorLiteral = "the name filter -n ^test_widget_passes$ is a regular expression that the runner matched as a literal test name, so nothing ran: 0 runs, 0 assertions, 0 failures, 0 errors, 0 skips"

func srorScenario(name string, line int, observer string, detail *string) verify.ScenarioResult {
	return verify.ScenarioResult{Name: name, Line: line, Status: "failed", RunnerObserver: observer, Detail: detail}
}

func srorObserve(scenarios ...verify.ScenarioResult) map[string]any {
	err := runnerObserverGate(&verify.Ready{Scenarios: scenarios})
	obs := map[string]any{"error": err != nil, "items": []string{}, "text": ""}
	if e, ok := err.(*runnerObserverError); ok {
		obs["items"] = e.items
		obs["text"] = e.text()
	}
	return obs
}

func TestShapeRunnerObserverRefusalShowsStepFailureAcceptance(t *testing.T) {
	s := func(v string) *string { return &v }
	out := map[string]any{
		"passing":    srorObserve(srorScenario("Even numbers", 2, srorPassing, s("assertion failed in scenario Even numbers\nsecond line\nthird line"))),
		"blank-lead": srorObserve(srorScenario("Even numbers", 2, srorPassing, s("\n  \nfirst real line\nsecond line"))),
		"long":       srorObserve(srorScenario("Even numbers", 2, srorPassing, s(strings.Repeat("é", 250)+"\nnext"))),
		"exact":      srorObserve(srorScenario("Even numbers", 2, srorPassing, s(strings.Repeat("a", 200)+"\nnext"))),
		"literal":    srorObserve(srorScenario("Odd numbers", 5, srorLiteral, s("should not appear\nnext"))),
		"nodetail":   srorObserve(srorScenario("Even numbers", 2, srorPassing, nil)),
		"mixed": srorObserve(
			srorScenario("Even numbers", 2, srorPassing, s("first line\nsecond line")),
			srorScenario("Odd numbers", 5, srorLiteral, s("literal detail"))),
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SROR_OUT"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
