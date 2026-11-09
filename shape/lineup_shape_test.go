package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
)

// shapeExec is a scripted Claude Code shaper: no process, no model. Each Start is a fresh
// session; each round is a function that may touch the checkout and returns the structured
// result. It records the tasks and the feedback it was given.
type shapeExec struct {
	t        *testing.T
	mu       sync.Mutex
	rounds   []func(n int, root string) (executor.RoundResult, error)
	starts   []executor.Task
	inputs   []executor.Input
	startErr error
	seq      int
	calls    int  // rounds over every session
	bash     bool // the main shaper session completes a Bash call in its first round
	noRead   bool // the main shaper session completes no Read call in its first round
	cap      int  // the executor's own shape-role round cap; 0: 6
	// contract, when set, answers the contract review and confirmation sessions in order (the
	// last repeats); they do not count in rounds or calls, and starts holds only the other
	// sessions. Without it every review finds no contradiction.
	contract       []func(n int, root string) (executor.RoundResult, error)
	contractCalls  int // contract review and confirmation rounds
	contractStarts []executor.Task
	// decision, when set, answers the decision check sessions in order (the last repeats); they
	// do not count in rounds or calls, and starts holds only the other sessions. Without it every
	// check has no item.
	decision       []func(n int, root string) (executor.RoundResult, error)
	decisionCalls  int // decision check rounds
	decisionStarts []executor.Task
	// literal, when set, answers the literal check sessions in order (the last repeats); they
	// do not count in rounds or calls, and starts holds only the other sessions. Without it every
	// check has no item.
	literal       []func(n int, root string) (executor.RoundResult, error)
	literalCalls  int // literal check rounds
	literalStarts []executor.Task
}

// isContractTask is whether a session is a contract review or confirmation.
func isContractTask(t executor.Task) bool {
	return strings.Contains(string(t.Schema), `"findings"`) || strings.Contains(string(t.Schema), `"verdicts"`)
}

// isLiteralTask is whether a session is a literal check.
func isLiteralTask(t executor.Task) bool {
	return strings.Contains(string(t.Schema), `"source_quote"`)
}

// isDecisionTask is whether a session is a decision check.
func isDecisionTask(t executor.Task) bool {
	return strings.Contains(string(t.Schema), `"request_quotes"`)
}

func (x *shapeExec) Describe() executor.Descriptor {
	rc := executor.RoleCapabilities{Resume: executor.ResumeNone, Continuation: true, Usage: executor.UsageHarness, Structured: true, MaxRounds: x.cap,
		Limitations: []string{"no Shape workers", "no Kogen scratch probe"}}
	return executor.Descriptor{Name: executor.ClaudeCode, Contract: executor.Contract, Version: "claude-code 9.9.9",
		RoleCapabilities: map[string]executor.RoleCapabilities{executor.RoleShape: rc}}
}

func (x *shapeExec) Start(_ context.Context, t executor.Task, emit executor.Sink) (executor.Session, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.startErr != nil {
		return nil, x.startErr
	}
	if isContractTask(t) {
		x.contractStarts = append(x.contractStarts, t)
	} else if isDecisionTask(t) {
		x.decisionStarts = append(x.decisionStarts, t)
	} else if isLiteralTask(t) {
		x.literalStarts = append(x.literalStarts, t)
	} else {
		x.starts = append(x.starts, t)
	}
	x.seq++
	emit(map[string]any{"event": "executor_started", "role": t.Role, "model": t.Model, "session_id": fmt.Sprintf("shape-session-%d", x.seq)})
	return &shapeSession{x: x, task: t, id: fmt.Sprintf("shape-session-%d", x.seq), emit: emit}, nil
}

type shapeSession struct {
	x    *shapeExec
	task executor.Task
	id   string
	n    int
	emit executor.Sink
}

