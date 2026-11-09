'use strict';
// The Go source of the shape-contract-refusal-fix-line acceptance harness, which
// shape-contract-refusal-fix-line__steps.js compiles into the shape package's tests through a Go
// overlay as shape/shape_contract_refusal_fix_line_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it byte
// for byte.
module.exports = String.raw`package shape

// The acceptance harness of shape-contract-refusal-fix-line: four scripted Shape runs through the
// real contract-review gate in the shape tests' is-even project. The scripted review and
// confirmation answers are fixed, so each run is deterministic. What each run showed is written as
// JSON to SCRFL_OUT for the acceptance steps to judge. The harness judges nothing itself.

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
)

const (
	scrflFeaturePath = ".kogen/intents/is-even/acceptance.feature"
	scrflIntentPath  = ".kogen/intents/is-even/intent.md"
	scrflRequest     = "Add calc.is_even(n)."
)

func scrflFinding(scenario, expectation, contract, why string) map[string]any {
	return map[string]any{
		"scenario":    scenario,
		"expectation": map[string]any{"path": scrflFeaturePath, "quote": expectation},
		"contract":    map[string]any{"path": scrflIntentPath, "quote": contract},
		"why":         why,
	}
}

func scrflReview(findings ...any) shpStep {
	return shpAnswer(shpCall("r1", SubmitContractReviewTool, map[string]any{"findings": findings, "environment": []any{}}))
}

func scrflConfirm(verdicts ...any) shpStep {
	return shpAnswer(shpCall("v1", SubmitContractConfirmationTool, map[string]any{"verdicts": verdicts}))
}

func scrflVerdict(n int, verdict, why string) map[string]any {
	return map[string]any{"finding": n, "verdict": verdict, "why": why}
}

func scrflIntentOnly(id, intent string) shpStep {
	return shpSubmit(id, map[string]any{"files": []any{map[string]any{"path": scrflIntentPath, "content": intent}}})
}

func scrflPackage(intent string) map[string]any {
	return map[string]any{"files": []any{
		map[string]any{"path": scrflIntentPath, "content": intent},
		map[string]any{"path": scrflFeaturePath, "content": shpFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
	}}
}

// scrflOutput is the text the shaper was given for call id in its request n, empty when absent.
func scrflOutput(f *shpFx, n int, id string) string {
	reqs := f.Prov.requests()
	if n >= len(reqs) {
		return ""
	}
	return shpOutputs(reqs[n])[id]
}

func scrflStrings(v any) []string {
	out := []string{}
	list, _ := v.([]any)
	for _, x := range list {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// scrflEvents are the contract_review events' counts and the problems of the first refusal by the
// contract_review gate.
func scrflEvents(f *shpFx) (reviews []any, refused int, problems []string) {
	problems = []string{}
	events := f.events()
	for _, e := range shpNamed(events, "contract_review") {
		reviews = append(reviews, map[string]any{"result": e["result"], "findings": e["findings"], "evidenced": e["evidenced"], "confirmed": e["confirmed"]})
	}
	for _, e := range shpNamed(events, "submission_refused") {
		if e["gate"] != "contract_review" {
			continue
		}
		if refused == 0 {
			problems = scrflStrings(e["problems"])
		}
		refused++
	}
	if reviews == nil {
		reviews = []any{}
	}
	return reviews, refused, problems
}

func scrflRecord(f *shpFx) (bool, []string) {
	b, err := os.ReadFile(f.Root + "/.kogen/local/shaping/is-even/contract-contradictions.json")
	if err != nil {
		return false, []string{}
	}
	var rec map[string]any
	if json.Unmarshal(b, &rec) != nil {
		return true, []string{}
	}
	return true, scrflStrings(rec["lines"])
}

func scrflError(err error) (string, []string) {
	if e, ok := sys.AsError(err); ok {
		return e.Reason, append([]string{}, e.Lines...)
	}
	if err != nil {
		return err.Error(), []string{}
	}
	return "", []string{}
}

// scrflRun is the observation every run shares.
func scrflRun(f *shpFx, exit int, err error) map[string]any {
	reviews, refused, problems := scrflEvents(f)
	exists, record := scrflRecord(f)
	reason, lines := scrflError(err)
	return map[string]any{
		"exit": exit, "error_reason": reason, "error_lines": lines,
		"contract_reviews": reviews, "contract_refusal_events": refused, "event_problems": problems,
		"record_exists": exists, "record_lines": record,
		"published": strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even"),
	}
}

func scrflTwoConfirmed(t *testing.T) map[string]any {
	f := newShpFx(t, shpSubmit("c1", shpFiles()), scrflIntentOnly("c2", shpScopedIntentMD), scrflIntentOnly("c3", shpScopedIntentMD))
	f.Prov.contract = []shpStep{
		scrflReview(
			scrflFinding("Even numbers", "Given is-even: the number 4", "calc.is_even(n) returns True for even integers.", "four is even"),
			scrflFinding("Odd numbers", "Given is-even: the number 3", "Non-integers raise TypeError.", "three is an integer")),
		scrflConfirm(scrflVerdict(1, "confirmed", "four is even"), scrflVerdict(2, "confirmed", "three is an integer")),
	}
	exit, err := f.shape(scrflRequest)
	obs := scrflRun(f, exit, err)
	obs["feedback_1"] = scrflOutput(f, 1, "c1")
	obs["feedback_2"] = scrflOutput(f, 2, "c2")
	s := sys.NewWith(f.Env, f.Root)
	var out strings.Builder
	s.Stdout, s.Stderr = &out, &out
	_, aerr := Approve(context.Background(), s, ApproveInput{Project: f.Root, Tools: fixtureTools(s), Slug: shpSlug})
	_, obs["approval_lines"] = scrflError(aerr)
	return obs
}

func scrflCorrected(t *testing.T) map[string]any {
	corrected := strings.Replace(shpFeature, "Given is-even: the number 4", "Given is-even: the number 6", 1)
	correctedPackage := map[string]any{"files": []any{map[string]any{"path": scrflFeaturePath, "content": corrected}}}
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpSubmit("c2", correctedPackage), scrflIntentOnly("c3", shpScopedIntentMD))
	f.setFeatureReport("corrected-report", corrected, "FAILED", 1)
	f.Prov.contract = []shpStep{
		scrflReview(scrflFinding("Even numbers", "Given is-even: the number 4", "calc.is_even(n) returns True for even integers.", "four is even")),
		scrflConfirm(scrflVerdict(1, "confirmed", "four is even")),
	}
	exit, err := f.shape(scrflRequest)
	obs := scrflRun(f, exit, err)
	obs["feedback_1"] = scrflOutput(f, 1, "c1")
	obs["feedback_2"] = scrflOutput(f, 2, "c2")
	obs["feature"] = f.read(scrflFeaturePath)
	return obs
}

func scrflRefuted(t *testing.T) map[string]any {
	f := newShpFx(t, shpSubmit("c1", shpFiles()), scrflIntentOnly("c2", shpScopedIntentMD), scrflIntentOnly("c3", shpScopedIntentMD))
	f.Prov.contract = []shpStep{
		scrflReview(
			scrflFinding("Even numbers", "Given is-even: the number 4", "calc.is_even(n) returns True for even integers.", "four is even"),
			scrflFinding("Even numbers", "Given is-even: the number 5", "calc.is_even(n) returns True for even integers.", "five is not in the feature"),
			scrflFinding("Even numbers", "Given is-even: the number 4", "calc.is_even(n) returns False for even integers.", "the intent says otherwise")),
		scrflConfirm(scrflVerdict(1, "refuted", "a correct implementation can pass it")),
	}
	exit, err := f.shape(scrflRequest)
	obs := scrflRun(f, exit, err)
	obs["feedback_1"] = scrflOutput(f, 1, "c1")
	obs["feedback_2"] = scrflOutput(f, 2, "c2")
	return obs
}

const scrflLongQuote = "calc.is_even(n) returns True for even integers, which are the integers divisible by two without a remainder, including zero and every negative even integer, and it returns False for every odd integer, whatever its sign or size."

func scrflLong(t *testing.T) map[string]any {
	intent := strings.Replace(shpScopedIntentMD, "calc.is_even(n) returns True for even integers.", scrflLongQuote, 1)
	f := newShpFx(t, shpSubmit("c1", scrflPackage(intent)), scrflIntentOnly("c2", intent), scrflIntentOnly("c3", intent))
	f.Prov.contract = []shpStep{
		scrflReview(scrflFinding("Even numbers", "Given is-even: the number 4", scrflLongQuote, "four is even")),
		scrflConfirm(scrflVerdict(1, "confirmed", "four is even")),
	}
	exit, err := f.shape(scrflRequest)
	obs := scrflRun(f, exit, err)
	obs["feedback_1"] = scrflOutput(f, 1, "c1")
	return obs
}

func TestShapeContractRefusalFixLineAcceptance(t *testing.T) {
	out := os.Getenv("SCRFL_OUT")
	if out == "" {
		t.Fatal("SCRFL_OUT must be set")
	}
	obs := map[string]any{
		"two_confirmed": scrflTwoConfirmed(t),
		"corrected":     scrflCorrected(t),
		"refuted":       scrflRefuted(t),
		"long":          scrflLong(t),
	}
	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
