package main

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestCommandInvalidUTF8(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "kogen-shape")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	for _, source := range []string{"file", "stdin"} {
		t.Run(source, func(t *testing.T) {
			request := "-"
			wantError := "request from stdin must be UTF-8 text"
			if source == "file" {
				request = filepath.Join(t.TempDir(), "invalid request.txt")
				if err := os.WriteFile(request, []byte{0xff}, 0600); err != nil {
					t.Fatal(err)
				}
				wantError = "request file " + request + " must be UTF-8 text"
			}
			cmd := exec.Command(binary, "--project", t.TempDir(), "--slug", "abc", "--request", request)
			if source == "stdin" {
				cmd.Stdin = bytes.NewReader([]byte{0xff})
			}
			var out, stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err := cmd.Run()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 2 {
				t.Fatalf("expected exit 2, got %v; stderr %s", err, &stderr)
			}
			var doc result
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatalf("stdout %q: %v", out.String(), err)
			}
			if doc.ExitCode != 2 || doc.Intent != nil || doc.Error == nil || *doc.Error != wantError {
				t.Fatalf("unexpected refusal: %s", &out)
			}
			if !bytes.HasSuffix(out.Bytes(), []byte("\n")) || bytes.Count(out.Bytes(), []byte("\n")) != 1 {
				t.Fatalf("expected one JSON line, got %q", out.String())
			}
		})
	}
}
