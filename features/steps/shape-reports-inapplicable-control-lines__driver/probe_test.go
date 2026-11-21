//go:build ignore

package shape

import (
	"encoding/json"
	"os"
	"testing"
)

// TestInapplicableControlLinesAcceptance runs the real controls check on a fixture whose patch
// is described by the JSON file named by ICL_IN, and writes the refusal lines to ICL_OUT.
func TestInapplicableControlLinesAcceptance(t *testing.T) {
	var in struct {
		Mode    string  `json:"mode"`
		Target  string  `json:"target"`
		Content *string `json:"content"`
		Patch   string  `json:"patch"`
	}
	raw, err := os.ReadFile(os.Getenv("ICL_IN"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		t.Fatal(err)
	}
	patch := func(p string) map[string]any { return map[string]any{"patch": p} }
	correct := map[string]any{"big_a.js": patch(controlsOffsetPatch("big_a.js", 1, 0)), "big_b.js": patch(controlsOffsetPatch("big_b.js", 1, 0))}
	addWrong := map[string]any{"big_a.js": patch(controlsOffsetPatch("big_a.js", 0, 7))}
	subWrong := map[string]any{"big_b.js": patch(controlsOffsetPatch("big_b.js", 0, 7))}
	if in.Mode == "wrong" {
		addWrong = map[string]any{in.Target: patch(in.Patch)}
	} else {
		correct[in.Target] = patch(in.Patch)
	}
	c, _ := controlsLargeCandidate(t, correct, addWrong, subWrong)
	if in.Content != nil {
		c.base[in.Target] = *in.Content
	}
	lines := controlsLargeLines(t, c)
	out, err := json.Marshal(map[string]any{"lines": append([]string{}, lines...)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("ICL_OUT"), out, 0o644); err != nil {
		t.Fatal(err)
	}
}
