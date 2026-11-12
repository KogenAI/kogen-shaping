package shape

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/KogenAI/kogen-executor/agent"
	"github.com/KogenAI/kogen-loop/tools"

	"github.com/KogenAI/kogen-coordinator/project"
)

// scopeGate is the gate name of a candidate refused for holding more review material than the
// review ceiling.
const scopeGate = "scope"

const scopeGuidance = "Above the limit this cannot be reviewed: the Intent package, the README and the testing rules go whole into every review prompt, and the review cuts the changed files' diffs and added files into parts of what remains. " +
	"Shrink the Intent package (intent.md, the acceptance feature, its step harnesses, the model) so that its estimate fits; changed files do not count, however large or many. " +
	"If the Intent package cannot shrink, narrow this Intent to its first part and list every remaining part in a ## Split section of intent.md, one bullet per part, `- <slug>: <what the part delivers>`; each part's estimate is the review inputs alone, and its own package must fit as well. " +
	"A ## Stages plan does not lower the estimate, because the final review carries every stage. " +
	"Do not satisfy the limit by deleting requested behaviour: every behaviour and example the request asks for stays in this Intent or in a part. " +
	"Resubmitting this candidate unchanged is refused again."

const scopePartsGuidance = "Above MaxReviewParts the review would take too many model calls, one per part: split the work into separately reviewable Intents, narrowing this Intent to its first part and listing every remaining part in a ## Split section of intent.md, one bullet per part, `- <slug>: <what the part delivers>`, naming in backticks the files that part changes. " +
	"Do not satisfy the limit by deleting requested behaviour. Resubmitting this candidate unchanged is refused again."

const (
	scopeReadme = "README.md"
	scopeRules  = "docs/testing-rules.md"
)

// scopeSourceRoom is the most source bytes one part of a review carries at the least
// (review.MaterialPartBytes): a review whose mandatory prompt (authority, check results, changed
// path catalog) leaves less than this under the ceiling stops before anything is sent. Everything
// else the review handles: it cuts the source (the diff of every changed file, the whole text of
// every added file) into as many ordered parts as it takes, each reviewed in its own session with
// the complete authority, and cuts one file's material across parts when it alone exceeds a part.
// No part count or review time bounds it. The README at HEAD is not a changed file: review sends
// it whole.
const scopeSourceRoom = 6000

// MaxReviewParts is the most review parts an Intent may need. Each part is one review model call,
// so an Intent whose estimate is above it is refused, to be split, rather than spend that many
// calls on one review.
const MaxReviewParts = 8

// scopeChangedFileMaxBytes is the most one changed file adds to the source of the part estimate.
// Review sends an existing changed file as its diff against the start HEAD, which does not exist
// when the Intent is shaped, so a file up to this size (32 KiB) is counted whole and a larger one
// at this allowance for its hunks; added files go whole.
const scopeChangedFileMaxBytes = 32 << 10

// scopeSpan is a span of text between a pair of single backticks on one line.
var scopeSpan = regexp.MustCompile("`([^`\n]+)`")

// scopeBullet is a bullet of the ## Split section: the slug of a part and what it delivers.
var scopeBullet = regexp.MustCompile(`^- ([a-z0-9-]{3,48}): (.*)$`)

// scopeFile is a file with its size in bytes.
type scopeFile struct {
	path string
	size int
}

// scopePart is a remaining part of a split Intent.
type scopePart struct {
	slug, text string
}

// scopeInput is what the scope check counts: the candidate package as written, the review
// inputs, the blobs of the HEAD tree and the Intent's scenarios, harnesses and split parts.
type scopeInput struct {
	intentText string      // intent.md without its ## Split section
	parts      []scopePart // the bullets of the ## Split section
	pkg        []scopeFile // intent.md, the Acceptance: paths, the model
	inputs     []scopeFile // README.md at HEAD and the testing rules
	head       map[string]int
	scenarios  int
	harnesses  int
	ceiling    int
}

