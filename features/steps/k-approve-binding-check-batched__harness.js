'use strict';
// The Go source of the k-approve-binding-check-batched acceptance harness, which
// k-approve-binding-check-batched__steps.js compiles into the shape package's tests through a Go overlay
// as shape/k_approve_binding_check_batched_acceptance_test.go; that path never exists in the repository.
// The source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// The acceptance harness of k-approve-binding-check-batched: one synthetic repository per
// KABC_CASE with Kogen's normal layout, whose KOGEN_NODE is a wrapper that appends one line to a
// log per invocation and then runs the real Node with the same arguments, up to kabcBudget per
// call. The real Preflight and Approve entry points run there; what they showed, with the Node
// invocations each started, is written as JSON to KABC_OUT for the acceptance steps to judge.

// kabcBudget is how many Node invocations one Preflight or Approve call may start: at most 30
// step-ownership loaders for thirty other Intents and the few of readiness, with room to spare.
const kabcBudget = 40

type kabcRepo struct {
	t    *testing.T
	root string
	env  []string
	log  string
}

// kabcAccept is the repository's acceptance command: the real Cucumber on the Intent's own step
// file only, so a step file of another Intent that throws on load cannot fail this Intent's run.
const kabcAccept = "#!/bin/sh\n" +
	"slug=$(basename \"$(dirname \"$KOGEN_ACCEPTANCE_FEATURES\")\")\n" +
	"steps=\"features/steps/$slug\"__steps.js\n" +
	"[ -f \"$steps\" ] || steps=empty-steps.js\n" +
	"NODE_PATH=\"$KOGEN_CUCUMBER\" exec \"$KOGEN_NODE\" \"$KOGEN_CUCUMBER/@cucumber/cucumber/bin/cucumber.js\" --strict --require \"$steps\" --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n"

func kabcNewRepo(t *testing.T) *kabcRepo {
	t.Helper()
	real, cucumber := os.Getenv("KOGEN_NODE"), os.Getenv("KOGEN_CUCUMBER")
	if real == "" || cucumber == "" {
		t.Fatalf("TestKApproveBindingCheckBatchedAcceptance: KOGEN_NODE %q and KOGEN_CUCUMBER %q are both required", real, cucumber)
	}
	aux := t.TempDir()
	r := &kabcRepo{t: t, root: t.TempDir(), log: filepath.Join(aux, "node.log")}
	wrapper := filepath.Join(aux, "node")
	// Past kabcBudget invocations in one call the wrapper logs each further invocation as over
	// and refuses it at once instead of running Node.
	script := "#!/bin/sh\nif [ \"$(wc -l < '" + r.log + "')\" -ge " + fmt.Sprint(kabcBudget) + " ]; then echo over >> '" + r.log + "'; exit 97; fi\n" +
		"echo node >> '" + r.log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") && !strings.HasPrefix(kv, "KOGEN_") {
			r.env = append(r.env, kv)
		}
	}
	r.env = append(r.env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "HOME="+aux,
		"KOGEN_NODE="+wrapper, "KOGEN_CUCUMBER="+cucumber)
	r.git(r.root, "init", "-q")
	r.git(r.root, "config", "user.name", "Fixture")
	r.git(r.root, "config", "user.email", "fixture@example.invalid")
	r.git(r.root, "config", "core.hooksPath", t.TempDir())
	r.git(r.root, "config", "commit.gpgsign", "false")
	r.write(r.root, ".gitignore", ".kogen/local/\n")
	r.write(r.root, ".kogen/project.yaml", "version: 1\ncommands:\n  acceptance:\n    run: 'sh accept.sh'\n    steps_dir: 'features/steps'\n")
	r.write(r.root, "accept.sh", kabcAccept)
	r.write(r.root, "empty-steps.js", "// no steps\n")
	return r
}

func (r *kabcRepo) git(dir string, args ...string) {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = dir, r.env
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
}

func (r *kabcRepo) write(dir, rel, content string) {
	r.t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *kabcRepo) commit(dir, message string) {
	r.t.Helper()
	r.git(dir, "add", "-A")
	r.git(dir, "commit", "-qm", message)
}

