package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
)

// The contract audit: two more kinds of finding over the payload of the one contract review call.
// invented_contract refuses a literal the acceptance drives or asserts that no source states, and
// uncovered_failure_mode refuses a deferral of controls.json that leaves a guarantee of the request
// without a wrong implementation to reject. Kogen alone checks the evidence; the audit makes no
// model request of its own.

const (
	auditInvented  = "invented_contract"
	auditUncovered = "uncovered_failure_mode"

	directionTooMuch     = "too_much"
	directionTooLittle   = "too_little"
	directionInterrupted = "interrupted"
)

// auditKeys are the optional top-level fields of a review payload that carry the audit.
var auditKeys = []string{"invented", "guarantees", "failure_modes"}

// auditInventedReport is one literal the review reports as invented.
type auditInventedReport struct{ scenario, literal, why string }

// auditGuarantee is one observable property of the request with a guarantee word.
type auditGuarantee struct {
	property, request string
	needsInterrupted  bool
}

// auditFailureMode is one wrong implementation the review reports for a guarantee.
type auditFailureMode struct{ property, direction, wrong, scenario, assertion string }

// auditReport is the audit part of a review payload; absent keys are empty lists.
type auditReport struct {
	invented   []auditInventedReport
	guarantees []auditGuarantee
	modes      []auditFailureMode
}

func auditSchemas() (invented, guarantee, mode map[string]any) {
	s := map[string]any{"type": "string"}
	object := func(props map[string]any, required ...string) map[string]any {
		req := make([]any, len(required))
		for i, r := range required {
			req[i] = r
		}
		return map[string]any{"type": "object", "properties": props, "required": req, "additionalProperties": false}
	}
	invented = object(map[string]any{"scenario": s, "literal": s, "why": s}, "scenario", "literal", "why")
	guarantee = object(map[string]any{"property": s, "request": s, "needs_interrupted": map[string]any{"type": "boolean"}}, "property", "request", "needs_interrupted")
	mode = object(map[string]any{
		"property":             s,
		"direction":            map[string]any{"type": "string", "enum": []any{directionTooMuch, directionTooLittle, directionInterrupted}},
		"wrong_implementation": s, "scenario": s, "assertion": s,
	}, "property", "direction", "wrong_implementation", "scenario", "assertion")
	return invented, guarantee, mode
}

// auditPayload is the review payload without the audit fields, and the audit fields parsed.
func auditPayload(args map[string]any) (map[string]any, auditReport, string) {
	rest := make(map[string]any, len(args))
	for k, v := range args {
		if !slices.Contains(auditKeys, k) {
			rest[k] = v
		}
	}
	var rep auditReport
	items := func(key string) ([]map[string]any, string) {
		v, has := args[key]
		if !has {
			return nil, ""
		}
		list, ok := v.([]any)
		if !ok {
			return nil, key + " must be a list"
		}
		out := make([]map[string]any, 0, len(list))
		for _, raw := range list {
			m, ok := raw.(map[string]any)
			if !ok {
				return nil, "an item of " + key + " must be an object"
			}
			out = append(out, m)
		}
		return out, ""
	}
	str := func(m map[string]any, key string) (string, bool) {
		s, ok := m[key].(string)
		return s, ok
	}
	list, why := items("invented")
	if why != "" {
		return nil, rep, why
	}
	for _, m := range list {
		scenario, ok1 := str(m, "scenario")
		literal, ok2 := str(m, "literal")
		why, ok3 := str(m, "why")
		if !exactKeys(m, "scenario", "literal", "why") || !ok1 || !ok2 || !ok3 {
			return nil, rep, "an invented item must be an object with exactly the string fields scenario, literal, why"
		}
		rep.invented = append(rep.invented, auditInventedReport{scenario, literal, why})
	}
	list, why = items("guarantees")
	if why != "" {
		return nil, rep, why
	}
	for _, m := range list {
		property, ok1 := str(m, "property")
		request, ok2 := str(m, "request")
		needs, ok3 := m["needs_interrupted"].(bool)
		if !exactKeys(m, "property", "request", "needs_interrupted") || !ok1 || !ok2 || !ok3 {
			return nil, rep, "a guarantee must be an object with exactly the fields property, request, needs_interrupted"
		}
		rep.guarantees = append(rep.guarantees, auditGuarantee{property, request, needs})
	}
	list, why = items("failure_modes")
	if why != "" {
		return nil, rep, why
	}
	for _, m := range list {
		property, ok1 := str(m, "property")
		direction, ok2 := str(m, "direction")
		wrong, ok3 := str(m, "wrong_implementation")
		scenario, ok4 := str(m, "scenario")
		assertion, ok5 := str(m, "assertion")
		if !exactKeys(m, "property", "direction", "wrong_implementation", "scenario", "assertion") || !ok1 || !ok2 || !ok3 || !ok4 || !ok5 ||
			!slices.Contains([]string{directionTooMuch, directionTooLittle, directionInterrupted}, direction) {
			return nil, rep, "a failure mode must be an object with exactly the fields property, direction, wrong_implementation, scenario, assertion, and a direction of too_much, too_little or interrupted"
		}
		rep.modes = append(rep.modes, auditFailureMode{property, direction, wrong, scenario, assertion})
	}
	return rest, rep, ""
}

