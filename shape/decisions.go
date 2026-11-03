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