func (s *shapeSession) Handle() executor.Handle {
	return executor.Handle{Role: s.task.Role, Executor: executor.ClaudeCode, Version: "claude-code 9.9.9", SessionID: s.id, Resume: executor.ResumeNone}
}
func (s *shapeSession) Close() error { return nil }
func (s *shapeSession) Round(_ context.Context, in executor.Input) (executor.RoundResult, error) {
	s.n++
	if isLiteralTask(s.task) {
		s.x.mu.Lock()
		s.x.literalCalls++
		fn := result(map[string]any{"items": []any{}})
		if len(s.x.literal) > 0 {
			fn = s.x.literal[min(s.x.literalCalls-1, len(s.x.literal)-1)]
		}
		s.x.mu.Unlock()
		res, err := fn(s.n, s.task.Root)
		if err != nil {
			return executor.RoundResult{}, err
		}
		res.Ending, res.Handle = executor.EndFinish, s.Handle()
		return res, nil
	}
	if isDecisionTask(s.task) {
		s.x.mu.Lock()
		s.x.decisionCalls++
		fn := result(map[string]any{"items": []any{}})
		if len(s.x.decision) > 0 {
			fn = s.x.decision[min(s.x.decisionCalls-1, len(s.x.decision)-1)]
		}
		s.x.mu.Unlock()
		res, err := fn(s.n, s.task.Root)
		if err != nil {
			return executor.RoundResult{}, err
		}
		res.Ending, res.Handle = executor.EndFinish, s.Handle()
		return res, nil
	}
	if isContractTask(s.task) {
		s.x.mu.Lock()
		s.x.contractCalls++
		fn := result(map[string]any{"findings": []any{}, "environment": []any{}})
		if strings.Contains(string(s.task.Schema), `"verdicts"`) {
			fn = result(map[string]any{"verdicts": []any{}})
		}
		if len(s.x.contract) > 0 {
			fn = s.x.contract[min(s.x.contractCalls-1, len(s.x.contract)-1)]
		}
		s.x.mu.Unlock()
		res, err := fn(s.n, s.task.Root)
		if err != nil {
			return executor.RoundResult{}, err
		}
		res.Ending, res.Handle = executor.EndFinish, s.Handle()
		return res, nil
	}

	s.x.mu.Lock()
	s.x.inputs = append(s.x.inputs, in)
	s.x.calls++
	i := min(s.x.calls-1, len(s.x.rounds)-1)
	fn := s.x.rounds[i]
	s.x.mu.Unlock()
	if s.n == 1 {
		if !s.x.noRead {
			s.emit(map[string]any{"event": "executor_tool", "name": "Read", "tool_use_id": "t1", "denied": false})
		}
		if s.x.bash {
			s.emit(map[string]any{"event": "executor_tool", "name": "Bash", "tool_use_id": "t2", "denied": false})
		}
	}
	res, err := fn(s.n, s.task.Root)
	if err != nil {
		return executor.RoundResult{}, err
	}
	if res.Ending == "" {
		res.Ending = executor.EndFinish
	}
	res.Handle = s.Handle()
	return res, nil
}

func (x *shapeExec) external() External {
	return func(project.Role) (executor.Executor, error) { return x, nil }
}

func result(v any) func(int, string) (executor.RoundResult, error) {
	return func(int, string) (executor.RoundResult, error) {
		b, _ := json.Marshal(v)
		return executor.RoundResult{Structured: b, Usage: executor.Usage{State: executor.UsageUnknown, Source: executor.UsageHarness}}, nil
	}
}

func (f *shpFx) useExternalShape() {
	f.t.Helper()
	f.writeLineup("version: 1\ncommands:\n  checks: [{name: unit, run: 'true'}]\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n", "shape: {executor: claude-code, model: claude-opus-5-5, effort: xhigh}\n")
	f.git("commit", "-qm", "claude-code shaper")
}

func (f *shpFx) shapeWith(x *shapeExec, request string) (int, error) {
	f.t.Helper()
	f.newSession()
	return Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Slug: shpSlug, Request: request, Client: f.Client, External: x.external()})
}

func (f *refFx) refineWith(x *shapeExec, hash string) (int, error) {
	f.t.Helper()
	f.newSession()
	return Refine(context.Background(), f.S, RefineInput{Project: f.Root, Slug: shpSlug, Hash: hash,
		Request: "Clarify only the existing draft; preserve its executable contract.", RequestPath: filepath.Join(f.Aux, "correction.txt"),
		Client: f.Client, External: x.external()})
}

func noNativeRequests(t *testing.T, f *shpFx) {
	t.Helper()
	if n := len(f.Prov.requests()); n != 0 {
		t.Fatalf("%d native provider requests: an external role fell back to the native provider", n)
	}
}

func shapeAbsent(t *testing.T, f *shpFx) {
	t.Helper()
	for _, p := range []string{".kogen/intents/is-even", "features/steps/is-even__steps.js", ".kogen/local/provenance/is-even/current.json", ".kogen/local/readiness/is-even.json"} {
		if f.exists(p) {
			t.Fatalf("%s was published", p)
		}
	}
}