// reviewParts estimates the parts the review cuts the source of changed into, the way kogen-review
// does: each part carries the whole authority and the room left under the ceiling for source.
func (in scopeInput) reviewParts(authority int, changed []scopeFile) int {
	source := 0
	for _, f := range changed {
		source += min(f.size, scopeChangedFileMaxBytes)
	}
	room := max(in.ceiling-authority, 1)
	return max((source+room-1)/room, 1)
}

// limit is the most mandatory review material an Intent may have: the review ceiling less the
// room the review keeps for source.
func (in scopeInput) limit() int { return in.ceiling - scopeSourceRoom }

// scopeSplit separates the ## Split section of an intent.md from the rest of its text.
func scopeSplit(text string) (string, []scopePart) {
	var rest []string
	var parts []scopePart
	in := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " \t\r")
		switch {
		case line == "## Split":
			in = true
			continue
		case in && strings.HasPrefix(line, "## "):
			in = false
		}
		if !in {
			rest = append(rest, line)
		} else if m := scopeBullet.FindStringSubmatch(line); m != nil {
			parts = append(parts, scopePart{m[1], m[2]})
		}
	}
	return strings.Join(rest, "\n"), parts
}

// changedFiles are the files the text names as changed: single-backtick spans that equal a HEAD
// blob path outside .kogen/, other than README.md and the Intent package, once each in order of
// first appearance.
func (in scopeInput) changedFiles(text string) []scopeFile {
	skip := map[string]bool{scopeReadme: true}
	for _, p := range in.pkg {
		skip[p.path] = true
	}
	var changed []scopeFile
	for _, m := range scopeSpan.FindAllStringSubmatch(text, -1) {
		size, ok := in.head[m[1]]
		if !ok || skip[m[1]] || strings.HasPrefix(m[1], ".kogen/") {
			continue
		}
		skip[m[1]] = true
		changed = append(changed, scopeFile{path: m[1], size: size})
	}
	return changed
}

func scopeSum(lists ...[]scopeFile) int {
	n := 0
	for _, list := range lists {
		for _, f := range list {
			n += f.size
		}
	}
	return n
}

func scopeAreas(files []scopeFile) int {
	areas := map[string]bool{}
	for _, f := range files {
		areas[path.Dir(f.path)] = true
	}
	return len(areas)
}

// estimate is the bytes of mandatory review material of the Intent: the authority every part of
// its review carries whole. Changed files are not in it; review sends them in parts.
func (in scopeInput) estimate() int {
	return scopeSum(in.pkg, in.inputs)
}

// partEstimate is the same for a part, whose own package does not exist yet and is not counted.
func (in scopeInput) partEstimate(p scopePart) int {
	return scopeSum(in.inputs)
}

// scopeReviewInputs are the review inputs every review carries: the README at HEAD (0 when
// absent) and the testing rules.
func scopeReviewInputs(head map[string]int) []scopeFile {
	return []scopeFile{{path: scopeReadme, size: head[scopeReadme]}, {path: scopeRules, size: len(agent.Prompt(agent.TestingRules))}}
}

func scopeItems(lines []string, kind string, files []scopeFile) []string {
	for _, f := range files {
		lines = append(lines, fmt.Sprintf("- %s: %d bytes (%s)", f.path, f.size, kind))
	}
	return lines
}

func scopeChangedItems(lines []string, changed []scopeFile) []string {
	for _, f := range changed {
		lines = append(lines, fmt.Sprintf("- %s: %d bytes (changed file, area %s; review sends its diff in parts)", f.path, f.size, path.Dir(f.path)))
	}
	return lines
}

