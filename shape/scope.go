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

const scopeGuidance = "Above the ceiling this cannot be approved as one Intent: split the work into separately reviewable Intents. " +
	"Narrow this Intent to its first part so that its estimate fits, and list every remaining part in a ## Split section of intent.md, one bullet per part, `- <slug>: <what the part delivers>`, " +
	"naming in backticks the files that part changes; a part's estimate is the review inputs plus those files, and each part must fit the ceiling too. " +
	"A ## Stages plan does not lower the estimate, because the final review carries every stage. " +
	"Do not satisfy the ceiling by deleting requested behaviour: every behaviour and example the request asks for stays in this Intent or in a part. " +
	"Resubmitting this candidate unchanged is refused again."

const (
	scopeReadme = "README.md"
	scopeRules  = "docs/testing-rules.md"
)

// scopeChangedFileMaxBytes is the most one changed file adds to the estimate. Review sends an
// existing changed file as its diff against the start HEAD (git's default three lines of context
// per hunk, never the whole file; added files go whole), and the diff does not exist when the
// Intent is shaped, so a file up to this size (32 KiB) is counted whole and a larger one at this
// allowance for its hunks. The README at HEAD is not a changed file: review sends it whole.
const scopeChangedFileMaxBytes = 32 << 10

// scopeSpan is a span of text between a pair of single backticks on one line.
var scopeSpan = regexp.MustCompile("`([^`\n]+)`")

// scopeBullet is a bullet of the ## Split section: the slug of a part and what it delivers.
var scopeBullet = regexp.MustCompile(`^- ([a-z0-9-]{3,48}): (.*)$`)

// scopeFile is a file with the bytes it adds to the review material.
type scopeFile struct {
	path string
	size int
	file int // the file's own size when size counts only its diff hunks, else 0
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
		changed = append(changed, scopeChangedFile(m[1], size))
	}
	return changed
}

// scopeChangedFile is a changed file as the estimate counts it: whole up to
// scopeChangedFileMaxBytes, above that at that allowance for its diff hunks.
func scopeChangedFile(path string, size int) scopeFile {
	if size <= scopeChangedFileMaxBytes {
		return scopeFile{path: path, size: size}
	}
	return scopeFile{path: path, size: scopeChangedFileMaxBytes, file: size}
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

// estimate is the bytes of mandatory review material of the Intent as one review.
func (in scopeInput) estimate() int {
	return scopeSum(in.pkg, in.inputs, in.changedFiles(in.intentText))
}

// partEstimate is the same for a part, whose own package does not exist yet.
func (in scopeInput) partEstimate(p scopePart) int {
	return scopeSum(in.inputs, in.changedFiles(p.text))
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
		note := ""
		if f.file > 0 {
			note = fmt.Sprintf("; diff hunks of a %d-byte file", f.file)
		}
		lines = append(lines, fmt.Sprintf("- %s: %d bytes (changed file, area %s%s)", f.path, f.size, path.Dir(f.path), note))
	}
	return lines
}

// scopeLines is the refusal of a candidate whose Intent or one of whose parts is over the
// ceiling (an estimate equal to it fits), nil when everything fits.
func scopeLines(in scopeInput) []string {
	var lines []string
	if estimate := in.estimate(); estimate > in.ceiling {
		changed := in.changedFiles(in.intentText)
		lines = append(lines, fmt.Sprintf("Scope check: the Intent is estimated at %d bytes of mandatory review material (scenarios %d, acceptance harnesses %d, changed files %d in areas %d), over the %d-byte review ceiling (tools.InputBudgetBytes).",
			estimate, in.scenarios, in.harnesses, len(changed), scopeAreas(changed), in.ceiling))
		lines = scopeItems(lines, "Intent package", in.pkg)
		lines = scopeItems(lines, "review input", in.inputs)
		lines = scopeChangedItems(lines, changed)
	}
	for _, p := range in.parts {
		estimate := in.partEstimate(p)
		if estimate <= in.ceiling {
			continue
		}
		changed := in.changedFiles(p.text)
		lines = append(lines, fmt.Sprintf("Scope check: part %q is estimated at %d bytes of mandatory review material (changed files %d in areas %d; its own package is not counted), over the %d-byte review ceiling (tools.InputBudgetBytes).",
			p.slug, estimate, len(changed), scopeAreas(changed), in.ceiling))
		lines = scopeItems(lines, "review input", in.inputs)
		lines = scopeChangedItems(lines, changed)
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

// scopeRefusal shows the review estimate of a written candidate and refuses it when the Intent
// or a part of its split would not fit the review ceiling; false when everything fits or the
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
	sh.progress(fmt.Sprintf("kogen: review estimate %d bytes of the %d-byte review ceiling (tools.InputBudgetBytes)", in.estimate(), in.ceiling))
	lines := scopeLines(in)
	if lines == nil {
		if len(in.parts) > 0 {
			var parts []string
			for _, p := range in.parts {
				parts = append(parts, fmt.Sprintf("%s (%d of %d bytes)", p.slug, in.partEstimate(p), in.ceiling))
			}
			sh.progress(fmt.Sprintf("kogen: split: this Intent is part 1 of %d; shape the other parts as their own Intents: %s", len(in.parts)+1, strings.Join(parts, ", ")))
		}
		return "", outcome{}, false
	}
	sh.refused(scopeGate, lines)
	return strings.Join(lines, "\n"), outcome{kind: outInvalid, problems: lines}, true
}
