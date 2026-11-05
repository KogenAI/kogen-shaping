package consumes

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"

	"github.com/KogenAI/kogen-executor/executor"
)

var (
	_ func(executor.Executor) executor.Descriptor                                                      = executor.Executor.Describe
	_ func(executor.Executor, context.Context, executor.Task, executor.Sink) (executor.Session, error) = executor.Executor.Start
	_ func(executor.Session, context.Context, executor.Input) (executor.RoundResult, error)            = executor.Session.Round
	_ func(executor.Session) executor.Handle                                                           = executor.Session.Handle
	_ func(executor.Session) error                                                                     = executor.Session.Close
	_ func(executor.Descriptor, string) (executor.RoleCapabilities, bool)                              = executor.Descriptor.For
	_ func(executor.Descriptor, string) error                                                          = executor.Descriptor.ValidateRole
	_ func(string, string) *sys.Error                                                                  = executor.Fail
	_ func(int) *sys.Error                                                                             = executor.RoundLimit
)

// Field types pinned at compile time.
var (
	_ string          = executor.Brief{}.Stable
	_ string          = executor.Brief{}.Dynamic
	_ string          = executor.Handle{}.Executor
	_ string          = executor.Handle{}.SessionID
	_ string          = executor.Handle{}.Version
	_ string          = executor.Input{}.Feedback
	_ executor.Ending = executor.RoundResult{}.Ending
	_ executor.Handle = executor.RoundResult{}.Handle
	_ json.RawMessage = executor.RoundResult{}.Structured
	_ executor.Usage  = executor.RoundResult{}.Usage
	_ executor.Brief  = executor.Task{}.Brief
	_ string          = executor.Task{}.Dir
	_ string          = executor.Task{}.Effort
	_ string          = executor.Task{}.Home
	_ int             = executor.Task{}.MaxRounds
	_ string          = executor.Task{}.Model
	_ string          = executor.Task{}.Role
	_ string          = executor.Task{}.Root
	_ string          = executor.Task{}.RunID
	_ json.RawMessage = executor.Task{}.Schema
)

// Constants pinned at compile time with their types.
const (
	_ string          = executor.ClaudeCode
	_ string          = executor.RoleShape
	_ executor.Ending = executor.EndFinish
	_ string          = executor.ReasonProtocol
	_ string          = executor.ReasonUnavailable
	_ string          = executor.UsagePartial
)

func ptr[T any](v T) *T { return &v }

