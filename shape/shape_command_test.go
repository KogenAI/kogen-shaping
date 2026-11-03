package shape

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandaloneShapeAnswersPendingQuestions(t *testing.T) {
	f := newShpFx(t,
		shpSubmit("q1", map[string]any{"questions": []any{shpQuestion("What should is_even do with a non-integer?", "raise TypeError", "return False")}}), shpUI(1),
		shpSubmit("c1", shpFilesWithDecisions("- Non-integers raise TypeError. (source: answer 1)")), shpAck("c2"))
	f.setReport("PASSED", 0)
	binary := filepath.Join(t.TempDir(), "kogen-shape")
	build := exec.Command("go", "build", "-o", binary, "../cmd/kogen-shape")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	run := func(input string, want int) (map[string]any, string) {
		t.Helper()
		cmd := exec.Command(binary, "--project", f.Root, "--slug", shpSlug, "--request", "-")
		cmd.Env = f.Env
		cmd.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		var doc map[string]any
		if e := json.Unmarshal(stdout.Bytes(), &doc); e != nil {
			t.Fatalf("stdout %q: %v; stderr %s", stdout.String(), e, &stderr)
		}
		if code != want || doc["exit_code"] != float64(want) {
			t.Fatalf("exit %d, doc %v, stderr %s", code, doc, &stderr)
		}
		return doc, stderr.String()
	}
	pending, stderr := run("Add calc.is_even(n).", 5)
	if pending["intent"] != nil || pending["error"] == nil || !strings.Contains(stderr, "Questions for is-even:") {
		t.Fatalf("pending %v stderr %s", pending, stderr)
	}
	done, stderr := run("1: a", 0)
	written, err := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md"))
	if err != nil {
		t.Fatalf("intent: %v; %v stderr %s", err, done, stderr)
	}
	intent, ok := done["intent"].(map[string]any)
	if !ok || intent["content"] != string(written) || !strings.Contains(string(written), "(source: answer 1)") || done["error"] != nil {
		t.Fatalf("result %v", done)
	}
}