// addIntent writes the Intent slug in dir: a feature using the ten steps "<slug>: step K" and
// its own step file defining them, after head and followed by extra.
func (r *kabcRepo) addIntent(dir, slug, head, extra string) {
	r.t.Helper()
	intent := "# Sample " + slug + "\n\nSlug: " + slug + "\nModel: not applicable — synthetic fixture\nAcceptance:\n" +
		"- .kogen/intents/" + slug + "/acceptance.feature\n- features/steps/" + slug + "__steps.js\n\nSynthetic fixture.\n\n" +
		"## Decisions\n- Fixture decision. (source: request)\n\n## Tests\n- none: fixture\n\n## Feasibility\n- none\n"
	feature := "Feature: " + slug + "\n  Scenario: All ten steps\n"
	steps := head + "const { Given } = require('@cucumber/cucumber');\n"
	for k := 1; k <= 10; k++ {
		keyword := "And"
		switch k {
		case 1:
			keyword = "Given"
		case 10:
			keyword = "Then"
		}
		feature += fmt.Sprintf("    %s %s: step %d\n", keyword, slug, k)
		steps += fmt.Sprintf("Given('%s: step %d', function () {});\n", slug, k)
	}
	r.write(dir, ".kogen/intents/"+slug+"/intent.md", intent)
	r.write(dir, ".kogen/intents/"+slug+"/acceptance.feature", feature)
	r.write(dir, "features/steps/"+slug+"__steps.js", steps+extra)
}

// watch starts the count of one call. Its context is cancelled as soon as the wrapper refuses an
// invocation past the budget, so a call that starts too many loaders ends early; stop ends the
// watch once the call returned.
func (r *kabcRepo) watch() (context.Context, func()) {
	r.t.Helper()
	if err := os.WriteFile(r.log, nil, 0o644); err != nil {
		r.t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(10 * time.Millisecond):
			}
			if b, _ := os.ReadFile(r.log); strings.Contains(string(b), "over\n") {
				cancel()
				return
			}
		}
	}()
	return ctx, func() { close(done); cancel() }
}

// count is how many Node invocations the wrapper logged since watch, and whether any of them
// was past the budget and refused.
func (r *kabcRepo) count(call map[string]any) {
	b, _ := os.ReadFile(r.log)
	over := strings.Count(string(b), "over\n")
	call["nodes"], call["over_budget"] = strings.Count(string(b), "node\n")+over, over > 0
}

func (r *kabcRepo) session() (*sys.Session, *strings.Builder) {
	s := sys.NewWith(r.env, r.root)
	var out, errOut strings.Builder
	s.Stdout, s.Stderr = &out, &errOut
	return s, &out
}

// approve is one Approve call (the preview without a hash) and the Node invocations it started.
func (r *kabcRepo) approve(slug, hash string) map[string]any {
	ctx, stop := r.watch()
	s, out := r.session()
	exit, err := Approve(ctx, s, ApproveInput{Project: r.root, Slug: slug, Hash: hash, By: "Fixture"})
	stop()
	call := map[string]any{"exit": exit, "output": out.String(), "error": "", "render": "", "lines": []string{}}
	r.count(call)
	if err != nil {
		call["error"] = err.Error()
		if e, ok := sys.AsError(err); ok {
			call["render"] = e.Render()
			if e.Lines != nil {
				call["lines"] = e.Lines
			}
		}
	}
	return call
}

// preflight is one JSON Preflight call: its package check, its step-ownership blockers and the
// Node invocations it started.
func (r *kabcRepo) preflight(slug string) map[string]any {
	ctx, stop := r.watch()
	s, out := r.session()
	exit, err := Preflight(ctx, s, PreflightInput{Project: r.root, Slug: slug, JSON: true})
	stop()
	call := map[string]any{"exit": exit, "error": "", "package": "", "ownership_blockers": []any{}}
	r.count(call)
	if err != nil {
		call["error"] = err.Error()
	}
	var doc struct {
		Data struct {
			Checks   []map[string]any
			Blockers []map[string]any
		}
	}
	if jerr := json.Unmarshal([]byte(out.String()), &doc); jerr != nil {
		call["error"] = fmt.Sprintf("%v; preflight output %q is not JSON: %v", call["error"], out.String(), jerr)
		return call
	}
	for _, c := range doc.Data.Checks {
		if c["name"] == "package" {
			call["package"] = c["status"]
		}
	}
	blockers := []any{}
	for _, b := range doc.Data.Blockers {
		if b["reason"] == "step_ownership_collision" {
			blockers = append(blockers, b)
		}
	}
	call["ownership_blockers"] = blockers
	return call
}

