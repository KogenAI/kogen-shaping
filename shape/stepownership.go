package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// scopedStepPath is the namespace for newly proposed step files: a step module directly under
// the steps folder, or a helper file inside the slug-owned directory <slug>__<suffix>/ there.
func scopedStepPath(stepsDir, slug, path string) bool {
	if filepath.Dir(path) == stepsDir {
		if filepath.Ext(path) != ".js" {
			return false
		}
		suffix, ok := strings.CutPrefix(strings.TrimSuffix(filepath.Base(path), ".js"), slug+"__")
		return ok && suffix != ""
	}
	if stepsDir == "" {
		return false
	}
	below, ok := strings.CutPrefix(path, stepsDir+"/")
	if !ok {
		return false
	}
	owned, rest, ok := strings.Cut(below, "/")
	suffix, owns := strings.CutPrefix(owned, slug+"__")
	return ok && owns && suffix != "" && rest != "" && normalPath(rest)
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

// bindingScript loads the step files the way verify's stepScript does (each file in its own
// try/catch, so a throwing file keeps the definitions it registered before the throw) and
// reports, as one marked stdout line, which loaded definitions' files match each text. No step
// body or hook is called. The input is {"files":[...],"texts":[...]} on stdin.
const bindingScript = `
const fs = require('fs'), path = require('path');
const cc = require('@cucumber/cucumber');
const b = cc.supportCodeLibraryBuilder;
const inp = JSON.parse(fs.readFileSync(0, 'utf8'));
let n = 0;
b.reset(process.cwd(), () => String(n++));
for (const f of inp.files) {
  try { require(path.resolve(f)); } catch (e) {}
}
const lib = b.finalize();
const matches = inp.texts.map(t => lib.stepDefinitions.filter(d => d.matchesStepName(t)).map(d => d.uri));
process.stdout.write('\nKOGEN_BINDING_RESULT:' + JSON.stringify({ matches }) + '\n');
`

const bindingResultPrefix = "KOGEN_BINDING_RESULT:"

// runBindingScript loads files in root and returns, per text, the files of the definitions
// matching it. loaded is false when no parsable result line came out; a cancelled context
// returns its error.
func runBindingScript(ctx context.Context, node, cucumber, root string, env, files, texts []string) (matches [][]string, loaded bool, err error) {
	in, _ := json.Marshal(map[string]any{"files": files, "texts": texts})
	cmd := exec.CommandContext(ctx, node, "-e", bindingScript)
	cmd.Dir = root
	cmd.Env = append(slices.Clone(env), "NODE_PATH="+cucumber)
	cmd.Stdin = bytes.NewReader(in)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Run()
	for _, line := range strings.Split(out.String(), "\n") {
		if rest, ok := strings.CutPrefix(line, bindingResultPrefix); ok {
			var res struct {
				Matches [][]string `json:"matches"`
			}
			if json.Unmarshal([]byte(rest), &res) == nil && len(res.Matches) == len(texts) {
				return res.Matches, true, nil
			}
		}
	}
	return nil, false, ctx.Err()
}

// stepTexts are the distinct step texts of the scenarios, in order, appended to texts.
func stepTexts(texts []string, scenarios []project.Scenario) []string {
	for _, sc := range scenarios {
		for _, st := range sc.Steps {
			if !slices.Contains(texts, st.Text) {
				texts = append(texts, st.Text)
			}
		}
	}
	return texts
}

// bindingCollision uses Cucumber's actual matching semantics for the current features: one
// loader per (own step file, other step file) pair matches every step text of both Intents.
// A pair collides when one text is matched by a definition of each file.
func bindingCollision(ctx context.Context, root, slug string, s *sys.Session, tools verify.Tools) error {
	intent, scenarios, err := project.CheckIntent(root, slug, nil)
	if err != nil {
		return nil
	}
	paths, err := filepath.Glob(filepath.Join(root, ".kogen/intents/*/intent.md"))
	if err != nil {
		return err
	}
	node, cucumber := tools.Node, tools.Cucumber
	if node == "" {
		node = s.Getenv("KOGEN_NODE")
	}
	if cucumber == "" {
		cucumber = s.Getenv("KOGEN_CUCUMBER")
	}
	own := intent.StepFiles()
	ownTexts := stepTexts(nil, scenarios)
	for _, path := range paths {
		otherSlug := filepath.Base(filepath.Dir(path))
		if otherSlug == slug {
			continue
		}
		other, otherScenarios, err := project.CheckIntent(root, otherSlug, nil)
		if err != nil {
			continue
		}
		texts := stepTexts(slices.Clone(ownTexts), otherScenarios)
		if len(texts) == 0 {
			continue
		}
		for _, a := range own {
			for _, b := range other.StepFiles() {
				if a == b {
					continue
				}
				matches, loaded, err := runBindingScript(ctx, node, cucumber, root, s.Env, []string{a, b}, texts)
				if err != nil {
					return err
				}
				if !loaded {
					continue
				}
				for _, files := range matches {
					if slices.Contains(files, a) && slices.Contains(files, b) {
						e := sys.Fail("intent", "step_ownership_collision", fmt.Sprintf("Intents %s and %s have overlapping step bindings in %s and %s", slug, otherSlug, a, b), 1)
						e.Lines = []string{a, b}
						return e
					}
				}
			}
		}
	}
	return nil
}
