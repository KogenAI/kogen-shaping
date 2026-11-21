package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

// The request check is the model-free first step of a new `kogen intent shape`: it reads the
// request text and the checkout, and spends no provider request, shaper round or Kogen check.
// It has four findings, each with one treatment:
//
//   - the request names a protected file as something to edit: the demand is removed, a line says
//     the change must not touch the file, and the repaired text replaces the request;
//   - two scenarios have the same observable outcome, a scenario has no observable outcome, or a
//     referenced path does not exist in the checkout: the request is refused.
//
// Scenarios are the numbered items under an `## Acceptance` heading, each stating its observable
// outcome after `Outcome:`; referenced paths are backticked tokens with a `/` or a file extension.

var (
	requestHeadingRe  = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.*?)\s*#*\s*$`)
	requestItemRe     = regexp.MustCompile(`^\s*(\d+)[.)]\s+(.*)$`)
	requestOutcomeRe  = regexp.MustCompile(`(?is)\boutcome:\s*(.*)$`)
	requestBacktickRe = regexp.MustCompile("`([^`\n]+)`")
	requestPathRe     = regexp.MustCompile(`^(?:\./)?[A-Za-z0-9_@+-][A-Za-z0-9_@+./-]*$`)
	requestExtRe      = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9]*$`)
	requestNegationRe = regexp.MustCompile(`(?i)\b(?:not|never|without|avoid|no)\b|n't\b`)
	requestWordRe     = regexp.MustCompile(`[\p{L}\p{N}_]+`)
	requestSentenceRe = regexp.MustCompile(`[.!?]+(?:\s+|$)`)
)

const (
	requestEditVerbs   = `edit|modify|change|update|rewrite|overwrite|replace|delete|remove|append|patch|alter|write|create|add|fix|adjust|extend`
	requestEditedForms = `edited|modified|changed|updated|rewritten|overwritten|replaced|deleted|removed|appended|patched|altered|written|created|added|fixed|adjusted|extended`
)

// requestOutcomeNoise are the words that carry no observable outcome of their own: the same
// outcome reads "x returns true" and "true is returned by x".
var requestOutcomeNoise = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`a an the is are was were be been being am it its this that these those and or of to by in on at for
		with as from into then than so shall should must will would can could does do did has have had there which who whom
		return returns returned returning give gives given giving yield yields yielded produce produces produced
		result results resulted show shows shown showing print prints printed printing output outputs outputted
		get gets got gotten value values`) {
		requestOutcomeNoise[w] = true
	}
}

// requestScenario is one numbered item under the acceptance heading.
type requestScenario struct {
	number  string
	text    string
	outcome string
}

// requestScenarios reads the numbered items under the `Acceptance` heading, up to the next heading.
func requestScenarios(request string) []requestScenario {
	var out []requestScenario
	inAcceptance := false
	for _, line := range strings.Split(request, "\n") {
		if m := requestHeadingRe.FindStringSubmatch(line); m != nil {
			inAcceptance = strings.EqualFold(strings.TrimSpace(m[1]), "acceptance")
			continue
		}
		if !inAcceptance {
			continue
		}
		if m := requestItemRe.FindStringSubmatch(line); m != nil {
			out = append(out, requestScenario{number: m[1], text: m[2]})
		} else if len(out) > 0 && strings.TrimSpace(line) != "" {
			out[len(out)-1].text += " " + strings.TrimSpace(line)
		}
	}
	for i := range out {
		if m := requestOutcomeRe.FindStringSubmatch(out[i].text); m != nil {
			out[i].outcome = strings.TrimSpace(m[1])
		}
	}
	return out
}

// outcomeKey is what two outcomes are compared by: their words without the words that only
// connect them, in sorted order. An outcome of only such words is compared by all its words.
func outcomeKey(outcome string) string {
	words := requestWordRe.FindAllString(strings.ToLower(outcome), -1)
	var kept []string
	for _, w := range words {
		if !requestOutcomeNoise[w] {
			kept = append(kept, w)
		}
	}
	if len(kept) == 0 {
		kept = words
	}
	sort.Strings(kept)
	return strings.Join(kept, " ")
}

// requestPaths are the backticked tokens of the request that name a path, in order of first use.
func requestPaths(request string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range requestBacktickRe.FindAllStringSubmatch(request, -1) {
		token := m[1]
		if !requestPathRe.MatchString(token) || !(strings.Contains(token, "/") || requestExtRe.MatchString(token)) {
			continue
		}
		if !seen[token] {
			seen[token] = true
			out = append(out, token)
		}
	}
	return out
}