// TestClaudeLineupShapeSubmission: a Claude Code shaper's final structured result is the existing
// submit payload, routed through the same validation, readiness, exact red acknowledgment and
// publication; actual feedback returns to the same session; questions keep their numbered CLI and
// start a fresh session; malformed output, auth refusal and the round cap publish nothing and
// never use the native provider.
func TestClaudeLineupShapeSubmission(t *testing.T) {
	parallel(t)
	t.Run("readiness feedback and exact red acknowledgment stay with one session", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		head := f.head()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}})}}
		exit, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError.")
		if err != nil || exit != 0 {
			t.Fatalf("exit %d, err %v\nstdout: %s\nstderr: %s", exit, err, f.Stdout, f.Stderr)
		}
		lines := strings.Split(strings.TrimSuffix(f.Stdout.String(), "\n"), "\n")
		shpEq(t, "line count", len(lines), 7)
		shpEq(t, "scenarios line", lines[4], "Scenarios: 2 (2 fail now, 0 pass now)")
		shpMatch(t, "usage line", lines[5], `^shape claude-opus-5-5/xhigh claude-code 5 round\(s\) wall_ms=\d+$`)
		shpEq(t, "feature", f.read(".kogen/intents/is-even/acceptance.feature"), shpFeature)
		shpEq(t, "steps", f.read("features/steps/is-even__steps.js"), shpSteps)
		shpEq(t, "intent", f.read(".kogen/intents/is-even/intent.md"), shpScopedIntentMD)
		shpEq(t, "HEAD", f.head(), head)
		if f.exists(".kogen/local/approvals/is-even.json") || !f.exists(".kogen/local/readiness/is-even.json") || !f.exists(".kogen/local/provenance/is-even/current.json") {
			t.Fatal("approval, readiness or provenance state is wrong")
		}
		noNativeRequests(t, f)
		if len(x.starts) != 1 || len(x.inputs) != 2 {
			t.Fatalf("starts %d, rounds %d", len(x.starts), len(x.inputs))
		}
		task := x.starts[0]
		shpEq(t, "role", task.Role, executor.RoleShape)
		shpEq(t, "model", task.Model, "claude-opus-5-5")
		shpEq(t, "effort", task.Effort, "xhigh")
		shpEq(t, "rounds", task.MaxRounds, 6)
		wantSchema, _ := json.Marshal(ShaperToolDefinitions(SubmitTool)[0]["parameters"])
		if string(task.Schema) != string(wantSchema) {
			t.Fatalf("external schema differs from local submit: %s != %s", task.Schema, wantSchema)
		}
		var schema map[string]any
		if json.Unmarshal(task.Schema, &schema) != nil || schema["properties"].(map[string]any)["files"] == nil || schema["properties"].(map[string]any)["questions"] == nil {
			t.Fatalf("schema %s", task.Schema)
		}
		for _, part := range []string{"Add calc.is_even(n). Non-integers raise TypeError.", "# Calc", "Use the narrowest test that exercises the rule", "Context navigation: ", "read-only external session"} {
			shpContains(t, "task", task.Brief.Stable, part)
		}
		shpEq(t, "commands paragraph", strings.Count(task.Brief.Stable, shpCommandsParagraph), 1)
		shpEq(t, "first feedback", x.inputs[0].Feedback, "")
		shpContains(t, "feedback", x.inputs[1].Feedback, "Readiness observations\nChecked baseline: ")
		shpContains(t, "feedback", x.inputs[1].Feedback, "Scenario: Even numbers (acceptance.feature line 2)")
		events := f.events()
		sum := shpNamed(events, "executor_summary")
		if len(sum) != 1 || sum[0]["rounds"].(float64) != 2 || sum[0]["session_id"] != "shape-session-1" || sum[0]["executor"] != "claude-code" || sum[0]["model"] != "claude-opus-5-5" {
			t.Fatalf("summary %v", sum)
		}
		if len(shpNamed(events, "provider_call")) != 0 || len(shpNamed(events, "documentation")) != 0 {
			t.Fatal("an external session produced native provider or documentation records")
		}
	})

	t.Run("an invalid submission is retried in the same session and three end shaping", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(map[string]any{"files": []any{map[string]any{"path": "calc.js", "content": "x"}}})}}
		_, err := f.shapeWith(x, "Add calc.is_even(n).")
		shpWantError(t, err, "shape", "invalid_output", "the shaper's output was not a valid Intent after 3 attempts", 1)
		if len(x.starts) != 1 || len(x.inputs) != 3 {
			t.Fatalf("starts %d, rounds %d", len(x.starts), len(x.inputs))
		}
		shpContains(t, "feedback", x.inputs[1].Feedback, "invalid submission")
		shapeAbsent(t, f)
		noNativeRequests(t, f)
	})

	t.Run("a model-supplied provenance line is not a Kogen documentation record", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		faked := strings.Replace(shpScopedIntentMD, "## Tests", "## Source comparison\n- documentation-lookup: retrieved https://example.invalid/doc\n\n## Tests", 1)
		files := shpFiles()
		files["files"].([]any)[0].(map[string]any)["content"] = faked
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(files)}}
		f.shapeWith(x, "Add calc.is_even(n).")
		shpContains(t, "feedback", x.inputs[1].Feedback, "shape/undocumented")
		shapeAbsent(t, f)
	})

	t.Run("questions keep the numbered CLI and the next invocation is a fresh session", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			result(map[string]any{"questions": []any{shpQuestion("What should is_even do with a non-integer?", "raise TypeError", "return False"), shpQuestion("How should it be named?", "snake_case", "camelCase")}}),
			result(map[string]any{"items": []any{map[string]any{"number": 1, "class": "ui_ux", "why": "a choice of behavior"}, map[string]any{"number": 2, "class": "ui_ux", "why": "a choice of wording"}}}),
			result(shpFiles()),
			result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}),
		}}
		exit, err := f.shapeWith(x, "Add calc.is_even(n).")
		if err != nil || exit != 5 {
			t.Fatalf("exit %d, err %v", exit, err)
		}
		shpEq(t, "stdout", f.Stdout.String(), "Questions for is-even:\n"+
			"1. What should is_even do with a non-integer?\n   a) raise TypeError\n   b) return False\n2. How should it be named?\n   a) snake_case\n   b) camelCase\n"+
			`Answer one per line as <n>: <letter or your words>, e.g. printf '1: a\n2: a\n' | kogen intent shape is-even -`+"\n"+
			"Or enter all: recommended to use every question's recommended option.\n")
		shapeAbsent(t, f)
		exit, err = f.shapeWith(x, "1: a\n2: is_even")
		if err != nil || exit != 0 {
			t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
		}
		if len(x.starts) != 3 {
			t.Fatalf("%d sessions, want a fresh one per invocation and one for the question check", len(x.starts))
		}
		shpContains(t, "second task", x.starts[2].Brief.Stable, "raise TypeError")
		noNativeRequests(t, f)
	})

	t.Run("startup, protocol and round-cap failures publish nothing and never fall back", func(t *testing.T) {
		parallel(t)
		for name, tc := range map[string]struct {
			x      *shapeExec
			class  string
			reason string
		}{
			"auth refused": {&shapeExec{startErr: executor.Fail(executor.ReasonAuthRefused, "api key route")}, "executor", "auth_refused"},
			"unavailable":  {&shapeExec{startErr: executor.Fail(executor.ReasonUnavailable, "too old")}, "executor", "unavailable"},
			"missing structured output": {&shapeExec{rounds: []func(int, string) (executor.RoundResult, error){
				func(int, string) (executor.RoundResult, error) {
					return executor.RoundResult{}, executor.Fail(executor.ReasonProtocol, "no structured_output")
				}}}, "executor", "protocol"},
			"round cap": {&shapeExec{cap: 1, rounds: []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(shpFiles())}}, "candidate", "executor_round_limit"},
		} {
			t.Run(name, func(t *testing.T) {
				parallel(t)
				f := newShpFx(t)
				f.useExternalShape()
				tc.x.t = t
				head := f.head()
				_, err := f.shapeWith(tc.x, "Add calc.is_even(n).")
				se, ok := sys.AsError(err)
				if !ok || se.Class != tc.class || se.Reason != tc.reason {
					t.Fatalf("error %v", err)
				}
				if name == "round cap" {
					// Native checks write the proposed package while it is red; what a stop never
					// leaves is a certified result.
					for _, p := range []string{".kogen/local/provenance/is-even/current.json", ".kogen/local/approvals/is-even.json"} {
						if f.exists(p) {
							t.Fatalf("%s exists after a round-cap stop", p)
						}
					}
				} else {
					shapeAbsent(t, f)
				}
				shpEq(t, "HEAD", f.head(), head)
				noNativeRequests(t, f)
				if name == "round cap" && len(tc.x.inputs) != 1 {
					t.Fatalf("%d rounds with a cap of 1", len(tc.x.inputs))
				}
			})
		}
	})
}

