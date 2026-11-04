package consumes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
	"kogen-loop/sys"
)

// Signatures kogen-shaping's calls rely on.
var (
	_ func(string) verify.Tools                                                        = verify.ToolsFromInstall
	_ func(*sys.Session, verify.Tools, string, *project.Config, bool) error            = verify.RequireTools
	_ func(context.Context, *sys.Session, verify.Tools, string, *project.Config) error = verify.RequireRunner
	_ string                                                                           = verify.Tools{}.Node
	_ string                                                                           = verify.Tools{}.Quint
	_ string                                                                           = verify.Tools{}.Cucumber
)

// verify.Tools is a comparable value of the three tool locations: kogen-shaping tells "no
// tools given" with `tools == (verify.Tools{})` and fills the fields from KOGEN_NODE,
// KOGEN_QUINT and KOGEN_CUCUMBER.
func TestTools(t *testing.T) {
	var none verify.Tools
	if none != (verify.Tools{}) {
		t.Error("verify.Tools: the zero value must equal verify.Tools{}")
	}
	given := verify.Tools{Node: "/n", Quint: "/q", Cucumber: "/c"}
	if given.Node != "/n" || given.Quint != "/q" || given.Cucumber != "/c" {
		t.Errorf("verify.Tools: Node, Quint and Cucumber must hold their locations, got %+v", given)
	}
	for name, other := range map[string]verify.Tools{
		"Tools.Node":     {Node: "/n2", Quint: "/q", Cucumber: "/c"},
		"Tools.Quint":    {Node: "/n", Quint: "/q2", Cucumber: "/c"},
		"Tools.Cucumber": {Node: "/n", Quint: "/q", Cucumber: "/c2"},
	} {
		if other == given || other == none {
			t.Errorf("verify.%s: a differing field must make the Tools differ", name)
		}
	}
}

// verify.ToolsFromInstall resolves Node from installRoot/node-path (the file's own path when
// it cannot be read) and Quint and Cucumber under installRoot/lib.
func TestToolsFromInstall(t *testing.T) {
	root := t.TempDir()
	wantQuint := filepath.Join(root, "lib/quint/node_modules/@informalsystems/quint/dist/src/cli.js")
	wantCucumber := filepath.Join(root, "lib/cucumber/node_modules")

	got := verify.ToolsFromInstall(root)
	want := verify.Tools{Node: filepath.Join(root, "node-path"), Quint: wantQuint, Cucumber: wantCucumber}
	if got != want {
		t.Errorf("verify.ToolsFromInstall: without a node-path file Node is that file's path:\n got: %+v\nwant: %+v", got, want)
	}

	write(t, root, "node-path", "/opt/node/bin/node\n")
	got = verify.ToolsFromInstall(root)
	want.Node = "/opt/node/bin/node"
	if got != want {
		t.Errorf("verify.ToolsFromInstall: Node is the trimmed text of node-path:\n got: %+v\nwant: %+v", got, want)
	}
}

