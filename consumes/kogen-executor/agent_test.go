package consumes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/git"

	"kogen-loop/provider"
	"kogen-loop/sys"
	"kogen-loop/tools"

	"github.com/KogenAI/kogen-executor/agent"
)

var (
	_ func(...[]map[string]any) []any        = agent.AsAny
	_ func(*provider.Client) map[string]any  = agent.BudgetEvent
	_ func(...string) []map[string]any       = agent.Defs
	_ func(*git.Repo) tools.Git              = agent.Git
	_ func(string) string                    = agent.Prompt
	_ func(string, map[string]string) string = agent.Render
	_ string                                 = agent.TestingRules
	_ func(*sys.Error) *sys.Error            = agent.WithOriginHint
)

// TestAgentPromptTestingRules pins agent.Prompt(agent.TestingRules), the text shape/context.go
// and shape/refine.go put in front of the model after their own heading.
func TestAgentPromptTestingRules(t *testing.T) {
	if agent.TestingRules != "testing-rules" {
		t.Errorf("agent.TestingRules = %q, want the prompt name testing-rules", agent.TestingRules)
	}
	text := agent.Prompt(agent.TestingRules)
	if !strings.HasPrefix(text, "# Testing rules\n\nThese rules decide which tests a change needs") {
		t.Errorf("agent.Prompt(agent.TestingRules) starts %q, want the testing rules heading and first sentence", text[:min(len(text), 80)])
	}
	if !strings.HasSuffix(text, "\n") {
		t.Errorf("agent.Prompt(agent.TestingRules) does not end with a newline: %q", text[max(0, len(text)-40):])
	}
	for _, want := range []string{"## 1. Identify what changed", "## 6. Specify the change so it can be built and tested", "## The reviewer"} {
		if !strings.Contains(text, want) {
			t.Errorf("agent.Prompt(agent.TestingRules) does not contain the section %q", want)
		}
	}
	if strings.Contains(text, "{{") {
		t.Errorf("agent.Prompt(agent.TestingRules) leaves a template variable unrendered")
	}
}

// TestAgentRender pins agent.Render as shape/policy.go uses it to fill the shaper's prompt
// templates: each {{name}} is replaced by its variable, anything else stays.
func TestAgentRender(t *testing.T) {
	vars := map[string]string{"feasibility-guidance": "FEAS", "readiness-acknowledgment-guidance": "READY"}
	for _, c := range []struct{ template, want string }{
		{"a {{feasibility-guidance}} b", "a FEAS b"},
		{"{{readiness-acknowledgment-guidance}}|{{feasibility-guidance}}|{{feasibility-guidance}}", "READY|FEAS|FEAS"},
		{"keep {{unknown}} and {{ feasibility-guidance }}", "keep {{unknown}} and {{ feasibility-guidance }}"},
		{"", ""},
	} {
		if got := agent.Render(c.template, vars); got != c.want {
			t.Errorf("agent.Render(%q) = %q, want %q", c.template, got, c.want)
		}
	}
	if got := agent.Render("x {{a}}\n", nil); got != "x {{a}}\n" {
		t.Errorf("agent.Render without variables = %q, want the template unchanged", got)
	}
	if got := agent.Render("x {{a}}", map[string]string{}); got != "x {{a}}" {
		t.Errorf("agent.Render with an empty map = %q, want the template unchanged", got)
	}
}

// TestAgentDefs pins agent.Defs as shape/contract.go calls it for the inspection tools: one
// definition per name, in order, each a function definition of that name; an unknown name
// panics.
func TestAgentDefs(t *testing.T) {
	names := []string{tools.ReadFileTool, tools.ListFilesTool, tools.SearchTextTool}
	defs := agent.Defs(names...)
	if len(defs) != len(names) {
		t.Fatalf("agent.Defs(%v) returned %d definitions, want %d", names, len(defs), len(names))
	}
	for i, d := range defs {
		if d["name"] != names[i] || d["type"] != "function" {
			t.Errorf("agent.Defs: definition %d = name %v, type %v, want function %s", i, d["name"], d["type"], names[i])
		}
		if desc, _ := d["description"].(string); desc == "" {
			t.Errorf("agent.Defs: the definition of %s has no description", names[i])
		}
		params, ok := d["parameters"].(map[string]any)
		if !ok || params["type"] != "object" {
			t.Errorf("agent.Defs: the definition of %s has parameters %v, want a JSON Schema object", names[i], d["parameters"])
		}
	}
	if got := agent.Defs(); len(got) != 0 {
		t.Errorf("agent.Defs() = %v, want no definitions", got)
	}
	for _, name := range []string{tools.ReadFileTool, "no_such_tool"} {
		func() {
			defer func() {
				recovered := recover()
				if (recovered != nil) != (name == "no_such_tool") {
					t.Errorf("agent.Defs(%q) panic = %v, want a panic only for an unknown name", name, recovered)
				}
			}()
			agent.Defs(name)
		}()
	}
}

