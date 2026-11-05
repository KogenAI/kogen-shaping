package consumes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// Compile-time pins: the types and signatures kogen-shaping's code relies on.
var (
	_ func(map[string]any, ...provider.Role) (tools.Measurement, error)                                                            = tools.Measure
	_ func(*tools.Measurement) (*int, *bool)                                                                                       = func(m *tools.Measurement) (*int, *bool) { return &m.RenderedBytes, &m.Fits }
	_ func(text string) string                                                                                                     = tools.Cut
	_ func(arguments string) map[string]any                                                                                        = tools.DecodeArguments
	_ func(name string) (map[string]any, bool)                                                                                     = tools.Definition
	_ func([]map[string]any) []any                                                                                                 = tools.Replay
	_ func([]map[string]any) []tools.Call                                                                                          = tools.Calls
	_ func(tools.Call, tools.Output, string) map[string]any                                                                        = tools.ReadEvent
	_ func([]any, string, func(map[string]any), func() (*provider.Response, error)) (*provider.Response, error)                    = tools.Deliver
	_ func(tools.Call) map[string]any                                                                                              = tools.Call.Args
	_ func(tools.Call, any) map[string]any                                                                                         = tools.Call.Output
	_ func(*tools.Call) *string                                                                                                    = func(c *tools.Call) *string { return &c.Name }
	_ func(tools.Output) tools.Output                                                                                              = tools.Output.Cut
	_ func(*tools.Output) *string                                                                                                  = func(o *tools.Output) *string { return &o.Text }
	_ func(*tools.Documentation) *string                                                                                           = func(d *tools.Documentation) *string { return &d.UserAgent }
	_ func(*tools.Tools, string, map[string]any) (tools.Output, error)                                                             = (*tools.Tools).Run
	_ func(*tools.Tools) (*string, **sys.Session, *tools.Git, *[]string, *func(string, any) (string, error), *tools.Documentation) = func(t *tools.Tools) (*string, **sys.Session, *tools.Git, *[]string, *func(string, any) (string, error), *tools.Documentation) {
		return &t.Root, &t.Session, &t.Repo, &t.Enabled, &t.Probe, &t.Documentation
	}
)

var (
	_ string = tools.ReadFileTool
	_ string = tools.ListFilesTool
	_ string = tools.SearchTextTool
	_ string = tools.ViewImageTool
	_ string = tools.DocumentationTool
	_ int    = tools.InputBudgetBytes
	_ int    = tools.OutputReserveTokens
	_ int    = provider.InputBudgetBytes
)

// fakeGit is the Git of Tools.Repo: it answers every command with its fixed result and keeps
// the commands it was asked to run.
type fakeGit struct {
	result tools.GitResult
	runs   [][]string
	tree   string
}

func (g *fakeGit) Run(args ...string) (tools.GitResult, error) {
	g.runs = append(g.runs, args)
	return g.result, nil
}
func (g *fakeGit) Head() string                          { return "headcommit" }
func (g *fakeGit) Tree(string, []string) (string, error) { return g.tree, nil }