// auditInventedFix is the remedy a refused invented literal is given.
func auditInventedFix(literal string) string {
	return fmt.Sprintf("- fix for invented literal \"%s\": find it at run time (for example find the form or link by its visible text), assert only what the request states, or move it to a Decision and stop asserting it", auditOneLine(literal))
}

func auditOneLine(s string) string { return normalizeWhitespace(s) }

// auditTrackedHas reports whether the literal occurs in a file tracked at HEAD that is not a
// file of the candidate package. A literal of several lines is not searched.
func (sh *shaping) auditTrackedHas(literal string, skip func(path string) bool) bool {
	if literal == "" || strings.ContainsAny(literal, "\n\r\x00") {
		return false
	}
	res, err := sh.repo.Run("grep", "-F", "-l", "-z", "--no-color", "-e", literal, "HEAD", "--")
	if err != nil || res.Status != 0 {
		return false
	}
	for _, hit := range strings.Split(res.Stdout, "\x00") {
		path := strings.TrimPrefix(hit, "HEAD:")
		if path != "" && !skip(path) {
			return true
		}
	}
	return false
}

// inventedFinding is one invented literal Kogen keeps, with the scenario and literal normalized as the
// run records them, and the refusal line and fix line it is refused with.
type inventedFinding struct{ scenario, literal, line, fix string }

// inventedDroppedFile is the per-run record of the literals the run dropped, replaced by each ready
// run like unresolvedFile; the approval card lists them.
const inventedDroppedFile = "invented_dropped.json"

// inventedHeading is the heading of the card section that lists the dropped literals.
const inventedHeading = "Literals chosen by the Shaper"

