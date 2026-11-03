package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandInput(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "kogen-shape")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	for _, tc := range []struct {
		slug, input string
		code        int
	}{
		{"ab", "request", 2}, {strings.Repeat("a", 49), "request", 2}, {"Abc", "request", 2}, {"abc_", "request", 2},
		{"abc", "", 2}, {"abc", "  \n", 2}, {"abc", "request", 3}, {strings.Repeat("a", 48), "request", 3},
	} {
		t.Run(tc.slug+"/"+tc.input, func(t *testing.T) {
			cmd := exec.Command(binary, "--project", t.TempDir(), "--slug", tc.slug, "--request", "-")
			cmd.Stdin = strings.NewReader(tc.input)
			var out, stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err := cmd.Run()
			code := 0
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					code = e.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			var doc struct {
				ExitCode int     `json:"exit_code"`
				Intent   any     `json:"intent"`
				Error    *string `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatalf("stdout %q: %v", out.String(), err)
			}
			if code != tc.code || doc.ExitCode != code || doc.Intent != nil || doc.Error == nil || *doc.Error == "" || !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
				t.Fatalf("code %d doc %+v stderr %s", code, doc, &stderr)
			}
		})
	}
}
