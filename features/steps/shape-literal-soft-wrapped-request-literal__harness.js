'use strict';
// The Go source of the shape-literal-soft-wrapped-request-literal acceptance harness, which
// shape-literal-soft-wrapped-request-literal__steps.js compiles into the shape package's tests
// through a Go overlay as shape/shape_literal_soft_wrapped_request_literal_acceptance_test.go; that
// path never exists in the repository. The source holds no backquote and no dollar-brace, so this
// raw template keeps it byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of shape-literal-soft-wrapped-request-literal. Each case calls
// literalOutcome with one required literal, the request that soft-wraps it and a package, and
// writes what it returned as JSON to SLSW_OUT for the acceptance steps to judge. The harness
// judges nothing itself.

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

func TestShapeLiteralSoftWrappedRequestLiteralAcceptance(t *testing.T) {
	const request = "Warn the user when the plan is nearly used up.\n\nShow the warning 50 cards\nleft in the header.\n\nIndented: 50 cards\n    left.\n\nPlain: 50 cards left.\n\nThen show\n\nleft on its own."
	type tc struct{ literal, pkg, packageValue string }
	cases := map[string]tc{
		"soft wrap":             {"50 cards\nleft", "The header shows 50 cards left.", ""},
		"soft wrap with indent": {"50 cards\n    left", "The header shows 50 cards left.", ""},
		"exact bytes":           {"50 cards\nleft", "The header shows 50 cards\nleft.", ""},
		"changed text":          {"50 cards\nleft", "The header shows 50 cards, left.", "50 cards, left"},
		"missing text":          {"50 cards\nleft", "The header shows 50 cards, left.", ""},
		"blank line":            {"Then show\n\nleft", "Then show left on its own.", ""},
		"no line break":         {"50 cards left", "The header shows 50  cards left.", ""},
	}
	out := map[string]any{}
	for name, c := range cases {
		it := litItem(c.literal, "request", c.literal, "required", c.packageValue)
		event, lines := literalOutcome([]literalItem{it}, map[string]string{"request": request}, []packageFile{{"intent.md", c.pkg}}, verify.NewRedactor(nil, ""))
		item := event["items"].([]any)[0].(map[string]any)
		if lines == nil {
			lines = []string{}
		}
		out[name] = map[string]any{"status": item["status"], "literal": item["literal"], "present": event["present"], "missing": event["missing"], "changed": event["changed"], "lines": lines}
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SLSW_OUT"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
