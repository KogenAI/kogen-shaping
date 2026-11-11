package shape

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const batchTestDir = ".kogen/local/batch-test"

// batchGate answers FIFO reads: each writer waits, once its reader is open, until want readers
// are open at once or two seconds pass; peak is how many were open when the first writer went on.
type batchGate struct {
	mu      sync.Mutex
	want    int
	arrived int
	peak    int
	release chan struct{}
	once    sync.Once
	log     string
}

func newBatchGate(want int, log string) *batchGate {
	return &batchGate{want: want, release: make(chan struct{}), log: log}
}

func (g *batchGate) note(line string) {
	if g.log == "" {
		return
	}
	if f, err := os.OpenFile(g.log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		f.WriteString(line + "\n")
		f.Close()
	}
}

func (g *batchGate) serve(t *testing.T, root, rel, name, content string) {
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Fatal(err)
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
		g.note(name + "-open")
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
		g.note(name + "-done")
		f.WriteString(content)
	}()
}

func (g *batchGate) peakOpen() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.peak
}

func batchOutputs(req map[string]any) (ids []string, texts map[string]string) {
	texts = map[string]string{}
	input, _ := req["input"].([]any)
	for _, it := range input {
		m, _ := it.(map[string]any)
		if m["type"] == "function_call_output" {
			id := fmt.Sprint(m["call_id"])
			ids = append(ids, id)
			texts[id], _ = m["output"].(string)
		}
	}
	return ids, texts
}

func batchRead(id, path string, lines ...int) map[string]any {
	args := map[string]any{"path": path}
	if len(lines) == 2 {
		args["start_line"], args["end_line"] = lines[0], lines[1]
	}
	return shpCall(id, "read_file", args)
}

func batchProbe(id, name, log string) map[string]any {
	return shpCall(id, "probe", map[string]any{"command": fmt.Sprintf("printf '%[1]s-start\\n' >> '%[2]s'; sleep 0.2; printf '%[1]s-end\\n' >> '%[2]s'", name, log)})
}

func batchEnd(w *forceWire) []shpStep {
	return []shpStep{w.prose("one"), w.prose("two"), w.prose("three")}
}

func TestReadOnlyCallsOfOneResponseRunConcurrently(t *testing.T) {
	w := responsesWire
	steps := append([]shpStep{w.calls(
		batchRead("read-a", batchTestDir+"/a.txt"),
		shpCall("search", "search_text", map[string]any{"pattern": "tiny", "path": "README.md"}),
		batchRead("read-b", batchTestDir+"/b.txt"),
		batchRead("read-c", batchTestDir+"/c.txt"),
	)}, batchEnd(w)...)
	f := w.newShape(t, steps...)
	gate := newBatchGate(3, "")
	var once sync.Once
	f.Prov.route = func(map[string]any) *shpStep {
		once.Do(func() {
			gate.serve(t, f.Root, batchTestDir+"/a.txt", "a", "alpha\n")
			gate.serve(t, f.Root, batchTestDir+"/b.txt", "b", "beta\n")
			gate.serve(t, f.Root, batchTestDir+"/c.txt", "c", "gamma\n")
		})
		return nil
	}
	f.shape("Batch reads.")
	reqs := f.Prov.requests()
	if len(reqs) < 2 {
		t.Fatalf("%d requests", len(reqs))
	}
	ids, texts := batchOutputs(reqs[1])
	if got := strings.Join(ids, ","); got != "read-a,search,read-b,read-c" {
		t.Fatalf("results in order %q, want the call order", got)
	}
	for id, want := range map[string]string{"read-a": "1\talpha", "read-b": "1\tbeta", "read-c": "1\tgamma"} {
		if texts[id] != want {
			t.Errorf("%s result %q, want %q", id, texts[id], want)
		}
	}
	if !strings.Contains(texts["search"], "A tiny calculator.") {
		t.Errorf("search result %q", texts["search"])
	}
	if p := gate.peakOpen(); p != 3 {
		t.Fatalf("%d of 3 reads were in flight together", p)
	}
}

func TestProbesRunOneAtATimeInCallOrderAroundReadGroups(t *testing.T) {
	w := responsesWire
	log := filepath.Join(t.TempDir(), "order.log")
	steps := append([]shpStep{w.calls(
		batchProbe("probe-1", "p1", log),
		batchRead("read-x", batchTestDir+"/x.txt"),
		batchRead("read-y", batchTestDir+"/y.txt"),
		batchProbe("probe-2", "p2", log),
	)}, batchEnd(w)...)
	f := w.newShape(t, steps...)
	gate := newBatchGate(2, log)
	var once sync.Once
	f.Prov.route = func(map[string]any) *shpStep {
		once.Do(func() {
			gate.serve(t, f.Root, batchTestDir+"/x.txt", "x", "ex\n")
			gate.serve(t, f.Root, batchTestDir+"/y.txt", "y", "why\n")
		})
		return nil
	}
	f.shape("Keep probes in order.")
	reqs := f.Prov.requests()
	if len(reqs) < 2 {
		t.Fatalf("%d requests", len(reqs))
	}
	ids, _ := batchOutputs(reqs[1])
	if got := strings.Join(ids, ","); got != "probe-1,read-x,read-y,probe-2" {
		t.Fatalf("results in order %q, want the call order", got)
	}
	b, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	at := map[string]int{}
	for i, l := range lines {
		at[l] = i
	}
	seq := strings.Join(lines, ",")
	if len(at) != 8 {
		t.Fatalf("events %s, want 8 distinct", seq)
	}
	if !(at["p1-end"] < at["x-open"] && at["p1-end"] < at["y-open"]) {
		t.Errorf("a read started before the earlier probe ended: %s", seq)
	}
	if !(at["x-done"] < at["p2-start"] && at["y-done"] < at["p2-start"]) {
		t.Errorf("the later probe started before the reads finished: %s", seq)
	}
	if gate.peakOpen() != 2 {
		t.Errorf("the two reads did not overlap: %s", seq)
	}
}