// writePNG writes a static non-interlaced RGB8 PNG into root/name and returns its bytes.
func writePNG(t *testing.T, root, name string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for x := range 3 {
		for y := range 2 {
			img.Set(x, y, color.RGBA{R: uint8(60 * x), G: uint8(100 * y), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// call returns the one function call of an answer made of one function_call item.
func call(t *testing.T, id, name, arguments string) tools.Call {
	t.Helper()
	calls := tools.Calls([]map[string]any{{"type": "function_call", "call_id": id, "name": name, "arguments": arguments}})
	if len(calls) != 1 {
		t.Fatalf("tools.Calls: got %d calls, want 1", len(calls))
	}
	return calls[0]
}

func TestToolNames(t *testing.T) {
	for got, want := range map[string]string{
		tools.ReadFileTool:      "read_file",
		tools.ListFilesTool:     "list_files",
		tools.SearchTextTool:    "search_text",
		tools.ViewImageTool:     "view_image",
		tools.DocumentationTool: "documentation",
	} {
		if got != want {
			t.Errorf("tools.%sTool: %q, want %q", want, got, want)
		}
	}
}

func TestInputBudgetBytes(t *testing.T) {
	if provider.InputBudgetBytes != 524288 {
		t.Errorf("provider.InputBudgetBytes = %d, want 524288", provider.InputBudgetBytes)
	}
	if tools.InputBudgetBytes != provider.InputBudgetBytes {
		t.Errorf("tools.InputBudgetBytes = %d, want provider.InputBudgetBytes (%d)", tools.InputBudgetBytes, provider.InputBudgetBytes)
	}
	m, err := tools.Measure(map[string]any{})
	if err != nil || m.InputBudgetBytes != tools.InputBudgetBytes {
		t.Errorf("tools.InputBudgetBytes: Measure budgets %+v, %v, want %d", m, err, tools.InputBudgetBytes)
	}
}

func TestOutputReserveTokens(t *testing.T) {
	if tools.OutputReserveTokens != 131072 {
		t.Errorf("tools.OutputReserveTokens = %d, want 131072", tools.OutputReserveTokens)
	}
	m, err := tools.Measure(map[string]any{})
	if err != nil || m.OutputReserveTokens != tools.OutputReserveTokens {
		t.Errorf("tools.OutputReserveTokens: Measure reserves %+v, %v, want %d", m, err, tools.OutputReserveTokens)
	}
}

func TestDefinition(t *testing.T) {
	// The bytes of each definition, as JSON with sorted keys, as the pinned kogen-loop defines them
	// (list_files and search_text now say how a path that is not plain text is printed).
	want := map[string]string{
		tools.ReadFileTool:      "8b7feea4738bbd82843a67aabbd538fa79e4beae6c15424eac0797663a1cefaa",
		tools.ListFilesTool:     "c0dc3b70c589de3ee7d2b338d92b99591260bd9d58f949d53a5da8d4500c5eb2",
		tools.SearchTextTool:    "06eace1e8df95233ab8379de5ca28495294570b3b693fd3fbf3ae327db22c955",
		tools.ViewImageTool:     "8fcab128c90478fbd42ff2995fc1c5b96294bb4b7aee9539d8e86037d6550287",
		tools.DocumentationTool: "14177cc5761e23a96700576afdff9bbc99d34bec2aeac7880fb3c2c5b751ca68",
	}
	required := map[string]string{
		tools.ReadFileTool: `["path"]`, tools.ListFilesTool: `[]`, tools.SearchTextTool: `["pattern"]`,
		tools.ViewImageTool: `["path"]`, tools.DocumentationTool: `["operation"]`,
	}
	for name, sum := range want {
		def, ok := tools.Definition(name)
		if !ok {
			t.Errorf("tools.Definition(%q): not a defined tool", name)
			continue
		}
		if def["type"] != "function" || def["name"] != name || def["strict"] != false {
			t.Errorf("tools.Definition(%q): %v is not a non-strict function definition of that name", name, def)
		}
		if text, _ := def["description"].(string); text == "" {
			t.Errorf("tools.Definition(%q): no description", name)
		}
		params, _ := def["parameters"].(map[string]any)
		if params["type"] != "object" || params["additionalProperties"] != false || jsonOf(t, params["required"]) != required[name] {
			t.Errorf("tools.Definition(%q): parameters %s, want an object requiring %s", name, jsonOf(t, params), required[name])
		}
		got := sha256.Sum256([]byte(jsonOf(t, def)))
		if hex.EncodeToString(got[:]) != sum {
			t.Errorf("tools.Definition(%q): the definition bytes changed: %s", name, jsonOf(t, def))
		}
	}
	for _, name := range []string{"", "probe", "submit", "finish", "READ_FILE"} {
		if def, ok := tools.Definition(name); ok || def != nil {
			t.Errorf("tools.Definition(%q) = %v, %v, want nil, false: only generic tools are defined here", name, def, ok)
		}
	}
}

func TestCallsAndOutput(t *testing.T) {
	items := []map[string]any{
		{"type": "reasoning", "id": "r1"},
		{"type": "function_call", "call_id": "c1", "name": "read_file", "arguments": `{"path":"a.txt","start_line":2}`},
		{"type": "message", "role": "assistant"},
		{"type": "function_call", "call_id": "c2", "name": "list_files"},
	}
	calls := tools.Calls(items)
	if len(calls) != 2 || calls[0].Name != "read_file" || calls[1].Name != "list_files" {
		t.Fatalf("tools.Calls: %+v, want the two function calls in order", calls)
	}
	args := calls[0].Args()
	if args["path"] != "a.txt" || len(args) != 2 {
		t.Errorf("tools.Call.Args: %v, want path a.txt and start_line", args)
	}
	if got := jsonOf(t, calls[1].Args()); got != `{"__invalid__":true}` {
		t.Errorf("tools.Call.Args: a call without arguments gave %s, want {\"__invalid__\":true}", got)
	}
	if got, want := jsonOf(t, calls[0].Output("file text")), `{"call_id":"c1","output":"file text","type":"function_call_output"}`; got != want {
		t.Errorf("tools.Call.Output: %s, want %s", got, want)
	}
	typed := tools.Output{Text: "ignored", Parts: []any{map[string]any{"type": "input_text", "text": "t"}}}
	if got := calls[1].Output(typed)["output"]; !reflect.DeepEqual(got, typed.Parts) {
		t.Errorf("tools.Call.Output: a typed Output must deliver its parts, got %v", got)
	}
	if got := tools.Calls(nil); len(got) != 0 {
		t.Errorf("tools.Calls: no items gave %v", got)
	}
}

func TestDecodeArguments(t *testing.T) {
	if got := tools.DecodeArguments(`{"summary":"s","n":3,"nested":{"a":[1,2]}}`); jsonOf(t, got) != `{"n":3,"nested":{"a":[1,2]},"summary":"s"}` {
		t.Errorf("tools.DecodeArguments: %s", jsonOf(t, got))
	}
	if got := tools.DecodeArguments(`{"summary":"x"}`)["summary"]; got != "x" {
		t.Errorf("tools.DecodeArguments: summary %v, want x", got)
	}
	for _, bad := range []string{"", "{", "[1]", `"s"`, "null", "7", `{"a":1} trailing`} {
		if got := jsonOf(t, tools.DecodeArguments(bad)); got != `{"__invalid__":true}` {
			t.Errorf("tools.DecodeArguments(%q) = %s, want {\"__invalid__\":true}", bad, got)
		}
	}
}

func TestReplay(t *testing.T) {
	items := []map[string]any{
		{"type": "reasoning", "id": "r1", "encrypted_content": "enc", "summary": []any{}, "status": "completed", "extra": 1},
		{"type": "function_call", "id": "f1", "call_id": "c1", "name": "read_file", "arguments": "{}", "status": "completed", "parsed": true},
		{"type": "message", "id": "m1", "role": "assistant", "phase": "final", "status": "completed", "content": []any{
			map[string]any{"type": "output_text", "text": "hello", "annotations": []any{}},
			map[string]any{"type": "refusal", "refusal": "no"},
			map[string]any{"type": "other"},
		}},
		{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "other"}}},
		{"type": "web_search_call", "id": "w1"},
	}
	got := tools.Replay(items)
	want := []any{
		map[string]any{"type": "reasoning", "id": "r1", "encrypted_content": "enc", "summary": []any{}, "status": "completed"},
		map[string]any{"type": "function_call", "id": "f1", "call_id": "c1", "name": "read_file", "arguments": "{}", "status": "completed"},
		map[string]any{"type": "message", "id": "m1", "role": "assistant", "phase": "final", "status": "completed", "content": []any{
			map[string]any{"type": "output_text", "text": "hello"},
			map[string]any{"type": "refusal", "refusal": "no"},
		}},
	}
	if jsonOf(t, got) != jsonOf(t, want) {
		t.Errorf("tools.Replay:\n got %s\nwant %s", jsonOf(t, got), jsonOf(t, want))
	}
	if got := tools.Replay(nil); len(got) != 0 {
		t.Errorf("tools.Replay: no items replayed %v", got)
	}
}

// cutMarker is the omission marker: "[... N bytes omitted ...]", which counts the complete lines
// omitted too ("[... L lines, N bytes omitted ...]", with their numbers for numbered text) when
// the cut falls on line boundaries.
var cutMarker = regexp.MustCompile(`\n\[\.\.\. (?:\d+ lines(?: \(\d+-\d+\))?, )?(\d+) bytes omitted \.\.\.\]\n`)

func TestCut(t *testing.T) {
	for _, text := range []string{"", "short", strings.Repeat("x", 16384), strings.Repeat("é", 8192)} {
		if got := tools.Cut(text); got != text {
			t.Errorf("tools.Cut: text of %d bytes within the bound changed", len(text))
		}
	}
	unit := "0123456789abcdef\n"
	for _, text := range []string{
		strings.Repeat("x", 16385),
		strings.Repeat(unit, 5000),
		strings.Repeat("é", 20000),
		strings.Repeat("日本", 10000),
	} {
		got := tools.Cut(text)
		loc := cutMarker.FindStringSubmatchIndex(got)
		if loc == nil {
			t.Errorf("tools.Cut: %d bytes cut without an omission marker", len(text))
			continue
		}
		head, tail := got[:loc[0]], got[loc[1]:]
		omitted, _ := strconv.Atoi(got[loc[2]:loc[3]])
		switch {
		case len(got) > 16384:
			t.Errorf("tools.Cut: result of %d bytes is above 16384", len(got))
		case !utf8.ValidString(got):
			t.Errorf("tools.Cut: the result is not valid UTF-8")
		case !strings.HasPrefix(text, head) || !strings.HasSuffix(text, tail) || len(head) < 4096-len(unit) || len(head) > 4096:
			t.Errorf("tools.Cut: head of %d bytes and tail of %d bytes are not the start and end of the text", len(head), len(tail))
		case omitted != len(text)-len(head)-len(tail):
			t.Errorf("tools.Cut: marker says %d bytes omitted, %d are absent", omitted, len(text)-len(head)-len(tail))
		}
		if again := tools.Cut(got); again != got {
			t.Errorf("tools.Cut: cutting a cut text changed it")
		}
	}
	if got, want := tools.Cut("a\xffb"), "a�b"; got != want {
		t.Errorf("tools.Cut: invalid UTF-8 gave %q, want %q", got, want)
	}
}

func TestOutputCut(t *testing.T) {
	long := strings.Repeat("line\n", 10000)
	cut := tools.Output{Text: long}.Cut()
	if cut.Text != tools.Cut(long) || len(cut.Text) > 16384 {
		t.Errorf("tools.Output.Cut: text of %d bytes was not cut as tools.Cut does (%d bytes)", len(long), len(cut.Text))
	}
	parts := []any{map[string]any{"type": "input_image", "image_url": strings.Repeat("A", 40000)}}
	typed := tools.Output{Parts: parts}.Cut()
	if !reflect.DeepEqual(typed.Parts, parts) {
		t.Errorf("tools.Output.Cut: typed parts must pass unchanged")
	}
	if short := (tools.Output{Text: "ok"}).Cut(); short.Text != "ok" {
		t.Errorf("tools.Output.Text: a short text became %q", short.Text)
	}
}

func TestMeasure(t *testing.T) {
	body := map[string]any{"b": []any{"<&>", 1, 2.5, true, nil}, "a": map[string]any{"z": "q\"\n"}}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(body); err != nil {
		t.Fatal(err)
	}
	m, err := tools.Measure(body)
	if err != nil || m.RenderedBytes != buf.Len()-1 || !m.Fits {
		t.Errorf("tools.Measurement.RenderedBytes: %+v, %v, want %d bytes of compact JSON that fits", m, err, buf.Len()-1)
	}
	role := provider.Role{Provider: "chatgpt", Model: "model-without-profile", Effort: "high"}
	if withRole, err := tools.Measure(body, role); err != nil || withRole.RenderedBytes != m.RenderedBytes {
		t.Errorf("tools.Measure: with a role %+v, %v, want the same size %d", withRole, err, m.RenderedBytes)
	}

	// {"x":"…"} is 8 bytes around the text: the budget is inclusive.
	for _, tc := range []struct {
		extra int
		fits  bool
	}{{-1, true}, {0, true}, {1, false}} {
		n := tools.InputBudgetBytes - 8 + tc.extra
		got, err := tools.Measure(map[string]any{"x": strings.Repeat("a", n)}, role)
		if err != nil || got.RenderedBytes != n+8 || got.Fits != tc.fits {
			t.Errorf("tools.Measurement.Fits: %d bytes against the budget of %d gave %+v, %v, want Fits=%v", n+8, tools.InputBudgetBytes, got, err, tc.fits)
		}
	}
	if _, err := tools.Measure(map[string]any{"f": make(chan int)}); err == nil {
		t.Errorf("tools.Measure: a body that is not JSON gave no error")
	}

	c := provider.NewClient("")
	req := provider.Request{Role: role, Instructions: "inst", Input: userInput, CacheID: "conversation-1"}
	sent, err := tools.Measure(c.Body(req), req.Role)
	if err != nil || sent.RenderedBytes < len("inst")+len("hi") || !sent.Fits {
		t.Errorf("tools.Measure: the body of a small request measured %+v, %v", sent, err)
	}
	req.Input = []any{map[string]any{"role": "user", "content": strings.Repeat("a", tools.InputBudgetBytes)}}
	if sent, err := tools.Measure(c.Body(req), req.Role); err != nil || sent.Fits {
		t.Errorf("tools.Measurement.Fits: a request of %d bytes must not fit, got %+v, %v", sent.RenderedBytes, sent, err)
	}
}

func TestToolsRun(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := &tools.Tools{Root: root, Enabled: []string{tools.ReadFileTool}}
	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{tools.ReadFileTool, map[string]any{"path": "a.txt"}, "1\tone\n2\ttwo\n3\tthree"},
		{tools.ReadFileTool, map[string]any{"path": "a.txt", "start_line": int64(2), "end_line": int64(2)}, "2\ttwo"},
		{tools.ReadFileTool, map[string]any{"path": "missing.txt"}, "error: missing.txt does not exist"},
		{tools.ReadFileTool, map[string]any{"path": 7}, "error: argument path must be a string"},
		{tools.ListFilesTool, map[string]any{}, "error: unknown tool list_files"},
		{"write_file", map[string]any{"path": "b.txt", "content": "x"}, "error: unknown tool write_file"},
	} {
		out, err := tl.Run(tc.name, tc.args)
		if err != nil || out.Text != tc.want || out.Typed() {
			t.Errorf("tools.Tools.Run(%q, %v) = %+v, %v, want text %q", tc.name, tc.args, out, err, tc.want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "b.txt")); err == nil {
		t.Errorf("tools.Tools.Enabled: a tool that is not enabled wrote a file")
	}
	outside, err := tl.Run(tools.ReadFileTool, map[string]any{"path": "../a.txt"})
	if err != nil || outside.Text != "error: ../a.txt: path is outside the repository root; reads are confined to the repository root" {
		t.Errorf("tools.Tools.Root: a path leaving the root gave %+v, %v, want the error that reads are confined to the root", outside, err)
	}

	c := call(t, "c9", tools.ReadFileTool, `{"path":"a.txt","end_line":1}`)
	out, err := tl.Run(c.Name, c.Args())
	if err != nil || out.Text != "1\tone" {
		t.Errorf("tools.Tools.Run: a decoded call gave %+v, %v", out, err)
	}
}

func TestToolsRepo(t *testing.T) {
	root := t.TempDir()
	git := &fakeGit{result: tools.GitResult{Stdout: "a.txt\x00sub/b.txt\x00"}}
	var repo tools.Git = git
	tl := &tools.Tools{Root: root, Repo: repo, Enabled: []string{tools.ListFilesTool, tools.SearchTextTool}}

	out, err := tl.Run(tools.ListFilesTool, map[string]any{})
	if err != nil || out.Text != "a.txt\nsub/b.txt" {
		t.Errorf("tools.Tools.Repo: list_files gave %+v, %v, want the files Git listed", out, err)
	}
	if len(git.runs) != 1 || git.runs[0][0] != "ls-files" {
		t.Errorf("tools.Tools.Repo: list_files ran %v, want one git ls-files", git.runs)
	}
	// git grep -z -n records, <path> NUL <number> NUL <text> LF; tools keeps the lines the pattern matches.
	git.result = tools.GitResult{Stdout: "a.txt\x001\x00needle\na.txt\x002\x00hay\n"}
	out, err = tl.Run(tools.SearchTextTool, map[string]any{"pattern": "needle"})
	if err != nil || out.Text != "a.txt:1:needle" || git.runs[1][0] != "grep" {
		t.Errorf("tools.Tools.Repo: search_text gave %+v, %v after %v", out, err, git.runs)
	}
	git.result = tools.GitResult{Status: 1}
	if out, _ := tl.Run(tools.SearchTextTool, map[string]any{"pattern": "absent"}); out.Text != "(no matches)" {
		t.Errorf("tools.Tools.Repo: search_text without a match gave %q, want (no matches)", out.Text)
	}
	git.result = tools.GitResult{}
	if out, _ := tl.Run(tools.ListFilesTool, map[string]any{}); out.Text != "(no files)" {
		t.Errorf("tools.Tools.Repo: list_files without files gave %q, want (no files)", out.Text)
	}
}

func TestToolsSession(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatalf("tools.Tools.Session: the suite needs git on PATH: %v", err)
	}
	root := t.TempDir()
	for name, text := range map[string]string{"a.txt": "alpha\nneedle here\n", "sub/b.txt": "beta\n"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := sys.New()
	s.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}
	s.Cwd = root
	if res, err := s.Run("git init -q", root, sys.Options{}); err != nil || res.Status != 0 {
		t.Fatalf("git init: %+v, %v", res, err)
	}
	tl := &tools.Tools{Root: root, Session: s, Enabled: []string{tools.ListFilesTool, tools.SearchTextTool}}
	out, err := tl.Run(tools.ListFilesTool, map[string]any{})
	if err != nil || out.Text != "a.txt\nsub/b.txt" {
		t.Errorf("tools.Tools.Session: list_files gave %+v, %v, want a.txt and sub/b.txt", out, err)
	}
	out, err = tl.Run(tools.SearchTextTool, map[string]any{"pattern": "needle"})
	if err != nil || out.Text != "a.txt:2:needle here" {
		t.Errorf("tools.Tools.Session: search_text gave %+v, %v, want a.txt:2:needle here", out, err)
	}
}

func TestToolsProbe(t *testing.T) {
	var gotCommand string
	var gotFiles any
	tl := &tools.Tools{
		Root: t.TempDir(), Enabled: []string{"probe"},
		Probe: func(command string, files any) (string, error) {
			gotCommand, gotFiles = command, files
			return "status 0", nil
		},
	}
	files := []any{map[string]any{"path": "p.txt", "content": "c"}}
	out, err := tl.Run("probe", map[string]any{"command": "true", "files": files})
	if err != nil || out.Text != "status 0" || gotCommand != "true" || !reflect.DeepEqual(gotFiles, files) {
		t.Errorf("tools.Tools.Probe: gave %+v, %v with command %q and files %v", out, err, gotCommand, gotFiles)
	}
	if _, err := tl.Run("probe", map[string]any{"command": "true"}); err != nil || gotFiles != nil {
		t.Errorf("tools.Tools.Probe: a call without files passed %v", gotFiles)
	}
	failing := errors.New("probe failed")
	tl.Probe = func(string, any) (string, error) { return "", failing }
	if _, err := tl.Run("probe", map[string]any{"command": "true"}); !errors.Is(err, failing) {
		t.Errorf("tools.Tools.Probe: the callback's error was returned as %v", err)
	}
	tl.Probe = nil
	if out, _ := tl.Run("probe", map[string]any{"command": "true"}); !strings.HasPrefix(out.Text, "error:") {
		t.Errorf("tools.Tools.Probe: without a callback got %q, want an error text", out.Text)
	}
}

func TestToolsDocumentation(t *testing.T) {
	tl := &tools.Tools{
		Root: t.TempDir(), Enabled: []string{tools.DocumentationTool},
		Documentation: tools.Documentation{UserAgent: "kogen/dev+12345678"},
	}
	// An operation that is neither search nor fetch is answered without any request.
	out, err := tl.Run(tools.DocumentationTool, map[string]any{"operation": "summarize", "query": "q"})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out.Text), &result); err != nil {
		t.Fatalf("tools.DocumentationTool: result %q is not JSON: %v", out.Text, err)
	}
	if result["status"] != "unsupported" || result["operation"] != "summarize" || result["query"] != "q" || result["error"] != "documentation source unsupported" {
		t.Errorf("tools.Documentation.UserAgent: an unsupported operation gave %v", result)
	}
	disabled := &tools.Tools{Root: t.TempDir()}
	if out, _ := disabled.Run(tools.DocumentationTool, map[string]any{"operation": "summarize"}); out.Text != "error: unknown tool documentation" {
		t.Errorf("tools.DocumentationTool: a tool that is not enabled gave %q", out.Text)
	}
}