// TestClaudeLineupShapeCheckoutGuard: Kogen compares the checkout (tracked, tracked-ignored and
// nonignored files, modes), the raw index and HEAD around every external round; a session change
// stops before publication with the existing custody reasons and is left as it is.
func TestClaudeLineupShapeCheckoutGuard(t *testing.T) {
	parallel(t)
	mutations := map[string]struct {
		do     func(t *testing.T, f *shpFx, root string)
		reason string
	}{
		"edit a tracked file":         {func(t *testing.T, f *shpFx, root string) { f.write("calc.js", "tampered\n") }, "baseline_changed"},
		"delete a file":               {func(t *testing.T, f *shpFx, root string) { os.Remove(filepath.Join(root, "app.txt")) }, "baseline_changed"},
		"change a mode":               {func(t *testing.T, f *shpFx, root string) { os.Chmod(filepath.Join(root, "calc.js"), 0o755) }, "baseline_changed"},
		"edit a tracked ignored file": {func(t *testing.T, f *shpFx, root string) { f.write("ignored-but-tracked.txt", "changed\n") }, "baseline_changed"},
		"stage a change": {func(t *testing.T, f *shpFx, root string) {
			f.write("staged.txt", "x\n")
			f.git("add", "staged.txt")
			os.Remove(filepath.Join(root, "staged.txt"))
		}, "baseline_changed"},
		"move HEAD": {func(t *testing.T, f *shpFx, root string) { f.git("commit", "-q", "--allow-empty", "-m", "sneaky") }, "head_moved"},
	}
	for name, m := range mutations {
		t.Run("shape "+name, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t)
			f.write(".gitignore", ".kogen/local/\nignored-but-tracked.txt\n")
			f.write("ignored-but-tracked.txt", "original\n")
			f.git("add", "-f", "-A")
			f.git("commit", "-qm", "tracked ignored")
			f.useExternalShape()
			x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
				func(n int, root string) (executor.RoundResult, error) {
					m.do(t, f, root)
					return result(shpFiles())(n, root)
				}}}
			head := f.head()
			_, err := f.shapeWith(x, "Add calc.is_even(n).")
			se, ok := sys.AsError(err)
			if !ok || se.Class != "candidate" || se.Reason != m.reason {
				t.Fatalf("error %v", err)
			}
			if len(x.inputs) != 1 {
				t.Fatalf("%d rounds: inference continued after the mutation", len(x.inputs))
			}
			for _, p := range []string{".kogen/intents/is-even", ".kogen/local/provenance/is-even/current.json", ".kogen/local/readiness/is-even.json"} {
				if f.exists(p) {
					t.Fatalf("%s was published after a checkout mutation", p)
				}
			}
			if m.reason == "baseline_changed" && name != "stage a change" && name != "delete a file" {
				// The detected edit is reported, not concealed or overwritten.
				if f.read("calc.js") == "function add(a, b) { return a + b }\n" && name == "edit a tracked file" {
					t.Fatal("the detected edit was reverted")
				}
			}
			if m.reason == "head_moved" && f.head() == head {
				t.Fatal("HEAD was reset")
			}
			shpContains(t, "stop", se.Render(), "nothing was published")
			noNativeRequests(t, f)
		})
	}

	t.Run("created untracked files are removed, pre-existing ones stay, a stat refresh is equal", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.write("pre-existing.txt", "mine\n")
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				f.write("features/steps/.keep-note", "note\n")
				f.git("status")
				return result(shpFiles())(n, root)
			}}}
		if exit, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d err %v", exit, err)
		}
		if f.exists("features/steps/.keep-note") {
			t.Fatal("the created untracked file was not removed")
		}
		if f.read("pre-existing.txt") != "mine\n" {
			t.Fatal("a pre-existing untracked file was touched")
		}
	})

	t.Run("created files are reported once in the evidence", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				f.write("features/steps/.keep-note", "note\n")
				f.write("b-note.txt", "note\n")
				return result(shpFiles())(n, root)
			}, result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}})}}
		if exit, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d err %v", exit, err)
		}
		ev := shpNamed(f.events(), "shape_untracked_removed")
		if len(ev) != 1 || fmt.Sprint(ev[0]["paths"]) != "[b-note.txt features/steps/.keep-note]" {
			t.Fatalf("removal events %v", ev)
		}
	})

	t.Run("a file the session untracked is not removed", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				f.git("rm", "-q", "--cached", "calc.js")
				return result(shpFiles())(n, root)
			}}}
		_, err := f.shapeWith(x, "Add calc.is_even(n).")
		if se, ok := sys.AsError(err); !ok || se.Reason != "baseline_changed" {
			t.Fatalf("error %v", err)
		}
		if !f.exists("calc.js") {
			t.Fatal("a file that existed before the round was removed")
		}
		if len(shpNamed(f.events(), "shape_untracked_removed")) != 0 {
			t.Fatal("a pre-existing file was reported as created")
		}
	})

	t.Run("a failed removal names the files and publishes nothing", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				f.write("stuck/note.txt", "note\n")
				dir := filepath.Join(root, "stuck")
				os.Chmod(dir, 0o500)
				t.Cleanup(func() { os.Chmod(dir, 0o700) })
				return result(shpFiles())(n, root)
			}}}
		_, err := f.shapeWith(x, "Add calc.is_even(n).")
		se, ok := sys.AsError(err)
		if !ok || se.Class != "candidate" || se.Reason != "shape_untracked_remove_failed" {
			t.Fatalf("error %v", err)
		}
		shpContains(t, "stop", se.Render(), "stuck/note.txt")
		shpContains(t, "stop", se.Render(), "nothing was published")
		shapeAbsent(t, f)
	})

	t.Run("kogen's own run evidence is not a checkout edit", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				f.write(".kogen/local/shaping/scratch-note.txt", "kogen-owned run evidence\n")
				return result(shpFiles())(n, root)
			}, result(map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}})}}
		if exit, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil || exit != 0 {
			t.Fatalf("exit %d err %v", exit, err)
		}
	})

	t.Run("refine mutation preserves the package, approval and the detected edit", func(t *testing.T) {
		parallel(t)
		f := newRefFx(t, refOpts{approved: true, external: true})
		before, approval, head, index := f.pkg(), f.read(".kogen/local/approvals/is-even.json"), f.head(), f.index()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
			func(n int, root string) (executor.RoundResult, error) {
				f.write("calc.js", "tampered\n")
				return result(map[string]any{"files": []any{map[string]any{"path": refIntentPath, "content": refTitled}}})(n, root)
			}}}
		_, err := f.refineWith(x, f.hash0)
		if se, ok := sys.AsError(err); !ok || se.Class != "candidate" || se.Reason != "baseline_changed" {
			t.Fatalf("error %v", err)
		}
		refEqual(t, "package", f.pkg(), before)
		shpEq(t, "approval", f.read(".kogen/local/approvals/is-even.json"), approval)
		shpEq(t, "HEAD", f.head(), head)
		shpEq(t, "index", f.index() == index, true)
		shpEq(t, "the detected edit stays", f.read("calc.js"), "tampered\n")
		noNativeRequests(t, f.shpFx)
	})
}

