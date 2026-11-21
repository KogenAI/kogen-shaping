package shape

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/KogenAI/kogen-verification/verify"
)

// repairDeclaration recognises the sentence of a request that says it repairs the named failing
// checks on main: "repairs the failing check "A" and "B" on main". The checks are the double
// quoted names that follow.
var repairDeclaration = regexp.MustCompile(`(?i)\brepairs?\s+(?:the\s+)?(?:named\s+)?(?:failing|red)\s+(?:checks?|tests?)\s*:?\s*((?:"[^"\n]+"\s*(?:,\s*and\s+|,\s*|and\s+|&\s*)?)+)on\s+(?:the\s+)?main\b`)

var repairQuoted = regexp.MustCompile(`"([^"\n]+)"`)

// repairNames are the checks a request declares that it repairs, in order and without repeats;
// nil when the request has no repair declaration.
func repairNames(request string) []string {
	var out []string
	for _, m := range repairDeclaration.FindAllStringSubmatch(request, -1) {
		for _, q := range repairQuoted.FindAllStringSubmatch(m[1], -1) {
			if name := normalizeWhitespace(q[1]); name != "" && !slices.Contains(out, name) {
				out = append(out, name)
			}
		}
	}
	return out
}

// repairObservation is what the starting revision shows of the checks a request names.
type repairObservation struct {
	// targets are the named checks that fail on the starting revision: the repair target.
	targets []string
	// refusals are the lines for the named checks that pass on the starting revision.
	refusals []string
}

// observeRepair runs the declaration against the readiness observation of the unchanged tree.
// A named check that no scenario of the observation carries is neither a target nor refused.
func observeRepair(names []string, ready *verify.Ready) repairObservation {
	var out repairObservation
	if ready == nil {
		return out
	}
	for _, name := range names {
		failed, passed := false, false
		for _, sc := range ready.Scenarios {
			if normalizeWhitespace(sc.Name) != name {
				continue
			}
			switch sc.Status {
			case "failed":
				failed = true
			case "passed":
				passed = true
			}
		}
		switch {
		case failed:
			out.targets = append(out.targets, name)
		case passed:
			out.refusals = append(out.refusals, fmt.Sprintf("- %s | the named check %q is not failing on the starting revision, so there is nothing to repair. Name only checks that fail on main, or drop the repair declaration from the request.", name, name))
		}
	}
	return out
}

// repairProgress is the progress line that records one repair target.
func repairProgress(name string) string {
	return fmt.Sprintf("kogen: repair target %q: red before on the starting revision, green after the change (not a kept check)", name)
}
