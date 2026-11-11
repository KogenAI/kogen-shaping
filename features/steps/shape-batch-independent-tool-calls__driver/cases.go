//go:build ignore

package shape

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// These cases are injected by the shape-batch-independent-tool-calls acceptance driver into a
// disposable copy of the candidate. They drive the real Shape entry point through the existing
// scripted provider. Slow reads are FIFOs under the ignored .kogen/local folder: a read_file of
// one blocks until its writer, started by this test, answers.

const acceptBatchDir = ".kogen/local/acceptance-batch"

// acceptBatchGate serves FIFO reads. Every writer waits, after its reader has opened the FIFO,
// until want readers are open at once or two seconds pass; peak is how many readers were open
// when the first writer went on.
type acceptBatchGate struct {
	mu      sync.Mutex
	want    int
	arrived int
	peak    int
	release chan struct{}
	once    sync.Once
	logPath string
}

func newAcceptBatchGate(want int, logPath string) *acceptBatchGate {
	return &acceptBatchGate{want: want, release: make(chan struct{}), logPath: logPath}
}

func (g *acceptBatchGate) log(line string) {
	if g.logPath == "" {
		return
	}
	f, err := os.OpenFile(g.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		f.WriteString(line + "\n")
		f.Close()
	}
}

func (g *acceptBatchGate) arrive() {
	g.mu.Lock()
	g.arrived++
	if g.arrived == g.want {
		g.once.Do(func() { close(g.release) })
	}
	g.mu.Unlock()
	select {
	case <-g.release:
	case <-time.After(2 * time.Second):
	}
	g.mu.Lock()
	if g.peak == 0 {
		g.peak = g.arrived
	}
	g.mu.Unlock()
}

func (g *acceptBatchGate) peakOpen() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.peak
}

// serve makes root/rel a FIFO whose reader receives content delay after the gate opens.
func (g *acceptBatchGate) serve(t *testing.T, root, rel, name, content string, delay time.Duration) {
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Errorf("fixture: %v", err)
		return
	}
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Errorf("fixture: mkfifo %s: %v", rel, err)
		return
	}
	t.Cleanup(func() {
		if fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0); err == nil {
			syscall.Close(fd)
		}
	})
	go func() {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer f.Close()
		os.Remove(path)
		g.log(name + "-open")
		g.arrive()
		time.Sleep(delay)
		g.log(name + "-done")
		f.WriteString(content)
	}()
}

// acceptBatchOutputs are the call ids and texts of the function_call_output items of a request,
// in order.
func acceptBatchOutputs(req map[string]any) (ids []string, texts map[string]string) {
	texts = map[string]string{}
	input, _ := req["input"].([]any)
	for _, it := range input {
		m, _ := it.(map[string]any)
		if m["type"] == "function_call_output" {
			id := fmt.Sprint(m["call_id"])
			ids = append(ids, id)
			s, _ := m["output"].(string)
			texts[id] = s
		}
	}
	return ids, texts
}

// acceptBatchAfter are the outputs a request carries for the given call ids, in request order.
func acceptBatchAfter(t *testing.T, reqs []map[string]any, n int, ids ...string) ([]string, map[string]string) {
	t.Helper()
	if len(reqs) <= n {
		t.Fatalf("Shape sent %d shaper request(s); want at least %d", len(reqs), n+1)
	}
	all, texts := acceptBatchOutputs(reqs[n])
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var got []string
	for _, id := range all {
		if want[id] {
			got = append(got, id)
		}
	}
	return got, texts
}

func acceptBatchRead(id, path string, lines ...int) map[string]any {
	args := map[string]any{"path": path}
	if len(lines) == 2 {
		args["start_line"], args["end_line"] = lines[0], lines[1]
	}
	return shpCall(id, "read_file", args)
}

func acceptBatchProbe(id, name, log string) map[string]any {
	return shpCall(id, "probe", map[string]any{"command": fmt.Sprintf("printf '%[1]s-start\\n' >> '%[2]s'; sleep 0.4; printf '%[1]s-end\\n' >> '%[2]s'", name, log)})
}

func acceptBatchEnd(w *forceWire) []shpStep {
	return []shpStep{w.prose("one"), w.prose("two"), w.prose("three")}
}

