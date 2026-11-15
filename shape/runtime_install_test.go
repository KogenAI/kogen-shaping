package shape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-executor/executor/claudecode"
	"github.com/KogenAI/kogen-loop/sys"
)

// runtimeInstallExec prepares a local runtime marker instead of downloading a runtime.
// The scripted rounds and fake clock still run through Shape's real accounting path.
type runtimeInstallExec struct {
	*shapeExec
	clock       *budgetClock
	preparation time.Duration
}

func (x *runtimeInstallExec) Start(ctx context.Context, task executor.Task, sink executor.Sink) (executor.Session, error) {
	if filepath.Base(task.Dir) == mainSession {
		x.clock.Advance(x.preparation)
		p := claudecode.RuntimePath(task.Home)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, []byte("scripted runtime\n"), 0700); err != nil {
			return nil, err
		}
	}
	return x.shapeExec.Start(ctx, task, sink)
}

func TestShapeRuntimeInstallAccounting(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		warm                        bool
		preparation, shaper, checks int64
		refuse                      bool
		progress                    string
	}{
		{name: "historical cold replay", preparation: 115939, shaper: 280916, checks: 511168,
			progress: "kogen: shape time 908.0 s: shaper 280.9 s (model unknown, tools unknown), Kogen checks 511.2 s (static 0.0 s, model checks 0.0 s, contract review 511.2 s, readiness 0.0 s), gaps 0.0 s, runtime install 115.9 s"},
		{name: "warm startup", warm: true, preparation: 2000, shaper: 280916, checks: 511168,
			progress: "kogen: shape time 794.1 s: shaper 280.9 s (model unknown, tools unknown), Kogen checks 511.2 s (static 0.0 s, model checks 0.0 s, contract review 511.2 s, readiness 0.0 s), gaps 2.0 s"},
		{name: "cold inside boundary", preparation: 115939, shaper: 899999,
			progress: "kogen: shape time 1015.9 s: shaper 900.0 s (model unknown, tools unknown), Kogen checks 0.0 s (static 0.0 s, model checks 0.0 s, contract review 0.0 s, readiness 0.0 s), gaps 0.0 s, runtime install 115.9 s"},
		{name: "cold boundary", preparation: 115939, shaper: 900000, refuse: true,
			progress: "kogen: shape time 1015.9 s: shaper 900.0 s (model unknown, tools unknown), Kogen checks 0.0 s (static 0.0 s, model checks 0.0 s, contract review 0.0 s, readiness 0.0 s), gaps 0.0 s, runtime install 115.9 s"},
		{name: "warm boundary", warm: true, shaper: 900000, refuse: true,
			progress: "kogen: shape time 900.0 s: shaper 900.0 s (model unknown, tools unknown), Kogen checks 0.0 s (static 0.0 s, model checks 0.0 s, contract review 0.0 s, readiness 0.0 s), gaps 0.0 s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newShpFx(t)
			f.useExternalShape()
			clock := newBudgetClock()
			if tc.warm {
				p := claudecode.RuntimePath(f.Home)
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("warm runtime\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			base := &shapeExec{t: t}
			first := shpFiles()
			if tc.refuse {
				first = map[string]any{"files": []any{map[string]any{"path": "../outside.txt", "content": "x\n"}}}
			}
			base.rounds = []func(int, string) (executor.RoundResult, error){
				func(n int, root string) (executor.RoundResult, error) {
					clock.Advance(time.Duration(tc.shaper) * time.Millisecond)
					return result(first)(n, root)
				},
				result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}),
			}
			checkCalls := 0
			base.contract = []func(int, string) (executor.RoundResult, error){func(n int, root string) (executor.RoundResult, error) {
				if checkCalls == 0 {
					clock.Advance(time.Duration(tc.checks) * time.Millisecond)
				}
				checkCalls++
				return result(map[string]any{"findings": []any{}, "environment": []any{}})(n, root)
			}}
			x := &runtimeInstallExec{shapeExec: base, clock: clock, preparation: time.Duration(tc.preparation) * time.Millisecond}
			f.newSession()
			exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
				Request: "Add calc.is_even(n). Non-integers raise TypeError.", Client: f.Client, Now: clock.Now,
				External: func(project.Role) (executor.Executor, error) { return x, nil }})
			install, gap := tc.preparation, int64(0)
			if tc.warm {
				install, gap = 0, tc.preparation
			}
			if tc.refuse {
				shpEq(t, "exit", exit, 1)
				shpEq(t, "rounds", len(base.inputs), 1)
				e, ok := sys.AsError(err)
				if !ok {
					t.Fatalf("expected budget refusal: %v", err)
				}
				shpEq(t, "class", e.Class, "shape")
				shpEq(t, "reason", e.Reason, "invalid_output")
				shpEq(t, "detail", e.Detail, "the shaper used its 15-minute shaping budget (15 min elapsed, 1 requests) without a submission that passed; shaping stopped")
				suffix := ""
				if install > 0 {
					suffix = fmt.Sprintf(", install_ms=%d", install)
				}
				for _, want := range []string{
					budgetFigureLine + suffix,
					"attempt 1: gate validation, model request 900 s, Kogen checks 0 s, 0 s remaining" + suffix,
				} {
					found := false
					for _, line := range e.Lines {
						if line == want {
							found = true
						}
					}
					if !found {
						t.Fatalf("missing exact refusal line %q in %v", want, e.Lines)
					}
				}
			} else {
				if err != nil || exit != 0 {
					t.Fatalf("exit %d: %v", exit, err)
				}
				shpEq(t, "rounds", len(base.inputs), 2)
			}
			accounts := shpNamed(f.events(), "shape_time")
			shpEq(t, "accounts", len(accounts), 1)
			a := accounts[0]
			for k, v := range map[string]int64{"elapsed_ms": tc.preparation + tc.shaper + tc.checks, "shaper_ms": tc.shaper, "checks_ms": tc.checks, "install_ms": install, "gap_ms": gap, "readiness_ms": 0} {
				shpEq(t, k, a[k], any(float64(v)))
				if a[k].(float64) < 0 {
					t.Fatalf("negative %s: %v", k, a[k])
				}
			}
			shpEq(t, "identity", a["gap_ms"].(float64), a["elapsed_ms"].(float64)-a["shaper_ms"].(float64)-a["checks_ms"].(float64)-a["install_ms"].(float64))
			shpEq(t, "unknown model", a["shaper_model_ms"], nil)
			shpEq(t, "unknown tools", a["shaper_tools_ms"], nil)
			var lines []string
			for _, line := range strings.Split(f.Stderr.String(), "\n") {
				if strings.HasPrefix(line, "kogen: shape time ") {
					lines = append(lines, line)
				}
			}
			shpEq(t, "progress count", len(lines), 1)
			shpEq(t, "progress", lines[0], tc.progress)
		})
	}
}