// auditInventedKept are the invented literals Kogen keeps from the review's reports: the request,
// the answers and the tracked files do not state them and the candidate drives or asserts them.
func (sh *shaping) auditInventedKept(ev contractEvidence, reports []auditInventedReport, questions string) (kept []inventedFinding) {
	skip := func(path string) bool {
		if path == ev.intentPath || path == ev.featurePath || path == project.ModelPath(sh.slug) {
			return true
		}
		if _, ok := ev.expectations[path]; ok {
			return true
		}
		_, proposed := sh.proposed[path]
		return proposed
	}
	var seen []string
	for _, r := range reports {
		literal := strings.TrimSpace(r.literal)
		if normalizeWhitespace(literal) == "" || strings.TrimSpace(r.why) == "" {
			continue
		}
		if key := normalizeWhitespace(r.scenario) + "\x00" + literal; slices.Contains(seen, key) {
			continue
		} else {
			seen = append(seen, key)
		}
		if containsNormalized(ev.request, literal) || containsNormalized(questions, literal) {
			continue
		}
		driven := false
		if normalizeWhitespace(r.scenario) == "" {
			// A sentence of intent.md that fixes a mechanism the request does not require.
			driven = containsNormalized(ev.intent, literal)
		} else {
			for _, text := range ev.expectations {
				if strings.Contains(text, literal) {
					driven = true
				}
			}
		}
		if !driven || sh.auditTrackedHas(literal, skip) {
			continue
		}
		where := "scenario: " + normalizeWhitespace(r.scenario)
		if normalizeWhitespace(r.scenario) == "" {
			where = "scenario: (none, a sentence of intent.md)"
		}
		kept = append(kept, inventedFinding{
			scenario: normalizeWhitespace(r.scenario), literal: auditOneLine(literal),
			line: fmt.Sprintf("- %s | %s | literal: \"%s\" | why: %s | no source states it: not the request, the answers or a tracked file", auditInvented, where, auditOneLine(literal), normalizeWhitespace(r.why)),
			fix:  auditInventedFix(literal),
		})
	}
	return kept
}

// auditInventedLines are the refusal lines and fix lines of the invented literals of a review. The
// first review of a Shape run that keeps any refuses them and records each kept literal and
// scenario and that its one rewrite was offered; every later review of the run drops them: one
// invented_literal_dropped event each, no refusal line.
func (sh *shaping) auditInventedLines(ev contractEvidence, reports []auditInventedReport, questions string) (lines, fixes []string) {
	kept := sh.auditInventedKept(ev, reports, questions)
	if len(kept) == 0 {
		return nil, nil
	}
	if !sh.dropInvented || !sh.inventedOffered {
		for _, k := range kept {
			lines, fixes = append(lines, k.line), append(fixes, k.fix)
		}
		if sh.dropInvented {
			sh.inventedOffered = true
			for _, k := range kept {
				sh.inventedKept = append(sh.inventedKept, inventedRecord{Scenario: k.scenario, Literal: k.literal})
			}
		}
		return lines, fixes
	}
	for _, k := range kept {
		rec := inventedRecord{Scenario: k.scenario, Literal: k.literal, Submission: sh.submissions}
		sh.inventedDropped = append(sh.inventedDropped, rec)
		sh.ev.Record(map[string]any{"event": "invented_literal_dropped", "literal": rec.Literal, "scenario": rec.Scenario, "submission": rec.Submission})
	}
	return nil, nil
}

// inventedRecord is a literal of a kept or dropped invented_contract finding.
type inventedRecord struct {
	Scenario   string `json:"scenario"`
	Literal    string `json:"literal"`
	Submission int    `json:"submission,omitempty"`
}

// inventedFinal are the dropped literals the ready package still drives or asserts: a literal the
// Shaper rewrote away in a later submission is not chosen by the Shaper any more.
func (sh *shaping) inventedFinal() []inventedRecord {
	if len(sh.inventedDropped) == 0 || sh.ready == nil || sh.ready.Intent == nil {
		return nil
	}
	intent := sh.ready.Intent.Text
	texts := []string{}
	for _, p := range append(sh.ready.Intent.StepFiles(), project.FeaturePath(sh.slug)) {
		if b, err := os.ReadFile(filepath.Join(sh.root, p)); err == nil {
			texts = append(texts, string(b))
		}
	}
	var out []inventedRecord
	for _, r := range sh.inventedDropped {
		driven := false
		if r.Scenario == "" {
			driven = containsNormalized(intent, r.Literal)
		} else {
			for _, t := range texts {
				driven = driven || containsNormalized(t, r.Literal)
			}
		}
		if driven {
			out = append(out, r)
		}
	}
	return out
}

