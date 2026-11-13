// shapebench replays recorded requests through the Shape stage only (shape.Shape, never
// kogen intent build) against a pinned project checkout and writes one JSON line per request:
// outcome, rounds, provider requests, wall time and the refusal gates hit. It compiles
// against any kogen-shaping commit whose shape.ShapeInput has Project, Slug, Request, so the
// same patch measures an old and a new commit.
//
//	shapebench --project <checkout> [--commit <rev>] --out <dir> [--cap 15m] [--max-calls N] [--label L] request.md...
//
// The shape model is whatever the checkout's .kogen/models.yaml names; the login comes from
// $HOME. Each request runs in its own detached git worktree of --commit (default HEAD) under --out, so
// runs never share state and the checkout itself is left as it was.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-executor/executor/claudecode"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-shaping/shape"
	"github.com/KogenAI/kogen-verification/verify"
)

// Shaper is shape.Shape; tests replace it with a fake that plays a provider.
type Shaper func(ctx context.Context, s *sys.Session, in shape.ShapeInput) (int, error)

// Result is one line of results.ndjson.
type Result struct {
	Label            string         `json:"label,omitempty"`
	Request          string         `json:"request"`
	Slug             string         `json:"slug"`
	Outcome          string         `json:"outcome"` // approved, refused:<reason>, round_limit, time_cap, questions, budget, error:<what>
	Detail           string         `json:"detail,omitempty"`
	Exit             int            `json:"exit"`
	Rounds           int            `json:"rounds"`
	ProviderRequests int            `json:"provider_requests"`
	WallS            float64        `json:"wall_s"`
	Gates            map[string]int `json:"gates"`      // refusals per gate, every gate of every refused submission
	LastGate         string         `json:"last_gate"`  // the gate of the last refusal
	GateOrder        []string       `json:"gate_order"` // gates in the order refused
}

// Config is one bench series.
type Config struct {
	Project  string
	Commit   string
	Out      string
	Label    string
	Cap      time.Duration
	MaxCalls int // provider requests for the whole series; 0 is unlimited
	Requests []string
	Shaper   Shaper
	Poll     time.Duration
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// SlugOf is a request file's slug: lowercase letters, digits and dashes, 3 to 48 long.
func SlugOf(path string) string {
	s := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))), "-"), "-")
	if len(s) > 48 {
		s = strings.Trim(s[:48], "-")
	}
	for len(s) < 3 {
		s += "x"
	}
	return s
}

