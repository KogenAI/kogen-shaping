//go:build ignore

package shape

import (
	"context"
	"encoding/json"
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

// The scripted executor performs no download or model work. Only its first main
// Start prepares the runtime; later check sessions reuse that runtime.
type stiExecutor struct {
	*shapeExec
	clock            *budgetClock
	install, startup time.Duration
	warm             bool
}

func (x *stiExecutor) Start(ctx context.Context, task executor.Task, emit executor.Sink) (executor.Session, error) {
	if filepath.Base(task.Dir) == mainSession {
		if !x.warm {
			x.clock.Advance(x.install)
			p := claudecode.RuntimePath(task.Home)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				return nil, err
			}
			if err := os.WriteFile(p, []byte("scripted installed runtime\n"), 0700); err != nil {
				return nil, err
			}
		} else {
			x.clock.Advance(x.startup)
		}
	}
	return x.shapeExec.Start(ctx, task, emit)
}

func TestShapeRuntimeInstallObservation(t *testing.T) {
	var spec struct {
		Install, Shaper, Checks, Startup int64
		Warm                             bool
		Mode                             string
	}
	if err := json.Unmarshal([]byte(os.Getenv("STI_SPEC")), &spec); err != nil {
		t.Fatal(err)
	}
	f := newShpFx(t)
	f.useExternalShape()
	clock := newBudgetClock()
	if spec.Warm {
		p := claudecode.RuntimePath(f.Home)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("scripted installed runtime\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	base := &shapeExec{t: t}
	first := shpFiles()
	if spec.Mode == "refuse" {
		first = map[string]any{"files": []any{map[string]any{"path": "../outside.txt", "content": "x\n"}}}
	}
	base.rounds = []func(int, string) (executor.RoundResult, error){
		func(n int, root string) (executor.RoundResult, error) {
			clock.Advance(time.Duration(spec.Shaper) * time.Millisecond)
			return result(first)(n, root)
		},
		result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}),
	}
	checkCalls := 0
	base.contract = []func(int, string) (executor.RoundResult, error){func(n int, root string) (executor.RoundResult, error) {
		if checkCalls == 0 {
			clock.Advance(time.Duration(spec.Checks) * time.Millisecond)
		}
		checkCalls++
		return result(map[string]any{"findings": []any{}, "environment": []any{}})(n, root)
	}}
	x := &stiExecutor{shapeExec: base, clock: clock, install: time.Duration(spec.Install) * time.Millisecond, startup: time.Duration(spec.Startup) * time.Millisecond, warm: spec.Warm}
	f.newSession()
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
		Request: "Add calc.is_even(n). Non-integers raise TypeError.", Client: f.Client, Now: clock.Now,
		External: func(project.Role) (executor.Executor, error) { return x, nil }})
	obs := map[string]any{"exit": exit, "rounds": len(base.inputs), "stderr": f.Stderr.String(), "error": "", "lines": []string{}}
	if err != nil {
		obs["error"] = err.Error()
		if e, ok := sys.AsError(err); ok {
			obs["class"], obs["reason"], obs["detail"], obs["lines"] = e.Class, e.Reason, e.Detail, e.Lines
		}
	}
	obs["accounts"] = shpNamed(f.events(), "shape_time")
	obs["feedback"] = base.inputs
	obs["time_lines"] = []string{}
	for _, l := range strings.Split(f.Stderr.String(), "\n") {
		if strings.HasPrefix(l, "kogen: shape time ") {
			obs["time_lines"] = append(obs["time_lines"].([]string), l)
		}
	}
	b, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("STI_OUTPUT"), b, 0600); err != nil {
		t.Fatal(err)
	}
	fmt.Println("runtime observation written")
}
