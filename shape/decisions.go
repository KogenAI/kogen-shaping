package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// decisionLines are the lines inside the `## Decisions` section of an Intent.
func decisionLines(intent string) []string {
	loc := decisionsHeading.FindStringIndex(intent)
	if loc == nil {
		return nil
	}
	rest := intent[loc[1]:]
	if n := nextHeading.FindStringIndex(rest); n != nil {
		rest = rest[:n[0]]
	}
	return strings.Split(rest, "\n")
}

// assumptionLines are the Decisions lines marked as assumptions, without their leading "- ",
// for the approval card; nil without such a line.
func assumptionLines(intent string) []string {
	var out []string
	for _, l := range decisionLines(intent) {
		if strings.Contains(l, "(assumption:") {
			out = append(out, sourceTagRe.ReplaceAllString(strings.TrimPrefix(strings.TrimSpace(l), "- "), ""))
		}
	}
	return out
}

var sourceTagRe = regexp.MustCompile(`\s*\(source: (?:request|answer \d+)\)\s*$`)

// normalizeWhitespace turns every run of space, tab, CR and LF into one space and trims.
func normalizeWhitespace(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r' || r == '\n'
	}), " ")
}

// containsNormalized is substring matching after normalizeWhitespace on both sides.
func containsNormalized(haystack, needle string) bool {
	return strings.Contains(normalizeWhitespace(haystack), normalizeWhitespace(needle))
}

// unresolvedFile is the per-run record of the Shape run that produced the current Intent; each
// ready run replaces it with `[]`. Records of earlier runs can remain in a checkout.
const unresolvedFile = "unresolved.json"

// unresolvedLines are the card lines of the request conflicts recorded by the run that produced
// the current Intent.
func unresolvedLines(root, slug string) []string {
	b, err := os.ReadFile(filepath.Join(root, ".kogen/local/shaping", slug, unresolvedFile))
	if err != nil {
		return nil
	}
	var last []map[string]any
	if json.Unmarshal(b, &last) != nil {
		return nil
	}
	var out []string
	for _, d := range last {
		out = append(out, fmt.Sprintf("  - quote_a: %v | quote_b: %v | exposing_input: %v", d["quote_a"], d["quote_b"], d["exposing_input"]))
	}
	return out
}

// repairCarriedDecisionDraft restores only the exact lines supplied by the gate.
func repairCarriedDecisionDraft(draft string, problems []string) string {
	loc := decisionsHeading.FindStringIndex(draft)
	if loc == nil {
		return draft
	}
	end := len(draft)
	if next := nextHeading.FindStringIndex(draft[loc[1]:]); next != nil {
		end = loc[1] + next[0]
	}
	have := map[string]bool{}
	for _, line := range decisionLines(draft) {
		have[strings.TrimRight(line, "\r")] = true
	}
	var missing []string
	for _, problem := range problems {
		_, line, ok := strings.Cut(problem, " is not in Decisions; keep it as: ")
		if ok && strings.HasPrefix(problem, "- D") && strings.HasPrefix(line, "- ") && !have[line] {
			missing = append(missing, line)
			have[line] = true
		}
	}
	if len(missing) == 0 {
		return draft
	}
	section := draft[:end]
	trimmed := strings.TrimRight(section, "\r\n")
	suffix := section[len(trimmed):]
	if len(trimmed) > 0 {
		trimmed += "\n"
	}
	return trimmed + strings.Join(missing, "\n") + suffix + draft[end:]
}

var decisionRoundRe = regexp.MustCompile(`\(decision: (D\d+), round \d+\)`)

// Round numbers belong to Kogen's ledger. A model may use its submission
// attempt instead; correct only that number when every other byte agrees.
// Changed choices, constraints, evidence and uncertainty still reach the gate.
func (sh *shaping) normalizeDecisionRounds(intent string) string {
	loc := decisionsHeading.FindStringIndex(intent)
	if loc == nil {
		return intent
	}
	end := len(intent)
	if next := nextHeading.FindStringIndex(intent[loc[1]:]); next != nil {
		end = loc[1] + next[0]
	}
	section := intent[loc[1]:end]
	for _, line := range strings.Split(section, "\n") {
		matches := decisionRoundRe.FindAllStringSubmatchIndex(line, -1)
		if len(matches) != 1 {
			continue
		}
		m := matches[0]
		d, ok := sh.ledger.standingByID(line[m[2]:m[3]])
		if !ok {
			continue
		}
		canonical := d.intentLine()
		marker := decisionRoundRe.FindString(canonical)
		corrected := line[:m[0]] + marker + line[m[1]:]
		if corrected == canonical && corrected != line {
			section = strings.Replace(section, line, corrected, 1)
		}
	}
	return intent[:loc[1]] + section + intent[end:]
}
