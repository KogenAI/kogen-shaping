package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
)

// dataStoresTool is the native shaper's model-free scan of where an entity's data lives.
const dataStoresTool = "data_stores"

// dataStoresHeader is the optional intent.md section the scan and the gate read.
const dataStoresHeader = "## Data stores"

const dataStoresInvalidKeys = "data_stores: keys must be a nonempty list of id key spellings without whitespace or backticks; nothing was scanned"

// dataStoresGuidance closes the tool's output: how to record the references.
const dataStoresGuidance = "Record them in a ## Data stores section of intent.md: a line - key `<key>` for each key, then, for each file above or a directory (a path ending in /) that holds some of them, either - `<path>`: in scope, observed by `<text>`, where acceptance.feature or a step file listed under Acceptance contains <text>, or - `<path>`: excluded by decision <n>, where line <n> of ## Decisions names <path> and keeps its source. Shape refuses a candidate whose section leaves one of these files uncovered."

// externalDataStoresGuidance is the paragraph of the external shaper's session text; it has no
// Kogen tools, so the paragraph stands in for the tool's definition and result.
const externalDataStoresGuidance = "Data stores: for a request about an entity's data (for example deleting, exporting or moving an account's data), establish where that data lives before you write scope and acceptance: its schemas, the associations its model reaches, and every other store keyed by the entity's id (other tables, files, directories, indexes), found by searching the repository for each spelling of the id key. Record them in a ## Data stores section of intent.md: a line - key `<key>` for each spelling, then, for each file or directory (a path ending in /) that references a key, either - `<path>`: in scope, observed by `<text>`, where acceptance.feature or a step file listed under Acceptance contains <text>, or - `<path>`: excluded by decision <n>, where line <n> of ## Decisions names <path> and keeps its source. Shape refuses a candidate whose section leaves uncovered a file tracked at HEAD that references a declared key.\n"

// dataStoresDef is the definition of the data_stores tool as the native shaper receives it.
func dataStoresDef() map[string]any {
	return map[string]any{
		"type": "function",
		"name": dataStoresTool,
		"description": "Find where an entity's data lives. For a request about an entity's data (for example deleting, exporting or moving an account's data), " +
			"call this with each spelling of the entity's id key before you write scope and acceptance. " +
			"It lists every file tracked at HEAD that references a key and how intent.md's ## Data stores section covers each one.",
		"parameters": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keys": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required":             []any{"keys"},
			"additionalProperties": false,
		},
		"strict": false,
	}
}

// dataStoreRef is one reference of the scan: a file whose content contains a key, with the
// 1-based numbers of the lines that contain it.
type dataStoreRef struct {
	file, key string
	lines     []int
}

// text is the reference as the tool and the gate write it, without the leading "- ".
func (r dataStoreRef) text() string {
	nums := make([]string, len(r.lines))
	for i, n := range r.lines {
		nums[i] = strconv.Itoa(n)
	}
	if len(nums) == 1 {
		return fmt.Sprintf("%s references %s (line %s)", r.file, r.key, nums[0])
	}
	return fmt.Sprintf("%s references %s (lines %s)", r.file, r.key, strings.Join(nums, ", "))
}

