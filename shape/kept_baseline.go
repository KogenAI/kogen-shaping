package shape

import (
	"fmt"
	"slices"
	"strings"

	"github.com/KogenAI/kogen-verification/verify"
)

// keptStepState is what one unchanged-tree observation shows of a step that a kept
// property names as asserting the kept behaviour itself.
type keptStepState struct {
	scenario string
	line     int
	red      bool   // the step ran and is the failed step
	blocked  bool   // an earlier step of the scenario failed, so the step never ran
	detail   string // the failure detail of the scenario
}

// keptStepStates observes the named kept steps in one readiness result.
func keptStepStates(ready *verify.Ready, kept map[string][]int) map[string]keptStepState {
	out := map[string]keptStepState{}
	if ready == nil {
		return out
	}
	for _, sc := range ready.Scenarios {
		lines := kept[normalizeWhitespace(sc.Name)]
		if len(lines) == 0 || sc.Status != "failed" || sc.Detail == nil {
			continue
		}
		failedAt := 0
		for _, st := range sc.Steps {
			if !slices.ContainsFunc(sc.UnexercisedSteps, func(u verify.StepRecord) bool { return u.Line == st.Line }) && st.Line > failedAt {
				failedAt = st.Line
			}
		}
		for _, l := range lines {
			s := keptStepState{scenario: normalizeWhitespace(sc.Name), line: l, detail: normalizeWhitespace(*sc.Detail)}
			switch {
			case slices.ContainsFunc(sc.UnexercisedSteps, func(u verify.StepRecord) bool { return u.Line == l }):
				s.blocked = true
			case l == failedAt && (strings.Contains(*sc.Detail, "AssertionError") || strings.Contains(strings.ToLower(*sc.Detail), "assertion failed")):
				s.red = true
			default:
				continue
			}
			out[fmt.Sprintf("%s:%d", s.scenario, l)] = s
		}
	}
	return out
}

// keptBaselineRefusals are the refusals of kept steps that are red, or never run because an
// earlier step failed, in both unchanged-tree observations. They state a claim about current
// behaviour that the run itself disproves or leaves unobserved, so no reviewer verdict clears them.
func keptBaselineRefusals(first, second map[string]keptStepState, props []propertyReport) []string {
	var out []string
	// Blocked steps share one refusal line per property, scenario and failure text.
	type blockedGroup struct {
		name, scenario, detail string
		lines                  []int
	}
	groups := map[string]*blockedGroup{}
	var order []string
	for key, a := range first {
		b, ok := second[key]
		if !ok || a.red != b.red {
			continue
		}
		name := ""
		for _, p := range props {
			if p.kind == "kept" && slices.Contains(p.keptSteps, a.line) {
				name = normalizeWhitespace(p.property)
				break
			}
		}
		if a.red {
			out = append(out, fmt.Sprintf("- %s (kept) | scenario %q line %d asserts unchanged behaviour but is red on the unchanged tree in both observations: %s. Fix the expected value to what the unchanged code returns, or mark the behaviour as changed in the Intent. Resubmitting unchanged is refused again.", name, a.scenario, a.line, b.detail))
			continue
		}
		gk := fmt.Sprintf("%q\x00%q\x00%q", name, a.scenario, b.detail)
		g := groups[gk]
		if g == nil {
			g = &blockedGroup{name: name, scenario: a.scenario, detail: b.detail}
			groups[gk] = g
			order = append(order, gk)
		}
		g.lines = append(g.lines, a.line)
	}
	for _, gk := range order {
		g := groups[gk]
		slices.Sort(g.lines)
		out = append(out, fmt.Sprintf("- %s (kept) | scenario %q %s is meant to assert unchanged behaviour but never runs on the unchanged tree: an earlier step of the scenario fails on new behaviour (%s). Assert the unchanged behaviour in a step that runs on the baseline, such as its own scenario. Resubmitting unchanged is refused again.", g.name, g.scenario, keptLinesPhrase(g.lines), g.detail))
	}
	slices.Sort(out)
	return out
}

// keptLinesPhrase names one step as "line N" and several as "lines A, B and C", ascending.
func keptLinesPhrase(lines []int) string {
	if len(lines) == 1 {
		return fmt.Sprintf("line %d", lines[0])
	}
	nums := make([]string, len(lines))
	for i, l := range lines {
		nums[i] = fmt.Sprint(l)
	}
	return "lines " + strings.Join(nums[:len(nums)-1], ", ") + " and " + nums[len(nums)-1]
}
