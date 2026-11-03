package shape

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

func prepStep(keyword, text string, line int) verify.StepRecord {
	return verify.StepRecord{Keyword: keyword, Text: text, Line: line}
}

func prepDetail(s string) *string { return &s }

// prepScenario is a failed scenario at line 2 with the steps given and the detail of its failure.
func prepScenario(steps []verify.StepRecord, unexercised []verify.StepRecord, detail string) verify.ScenarioResult {
	return verify.ScenarioResult{Name: "Count", Line: 2, Steps: steps, UnexercisedSteps: unexercised, Status: "failed", Detail: prepDetail(detail)}
}

func prepJudge(sc verify.ScenarioResult, known ...preparationSignature) []preparationFailure {
	return preparationFailures(&verify.Ready{Scenarios: []verify.ScenarioResult{sc}, AcceptanceExit: 1}, known, verify.NewRedactor(nil, ""))
}

func TestPreparationFailures(t *testing.T) {
	given := prepStep("Given ", "a packet", 3)
	when := prepStep("When ", "I count", 4)
	then := prepStep("Then ", "the count is 2", 5)
	and := prepStep("And ", "more", 6)
	frames := "\n    at Object.<anonymous> (steps.js:15:10)\n    at run (x.js:1:1)"
	t.Run("a Given step is refused with its item, output and not-run steps", func(t *testing.T) {
		sc := prepScenario([]verify.StepRecord{given, when, then}, []verify.StepRecord{when, then},
			"acceptance.feature line 3: Given a packet\nAssertionError [ERR_ASSERTION]: fixturecheck exit=2\nmissing kind=packet"+frames)
		got := prepJudge(sc)
		if len(got) != 1 {
			t.Fatalf("failures = %v", got)
		}
		want := []string{
			"- Count (acceptance.feature line 2): acceptance.feature line 3: Given a packet failed (a Given step); acceptance exit 1",
			"  AssertionError [ERR_ASSERTION]: fixturecheck exit=2",
			"  missing kind=packet",
			"  " + "    at Object.<anonymous> (steps.js:15:10)",
			"  " + "    at run (x.js:1:1)",
			"  not run: acceptance.feature line 4: When I count",
			"  not run: acceptance.feature line 5: Then the count is 2",
		}
		shpEq(t, "item", strings.Join(got[0].items, "\n"), strings.Join(want, "\n"))
		shpEq(t, "signature", got[0].signature, "AssertionError [ERR_ASSERTION]: fixturecheck exit=2\nmissing kind=packet")
	})
	t.Run("a conjunction continues the role of the step before it", func(t *testing.T) {
		afterGiven := prepScenario([]verify.StepRecord{given, and, when, then}, nil, "acceptance.feature line 6: And more\nboom")
		if got := prepJudge(afterGiven); len(got) != 1 || !strings.Contains(got[0].items[0], "(a Given step)") {
			t.Fatalf("And after a Given: %v", got)
		}
		afterWhen := prepScenario([]verify.StepRecord{given, when, and, then}, nil, "acceptance.feature line 5: And more\nboom")
		afterWhen.Steps[2].Line = 5
		if got := prepJudge(afterWhen); len(got) != 0 {
			t.Fatalf("And after a When: %v", got)
		}
	})
	t.Run("a scenario that did not fail is never judged", func(t *testing.T) {
		sc := prepScenario([]verify.StepRecord{given, then}, nil, "acceptance.feature line 3: Given a packet\nboom")
		sc.Status = "passed"
		if got := prepJudge(sc); len(got) != 0 {
			t.Fatalf("passed: %v", got)
		}
		sc.Status, sc.Detail = "failed", nil
		if got := prepJudge(sc); len(got) != 0 {
			t.Fatalf("no detail: %v", got)
		}
	})
	t.Run("a Background step is found by its line, also for an Examples row", func(t *testing.T) {
		bg := prepStep("Given ", "the base", 2)
		sc := prepScenario([]verify.StepRecord{bg, prepStep("When ", "I count", 6)}, nil, "acceptance.feature line 2: Given the base\nboom")
		sc.Line = 4
		if got := prepJudge(sc); len(got) != 1 || !strings.Contains(got[0].items[0], "(a Background step)") {
			t.Fatalf("background: %v", got)
		}
		// A Background step whose keyword is When still is one.
		sc.Steps[0].Keyword = "When "
		if got := prepJudge(sc); len(got) != 1 || !strings.Contains(got[0].items[0], "(a Background step)") {
			t.Fatalf("background When: %v", got)
		}
		row := 10
		outline := prepScenario([]verify.StepRecord{bg, prepStep("When ", "I count", 6)}, nil, "acceptance.feature line 2: Given the base\nboom")
		outline.Line, outline.OutlineLine = 12, &row
		if got := prepJudge(outline); len(got) != 1 || !strings.Contains(got[0].items[0], "(a Background step)") {
			t.Fatalf("outline background: %v", got)
		}
		// An outline step below the outline's own line is not a Background step.
		outline.Steps[1].Line = 11
		outline.Detail = prepDetail("acceptance.feature line 11: When I count\nboom")
		if got := prepJudge(outline); len(got) != 0 {
			t.Fatalf("outline step: %v", got)
		}
	})
	t.Run("a hook fails the preparation only before the first step ran", func(t *testing.T) {
		before := prepScenario([]verify.StepRecord{given, then}, []verify.StepRecord{given, then}, "a hook at features/steps/x.js line 3\nboom")
		if got := prepJudge(before); len(got) != 1 || !strings.Contains(got[0].items[0], "a hook at features/steps/x.js line 3 failed (a Before hook)") {
			t.Fatalf("before hook: %v", got)
		}
		after := prepScenario([]verify.StepRecord{given, then}, nil, "a hook at features/steps/x.js line 5\nboom")
		if got := prepJudge(after); len(got) != 0 {
			t.Fatalf("after hook: %v", got)
		}
		partial := prepScenario([]verify.StepRecord{given, then}, []verify.StepRecord{then}, "a hook (location not reported)\nboom")
		if got := prepJudge(partial); len(got) != 0 {
			t.Fatalf("hook after the first step: %v", got)
		}
	})
	t.Run("a When or Then failure is refused only as an earlier refused signature, whatever its frames", func(t *testing.T) {
		sc := prepScenario([]verify.StepRecord{given, when, then}, []verify.StepRecord{then},
			"acceptance.feature line 4: When I count\nAssertionError: fixturecheck exit=2\nmissing kind=packet\n    at Object.<anonymous> (steps.js:17:10)  ")
		if got := prepJudge(sc); len(got) != 0 {
			t.Fatalf("no earlier signature: %v", got)
		}
		earlier := preparationSignature{signature: "AssertionError: fixturecheck exit=2\nmissing kind=packet", attempt: 1}
		later := preparationSignature{signature: earlier.signature, attempt: 4}
		got := prepJudge(sc, later, earlier)
		if len(got) != 1 || !strings.Contains(got[0].items[0], "failed (the same failure as the preparation refused at attempt 1); acceptance exit 1") {
			t.Fatalf("same failure: %v", got)
		}
		if got := prepJudge(sc, preparationSignature{signature: "other", attempt: 1}); len(got) != 0 {
			t.Fatalf("different message: %v", got)
		}
		empty := prepScenario([]verify.StepRecord{given, when, then}, nil, "acceptance.feature line 4: When I count\n    at x (y.js:1:1)")
		if got := prepJudge(empty, preparationSignature{signature: "", attempt: 1}); len(got) != 0 {
			t.Fatalf("empty signature: %v", got)
		}
	})
	t.Run("the output is redacted", func(t *testing.T) {
		sc := prepScenario([]verify.StepRecord{given, then}, nil, "acceptance.feature line 3: Given a packet\nrejected with api_key=abc123")
		got := prepJudge(sc)
		if len(got) != 1 {
			t.Fatalf("failures = %v", got)
		}
		text := strings.Join(got[0].items, "\n")
		shpContains(t, "item", text, "  rejected with api_key=[REDACTED]")
		if strings.Contains(text, "abc123") || strings.Contains(got[0].signature, "abc123") {
			t.Fatalf("secret leaked: %q %q", text, got[0].signature)
		}
	})
}

