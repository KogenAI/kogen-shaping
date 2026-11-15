'use strict';
// The Go source of the shape-title-only-lint-fixed-in-place acceptance harness, which
// shape-title-only-lint-fixed-in-place__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_title_only_acceptance_test.go; that path never exists in the repository. The
// source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// One native Shape per case with a scripted shaper and no model. STO_CASE names the case and
// STO_OUT receives what the run showed as JSON.

type stoCase struct {
	title    string
	decision string // replaces the Non-integers Decisions line when set
	submits  int
}

var stoCases = map[string]stoCase{
	"long-title":       {title: "Add an is-even check for integers and also reject all", submits: 1},
	"long-title-other": {title: "Add an is-even check for integers and also reject all", decision: "- Non-integers raise TypeError.", submits: 3},
	"fifty-char-title": {title: "Add an is-even check for integers and also rejects", submits: 1},
}

func TestShapeTitleOnlyLintAcceptance(t *testing.T) {
	name, out := os.Getenv("STO_CASE"), os.Getenv("STO_OUT")
	c, ok := stoCases[name]
	if !ok || out == "" {
		t.Fatalf("shape-title-only-lint-fixed-in-place: STO_CASE %q is not a case of the harness, or STO_OUT is empty", name)
	}
	intent := strings.Replace(shpScopedIntentMD, "# Add an is-even check\n", "# "+c.title+"\n", 1)
	if c.decision != "" {
		intent = strings.Replace(intent, "- Non-integers raise TypeError. (source: request)", c.decision, 1)
	}
	files := map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
	}}
	var steps []shpStep
	for i := 1; i <= c.submits; i++ {
		steps = append(steps, shpSubmit(fmt.Sprintf("c%d", i), files))
	}
	f := newShpFx(t, steps...)
	f.setReport("PASSED", 0)
	exit, err := f.shape("Add calc.is_even(n).")
	seen := map[string]any{"case": name, "exit": exit, "err": fmt.Sprint(err), "stderr": f.Stderr.String()}
	seen["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	seen["intent"] = ""
	if f.exists(".kogen/intents/is-even/intent.md") {
		seen["intent"] = f.read(".kogen/intents/is-even/intent.md")
	}
	feedback := map[string]any{}
	for _, r := range f.Prov.allRequests() {
		for id, o := range shpOutputs(r) {
			if _, seenID := feedback[id]; !seenID {
				feedback[id] = o
			}
		}
	}
	seen["feedback"] = feedback
	refused := []any{}
	for _, e := range shpNamed(f.events(), "submission_refused") {
		refused = append(refused, fmt.Sprintf("%v:%v", e["attempt"], e["gate"]))
	}
	seen["refused"] = refused
	shortened := []any{}
	for _, e := range shpNamed(f.events(), "title_shortened") {
		shortened = append(shortened, map[string]any{"original": e["original"], "title": e["title"]})
	}
	seen["shortened"] = shortened
	b, err := json.MarshalIndent(seen, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("shape-title-only-lint-fixed-in-place: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
