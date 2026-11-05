package consumes

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// Compile-time pins: the types and signatures kogen-shaping's code relies on.
var (
	_ func(string) *provider.Client                                                                                          = provider.NewClient
	_ func(*provider.Client, provider.Request) map[string]any                                                                = (*provider.Client).Body
	_ func(*provider.Client, context.Context, *sys.Session, provider.Request, provider.Observer) (*provider.Response, error) = (*provider.Client).Respond
	_ func(*provider.Client) *string                                                                                         = func(c *provider.Client) *string { return &c.Version }
	_ func(stable, dynamic string) []any                                                                                     = provider.ContractBlocks
	_ func() string                                                                                                          = provider.NewCacheID
	_ func(string, []provider.Role) error                                                                                    = provider.RequireLogin
	_ func(provider.Role) bool                                                                                               = provider.SupportsToolChoice
	_ func(provider.KnownUsage) map[string]any                                                                               = provider.KnownUsage.Map
	_ func(provider.Usage, provider.Usage) provider.Usage                                                                    = provider.Usage.Add
	_ func(provider.Usage) string                                                                                            = provider.Usage.Line
	_ interface{ Attempt(record map[string]any) }                                                                            = provider.Observer(nil)
	_ func(*provider.Role) (*string, *string, *string, *string)                                                              = func(r *provider.Role) (*string, *string, *string, *string) {
		return &r.Provider, &r.Model, &r.Effort, &r.BaseURL
	}
	_ func(*provider.Usage) (*int, *int, *int, *int) = func(u *provider.Usage) (*int, *int, *int, *int) {
		return &u.Input, &u.Cached, &u.Output, &u.Reasoning
	}
	_ func(*provider.Request) (*provider.Role, *string, *[]any, *[]any, *string, *string, *any, *map[string]any) = func(r *provider.Request) (*provider.Role, *string, *[]any, *[]any, *string, *string, *any, *map[string]any) {
		return &r.Role, &r.Instructions, &r.Input, &r.Tools, &r.ToolChoice, &r.CacheID, &r.Scope, &r.Context
	}
	_ func(*provider.Response) (*[]map[string]any, *provider.Usage, *provider.KnownUsage) = func(r *provider.Response) (*[]map[string]any, *provider.Usage, *provider.KnownUsage) {
		return &r.Items, &r.Usage, &r.Known
	}
)

const (
	chatgptRole = "chatgpt"
	hexSuffix   = `[0-9a-f]{16}`
)

var userInput = []any{map[string]any{"role": "user", "content": "hi"}}

// recorder is a provider.Observer that keeps the records it receives.
type recorder struct{ records []map[string]any }

func (r *recorder) Attempt(record map[string]any) { r.records = append(r.records, record) }

// loginSession is a Session whose HOME holds no login, in a working directory of its own.
func loginSession(t *testing.T) *sys.Session {
	t.Helper()
	s := sys.New()
	s.Env = []string{"HOME=" + t.TempDir()}
	s.Cwd = t.TempDir()
	return s
}

func wantProviderFailure(t *testing.T, item string, err error, reason string, exit int) *sys.Error {
	t.Helper()
	e, ok := sys.AsError(err)
	if !ok {
		t.Fatalf("%s: error %v is not a *sys.Error", item, err)
	}
	if e.Class != "provider" || e.Reason != reason || e.Exit != exit {
		t.Errorf("%s: refused as %s/%s exit %d, want provider/%s exit %d (%v)", item, e.Class, e.Reason, e.Exit, reason, exit, err)
	}
	return e
}