func TestAcceptBatchOverlap(t *testing.T) {
	w := responsesWire
	steps := append([]shpStep{w.calls(
		acceptBatchRead("read-a", acceptBatchDir+"/a.txt"),
		shpCall("search", "search_text", map[string]any{"pattern": "tiny", "path": "README.md"}),
		acceptBatchRead("read-b", acceptBatchDir+"/b.txt"),
		acceptBatchRead("read-c", acceptBatchDir+"/c.txt"),
	)}, acceptBatchEnd(w)...)
	f := w.newShape(t, steps...)
	gate := newAcceptBatchGate(3, "")
	var once sync.Once
	f.Prov.route = func(map[string]any) *shpStep {
		once.Do(func() {
			// Completion order differs from call order: c, then b, then a.
			gate.serve(t, f.Root, acceptBatchDir+"/a.txt", "a", "alpha\n", 300*time.Millisecond)
			gate.serve(t, f.Root, acceptBatchDir+"/b.txt", "b", "beta\n", 200*time.Millisecond)
			gate.serve(t, f.Root, acceptBatchDir+"/c.txt", "c", "gamma\n", 100*time.Millisecond)
		})
		return nil
	}
	f.shape("Batch independent reads.")
	order, texts := acceptBatchAfter(t, f.Prov.requests(), 1, "read-a", "search", "read-b", "read-c")
	if got := strings.Join(order, ","); got != "read-a,search,read-b,read-c" {
		t.Fatalf("tool results in order %q, want read-a,search,read-b,read-c (the call order)", got)
	}
	for id, want := range map[string]string{"read-a": "1\talpha", "read-b": "1\tbeta", "read-c": "1\tgamma"} {
		if texts[id] != want {
			t.Fatalf("%s result %q, want %q", id, texts[id], want)
		}
	}
	if !strings.Contains(texts["search"], "A tiny calculator.") {
		t.Fatalf("search result %q does not contain the README match", texts["search"])
	}
	if p := gate.peakOpen(); p != 3 {
		t.Fatalf("%d of the 3 reads of one response were in flight together, want 3: the reads did not overlap", p)
	}
}

func TestAcceptBatchBarriers(t *testing.T) {
	w := responsesWire
	var f *shpFx
	log := filepath.Join(t.TempDir(), "order.log")
	steps := append([]shpStep{w.calls(
		acceptBatchProbe("probe-1", "p1", log),
		acceptBatchRead("read-x", acceptBatchDir+"/x.txt"),
		acceptBatchRead("read-y", acceptBatchDir+"/y.txt"),
		acceptBatchProbe("probe-2", "p2", log),
	)}, acceptBatchEnd(w)...)
	f = w.newShape(t, steps...)
	gate := newAcceptBatchGate(2, log)
	var once sync.Once
	f.Prov.route = func(map[string]any) *shpStep {
		once.Do(func() {
			gate.serve(t, f.Root, acceptBatchDir+"/x.txt", "x", "ex\n", 1500*time.Millisecond)
			gate.serve(t, f.Root, acceptBatchDir+"/y.txt", "y", "why\n", 1500*time.Millisecond)
		})
		return nil
	}
	f.shape("Keep probes in order.")
	order, texts := acceptBatchAfter(t, f.Prov.requests(), 1, "probe-1", "read-x", "read-y", "probe-2")
	if got := strings.Join(order, ","); got != "probe-1,read-x,read-y,probe-2" {
		t.Fatalf("tool results in order %q, want probe-1,read-x,read-y,probe-2 (the call order)", got)
	}
	if texts["read-x"] != "1\tex" || texts["read-y"] != "1\twhy" {
		t.Fatalf("read results %q and %q, want %q and %q", texts["read-x"], texts["read-y"], "1\tex", "1\twhy")
	}
	for _, id := range []string{"probe-1", "probe-2"} {
		if !strings.HasPrefix(texts[id], "exit status: 0\n") {
			t.Fatalf("%s result %q, want a completed probe", id, texts[id])
		}
	}
	b, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	at := map[string]int{}
	for i, l := range lines {
		at[l] = i
	}
	seq := strings.Join(lines, ",")
	for _, e := range []string{"p1-start", "p1-end", "x-open", "y-open", "x-done", "y-done", "p2-start", "p2-end"} {
		if _, ok := at[e]; !ok {
			t.Fatalf("event %s missing from %s", e, seq)
		}
	}
	if !(at["p1-end"] < at["x-open"] && at["p1-end"] < at["y-open"]) {
		t.Fatalf("a read started before the earlier probe ended: %s", seq)
	}
	if !(at["x-done"] < at["p2-start"] && at["y-done"] < at["p2-start"]) {
		t.Fatalf("the later probe started before the earlier reads finished: %s", seq)
	}
	if !(at["p1-end"] < at["p2-start"]) {
		t.Fatalf("the probes overlapped: %s", seq)
	}
	if p := gate.peakOpen(); p != 2 {
		t.Fatalf("%d of the 2 consecutive reads were in flight together, want 2: the reads did not overlap (%s)", p, seq)
	}
}