// TestClaudeLineupRefinePreservation: an external refiner's submit_refinement goes through the
// same ownership, cumulative-proposal and publication rules; success revokes and archives the
// approval, every failure preserves the package, approval, HEAD and index.
func TestClaudeLineupRefinePreservation(t *testing.T) {
	parallel(t)
	t.Run("an owned correction publishes and revokes the approval", func(t *testing.T) {
		parallel(t)
		f := newRefFx(t, refOpts{approved: true, external: true})
		before, head, index := f.pkg(), f.head(), f.index()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(map[string]any{"files": []any{map[string]any{"path": refIntentPath, "content": refTitled}}})}}
		exit, err := f.refineWith(x, f.hash0)
		if err != nil || exit != 0 {
			t.Fatalf("exit %d err %v\n%s\n%s", exit, err, f.Stdout, f.Stderr)
		}
		refEqual(t, "package", f.pkg(), refWith(before, refIntentPath, refTitled))
		shpEq(t, "HEAD", f.head(), head)
		shpEq(t, "index", f.index() == index, true)
		if f.exists(".kogen/local/approvals/is-even.json") {
			t.Fatal("success must revoke the approval")
		}
		refText(t, "archived approval", f.read(strings.TrimPrefix(f.evidence(), f.Root+"/")+"/revoked-approval.json"), f.approval)
		task := x.starts[0]
		shpEq(t, "role", task.Role, executor.RoleShape)
		shpContains(t, "task", task.Brief.Stable, "Clarify only the existing draft; preserve its executable contract.")
		shpContains(t, "task", task.Brief.Stable, "--- "+refIntentPath+"\n"+shpScopedIntentMD)
		shpContains(t, "task", task.Brief.Stable, "# Exclusively owned editable files")
		shpEq(t, "commands paragraph", strings.Count(task.Brief.Stable, shpCommandsParagraph), 1)
		var schema map[string]any
		if json.Unmarshal(task.Schema, &schema) != nil || schema["properties"].(map[string]any)["constraint"] == nil {
			t.Fatalf("schema %s", task.Schema)
		}
		sum := shpNamed(f.evidenceEvents(), "executor_summary")
		if len(sum) != 1 || sum[0]["rounds"].(float64) != 1 || sum[0]["executor"] != "claude-code" {
			t.Fatalf("summary %v", sum)
		}
		noNativeRequests(t, f.shpFx)
	})

	t.Run("failures preserve the package, approval, HEAD and index", func(t *testing.T) {
		parallel(t)
		newPath := ".kogen/intents/is-even/model.qnt"
		for name, tc := range map[string]struct {
			rounds []func(int, string) (executor.RoundResult, error)
			class  string
			reason string
			cap    int
		}{
			"a new file":   {[]func(int, string) (executor.RoundResult, error){result(map[string]any{"files": []any{map[string]any{"path": newPath, "content": "module m {}\n"}}})}, "shape", "refinement_invalid", 0},
			"a constraint": {[]func(int, string) (executor.RoundResult, error){result(map[string]any{"constraint": "needs a new scenario file"})}, "", "", 0},
			"no structured": {[]func(int, string) (executor.RoundResult, error){func(int, string) (executor.RoundResult, error) {
				return executor.RoundResult{}, executor.Fail(executor.ReasonProtocol, "none")
			}}, "executor", "protocol", 0},
			"the round cap": {[]func(int, string) (executor.RoundResult, error){result(map[string]any{"files": []any{map[string]any{"path": newPath, "content": "module m {}\n"}}})}, "candidate", "executor_round_limit", 1},
		} {
			t.Run(name, func(t *testing.T) {
				parallel(t)
				f := newRefFx(t, refOpts{approved: true, external: true})
				before, approval, head, index := f.pkg(), f.read(".kogen/local/approvals/is-even.json"), f.head(), f.index()
				x := &shapeExec{t: t, cap: tc.cap, rounds: tc.rounds}
				exit, err := f.refineWith(x, f.hash0)
				if err == nil || exit == 0 {
					t.Fatalf("refine succeeded: exit %d", exit)
				}
				if se, ok := sys.AsError(err); tc.class != "" && (!ok || se.Class != tc.class || se.Reason != tc.reason) {
					t.Fatalf("error %v", err)
				}
				refEqual(t, "package", f.pkg(), before)
				shpEq(t, "approval", f.read(".kogen/local/approvals/is-even.json"), approval)
				shpEq(t, "HEAD", f.head(), head)
				shpEq(t, "index", f.index() == index, true)
				noNativeRequests(t, f.shpFx)
			})
		}
	})

	t.Run("startup failure keeps everything", func(t *testing.T) {
		parallel(t)
		f := newRefFx(t, refOpts{approved: true, external: true})
		before, approval := f.pkg(), f.read(".kogen/local/approvals/is-even.json")
		x := &shapeExec{t: t, startErr: executor.Fail(executor.ReasonAuthRefused, "api key")}
		_, err := f.refineWith(x, f.hash0)
		if se, ok := sys.AsError(err); !ok || se.Reason != "auth_refused" {
			t.Fatalf("error %v", err)
		}
		refEqual(t, "package", f.pkg(), before)
		shpEq(t, "approval", f.read(".kogen/local/approvals/is-even.json"), approval)
		noNativeRequests(t, f.shpFx)
	})
}

