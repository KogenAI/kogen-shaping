// kogen-shape is a developer-only JSON adapter for the Shape workflow.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-shaping/shape"
	"github.com/KogenAI/kogen-verification/verify"
)

type writtenIntent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type result struct {
	ExitCode int            `json:"exit_code"`
	Intent   *writtenIntent `json:"intent"`
	Error    *string        `json:"error"`
}

func failure(code int, message string) result { return result{ExitCode: code, Error: &message} }

func run(args []string) result {
	flags := flag.NewFlagSet("kogen-shape", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	project := flags.String("project", "", "checkout")
	slug := flags.String("slug", "", "slug")
	request := flags.String("request", "", "request file or -")
	if err := flags.Parse(args); err != nil {
		return failure(2, err.Error())
	}
	if flags.NArg() != 0 || *project == "" || *request == "" || !regexp.MustCompile(`^[a-z0-9-]{3,48}$`).MatchString(*slug) {
		return failure(2, "usage: kogen-shape --project <checkout> --slug <3-48 lowercase letters, digits or dashes> --request <file|->")
	}
	var data []byte
	var err error
	requestPath := ""
	if *request == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		requestPath, err = filepath.Abs(*request)
		if err == nil {
			data, err = os.ReadFile(requestPath)
		}
	}
	if err != nil {
		return failure(2, fmt.Sprintf("read request: %v", err))
	}
	if !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return failure(2, "request must be nonempty UTF-8 text")
	}
	s := sys.New()
	s.Stdout = os.Stderr
	stop := s.WatchSignals()
	defer stop()
	code, err := shape.Shape(context.Background(), s, shape.ShapeInput{Project: *project, Slug: *slug, Request: string(data), RequestPath: requestPath, Tools: verify.Tools{Node: s.Getenv("KOGEN_NODE"), Quint: s.Getenv("KOGEN_QUINT"), Cucumber: s.Getenv("KOGEN_CUCUMBER")}})
	if err != nil {
		return failure(code, err.Error())
	}
	if code != 0 {
		description := fmt.Sprintf("Shape did not publish an Intent (exit %d)", code)
		if code == 5 {
			description = "Shape has pending questions or refused an existing Intent (exit 5)"
		}
		return failure(code, description)
	}
	dir := *project
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(s.Cwd, dir)
	}
	repo, err := git.Open(dir, s.Env)
	if err != nil {
		return failure(1, fmt.Sprintf("locate published Intent: %v", err))
	}
	path := filepath.Join(repo.Dir, ".kogen/intents", *slug, "intent.md")
	data, err = os.ReadFile(path)
	if err != nil {
		return failure(1, fmt.Sprintf("read published Intent: %v", err))
	}
	return result{Intent: &writtenIntent{Path: path, Content: string(data)}}
}
func main() { r := run(os.Args[1:]); _ = json.NewEncoder(os.Stdout).Encode(r); os.Exit(r.ExitCode) }
