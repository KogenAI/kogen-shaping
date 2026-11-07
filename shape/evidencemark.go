package shape

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// ---- (evidence: ...) markers of the Decisions section and their gate ----

const evidenceOpen = "(evidence: "

// evidenceCitations are the citations of the markers of a Decisions line, in order: a marker
// starts at "(evidence: " before the trailing source tag and ends at the first ")" outside a
// backtick-quoted span, or at the end of the line.
func evidenceCitations(line string) []string {
	line = sourceTagRe.ReplaceAllString(line, "")
	var out []string
	for {
		i := strings.Index(line, evidenceOpen)
		if i < 0 {
			return out
		}
		rest := line[i+len(evidenceOpen):]
		end, quoted := len(rest), false
		for j, r := range rest {
			if r == '`' {
				quoted = !quoted
			} else if r == ')' && !quoted {
				end = j
				break
			}
		}
		out = append(out, rest[:end])
		line = rest[end:]
	}
}

// splitCitation cuts a citation at each "; " outside a backtick-quoted span.
func splitCitation(citation string) []string {
	var items []string
	quoted, start := false, 0
	for i := 0; i < len(citation); i++ {
		switch {
		case citation[i] == '`':
			quoted = !quoted
		case !quoted && strings.HasPrefix(citation[i:], "; "):
			items = append(items, citation[start:i])
			start = i + 2
			i++
		}
	}
	return append(items, citation[start:])
}

// evidenceItem is one item of a citation: a command the shaping ran, or a path or directory
// tracked at HEAD; empty for a blank item.
type evidenceItem struct {
	command, path string
	empty         bool
}

func parseEvidenceItem(s string) evidenceItem {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, "ran `"); ok && strings.HasSuffix(rest, "`") {
		command := strings.TrimSuffix(rest, "`")
		return evidenceItem{command: command, empty: strings.TrimSpace(command) == ""}
	}
	if len(s) >= 2 && strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return evidenceItem{path: s, empty: s == ""}
}

// cleanCitedPath is a cited or tool path after path.Clean of its trimmed text; empty is the
// whole repository.
func cleanCitedPath(p string) string {
	return path.Clean(strings.TrimSpace(p))
}

// listedCovers reports whether a list_files call covered the directory: the directory itself,
// an ancestor of it or the whole repository.
func (sh *shaping) listedCovers(dir string) bool {
	for d := dir; ; d = path.Dir(d) {
		if sh.listedPaths[d] {
			return true
		}
		if d == "." || d == "/" {
			return false
		}
	}
}

// evidenceProblems checks the (evidence: ...) markers of a written intent.md's Decisions: the
// problem lines, empty when there is no marker or every item holds.
func (sh *shaping) evidenceProblems(intent string) ([]string, error) {
	problems, _, err := sh.evidenceCheck(intent)
	return problems, err
}

// carriedItem is a cited item that rested only on a receipt of the decision ledger.
type carriedItem struct {
	decision int
	item     string
	receipt  receipt
}

// evidenceCheck is evidenceProblems with the items that rested only on a carried receipt.
func (sh *shaping) evidenceCheck(intent string) ([]string, []carriedItem, error) {
	var carried []carriedItem
	var tracked []string
	var listed bool
	var problems []string
	n := 0
	for _, line := range decisionLines(intent) {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		n++
		for _, citation := range evidenceCitations(line) {
			for _, raw := range splitCitation(citation) {
				it := parseEvidenceItem(raw)
				switch {
				case it.empty:
					problems = append(problems, fmt.Sprintf("- decision %d: an empty (evidence: ...) item", n))
				case it.command != "":
					switch {
					case sh.ext != nil && !sh.ranCommand:
						problems = append(problems, fmt.Sprintf("- decision %d: this shaping's session ran no command, so `%s` was not run", n, it.command))
					case sh.ext == nil && !sh.probed[it.command]:
						if r, ok := sh.ledger.probeReceiptOf(it.command); ok {
							carried = append(carried, carriedItem{n, strings.TrimSpace(raw), r})
						} else {
							problems = append(problems, fmt.Sprintf("- decision %d: no probe of this shaping ran `%s`", n, it.command))
						}
					}
				default:
					if !listed {
						out, err := sh.repo.OK("ls-tree", "-r", "-z", "--name-only", "HEAD")
						if err != nil {
							return nil, nil, err
						}
						tracked, listed = strings.Split(strings.TrimSuffix(out, "\x00"), "\x00"), true
					}
					var cited []string
					switch {
					case isTracked(tracked, it.path):
						cited = []string{it.path}
					default:
						var untracked string
						cited, untracked = wordCitations(tracked, it.path)
						switch {
						case len(cited) == 0:
							problems = append(problems, fmt.Sprintf("- decision %d: %s is not a file or directory tracked at HEAD", n, it.path))
						case untracked != "":
							problems = append(problems, fmt.Sprintf("- decision %d: %s is not a file or directory tracked at HEAD", n, untracked))
							cited = nil
						}
					}
					for _, cite := range cited {
						itemText := strings.TrimSpace(raw)
						if cite != it.path {
							itemText = cite
						}
						problems, carried = sh.checkCitedPath(problems, carried, n, cite, itemText)
					}
				}
			}
		}
	}
	return problems, carried, nil
}