// TestClaudeLineupUnsupportedCapabilities: the unsupported Kogen facilities stay explicit — the
// descriptor lists the limitations, the task tells the session which native tools it lacks, and
// the session's own research becomes no Kogen record.
func TestClaudeLineupUnsupportedCapabilities(t *testing.T) {
	parallel(t)
	t.Run("a descriptor without the shape role is refused", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		f.newSession()
		_, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Slug: shpSlug, Request: "x", Client: f.Client,
			External: func(project.Role) (executor.Executor, error) { return buildOnlyExec{}, nil }})
		if se, ok := sys.AsError(err); !ok || se.Reason != "unavailable" {
			t.Fatalf("error %v", err)
		}
		noNativeRequests(t, f)
	})

	t.Run("limitations are reported and research is not a Kogen record", func(t *testing.T) {
		parallel(t)
		f := newShpFx(t)
		f.useExternalShape()
		x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(map[string]any{"files": []any{
			map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}})}}
		caps, _ := x.Describe().For(executor.RoleShape)
		if caps.Probe || caps.Images || caps.Workers || caps.Resume != executor.ResumeNone {
			t.Fatalf("capabilities %+v", caps)
		}
		if _, err := f.shapeWith(x, "Add calc.is_even(n)."); err != nil {
			t.Fatal(err)
		}
		for _, part := range []string{"probe, view_image, documentation records, shape_batch", "harness only", "not available here"} {
			shpContains(t, "task", x.starts[0].Brief.Stable, part)
		}
		if docs, _ := filepath.Glob(filepath.Join(f.Root, ".kogen/local/shaping/is-even/documentation*")); len(docs) != 0 {
			t.Fatalf("documentation records %v", docs)
		}
		for _, e := range f.events() {
			if e["event"] == "documentation" || e["event"] == "documentation_lookup" {
				t.Fatalf("a Kogen documentation record from an external session: %v", e)
			}
		}
	})
}

