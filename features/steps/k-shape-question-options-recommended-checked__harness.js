'use strict';
// The Go source of the k-shape-question-options-recommended-checked acceptance harness, which
// k-shape-question-options-recommended-checked__steps.js compiles into the shape package's tests through a
// Go overlay as shape/k_shape_question_options_recommended_checked_acceptance_test.go; that path never
// exists in the repository. The source holds no backquote and no dollar-brace, so this raw template
// keeps it byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of k-shape-question-options-recommended-checked: synthetic shaping
// journeys of the native shaper against the shape package's scripted local provider, which plays
// the shaper and Kogen's question check, in the shape tests' invented is-even project. KQOR_CASE
// names one case; what the runs showed is written as JSON to KQOR_OUT for the acceptance steps to
// judge. The harness judges nothing itself.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
)

const (
	kqorRequest = "Add calc.is_even(n), which returns true for even integers.\n"
	kqorUI      = "How should calc.is_even(n) answer a number that is not an integer?"
	kqorHandoff = "Sign in to the npm registry in your browser so that acceptance can install the is-even test tools."
)

// kqorOpt is one option; recommended nil leaves the mark out.
func kqorOpt(label string, recommended any) any {
	o := map[string]any{"label": label}
	if recommended != nil {
		o["recommended"] = recommended
	}
	return o
}

// kqorQ is one question; options nil leaves the options out.
func kqorQ(text, kind string, options []any) map[string]any {
	q := map[string]any{"question": text}
	if kind != "" {
		q["kind"] = kind
	}
	if options != nil {
		q["options"] = options
	}
	return q
}

func kqorAsk(questions ...map[string]any) map[string]any {
	list := []any{}
	for _, q := range questions {
		list = append(list, q)
	}
	return map[string]any{"questions": list}
}

var (
	kqorGood          = kqorQ(kqorUI, "", []any{kqorOpt("Raise TypeError", true), kqorOpt("Return false", nil)})
	kqorNoOptions     = kqorQ(kqorUI, "", nil)
	kqorEmptyList     = kqorQ(kqorUI, "", []any{})
	kqorOneOption     = kqorQ(kqorUI, "", []any{kqorOpt("Raise TypeError", true)})
	kqorNoRecommended = kqorQ(kqorUI, "", []any{kqorOpt("Raise TypeError", nil), kqorOpt("Return false", false)})
	kqorTwoMarked     = kqorQ(kqorUI, "", []any{kqorOpt("Raise TypeError", true), kqorOpt("Return false", true)})
	kqorRepeated      = kqorQ(kqorUI, "", []any{kqorOpt("Raise TypeError", true), kqorOpt("Raise TypeError", nil)})
	kqorEmptyLabel    = kqorQ(kqorUI, "", []any{kqorOpt("Raise TypeError", true), kqorOpt("  ", nil)})
	kqorSecond        = kqorQ("Should calc.is_even(n) accept numbers given as strings?", "", []any{kqorOpt("No, only numbers", true), kqorOpt("Yes, parse them", nil)})
	kqorSecondBad     = kqorQ("Should calc.is_even(n) accept numbers given as strings?", "", []any{kqorOpt("No, only numbers", nil), kqorOpt("Yes, parse them", nil)})
	kqorSignIn        = kqorQ(kqorHandoff, "handoff", []any{kqorOpt("I have signed in", true), kqorOpt("Pause shaping until I have signed in", nil)})
	kqorSignInWait    = kqorQ(kqorHandoff, "handoff", []any{kqorOpt("Wait: pause shaping until I have signed in", true), kqorOpt("I have signed in", nil)})
)

// kqorPackage is the published package: the shape tests' scoped is-even package with one Decision.
func kqorPackage(decision string) (map[string]any, string) {
	files := shpFilesWithDecisions(decision)
	intent := files["files"].([]any)[0].(map[string]any)["content"].(string)
	return files, intent
}

type kqorCase struct {
	steps   []shpStep // the shaper's answers in order
	classes []string  // the scripted question check's class for each question, whenever it is asked
	answer  string    // the second run's input; "" when the case has one run
	want    string    // the intent.md the second run publishes
}