func TestUsageAddLine(t *testing.T) {
	a := provider.Usage{Input: 100, Cached: 40, Output: 10, Reasoning: 4}
	b := provider.Usage{Input: 7, Cached: 3, Output: 5, Reasoning: 1}
	sum := a.Add(b)
	if want := (provider.Usage{Input: 107, Cached: 43, Output: 15, Reasoning: 5}); sum != want {
		t.Errorf("provider.Usage.Add: %+v + %+v = %+v, want %+v", a, b, sum, want)
	}
	if a.Add(provider.Usage{}) != a {
		t.Errorf("provider.Usage.Add: adding the zero Usage changed %+v", a)
	}
	var total provider.Usage
	for _, u := range []provider.Usage{a, b, a} {
		total = total.Add(u)
	}
	if want := (provider.Usage{Input: 207, Cached: 83, Output: 25, Reasoning: 9}); total != want {
		t.Errorf("provider.Usage.Add: accumulating gave %+v, want %+v", total, want)
	}
	if got, want := a.Line(), "input=100 cached=40 output=10 reasoning=4"; got != want {
		t.Errorf("provider.Usage.Line: %q, want %q", got, want)
	}
	if got, want := (provider.Usage{}).Line(), "input=0 cached=0 output=0 reasoning=0"; got != want {
		t.Errorf("provider.Usage.Line: zero Usage is %q, want %q", got, want)
	}
	if a.Input != 100 || a.Cached != 40 || a.Output != 10 || a.Reasoning != 4 {
		t.Errorf("provider.Usage.Input/Cached/Output/Reasoning: %+v does not keep its fields", a)
	}
}

func TestKnownUsageMap(t *testing.T) {
	in, cached, out := 120, 20, 30
	m := provider.KnownUsage{Input: &in, Cached: &cached, Output: &out}.Map()
	want := map[string]any{"input": 120, "cached": 20, "write": nil, "write_30m": nil, "write_12h": nil, "output": 30, "reasoning": nil}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("provider.KnownUsage.Map: %v, want %v", m, want)
	}
	zero := 0
	m = provider.KnownUsage{Reasoning: &zero}.Map()
	if v, ok := m["reasoning"]; !ok || v != 0 || m["input"] != nil || len(m) != 7 {
		t.Errorf("provider.KnownUsage.Map: a reported zero must stay 0 and an unreported category nil, got %v", m)
	}
}

func TestResponseFields(t *testing.T) {
	in := 9
	resp := &provider.Response{
		Items: []map[string]any{{"type": "function_call", "name": "read_file", "call_id": "c1", "arguments": `{"path":"a"}`}},
		Usage: provider.Usage{Input: 9, Output: 2},
		Known: provider.KnownUsage{Input: &in},
	}
	if calls := tools.Calls(resp.Items); len(calls) != 1 || calls[0].Name != "read_file" {
		t.Errorf("provider.Response.Items: tools.Calls of %v gave %v", resp.Items, calls)
	}
	if got := resp.Usage.Line(); got != "input=9 cached=0 output=2 reasoning=0" {
		t.Errorf("provider.Response.Usage: line %q", got)
	}
	if got := resp.Known.Map()["input"]; got != 9 {
		t.Errorf("provider.Response.Known: input %v, want 9", got)
	}
}

func TestNewClientVersion(t *testing.T) {
	const line = "kogen dev+12345678 (test)"
	c := provider.NewClient(line)
	if c == nil || c.Version != line {
		t.Fatalf("provider.NewClient: %+v does not keep the version line %q", c, line)
	}
	if f := strings.Fields(c.Version); len(f) != 3 || f[1] != "dev+12345678" {
		t.Errorf("provider.Client.Version: second word of %q is not the Kogen version", c.Version)
	}
	if empty := provider.NewClient(""); empty == nil || empty.Version != "" {
		t.Errorf("provider.NewClient: NewClient(\"\") gave %+v, want a client with no version", empty)
	}
	var client *provider.Client = provider.NewClient("")
	if client.Body(provider.Request{Role: provider.Role{Model: "m"}, Input: userInput}) == nil {
		t.Errorf("provider.Client: a new client cannot render a request")
	}
}

