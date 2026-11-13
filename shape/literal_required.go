package shape

import (
	"regexp"
	"strings"
)

// Required literals that do not depend on the model. The model lists the literals of a request
// and it is unreliable at it: the same request lists a required output line in one run and omits
// it in the next. Kogen therefore reads the literals that a request gives explicitly itself and
// merges the model's list on top of them.
//
// A request gives a literal explicitly when it writes it
//   - in a fenced code block (each non-blank line),
//   - in a backtick code span, or
//   - in double quotes right after a word that marks it exact ("exactly", "verbatim", ...).
//
// Provenance lines (a Studio issue, ticket, blueprint or plan id) are not literals, and neither
// is a command that the request only wants to keep working ("so `sh x.sh` still succeeds").

var (
	fenceLine     = regexp.MustCompile("^ {0,3}(```+|~~~+)")
	codeSpan      = regexp.MustCompile("`([^`\n]+)`")
	exactQuoteFwd = regexp.MustCompile(`(?i)\b(?:exactly|verbatim|word for word|literally|exact (?:text|wording|message|output|line)s?)\b[^"“\n]{0,40}["“]([^"”\n]{2,})["”]`)
	exactQuoteRev = regexp.MustCompile(`(?i)["“]([^"”\n]{2,})["”],?\s+(?:exactly|verbatim)\b`)
	keepsWorking  = regexp.MustCompile(`(?i)\bstill\s+(?:succeeds|works|passes|runs|exits)\b`)
	sentenceBreak = regexp.MustCompile(`[.!?;]\s+|;|\n`)
)

// requestLiterals are the literals that request gives explicitly, in order, each once, as
// required items of the request. The quote of an item is the line or code span it comes from.
func requestLiterals(request string) []literalItem {
	var items []literalItem
	seen := map[string]bool{}
	add := func(literal, quote, why string) {
		literal = strings.TrimSpace(literal)
		if len([]rune(literal)) < 2 || seen[literal] || provenanceParagraph.MatchString(literal) {
			return
		}
		seen[literal] = true
		items = append(items, literalItem{literal: literal, source: "request", quote: quote, status: "required", why: why})
	}
	inFence := false
	for _, line := range strings.Split(request, "\n") {
		if fenceLine.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			if t := strings.TrimSpace(line); t != "" {
				add(t, t, "the request shows this line in a code block")
			}
			continue
		}
		for _, m := range codeSpan.FindAllStringSubmatchIndex(line, -1) {
			if !keepsWorking.MatchString(sentenceAround(line, m[0])) {
				add(line[m[2]:m[3]], line[m[2]:m[3]], "the request writes this as code")
			}
		}
		for _, re := range []*regexp.Regexp{exactQuoteFwd, exactQuoteRev} {
			for _, m := range re.FindAllStringSubmatch(line, -1) {
				add(m[1], m[1], "the request marks this text as exact")
			}
		}
	}
	return items
}

// sentenceAround is the sentence of line that contains the byte offset at.
func sentenceAround(line string, at int) string {
	start, end := 0, len(line)
	for _, b := range sentenceBreak.FindAllStringIndex(line, -1) {
		if b[1] <= at {
			start = b[1]
		} else if b[0] >= at {
			end = b[0]
			break
		}
	}
	return line[start:end]
}

// withRequestLiterals is the literal list to compare: the literals that the request gives
// explicitly, required, then the model's other items. The model never removes a required line
// or code block literal; for a code span it may say informal (an instruction to the shaper, not
// a product value), and for any of them superseded (the request or an answer replaced it); its
// package_value for a literal is kept.
func withRequestLiterals(request string, model []literalItem) []literalItem {
	det := requestLiterals(request)
	if len(det) == 0 {
		return model
	}
	byLiteral := map[string]literalItem{}
	for _, it := range model {
		if _, ok := byLiteral[it.literal]; !ok || it.status == "required" {
			byLiteral[it.literal] = it
		}
	}
	out := make([]literalItem, 0, len(det)+len(model))
	taken := map[string]bool{}
	for _, d := range det {
		taken[d.literal] = true
		if m, ok := byLiteral[d.literal]; ok {
			d.packageValue = m.packageValue
			if m.status == "superseded" || m.status == "secret" || (m.status == "informal" && d.why == "the request writes this as code") {
				d.status = m.status
				d.why = m.why
			}
		}
		out = append(out, d)
	}
	for _, it := range model {
		if !taken[it.literal] {
			out = append(out, it)
		}
	}
	return out
}
