package shape

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// The Shaper's and Refine's prompts and Kogen-owned tool definitions moved here from
// kogen-executor's agent package together with their goldens: testdata/prompts_golden.json and
// testdata/defs_golden.json hold exactly the entries executor's agent/testdata goldens held for
// them, copied unchanged. They record the prompt and tool bytes the models have always received,
// so they change only deliberately; "go test ./... -update" rewrites them from the local policy.

// policyPrompts are the shaping prompts the golden records, each rendered as the model receives
// it (with its final newline).
var policyPrompts = map[string]func() string{
	"shaper":                            func() string { return shaperPrompt("shaper") },
	"refine":                            func() string { return shaperPrompt("refine") },
	"feasibility-guidance":              func() string { return shaperPrompt("feasibility-guidance") },
	"readiness-acknowledgment-guidance": func() string { return shaperPrompt("readiness-acknowledgment-guidance") },
	"shaper-context-guidance":           func() string { return shaperPrompt("shaper-context-guidance") },
}

// ownedTools are the shaping-owned tool definitions, in executor's order.
var ownedTools = []string{ProbeTool, ProposeFilesTool, SubmitTool, SubmitRefinementTool}

// ownedDefinition is the local definition of an owned tool.
func ownedDefinition(name string) map[string]any {
	if name == SubmitRefinementTool {
		return RefineToolDefinitions(name)[0]
	}
	return ShaperToolDefinitions(name)[0]
}

// writePolicyGolden writes v as executor's goldens are written: two-space-indented JSON with
// sorted object keys, unescaped HTML characters and exactly one final newline.
func writePolicyGolden(t *testing.T, path string, v any) {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func policyJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestPromptsAreByteEqualToTheElixirFunctions compares the length and SHA-256 of each shaping
// prompt with testdata/prompts_golden.json (moved from executor's agent tests).
func TestPromptsAreByteEqualToTheElixirFunctions(t *testing.T) {
	type entry struct {
		Bytes  int    `json:"bytes"`
		Sha256 string `json:"sha256"`
	}
	path := "testdata/prompts_golden.json"
	if *updateGoldens {
		entries := map[string]entry{}
		for name, text := range policyPrompts {
			sum := sha256.Sum256([]byte(text()))
			entries[name] = entry{Bytes: len(text()), Sha256: hex.EncodeToString(sum[:])}
		}
		writePolicyGolden(t, path, entries)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]entry
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(policyPrompts) {
		t.Fatalf("golden has %d prompts, package has %d", len(want), len(policyPrompts))
	}
	for name, text := range policyPrompts {
		t.Run(name, func(t *testing.T) {
			w, ok := want[name]
			if !ok {
				t.Fatalf("no golden entry for %s", name)
			}
			text := text()
			sum := sha256.Sum256([]byte(text))
			if len(text) != w.Bytes || hex.EncodeToString(sum[:]) != w.Sha256 {
				t.Errorf("%s: %d bytes sha256 %x, want %d bytes sha256 %s", name, len(text), sum, w.Bytes, w.Sha256)
			}
			if strings.Contains(text, "{{") {
				t.Errorf("%s has an unreplaced placeholder", name)
			}
		})
	}
}

// TestDefinitionsMatchTheElixirDefinitions compares every shaping-owned tool definition with
// testdata/defs_golden.json (moved from executor's agent tests), as parsed values.
func TestDefinitionsMatchTheElixirDefinitions(t *testing.T) {
	path := "testdata/defs_golden.json"
	if *updateGoldens {
		defs := map[string]any{}
		for _, name := range ownedTools {
			var def any
			if err := json.Unmarshal(policyJSON(t, ownedDefinition(name)), &def); err != nil {
				t.Fatal(err)
			}
			defs[name] = def
		}
		writePolicyGolden(t, path, defs)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if len(ownedTools) != len(want) {
		t.Fatalf("%d shaping-owned names, golden has %d", len(ownedTools), len(want))
	}
	for _, name := range ownedTools {
		t.Run(name, func(t *testing.T) {
			var got any
			if err := json.Unmarshal(policyJSON(t, ownedDefinition(name)), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want[name]) {
				t.Errorf("definition differs\n got: %s\nwant: %s", policyJSON(t, got), policyJSON(t, want[name]))
			}
		})
	}
}

func TestEveryDefinitionIsPlainJSON(t *testing.T) {
	// Provider requests encode []any and map[string]any only through their own encoder; a
	// typed slice or a struct in a definition would be encoded differently.
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, e := range x {
				walk(path+"."+k, e)
			}
		case []any:
			for _, e := range x {
				walk(path+"[]", e)
			}
		case string, bool, int, int64, float64, nil:
		default:
			t.Errorf("%s is %T", path, v)
		}
	}
	for _, name := range ownedTools {
		walk(name, ownedDefinition(name))
	}
}

// TestShaperAndRefineShareTheGuidancePassages: the Shaper's and Refine's prompts each hold the
// feasibility and readiness-acknowledgment passages once, and submit's description holds the
// acknowledgment. (Executor's version also checked that the builder's, reviewer's and init's
// prompts do not hold the feasibility guidance; those prompts now live in kogen-build,
// kogen-review and kogen-setup, which hold no copy of this passage.)
func TestShaperAndRefineShareTheGuidancePassages(t *testing.T) {
	feasibility, acknowledgment := shaperPrompt("feasibility-guidance"), shaperPrompt("readiness-acknowledgment-guidance")
	for _, name := range []string{"shaper", "refine"} {
		text := shaperPrompt(name)
		if strings.Count(text, feasibility) != 1 || strings.Count(text, acknowledgment) != 1 {
			t.Errorf("%s must hold each guidance passage exactly once", name)
		}
	}
	quoted := policyJSON(t, acknowledgment)
	if !strings.Contains(string(policyJSON(t, ShaperToolDefinitions(SubmitTool)[0])), string(quoted[1:len(quoted)-1])) {
		t.Error("the submit definition lost the acknowledgment guidance")
	}
}