// TestExecutorConstants pins the role, executor name, ending, reason and usage-state words
// kogen-shaping writes to its run log and compares against.
func TestExecutorConstants(t *testing.T) {
	for _, c := range []struct{ name, got, want string }{
		{"executor.ClaudeCode", executor.ClaudeCode, "claude-code"},
		{"executor.RoleShape", executor.RoleShape, "shape"},
		{"executor.EndFinish", string(executor.EndFinish), "finish"},
		{"executor.ReasonProtocol", executor.ReasonProtocol, "protocol"},
		{"executor.ReasonUnavailable", executor.ReasonUnavailable, "unavailable"},
		{"executor.UsagePartial", executor.UsagePartial, "partial"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestExecutorFail pins the stop shape/external.go and shape/shape.go make: class executor,
// the given reason and detail, exit 1.
func TestExecutorFail(t *testing.T) {
	err := executor.Fail(executor.ReasonProtocol, "the shaper ended its round without a submission")
	if err == nil || err.Class != "executor" || err.Reason != "protocol" || err.Detail != "the shaper ended its round without a submission" || err.Exit != 1 {
		t.Fatalf("executor.Fail(ReasonProtocol, detail) = %+v, want class executor, reason protocol, that detail, exit 1", err)
	}
	if got, want := err.Error(), "executor/protocol: the shaper ended its round without a submission"; got != want {
		t.Errorf("executor.Fail(...).Error() = %q, want %q", got, want)
	}
	var wrapped error = executor.Fail(executor.ReasonUnavailable, "no executor")
	e, ok := sys.AsError(wrapped)
	if !ok || e.Class != "executor" || e.Reason != "unavailable" || e.Detail != "no executor" || e.Exit != 1 || len(e.Lines) != 0 {
		t.Errorf("executor.Fail(ReasonUnavailable, ...) as an error = %+v, %v, want class executor, reason unavailable, exit 1 and no lines", e, ok)
	}
}

// TestExecutorRoundLimit pins the stop of an external session that used its rounds.
func TestExecutorRoundLimit(t *testing.T) {
	err := executor.RoundLimit(6)
	if err == nil || err.Class != "candidate" || err.Reason != "executor_round_limit" || err.Exit != 1 {
		t.Fatalf("executor.RoundLimit(6) = %+v, want class candidate, reason executor_round_limit, exit 1", err)
	}
	const want = "the executor used its 6 rounds without a verified, reviewed candidate; the candidate is left in place"
	if err.Detail != want {
		t.Errorf("executor.RoundLimit(6).Detail = %q, want %q", err.Detail, want)
	}
	if got := executor.RoundLimit(12).Detail; got != "the executor used its 12 rounds without a verified, reviewed candidate; the candidate is left in place" {
		t.Errorf("executor.RoundLimit(12).Detail = %q, want the round count in the detail", got)
	}
}

// TestExecutorDescriptor pins what shape/external.go reads of an executor's descriptor: its
// name, whether it serves the shape role (ValidateRole) and that role's capabilities (For,
// with MaxRounds).
func TestExecutorDescriptor(t *testing.T) {
	serving := executor.Descriptor{
		Name:     "claude-code",
		Contract: executor.Contract,
		RoleCapabilities: map[string]executor.RoleCapabilities{
			executor.RoleShape: {MaxRounds: 8, Structured: true},
		},
	}
	var name string = serving.Name
	if name != "claude-code" {
		t.Errorf("executor.Descriptor.Name = %q, want claude-code", name)
	}
	caps, ok := serving.For(executor.RoleShape)
	var rounds int = caps.MaxRounds
	if !ok || rounds != 8 {
		t.Errorf("executor.Descriptor.For(RoleShape) = %+v, %v, want the shape capabilities with MaxRounds 8", caps, ok)
	}
	if caps, ok := serving.For(executor.RoleReview); ok || caps.MaxRounds != 0 {
		t.Errorf("executor.Descriptor.For(review) = %+v, %v, want no capabilities for a role the descriptor does not list", caps, ok)
	}
	buildOnly := executor.Descriptor{Name: "builder", Contract: executor.Contract, Capabilities: executor.Capabilities{Roles: []string{"build"}, MaxRounds: 3}}
	if caps, ok := buildOnly.For("build"); !ok || caps.MaxRounds != 3 {
		t.Errorf("executor.Descriptor.For(build) of an aggregate build claim = %+v, %v, want MaxRounds 3", caps, ok)
	}
	if _, ok := buildOnly.For(executor.RoleShape); ok {
		t.Errorf("executor.Descriptor.For(RoleShape) of a build-only descriptor = true, want false")
	}

	if err := serving.ValidateRole(executor.RoleShape); err != nil {
		t.Errorf("executor.Descriptor.ValidateRole(RoleShape) of a serving descriptor = %v, want nil", err)
	}
	for _, c := range []struct {
		name   string
		d      executor.Descriptor
		role   string
		detail string
	}{
		{"role not served", buildOnly, executor.RoleShape, "executor builder does not serve the shape role"},
		{"wrong contract", executor.Descriptor{Name: "old", Contract: "kogen.executor/v1", RoleCapabilities: serving.RoleCapabilities}, executor.RoleShape, "executor old implements kogen.executor/v1, not kogen.executor/v2"},
	} {
		err := c.d.ValidateRole(c.role)
		var e *sys.Error
		if !errors.As(err, &e) || e.Class != "executor" || e.Reason != executor.ReasonUnavailable || e.Detail != c.detail || e.Exit != 1 {
			t.Errorf("executor.Descriptor.ValidateRole(%s) = %v, want executor/unavailable with detail %q and exit 1", c.name, err, c.detail)
		}
	}
}

// fakeExecutor and fakeSession implement the interfaces only to be called through them.
type fakeExecutor struct {
	desc    executor.Descriptor
	started executor.Task
	emitted []map[string]any
	session *fakeSession
}

func (f *fakeExecutor) Describe() executor.Descriptor { return f.desc }

func (f *fakeExecutor) Start(ctx context.Context, t executor.Task, emit executor.Sink) (executor.Session, error) {
	f.started = t
	emit(map[string]any{"event": "executor_start"})
	return f.session, nil
}

type fakeSession struct {
	inputs []executor.Input
	closed int
}

func (s *fakeSession) Round(ctx context.Context, in executor.Input) (executor.RoundResult, error) {
	s.inputs = append(s.inputs, in)
	return executor.RoundResult{
		Ending:     executor.EndFinish,
		Structured: json.RawMessage(`{"files":[]}`),
		Usage:      executor.Usage{Input: ptr(10), State: executor.UsagePartial},
		Handle:     s.Handle(),
	}, nil
}

func (s *fakeSession) Handle() executor.Handle {
	return executor.Handle{Executor: "claude-code", Version: "1.2.3", SessionID: "session-1"}
}

func (s *fakeSession) Close() error { s.closed++; return nil }

// TestExecutorInterfaces pins the Executor and Session methods and the Task, Input, Brief,
// RoundResult and Handle values shape/external.go passes through them, called on interface
// values as shape does.
func TestExecutorInterfaces(t *testing.T) {
	session := &fakeSession{}
	fake := &fakeExecutor{
		desc:    executor.Descriptor{Name: "claude-code", Contract: executor.Contract, RoleCapabilities: map[string]executor.RoleCapabilities{executor.RoleShape: {MaxRounds: 6}}},
		session: session,
	}
	var ex executor.Executor = fake

	if d := ex.Describe(); d.Name != "claude-code" {
		t.Errorf("executor.Executor.Describe().Name = %q, want claude-code", d.Name)
	}
	if got := executor.Executor.Describe(ex); !reflect.DeepEqual(got, fake.desc) {
		t.Errorf("executor.Executor.Describe(ex) = %+v, want the descriptor %+v", got, fake.desc)
	}

	schema := json.RawMessage(`{"type":"object"}`)
	task := executor.Task{
		Role:      executor.RoleShape,
		Root:      "/root",
		Brief:     executor.Brief{Stable: "stable text", Dynamic: "dynamic text"},
		Model:     "model-x",
		Effort:    "high",
		RunID:     "run-1",
		Home:      "/home/user",
		MaxRounds: 6,
		Schema:    schema,
		Dir:       "/run/dir",
	}
	var events []map[string]any
	var sess executor.Session
	sess, err := ex.Start(context.Background(), task, func(e map[string]any) { events = append(events, e) })
	if err != nil || sess != executor.Session(session) {
		t.Fatalf("executor.Executor.Start = %v, %v, want the executor's session", sess, err)
	}
	if !reflect.DeepEqual(events, []map[string]any{{"event": "executor_start"}}) {
		t.Errorf("executor.Executor.Start emitted %v to its sink, want the executor_start event", events)
	}
	got := fake.started
	if got.Role != "shape" || got.Root != "/root" || got.Brief.Stable != "stable text" || got.Brief.Dynamic != "dynamic text" ||
		got.Model != "model-x" || got.Effort != "high" || got.RunID != "run-1" || got.Home != "/home/user" ||
		got.MaxRounds != 6 || string(got.Schema) != `{"type":"object"}` || got.Dir != "/run/dir" {
		t.Errorf("executor.Task as received by Start = %+v, want the values passed (Role, Root, Brief.Stable, Brief.Dynamic, Model, Effort, RunID, Home, MaxRounds, Schema, Dir)", got)
	}
	var brief executor.Brief = got.Brief
	if b, _ := json.Marshal(brief); string(b) != `{"stable":"stable text","dynamic":"dynamic text"}` {
		t.Errorf("executor.Brief JSON = %s, want its stable and dynamic fields", b)
	}

	res, err := sess.Round(context.Background(), executor.Input{Feedback: "fix the lint"})
	if err != nil {
		t.Fatalf("executor.Session.Round: %v", err)
	}
	if !reflect.DeepEqual(session.inputs, []executor.Input{{Feedback: "fix the lint"}}) {
		t.Errorf("executor.Session.Round received %+v, want the feedback fix the lint", session.inputs)
	}
	if res.Ending != executor.EndFinish || string(res.Structured) != `{"files":[]}` || res.Usage.State != executor.UsagePartial || *res.Usage.Input != 10 {
		t.Errorf("executor.Session.Round result = %+v, want ending finish, the structured object and the partial usage", res)
	}
	if h := res.Handle; h.Executor != "claude-code" || h.Version != "1.2.3" || h.SessionID != "session-1" {
		t.Errorf("executor.RoundResult.Handle = %+v, want executor claude-code, version 1.2.3, session session-1", h)
	}
	if h := sess.Handle(); h != res.Handle {
		t.Errorf("executor.Session.Handle() = %+v, want the round's handle %+v", h, res.Handle)
	}
	if h := executor.Session.Handle(sess); h.SessionID != "session-1" {
		t.Errorf("executor.Session.Handle(sess).SessionID = %q, want session-1", h.SessionID)
	}
	if _, err := executor.Session.Round(sess, context.Background(), executor.Input{}); err != nil || len(session.inputs) != 2 || session.inputs[1].Feedback != "" {
		t.Errorf("executor.Session.Round(sess, ctx, Input{}) = %v with inputs %+v, want a first-round input without feedback", err, session.inputs)
	}
	if err := sess.Close(); err != nil || session.closed != 1 {
		t.Errorf("executor.Session.Close() = %v after %d closes, want nil after 1", err, session.closed)
	}
	if err := executor.Session.Close(sess); err != nil || session.closed != 2 {
		t.Errorf("executor.Session.Close(sess) = %v after %d closes, want nil after 2", err, session.closed)
	}
}

// TestExecutorUsage pins the usage record shape/external.go sums and writes to the run log as
// JSON: pointer counts that stay null when not reported (the cache_write count too, which
// shaping does not set or read), the state and source words, and the
// optional cost and model.
func TestExecutorUsage(t *testing.T) {
	var u executor.Usage
	var input, cached, output, reasoning *int = u.Input, u.Cached, u.Output, u.Reasoning
	var cost *float64 = u.CostUSD
	var state, source, model string = u.State, u.Source, u.Model
	if input != nil || cached != nil || output != nil || reasoning != nil || cost != nil || state != "" || source != "" || model != "" {
		t.Errorf("zero executor.Usage = %+v, want nothing reported", u)
	}
	if b, _ := json.Marshal(u); string(b) != `{"input":null,"cached":null,"cache_write":null,"output":null,"reasoning":null,"state":"","source":""}` {
		t.Errorf("executor.Usage zero JSON = %s, want null counts (cache_write included) and no cost_usd or model", b)
	}
	full := executor.Usage{Input: ptr(1), Cached: ptr(2), Output: ptr(3), Reasoning: ptr(4), State: executor.UsagePartial, Source: "harness", CostUSD: ptr(0.25), Model: "m"}
	if b, _ := json.Marshal(full); string(b) != `{"input":1,"cached":2,"cache_write":null,"output":3,"reasoning":4,"state":"partial","source":"harness","cost_usd":0.25,"model":"m"}` {
		t.Errorf("executor.Usage JSON = %s, want every field named input, cached, cache_write (null when unreported), output, reasoning, state, source, cost_usd, model", b)
	}
	var back executor.Usage
	if err := json.Unmarshal([]byte(`{"input":5,"state":"complete","source":"provider"}`), &back); err != nil || *back.Input != 5 || back.Cached != nil || back.State != "complete" || back.Source != "provider" {
		t.Errorf("executor.Usage decoded = %+v, %v, want input 5, other counts unreported", back, err)
	}
}