type buildOnlyExec struct{}

func (buildOnlyExec) Describe() executor.Descriptor {
	return executor.Descriptor{Name: executor.ClaudeCode, Contract: executor.Contract, Capabilities: executor.Capabilities{Roles: []string{executor.RoleBuild}}}
}
func (buildOnlyExec) Start(context.Context, executor.Task, executor.Sink) (executor.Session, error) {
	return nil, fmt.Errorf("must not start")
}

// shpCommandsParagraph is the last line of the session note every external session receives.
const shpCommandsParagraph = "Commands: Kogen runs readiness on every submission (the configured setup, the acceptance command, the Quint model and the Feasibility checks) and returns its actual results to this session. Do not repeat that work yourself: do not run the configured setup or checks, the acceptance command or a whole test suite, and do not prototype the implementation. Run only short commands that answer a named concrete uncertainty, such as one focused test, a probe or a tool query, each finishing on its own within 5 minutes. Never start a command in the background and never wait or poll with sleep: when you would need a longer run, submit your complete candidate and use Kogen's results."

// TestExecutorRoundTiming: every executor_round event names its session and carries Kogen's span
// of the round and the model time of the last result line of the round's retained raw stream, null
// when the stream is missing, has no usable result line or a non-integer duration_api_ms.
func TestExecutorRoundTiming(t *testing.T) {
	parallel(t)
	for name, tc := range map[string]struct {
		stream string // the round's raw stream; "" writes none
		want   any
	}{
		"the last result line":                      {`{"type":"system"}` + "\n" + `{"type":"result","duration_api_ms":10}` + "\n" + `{"type":"result","duration_api_ms":1368516}` + "\n", float64(1368516)},
		"an earlier line when the last is unusable": {`{"type":"result","duration_api_ms":7}` + "\n" + `{"type":"result","duration_api_ms":-1}` + "\n" + `{"type":"assistant","duration_api_ms":9}` + "\n" + "not json\n", float64(7)},
		"a missing file":                            {"", nil},
		"no result line":                            {`{"type":"system","duration_api_ms":5}` + "\n", nil},
		"a fractional value":                        {`{"type":"result","duration_api_ms":1.5}` + "\n", nil},
		"a string value":                            {`{"type":"result","duration_api_ms":"12"}` + "\n", nil},
		"an absent value":                           {`{"type":"result"}` + "\n", nil},
		"a null value":                              {`{"type":"result","duration_api_ms":null}` + "\n", nil},
	} {
		t.Run(name, func(t *testing.T) {
			parallel(t)
			f := newShpFx(t)
			f.useExternalShape()
			dir := filepath.Join(f.Root, ".kogen/local/shaping", shpSlug, "executor-shape")
			round := func(answer any) func(int, string) (executor.RoundResult, error) {
				return func(n int, root string) (executor.RoundResult, error) {
					if tc.stream != "" {
						if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("round-%d.jsonl", n)), []byte(tc.stream), 0o600); err != nil {
							return executor.RoundResult{}, err
						}
					}
					return result(answer)(n, root)
				}
			}
			ack := map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}
			x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){round(shpFiles()), round(ack)}}
			if exit, err := f.shapeWith(x, "Add calc.is_even(n). Non-integers raise TypeError."); err != nil || exit != 0 {
				t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
			}
			events := f.events()
			rounds := shpNamed(events, "executor_round")
			var main, review, decision, literal int
			for _, e := range rounds {
				if _, ok := e["model_ms"]; !ok {
					t.Fatalf("no model_ms key: %v", e)
				}
				for _, k := range []string{"started_ms", "wall_ms"} {
					if n, ok := e[k].(float64); !ok || n < 0 {
						t.Fatalf("%s of %v", k, e)
					}
				}
				switch e["session"] {
				case "executor-shape":
					main++
					shpEq(t, "round "+fmt.Sprint(e["round"]), e["model_ms"], tc.want)
				case "executor-contract-review-1":
					review++
					shpEq(t, "review model_ms", e["model_ms"], nil)
				case "executor-decision-1":
					decision++
					shpEq(t, "decision model_ms", e["model_ms"], nil)
				case "executor-literal-1":
					literal++
					shpEq(t, "literal model_ms", e["model_ms"], nil)
				default:
					t.Fatalf("session %v", e["session"])
				}
			}
			shpEq(t, "main rounds", main, 2)
			shpEq(t, "review rounds", review, 1)
			shpEq(t, "decision rounds", decision, 1)
			shpEq(t, "literal rounds", literal, 1)
			accounts := shpNamed(events, "shape_time")
			if len(accounts) != 1 {
				t.Fatalf("accounts %v", accounts)
			}
			shpEq(t, "unknown model time", accounts[0]["shaper_model_ms"] == nil, tc.want == nil)
		})
	}
}