// writeInventedDropped replaces the run's record of dropped literals; the approval card reads it.
func (sh *shaping) writeInventedDropped() error {
	list := sh.inventedFinal()
	if list == nil {
		list = []inventedRecord{}
	}
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sh.ev.Dir, inventedDroppedFile), b, 0o600)
}

// inventedDroppedLines are the card lines of the literals the run that produced the current Intent
// dropped; nil when it dropped none.
func inventedDroppedLines(root, slug string) []string {
	b, err := os.ReadFile(filepath.Join(root, ".kogen/local/shaping", slug, inventedDroppedFile))
	if err != nil {
		return nil
	}
	var list []inventedRecord
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return inventedCardLines(list)
}

// inventedCardLines are the lines under the card heading, one per distinct literal and scenario.
func inventedCardLines(list []inventedRecord) []string {
	var out, seen []string
	for _, r := range list {
		key := r.Scenario + "\x00" + r.Literal
		if slices.Contains(seen, key) {
			continue
		}
		seen = append(seen, key)
		where := "scenario: " + r.Scenario
		if r.Scenario == "" {
			where = "scenario: (none, a sentence of intent.md)"
		}
		out = append(out, fmt.Sprintf("  - %s | literal: \"%s\"", where, r.Literal))
	}
	if len(out) == 0 {
		return nil
	}
	return append([]string{"  The Request does not state them; check that each is acceptable:"}, out...)
}

// auditDeferred reports whether the candidate's controls.json is a deferral.
func (sh *shaping) auditDeferred() bool {
	raw, err := os.ReadFile(filepath.Join(sh.root, project.IntentDir(sh.slug), "controls.json"))
	if err != nil {
		return false
	}
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		return false
	}
	_, problems, isDeferred := controlsDeferral(top)
	return isDeferred && len(problems) == 0
}

// auditThenSteps are the texts of the Then steps of a scenario, with the And and But steps that
// follow a Then, in the scenario's own steps (Background steps included).
func auditThenSteps(sc project.Scenario) []string {
	var out []string
	then := false
	for _, st := range sc.Steps {
		switch strings.TrimSpace(st.Keyword) {
		case "Then":
			then = true
		case "And", "But", "*":
		default:
			then = false
		}
		if then {
			out = append(out, st.Text)
		}
	}
	return out
}

