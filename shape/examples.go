package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
	"github.com/KogenAI/kogen-verification/verify"
)

const exampleTool = "submit_example_check"
const exampleInstructions = `Check every Examples row in the candidate against the rules in the request and Intent. Return one item per row with scenario, expected, derived, verdict and why. Derived is the value given by the rules, or an empty string when not given. Verdict is follows, contradicts or unsupported. Compare values by meaning, not formatting. Call submit_example_check.`

func exampleSchema() map[string]any {
	props := map[string]any{}
	for _, k := range []string{"scenario", "expected", "derived", "verdict", "why"} {
		props[k] = str()
	}
	return map[string]any{"type": "object", "properties": map[string]any{"items": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": props, "required": []any{"scenario", "expected", "derived", "verdict", "why"}, "additionalProperties": false}}}, "required": []any{"items"}, "additionalProperties": false}
}

// exampleValue parses JSON and the equivalent unquoted member-list notation.
func exampleValue(s string) (any, bool) {
	s = strings.TrimSpace(s)
	if json.Valid([]byte(s)) {
		var v any
		decoder := json.NewDecoder(strings.NewReader(s))
		decoder.UseNumber()
		if decoder.Decode(&v) == nil {
			return v, true
		}
	}
	if !strings.HasPrefix(s, "{") && strings.Contains(s, ":") {
		s = "{" + s + "}"
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if c == '"' {
			start := i
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			if i > len(s) {
				return nil, false
			}
			b.WriteString(s[start:i])
			continue
		}
		if strings.ContainsRune("{}[]:,", rune(c)) || unicode.IsSpace(rune(c)) {
			b.WriteByte(c)
			i++
			continue
		}
		start := i
		for i < len(s) && !strings.ContainsRune("{}[]:, \t\r\n", rune(s[i])) {
			i++
		}
		token := s[start:i]
		j := i
		for j < len(s) && unicode.IsSpace(rune(s[j])) {
			j++
		}
		if (j < len(s) && s[j] == ':') || !json.Valid([]byte(token)) {
			q, _ := json.Marshal(token)
			b.Write(q)
		} else {
			b.WriteString(token)
		}
	}
	var v any
	if !json.Valid([]byte(b.String())) {
		return nil, false
	}
	decoder := json.NewDecoder(strings.NewReader(b.String()))
	decoder.UseNumber()
	if decoder.Decode(&v) != nil {
		return nil, false
	}
	return v, true
}

func exampleEqual(a, b any) bool {
	switch x := a.(type) {
	case json.Number:
		y, ok := b.(json.Number)
		if !ok {
			return false
		}
		ar, aok := new(big.Rat).SetString(string(x))
		br, bok := new(big.Rat).SetString(string(y))
		return aok && bok && ar.Cmp(br) == 0
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, exists := y[k]
			if !exists || !exampleEqual(v, w) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !exampleEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

func exampleVerdict(expected, derived, verdict string) string {
	if strings.TrimSpace(derived) == "" {
		return "unsupported"
	}
	a, okA := exampleValue(expected)
	b, okB := exampleValue(derived)
	if okA && okB {
		if exampleEqual(a, b) {
			return "follows"
		}
		return "contradicts"
	}
	if strings.TrimSpace(expected) == strings.TrimSpace(derived) {
		return "follows"
	}
	return verdict
}

func (sh *shaping) exampleCheck(ctx context.Context) ([]string, error) {
	files, err := sh.candidatePackage()
	if err != nil {
		return nil, err
	}
	hasExamples := false
	for _, f := range files {
		if strings.HasSuffix(f.path, ".feature") {
			for _, line := range strings.Split(f.content, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "Examples:") {
					hasExamples = true
				}
			}
		}
	}
	if !hasExamples {
		return nil, nil
	}
	input := literalInput(sh.slug, sh.request, sh.rounds, files)
	var args map[string]any
	if sh.ext != nil {
		x, e := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"), fmt.Sprintf("%s/executor-example-%d", sh.ev.Dir, sh.submissions), sh.slug, sh.role, exampleInstructions+"\n\n"+input, "", mustJSON(exampleSchema()), sh.ev, nil)
		if e != nil {
			return nil, e
		}
		defer x.close()
		res, e := x.round(ctx, "")
		if e != nil {
			return nil, e
		}
		if res.Ending != executor.EndFinish {
			return nil, sys.Fail("shape", "example_check_invalid", "example check ended without a result", 1)
		}
		args, e = submissionArgs(res.Structured)
		if e != nil {
			return nil, e
		}
	} else {
		revision, e := CurrentRevision(sh.repo)
		if e != nil {
			return nil, e
		}
		req := conversationRequest(provider.Request{Role: ProviderRole(sh.role), Instructions: exampleInstructions, Input: []any{map[string]any{"role": "user", "content": input}}, Tools: []any{map[string]any{"type": "function", "name": exampleTool, "strict": true, "description": "Submit the example check.", "parameters": exampleSchema()}}, Scope: sh.ev.Dir, Context: map[string]any{"phase": "shape", "role": "shape", "slug": sh.slug}}, startConversation(sh.ev, "example-check"))
		call := sh.ev.BeginCall("shape", "", revision.Map(), "example check", sh.role).Sized(sh.client, req)
		resp, e := sh.client.Respond(ctx, sh.s, req, call)
		call.End(resp, e)
		if e != nil {
			if p, ok := sys.AsError(e); ok && p.Class == "provider" {
				return nil, sh.providerFailed(p)
			}
			return nil, e
		}
		calls := tools.Calls(resp.Items)
		if len(calls) != 1 || calls[0].Name != exampleTool {
			return nil, sys.Fail("shape", "example_check_invalid", "expected one example check result", 1)
		}
		args = calls[0].Args()
	}
	list, ok := args["items"].([]any)
	if !ok {
		return nil, sys.Fail("shape", "example_check_invalid", "invalid example check items", 1)
	}
	var problems []string
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, sys.Fail("shape", "example_check_invalid", "invalid example check item", 1)
		}
		for _, k := range []string{"scenario", "expected", "derived", "verdict", "why"} {
			if _, ok := m[k].(string); !ok {
				return nil, sys.Fail("shape", "example_check_invalid", "invalid example check item", 1)
			}
		}
		if !slices.Contains([]string{"follows", "contradicts", "unsupported"}, m["verdict"].(string)) {
			return nil, sys.Fail("shape", "example_check_invalid", "invalid example verdict", 1)
		}
		m["verdict"] = exampleVerdict(m["expected"].(string), m["derived"].(string), m["verdict"].(string))
		if m["verdict"] == "contradicts" {
			problems = append(problems, fmt.Sprintf("%s: expected %s, rules give %s", m["scenario"], m["expected"], m["derived"]))
		}
	}
	home := sh.s.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	red := verify.NewRedactor(sh.s.Env, home)
	events := make([]any, 0, len(list))
	for _, raw := range list {
		m := raw.(map[string]any)
		item := map[string]any{}
		for _, key := range []string{"scenario", "expected", "derived", "verdict", "why"} {
			item[key] = red.Redact(m[key].(string))
		}
		events = append(events, item)
	}
	for i := range problems {
		problems[i] = red.Redact(problems[i])
	}
	sh.ev.Record(map[string]any{"event": "example_check", "items": events})
	return problems, nil
}