// TestShapeTimeWithQuestions: a shape that ends with questions records its account and ends on
// the time line.
func TestShapeTimeWithQuestions(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	x := &shapeExec{t: t, rounds: []func(int, string) (executor.RoundResult, error){
		result(map[string]any{"questions": []any{shpQuestion("What should is_even do with a non-integer?", "raise TypeError", "return False")}}),
		result(map[string]any{"items": []any{map[string]any{"number": 1, "class": "ui_ux", "why": "a choice of behavior"}}}),
	}}
	exit, err := f.shapeWith(x, "Add calc.is_even(n).")
	if err != nil || exit != 5 {
		t.Fatalf("exit %d, err %v", exit, err)
	}
	events := f.events()
	accounts := shpNamed(events, "shape_time")
	if len(accounts) != 1 || events[len(events)-2]["event"] != "shape_time" || events[len(events)-1]["event"] != "session_end" {
		t.Fatalf("the record does not end with one account before the session end: %v", events)
	}
	lines := strings.Split(strings.TrimSuffix(f.Stderr.String(), "\n"), "\n")
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, "kogen: shape time ") || !strings.Contains(last, "shaper ") {
		t.Fatalf("last stderr line %q", last)
	}
	shpEq(t, "time lines", strings.Count(f.Stderr.String(), "kogen: shape time"), 1)
}