// TestAgentAsAny pins agent.AsAny as shape/shape.go, shape/refine.go and shape/contract.go use
// it to make the Tools list of a provider request: the definitions flattened in order, each
// kept as the map it was.
func TestAgentAsAny(t *testing.T) {
	first := agent.Defs(tools.ReadFileTool, tools.ListFilesTool)
	second := agent.Defs(tools.SearchTextTool)
	got := agent.AsAny(first, second)
	want := []any{first[0], first[1], second[0]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("agent.AsAny(first, second) = %v, want the three definitions in order", got)
	}
	for i, item := range got {
		if _, ok := item.(map[string]any); !ok {
			t.Errorf("agent.AsAny: item %d is a %T, want map[string]any", i, item)
		}
	}
	if single := agent.AsAny(first); len(single) != 2 {
		t.Errorf("agent.AsAny(first) has %d items, want 2", len(single))
	}
	if none := agent.AsAny(); len(none) != 0 {
		t.Errorf("agent.AsAny() = %v, want none", none)
	}
}

// TestAgentBudgetEvent pins the run-log record shape/shape.go and shape/refine.go write once
// when a run starts: the client's response budget, or the library's 30 minutes.
func TestAgentBudgetEvent(t *testing.T) {
	for _, c := range []struct {
		name   string
		client *provider.Client
		ms     int64
		text   string
	}{
		{"nil client", nil, 30 * 60 * 1000, "30 minutes"},
		{"zero budget", &provider.Client{}, 30 * 60 * 1000, "30 minutes"},
		{"ten minutes", &provider.Client{ResponseBudget: 10 * time.Minute}, 10 * 60 * 1000, "10 minutes"},
		{"ninety seconds", &provider.Client{ResponseBudget: 90 * time.Second}, 90 * 1000, "1m30s"},
	} {
		want := map[string]any{"event": "response_budget", "budget_ms": c.ms, "budget": c.text}
		if got := agent.BudgetEvent(c.client); !reflect.DeepEqual(got, want) {
			t.Errorf("agent.BudgetEvent(%s) = %v, want %v", c.name, got, want)
		}
	}
}

// TestAgentWithOriginHint pins how shape/shape.go renders a provider failure: the refusal to
// send a saved key to another origin gains the line that names the login command; every
// other error is returned as it was.
func TestAgentWithOriginHint(t *testing.T) {
	const relogin = "log in again for that origin with: kogen provider login openai-compatible"
	refused := &sys.Error{Class: "provider", Reason: "refused", Detail: "the saved key belongs to https://a.example, not https://b.example", Lines: []string{"first"}, Exit: 4}

	got := agent.WithOriginHint(refused)
	if got == refused || !reflect.DeepEqual(got.Lines, []string{"first", relogin}) {
		t.Errorf("agent.WithOriginHint(origin refusal) lines = %q (same error: %v), want %q appended on a copy", got.Lines, got == refused, []string{"first", relogin})
	}
	if got.Class != "provider" || got.Reason != "refused" || got.Detail != refused.Detail || got.Exit != 4 {
		t.Errorf("agent.WithOriginHint(origin refusal) = %+v, want class, reason, detail and exit unchanged", *got)
	}
	if !reflect.DeepEqual(refused.Lines, []string{"first"}) {
		t.Errorf("agent.WithOriginHint changed the lines of its argument: %q", refused.Lines)
	}
	if want := "provider/refused: " + refused.Detail + "\n  first\n  " + relogin + "\n"; got.Render() != want {
		t.Errorf("agent.WithOriginHint(origin refusal).Render() = %q, want %q", got.Render(), want)
	}
	if again := agent.WithOriginHint(got); again != got {
		t.Errorf("agent.WithOriginHint of an error that already names the command returned a different error: %q", again.Lines)
	}
	if got := agent.WithOriginHint(nil); got != nil {
		t.Errorf("agent.WithOriginHint(nil) = %v, want nil", got)
	}
	for _, other := range []*sys.Error{
		{Class: "provider", Reason: "unavailable", Detail: "the saved key belongs to https://a.example"},
		{Class: "provider", Reason: "refused", Detail: "no origin sentence"},
		{Class: "executor", Reason: "refused", Detail: "the saved key belongs to https://a.example"},
	} {
		if got := agent.WithOriginHint(other); got != other || len(got.Lines) != 0 {
			t.Errorf("agent.WithOriginHint(%s/%s: %s) = %+v, want the same error without a hint", other.Class, other.Reason, other.Detail, *got)
		}
	}
}

