'use strict';
// The Go source of the k-shape-request-literal-coverage acceptance harness, which
// k-shape-request-literal-coverage__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_request_literal_coverage_acceptance_test.go; that path never exists in the
// repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// The acceptance harness of k-shape-request-literal-coverage: one native Shape per synthetic
// request and candidate, with a scripted shaper and a scripted literal-check model. KRLC_CASE
// names the case; what a user and the shaper see of the run is written as JSON to KRLC_OUT for
// the acceptance steps to judge. On a tree without a literal check no request offers
// submit_literal_check and the script is never used.

const krlcTool = "submit_literal_check"

const (
	krlcRequest     = "Add a registration example to the README.\nThe example must register exactly these two repositories: /demo_one and /demo-two.\nAn earlier draft used /demo_old; it is superseded by /demo_one.\nKeep the wording friendly."
	krlcQuote       = "The example must register exactly these two repositories: /demo_one and /demo-two."
	krlcOldQuote    = "An earlier draft used /demo_old; it is superseded by /demo_one."
	krlcSecretLine  = "password: hunter2-demo-secret"
	krlcInformal    = "For background only, an old prototype once used /legacy-demo; it is not part of this change."
	krlcPortRequest = "Add a port example to the README."
	krlcPortAnswer  = "The example uses port 7443"

	krlcNone        = "- The README example registers both repositories. (source: request)"
	krlcBoth        = "- The README example registers /demo_one and /demo-two. (source: request)"
	krlcSubstituted = "- The README example registers /demo-one and /demo-two. (source: request)"
	krlcOne         = "- The README example registers /demo_one. (source: request)"
	krlcFriendly    = "- The README example registers /demo_one and /demo-two in a welcoming tone. (source: request)"
	krlcPortNone    = "- The README example shows the chosen port. (source: answer 1)"
	krlcPort        = "- The README example shows port 7443. (source: answer 1)"
)

func krlcItem(literal, source, quote, status, packageValue string) map[string]any {
	return map[string]any{"literal": literal, "source": source, "source_quote": quote, "status": status,
		"package_value": packageValue, "why": "synthetic scripted item"}
}

var (
	krlcDemoOne = krlcItem("/demo_one", "request", krlcQuote, "required", "")
	krlcDemoTwo = krlcItem("/demo-two", "request", krlcQuote, "required", "")
	krlcDemoOld = krlcItem("/demo_old", "request", krlcOldQuote, "superseded", "")
)

type krlcCase struct {
	request     string
	answer      string // the answers of a second run after a question round; "" for one run
	dropRequest bool   // the shaping state loses its request text before the second run
	first       string // the Decisions line of the first submission
	second      string // the Decisions line of the corrected submission; "" when there is none
	items       []map[string]any
}

var krlcCases = map[string]krlcCase{
	"omitted": {request: krlcRequest, first: krlcNone, second: krlcBoth,
		items: []map[string]any{krlcDemoOne, krlcDemoTwo, krlcDemoOld}},
	"substituted": {request: krlcRequest, first: krlcSubstituted, second: krlcBoth,
		items: []map[string]any{krlcItem("/demo_one", "request", krlcQuote, "required", "/demo-one"), krlcDemoTwo, krlcDemoOld}},
	"complete": {request: krlcRequest, first: krlcFriendly,
		items: []map[string]any{krlcDemoOne, krlcDemoTwo, krlcDemoOld,
			krlcItem("/demo_three", "request", "The example must register /demo_three.", "required", "")}},
	"informal": {request: krlcRequest + "\n" + krlcInformal, first: krlcBoth,
		items: []map[string]any{krlcDemoOne, krlcDemoTwo, krlcDemoOld,
			krlcItem("/legacy-demo", "request", krlcInformal, "informal", "")}},
	"answer": {request: krlcPortRequest, answer: "1: " + krlcPortAnswer, first: krlcPortNone, second: krlcPort,
		items: []map[string]any{krlcItem("7443", "answer 1", krlcPortAnswer, "required", "")}},
	"unknown": {request: krlcRequest, answer: "1: " + krlcPortAnswer, dropRequest: true, first: krlcNone, second: krlcBoth,
		items: []map[string]any{krlcDemoOne, krlcDemoTwo}},
	"secret": {request: krlcRequest + "\nSign in with " + krlcSecretLine + ".", first: krlcOne, second: krlcBoth,
		items: []map[string]any{krlcDemoOne, krlcDemoTwo, krlcDemoOld,
			krlcItem("hunter2-demo-secret", "request", krlcSecretLine, "required", "")}},
}

func krlcIntent(line string) string {
	return strings.Replace(shpScopedIntentMD, "- Non-integers raise TypeError. (source: request)", line, 1)
}