func TestViewImageAndReadEvent(t *testing.T) {
	root := t.TempDir()
	data := writePNG(t, root, "shot.png")
	sum := sha256.Sum256(data)
	git := &fakeGit{tree: "tree0123"}
	tl := &tools.Tools{Root: root, Repo: git, Enabled: []string{tools.ViewImageTool}}

	c := call(t, "img1", tools.ViewImageTool, `{"path":"shot.png"}`)
	out, err := tl.Run(c.Name, c.Args())
	if err != nil || !out.Typed() || out.Text != "" || len(out.Parts) != 2 {
		t.Fatalf("tools.ViewImageTool: got %+v, %v, want typed parts of metadata text and image", out, err)
	}
	event := tools.ReadEvent(c, out, "shape")
	want := map[string]any{
		"event": "image_read", "owner": "shape", "call_id": "img1", "status": "read",
		"path": "shot.png", "format": "png", "bytes": float64(len(data)), "width": float64(3), "height": float64(2),
		"sha256": hex.EncodeToString(sum[:]), "read_candidate": "tree0123",
	}
	for key, value := range want {
		if jsonOf(t, event[key]) != jsonOf(t, value) {
			t.Errorf("tools.ReadEvent: %s is %v, want %v", key, event[key], value)
		}
	}
	if event == nil || len(event) < len(want) {
		t.Errorf("tools.ReadEvent: event %v lacks the image facts", event)
	}
	if cut := out.Cut(); !reflect.DeepEqual(cut.Parts, out.Parts) {
		t.Errorf("tools.Output.Cut: the pixels of view_image must pass unchanged")
	}

	text := tools.Output{Text: "plain text"}
	if got := tools.ReadEvent(c, text, "shape"); got != nil {
		t.Errorf("tools.ReadEvent: a text result gave %v, want nil", got)
	}
	refused, err := tl.Run(tools.ViewImageTool, map[string]any{"path": "../shot.png"})
	if err != nil || refused.Typed() || !strings.Contains(refused.Text, "path is outside the repository root") {
		t.Errorf("tools.ViewImageTool: a path outside the root gave %+v, %v, want the outside-root error", refused, err)
	}
	notPNG := filepath.Join(root, "fake.png")
	if err := os.WriteFile(notPNG, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if bad, _ := tl.Run(tools.ViewImageTool, map[string]any{"path": "fake.png"}); bad.Typed() || !strings.HasPrefix(bad.Text, "error: fake.png") {
		t.Errorf("tools.ViewImageTool: a file that is not a PNG gave %+v", bad)
	}
}

func TestDeliver(t *testing.T) {
	root := t.TempDir()
	data := writePNG(t, root, "shot.png")
	sum := sha256.Sum256(data)
	tl := &tools.Tools{Root: root, Enabled: []string{tools.ViewImageTool}}
	c := call(t, "img1", tools.ViewImageTool, `{"path":"shot.png"}`)
	pixels, err := tl.Run(c.Name, c.Args())
	if err != nil || !pixels.Typed() {
		t.Fatalf("tools.Tools.Run: %+v, %v", pixels, err)
	}
	withImage := []any{map[string]any{"role": "user", "content": "look"}, c.Output(pixels)}
	plain := []any{map[string]any{"role": "user", "content": "look"}}

	collect := func(input []any, send func() (*provider.Response, error)) (*provider.Response, error, []map[string]any) {
		var events []map[string]any
		resp, err := tools.Deliver(input, "shape", func(e map[string]any) { events = append(events, e) }, send)
		return resp, err, events
	}

	answer := &provider.Response{ID: "resp_1", Usage: provider.Usage{Input: 5}}
	resp, err, events := collect(withImage, func() (*provider.Response, error) { return answer, nil })
	if resp != answer || err != nil {
		t.Errorf("tools.Deliver: returned %v, %v, want the response of send unchanged", resp, err)
	}
	if len(events) != 1 {
		t.Fatalf("tools.Deliver: %d events for one image, want 1: %v", len(events), events)
	}
	e := events[0]
	if e["event"] != "image_delivery" || e["owner"] != "shape" || e["call_id"] != "img1" || e["status"] != "completed" ||
		e["response_id"] != "resp_1" || e["sha256"] != hex.EncodeToString(sum[:]) {
		t.Errorf("tools.Deliver: event %v is not the completed image_delivery of the image", e)
	}

	rejected := sys.Fail("provider", "request_failed", "rejected", 4)
	rejected.ProviderFailure = map[string]any{"http_status": 400}
	_, err, events = collect(withImage, func() (*provider.Response, error) { return nil, rejected })
	if err != error(rejected) || len(events) != 1 || events[0]["status"] != "failed" || events[0]["response_id"] != nil {
		t.Errorf("tools.Deliver: a rejected request gave %v, %v, want the error unchanged and a failed event", err, events)
	}
	lost := errors.New("connection lost")
	_, err, events = collect(withImage, func() (*provider.Response, error) { return nil, lost })
	if err != lost || len(events) != 1 || events[0]["status"] != "unknown" {
		t.Errorf("tools.Deliver: an unreadable answer gave %v, %v, want an unknown event", err, events)
	}

	sent := 0
	resp, err, events = collect(plain, func() (*provider.Response, error) { sent++; return answer, nil })
	if resp != answer || err != nil || sent != 1 || len(events) != 0 {
		t.Errorf("tools.Deliver: a request without images gave %v, %v, %d sends and events %v, want one send and no event", resp, err, sent, events)
	}
	resp, err = tools.Deliver(withImage, "shape", nil, func() (*provider.Response, error) { return answer, nil })
	if resp != answer || err != nil {
		t.Errorf("tools.Deliver: without an emit function gave %v, %v", resp, err)
	}
}
