'use strict';
// The Go source of the k-shape-prompt-states-gate-forms acceptance harness, which
// k-shape-prompt-states-gate-forms__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_shape_prompt_states_gate_forms_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
)

// The acceptance harness of k-shape-prompt-states-gate-forms: one native Shape per case, with a
// scripted shaper and a scripted decision-check model, on the scripted-provider fixture project of
// helper_test.go. KSPG_OUT names the file that receives what every case showed, as JSON, for the
// acceptance steps to judge. The harness judges nothing itself.

type kspgCase struct {
	request string
	intent  string            // the intent.md of the first submission
	extra   map[string]string // further files of the first submission
	reads   []string          // files read before the first submission, besides README.md
	checks  []map[string]any  // the items of the first decision check answer; later answers have none
	second  bool              // the plain package is submitted again after the first submission
	watch   []string          // paths whose content is observed before and after the run
}

const (
	kspgRequest  = "Add calc.is_even(n) that returns True for even integers."
	kspgDecision = "- Non-integers raise TypeError. (source: request)"
	kspgProse    = "calc.is_even(n) returns True for even integers."
	kspgZero     = "- Zero counts as even (assumption: the user confirms that zero is even) (source: request)"
	kspgOdd      = "- Odd numbers return False (source: request)"
	kspgDir      = ".kogen/intents/is-even/"
)

func kspgDecisions(lines ...string) string {
	return strings.Replace(shpScopedIntentMD, kspgDecision, strings.Join(lines, "\n"), 1)
}

func kspgEvidence(item string) string {
	return kspgDecisions("- Tests live in tests/x_test.py (evidence: " + item + ") (source: request)")
}

func kspgItem(statement, quote, verdict, intentValue, requestValue string) map[string]any {
	quotes := []any{}
	if quote != "" {
		quotes = append(quotes, quote)
	}
	return map[string]any{"statement": statement, "request_quotes": quotes, "verdict": verdict,
		"intent_value": intentValue, "request_value": requestValue, "difference": "", "why": "synthetic scripted item"}
}

// kspgFiles are n files of size bytes under the Intent's fixtures folder, named <prefix><nn>.txt.
func kspgFiles(prefix string, n, size int) map[string]string {
	out := map[string]string{}
	for i := 1; i <= n; i++ {
		out[fmt.Sprintf("%sfixtures/%s%02d.txt", kspgDir, prefix, i)] = strings.Repeat("x", size)
	}
	return out
}

func kspgKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var kspgCases = func() map[string]kspgCase {
	evidenceReads := []string{"tests/x_test.py", "src/calc.py"}
	c := map[string]kspgCase{
		"prompt": {request: kspgRequest, intent: shpScopedIntentMD},
		"decision-marker-on-line": {request: kspgRequest,
			intent: kspgDecisions("- Even numbers are found with the remainder of division by 2 (evidence: README.md) (source: request)", kspgZero),
			checks: []map[string]any{
				kspgItem("Even numbers are found with the remainder of division by 2", "", "unsupported", "", ""),
				kspgItem("Zero counts as even", "", "unsupported", "", "")}},
		"decision-marker-next-line": {request: kspgRequest, intent: kspgDecisions(kspgOdd, kspgZero), second: true,
			checks: []map[string]any{kspgItem("Odd numbers return False", "", "unsupported", "", "")}},
		"decision-two-lines-one-marker": {request: kspgRequest, second: true,
			intent: strings.Replace(shpScopedIntentMD, kspgProse, kspgProse+" Negative numbers follow the same rule (evidence: README.md)\nand zero is even.", 1),
			checks: []map[string]any{kspgItem("Negative numbers follow the same rule (evidence: README.md) and zero is even.", "", "unsupported", "", "")}},
		"decision-fabricated-quote": {request: kspgRequest, intent: kspgDecisions(kspgOdd, kspgZero), second: true,
			checks: []map[string]any{kspgItem(kspgOdd, "is_even returns False for odd numbers", "follows", "", "")}},
		"decision-contradiction-beside-marked": {request: kspgRequest + " Limit inputs to 1,000.", second: true,
			intent: kspgDecisions("- Inputs are limited to 10,000 (source: request)", kspgZero),
			checks: []map[string]any{kspgItem("- Inputs are limited to 10,000 (source: request)", "Limit inputs to 1,000.", "contradicts", "10,000", "1,000")}},
		"decision-unmarked-unsupported": {request: kspgRequest, intent: kspgDecisions(kspgOdd), second: true,
			checks: []map[string]any{kspgItem(kspgOdd, "", "unsupported", "", "")}},
		"evidence-path-then-prose":   {request: kspgRequest, intent: kspgEvidence("tests/x_test.py and the check runs it"), reads: evidenceReads},
		"evidence-missing":           {request: kspgRequest, intent: kspgEvidence("src/missing.py"), reads: evidenceReads, second: true},
		"evidence-symbol":            {request: kspgRequest, intent: kspgEvidence("calc.add"), reads: evidenceReads, second: true},
		"evidence-see":               {request: kspgRequest, intent: kspgEvidence("see src/calc.py"), reads: evidenceReads, second: true},
		"evidence-untracked-later":   {request: kspgRequest, intent: kspgEvidence("tests/x_test.py and src/new.py"), reads: evidenceReads, second: true},
		"evidence-unread":            {request: kspgRequest, intent: kspgEvidence("src/calc.py and the check runs it"), reads: []string{"tests/x_test.py"}, second: true},
	}
	written := map[string]string{kspgDir + "fixtures/input.txt": "4\n"}
	for p, content := range kspgFiles("g", 14, 8) {
		written[p] = content
	}
	written[kspgDir+"fixtures/big.txt"] = strings.Repeat("x", 65536)
	c["write-fixture"] = kspgCase{request: kspgRequest, intent: shpScopedIntentMD, extra: written}
	refusedPath := func(path, content string) kspgCase {
		return kspgCase{request: kspgRequest, intent: shpScopedIntentMD, extra: map[string]string{path: content}, second: true, watch: []string{path}}
	}
	c["write-product"] = refusedPath("calc.js", "function add(a, b) { return a - b }\n")
	c["write-project"] = refusedPath(".kogen/project.yaml", "version: 1\n")
	c["write-other-intent"] = refusedPath(".kogen/intents/other/input.txt", "4\n")
	c["write-step-unprefixed"] = refusedPath("features/steps/helper.js", "// helper\n")
	c["write-steps-json"] = refusedPath("features/steps/is-even__data.json", "{}\n")
	c["write-traversal"] = refusedPath(kspgDir+"../other/x.txt", "4\n")
	many := kspgFiles("f", 17, 8)
	c["write-17-extra"] = kspgCase{request: kspgRequest, intent: shpScopedIntentMD, extra: many, second: true, watch: kspgKeys(many)}
	c["write-oversize"] = refusedPath(kspgDir+"fixtures/big.txt", strings.Repeat("x", 65537))
	return c
}()

