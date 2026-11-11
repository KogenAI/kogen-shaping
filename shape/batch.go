package shape

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/KogenAI/kogen-loop/tools"
)

// batchSentence is the sentence that tells the shaper to batch independent calls and not to
// re-read a file it has already read; shaper.md carries it as its own bullet and the shaper's
// read_file, search_text, list_files and probe descriptions end with it.
const batchSentence = "Request every independent read_file, search_text, list_files and probe call you need in one response, and do not re-read a file you have already read unless it changed."

// readOnlyCall is whether a call of the main conversation only reads the checkout and may run
// together with the consecutive read-only calls around it.
func readOnlyCall(name string) bool {
	switch name {
	case tools.ReadFileTool, tools.SearchTextTool, tools.ListFilesTool:
		return true
	}
	return false
}

// readRun returns how many calls from the start of calls are consecutive read-only calls.
func readRun(calls []tools.Call) int {
	n := 0
	for n < len(calls) && readOnlyCall(calls[n].Name) {
		n++
	}
	return n
}

// readMemory remembers the result text of the read_file calls of one working window by cleaned
// path and line range; the zero value is empty and a reset forgets everything.
type readMemory struct {
	seen map[string]string
}

func (m *readMemory) reset() { m.seen = nil }

// answer is the text the model receives for a read_file call that returned text: a short note
// when an earlier read of the same path and lines in this window returned exactly that text,
// else text itself, which then becomes the remembered result. Error results are never replaced
// or remembered.
func (m *readMemory) answer(args map[string]any, text string) string {
	if strings.HasPrefix(text, "error:") {
		return text
	}
	path, _ := args["path"].(string)
	key := fmt.Sprintf("%s\x00%v\x00%v", cleanCitedPath(path), args["start_line"], args["end_line"])
	if prior, ok := m.seen[key]; ok && prior == text {
		sum := sha256.Sum256([]byte(text))
		return fmt.Sprintf("already read in this window, unchanged (%d bytes, sha %s)", len(text), hex.EncodeToString(sum[:])[:8])
	}
	if m.seen == nil {
		m.seen = map[string]string{}
	}
	m.seen[key] = text
	return text
}

// runReadGroup runs consecutive read-only calls concurrently and returns their answers in call
// order. Shared shaping state is updated only after every call returned, in call order.
func (sh *shaping) runReadGroup(ctx context.Context, group []tools.Call) ([]tools.Output, error) {
	outs := make([]tools.Output, len(group))
	errs := make([]error, len(group))
	var wg sync.WaitGroup
	for i, c := range group {
		wg.Add(1)
		go func() {
			defer wg.Done()
			outs[i], errs[i] = sh.agentTools(ctx, sh.mainTools()).Run(c.Name, c.Args())
		}()
	}
	wg.Wait()
	answers := make([]tools.Output, 0, len(group))
	for i, c := range group {
		if errs[i] != nil {
			return nil, errs[i]
		}
		args := c.Args()
		sh.noteReach(c.Name, args)
		sh.noteTool(c.Name, outs[i])
		answer := outs[i]
		if c.Name == tools.ReadFileTool {
			if sh.readTexts == nil {
				sh.readTexts = map[string]string{}
			}
			path, _ := args["path"].(string)
			sh.readTexts[cleanCitedPath(path)] = outs[i].Text
			answer.Text = sh.window.reads.answer(args, outs[i].Text)
		}
		answers = append(answers, answer)
	}
	return answers, nil
}
