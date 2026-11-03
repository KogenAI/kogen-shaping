package shape

import (
	"fmt"
	"slices"
	"strings"
)

// The property check: the contract review also identifies the properties the request and the
// caller's answers ask for. Kogen alone decides which reported properties count and what is wrong
// with how acceptance covers them; the confirmation verifies the problems it found.

const (
	propertyMissing   = "missing"
	propertyTestCount = "test_count"
	propertyHumanText = "needs human evaluation"
)

// propertyIssue is a scenario of a property whose evidence does not establish it, as the review
// reports it.
type propertyIssue struct {
	problem, scenario, quote, why string
}

// propertyReport is one property the review reports.
type propertyReport struct {
	property, request, kind string
	observable              bool
	scenarios               []string
	decision                string
	issues                  []propertyIssue
}

// propertyProblem is one problem Kogen found with how acceptance covers a counted property.
type propertyProblem struct {
	prop         int    // index of the property in the review's order
	reason       string // uncovered, passes_unchanged, missing, test_count, claimed or unrecorded
	scenario     string // the scenario the problem is about, normalized; empty when none
	text         string // the reason as the refusal states it
	verdict      string // "", confirmed or refuted
	confirmation *string
}

func propertySchema() map[string]any {
	s := map[string]any{"type": "string"}
	issue := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"problem":  map[string]any{"type": "string", "enum": []any{propertyMissing, propertyTestCount}},
			"scenario": s, "quote": s, "why": s,
		},
		"required":             []any{"problem", "scenario", "quote", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"property":   s,
			"request":    s,
			"kind":       map[string]any{"type": "string", "enum": []any{"new", "kept"}},
			"observable": map[string]any{"type": "boolean"},
			"scenarios":  map[string]any{"type": "array", "items": s},
			"decision":   s,
			"issues":     map[string]any{"type": "array", "items": issue},
		},
		"required":             []any{"property", "request", "kind", "observable", "scenarios", "decision", "issues"},
		"additionalProperties": false,
	}
}

// parsePropertyReports is the properties of a valid "properties" field, or why it is not one.
func parsePropertyReports(v any) ([]propertyReport, string) {
	list, ok := v.([]any)
	if !ok {
		return nil, "properties must be a list"
	}
	out := make([]propertyReport, 0, len(list))
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok || !exactKeys(m, "property", "request", "kind", "observable", "scenarios", "decision", "issues") {
			return nil, "a property must be an object with exactly the fields property, request, kind, observable, scenarios, decision, issues"
		}
		var p propertyReport
		var ok1, ok2, ok3, ok4, ok5 bool
		p.property, ok1 = m["property"].(string)
		p.request, ok2 = m["request"].(string)
		p.kind, ok3 = m["kind"].(string)
		p.observable, ok4 = m["observable"].(bool)
		p.decision, ok5 = m["decision"].(string)
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || (p.kind != "new" && p.kind != "kept") {
			return nil, "a property has a field of the wrong type or value"
		}
		scenarios, ok := m["scenarios"].([]any)
		if !ok {
			return nil, "a property has a field of the wrong type or value"
		}
		for _, raw := range scenarios {
			name, ok := raw.(string)
			if !ok {
				return nil, "a property has a field of the wrong type or value"
			}
			p.scenarios = append(p.scenarios, name)
		}
		issues, ok := m["issues"].([]any)
		if !ok {
			return nil, "a property has a field of the wrong type or value"
		}
		for _, raw := range issues {
			im, ok := raw.(map[string]any)
			if !ok || !exactKeys(im, "problem", "scenario", "quote", "why") {
				return nil, "an issue must be an object with exactly the fields problem, scenario, quote, why"
			}
			var is propertyIssue
			var ok1, ok2, ok3, ok4 bool
			is.problem, ok1 = im["problem"].(string)
			is.scenario, ok2 = im["scenario"].(string)
			is.quote, ok3 = im["quote"].(string)
			is.why, ok4 = im["why"].(string)
			if !ok1 || !ok2 || !ok3 || !ok4 || (is.problem != propertyMissing && is.problem != propertyTestCount) {
				return nil, "an issue has a field of the wrong type or value"
			}
			p.issues = append(p.issues, is)
		}
		out = append(out, p)
	}
	return out, ""
}