func TestKShapeRequestLiteralCoverageAcceptance(t *testing.T) {
	name, out := os.Getenv("KRLC_CASE"), os.Getenv("KRLC_OUT")
	c, ok := krlcCases[name]
	if !ok || out == "" {
		t.Fatalf("k-shape-request-literal-coverage: KRLC_CASE %q is not a case of the harness, or KRLC_OUT is empty", name)
	}
	files := func(line string) map[string]any {
		return map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": krlcIntent(line)},
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
			map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
		}}
	}
	var steps []shpStep
	if c.answer != "" {
		question := map[string]any{"question": "Which port should the README example show?", "options": []any{
			map[string]any{"label": "8080", "recommended": true}, map[string]any{"label": "Another port"}}}
		steps = append(steps, shpSubmit("q1", map[string]any{"questions": []any{question}}), shpUI(1))
	}
	steps = append(steps, shpSubmit("c1", files(c.first)))
	if c.second != "" {
		steps = append(steps, shpSubmit("c2", files(c.second)))
	}
	f := newShpFx(t, steps...)
	f.setReport("PASSED", 0)
	items := make([]any, len(c.items))
	for i, it := range c.items {
		items[i] = it
	}
	var mu sync.Mutex
	answered := 0
	f.Prov.route = func(req map[string]any) *shpStep {
		if !slices.Contains(shpToolNames(req), krlcTool) {
			return nil
		}
		mu.Lock()
		answered++
		n := answered
		mu.Unlock()
		st := shpAnswer(shpCall(fmt.Sprintf("literal-%d", n), krlcTool, map[string]any{"items": items}))
		return &st
	}
	before := 0
	var exit int
	var err error
	if c.answer != "" {
		qexit, qerr := f.shape(c.request)
		if qerr != nil || qexit != 5 {
			t.Fatalf("k-shape-request-literal-coverage (KRLC_CASE=%s): the question round ended with exit %d, err %v, want exit 5 (questions)\nstdout:\n%s\nstderr:\n%s", name, qexit, qerr, f.Stdout.String(), f.Stderr.String())
		}
		if c.dropRequest {
			path := filepath.Join(f.Root, ".kogen/local/shaping/is-even/state.json")
			var doc map[string]any
			b, rerr := os.ReadFile(path)
			if rerr == nil {
				rerr = json.Unmarshal(b, &doc)
			}
			if rerr != nil {
				t.Fatalf("k-shape-request-literal-coverage (KRLC_CASE=%s): reading the shaping state %s after the question round: %v", name, path, rerr)
			}
			doc["request"] = ""
			if b, rerr = json.Marshal(doc); rerr == nil {
				rerr = os.WriteFile(path, b, 0o600)
			}
			if rerr != nil {
				t.Fatalf("k-shape-request-literal-coverage (KRLC_CASE=%s): removing the request from %s: %v", name, path, rerr)
			}
		}
		before = len(f.Prov.allRequests())
		exit, err = f.shape(c.answer)
	} else {
		exit, err = f.shape(c.request)
	}
	seen := map[string]any{"case": name, "request": c.request, "answer": c.answer, "first": c.first, "second": c.second,
		"first_intent": krlcIntent(c.first), "exit": exit, "err": fmt.Sprint(err), "stdout": f.Stdout.String(), "stderr": f.Stderr.String()}
	seen["published"] = exit == 0 && err == nil && strings.Contains(f.Stdout.String(), "Next: kogen intent approve is-even")
	seen["intent"] = ""
	if f.exists(".kogen/intents/is-even/intent.md") {
		seen["intent"] = f.read(".kogen/intents/is-even/intent.md")
	}
	seen["approved"] = f.exists(".kogen/local/approvals/is-even.json")
	shaper, literal := 0, 0
	feedback := []any{}
	fed := map[string]bool{}
	firstFeedback, body, input := "", "", ""
	for i, r := range f.Prov.allRequests() {
		names := shpToolNames(r)
		switch {
		case slices.Contains(names, krlcTool):
			if literal == 0 {
				body = shpBody(r)
				input = shpBody(map[string]any{"input": r["input"]})
			}
			literal++
		case contractKind(r) != "" || decisionRequest(r) || slices.Contains(names, SubmitQuestionCheckTool):
		case i >= before:
			shaper++
			outs := shpOutputs(r)
			for _, id := range []string{"c1", "c2"} {
				if o, ok := outs[id]; ok && !fed[id] {
					fed[id] = true
					feedback = append(feedback, o)
					if id == "c1" {
						firstFeedback = o
					}
				}
			}
		}
	}
	seen["shaper"], seen["literal"] = shaper, literal
	seen["feedback"], seen["first_feedback"] = feedback, firstFeedback
	seen["body"], seen["input"] = body, input
	events, gates := []any{}, []any{}
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping/is-even/events.ndjson")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			switch ev["event"] {
			case "literal_check":
				events = append(events, ev)
			case "submission_refused":
				gates = append(gates, ev["gate"])
			}
		}
	}
	seen["events"], seen["gates"] = events, gates
	b, err := json.MarshalIndent(seen, "", "  ")
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		t.Fatalf("k-shape-request-literal-coverage: writing the observations of case %s to %s: %v", name, out, err)
	}
}
`;
