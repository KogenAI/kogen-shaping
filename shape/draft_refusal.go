package shape

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

// custodyBytes never follows a symlink below the project root. An ignored
// proposal copy is evidence only when it and the current file are regular text.
func custodyBytes(root, path string) ([]byte, bool) {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil || !normalPath(path) {
		return nil, false
	}
	full := filepath.Join(realRoot, path)
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil || resolved != full {
		return nil, false
	}
	st, err := os.Lstat(full)
	if err != nil || !st.Mode().IsRegular() {
		return nil, false
	}
	data, err := os.ReadFile(full)
	return data, err == nil && utf8.Valid(data)
}

// restoreDraftHelpers carries accepted ownership across the documented retry.
// It restores no readiness or approval: the complete candidate is checked again.
func (sh *shaping) restoreDraftHelpers() ([]string, error) {
	owned, err := recoveryOwnedPaths(sh.root, sh.slug, sh.cfg)
	if err != nil || len(owned) == 0 || sh.cfg.Acceptance == nil {
		return nil, err
	}
	var restored []string
	for _, path := range sortedKeys(owned) {
		if !scopedStepPath(sh.cfg.Acceptance.StepsDir, sh.slug, path) {
			continue
		}
		if sh.repo == nil {
			continue
		}
		private := proposedDir(sh.slug) + path
		tracked, err := sh.trackedAtHead(private)
		if err != nil {
			return nil, err
		}
		if tracked {
			continue // Repository content cannot attest private proposal custody.
		}
		current, currentOK := custodyBytes(sh.root, path)
		prior, priorOK := custodyBytes(sh.root, private)
		if !currentOK || !priorOK || !bytes.Equal(current, prior) {
			continue
		}
		if sh.proposed == nil {
			sh.proposed = map[string]string{}
		}
		sh.proposed[path] = string(current)
		restored = append(restored, path)
	}
	if len(restored) > 0 && sh.ev != nil {
		sh.ev.Record(map[string]any{"event": "draft_helpers_restored", "paths": restored})
	}
	return restored, nil
}

// recoveryOwnedPaths distinguishes an unapproved Shape draft from the baseline
// that a later approval freezes for Build. Configured protection always wins.
func recoveryOwnedPaths(root, slug string, cfg *project.Config) (map[string]bool, error) {
	refusal, err := readDraftRefusal(root, slug)
	if err != nil || refusal == nil {
		return nil, err
	}
	explicit, err := project.Protected(root, cfg)
	if err != nil {
		return nil, err
	}
	baseline, err := project.Baseline(root, slug, cfg)
	if err != nil {
		return nil, err
	}
	steps := ""
	if cfg.Acceptance != nil {
		steps = cfg.Acceptance.StepsDir
	}
	shared := map[string]bool{}
	others, err := filepath.Glob(filepath.Join(root, ".kogen/intents/*/intent.md"))
	if err != nil {
		return nil, err
	}
	for _, path := range others {
		other := filepath.Base(filepath.Dir(path))
		if other != slug {
			if intent, err := project.LoadIntent(root, other); err == nil {
				for _, step := range intent.StepFiles() {
					shared[step] = true
				}
			}
		}
	}
	owned := map[string]bool{}
	for _, path := range baseline {
		if !shared[path] && !slices.Contains(explicit, path) && ((strings.HasPrefix(path, project.IntentDir(slug)+"/") && normalPath(path)) || (steps != "" && scopedStepPath(steps, slug, path))) {
			owned[path] = true
		}
	}
	return owned, nil
}

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
