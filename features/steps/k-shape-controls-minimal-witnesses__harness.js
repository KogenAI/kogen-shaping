'use strict';
// The Go source of the k-shape-controls-minimal-witnesses acceptance harness, which
// k-shape-controls-minimal-witnesses__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_shape_controls_minimal_witnesses_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of k-shape-controls-minimal-witnesses. Each case builds the real-run
// fixture of the wrong-implementation check (controlsReal), wraps its fake acceptance runner so
// that the report of every run is kept, calls wrongImplementationCheck as checkSubmission does
// and records what it showed as JSON in KSCMW_OUT for the acceptance steps to judge. The harness
// judges nothing itself.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// kscmwExecuted are the names of the scenarios a Cucumber Messages report shows as started, in order.
func kscmwExecuted(path string) []string {
	out := []string{}
	file, err := os.Open(path)
	if err != nil {
		return out
	}
	defer file.Close()
	names, pickleOf := map[string]string{}, map[string]string{}
	str := func(m map[string]any, k string) string {
		v, _ := m[k].(string)
		return v
	}
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var m map[string]any
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		if p, ok := m["pickle"].(map[string]any); ok {
			names[str(p, "id")] = str(p, "name")
		}
		if tc, ok := m["testCase"].(map[string]any); ok {
			pickleOf[str(tc, "id")] = str(tc, "pickleId")
		}
		if st, ok := m["testCaseStarted"].(map[string]any); ok {
			out = append(out, names[pickleOf[str(st, "testCaseId")]])
		}
	}
	return out
}

var kscmwLocation = regexp.MustCompile("^(.*):(\\d+):\\d+$")

// kscmwEvents are the wrong_implementation_check events recorded so far.
func kscmwEvents(f *shpFx) []map[string]any {
	out := []map[string]any{}
	file, err := os.Open(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug, "events.ndjson"))
	if err != nil {
		return out
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var e map[string]any
		if json.Unmarshal(sc.Bytes(), &e) == nil && e["event"] == "wrong_implementation_check" {
			out = append(out, e)
		}
	}
	return out
}

// kscmwObserve runs the wrong-implementation check of one candidate and observes it.
func kscmwObserve(t *testing.T, c controlsCandidate) map[string]any {
	f, sh := controlsReal(t, c)
	sh.cfg.Acceptance.Run = fmt.Sprintf("sh accept.sh; s=$?; n=$(echo wrap >> '%[1]s/wrap'; wc -l < '%[1]s/wrap' | tr -d ' '); cp \"$KOGEN_ACCEPTANCE_REPORT\" '%[1]s/report-'$n'.ndjson' 2>/dev/null; exit $s", f.Aux)
	confirmed, err := sh.wrongImplementationCheck(context.Background(), "h-kscmw")
	obs := map[string]any{"error": "", "refusal": append([]string{}, confirmed.wrong...),
		"headline_caps": strings.Contains(confirmed.wrongHeadline, "at most 8 runs and 300 seconds per candidate"),
		"runs":          strings.Count(f.Stderr.String(), "kogen: wrong-implementation check: run ")}
	if err != nil {
		obs["error"] = err.Error()
	}
	executed := [][]string{}
	for n := 1; ; n++ {
		p := filepath.Join(f.Aux, fmt.Sprintf("report-%d.ndjson", n))
		if _, err := os.Stat(p); err != nil {
			break
		}
		executed = append(executed, kscmwExecuted(p))
	}
	obs["executed"] = executed
	events := kscmwEvents(f)
	obs["events"] = len(events)
	obs["event_result"], obs["event_runs"], obs["controls"] = "", 0, []map[string]any{}
	if len(events) == 1 {
		obs["event_result"], obs["event_runs"] = events[0]["result"], events[0]["runs"]
		controls := []map[string]any{}
		list, _ := events[0]["controls"].([]any)
		for _, raw := range list {
			m, _ := raw.(map[string]any)
			row := map[string]any{}
			for _, k := range []string{"name", "scenario", "step", "failed_step", "assertion"} {
				row[k] = m[k]
			}
			loc, _ := m["location"].(string)
			row["location_file"], row["location_line"] = loc, 0
			if mm := kscmwLocation.FindStringSubmatch(loc); mm != nil {
				line, _ := strconv.Atoi(mm[2])
				row["location_file"], row["location_line"] = mm[1], line
			}
			controls = append(controls, row)
		}
		obs["controls"] = controls
	}
	return obs
}

func TestKShapeControlsMinimalWitnessesAcceptance(t *testing.T) {
	out := os.Getenv("KSCMW_OUT")
	if out == "" {
		t.Fatal("KSCMW_OUT must be set")
	}
	const exact, subStep = "Then is-even: add of 2 and 3 is 5", "Then is-even: sub of 5 and 3 is 2"
	const loose = "Then is-even: add of 2 and 3 gives an answer"
	correct := map[string]string{"calc.js": controlsCalc("add", "sub")}
	obs := map[string]any{}

	obs["clean"] = kscmwObserve(t, controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef,
		correct: correct, wrong: controlsWrongs(exact, controlsCalc("sub"))})

	obs["outline"] = kscmwObserve(t, controlsCandidate{
		feature: "Feature: Calc\n  Scenario: Add keeps working\n    " + exact + "\n  Scenario Outline: Sub keeps working\n    Then is-even: sub of <a> and <b> is <c>\n    Examples:\n      | a | b | c |\n      | 5 | 3 | 2 |\n      | 9 | 4 | 5 |\n",
		steps:   controlsExactDef, correct: correct,
		wrong: []controlsWrong{
			{"add removed", "Add keeps working", exact, map[string]string{"calc.js": controlsCalc("sub")}},
			{"sub removed", "Sub keeps working", "Then is-even: sub of <a> and <b> is <c>", map[string]string{"calc.js": controlsCalc("add")}}}})

	obs["target-scenario-only"] = kscmwObserve(t, controlsCandidate{feature: controlsCalcFeature(exact, "And is-even: sub of 5 and 3 is 2"),
		steps: controlsExactDef, correct: correct,
		wrong: []controlsWrong{
			{"sub dropped from add", "Add keeps working", "And is-even: sub of 5 and 3 is 2", map[string]string{"calc.js": controlsCalc("add")}},
			{"sub removed", "Sub keeps working", subStep, map[string]string{"calc.js": controlsCalc("add")}}}})

	obs["passes"] = kscmwObserve(t, controlsCandidate{feature: controlsCalcFeature(loose), steps: controlsExactDef + controlsLooseDef,
		correct: correct, wrong: controlsWrongs(loose, controlsCalc("sub"))})

	obs["earlier"] = kscmwObserve(t, controlsCandidate{
		feature: controlsCalcFeature("Then is-even: calc takes 3 arguments", "And is-even: add of 2 and 3 gives an answer"),
		steps:   controlsExactDef + controlsLooseDef + controlsArityDef, correct: correct,
		wrong: controlsWrongs("And is-even: add of 2 and 3 gives an answer", "exports.calc = (op, ...args) => {\n  const [a, b] = args;\n  if (op === 'sub') return a - b;\n};\n")})

	many := controlsCandidate{feature: controlsCalcFeature(exact), steps: controlsExactDef, correct: correct}
	for i := 0; i < 8; i++ {
		many.wrong = append(many.wrong, controlsWrong{fmt.Sprintf("w%d", i), "Add keeps working", exact, map[string]string{"calc.js": controlsCalc("sub")}})
	}
	obs["run-cap"] = kscmwObserve(t, many)

	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