// scopeLines is the refusal of a candidate whose Intent or one of whose parts is over the
// limit (an estimate equal to it fits), nil when everything fits.
func scopeLines(in scopeInput) []string {
	var lines []string
	if estimate := in.estimate(); estimate > in.limit() {
		changed := in.changedFiles(in.intentText)
		lines = append(lines, fmt.Sprintf("Scope check: the Intent is estimated at %d bytes of mandatory review material (scenarios %d, acceptance harnesses %d, changed files %d in areas %d), over the %d-byte limit (the %d-byte review ceiling, tools.InputBudgetBytes, less the %d bytes the review keeps for source).",
			estimate, in.scenarios, in.harnesses, len(changed), scopeAreas(changed), in.limit(), in.ceiling, scopeSourceRoom))
		lines = scopeItems(lines, "Intent package", in.pkg)
		lines = scopeItems(lines, "review input", in.inputs)
		lines = scopeChangedItems(lines, changed)
	}
	for _, p := range in.parts {
		estimate := in.partEstimate(p)
		if estimate <= in.limit() {
			continue
		}
		changed := in.changedFiles(p.text)
		lines = append(lines, fmt.Sprintf("Scope check: part %q is estimated at %d bytes of mandatory review material (changed files %d in areas %d; its own package is not counted), over the %d-byte limit (the %d-byte review ceiling, tools.InputBudgetBytes, less the %d bytes the review keeps for source).",
			p.slug, estimate, len(changed), scopeAreas(changed), in.limit(), in.ceiling, scopeSourceRoom))
		lines = scopeItems(lines, "review input", in.inputs)
		lines = scopeChangedItems(lines, changed)
	}
	over, tooMany := lines != nil, false
	if estimate := in.estimate(); estimate <= in.limit() {
		if n := in.reviewParts(estimate, in.changedFiles(in.intentText)); n > MaxReviewParts {
			tooMany = true
			lines = append(lines, fmt.Sprintf("Scope check: the Intent is estimated at %d review parts (authority %d bytes in every part), over MaxReviewParts %d; each part is one review model call.", n, estimate, MaxReviewParts))
		}
	}
	for _, p := range in.parts {
		if estimate := in.partEstimate(p); estimate <= in.limit() {
			if n := in.reviewParts(estimate, in.changedFiles(p.text)); n > MaxReviewParts {
				tooMany = true
				lines = append(lines, fmt.Sprintf("Scope check: part %q is estimated at %d review parts (authority %d bytes in every part), over MaxReviewParts %d; each part is one review model call.", p.slug, n, estimate, MaxReviewParts))
			}
		}
	}
	if lines == nil {
		return nil
	}
	if over {
		lines = append(lines, scopeGuidance)
	}
	if tooMany {
		lines = append(lines, scopePartsGuidance)
	}
	return lines
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

// scopeRefusal shows the review estimate of a written candidate and refuses it when the Intent
// or a part of its split leaves the review no room for source; false when everything fits or the
// package cannot be counted (later gates report that).
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
	in := scopeInput{head: sh.headBlobs(), scenarios: len(scenarios), harnesses: len(intent.StepFiles()), ceiling: tools.InputBudgetBytes}
	in.intentText, in.parts = scopeSplit(intent.Text)
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
		in.pkg = append(in.pkg, scopeFile{path: p, size: len(b)})
	}
	in.inputs = scopeReviewInputs(in.head)
	sh.progress(fmt.Sprintf("kogen: review estimate %d bytes of the %d-byte limit (review ceiling %d, tools.InputBudgetBytes, less %d for source)", in.estimate(), in.limit(), in.ceiling, scopeSourceRoom))
	lines := scopeLines(in)
	if lines == nil {
		if len(in.parts) > 0 {
			var parts []string
			for _, p := range in.parts {
				parts = append(parts, fmt.Sprintf("%s (%d of %d bytes)", p.slug, in.partEstimate(p), in.limit()))
			}
			sh.progress(fmt.Sprintf("kogen: split: this Intent is part 1 of %d; shape the other parts as their own Intents: %s", len(in.parts)+1, strings.Join(parts, ", ")))
		}
		return "", outcome{}, false
	}
	sh.refused(scopeGate, lines)
	return strings.Join(lines, "\n"), outcome{kind: outInvalid, problems: lines}, true
}