func requestPathExists(root, token string) bool {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(token, "./")))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	_, err := os.Lstat(filepath.Join(root, clean))
	return err == nil
}

// editDemandRe matches a sentence that demands an edit of the path: an edit verb with the path as
// its object, or the path followed by what must be done to it.
func editDemandRe(path string) *regexp.Regexp {
	p := "`" + regexp.QuoteMeta(path) + "`"
	return regexp.MustCompile(`(?i)\b(?:` + requestEditVerbs + `)\b[^` + "`" + `\n]{0,40}` + p +
		`|` + p + `[^` + "`" + `\n.]{0,40}\b(?:` + requestEditedForms + `)\b`)
}

// repairProtectedDemands removes the sentences that demand an edit of a protected file and puts a
// line that says the change must not touch it where the first of them stood. It returns the
// repaired request and the protected files it repaired for, which are empty when nothing changed.
func repairProtectedDemands(request string, protected []string) (string, []string) {
	protectedSet := map[string]bool{}
	for _, p := range protected {
		protectedSet[p] = true
	}
	var named []string
	for _, token := range requestPaths(request) {
		if p := strings.TrimPrefix(token, "./"); protectedSet[p] {
			named = append(named, token)
		}
	}
	var repaired []string
	lines := strings.Split(request, "\n")
	var out []string
	for _, line := range lines {
		var keep strings.Builder
		var limits []string
		rest := line
		for rest != "" {
			end := len(rest)
			if loc := requestSentenceRe.FindStringIndex(rest); loc != nil {
				end = loc[1]
			}
			sentence := rest[:end]
			rest = rest[end:]
			demand := ""
			if !requestNegationRe.MatchString(sentence) {
				for _, token := range named {
					if editDemandRe(token).MatchString(sentence) {
						demand = token
						break
					}
				}
			}
			if demand == "" {
				keep.WriteString(sentence)
				continue
			}
			if !slices.Contains(repaired, demand) {
				repaired = append(repaired, demand)
				limits = append(limits, "`"+demand+"` is protected: the change must not touch it.")
			}
		}
		if len(limits) == 0 && keep.String() == line {
			out = append(out, line)
			continue
		}
		if strings.TrimSpace(keep.String()) != "" {
			out = append(out, strings.TrimRight(keep.String(), " \t"))
		}
		out = append(out, limits...)
	}
	if len(repaired) == 0 {
		return request, nil
	}
	return strings.Join(out, "\n"), repaired
}

// requestFindings are the rewrite lines of the refusing findings of the request, one per finding.
func requestFindings(root, request string) []string {
	var lines []string
	scenarios := requestScenarios(request)
	first := map[string]string{}
	for _, sc := range scenarios {
		if sc.outcome == "" {
			lines = append(lines, fmt.Sprintf("scenario %s states no observable outcome: add \"Outcome:\" followed by what a user can observe", sc.number))
			continue
		}
		key := outcomeKey(sc.outcome)
		if other, ok := first[key]; ok {
			lines = append(lines, fmt.Sprintf("scenarios %s and %s have the same observable outcome: merge them into one scenario or give one of them a different outcome", other, sc.number))
			continue
		}
		first[key] = sc.number
	}
	for _, token := range requestPaths(request) {
		if !requestPathExists(root, token) {
			lines = append(lines, fmt.Sprintf("`%s` does not exist in the checkout: name a path that exists, or say what to create and where", token))
		}
	}
	return lines
}

// checkRequest is the request check of a new shaping. It returns the request Shape continues
// with, and prints the repair it made; a refusal is request/needs_rewrite, returned before any
// provider request.
func checkRequest(s *sys.Session, root string, cfg *project.Config, request string) (string, error) {
	if lines := requestFindings(root, request); len(lines) > 0 {
		e := sys.Fail("request", "needs_rewrite", "rewrite the request before Shape spends a model call", 1)
		e.Lines = lines
		return "", e
	}
	protected, err := project.Protected(root, cfg)
	if err != nil {
		return "", err
	}
	repaired, files := repairProtectedDemands(request, protected)
	if len(files) > 0 {
		quoted := make([]string, len(files))
		for i, f := range files {
			quoted[i] = "`" + f + "`"
		}
		s.Progress("request repaired: removed the instruction to edit " + strings.Join(quoted, ", ") +
			", which is protected, and added a line saying the change must not touch it")
	}
	return repaired, nil
}