func TestRoleRoutesBody(t *testing.T) {
	c := provider.NewClient("")
	req := func(role provider.Role) provider.Request {
		return provider.Request{Role: role, Instructions: "inst", Input: userInput, CacheID: "conversation-1"}
	}
	body := c.Body(req(provider.Role{Provider: "chatgpt", Model: "gpt-x", Effort: "high"}))
	if body["model"] != "gpt-x" {
		t.Errorf("provider.Role.Model: body model %v, want gpt-x", body["model"])
	}
	if r, _ := body["reasoning"].(map[string]any); r["effort"] != "high" {
		t.Errorf("provider.Role.Effort: body reasoning %v, want effort high", body["reasoning"])
	}
	if _, ok := body["input"]; !ok {
		t.Errorf("provider.Role.Provider: chatgpt body %v is not a Responses body", body)
	}
	legacy := c.Body(req(provider.Role{Model: "gpt-x", Effort: "high"}))
	if jsonOf(t, legacy) != jsonOf(t, body) {
		t.Errorf("provider.Role.Provider: a role without provider must route like chatgpt:\n%s\n%s", jsonOf(t, legacy), jsonOf(t, body))
	}
	anthropic := c.Body(req(provider.Role{Provider: "anthropic", Model: "claude-opus-5-5", Effort: "high"}))
	if _, ok := anthropic["messages"]; !ok || anthropic["model"] != "claude-opus-5-5" || anthropic["system"] == nil {
		t.Errorf("provider.Role.Provider: anthropic body %v is not a Messages body", anthropic)
	}
	compatible := c.Body(req(provider.Role{Provider: "openai-compatible", Model: "local", BaseURL: "http://127.0.0.1:1"}))
	if msgs, _ := compatible["messages"].([]any); len(msgs) != 2 || compatible["model"] != "local" || compatible["stream"] != false {
		t.Errorf("provider.Role.BaseURL: openai-compatible body %v is not a Chat Completions body", compatible)
	}
}

func TestRequestBody(t *testing.T) {
	c := provider.NewClient("")
	tool := map[string]any{"type": "function", "name": "read_file"}
	req := provider.Request{
		Role:         provider.Role{Provider: "chatgpt", Model: "gpt-x", Effort: "low"},
		Instructions: "be brief",
		Input:        userInput,
		Tools:        []any{tool},
		ToolChoice:   "read_file",
		CacheID:      "conversation-fixed",
		Scope:        "evidence-dir",
		Context:      map[string]any{"phase": "shape", "role": "shape", "slug": "s"},
	}
	body := c.Body(req)
	if body["instructions"] != "be brief" {
		t.Errorf("provider.Request.Instructions: body instructions %v", body["instructions"])
	}
	if jsonOf(t, body["input"]) != jsonOf(t, userInput) {
		t.Errorf("provider.Request.Input: body input %s", jsonOf(t, body["input"]))
	}
	if jsonOf(t, body["tools"]) != jsonOf(t, []any{tool}) {
		t.Errorf("provider.Request.Tools: body tools %s", jsonOf(t, body["tools"]))
	}
	if got, want := jsonOf(t, body["tool_choice"]), `{"name":"read_file","type":"function"}`; got != want {
		t.Errorf("provider.Request.ToolChoice: body tool_choice %s, want %s", got, want)
	}
	if body["prompt_cache_key"] != "conversation-fixed" {
		t.Errorf("provider.Request.CacheID: body prompt_cache_key %v, want conversation-fixed", body["prompt_cache_key"])
	}
	if jsonOf(t, c.Body(req)) != jsonOf(t, body) {
		t.Errorf("provider.Client.Body: two renderings of one request differ")
	}

	noContext := req
	noContext.Context = nil
	if jsonOf(t, c.Body(noContext)) != jsonOf(t, body) {
		t.Errorf("provider.Request.Context: the context for attempt records changed the body")
	}
	noChoice := req
	noChoice.ToolChoice = ""
	if got := c.Body(noChoice)["tool_choice"]; got != "auto" {
		t.Errorf("provider.Request.ToolChoice: an empty ToolChoice sent tool_choice %v, want auto", got)
	}

	scoped := func(scope any) string {
		r := req
		r.CacheID, r.Scope = "", scope
		key, _ := c.Body(r)["prompt_cache_key"].(string)
		return key
	}
	first := scoped("dir-a")
	if first == "" || first == "conversation-fixed" || scoped("dir-a") != first {
		t.Errorf("provider.Request.Scope: one scope must keep one cache identity, got %q then %q", first, scoped("dir-a"))
	}
	if other := scoped("dir-b"); other == first || other == "" {
		t.Errorf("provider.Request.Scope: scopes dir-a and dir-b share the cache identity %q", first)
	}
}