func TestKShapePromptStatesGateFormsAcceptance(t *testing.T) {
	out := os.Getenv("KSPG_OUT")
	if out == "" {
		t.Fatal("k-shape-prompt-states-gate-forms: KSPG_OUT is empty")
	}
	var mu sync.Mutex
	seen := map[string]any{}
	t.Run("cases", func(t *testing.T) {
		for _, name := range slices.Sorted(func(yield func(string) bool) {
			for k := range kspgCases {
				if !yield(k) {
					return
				}
			}
		}) {
			c := kspgCases[name]
			t.Run(name, func(t *testing.T) {
				o := kspgShape(t, name, c)
				mu.Lock()
				seen[name] = o
				mu.Unlock()
			})
		}
	})
	b, err := json.MarshalIndent(seen, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("k-shape-prompt-states-gate-forms: writing the observations to %s: %v", out, err)
	}
}

func kspgShape(t *testing.T, name string, c kspgCase) map[string]any {
	first := []any{map[string]any{"path": kspgDir + "intent.md", "content": c.intent},
		map[string]any{"path": kspgDir + "acceptance.feature", "content": shpFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps}}
	for _, p := range kspgKeys(c.extra) {
		first = append(first, map[string]any{"path": p, "content": c.extra[p]})
	}
	calls := []map[string]any{shpCall("c1-readme", "read_file", map[string]any{"path": "README.md"})}
	for i, p := range c.reads {
		calls = append(calls, shpCall(fmt.Sprintf("c1-read-%d", i+1), "read_file", map[string]any{"path": p}))
	}
	calls = append(calls, shpCall("c1", "submit", map[string]any{"files": first}))
	steps := []shpStep{shpAnswer(calls...)}
	if c.second {
		steps = append(steps, shpSubmit("c2", shpFiles()))
	}
	f := newShpFx(t, steps...)
	f.write("tests/x_test.py", "def test_x():\n    assert True\n")
	f.write("src/calc.py", "def add(a, b):\n    return a + b\n")
	f.git("add", "-A")
	f.git("commit", "-qm", "evidence files")
	f.setReport("PASSED", 0)
	watch := append(append([]string{}, c.watch...), kspgKeys(c.extra)...)
	slices.Sort(watch)
	watch = slices.Compact(watch)
	content := func(p string) any {
		b, err := os.ReadFile(filepath.Join(f.Root, p))
		if err != nil {
			return nil
		}
		return string(b)
	}
	before := map[string]any{}
	for _, p := range watch {
		before[p] = content(p)
	}
	var mu sync.Mutex
	answered := 0
	f.Prov.route = func(req map[string]any) *shpStep {
		if !decisionRequest(req) {
			return nil
		}
		mu.Lock()
		n := answered
		answered++
		mu.Unlock()
		items := []any{}
		if n == 0 {
			for _, it := range c.checks {
				items = append(items, it)
			}
		}
		st := shpAnswer(shpCall(fmt.Sprintf("decision-%d", n+1), SubmitDecisionCheckTool, map[string]any{"items": items}))
		return &st
	}
	exit, err := f.shape(c.request)
	o := map[string]any{"case": name, "request": c.request, "first_intent": c.intent, "exit": exit, "err": fmt.Sprint(err),
		"stderr": f.Stderr.String()}
	o["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	o["intent"] = content(kspgDir + "intent.md")
	shaper := f.Prov.requests()
	o["shaper"], o["decision"] = len(shaper), len(f.Prov.decisionRequests())
	o["instructions"], o["first_feedback"] = "", ""
	if len(shaper) > 0 {
		o["instructions"] = fmt.Sprint(shaper[0]["instructions"])
	}
	for _, r := range shaper {
		if text, ok := shpOutputs(r)["c1"]; ok {
			o["first_feedback"] = text
			break
		}
	}
	files := []any{}
	for _, p := range watch {
		var submitted any
		if s, ok := c.extra[p]; ok {
			submitted = s
		}
		files = append(files, map[string]any{"path": p, "submitted": submitted, "before": before[p], "after": content(p)})
	}
	o["files"] = files
	return o
}
`;
