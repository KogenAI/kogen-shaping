package shape

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
)

func refusalGates(f *shpFx) [][]string {
	var rounds [][]string
	last := -1
	for _, e := range shpNamed(f.events(), "submission_refused") {
		at := int(e["attempt"].(float64))
		if at != last {
			rounds = append(rounds, nil)
			last = at
		}
		rounds[len(rounds)-1] = append(rounds[len(rounds)-1], e["gate"].(string))
	}
	return rounds
}

// A decision record refusal on a files submission is reported with the decision and literal
// checks' refusals, today's text for each, and no readiness or contract review runs.
func TestDecisionRecordRefusalIsReportedWithTheLaterGates(t *testing.T) {
	parallel(t)
	args := shpFiles()
	args["decisions"] = []any{map[string]any{"decision": "A"}}
	f := newShpFx(t, shpSubmit("c1", args), shpSubmit("c2", args), shpSubmit("c3", args))
	decision, literal := orderDecisionRefusal(), orderLiteralRefusal()
	f.Prov.route = func(req map[string]any) *shpStep {
		switch {
		case decisionRequest(req):
			return &decision
		case literalRequest(req):
			return &literal
		}
		return nil
	}
	if _, err := f.shape(orderRequest); err == nil {
		t.Fatal("a refused package was published")
	}
	rounds := refusalGates(f)
	if len(rounds) == 0 || strings.Join(rounds[0], ",") != "decision_record,decision_check,literal_check" {
		t.Fatalf("first round refused by %v", rounds)
	}
	fed := shpOutputs(f.Prov.requests()[1])["c1"]
	for _, want := range []string{"sourced decision 1: no evidence and no uncertainty", "Decision check: ", "Literal check: "} {
		if !strings.Contains(fed, want) {
			t.Fatalf("feedback lacks %q:\n%s", want, fed)
		}
	}
	names := strings.Join(eventNames(f.events()), ",")
	for _, name := range []string{"contract_review", "unpassable_check"} {
		if strings.Contains(names, name) {
			t.Fatalf("%s ran on a refused submission", name)
		}
	}
}

// Fixing some of the refused gates each round is progress: the shaping goes on to a fourth
// submission and is accepted, although the first refusing gate is the same every round.
func TestFixingSomeGatesEachRoundIsNotStoppedEarly(t *testing.T) {
	parallel(t)
	bad := shpFiles()
	bad["decisions"] = []any{map[string]any{"decision": "A"}}
	f := newShpFx(t, shpSubmit("c1", bad), shpSubmit("c2", bad), shpSubmit("c3", bad), shpSubmit("c4", shpFiles()), shpAck("ack"))
	var decisions, literals atomic.Int32
	decision, literal := orderDecisionRefusal(), orderLiteralRefusal()
	f.Prov.route = func(req map[string]any) *shpStep {
		switch {
		case decisionRequest(req):
			if decisions.Add(1) <= 2 {
				return &decision
			}
		case literalRequest(req):
			if literals.Add(1) <= 1 {
				return &literal
			}
		}
		return nil
	}
	exit, err := f.shape(orderRequest)
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	want := fmt.Sprint([][]string{{"decision_record", "decision_check", "literal_check"}, {"decision_record", "decision_check"}, {"decision_record"}})
	if got := fmt.Sprint(refusalGates(f)); got != want {
		t.Fatalf("refusals %s, want %s", got, want)
	}
}

// The same refusal repeated without fixing anything still ends the shaping.
func TestUnfixedCombinedRefusalStillEndsTheShaping(t *testing.T) {
	parallel(t)
	b := retryBudget{limit: 3}
	set := []gateProblems{{"a", []string{"1", "2"}}, {"b", []string{"3"}}}
	if b.refuseSet(set) || b.refuseSet(set) || !b.refuseSet(set) {
		t.Fatalf("budget %+v", b)
	}
}

func TestRetryBudgetProgressIsComparedByProblems(t *testing.T) {
	parallel(t)
	b := retryBudget{limit: 3}
	rounds := [][]gateProblems{
		{{"a", []string{"1", "2"}}, {"b", []string{"3"}}, {"c", []string{"4"}}},
		{{"a", []string{"1", "2"}}, {"b", []string{"3"}}},
		{{"a", []string{"1"}}, {"b", []string{"3"}}},
		{{"a", []string{"1"}}},
	}
	for i, r := range rounds {
		if b.refuseSet(r) {
			t.Fatalf("round %d stopped the shaping: %+v", i+1, b)
		}
		if b.consecutive != 1 && i > 0 {
			t.Fatalf("round %d consecutive %d", i+1, b.consecutive)
		}
	}
	if b.total != 4 {
		t.Fatalf("total %d", b.total)
	}
	// A single gate keeps today's rule: the same gate repeats whatever its problems say.
	s := retryBudget{limit: 3}
	s.refuse("a")
	s.refuse("a")
	if !s.refuse("a") {
		t.Fatalf("budget %+v", s)
	}
}