// kqorRepair is a case whose first questions submission is malformed and whose second is
// well-formed; with an answer, the second run publishes the package.
func kqorRepair(bad, good map[string]any, n int, answer string) kqorCase {
	classes := make([]string, n)
	for i := range classes {
		classes[i] = "ui_ux"
	}
	c := kqorCase{steps: []shpStep{shpSubmit("q1", bad), shpSubmit("q2", good)}, classes: classes}
	if answer != "" {
		files, intent := kqorPackage(kqorDecision)
		c.steps, c.answer, c.want = append(c.steps, shpSubmit("f1", files)), answer, intent
	}
	return c
}

const kqorDecision = "- Non-integers raise TypeError. (source: answer 1)"

func kqorCases() map[string]kqorCase {
	signedIn, signedInIntent := kqorPackage("- Acceptance installs the is-even test tools after the sign-in. (source: answer 1)")
	returnsFalse, returnsFalseIntent := kqorPackage("- Non-integers return false. (source: answer 1)")
	return map[string]kqorCase{
		"no-options":      kqorRepair(kqorAsk(kqorNoOptions), kqorAsk(kqorGood), 1, "all: recommended"),
		"empty-options":   kqorRepair(kqorAsk(kqorEmptyList), kqorAsk(kqorGood), 1, "all: recommended"),
		"no-recommended":  kqorRepair(kqorAsk(kqorNoRecommended), kqorAsk(kqorGood), 1, "all: recommended"),
		"two-recommended": kqorRepair(kqorAsk(kqorTwoMarked), kqorAsk(kqorGood), 1, ""),
		"repeated-label":  kqorRepair(kqorAsk(kqorRepeated), kqorAsk(kqorGood), 1, ""),
		"empty-label":     kqorRepair(kqorAsk(kqorEmptyLabel), kqorAsk(kqorGood), 1, ""),
		"one-option":      kqorRepair(kqorAsk(kqorOneOption), kqorAsk(kqorGood), 1, ""),
		"second-of-two":   kqorRepair(kqorAsk(kqorGood, kqorSecondBad), kqorAsk(kqorGood, kqorSecond), 2, ""),
		"bound": {steps: []shpStep{
			shpSubmit("q1", kqorAsk(kqorNoOptions)), shpSubmit("q2", kqorAsk(kqorNoRecommended)), shpSubmit("q3", kqorAsk(kqorTwoMarked))},
			classes: []string{"ui_ux"}},
		"handoff": {steps: []shpStep{shpSubmit("q1", kqorAsk(kqorSignIn)), shpSubmit("f1", signedIn)},
			classes: []string{"handoff"}, answer: "all: recommended", want: signedInIntent},
		"handoff-wait": {steps: []shpStep{shpSubmit("q1", kqorAsk(kqorSignInWait))}, classes: []string{"handoff"}},
		"ui": {steps: []shpStep{shpSubmit("q1", kqorAsk(kqorGood)), shpSubmit("f1", returnsFalse)},
			classes: []string{"ui_ux"}, answer: "1: b", want: returnsFalseIntent},
	}
}

// kqorKind is what a provider request is: the shaper, or one of Kogen's checks.
func kqorKind(req map[string]any) string {
	switch {
	case contractKind(req) != "":
		return "contract_check"
	case decisionRequest(req):
		return "decision_check"
	case literalRequest(req):
		return "literal_check"
	}
	for _, n := range shpToolNames(req) {
		if n == SubmitQuestionCheckTool {
			return "question_check"
		}
	}
	return "shaper"
}

// kqorUserText is the text of the request's first user message.
func kqorUserText(req map[string]any) string {
	input, _ := req["input"].([]any)
	for _, it := range input {
		if m, _ := it.(map[string]any); m != nil && m["role"] == "user" {
			if s, ok := m["content"].(string); ok {
				return s
			}
		}
	}
	return ""
}

// kqorJSON is a JSON file of the fixture decoded, nil when it does not exist.
func kqorJSON(f *shpFx, rel string) map[string]any {
	b, err := os.ReadFile(filepath.Join(f.Root, rel))
	if err != nil {
		return nil
	}
	var doc map[string]any
	if json.Unmarshal(b, &doc) != nil {
		return map[string]any{"undecodable": string(b)}
	}
	return doc
}

var (
	kqorCallID = regexp.MustCompile("^[qf][0-9]$")
	kqorRound  = regexp.MustCompile("Round 1:\n[\\s\\S]*?\n   answer: 1: [^\n]*")
)