// auditFailureLines are the refusal lines and fix lines of the guarantees of the request that the
// failure modes do not cover. A mode counts when its scenario is declared and failed on the unchanged
// tree and its assertion is text of a Then step of that scenario.
func (sh *shaping) auditFailureLines(ev contractEvidence, scenarios []project.Scenario, audit auditReport, questions string) (lines, fixes []string) {
	failed := map[string]bool{}
	for name := range ev.failures {
		failed[name] = true
	}
	for _, sc := range scenarios {
		if sc.Outline != nil && failed[normalizeWhitespace(sc.Name)] {
			failed[normalizeWhitespace(sc.Outline.Name)] = true
		}
	}
	steps := func(name string) ([]string, bool) {
		var out []string
		found := false
		for _, sc := range scenarios {
			if normalizeWhitespace(sc.Name) == name || (sc.Outline != nil && normalizeWhitespace(sc.Outline.Name) == name) {
				found = true
				out = append(out, auditThenSteps(sc)...)
			}
		}
		return out, found
	}
	same := func(a, b string) bool { return strings.EqualFold(normalizeWhitespace(a), normalizeWhitespace(b)) }
	var done []string
	for _, g := range auditGuaranteesCounted(audit.guarantees, ev.request, questions) {
		if slices.ContainsFunc(done, func(p string) bool { return same(p, g.property) }) {
			continue
		}
		done = append(done, g.property)
		needs := g.needsInterrupted
		for _, other := range audit.guarantees {
			if same(other.property, g.property) && other.needsInterrupted {
				needs = true
			}
		}
		directions := []string{directionTooMuch, directionTooLittle}
		if needs {
			directions = append(directions, directionInterrupted)
		}
		property := normalizeWhitespace(g.property)
		for _, direction := range directions {
			var entries []auditFailureMode
			for _, m := range audit.modes {
				if same(m.property, g.property) && m.direction == direction {
					entries = append(entries, m)
				}
			}
			if len(entries) == 0 {
				lines = append(lines, fmt.Sprintf("- %s | property: %s | missing direction: %s | no wrong implementation of this direction is reported", auditUncovered, property, direction))
				fixes = append(fixes, fmt.Sprintf("- fix for property %s: add a scenario that fails on the unchanged tree and whose Then step rejects a wrong implementation that does %s", property, auditDirectionText(direction)))
				continue
			}
			var bad []string
			var badFix []string
			covered := false
			for _, m := range entries {
				name := normalizeWhitespace(m.scenario)
				thens, declared := steps(name)
				var problem string
				switch {
				case normalizeWhitespace(m.wrong) == "":
					problem = "the wrong implementation is not described"
				case !declared:
					problem = "the scenario is not in acceptance.feature"
				case normalizeWhitespace(m.assertion) == "" || !slices.ContainsFunc(thens, func(t string) bool { return containsNormalized(t, m.assertion) }):
					problem = "its assertion is not the text of a Then step of the scenario"
				case !failed[name]:
					problem = "the scenario passes on the unchanged tree, so it is a happy path"
				}
				if problem == "" {
					covered = true
					break
				}
				bad = append(bad, fmt.Sprintf("- %s | property: %s | direction: %s | scenario: %s | assertion: \"%s\" | %s", auditUncovered, property, direction, name, normalizeWhitespace(m.assertion), problem))
				badFix = append(badFix, fmt.Sprintf("- fix for scenario %s: make a Then step of it assert that a wrong implementation doing %s is rejected, in a scenario that fails on the unchanged tree", name, auditDirectionText(direction)))
			}
			if !covered {
				lines = append(lines, bad...)
				fixes = append(fixes, badFix...)
			}
		}
	}
	return lines, fixes
}

func auditDirectionText(direction string) string {
	switch direction {
	case directionTooMuch:
		return "too much (more than the guarantee allows)"
	case directionTooLittle:
		return "too little (less than the guarantee requires)"
	}
	return "it interrupted (an attempt cut off midway)"
}

// auditGuaranteesCounted are the reported guarantees whose request text the Request or the
// Questions and answers contain, ignoring case.
func auditGuaranteesCounted(list []auditGuarantee, request, questions string) []auditGuarantee {
	var out []auditGuarantee
	lower := func(s string) string { return strings.ToLower(normalizeWhitespace(s)) }
	for _, g := range list {
		if normalizeWhitespace(g.property) == "" || normalizeWhitespace(g.request) == "" {
			continue
		}
		if strings.Contains(lower(request), lower(g.request)) || strings.Contains(lower(questions), lower(g.request)) {
			out = append(out, g)
		}
	}
	return out
}

// auditConfirmed is the audit's refusal lines of a review, added after the confirmed contradictions.
func (sh *shaping) auditConfirmed(ev contractEvidence, scenarios []project.Scenario, audit auditReport, questions string) contractConfirmed {
	var out contractConfirmed
	lines, fixes := sh.auditInventedLines(ev, audit.invented, questions)
	out.contradictions, out.fixes = append(out.contradictions, lines...), append(out.fixes, fixes...)
	out.invented = len(lines) > 0
	if sh.auditDeferred() {
		lines, fixes = sh.auditFailureLines(ev, scenarios, audit, questions)
		out.contradictions, out.fixes = append(out.contradictions, lines...), append(out.fixes, fixes...)
	}
	return out
}

// addAudit appends the audit's lines to the contradictions of c.
func (c *contractConfirmed) addAudit(a contractConfirmed) {
	c.invented = c.invented || a.invented
	c.contradictions = append(c.contradictions, a.contradictions...)
	c.fixes = append(c.fixes, a.fixes...)
}
