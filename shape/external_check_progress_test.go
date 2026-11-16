package shape

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
)

type checkProgressWriter struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *checkProgressWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *checkProgressWriter) text() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func TestExternalCheckProgress(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.useExternalShape()
	w := &checkProgressWriter{}
	callback := func(kind, wording string) func(int, string) (executor.RoundResult, error) {
		calls := 0
		return func(n int, root string) (executor.RoundResult, error) {
			calls++
			output := w.text()
			start := "kogen: checking the " + wording
			finish := "kogen: " + kind + " check finished"
			// At executor entry, this round has started but has not finished.
			if strings.Count(output, start) != calls || strings.Count(output, finish) != calls-1 {
				t.Errorf("%s round %d entry progress:\n%s", kind, calls, output)
			}
			payload := map[string]any{"items": []any{}}
			if kind == "literal" {
				status := "required"
				if calls > 1 {
					status = "informal"
				}
				payload["items"] = []any{map[string]any{"literal": "ZeroDivisionError", "source": "request", "source_quote": "Raise exactly ZeroDivisionError for 0.", "status": status, "package_value": "", "why": "because"}}
			}
			res, err := result(payload)(n, root)
			if strings.Count(w.text(), finish) != calls-1 {
				t.Errorf("%s round %d finished before executor returned", kind, calls)
			}
			return res, err
		}
	}
	ack := map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}
	x := &shapeExec{t: t,
		rounds:   []func(int, string) (executor.RoundResult, error){result(shpFiles()), result(shpFiles()), result(ack)},
		decision: []func(int, string) (executor.RoundResult, error){callback("decision", "stated decisions")},
		literal:  []func(int, string) (executor.RoundResult, error){callback("literal", "required literals")},
	}
	f.newSession()
	f.S.Stderr = w
	exit, err := Shape(context.Background(), f.S, ShapeInput{Project: f.Root, Tools: fixtureTools(f.S), Slug: shpSlug,
		Request: "Add calc.is_even(n). Raise exactly ZeroDivisionError for 0.", Client: f.Client, External: x.external()})
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, w.text())
	}
	// Both submissions give the decision check the same input, so its answer is reused.
	if x.decisionCalls != 1 || x.literalCalls != 2 {
		t.Fatalf("decision rounds %d, literal rounds %d", x.decisionCalls, x.literalCalls)
	}
	for kind, wording := range map[string]string{"decision": "stated decisions", "literal": "required literals"} {
		pattern := `(?m)^kogen: (?:checking the ` + wording + `|` + kind + ` check finished) \([0-9]+ s\)$`
		lines := regexp.MustCompile(pattern).FindAllString(w.text(), -1)
		pairs := 2
		if kind == "decision" {
			pairs = 1
		}
		if len(lines) != 2*pairs {
			t.Fatalf("%s: expected %d start/finish pairs, got %v\n%s", kind, pairs, lines, w.text())
		}
		for i, line := range lines {
			if strings.Contains(line, "finished") != (i%2 == 1) {
				t.Errorf("%s progress order: %v", kind, lines)
			}
		}
	}
}
