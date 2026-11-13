package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-shaping/shape"
)

// fixture is a pinned project checkout with one commit and n request files.
func fixture(t *testing.T, names ...string) (project string, requests []string) {
	t.Helper()
	dir := t.TempDir()
	project = filepath.Join(dir, "project")
	os.MkdirAll(project, 0o755)
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "pin"}} {
		if out, err := exec.Command("git", append([]string{"-C", project}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	for _, n := range names {
		p := filepath.Join(dir, n)
		os.WriteFile(p, []byte("# REQUEST: "+n+"\n\nChange something.\n"), 0o444)
		requests = append(requests, p)
	}
	return project, requests
}

// fakeProvider plays the shaper and its gates: it records the events a real shaping records
// (provider_call per model request, submission_refused per refused gate) and then returns
// what shape.Shape returns.
type fakeProvider struct {
	calls   int
	refused [][2]any // attempt, gate
	exit    int
	err     error
	block   bool // wait for the context, as a shaping that never finishes
}

func (f fakeProvider) shaper() Shaper {
	return func(ctx context.Context, s *sys.Session, in shape.ShapeInput) (int, error) {
		dir := filepath.Join(in.Project, ".kogen/local/shaping", in.Slug)
		os.MkdirAll(dir, 0o755)
		file, _ := os.OpenFile(filepath.Join(dir, "events.ndjson"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		defer file.Close()
		for i := 0; i < f.calls; i++ {
			fmt.Fprintln(file, `{"event":"provider_call","role":"shape"}`)
		}
		for _, r := range f.refused {
			b, _ := json.Marshal(map[string]any{"event": "submission_refused", "attempt": r[0], "gate": r[1]})
			fmt.Fprintf(file, "%s\n", b)
		}
		if f.block {
			<-ctx.Done()
			return 0, ctx.Err()
		}
		return f.exit, f.err
	}
}

func run(t *testing.T, f fakeProvider, cfg Config, names ...string) []Result {
	t.Helper()
	project, reqs := fixture(t, names...)
	cfg.Project, cfg.Requests, cfg.Out, cfg.Shaper = project, reqs, filepath.Join(t.TempDir(), "out"), f.shaper()
	if cfg.Poll == 0 {
		cfg.Poll = 10 * time.Millisecond
	}
	results, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return results
}

func TestApprovedAfterBatchedRefusals(t *testing.T) {
	r := run(t, fakeProvider{calls: 4, refused: [][2]any{{1, "decision_check"}, {1, "literal_check"}, {2, "contract_review"}}}, Config{Label: "main"}, "add-a-thing.md")[0]
	if r.Outcome != "approved" || r.Rounds != 3 || r.ProviderRequests != 4 || r.Exit != 0 {
		t.Fatalf("got %+v", r)
	}
	if r.Gates["decision_check"] != 1 || r.Gates["literal_check"] != 1 || r.Gates["contract_review"] != 1 || r.LastGate != "contract_review" {
		t.Fatalf("gates %+v last %q", r.Gates, r.LastGate)
	}
	if strings.Join(r.GateOrder, ",") != "decision_check,literal_check,contract_review" || r.WallS < 0 || r.Label != "main" || r.Slug != "add-a-thing" {
		t.Fatalf("got %+v", r)
	}
}

func TestOutcomes(t *testing.T) {
	for _, c := range []struct {
		name string
		f    fakeProvider
		want string
		exit int
	}{
		{"round limit", fakeProvider{calls: 3, refused: [][2]any{{1, "decision_check"}, {2, "decision_check"}, {3, "scope"}}, exit: 1, err: sys.Fail("shape", "invalid_output", "the shaper's output was not a valid Intent after 3 attempts", 1)}, "round_limit", 1},
		{"not ready", fakeProvider{calls: 2, refused: [][2]any{{1, "validation"}}, exit: 1, err: sys.Fail("shape", "not_ready", "the Intent is not ready to build", 1)}, "refused:not_ready", 1},
		{"questions", fakeProvider{calls: 1, exit: 5}, "questions", 5},
		{"shape budget", fakeProvider{calls: 9, err: sys.Fail("shape", "invalid_output", "the shaper used its 15-minute shaping budget (15m elapsed, 9 requests) without a submission that passed; shaping stopped", 1)}, "time_cap", 1},
		{"provider", fakeProvider{err: sys.Fail("shape", "provider_failed", "shaping stopped on a provider error", 4)}, "refused:provider_failed", 4},
		{"other error", fakeProvider{err: fmt.Errorf("disk full")}, "error:disk full", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := run(t, c.f, Config{}, "some-request.md")[0]
			if r.Outcome != c.want || r.Exit != c.exit {
				t.Fatalf("got %+v, want %s exit %d", r, c.want, c.exit)
			}
		})
	}
}

func TestTimeCap(t *testing.T) {
	start := time.Now()
	r := run(t, fakeProvider{calls: 1, block: true}, Config{Cap: 150 * time.Millisecond}, "never-finishes.md")[0]
	if r.Outcome != "time_cap" || r.ProviderRequests != 1 || time.Since(start) > 10*time.Second {
		t.Fatalf("got %+v after %v", r, time.Since(start))
	}
}

func TestCallBudgetStopsTheSeries(t *testing.T) {
	// 2 calls per request, 3 allowed: the second request still starts (1 call left), the third never does.
	rs := run(t, fakeProvider{calls: 2, block: false}, Config{MaxCalls: 3}, "first-one.md", "second-one.md", "third-one.md")
	if len(rs) != 2 || rs[0].Outcome != "approved" {
		t.Fatalf("got %+v", rs)
	}
}

func TestBudgetCutsARunningShaping(t *testing.T) {
	rs := run(t, fakeProvider{calls: 5, block: true}, Config{MaxCalls: 3, Cap: time.Minute}, "first-one.md", "second-one.md")
	if len(rs) != 1 || rs[0].Outcome != "budget" || rs[0].ProviderRequests != 5 {
		t.Fatalf("got %+v", rs)
	}
}

func TestRunsStartFromThePinnedCommit(t *testing.T) {
	project, reqs := fixture(t, "pinned-one.md")
	pin := strings.TrimSpace(gitOut(t, project, "rev-parse", "HEAD"))
	gitOut(t, project, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "later")
	var head string
	shaper := func(ctx context.Context, s *sys.Session, in shape.ShapeInput) (int, error) {
		head = strings.TrimSpace(gitOut(t, in.Project, "rev-parse", "HEAD"))
		return 0, nil
	}
	if _, err := Run(context.Background(), Config{Project: project, Commit: pin, Out: t.TempDir(), Requests: reqs, Shaper: shaper}); err != nil {
		t.Fatal(err)
	}
	if head != pin {
		t.Fatalf("shaped at %s, pinned %s", head, pin)
	}
	if out := gitOut(t, project, "worktree", "list"); strings.Count(out, "\n") != 1 {
		t.Fatalf("worktree left behind: %s", out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return string(out)
}

func TestSlugOf(t *testing.T) {
	for in, want := range map[string]string{
		"/x/Add-Thing_to.Some-request.md":       "add-thing-to-some-request",
		"/x/ab.md":                              "abx",
		"/x/" + strings.Repeat("a", 60) + ".md": strings.Repeat("a", 48),
	} {
		if got := SlugOf(in); got != want {
			t.Errorf("%s: %q, want %q", in, got, want)
		}
	}
}