var kabcDigestRe = regexp.MustCompile("kogen intent approve (\\S+) ([0-9a-f]{64})\n")

// journey previews slug's approval and, when the card names its digest, approves with it, as the
// launcher does; preflight comes first when asked.
func (r *kabcRepo) journey(slug string, preflight bool) map[string]any {
	obs := map[string]any{}
	if preflight {
		obs["preflight"] = r.preflight(slug)
	}
	preview := r.approve(slug, "")
	obs["preview"], obs["approve"] = preview, nil
	if m := kabcDigestRe.FindStringSubmatch(preview["output"].(string)); m != nil && m[1] == slug {
		obs["approve"] = r.approve(slug, m[2])
	}
	_, err := os.Stat(filepath.Join(r.root, ".kogen/local/approvals", slug+".json"))
	obs["approved"] = err == nil
	return obs
}

// kabcSynthetic is the repository of Intents syn-01 … syn-30 and syn-new, whose step file ends
// with extra; syn-09's step file throws on load when throw09 is set.
func kabcSynthetic(t *testing.T, extra string, throw09 bool) *kabcRepo {
	r := kabcNewRepo(t)
	for i := 1; i <= 30; i++ {
		head := ""
		if i == 9 && throw09 {
			head = "throw new Error('syn-09 cannot load');\n"
		}
		r.addIntent(r.root, fmt.Sprintf("syn-%02d", i), head, "")
	}
	r.addIntent(r.root, "syn-new", "", extra)
	r.commit(r.root, "thirty-one Intents")
	return r
}

func TestKApproveBindingCheckBatchedAcceptance(t *testing.T) {
	name, out := os.Getenv("KABC_CASE"), os.Getenv("KABC_OUT")
	if out == "" {
		t.Fatalf("TestKApproveBindingCheckBatchedAcceptance: no KABC_OUT for KABC_CASE %q", name)
	}
	obs := map[string]any{"case": name}
	intents := map[string]any{}
	switch name {
	case "pairs", "load-failure":
		extra := ""
		if name == "load-failure" {
			extra = "Given('syn-09: step 2', function () {});\n"
		}
		// The same syn-new alone: the Node invocations its preview starts outside the check.
		alone := kabcNewRepo(t)
		alone.addIntent(alone.root, "syn-new", "", extra)
		alone.commit(alone.root, "syn-new alone")
		obs["alone_preview"] = alone.approve("syn-new", "")
		intents["syn-new"] = kabcSynthetic(t, extra, name == "load-failure").journey("syn-new", name == "pairs")
	case "one-collision":
		intents["syn-new"] = kabcSynthetic(t, "Given('syn-17: step 4', function () {});\n", false).journey("syn-new", true)
	case "two-collisions":
		intents["syn-new"] = kabcSynthetic(t, "Given('syn-03: step 9', function () {});\nGiven('syn-17: step 4', function () {});\n", false).journey("syn-new", true)
	case "worktrees":
		r := kabcNewRepo(t)
		r.commit(r.root, "project")
		for _, slug := range []string{"syn-alpha", "syn-beta"} {
			wt := filepath.Join(t.TempDir(), slug)
			r.git(r.root, "worktree", "add", "-q", "-b", slug, wt)
			r.addIntent(wt, slug, "", "")
			r.commit(wt, "shape "+slug)
		}
		r.git(r.root, "merge", "-q", "--no-edit", "syn-alpha")
		r.git(r.root, "merge", "-q", "--no-edit", "syn-beta")
		for _, slug := range []string{"syn-alpha", "syn-beta"} {
			intents[slug] = r.journey(slug, true)
		}
	default:
		t.Fatalf("TestKApproveBindingCheckBatchedAcceptance: unknown KABC_CASE %q", name)
	}
	obs["intents"] = intents
	b, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