// kqorRun is one run's exit, error and output.
func kqorRun(obs map[string]any, prefix string, f *shpFx, exit int, err error) {
	obs[prefix+"exit"], obs[prefix+"error"], obs[prefix+"error_lines"] = exit, "", []string{}
	lines := strings.Split(strings.TrimRight(f.Stdout.String(), "\n"), "\n")
	obs[prefix+"stdout"], obs[prefix+"last_stdout_line"] = f.Stdout.String(), lines[len(lines)-1]
	if err != nil {
		obs[prefix+"error"] = err.Error()
		if e, ok := sys.AsError(err); ok {
			obs[prefix+"exit"] = e.Exit
			if len(e.Lines) > 0 {
				obs[prefix+"error_lines"] = e.Lines
			}
		}
	}
}

func TestKShapeQuestionOptionsRecommendedCheckedAcceptance(t *testing.T) {
	name, out := os.Getenv("KQOR_CASE"), os.Getenv("KQOR_OUT")
	c, ok := kqorCases()[name]
	if !ok || out == "" {
		t.Fatalf("KQOR_CASE must name a case (no-options, empty-options, no-recommended, two-recommended, repeated-label, empty-label, one-option, second-of-two, bound, handoff, handoff-wait, ui), not %q, and KQOR_OUT must be set", name)
	}
	f := newShpFx(t, c.steps...)
	f.setReport("PASSED", 0)
	check := shpQuestionCheck("qc", c.classes...)
	f.Prov.route = func(req map[string]any) *shpStep {
		if kqorKind(req) == "question_check" {
			return &check
		}
		return nil
	}

	obs := map[string]any{"case": name}
	exit, err := f.shape(kqorRequest)
	kqorRun(obs, "", f, exit, err)
	first := len(f.Prov.allRequests())
	gates := []any{}
	if b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping/is-even/events.ndjson")); rerr == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if line != "" && json.Unmarshal([]byte(line), &ev) == nil && ev["event"] == "submission_refused" {
				gates = append(gates, ev["gate"])
			}
		}
	}
	obs["refused_gates"] = gates
	var pending any
	if doc := kqorJSON(f, ".kogen/local/shaping/is-even/questions.json"); doc != nil {
		pending = doc["data"]
	}
	obs["questions_json"] = pending
	var stored any
	if doc := kqorJSON(f, ".kogen/local/shaping/is-even/state.json"); doc != nil {
		rounds, _ := doc["rounds"].([]any)
		list := []any{}
		for _, r := range rounds {
			if m, _ := r.(map[string]any); m != nil {
				list = append(list, m["questions"])
			}
		}
		stored = list
	}
	obs["state_rounds"] = stored

	if c.answer != "" {
		exit, err := f.shape(c.answer)
		kqorRun(obs, "answer_", f, exit, err)
		delete(obs, "answer_stdout")
		published := false
		if b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md")); rerr == nil {
			published = string(b) == c.want && f.exists(".kogen/local/readiness/is-even.json")
		}
		obs["published"] = published
		obs["pending_after_answer"] = f.exists(".kogen/local/shaping/is-even/questions.json") || f.exists(".kogen/local/shaping/is-even/state.json")
		var answers, resolved any
		if doc := kqorJSON(f, ".kogen/local/provenance/is-even/current.json"); doc != nil {
			if rounds, _ := doc["rounds"].([]any); len(rounds) > 0 {
				if r, _ := rounds[0].(map[string]any); r != nil {
					answers, resolved = r["answers"], r["resolved_answers"]
				}
			}
		}
		obs["receipt_answers"], obs["receipt_resolved_answers"] = answers, resolved
		obs["earlier_round"] = ""
		for _, r := range f.Prov.allRequests()[first:] {
			if kqorKind(r) == "shaper" {
				obs["earlier_round"] = kqorRound.FindString(shpBody(r))
				break
			}
		}
	}

	kinds := []any{}
	checks := 0
	for _, r := range f.Prov.allRequests() {
		k := kqorKind(r)
		kinds = append(kinds, k)
		switch k {
		case "question_check":
			checks++
			obs["question_check_input_"+string(rune('0'+checks))] = kqorUserText(r)
		case "shaper":
			for id, text := range shpOutputs(r) {
				if kqorCallID.MatchString(id) {
					obs["feedback_"+id] = text
				}
			}
		}
	}
	obs["request_kinds"] = kinds
	b, jerr := json.MarshalIndent(obs, "", " ")
	if jerr != nil {
		t.Fatal(jerr)
	}
	if werr := os.WriteFile(out, b, 0o644); werr != nil {
		t.Fatal(werr)
	}
}
`;