// gitRepo makes a Git repository with one commit in a temporary directory.
func gitRepo(t *testing.T) (*git.Repo, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=consumes", "-c", "user.email=consumes@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("tracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "tracked.txt"},
		{"-c", "user.name=consumes", "-c", "user.email=consumes@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "tracked"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return git.New(dir, nil), dir
}

// TestAgentGit pins agent.Git as shape/shape.go and shape/refine.go give it to tools.Tools: a
// nil repository is a nil Git, and a real one runs, reports its HEAD and snapshots its tree
// like the repository itself.
func TestAgentGit(t *testing.T) {
	if g := agent.Git(nil); g != nil {
		t.Errorf("agent.Git(nil) = %v, want a nil tools.Git", g)
	}

	repo, dir := gitRepo(t)
	g := agent.Git(repo)
	if g == nil {
		t.Fatal("agent.Git(repo) = nil, want a tools.Git")
	}

	head := repo.Head()
	if len(head) != 40 {
		t.Fatalf("repository HEAD = %q, want a commit id", head)
	}
	if got := g.Head(); got != head {
		t.Errorf("agent.Git(repo).Head() = %q, want the repository's %q", got, head)
	}

	res, err := g.Run("rev-parse", "--is-inside-work-tree")
	if err != nil || res.Status != 0 || res.Stdout != "true\n" || res.Stderr != "" {
		t.Errorf("agent.Git(repo).Run(rev-parse) = %+v, %v, want status 0 and stdout true", res, err)
	}
	res, err = g.Run("rev-parse", "--verify", "no-such-ref")
	if err != nil || res.Status == 0 || res.Stderr == "" {
		t.Errorf("agent.Git(repo).Run of a failing command = %+v, %v, want a nonzero status and stderr without an error", res, err)
	}
	if direct, derr := repo.Run("rev-parse", "--show-toplevel"); derr != nil {
		t.Fatal(derr)
	} else if viaAdapter, aerr := g.Run("rev-parse", "--show-toplevel"); aerr != nil || fmt.Sprint(viaAdapter) != fmt.Sprint(direct) {
		t.Errorf("agent.Git(repo).Run(rev-parse --show-toplevel) = %+v, %v, want the repository's own %+v", viaAdapter, aerr, direct)
	}

	headTree, err := repo.OK("rev-parse", "HEAD^{tree}")
	if err != nil {
		t.Fatal(err)
	}
	headTree = strings.TrimSpace(headTree)
	if tree, err := g.Tree(head, nil); err != nil || tree != headTree {
		t.Errorf("agent.Git(repo).Tree(HEAD, nil) = %q, %v, want the clean tree %q", tree, err, headTree)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withNew, err := g.Tree(head, nil)
	if err != nil || withNew == headTree {
		t.Errorf("agent.Git(repo).Tree with an untracked file = %q, %v, want a tree different from %q", withNew, err, headTree)
	}
	if want, err := repo.Tree(head, nil); err != nil || withNew != want {
		t.Errorf("agent.Git(repo).Tree = %q, want the repository's own %q (%v)", withNew, want, err)
	}
	if excluded, err := g.Tree(head, []string{"new.txt"}); err != nil || excluded != headTree {
		t.Errorf("agent.Git(repo).Tree(HEAD, excluding new.txt) = %q, %v, want %q", excluded, err, headTree)
	}
	if _, err := g.Tree("--not-a-revision", nil); err == nil {
		t.Errorf("agent.Git(repo).Tree of an option-like revision succeeded, want an error")
	}
}
