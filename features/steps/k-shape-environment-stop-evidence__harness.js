'use strict';
// The Go source of the k-shape-environment-stop-evidence acceptance harness, which
// k-shape-environment-stop-evidence__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_shape_environment_stop_evidence_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of k-shape-environment-stop-evidence: synthetic shaping journeys of the
// native shaper against the shape package's scripted local provider, which plays both the shaper
// and Kogen's question check, in the shape tests' is-even project with an invented frost forecast
// fixture committed on top. KSES_CASE names one case; what the run showed is written as JSON to
// KSES_OUT for the acceptance steps to judge. The harness judges nothing itself.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
)

const kseBacktick = "\x60"

// The stand-in fixture: production reads the key, and test support has a fake client.
var kseStandIn = map[string]string{
	"lib/weather.js": "// The Skyline weather client: production reads the API key from SKYLINE_API_KEY.\n" +
		"exports.client = function () {\n  return {key: process.env.SKYLINE_API_KEY}\n}\n",
	"test/support/fake_weather.js": "// A fake Skyline weather client: acceptance uses it, and no API key is needed.\n" +
		"exports.client = function () {\n  return {forecast: () => ({frost: true})}\n}\n",
}

// The live fixture: every call reaches the live service, and there is no fake client.
var kseLive = map[string]string{
	"lib/weather.js": "// The Skyline weather client. There is no fake client: every call reaches the live Skyline service with the API key from SKYLINE_API_KEY.\n" +
		"exports.client = function () {\n  return {key: process.env.SKYLINE_API_KEY}\n}\n",
}

const (
	kseProductionRequest = "bin/frost-forecast must print tomorrow's frost risk for the orchard from the Skyline weather API; production runs read the API key from SKYLINE_API_KEY.\n"
	kseLiveRequest       = "bin/frost-forecast must print tomorrow's frost risk for the orchard from the Skyline weather API, and its acceptance must call the live Skyline service with the API key from SKYLINE_API_KEY.\n"
	kseProbe             = "ls test/support 2>/dev/null || echo no test/support folder"
)

func kseQ(text string, options ...string) map[string]any {
	list := []any{}
	for i, o := range options {
		list = append(list, map[string]any{"label": o, "recommended": i == 0})
	}
	return map[string]any{"questions": []any{map[string]any{"question": text, "options": list}}}
}

var (
	kseKey          = kseQ("Is SKYLINE_API_KEY set where bin/frost-forecast runs?", "Assume it is set", "Assume it is not")
	kseContradicted = kseQ("Is SKYLINE_API_KEY set where bin/frost-forecast runs? A live Skyline service is essential (evidence: test/support/fake_weather.js)", "Assume it is set", "Assume it is not")
	kseLiveKey      = kseQ("Is SKYLINE_API_KEY set where the acceptance of bin/frost-forecast runs? Its acceptance must call the live Skyline service (evidence: lib/weather.js; ran "+kseBacktick+kseProbe+kseBacktick+")", "Assume it is set", "Assume it is not")
	ksePolicy       = kseQ("Should a frost risk below 10% be printed at all?", "Print it", "Skip it")
	kseUI           = kseQ("Should bin/frost-forecast print the risk as a percentage or as a word (low, medium, high)?", "A percentage", "A word")
)

// kseCheck is a question check answer for one question.
func kseCheck(id, class, why string) shpStep {
	return shpAnswer(shpCall(id, "submit_question_check", map[string]any{"items": []any{map[string]any{"number": 1, "class": class, "why": why}}}))
}

const (
	kseWhyProduction = "the key is needed only to run bin/frost-forecast in production"
	kseWhyStandIn    = "test/support/fake_weather.js is a fake Skyline client: acceptance uses it, and no API key is needed"
	kseWhyLive       = "the acceptance step that calls the live Skyline service cannot run without SKYLINE_API_KEY: lib/weather.js has no fake client, and the probe found no test/support folder"
	kseWhyPolicy     = "whether a low risk is printed is a product rule"
	kseWhyUI         = "how the risk is shown is an interface choice"
)

