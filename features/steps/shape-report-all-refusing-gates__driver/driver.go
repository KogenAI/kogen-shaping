//go:build ignore

package shape

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestShapeReportAllRefusingGatesObservation shapes a draft that the scripted provider and the
// replayed acceptance report refuse, and writes what the shaper saw to SRAG_OUT.
func TestShapeReportAllRefusingGatesObservation(t *testing.T) {
	out := os.Getenv("SRAG_OUT")
	if out == "" {
		t.Skip("observation driver")
	}
	args := shpFiles()
	request := "Add calc.is_even(n)."
	both := os.Getenv("SRAG_MODE") == "both"
	steps := []shpStep{shpSubmit("c1", args), shpAck("c2"), shpAck("c3")}
	if both {
		args["decisions"] = []any{map[string]any{"decision": "A"}}
		request = orderRequest
		steps = []shpStep{shpSubmit("c1", args), shpSubmit("c2", args), shpSubmit("c3", args)}
	}
	f := newShpFx(t, steps...)
	if both {
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
	}
	f.setReport("UNDEFINED", 1)
	f.shape(request)
	result := map[string]any{"names": strings.Join(eventNames(f.events()), ","), "total": -1}
	if rounds := refusalGates(f); len(rounds) > 0 {
		result["gates"] = strings.Join(rounds[0], ",")
	}
	refused := shpNamed(f.events(), "submission_refused")
	if len(refused) > 0 {
		result["total"] = refused[0]["total"]
	}
	for _, e := range refused {
		if e["gate"] == "readiness" {
			result["readiness"] = e["problems"]
			break
		}
	}
	if reqs := f.Prov.requests(); len(reqs) > 1 {
		result["fed"] = shpOutputs(reqs[1])["c1"]
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
