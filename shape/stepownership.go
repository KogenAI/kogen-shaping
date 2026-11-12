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
// reports, as one marked stdout line per job, which loaded definitions' files match each text.
// No step body or hook is called. The input is {"jobs":[{"files":[...],"texts":[...]},...]} on
// stdin. Every job gets a fresh support-code library and a fresh module cache (the modules
// loaded for the previous job are forgotten), so a job sees what a process of its own would,
// while the Cucumber package itself is loaded once for all jobs.
const bindingScript = `
const fs = require('fs'), path = require('path');
const cc = require('@cucumber/cucumber');
const b = cc.supportCodeLibraryBuilder;
const inp = JSON.parse(fs.readFileSync(0, 'utf8'));
const base = new Set(Object.keys(require.cache));
inp.jobs.forEach((job, i) => {
  try {
    let n = 0;
    b.reset(process.cwd(), () => String(n++));
    for (const f of job.files) {
      try { require(path.resolve(f)); } catch (e) {}
    }
    const lib = b.finalize();
    const matches = job.texts.map(t => lib.stepDefinitions.filter(d => d.matchesStepName(t)).map(d => d.uri));
    process.stdout.write('\nKOGEN_BINDING_RESULT:' + JSON.stringify({ job: i, matches }) + '\n');
  } catch (e) {}
  for (const k of Object.keys(require.cache)) if (!base.has(k)) delete require.cache[k];
});
`

const bindingResultPrefix = "KOGEN_BINDING_RESULT:"

// bindingJob is one step-file pair (or any set of files) and the texts to match against it.
type bindingJob struct {
	files, texts []string
}

// bindingResult is what one job matched: per text, the files of the definitions matching it.
// loaded is false when the job produced no parsable result.
type bindingResult struct {
	matches [][]string
	loaded  bool
}

// runBindingScript loads files in root and returns, per text, the files of the definitions
// matching it. loaded is false when no parsable result line came out; a cancelled context
// returns its error.
func runBindingScript(ctx context.Context, node, cucumber, root string, env, files, texts []string) (matches [][]string, loaded bool, err error) {
	res, err := runBindingJobs(ctx, node, cucumber, root, env, []bindingJob{{files, texts}})
	if err != nil || len(res) == 0 {
		return nil, false, err
	}
	return res[0].matches, res[0].loaded, nil
}

// runBindingJobs evaluates every job, normally in one Node process (loading Cucumber once
// instead of once per job). A job whose result is missing counts as not loaded, exactly as a
// process of its own that printed nothing would; when it took the process down, the jobs after
// it run in a new process.
func runBindingJobs(ctx context.Context, node, cucumber, root string, env []string, jobs []bindingJob) ([]bindingResult, error) {
	results := make([]bindingResult, len(jobs))
	for start := 0; start < len(jobs); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		type wire struct {
			Files []string `json:"files"`
			Texts []string `json:"texts"`
		}
		rest := make([]wire, 0, len(jobs)-start)
		for _, j := range jobs[start:] {
			rest = append(rest, wire{j.files, j.texts})
		}
		in, _ := json.Marshal(map[string]any{"jobs": rest})
		cmd := exec.CommandContext(ctx, node, "-e", bindingScript)
		cmd.Dir = root
		cmd.Env = append(slices.Clone(env), "NODE_PATH="+cucumber)
		cmd.Stdin = bytes.NewReader(in)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Run()
		last := -1
		for _, line := range strings.Split(out.String(), "\n") {
			rest, ok := strings.CutPrefix(line, bindingResultPrefix)
			if !ok {
				continue
			}
			var res struct {
				Job     int        `json:"job"`
				Matches [][]string `json:"matches"`
			}
			if json.Unmarshal([]byte(rest), &res) != nil || res.Job < 0 || start+res.Job >= len(jobs) || len(res.Matches) != len(jobs[start+res.Job].texts) {
				continue
			}
			results[start+res.Job] = bindingResult{res.Matches, true}
			last = max(last, res.Job)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Jobs up to the last result ran; the next one, if any, ended the process.
		start += last + 2
	}
	return results, nil
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

// bindingCollision uses Cucumber's actual matching semantics for the current features: every
// (own step file, other step file) pair is loaded into a library of its own and matches every
// step text of both Intents, all pairs in one Node process. A pair collides when one text is
// matched by a definition of each file; the first colliding pair in Intent order is refused.
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
	type pair struct{ otherSlug, a, b string }
	var pairs []pair
	var jobs []bindingJob
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
				pairs = append(pairs, pair{otherSlug, a, b})
				jobs = append(jobs, bindingJob{[]string{a, b}, texts})
			}
		}
	}
	if len(jobs) == 0 {
		return nil
	}
	results, err := runBindingJobs(ctx, node, cucumber, root, s.Env, jobs)
	if err != nil {
		return err
	}
	for i, p := range pairs {
		if !results[i].loaded {
			continue
		}
		for _, files := range results[i].matches {
			if slices.Contains(files, p.a) && slices.Contains(files, p.b) {
				e := sys.Fail("intent", "step_ownership_collision", fmt.Sprintf("Intents %s and %s have overlapping step bindings in %s and %s", slug, p.otherSlug, p.a, p.b), 1)
				e.Lines = []string{p.a, p.b}
				return e
			}
		}
	}
	return nil
}
