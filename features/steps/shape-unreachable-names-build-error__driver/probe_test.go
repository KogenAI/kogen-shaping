//go:build ignore

package shape

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

func TestUnreachableNamesAcceptance(t *testing.T) {
	message := os.Getenv("SUN_MESSAGE")
	text := "Feature: compile\nScenario: compile\nThen compile\n"
	scenarios, err := project.ParseGherkin(text)
	if err != nil {
		t.Fatal(err)
	}
	sc := scenarios[0]
	feature := strings.Split(text, "\n")
	c := &controlsCase{steps: []controlsStep{{line: 3, status: "FAILED", kind: "Error", message: message}}}
	run := controlsRunResult{complete: true, cases: map[int]*controlsCase{sc.Line: c}}
	lines := controlsUnreachable([]controlsProp{{report: propertyReport{property: "compile", kind: "new", request: "compile"}, scenarios: []int{0}}}, scenarios, feature, run)
	_, wrong := controlsJudge(controlsImpl{name: "broken", step: "Then compile"}, sc, c, []int{3}, feature, run, nil)
	ev := controlsEvidenceOf(controlsImpl{name: "broken", scenario: "compile", step: "Then compile"}, scenarios, feature, run, nil)
	b, err := json.Marshal(map[string]any{"unreachable": lines, "wrong": wrong, "assertion": ev.Assertion})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("SUN_OUT"), b, 0600); err != nil {
		t.Fatal(err)
	}
}
