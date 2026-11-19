'use strict';
// The Go source of the acceptance harness of shape-missing-controls-reported-with-findings, which
// shape-missing-controls-reported-with-findings__steps.js compiles into the shape package's tests
// through a Go overlay as shape/shape_missing_controls_reported_with_findings_acceptance_test.go;
// that path never exists in the repository. The source holds no backquote and no dollar-brace.
module.exports = String.raw`package shape

// Each case runs the real Shape flow with a scripted provider, a scripted contract review and
// confirmation, and records what the first refusal showed, as JSON in SMCRWF_OUT. It judges nothing.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func smcrwfProp() any {
	return map[string]any{"property": "is_even is new", "request": "calc.is_even(n)", "kind": "new", "observable": true,
		"scenarios": []any{"Even numbers"}, "decision": "", "issues": []any{}}
}

// smcrwfCase scripts one first submission with the given controls.json (none when empty), a review
// with or without the confirmed finding and with or without the counted property.
func smcrwfCase(t *testing.T, controls string, finding, counted bool) map[string]any {
	files := shpFiles()
	if controls != "" {
		files["files"] = append(files["files"].([]any), map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": controls})
	}
	f := newShpFx(t, shpSubmit("c1", files))
	findings, props := []any{}, []any{}
	f.Prov.contract = nil
	if finding {
		findings = []any{ccContradiction()}
	}
	if counted {
		props = []any{smcrwfProp()}
	}
	f.Prov.contract = append(f.Prov.contract, shpAnswer(shpCall("r1", SubmitContractReviewTool, map[string]any{"findings": findings, "environment": []any{}, "properties": props})))
	if finding {
		f.Prov.contract = append(f.Prov.contract, ccConfirm("v1", ccVerdict(1, "confirmed", "four is even")))
	}
	// The script ends after the first submission; the end of Shape is not judged.
	f.shape("Add calc.is_even(n).")
	reqs := f.Prov.requests()
	if len(reqs) < 2 {
		t.Fatalf("the shaper got no feedback to the first submission:\n%s", f.Stderr.String())
	}
	refused := ccRefused(f)
	problems := []any{}
	if len(refused) > 0 {
		problems = refused[0]["problems"].([]any)
	}
	return map[string]any{"feedback": shpOutputs(reqs[1])["c1"], "problems": problems,
		"runs": strings.Count(f.Stderr.String(), "kogen: wrong-implementation check: run ")}
}

func TestShapeMissingControlsReportedWithFindingsAcceptance(t *testing.T) {
	out := os.Getenv("SMCRWF_OUT")
	if out == "" {
		t.Fatal("SMCRWF_OUT must be set")
	}
	obs := map[string]any{}
	obs["missing"] = smcrwfCase(t, "", true, true)
	obs["schema"] = smcrwfCase(t, "{\"correct\":{\"files\":{\"calc.js\":\"x\\n\"}},\"wrong\":[{\"name\":\"w\",\"scenario\":\"No such scenario\",\"step\":\"Then is-even: it is even\",\"files\":{\"calc.js\":\"y\\n\"}}]}", true, true)
	obs["uncounted"] = smcrwfCase(t, "", true, false)
	obs["clean"] = smcrwfCase(t, "", false, true)
	obs["valid"] = smcrwfCase(t, "{\"correct\":{\"files\":{\"calc.js\":\"x\\n\"}},\"wrong\":[]}", true, true)
	obs["deferred"] = smcrwfCase(t, "{\"deferred\":\"too large to write in the shaping budget\"}", true, true)
	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