// scanDataStores is the references of keys among the files tracked at HEAD, outside .kogen/:
// git grep -I -n -w -F for each key in order, then ordered by file path and key order.
func scanDataStores(repo *git.Repo, keys []string) ([]dataStoreRef, error) {
	var refs []dataStoreRef
	for _, key := range keys {
		args := []string{"grep", "-z", "-I", "-n", "-w", "-F", "-e", key, "HEAD", "--", ".", ":(exclude).kogen"}
		res, err := repo.Run(args...)
		if err != nil {
			return nil, err
		}
		if res.Status != 0 && res.Status != 1 {
			out := strings.TrimSpace(res.Stderr)
			if out == "" {
				out = strings.TrimSpace(res.Stdout)
			}
			return nil, &git.CommandError{Args: args, Status: res.Status, Output: out}
		}
		var found []dataStoreRef
		out := res.Stdout
		for out != "" {
			// HEAD:<path> NUL <line> NUL <content> NL
			name, rest, ok := strings.Cut(out, "\x00")
			if !ok {
				break
			}
			num, rest, ok := strings.Cut(rest, "\x00")
			if !ok {
				break
			}
			_, out, _ = strings.Cut(rest, "\n")
			n, err := strconv.Atoi(num)
			if err != nil {
				continue
			}
			file := strings.TrimPrefix(name, "HEAD:")
			if k := len(found) - 1; k >= 0 && found[k].file == file {
				found[k].lines = append(found[k].lines, n)
			} else {
				found = append(found, dataStoreRef{file: file, key: key, lines: []int{n}})
			}
		}
		refs = append(refs, found...)
	}
	order := make(map[string]int, len(keys))
	for i, k := range keys {
		if _, ok := order[k]; !ok {
			order[k] = i
		}
	}
	slices.SortStableFunc(refs, func(a, b dataStoreRef) int {
		if c := strings.Compare(a.file, b.file); c != 0 {
			return c
		}
		return order[a.key] - order[b.key]
	})
	return refs, nil
}

// validStoreKey is a key spelling: nonempty, without whitespace or a backtick.
func validStoreKey(k string) bool {
	return k != "" && !strings.ContainsFunc(k, func(r rune) bool { return unicode.IsSpace(r) || r == '`' })
}

// dataStores is the data_stores tool: the scan of the given keys and how to record it.
func (sh *shaping) dataStores(args map[string]any) (string, error) {
	list, _ := args["keys"].([]any)
	var keys []string
	for _, v := range list {
		k, ok := v.(string)
		if !ok || !validStoreKey(k) {
			return dataStoresInvalidKeys, nil
		}
		if !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return dataStoresInvalidKeys, nil
	}
	refs, err := scanDataStores(sh.repo, keys)
	if err != nil {
		return "", err
	}
	lines := []string{"Data stores for keys " + strings.Join(keys, ", ") + ": files tracked at HEAD that reference a key"}
	for _, r := range refs {
		lines = append(lines, "- "+r.text())
	}
	if len(refs) == 0 {
		lines = append(lines, "- none")
	}
	return strings.Join(append(lines, dataStoresGuidance), "\n"), nil
}

// ---- the ## Data stores section and its gate ----

var (
	storeKeyLine      = regexp.MustCompile("^- key `([^`]*)`$")
	storeInScopeLine  = regexp.MustCompile("^- `([^`]*)`: in scope, observed by `([^`]*)`$")
	storeExcludedLine = regexp.MustCompile("^- `([^`]*)`: excluded by decision ([1-9][0-9]*)$")
)

// storeItem is a path item of the section: in scope with the text that observes it, or
// excluded by decision n (digits as written).
type storeItem struct {
	path, text, digits string
	excluded           bool
	n                  int
}

func (i storeItem) matches(file string) bool {
	if strings.HasSuffix(i.path, "/") {
		return strings.HasPrefix(file, i.path)
	}
	return file == i.path
}

// dataStoresSection is the `- ` lines of the ## Data stores section of an intent.md, up to the
// next line starting with #; ok is false when there is no such section.
func dataStoresSection(text string) (bullets []string, ok bool) {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != dataStoresHeader {
			continue
		}
		for _, l := range lines[i+1:] {
			if strings.HasPrefix(l, "#") {
				break
			}
			if strings.HasPrefix(l, "- ") {
				bullets = append(bullets, l)
			}
		}
		return bullets, true
	}
	return nil, false
}