// isTracked reports whether p is a file tracked at HEAD, or a directory written with a trailing
// slash that holds a tracked file.
func isTracked(tracked []string, p string) bool {
	for _, f := range tracked {
		if f == p || strings.HasSuffix(p, "/") && strings.HasPrefix(f, p) {
			return true
		}
	}
	return false
}

// wordCitations reads an item that is not a whole tracked path as space-separated words: the
// cited paths are its leading words that are each tracked (after removing one trailing comma and
// enclosing backticks), none when the first word is not; the rest is prose, except that the first
// later word that looks like a file path (a slash and a file name with an extension, after
// removing enclosing backticks, quotes or parentheses and trailing periods, commas, semicolons
// or colons) and is not a file tracked at HEAD is returned as untracked.
func wordCitations(tracked []string, item string) (cited []string, untracked string) {
	words := strings.Fields(item)
	i := 0
	for ; i < len(words); i++ {
		w := strings.TrimSuffix(words[i], ",")
		if len(w) >= 2 && strings.HasPrefix(w, "`") && strings.HasSuffix(w, "`") {
			w = w[1 : len(w)-1]
		}
		if !isTracked(tracked, w) {
			break
		}
		cited = append(cited, w)
	}
	if len(cited) == 0 {
		return nil, ""
	}
	for _, w := range words[i:] {
		w = strings.TrimLeft(w, "`\"'(")
		w = strings.TrimRight(w, "`\"')"+".,;:")
		base := path.Base(w)
		dot := strings.LastIndex(base, ".")
		if strings.Contains(w, "/") && dot > 0 && dot < len(base)-1 && !slices.Contains(tracked, w) {
			return cited, w
		}
	}
	return cited, ""
}

// checkCitedPath applies the read and listed rules to one cited tracked path of decision n and
// returns the problems and carried items with its result added; item is the text a carried
// receipt is recorded under.
func (sh *shaping) checkCitedPath(problems []string, carried []carriedItem, n int, citedPath, item string) ([]string, []carriedItem) {
	dir := strings.HasSuffix(citedPath, "/")
	cleaned := cleanCitedPath(citedPath)
	var held receipt
	var heldOK, changed bool
	if !dir && sh.ledger != nil && (sh.ext != nil || !sh.readPaths[cleaned]) {
		held, heldOK, changed = sh.ledger.fileReceipt(cleaned, sh.checkoutDigest(cleaned))
	}
	changedLine := fmt.Sprintf("- decision %d: %s changed since round %d read it; read it again", n, citedPath, held.Round)
	switch {
	case heldOK:
		carried = append(carried, carriedItem{n, item, held})
	case sh.ext != nil:
		if !sh.inspected {
			if changed {
				problems = append(problems, changedLine)
			} else {
				problems = append(problems, fmt.Sprintf("- decision %d: this shaping's session read nothing, so %s was not read", n, citedPath))
			}
		}
	case dir && !sh.listedCovers(cleaned):
		problems = append(problems, fmt.Sprintf("- decision %d: this shaping did not list %s", n, citedPath))
	case !dir && !sh.readPaths[cleaned]:
		if changed {
			problems = append(problems, changedLine)
		} else {
			problems = append(problems, fmt.Sprintf("- decision %d: this shaping did not read %s", n, citedPath))
		}
	}
	return problems, carried
}

// evidenceGate refuses a written candidate whose Decisions cite a path that is not tracked at
// HEAD, a file this shaping did not read, a directory it did not list or a command it did not run; ok is true for a refusal.
func (sh *shaping) evidenceGate(intent string) (string, outcome, bool, error) {
	problems, carried, err := sh.evidenceCheck(intent)
	if err != nil {
		return "", outcome{}, false, err
	}
	if len(problems) == 0 {
		if len(carried) > 0 {
			items := make([]any, len(carried))
			for i, c := range carried {
				items[i] = map[string]any{"decision": c.decision, "item": c.item, "kind": c.receipt.Kind, "round": c.receipt.Round}
			}
			sh.ev.Record(map[string]any{"event": "carried_evidence", "items": items})
		}
		return "", outcome{}, false, nil
	}
	sh.refused("decision_evidence", problems)
	headline := fmt.Sprintf("Decision evidence: %d problem(s) with what the Decisions cite; nothing was published. "+
		"A Decision's (evidence: ...) names files this shaping read or directories it listed, tracked at HEAD, or ran `<command>` for a command this shaping ran, separated by \"; \". "+
		"Where the repository has no evidence or conflicting evidence, mark the Decision (assumption: <the uncertainty you decided under>) instead. "+
		"Resubmitting this candidate unchanged is refused again.", len(problems))
	return strings.Join(append([]string{headline}, problems...), "\n"), outcome{kind: outInvalid, problems: problems}, true, nil
}
