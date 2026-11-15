//go:build ignore

package shape

import (
	"encoding/json"
	"os"
	"testing"
)

// This bridge only calls the real transformation; expectations live in the feature.
func TestRefusalCutAcceptanceBridge(t *testing.T) {
	input, err := os.ReadFile(os.Getenv("REFUSAL_CUT_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	var cases [][]string
	if err := json.Unmarshal(input, &cases); err != nil {
		t.Fatal(err)
	}
	results := make([][]string, len(cases))
	for i, lines := range cases {
		results[i] = cutProblems(lines)
	}
	output, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("REFUSAL_CUT_OUTPUT"), output, 0600); err != nil {
		t.Fatal(err)
	}
}