func TestClientRespondRefusals(t *testing.T) {
	c := provider.NewClient("")
	ctx := context.Background()
	obs := &recorder{}
	respond := func(req provider.Request) (*provider.Response, error) {
		return c.Respond(ctx, loginSession(t), req, obs)
	}

	resp, err := c.Respond(ctx, nil, provider.Request{Role: provider.Role{Model: "m"}}, obs)
	if e, ok := sys.AsError(err); resp != nil || !ok || e.Class != "validation" || e.Reason != "missing_session" || e.Exit != 2 {
		t.Errorf("provider.Client.Respond: without a session got %v, %v, want validation/missing_session exit 2", resp, err)
	}

	resp, err = respond(provider.Request{Role: provider.Role{Provider: "no-such-provider"}, Input: userInput})
	e := wantProviderFailure(t, "provider.Client.Respond (unknown provider)", err, "refused", 4)
	if resp != nil || e.Detail != "unknown provider no-such-provider" {
		t.Errorf("provider.Client.Respond: unknown provider gave %v, %q", resp, e.Detail)
	}

	_, err = respond(provider.Request{Role: provider.Role{Provider: "openai-compatible", Model: "m"}, Input: userInput})
	e = wantProviderFailure(t, "provider.Client.Respond (no base URL)", err, "refused", 4)
	if !strings.Contains(e.Detail, "base_url") {
		t.Errorf("provider.Role.BaseURL: openai-compatible without a base URL was refused as %q, which does not name base_url", e.Detail)
	}

	_, err = respond(provider.Request{
		Role:  provider.Role{Provider: "anthropic", Model: "claude-opus-5-5"},
		Input: userInput, Tools: []any{map[string]any{"type": "function", "name": "f"}}, ToolChoice: "f",
	})
	wantProviderFailure(t, "provider.Request.ToolChoice (unsupported route)", err, "unsupported_capability", 4)

	_, err = respond(provider.Request{Role: provider.Role{Model: "gpt-x"}, Input: userInput, Scope: "s", Context: map[string]any{"phase": "shape"}})
	e = wantProviderFailure(t, "provider.Client.Respond (not signed in)", err, "login", 4)
	if want := "not signed in; run kogen provider login chatgpt"; e.Detail != want {
		t.Errorf("provider.Client.Respond: without a login the detail is %q, want %q", e.Detail, want)
	}
	if len(obs.records) != 0 {
		t.Errorf("provider.Observer: refusals before any transport attempt delivered %d records: %v", len(obs.records), obs.records)
	}
}