func TestShapePreparationGate(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpSubmit("c2", shpFiles()), shpSubmit("c3", shpFiles()))
	f.setReport("PREPARATION_FAILED", 1)
	exit, err := f.shape("Add calc.is_even(n).")
	e := shpWantError(t, err, "intent", "not_ready", "the Intent is not ready to build", 1)
	shpEq(t, "exit", exit, 1)
	reqs := f.Prov.requests()
	first := shpOutputs(reqs[1])["c1"]
	shpContains(t, "feedback", first, "Acceptance preparation failed on the unchanged tree: 2 scenario(s) fail before they reach the behaviour under test, so this is not missing-feature red; nothing was published.")
	shpContains(t, "feedback", first, "- Even numbers (acceptance.feature line 2): acceptance.feature line 3: Given is-even: the number 4 failed (a Given step); acceptance exit 1\n  fixture rejected in scenario Even numbers\n  not run: acceptance.feature line 4: Then is-even: it is even")
	shpContains(t, "feedback", first, "- Odd numbers (acceptance.feature line 5): acceptance.feature line 6: Given is-even: the number 3 failed (a Given step)")
	if strings.Contains(first, "Unchanged:") {
		t.Fatalf("the first refusal is not unchanged: %s", first)
	}
	shpContains(t, "resubmission", shpOutputs(reqs[2])["c2"], unchangedLine(1))
	if len(f.Prov.contractRequests()) != 0 {
		t.Fatalf("the contract review ran for a preparation failure")
	}
	if len(shpNamed(f.events(), "unpassable_check")) != 0 {
		t.Fatal("the unpassable control ran for a preparation failure")
	}
	if f.exists(".kogen/local/readiness/is-even.json") {
		t.Fatal("a refused preparation is never recorded as readiness")
	}
	refused := shpNamed(f.events(), "submission_refused")
	if len(refused) != 3 {
		t.Fatalf("refusals = %v", refused)
	}
	for i, r := range refused {
		shpEq(t, "gate", fmt.Sprint(r["gate"]), "preparation")
		shpEq(t, "attempt", fmt.Sprint(r["attempt"]), fmt.Sprint(i+1))
	}
	shpContains(t, "stop lines", strings.Join(e.Lines, "\n"), "- Even numbers (acceptance.feature line 2): acceptance.feature line 3: Given is-even: the number 4 failed (a Given step)")
}