// kseIntent is the published Intent: the shape tests' scoped is-even package with one assumption
// Decision about the production key.
const kseDecision = "- Acceptance uses the fake Skyline client of test/support/fake_weather.js, so shaping and acceptance need no SKYLINE_API_KEY (assumption: whether SKYLINE_API_KEY is set where bin/frost-forecast runs in production) (source: request)"

var kseIntent = strings.Replace(shpScopedIntentMD, "- Non-integers raise TypeError. (source: request)", kseDecision, 1)

func ksePackage() map[string]any {
	return map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": kseIntent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
	}}
}

var (
	kseReadFake   = shpAnswer(shpCall("r1", "read_file", map[string]any{"path": "test/support/fake_weather.js"}))
	kseReadAndRun = shpAnswer(shpCall("r1", "read_file", map[string]any{"path": "lib/weather.js"}), shpCall("p1", "probe", map[string]any{"command": kseProbe}))
	kseCallID     = regexp.MustCompile("^[qfrp][0-9]$")
	kseWall       = regexp.MustCompile("(?m)^wall: [0-9.]+ s$")
)

type kseCase struct {
	fixture map[string]string
	request string
	steps   []shpStep
	want    string // the intent.md a published run writes; "" when the case publishes nothing
}

var kseCases = map[string]kseCase{
	"stand-in": {kseStandIn, kseProductionRequest, []shpStep{
		shpSubmit("q1", kseKey), kseCheck("qc1", "environment", kseWhyProduction),
		shpSubmit("f1", ksePackage())}, kseIntent},
	"contradicted": {kseStandIn, kseProductionRequest, []shpStep{
		kseReadFake,
		shpSubmit("q1", kseContradicted), kseCheck("qc1", "environment", kseWhyStandIn),
		shpSubmit("q2", kseContradicted), kseCheck("qc2", "environment", kseWhyStandIn),
		shpSubmit("f1", ksePackage())}, kseIntent},
	"live": {kseLive, kseLiveRequest, []shpStep{
		kseReadAndRun,
		shpSubmit("q1", kseLiveKey), kseCheck("qc1", "environment_prerequisite", kseWhyLive),
		shpSubmit("q2", kseLiveKey), kseCheck("qc2", "environment_prerequisite", kseWhyLive)}, ""},
	"spent": {kseStandIn, kseProductionRequest, []shpStep{
		shpSubmit("q1", ksePolicy), kseCheck("qc1", "product_policy", kseWhyPolicy),
		shpSubmit("q2", ksePolicy), kseCheck("qc2", "product_policy", kseWhyPolicy),
		shpSubmit("q3", kseKey), kseCheck("qc3", "environment", kseWhyProduction)}, ""},
	"spent-evidenced": {kseLive, kseLiveRequest, []shpStep{
		shpSubmit("q1", ksePolicy), kseCheck("qc1", "product_policy", kseWhyPolicy),
		shpSubmit("q2", ksePolicy), kseCheck("qc2", "product_policy", kseWhyPolicy),
		kseReadAndRun,
		shpSubmit("q3", kseLiveKey), kseCheck("qc3", "environment_prerequisite", kseWhyLive)}, ""},
	"spent-ui": {kseStandIn, kseProductionRequest, []shpStep{
		shpSubmit("q1", ksePolicy), kseCheck("qc1", "product_policy", kseWhyPolicy),
		shpSubmit("q2", ksePolicy), kseCheck("qc2", "product_policy", kseWhyPolicy),
		shpSubmit("q3", kseUI), kseCheck("qc3", "ui_ux", kseWhyUI)}, ""},
}

// kseIsCheck is whether a request is a question check: it offers submit_question_check.
func kseIsCheck(req map[string]any) bool {
	for _, n := range shpToolNames(req) {
		if n == "submit_question_check" {
			return true
		}
	}
	return false
}