func git(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Run shapes every request in turn and appends a Result per request to <Out>/results.ndjson.
// It stops starting new requests once MaxCalls provider requests are spent.
func Run(ctx context.Context, cfg Config) ([]Result, error) {
	if cfg.Shaper == nil {
		cfg.Shaper = shape.Shape
	}
	if cfg.Cap <= 0 {
		cfg.Cap = 15 * time.Minute
	}
	if cfg.Poll <= 0 {
		cfg.Poll = 500 * time.Millisecond
	}
	if cfg.Commit == "" {
		cfg.Commit = "HEAD"
	}
	if err := os.MkdirAll(cfg.Out, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(cfg.Out, "results.ndjson"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var results []Result
	spent := 0
	for _, req := range cfg.Requests {
		if cfg.MaxCalls > 0 && spent >= cfg.MaxCalls {
			break
		}
		r, err := runOne(ctx, cfg, req, cfg.MaxCalls-spent)
		if err != nil {
			return results, err
		}
		spent += r.ProviderRequests
		results = append(results, r)
		b, _ := json.Marshal(r)
		fmt.Fprintf(f, "%s\n", b)
	}
	return results, nil
}

func runOne(ctx context.Context, cfg Config, reqPath string, remaining int) (Result, error) {
	res := Result{Label: cfg.Label, Request: filepath.Base(reqPath), Slug: SlugOf(reqPath), Gates: map[string]int{}}
	data, err := os.ReadFile(reqPath)
	if err != nil {
		return res, err
	}
	if !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return res, fmt.Errorf("%s: request must be nonempty UTF-8 text", reqPath)
	}
	tmp, err := os.MkdirTemp(cfg.Out, ".work-")
	if err != nil {
		return res, err
	}
	work := filepath.Join(tmp, "work")
	if err := git(cfg.Project, "worktree", "add", "--detach", work, cfg.Commit); err != nil {
		os.RemoveAll(tmp)
		return res, err
	}
	defer func() {
		git(cfg.Project, "worktree", "remove", "--force", work)
		os.RemoveAll(tmp)
	}()
	events := filepath.Join(work, ".kogen/local/shaping", res.Slug, "events.ndjson")

	runCtx, cancel := context.WithTimeout(ctx, cfg.Cap)
	defer cancel()
	var overBudget atomic.Bool
	done := make(chan struct{})
	if remaining > 0 {
		go func() { // stop this request when the series' provider-request budget is spent
			t := time.NewTicker(cfg.Poll)
			defer t.Stop()
			for {
				select {
				case <-done:
					return
				case <-t.C:
					if n := parseEvents(events).calls; n >= remaining {
						overBudget.Store(true)
						cancel()
						return
					}
				}
			}
		}()
	}
	s := sys.New()
	s.Stdout = os.Stderr
	s.Cwd = work
	start := time.Now()
	exit, runErr := cfg.Shaper(runCtx, s, shape.ShapeInput{Project: work, Slug: res.Slug, Request: string(data),
		External: externalExecutor,
		Tools:    verify.Tools{Node: s.Getenv("KOGEN_NODE"), Quint: s.Getenv("KOGEN_QUINT"), Cucumber: s.Getenv("KOGEN_CUCUMBER")}})
	res.WallS = time.Since(start).Seconds()
	close(done)
	capHit := errors.Is(runCtx.Err(), context.DeadlineExceeded)

	ev := parseEvents(events)
	res.ProviderRequests, res.Rounds, res.Gates, res.GateOrder, res.LastGate = ev.calls, ev.rounds, ev.gates, ev.order, ev.last
	res.Exit = exit
	var se *sys.Error
	if errors.As(runErr, &se) {
		res.Detail, res.Exit = se.Detail, se.Exit
	}
	switch {
	case overBudget.Load():
		res.Outcome = "budget"
	case runErr == nil && exit == 0:
		res.Outcome = "approved"
		res.Rounds++ // the accepted submission
	case runErr == nil && exit == 5:
		res.Outcome = "questions"
	case capHit || (runErr != nil && strings.Contains(runErr.Error(), "shaping budget")) || (se != nil && strings.Contains(se.Detail, "shaping budget")):
		res.Outcome = "time_cap"
	case se != nil:
		switch {
		case se.Class == "shape" && (se.Reason == "invalid_output" || se.Reason == "incomplete_package"):
			res.Outcome = "round_limit"
		default:
			res.Outcome = "refused:" + se.Reason
		}
	case runErr != nil:
		res.Outcome, res.Detail = "error:"+firstLine(runErr.Error()), runErr.Error()
	default:
		res.Outcome = fmt.Sprintf("refused:exit_%d", exit)
	}
	if data, err := os.ReadFile(events); err == nil {
		os.WriteFile(filepath.Join(cfg.Out, res.Slug+"."+safe(cfg.Label)+".events.ndjson"), data, 0o644)
	}
	return res, nil
}

func safe(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func firstLine(s string) string { return strings.SplitN(s, "\n", 2)[0] }

type evSummary struct {
	calls, rounds int
	gates         map[string]int
	order         []string
	last          string
}

// parseEvents reads a shaping's events.ndjson: provider_call events are the provider
// requests, submission_refused events carry the attempt (round) and the gate refused.
func parseEvents(path string) evSummary {
	sum := evSummary{gates: map[string]int{}}
	f, err := os.Open(path)
	if err != nil {
		return sum
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			var e struct {
				Event   string  `json:"event"`
				Gate    string  `json:"gate"`
				Attempt float64 `json:"attempt"`
			}
			if json.Unmarshal(line, &e) == nil {
				switch e.Event {
				case "provider_call", "executor_round_started":
					sum.calls++ // a native model request, or one Claude Code round of the shaper
				case "submission_refused":
					sum.gates[e.Gate]++
					sum.order = append(sum.order, e.Gate)
					sum.last = e.Gate
					sum.rounds = max(sum.rounds, int(e.Attempt))
				}
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				break
			}
			break
		}
	}
	return sum
}

// GateList is the gates of a Result sorted by count then name, for the summary line.
func GateList(g map[string]int) string {
	var names []string
	for n := range g {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		return g[names[i]] > g[names[j]] || g[names[i]] == g[names[j]] && names[i] < names[j]
	})
	for i, n := range names {
		names[i] = fmt.Sprintf("%s×%d", n, g[n])
	}
	return strings.Join(names, ",")
}

// externalExecutor is kogen's own wiring of the claude-code executor (cli/engine.go): the pinned
// runtime under $HOME/.kogen/claude-code on the saved token, with this binary as its hook.
func externalExecutor(role project.Role) (executor.Executor, error) {
	if role.Executor != project.ExecutorClaudeCode {
		return nil, executor.Fail(executor.ReasonUnavailable, "executor "+role.Executor+" is not available in shapebench")
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, executor.Fail(executor.ReasonUnavailable, "cannot locate shapebench for the executor hooks: "+err.Error())
	}
	home, _ := os.UserHomeDir()
	token, _ := provider.ClaudeCodeToken(home)
	return claudecode.New(claudecode.Options{Home: home, HookCommand: []string{exe, claudecode.HookSubcommand}, Token: token}), nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == claudecode.HookSubcommand {
		os.Exit(claudecode.RunHook(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
	}
	fs := flag.NewFlagSet("shapebench", flag.ContinueOnError)
	project := fs.String("project", "", "pinned project checkout (a git repository)")
	commit := fs.String("commit", "", "project revision each run starts from (default HEAD)")
	out := fs.String("out", "", "output directory for results.ndjson and per-request events")
	label := fs.String("label", "", "label stored in every result (for example the kogen-shaping commit)")
	capd := fs.Duration("cap", 15*time.Minute, "wall-time cap per request")
	maxCalls := fs.Int("max-calls", 0, "provider requests for the whole series; 0 is unlimited")
	if err := fs.Parse(os.Args[1:]); err != nil || *project == "" || *out == "" || fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: shapebench --project <checkout> [--commit rev] --out <dir> [--label L] [--cap 15m] [--max-calls N] request.md...")
		os.Exit(2)
	}
	abs := func(p string) string { a, _ := filepath.Abs(p); return a }
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	results, err := Run(ctx, Config{Project: abs(*project), Commit: *commit, Out: abs(*out), Label: *label, Cap: *capd, MaxCalls: *maxCalls, Requests: fs.Args()})
	for _, r := range results {
		fmt.Printf("%-48s %-28s rounds=%d requests=%d wall=%.0fs gates=%s\n", r.Slug, r.Outcome, r.Rounds, r.ProviderRequests, r.WallS, GateList(r.Gates))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "shapebench:", err)
		os.Exit(1)
	}
}
