package shape

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/KogenAI/kogen-loop/tools"

	"github.com/KogenAI/kogen-coordinator/project"
)

// scopeGate is the gate name of a candidate refused for being too large for one review.
const scopeGate = "scope"

const scopeGuidance = "Above the bound, split the work into an ordered ## Stages plan of at least two stages, each with a real gate in its checks and each inside the bound, " +
	"or narrow the Intent to the requested behaviour. A stage's goal names in backticks the files that stage changes; a named file that no stage goal names counts in every stage, " +
	"and every stage carries the whole Intent package. Do not satisfy the bound by deleting requested behaviour: every behaviour and example the request asks for keeps its scenario. " +
	"Resubmitting this candidate unchanged is refused again."

// scopeSpan is a span of text between a pair of single backticks on one line.
var scopeSpan = regexp.MustCompile("`([^`\n]+)`")

// scopeFile is a file with the bytes it adds to one review.
type scopeFile struct {
	path string
	size int
}

// scopeInput is what the scope check counts: the candidate package as written, the blobs of
// the HEAD tree and the Intent's scenarios, harnesses and stages.
type scopeInput struct {
	intentText string
	pkg        []scopeFile    // the Intent package in order: intent.md, the Acceptance: paths, the model
	head       map[string]int // blob path -> size at HEAD
	scenarios  int
	harnesses  int
	stages     []project.Stage // nil without a plan
	bound      int
}

// scopeSpans are the texts of the single-backtick spans of text, in order.
func scopeSpans(text string) []string {
	var spans []string
	for _, m := range scopeSpan.FindAllStringSubmatch(text, -1) {
		spans = append(spans, m[1])
	}
	return spans
}

// namedFiles are the files the Intent text names: spans that equal a HEAD blob path outside
// .kogen/ and outside the Intent package, once each, in order of first appearance.
func (in scopeInput) namedFiles() []scopeFile {
	inPackage := map[string]bool{}
	for _, p := range in.pkg {
		inPackage[p.path] = true
	}
	var named []scopeFile
	seen := map[string]bool{}
	for _, span := range scopeSpans(in.intentText) {
		size, ok := in.head[span]
		if !ok || seen[span] || inPackage[span] || strings.HasPrefix(span, ".kogen/") {
			continue
		}
		seen[span] = true
		named = append(named, scopeFile{span, size})
	}
	return named
}

// scopeUnit is one thing a review must hold: the whole Intent or one stage.
type scopeUnit struct {
	name  string
	named []scopeFile
}

func (in scopeInput) units() []scopeUnit {
	named := in.namedFiles()
	if in.stages == nil {
		return []scopeUnit{{"the Intent", named}}
	}
	goals := make([]map[string]bool, len(in.stages))
	assigned := map[string]bool{}
	for i, st := range in.stages {
		goals[i] = map[string]bool{}
		for _, span := range scopeSpans(st.Goal) {
			goals[i][span] = true
			assigned[span] = true
		}
	}
	units := make([]scopeUnit, len(in.stages))
	for i, st := range in.stages {
		units[i].name = fmt.Sprintf("stage %q", st.ID)
		for _, f := range named {
			if goals[i][f.path] || !assigned[f.path] {
				units[i].named = append(units[i].named, f)
			}
		}
	}
	return units
}

// scopeLines is the refusal of a candidate with a unit over the bound, nil when every unit fits.
func scopeLines(in scopeInput) []string {
	var lines []string
	pkgBytes := 0
	for _, p := range in.pkg {
		pkgBytes += p.size
	}
	for _, u := range in.units() {
		estimate := pkgBytes
		areas := map[string]bool{}
		for _, f := range u.named {
			areas[path.Dir(f.path)] = true
		}
		if estimate <= in.bound {
			continue
		}
		lines = append(lines, fmt.Sprintf("Scope check: %s is estimated at %d bytes of one review (scenarios %d, acceptance harnesses %d, named files %d in areas %d), over the %d-byte bound of one review (tools.InputBudgetBytes).",
			u.name, estimate, in.scenarios, in.harnesses, len(u.named), len(areas), in.bound))
		for _, p := range in.pkg {
			lines = append(lines, fmt.Sprintf("- %s: %d bytes (Intent package)", p.path, p.size))
		}
	}
	if lines == nil {
		return nil
	}
	return append(lines, scopeGuidance)
}

// headBlobs are the blobs of the HEAD tree with their sizes; empty without a commit.
func (sh *shaping) headBlobs() map[string]int {
	blobs := map[string]int{}
	out, err := sh.repo.OK("ls-tree", "-r", "-l", "-z", "HEAD")
	if err != nil {
		return blobs
	}
	for _, rec := range strings.Split(out, "\x00") {
		meta, name, ok := strings.Cut(rec, "\t")
		if !ok {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) != 4 || fields[1] != "blob" {
			continue
		}
		if size, err := strconv.Atoi(fields[3]); err == nil {
			blobs[name] = size
		}
	}
	return blobs
}

// scopeRefusal is the refusal of a written candidate whose review would not fit the review
// stage's bound; false when every unit fits or the package cannot be counted (later gates
// report that).
func (sh *shaping) scopeRefusal() (string, outcome, bool) {
	intent, scenarios, err := project.CheckIntent(sh.root, sh.slug, nil)
	if err != nil {
		return "", outcome{}, false
	}
	paths := []string{project.IntentMDPath(sh.slug)}
	paths = append(paths, intent.Acceptance...)
	if intent.Model != nil && !intent.Model.None {
		paths = append(paths, intent.Model.Path)
	}
	in := scopeInput{intentText: intent.Text, head: sh.headBlobs(), scenarios: len(scenarios), harnesses: len(intent.StepFiles()), bound: tools.InputBudgetBytes}
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		b, err := os.ReadFile(filepath.Join(sh.root, p))
		if err != nil {
			return "", outcome{}, false
		}
		in.pkg = append(in.pkg, scopeFile{p, len(b)})
	}
	if stages, err := project.ParseStages(intent.Text); err == nil {
		in.stages = stages
	}
	lines := scopeLines(in)
	if lines == nil {
		return "", outcome{}, false
	}
	sh.refused(scopeGate, lines)
	return strings.Join(lines, "\n"), outcome{kind: outInvalid, problems: lines}, true
}
