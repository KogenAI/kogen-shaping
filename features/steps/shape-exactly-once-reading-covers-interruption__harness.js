'use strict';
// The Go source of the acceptance harness of shape-exactly-once-reading-covers-interruption. The
// steps file compiles it into the shape package's tests through a Go overlay as
// shape/ske_acceptance_test.go; that path never exists in the repository. The source holds no
// backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"os"
	"testing"
)

// TestSkeAcceptance feeds scripted decision-check items through the real decision outcome rules
// and writes the instructions and what each scripted answer produced as JSON to SKE_OUT.
func TestSkeAcceptance(t *testing.T) {
	out := os.Getenv("SKE_OUT")
	if out == "" {
		t.Fatal("SKE_OUT must be set")
	}
	const request = "Process each job exactly once."
	const quote = "Process each job exactly once."
	const weak = "- Exactly once means at-most-once on a crash. (assumption: crash handling is silent) (source: request)"
	const full = "- Exactly once means each job is run once, and a job interrupted by a crash is resumed so it is neither lost nor repeated. (assumption: crash handling is silent) (source: request)"
	run := func(statement, verdict, intentValue, requestValue string) []string {
		intent := "## Decisions\n" + statement + "\n"
		it := decisionItem{statement: statement, quotes: []string{quote}, verdict: verdict, intentValue: intentValue, requestValue: requestValue, difference: "d", why: "w"}
		_, lines, _, _ := decisionResolveNotes([]decisionItem{it}, request, intent, nil)
		if lines == nil {
			lines = []string{}
		}
		return lines
	}
	obs := map[string]any{
		"instructions":   decisionInstructions,
		"weakened_lines": run(weak, "contradicts", "at-most-once on a crash", "exactly once"),
		"full_lines":     run(full, "follows", "", ""),
	}
	b, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
