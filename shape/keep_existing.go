package shape

import (
	"regexp"
	"strings"
	"unicode"
)

// A keep-existing Decision says that existing values or behaviour stay as they are and cites the
// repository files that show them. The checking model's verdict on such a Decision is not
// reliable, so Kogen decides it from the cited HEAD bytes.

var (
	keepWordRe       = regexp.MustCompile(`(?i)\b(keep|keeps|kept|unchanged|preserve|preserves|retain|retains)\b`)
	assumptionMarkRe = regexp.MustCompile(`\(assumption: [^)]*\)`)
	backtickSpanRe   = regexp.MustCompile("`([^`]+)`")
)

// withoutEvidenceMarkers is the line with every (evidence: ...) marker removed, using the
// marker's own end rule.
func withoutEvidenceMarkers(line string) string {
	var b strings.Builder
	for {
		i := strings.Index(line, evidenceOpen)
		if i < 0 {
			b.WriteString(line)
			return b.String()
		}
		b.WriteString(line[:i])
		rest := line[i+len(evidenceOpen):]
		end, quoted := len(rest), false
		for j, r := range rest {
			if r == '`' {
				quoted = !quoted
			} else if r == ')' && !quoted {
				end = j + 1
				break
			}
		}
		line = rest[end:]
	}
}

// keepExistingValues are the backtick-quoted spans and the decimal numbers (digit runs touching
// neither letters nor underscores) of the text.
func keepExistingValues(text string) []string {
	var values []string
	for _, m := range backtickSpanRe.FindAllStringSubmatch(text, -1) {
		values = append(values, m[1])
	}
	rs := []rune(text)
	word := func(i int) bool { return i >= 0 && i < len(rs) && (unicode.IsLetter(rs[i]) || rs[i] == '_') }
	for i := 0; i < len(rs); {
		if rs[i] < '0' || rs[i] > '9' {
			i++
			continue
		}
		j := i
		for j < len(rs) && rs[j] >= '0' && rs[j] <= '9' {
			j++
		}
		if !word(i-1) && !word(j) {
			values = append(values, string(rs[i:j]))
		}
		i = j
	}
	return values
}

// keepExistingClaim is the cited file paths and the values of a keep-existing statement; ok is
// false when the statement is not one (no keep word outside the markers, or no cited path).
func keepExistingClaim(statement string) (paths, values []string, ok bool) {
	for _, citation := range evidenceCitations(statement) {
		for _, raw := range splitCitation(citation) {
			if item := parseEvidenceItem(raw); !item.empty && item.command == "" {
				paths = append(paths, cleanCitedPath(item.path))
			}
		}
	}
	text := strings.TrimPrefix(strings.TrimSpace(statement), "- ")
	text = sourceTagRe.ReplaceAllString(withoutEvidenceMarkers(text), "")
	text = assumptionMarkRe.ReplaceAllString(text, "")
	if len(paths) == 0 || !keepWordRe.MatchString(text) {
		return nil, nil, false
	}
	return paths, keepExistingValues(text), true
}

// keepExisting classifies a statement: keep is whether it is a keep-existing Decision, shown
// whether every cited path is a tracked file at HEAD and every value occurs in their combined
// HEAD bytes. A statement without values is never shown.
func (sh *shaping) keepExisting(statement string) (keep, shown bool) {
	paths, values, ok := keepExistingClaim(statement)
	if !ok {
		return false, false
	}
	if len(values) == 0 {
		return true, false
	}
	tracked, err := sh.headFiles()
	if err != nil {
		return true, false
	}
	var combined strings.Builder
	for _, p := range paths {
		body, ok := "", tracked[p]
		if ok {
			body, ok = sh.repo.Show("HEAD", p)
		}
		if !ok {
			return true, false
		}
		combined.WriteString(body)
		combined.WriteString("\n")
	}
	for _, v := range values {
		if !strings.Contains(combined.String(), v) {
			return true, false
		}
	}
	return true, true
}
