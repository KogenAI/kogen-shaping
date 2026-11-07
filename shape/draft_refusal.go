package shape

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

// The local latch is independent of the draft contents and is reset when Shape starts
// with no Intent directory.
func draftRefusalPath(root, slug string) string {
	return filepath.Join(root, ".kogen", "local", "shaping", slug, "not-ready.json")
}

type draftRefusal struct {
	Gate     string   `json:"gate"`
	Problems []string `json:"problems"`
}

func readDraftRefusal(root, slug string) (*draftRefusal, error) {
	b, err := os.ReadFile(draftRefusalPath(root, slug))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r draftRefusal
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func draftRecovery(slug string) []string {
	return []string{"Retry Shape on the retained draft, or delete .kogen/intents/" + slug + "/ and shape again.", "Next: kogen intent shape " + slug + " <request-file>"}
}

func draftFiles(root, slug string, extra []string) []string {
	paths := map[string]bool{}
	dir := filepath.Join(root, project.IntentDir(slug))
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if rel, e := filepath.Rel(root, path); e == nil {
				paths[filepath.ToSlash(rel)] = true
			}
		}
		return err
	})
	if intent, err := project.ReadIntent(root, slug, nil); err == nil {
		extra = append(extra, intent.Acceptance...)
		if intent.Model != nil && !intent.Model.None {
			extra = append(extra, intent.Model.Path)
		}
	}
	for _, p := range extra {
		if fi, err := os.Stat(filepath.Join(root, p)); err == nil && fi.Mode().IsRegular() {
			paths[filepath.ToSlash(p)] = true
		}
	}
	list := make([]string, 0, len(paths))
	for p := range paths {
		list = append(list, p)
	}
	sort.Strings(list)
	return list
}

func draftDiagnostic(root, slug, reason, detail string, exit int, problems, extra []string) error {
	e := sys.Fail("intent", reason, detail, exit)
	e.Lines = append(e.Lines, problems...)
	e.Lines = append(e.Lines, "Retained draft files:")
	e.Lines = append(e.Lines, draftFiles(root, slug, extra)...)
	e.Lines = append(e.Lines, draftRecovery(slug)...)
	return e
}
