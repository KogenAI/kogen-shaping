package shape

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// scopedStepPath is the flat namespace for newly proposed step files.
func scopedStepPath(stepsDir, slug, path string) bool {
	if filepath.Dir(path) != stepsDir || filepath.Ext(path) != ".js" {
		return false
	}
	suffix, ok := strings.CutPrefix(strings.TrimSuffix(filepath.Base(path), ".js"), slug+"__")
	return ok && suffix != ""
}

// sharedStepCollision permits old shared dependencies only when both existing approval
// hashes still describe the complete current packages.
func sharedStepCollision(root, slug string, cfg *project.Config) error {
	intent, err := project.LoadIntent(root, slug)
	if err != nil {
		return nil
	} // package validation reports unreadable packages
	paths, err := filepath.Glob(filepath.Join(root, ".kogen/intents/*/intent.md"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		otherSlug := filepath.Base(filepath.Dir(path))
		if otherSlug == slug {
			continue
		}
		other, err := project.LoadIntent(root, otherSlug)
		if err != nil {
			continue
		}
		for _, p := range intent.StepFiles() {
			if !slices.Contains(other.StepFiles(), p) {
				continue
			}
			unchanged := true
			for _, owner := range []string{slug, otherSlug} {
				a, err := project.ReadApproval(root, owner)
				if err != nil {
					return err
				}
				baseline, err := project.Baseline(root, owner, cfg)
				if err != nil {
					return err
				}
				if a == nil || a.SHA256 != project.Hash(root, baseline) {
					unchanged = false
				}
			}
			if !unchanged {
				e := sys.Fail("intent", "step_ownership_collision", fmt.Sprintf("Intents %s and %s share changed or newly adopted step file %s; restore their approved dependencies or correct the package", slug, otherSlug, p), 1)
				e.Lines = []string{p}
				return e
			}
		}
	}
	return nil
}

// bindingCollision uses Cucumber's actual matching semantics for the current features.
func bindingCollision(ctx context.Context, root, slug string, s *sys.Session, tools verify.Tools) error {
	intent, scenarios, err := project.CheckIntent(root, slug, nil)
	if err != nil {
		return nil
	}
	paths, err := filepath.Glob(filepath.Join(root, ".kogen/intents/*/intent.md"))
	if err != nil {
		return err
	}
	own := intent.StepFiles()
	for _, path := range paths {
		otherSlug := filepath.Base(filepath.Dir(path))
		if otherSlug == slug {
			continue
		}
		other, otherScenarios, err := project.CheckIntent(root, otherSlug, nil)
		if err != nil {
			continue
		}
		for _, a := range own {
			for _, b := range other.StepFiles() {
				if a == b {
					continue
				}
				for _, ss := range [][]project.Scenario{scenarios, otherScenarios} {
					var texts []string
					for _, sc := range ss {
						for _, st := range sc.Steps {
							texts = append(texts, st.Text)
						}
					}
					node, cucumber := tools.Node, tools.Cucumber
					if node == "" {
						node = s.Getenv("KOGEN_NODE")
					}
					if cucumber == "" {
						cucumber = s.Getenv("KOGEN_CUCUMBER")
					}
					for _, text := range texts {
						raw, loaded, err := verify.RunStepScript(ctx, node, cucumber, root, s.Env, []string{a, b}, []string{text})
						if err != nil {
							return err
						}
						if !loaded {
							continue
						}
						usedA, usedB := false, false
						for _, d := range raw.Defs {
							if d.Used && d.File == a {
								usedA = true
							}
							if d.Used && d.File == b {
								usedB = true
							}
						}
						if usedA && usedB {
							e := sys.Fail("intent", "step_ownership_collision", fmt.Sprintf("Intents %s and %s have overlapping step bindings in %s and %s", slug, otherSlug, a, b), 1)
							e.Lines = []string{a, b}
							return e
						}
					}
				}
			}
		}
	}
	return nil
}