func TestUnchangedRepeatReadAnswersNoteAndChangedFileReturnsContent(t *testing.T) {
	w := responsesWire
	intent := ".kogen/intents/is-even/intent.md"
	custody := ".kogen/local/shaping/is-even/proposed/" + intent
	changed := strings.Replace(shpScopedIntentMD, "returns True for even integers.", "returns True for every even integer.", 1)
	propose := func(id, content string) shpStep {
		return w.calls(shpCall(id, "propose_files", map[string]any{"files": []any{map[string]any{"path": intent, "content": content}}}))
	}
	steps := append([]shpStep{
		propose("propose-1", shpScopedIntentMD),
		w.calls(batchRead("custody-1", custody), batchRead("readme-1", "README.md")),
		w.calls(batchRead("custody-2", custody), batchRead("readme-2", "README.md"), batchRead("readme-line", "README.md", 3, 3)),
		propose("propose-2", changed),
		w.calls(batchRead("custody-3", custody), batchRead("readme-3", "README.md")),
	}, batchEnd(w)...)
	f := w.newShape(t, steps...)
	f.shape("Do not repeat unchanged reads.")
	reqs := f.Prov.requests()
	if len(reqs) < 6 {
		t.Fatalf("%d requests", len(reqs))
	}
	_, first := batchOutputs(reqs[2])
	_, second := batchOutputs(reqs[3])
	_, third := batchOutputs(reqs[5])
	var m readMemory
	m.answer(map[string]any{"path": "README.md"}, first["readme-1"])
	note := m.answer(map[string]any{"path": "README.md"}, first["readme-1"])
	if !strings.HasPrefix(note, "already read in this window, unchanged (") {
		t.Fatalf("note %q", note)
	}
	if second["readme-2"] != note {
		t.Errorf("repeated README.md read %q, want %q", second["readme-2"], note)
	}
	if !strings.HasPrefix(second["custody-2"], "already read in this window, unchanged (") {
		t.Errorf("repeated custody read %q, want a note", second["custody-2"])
	}
	if second["readme-line"] != "3\tA tiny calculator." {
		t.Errorf("first read of lines 3-3 %q", second["readme-line"])
	}
	if !strings.Contains(third["custody-3"], "returns True for every even integer.") {
		t.Errorf("read after the proposal changed the file %q, want its new content", third["custody-3"])
	}
	if third["readme-3"] != note {
		t.Errorf("third README.md read %q, want %q", third["readme-3"], note)
	}
}

func TestReadMemory(t *testing.T) {
	var m readMemory
	args := map[string]any{"path": "./a.txt"}
	if got := m.answer(args, "1\tx"); got != "1\tx" {
		t.Fatalf("first read %q", got)
	}
	want := "already read in this window, unchanged (3 bytes, sha " // 8 hex digits follow
	if got := m.answer(map[string]any{"path": "a.txt"}, "1\tx"); !strings.HasPrefix(got, want) || len(got) != len(want)+9 {
		t.Fatalf("repeat %q", got)
	}
	if got := m.answer(map[string]any{"path": "a.txt", "start_line": 1.0, "end_line": 1.0}, "1\tx"); got != "1\tx" {
		t.Fatalf("other line range %q", got)
	}
	if got := m.answer(args, "1\ty"); got != "1\ty" {
		t.Fatalf("changed result %q", got)
	}
	if got := m.answer(args, "error: no"); got != "error: no" {
		t.Fatalf("error result %q", got)
	}
	if got := m.answer(args, "error: no"); got != "error: no" {
		t.Fatalf("repeated error result %q", got)
	}
	m.reset()
	if got := m.answer(args, "1\ty"); got != "1\ty" {
		t.Fatalf("first read after a reset %q, want its content", got)
	}
}

func TestShaperIsToldToBatchIndependentCalls(t *testing.T) {
	b, err := os.ReadFile("prompts/shaper.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "\n- "+batchSentence+"\n") {
		t.Error("shaper.md lacks the batching bullet")
	}
	if !strings.Contains(ShaperInstructions(), batchSentence) {
		t.Error("the shaper instructions lack the batching sentence")
	}
	for _, name := range []string{"read_file", "search_text", "list_files", ProbeTool} {
		d, _ := ShaperToolDefinitions(name)[0]["description"].(string)
		if !strings.HasSuffix(d, " "+batchSentence) {
			t.Errorf("%s description does not end with the batching sentence", name)
		}
	}
	for _, name := range []string{"read_file", ProbeTool} {
		if d, _ := RefineToolDefinitions(name)[0]["description"].(string); strings.Contains(d, batchSentence) {
			t.Errorf("the refine %s description carries the batching sentence", name)
		}
	}
	if strings.Contains(shaperPrompt("refine"), batchSentence) {
		t.Error("refine's prompt carries the batching sentence")
	}
}
