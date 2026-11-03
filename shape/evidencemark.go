package shape

import (
	"fmt"
	"path"
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
						problems = append(problems, fmt.Sprintf("- decision %d: no probe of this shaping ran `%s`", n, it.command))
					}
				default:
					if !listed {
						out, err := sh.repo.OK("ls-tree", "-r", "-z", "--name-only", "HEAD")
						if err != nil {
							return nil, err
						}
						tracked, listed = strings.Split(strings.TrimSuffix(out, "\x00"), "\x00"), true
					}
					found := false
					for _, f := range tracked {
						if f == it.path || strings.HasSuffix(it.path, "/") && strings.HasPrefix(f, it.path) {
							found = true
							break
						}
					}
					switch dir := strings.HasSuffix(it.path, "/"); {
					case !found:
						problems = append(problems, fmt.Sprintf("- decision %d: %s is not a file or directory tracked at HEAD", n, it.path))
					case sh.ext != nil:
						if !sh.inspected {
							problems = append(problems, fmt.Sprintf("- decision %d: this shaping's session read nothing, so %s was not read", n, it.path))
						}
					case dir && !sh.listedCovers(cleanCitedPath(it.path)):
						problems = append(problems, fmt.Sprintf("- decision %d: this shaping did not list %s", n, it.path))
					case !dir && !sh.readPaths[cleanCitedPath(it.path)]:
						problems = append(problems, fmt.Sprintf("- decision %d: this shaping did not read %s", n, it.path))
					}
				}
			}
		}
	}
	return problems, nil
}

// evidenceGate refuses a written candidate whose Decisions cite a path that is not tracked at
// HEAD, a file this shaping did not read, a directory it did not list or a command it did not run; ok is true for a refusal.
func (sh *shaping) evidenceGate(intent string) (string, outcome, bool, error) {
	problems, err := sh.evidenceProblems(intent)
	if err != nil || len(problems) == 0 {
		return "", outcome{}, false, err
	}
	sh.refused("decision_evidence", problems)
	headline := fmt.Sprintf("Decision evidence: %d problem(s) with what the Decisions cite; nothing was published. "+
		"A Decision's (evidence: ...) names files this shaping read or directories it listed, tracked at HEAD, or ran `<command>` for a command this shaping ran, separated by \"; \". "+
		"Where the repository has no evidence or conflicting evidence, mark the Decision (assumption: <the uncertainty you decided under>) instead. "+
		"Resubmitting this candidate unchanged is refused again.", len(problems))
	return strings.Join(append([]string{headline}, problems...), "\n"), outcome{kind: outInvalid, problems: problems}, true, nil
}