func acceptBatchNote(text string) string {
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("already read in this window, unchanged (%d bytes, sha %s)", len(text), hex.EncodeToString(sum[:])[:8])
}

func TestAcceptBatchRepeat(t *testing.T) {
	w := responsesWire
	readmeNote := os.Getenv("ACCEPT_BATCH_README_NOTE")
	if readmeNote == "" {
		t.Fatal("ACCEPT_BATCH_README_NOTE is not set")
	}
	intent := ".kogen/intents/is-even/intent.md"
	custody := ".kogen/local/shaping/is-even/proposed/" + intent
	changed := strings.Replace(shpScopedIntentMD, "returns True for even integers.", "returns True for every even integer.", 1)
	propose := func(id, content string) shpStep {
		return w.calls(shpCall(id, "propose_files", map[string]any{"files": []any{map[string]any{"path": intent, "content": content}}}))
	}
	steps := append([]shpStep{
		propose("propose-1", shpScopedIntentMD),
		w.calls(acceptBatchRead("custody-1", custody), acceptBatchRead("readme-1", "README.md")),
		w.calls(acceptBatchRead("custody-2", custody), acceptBatchRead("readme-2", "README.md"), acceptBatchRead("readme-line", "README.md", 3, 3)),
		propose("propose-2", changed),
		w.calls(acceptBatchRead("custody-3", custody), acceptBatchRead("readme-3", "README.md")),
	}, acceptBatchEnd(w)...)
	f := w.newShape(t, steps...)
	f.shape("Do not repeat unchanged reads.")
	reqs := f.Prov.requests()
	_, first := acceptBatchAfter(t, reqs, 2, "custody-1", "readme-1")
	if !strings.HasPrefix(first["custody-1"], "1\t# Add an is-even check\n") {
		t.Fatalf("first read of the proposal custody file %q, want its numbered content", first["custody-1"])
	}
	if first["readme-1"] != "1\t# Calc\n2\t\n3\tA tiny calculator." {
		t.Fatalf("first read of README.md %q, want its numbered content", first["readme-1"])
	}
	_, second := acceptBatchAfter(t, reqs, 3, "custody-2", "readme-2", "readme-line")
	if second["readme-2"] != readmeNote {
		t.Fatalf("repeated read of unchanged README.md answered %q, want %q", second["readme-2"], readmeNote)
	}
	if want := acceptBatchNote(first["custody-1"]); second["custody-2"] != want {
		t.Fatalf("repeated read of the unchanged custody file answered %q, want %q", second["custody-2"], want)
	}
	if second["readme-line"] != "3\tA tiny calculator." {
		t.Fatalf("first read of README.md lines 3-3 answered %q, want %q", second["readme-line"], "3\tA tiny calculator.")
	}
	_, third := acceptBatchAfter(t, reqs, 5, "custody-3", "readme-3")
	if !strings.Contains(third["custody-3"], "returns True for every even integer.") || !strings.HasPrefix(third["custody-3"], "1\t# Add an is-even check\n") {
		t.Fatalf("read of the custody file after a proposal changed it answered %q, want its new numbered content", third["custody-3"])
	}
	if third["readme-3"] != readmeNote {
		t.Fatalf("third read of unchanged README.md answered %q, want %q", third["readme-3"], readmeNote)
	}
}

func TestAcceptBatchPolicy(t *testing.T) {
	sentence := os.Getenv("ACCEPT_BATCH_SENTENCE")
	if sentence == "" {
		t.Fatal("ACCEPT_BATCH_SENTENCE is not set")
	}
	b, err := os.ReadFile("prompts/shaper.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), sentence) {
		t.Fatalf("shape/prompts/shaper.md does not contain %q", sentence)
	}
	w := responsesWire
	f := w.newShape(t, acceptBatchEnd(w)...)
	f.shape("Read the policy.")
	reqs := f.Prov.requests()
	if len(reqs) == 0 {
		t.Fatal("Shape sent no shaper request")
	}
	if s, _ := reqs[0]["instructions"].(string); !strings.Contains(s, sentence) {
		t.Fatalf("the shaper's instructions do not contain %q", sentence)
	}
	found := map[string]string{}
	list, _ := reqs[0]["tools"].([]any)
	for _, it := range list {
		m, _ := it.(map[string]any)
		name, _ := m["name"].(string)
		d, _ := m["description"].(string)
		found[name] = d
	}
	for _, name := range []string{"read_file", "search_text", "list_files", "probe"} {
		d, ok := found[name]
		if !ok {
			t.Fatalf("the shaper request has no %s tool", name)
		}
		if !strings.Contains(d, sentence) {
			t.Fatalf("the %s tool description %q does not contain %q", name, d, sentence)
		}
	}
}