// dataStoresProblems checks a written intent.md that has a ## Data stores section: the problem
// lines, empty when the section is sound and covers every reference of its keys.
func (sh *shaping) dataStoresProblems(intent *project.Intent) ([]string, error) {
	bullets, ok := dataStoresSection(intent.Text)
	if !ok {
		return nil, nil
	}
	var keys []string
	var items []storeItem
	var malformed []string
	for _, b := range bullets {
		if m := storeKeyLine.FindStringSubmatch(b); m != nil && validStoreKey(m[1]) {
			if !slices.Contains(keys, m[1]) {
				keys = append(keys, m[1])
			}
		} else if m := storeInScopeLine.FindStringSubmatch(b); m != nil && m[1] != "" && m[2] != "" {
			items = append(items, storeItem{path: m[1], text: m[2]})
		} else if m := storeExcludedLine.FindStringSubmatch(b); m != nil && m[1] != "" {
			n, err := strconv.Atoi(m[2])
			if err != nil {
				n = int(^uint(0) >> 1)
			}
			items = append(items, storeItem{path: m[1], excluded: true, digits: m[2], n: n})
		} else {
			malformed = append(malformed, "- not a ## Data stores item: "+b)
		}
	}
	var problems []string
	if len(keys) == 0 {
		problems = append(problems, "- "+dataStoresHeader+" declares no key")
	}
	problems = append(problems, malformed...)

	out, err := sh.repo.OK("ls-tree", "-r", "-z", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	tracked := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	acceptance := sh.acceptanceTexts(intent)

	valid := items[:0:0]
	for _, it := range items {
		switch {
		case !slices.ContainsFunc(tracked, it.matches):
			problems = append(problems, "- "+it.path+": no file tracked at HEAD has this path")
		case !it.excluded && !slices.ContainsFunc(acceptance, func(t string) bool { return strings.Contains(t, it.text) }):
			problems = append(problems, "- "+it.path+": acceptance.feature and the step files listed under Acceptance do not contain "+it.text)
		case it.excluded && len(intent.Decisions) < it.n:
			problems = append(problems, "- "+it.path+": "+"## Decisions has no line "+it.digits)
		case it.excluded && !strings.Contains(intent.Decisions[it.n-1], it.path):
			problems = append(problems, "- "+it.path+": decision "+it.digits+" does not name "+it.path)
		default:
			valid = append(valid, it)
		}
	}

	refs, err := scanDataStores(sh.repo, keys)
	if err != nil {
		return nil, err
	}
	for _, r := range refs {
		if !slices.ContainsFunc(valid, func(it storeItem) bool { return it.matches(r.file) }) {
			problems = append(problems, "- "+r.text()+"; no "+dataStoresHeader+" item covers it")
		}
	}
	return problems, nil
}

// acceptanceTexts are the contents on disk of acceptance.feature and the step files listed
// under Acceptance; a file that cannot be read contains nothing.
func (sh *shaping) acceptanceTexts(intent *project.Intent) []string {
	var texts []string
	for _, p := range append([]string{project.FeaturePath(sh.slug)}, intent.StepFiles()...) {
		if b, err := os.ReadFile(filepath.Join(sh.root, p)); err == nil {
			texts = append(texts, string(b))
		}
	}
	return texts
}

// dataStoresGate refuses a written candidate whose ## Data stores section is unsound or leaves
// a referencing file uncovered; ok is true for a refusal.
func (sh *shaping) dataStoresGate() (string, outcome, bool, error) {
	intent, err := project.ReadIntent(sh.root, sh.slug, nil)
	if err != nil {
		return "", outcome{}, false, err
	}
	problems, err := sh.dataStoresProblems(intent)
	if err != nil || len(problems) == 0 {
		return "", outcome{}, false, err
	}
	sh.refused("data_stores", problems)
	headline := fmt.Sprintf("Data stores: %d problem(s) with where the data of the declared keys lives; nothing was published. "+
		"Every file tracked at HEAD that references a key declared under %s needs an item that covers it: "+
		"in scope, observed by text that acceptance.feature or a listed step file contains, or excluded by a numbered Decision that names it. "+
		"Resubmitting this candidate unchanged is refused again.", len(problems), dataStoresHeader)
	return strings.Join(append([]string{headline}, problems...), "\n"), outcome{kind: outInvalid, problems: problems}, true, nil
}