func TestRequireLogin(t *testing.T) {
	home := t.TempDir()
	err := provider.RequireLogin(home, []provider.Role{{Model: "gpt-x"}})
	e := wantProviderFailure(t, "provider.RequireLogin", err, "login", 4)
	if got, want := err.Error(), "provider/login: not signed in; run kogen provider login chatgpt"; got != want {
		t.Errorf("provider.RequireLogin: error %q, want %q", got, want)
	}
	if e.Detail != "not signed in; run kogen provider login chatgpt" {
		t.Errorf("provider.RequireLogin: detail %q", e.Detail)
	}
	if again := provider.RequireLogin(home, []provider.Role{{Provider: "chatgpt"}, {}}); again == nil || again.Error() != err.Error() {
		t.Errorf("provider.RequireLogin: roles of one provider must refuse once, as %q, got %v", err, again)
	}

	err = provider.RequireLogin(home, []provider.Role{{Provider: "openai-compatible", BaseURL: "http://127.0.0.1:1"}, {Provider: "anthropic", Model: "claude-opus-5-5"}})
	e = wantProviderFailure(t, "provider.RequireLogin (anthropic)", err, "login", 4)
	if !strings.HasSuffix(e.Detail, "run kogen provider login anthropic") {
		t.Errorf("provider.RequireLogin: detail %q does not name the login command of anthropic", e.Detail)
	}

	for name, roles := range map[string][]provider.Role{
		"no roles":                 nil,
		"keyless openai-compat.":   {{Provider: "openai-compatible", Model: "m", BaseURL: "http://127.0.0.1:1"}},
		"unknown provider ignored": {{Provider: "no-such-provider"}},
	} {
		if err := provider.RequireLogin(home, roles); err != nil {
			t.Errorf("provider.RequireLogin: %s must not need a login, got %v", name, err)
		}
	}
	var asErr *sys.Error
	if !errors.As(provider.RequireLogin(home, []provider.Role{{}}), &asErr) {
		t.Errorf("provider.RequireLogin: the refusal is not a *sys.Error")
	}
}

func TestSupportsToolChoice(t *testing.T) {
	for _, tc := range []struct {
		role provider.Role
		want bool
	}{
		{provider.Role{}, true},
		{provider.Role{Provider: "chatgpt", Model: "gpt-x"}, true},
		{provider.Role{Provider: "supergrok", Model: "grok-4.7"}, true},
		{provider.Role{Provider: "opencode-go", Model: "kimi-k3"}, true},
		{provider.Role{Provider: "openai-compatible", Model: "m", BaseURL: "http://127.0.0.1:1"}, true},
		{provider.Role{Provider: "anthropic", Model: "claude-opus-5-5"}, false},
		{provider.Role{Provider: "anthropic", Model: "claude-sonnet-5-5", Effort: "low"}, false},
		{provider.Role{Provider: "no-such-provider"}, false},
	} {
		if got := provider.SupportsToolChoice(tc.role); got != tc.want {
			t.Errorf("provider.SupportsToolChoice(%+v) = %v, want %v", tc.role, got, tc.want)
		}
	}
}

func TestContractBlocks(t *testing.T) {
	got := provider.ContractBlocks("stable authority", "dynamic evidence")
	want := []any{
		map[string]any{"type": "input_text", "text": "stable authority"},
		map[string]any{"type": "input_text", "text": "dynamic evidence"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("provider.ContractBlocks: %v, want %v", got, want)
	}
	if got := provider.ContractBlocks("", ""); len(got) != 2 {
		t.Errorf("provider.ContractBlocks: empty texts gave %d blocks, want 2", len(got))
	}
}

func TestNewCacheID(t *testing.T) {
	id := provider.NewCacheID()
	if !regexp.MustCompile(`^conversation-`+hexSuffix+`$`).MatchString(id) || len(id) != 29 {
		t.Errorf("provider.NewCacheID: %q is not conversation-<16 hex> of 29 characters", id)
	}
	seen := map[string]bool{id: true}
	for range 50 {
		next := provider.NewCacheID()
		if seen[next] {
			t.Fatalf("provider.NewCacheID: %q repeated", next)
		}
		seen[next] = true
	}
	req := provider.Request{Role: provider.Role{Model: "m"}, Input: userInput, CacheID: id}
	if provider.NewClient("").Body(req)["prompt_cache_key"] != id {
		t.Errorf("provider.NewCacheID: %q is not accepted as a Request.CacheID", id)
	}
}

func TestObserverInterface(t *testing.T) {
	r := &recorder{}
	var o provider.Observer = r
	o.Attempt(map[string]any{"outcome": "completed", "wall_ms": int64(7)})
	if len(r.records) != 1 || r.records[0]["outcome"] != "completed" {
		t.Errorf("provider.Observer: the record was not delivered: %v", r.records)
	}
}