// verify.RequireTools refuses with environment/tool_missing (exit 3) when Git or, for a
// Build, the program of a configured command is missing; nothing is executed.
func TestRequireTools(t *testing.T) {
	root := t.TempDir()
	s := session(root)

	if err := verify.RequireTools(s, verify.Tools{}, root, &project.Config{}, false); err != nil {
		t.Errorf("verify.RequireTools: git and /bin/sh are present, got %v", err)
	}

	missing := &project.Config{Setup: []string{"FOO=1 BAR=2 (no-such-program-xyz --flag)"}}
	if err := verify.RequireTools(s, verify.Tools{}, root, missing, false); err != nil {
		t.Errorf("verify.RequireTools: configured commands are not looked up unless configured is true, got %v", err)
	}
	err := verify.RequireTools(s, verify.Tools{}, root, missing, true)
	e := sysError(t, err, "environment", "tool_missing")
	if e.Exit != 3 || e.Detail != "no-such-program-xyz is not available" {
		t.Errorf("verify.RequireTools: want exit 3 \"no-such-program-xyz is not available\" (assignments and the subshell paren skipped), got exit %d %q", e.Exit, e.Detail)
	}

	present := &project.Config{
		Setup:  []string{"FOO=1 (sh -c true)"},
		Checks: []project.Check{{Name: "vet", Run: "! git status"}},
	}
	if err := verify.RequireTools(s, verify.Tools{}, root, present, true); err != nil {
		t.Errorf("verify.RequireTools: programs found with command -v after assignments, grouping and negation are fine, got %v", err)
	}

	// The generated runner's "$KOGEN_NODE" resolves to the Node the tools record.
	node := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(node, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &project.Config{Acceptance: &project.Acceptance{Run: `NODE_PATH="$KOGEN_CUCUMBER" "$KOGEN_NODE" cucumber.js`}}
	if err := verify.RequireTools(s, verify.Tools{Node: node}, root, runner, true); err != nil {
		t.Errorf("verify.RequireTools: \"$KOGEN_NODE\" must resolve to Tools.Node, got %v", err)
	}
	err = verify.RequireTools(s, verify.Tools{Node: filepath.Join(root, "missing-node")}, root, runner, true)
	if e := sysError(t, err, "environment", "tool_missing"); e.Detail != "$KOGEN_NODE is not available" {
		t.Errorf("verify.RequireTools: a missing Tools.Node must name $KOGEN_NODE, got %q", e.Detail)
	}
}

// verify.RequireRunner runs the acceptance command on a feature with no scenarios and
// refuses with environment/acceptance_unavailable (exit 3) unless it exits 0 and writes its
// report; the command gets the tools as KOGEN_NODE, KOGEN_QUINT and KOGEN_CUCUMBER.
func TestRequireRunner(t *testing.T) {
	isolateTmp(t)
	root := t.TempDir()
	tools := verify.Tools{Node: "/tools/node", Quint: "/tools/quint", Cucumber: "/tools/cucumber"}
	cfg := func(run string) *project.Config { return &project.Config{Acceptance: &project.Acceptance{Run: run}} }
	ctx := context.Background()

	s := session(root)
	runs := `test "$KOGEN_NODE|$KOGEN_QUINT|$KOGEN_CUCUMBER" = "/tools/node|/tools/quint|/tools/cucumber" && test -f "$KOGEN_ACCEPTANCE_FEATURES" && touch "$KOGEN_ACCEPTANCE_REPORT"`
	if err := verify.RequireRunner(ctx, s, tools, root, cfg(runs)); err != nil {
		t.Errorf("verify.RequireRunner: a runner that exits 0 and writes its report must pass, got %v", err)
	}
	if !strings.Contains(s.Stderr.(*strings.Builder).String(), "kogen: checking the acceptance runner") {
		t.Errorf("verify.RequireRunner: progress must say the runner is checked, got %q", s.Stderr.(*strings.Builder).String())
	}

	err := verify.RequireRunner(ctx, session(root), tools, root, cfg("echo broken runner; exit 3"))
	e := sysError(t, err, "environment", "acceptance_unavailable")
	if want := "the runner check failed with status 3; fix commands.acceptance.run in .kogen/project.yaml"; e.Exit != 3 || e.Detail != want || !equalStrings(e.Lines, []string{"broken runner"}) {
		t.Errorf("verify.RequireRunner: want exit 3 %q with the output as lines, got exit %d %q %q", want, e.Exit, e.Detail, e.Lines)
	}

	err = verify.RequireRunner(ctx, session(root), tools, root, cfg("echo no report"))
	e = sysError(t, err, "environment", "acceptance_unavailable")
	if want := "the runner check wrote no report; fix commands.acceptance.run in .kogen/project.yaml"; e.Detail != want {
		t.Errorf("verify.RequireRunner: a runner without a report must refuse with %q, got %q", want, e.Detail)
	}

	err = verify.RequireRunner(ctx, session(root), tools, root, cfg("no-such-program-xyz"))
	if e := sysError(t, err, "environment", "tool_missing"); e.Detail != "no-such-program-xyz is not available" {
		t.Errorf("verify.RequireRunner: a missing runner program is refused first as tool_missing, got %q", e.Detail)
	}
}