// kseUserText is the text of the request's first user message.
func kseUserText(req map[string]any) string {
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

// kseClassEnum is the class enum of a question check request's one tool.
func kseClassEnum(req map[string]any) []any {
	tl, _ := req["tools"].([]any)
	if len(tl) != 1 {
		return nil
	}
	m, _ := tl[0].(map[string]any)
	for _, k := range []string{"parameters", "properties", "items", "items", "properties", "class"} {
		m, _ = m[k].(map[string]any)
	}
	e, _ := m["enum"].([]any)
	return e
}

func TestKShapeEnvironmentStopEvidenceAcceptance(t *testing.T) {
	name, out := os.Getenv("KSES_CASE"), os.Getenv("KSES_OUT")
	c, ok := kseCases[name]
	if !ok || out == "" {
		t.Fatalf("KSES_CASE must name a case (stand-in, contradicted, live, spent, spent-evidenced, spent-ui), not %q, and KSES_OUT must be set", name)
	}
	f := newShpFx(t, c.steps...)
	for p, content := range c.fixture {
		f.write(p, content)
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "Add the frost forecast fixture")
	f.setReport("PASSED", 0)

	exit, err := f.shape(c.request)
	lines := strings.Split(strings.TrimRight(f.Stdout.String(), "\n"), "\n")
	obs := map[string]any{"case": name, "exit": exit, "error": "", "error_lines": []string{}, "error_last_line": "",
		"stdout": f.Stdout.String(), "last_stdout_line": lines[len(lines)-1], "stderr": f.Stderr.String()}
	if err != nil {
		obs["error"] = err.Error()
		if e, ok := sys.AsError(err); ok {
			obs["exit"] = e.Exit
			if len(e.Lines) > 0 {
				obs["error_lines"] = e.Lines
				obs["error_last_line"] = e.Lines[len(e.Lines)-1]
			}
		}
	}

	var shaper, checks []map[string]any
	for _, r := range f.Prov.requests() {
		if kseIsCheck(r) {
			checks = append(checks, r)
		} else {
			shaper = append(shaper, r)
		}
	}
	obs["shaper_requests"], obs["question_check_requests"] = len(shaper), len(checks)
	for i, r := range checks {
		obs["question_check_input_"+string(rune('1'+i))] = kseWall.ReplaceAllString(kseUserText(r), "wall: <wall> s")
	}
	if len(checks) > 0 {
		obs["question_check_instructions"], _ = checks[0]["instructions"].(string)
		obs["question_check_class_enum"] = kseClassEnum(checks[0])
	}
	for _, r := range shaper {
		for id, text := range shpOutputs(r) {
			switch {
			case !kseCallID.MatchString(id):
			case id[0] == 'q' || id[0] == 'f':
				obs["feedback_"+id] = text
			default:
				obs["output_"+id] = kseWall.ReplaceAllString(text, "wall: <wall> s")
			}
		}
	}

	gates, classes := []any{}, []any{}
	if b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping/is-even/events.ndjson")); rerr == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if line == "" || json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			switch ev["event"] {
			case "submission_refused":
				gates = append(gates, ev["gate"])
			case "question_check":
				cs := []any{}
				items, _ := ev["items"].([]any)
				for _, it := range items {
					if m, _ := it.(map[string]any); m != nil {
						cs = append(cs, m["class"])
					}
				}
				classes = append(classes, cs)
			}
		}
	}
	obs["refused_gates"], obs["question_check_classes"] = gates, classes
	obs["questions_pending"] = f.exists(".kogen/local/shaping/is-even/questions.json")
	obs["intent_exists"] = f.exists(".kogen/intents/is-even/intent.md")
	published := false
	if b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md")); rerr == nil {
		published = c.want != "" && string(b) == c.want && f.exists(".kogen/local/readiness/is-even.json")
	}
	obs["published"] = published
	b, jerr := json.MarshalIndent(obs, "", " ")
	if jerr != nil {
		t.Fatal(jerr)
	}
	if werr := os.WriteFile(out, b, 0o644); werr != nil {
		t.Fatal(werr)
	}
}
`;
