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

type policySnapshot struct {
	Prompt string                      `json:"prompt"`
	Tools  map[string][]map[string]any `json:"tools"`
}

func policyBytes() []byte {
	names := map[string][]string{}
	main := (&shaping{}).mainTools()
	names["default"] = append(append([]string{}, main...), ProposeFilesTool, SubmitTool)
	snap := policySnapshot{Prompt: ShaperInstructions(), Tools: map[string][]map[string]any{}}
	for name, n := range names {
		snap.Tools[name] = ShaperToolDefinitions(n...)
	}
	b, _ := json.MarshalIndent(snap, "", "  ")
	return append(b, '\n')
}

// The golden was derived in K-SHAPE-PROMPT-MOVE from executor v0.0.0-20261102193603-1aea0728c24b
// (its Shaper prompt and definitions, with Shape's existing suffixes); executor no longer holds
// that policy, which shaping owns now, so -update rewrites it from the local policy. Use it only
// after an intended policy change, and review the diff.
func TestShaperPolicyGolden(t *testing.T) {
	path := "testdata/shaper_policy_golden.json"
	if *updateGoldens {
		if err := os.WriteFile(path, policyBytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(policyBytes(), want) {
		t.Fatal("local Shaper policy differs from executor-derived golden")
	}
}

func TestShaperPolicyRequest(t *testing.T) {
	f := newShpFx(t, shpSubmit("q", map[string]any{"questions": []any{shpQuestion("Q?")}}), shpUI(1))
	if exit, err := f.shape("A small change"); err != nil || exit != 5 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	req := shpShaperRequests(f.Prov.requests())[0]
	if req["instructions"] != ShaperInstructions() {
		t.Fatal("request did not use local instructions")
	}
	got, _ := json.Marshal(req["tools"])
	names := append((&shaping{}).mainTools(), ProposeFilesTool, SubmitTool)
	want, _ := json.Marshal(ShaperToolDefinitions(names...))
	if !bytes.Equal(got, want) {
		t.Fatalf("request tools differ from local policy\ngot %s\nwant %s", got, want)
	}
	if os.Getenv("SHAPER_POLICY_MUTATION_CHILD") == "1" {
		if !strings.Contains(req["instructions"].(string), "LOCAL POLICY MUTATION") {
			t.Fatal("mutation did not reach provider")
		}
		baseline, err := os.ReadFile("testdata/shaper_policy_golden.json")
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(policyBytes(), baseline) {
			t.Fatal("mutation did not cause golden drift")
		}
	}
}

// Compile with a Go overlay, editing only the local embedded policy without touching the pin.
func TestShaperPolicyMutation(t *testing.T) {
	source, err := filepath.Abs("prompts/shaper.md")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	modified := filepath.Join(dir, "shaper.md")
	if err := os.WriteFile(modified, append(b, []byte("\nLOCAL POLICY MUTATION\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	overlay, _ := json.Marshal(map[string]any{"Replace": map[string]string{source: modified}})
	overlayPath := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-overlay", overlayPath, "-run", "^TestShaperPolicyRequest$", "-count=1", ".")
	cmd.Env = append(os.Environ(), "SHAPER_POLICY_MUTATION_CHILD=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("mutated policy request: %v\n%s", err, out)
	}
}

func TestRenderedDecisionAndControlsGuidance(t *testing.T) {
	text := ShaperInstructions()
	for _, want := range []string{"D<n>", "(decision: D<n>, round <r>)", "(constraint: <c>)", "(evidence: <items joined by \"; \">)", "(assumption: <uncertainty>)", "(revises: <id> because <why>)", "tracked at HEAD", "never cite a file added", ".kogen/intents/<slug>/controls.json", "keyword included", "65536", "300-second", "not Feasibility's observer controls"} {
		if !strings.Contains(text, want) {
			t.Errorf("guidance missing %q", want)
		}
	}
	if !strings.Contains(externalNote, "A command this external session ran is no evidence of a submitted decision; cite the files read and put what the command showed in the decision's uncertainty.") {
		t.Error("external note missing evidence rule")
	}
}