// counts reports whether a reported property counts: it has a name and a request text that the
// Request or the Questions and answers contain.
func propertyCounts(p propertyReport, request, questions string) bool {
	if normalizeWhitespace(p.property) == "" || normalizeWhitespace(p.request) == "" {
		return false
	}
	return containsNormalized(request, p.request) || containsNormalized(questions, p.request)
}

// declaredScenarios are the listed scenarios that the candidate declares, in listed order,
// without repeats, normalized.
func (e contractEvidence) declaredScenarios(listed []string) []string {
	var out []string
	for _, name := range listed {
		n := normalizeWhitespace(name)
		if slices.Contains(e.scenarios, n) && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// decisionRecorded reports whether a line of the Decisions section of the candidate intent.md
// contains the decision and says it needs human evaluation.
func (e contractEvidence) decisionRecorded(decision string) bool {
	if normalizeWhitespace(decision) == "" {
		return false
	}
	in := false
	for _, line := range strings.Split(e.intent, "\n") {
		trimmed := strings.TrimRight(line, " \t\r")
		switch {
		case !in:
			in = trimmed == "## Decisions"
		case strings.HasPrefix(line, "## "):
			return false
		case containsNormalized(line, decision) && containsNormalized(strings.ToLower(line), propertyHumanText):
			return true
		}
	}
	return false
}

// propertyProblems is what Kogen finds wrong with the counted properties, ordered by property in
// the review's order and within a property as the rules list them.
func (e contractEvidence) propertyProblems(props []propertyReport, request, questions string) []propertyProblem {
	var out []propertyProblem
	for i, p := range props {
		if !propertyCounts(p, request, questions) {
			continue
		}
		add := func(reason, scenario, text string) {
			out = append(out, propertyProblem{prop: i, reason: reason, scenario: scenario, text: text})
		}
		declared := e.declaredScenarios(p.scenarios)
		if !p.observable {
			for _, s := range declared {
				add("claimed", s, fmt.Sprintf("no automatic step can observe it, but scenario \"%s\" claims it holds", s))
			}
			if !e.decisionRecorded(p.decision) {
				add("unrecorded", "", "no automatic step can observe it, and no Decision records it as needing human evaluation")
			}
			continue
		}
		failed := func(s string) bool { return len(e.failures[s]) > 0 }
		if len(declared) == 0 {
			add("uncovered", "", "no scenario exercises it")
			continue
		}
		if p.kind == "new" && !slices.ContainsFunc(declared, failed) {
			add("passes_unchanged", "", "every scenario that exercises it passes on the unchanged tree, so none is red at an assertion about it")
		}
		for _, is := range p.issues {
			s := normalizeWhitespace(is.scenario)
			if is.problem != propertyMissing || p.kind != "new" || !slices.Contains(declared, s) || !failed(s) || normalizeWhitespace(is.quote) == "" {
				continue
			}
			if slices.ContainsFunc(e.failures[s], func(detail string) bool { return containsNormalized(detail, is.quote) }) {
				add(propertyMissing, s, fmt.Sprintf("scenario \"%s\" is red on the unchanged tree only because something it needs is missing: %s", s, normalizeWhitespace(is.quote)))
			}
		}
		for _, is := range p.issues {
			s := normalizeWhitespace(is.scenario)
			if is.problem != propertyTestCount || !slices.Contains(declared, s) || normalizeWhitespace(is.quote) == "" {
				continue
			}
			found := false
			for path, text := range e.expectations {
				if path != e.featurePath && containsNormalized(text, is.quote) {
					found = true
					break
				}
			}
			if found {
				add(propertyTestCount, s, fmt.Sprintf("scenario \"%s\" only checks that tests the Build writes exist or pass in some number: %s", s, normalizeWhitespace(is.quote)))
			}
		}
	}
	return out
}

// propertyConfirmLine is the confirmation input's line for problem number n (from 1).
func propertyConfirmLine(n int, p propertyReport, pr propertyProblem) string {
	return fmt.Sprintf("%d. property: %s (%s) | request: %s | problem: %s",
		n, normalizeWhitespace(p.property), p.kind, normalizeWhitespace(p.request), pr.text)
}

// propertyItemLine is the feedback line of a confirmed property problem.
func propertyItemLine(p propertyReport, pr propertyProblem, confirmation string) string {
	return fmt.Sprintf("- %s (%s) | request: %s | problem: %s | confirmed: %s",
		normalizeWhitespace(p.property), p.kind, normalizeWhitespace(p.request), pr.text, normalizeWhitespace(confirmation))
}

// propertyHeadline is the first line of the property part of a refusal of n confirmed problems.
func propertyHeadline(n int) string {
	return fmt.Sprintf("Property check: %d problem(s) with how acceptance covers the properties the request asks for; nothing was published. "+
		"Every property the request or the answers ask for that a step can observe needs a scenario that exercises it, and a new one's scenario must be red on the unchanged tree at an assertion about that property, not only because a file, module, function, command or test is missing: give the missing piece a do-nothing form first, then fail on its behaviour. "+
		"A step whose only evidence is that tests the Build writes exist or pass in some number is not evidence for a property unless it asserts the property itself. "+
		"A property no automatic step can observe needs a Decisions line saying it needs human evaluation, with its source, and no scenario may claim it holds, also not through a figure, log line or document sentence. "+
		"Resubmitting this candidate unchanged is refused again.", n)
}

// applyPropertyVerdicts gives each problem its counted verdict: the first verdict for its number,
// numbered from first (the problems follow the evidenced contradictions and environment
// findings). A problem without a counted verdict is refuted. It returns the item lines of the
// confirmed problems.
func applyPropertyVerdicts(props []propertyReport, problems []propertyProblem, first int, verdicts []contractVerdict) []string {
	counted := map[int]contractVerdict{}
	for _, v := range verdicts {
		if _, seen := counted[v.finding]; !seen {
			counted[v.finding] = v
		}
	}
	var lines []string
	for i := range problems {
		pr := &problems[i]
		v, ok := counted[first+i]
		if !ok {
			pr.verdict = "refuted"
			continue
		}
		pr.verdict = v.verdict
		why := v.why
		pr.confirmation = &why
		if v.verdict == "confirmed" {
			lines = append(lines, propertyItemLine(props[pr.prop], *pr, v.why))
		}
	}
	return lines
}

// propertyEvent is the property_check event of a review that reported properties.
func propertyEvent(hash string, props []propertyReport, counted []bool, problems []propertyProblem) map[string]any {
	list := make([]any, len(props))
	for i, p := range props {
		scenarios := make([]any, len(p.scenarios))
		for j, s := range p.scenarios {
			scenarios[j] = s
		}
		list[i] = map[string]any{"property": p.property, "request": p.request, "kind": p.kind,
			"observable": p.observable, "scenarios": scenarios, "evidenced": counted[i]}
	}
	probs := make([]any, len(problems))
	for i, pr := range problems {
		var verdict, confirmation any
		if pr.verdict != "" {
			verdict = pr.verdict
		}
		if pr.confirmation != nil {
			confirmation = *pr.confirmation
		}
		probs[i] = map[string]any{"property": props[pr.prop].property, "reason": pr.reason, "scenario": pr.scenario,
			"verdict": verdict, "confirmation": confirmation}
	}
	return map[string]any{"event": "property_check", "package": hash, "properties": list, "problems": probs}
}
