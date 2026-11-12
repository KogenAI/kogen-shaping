package shape

import (
	"fmt"
	"slices"
	"strings"

	"github.com/KogenAI/kogen-verification/verify"
)

// The step-limit judgement of the step self-check: approval stops every step at
// verify.StepTimeout, but readiness runs on the unchanged tree where a step that runs a whole
// test suite fails fast, so Shape never sees its runtime. The judgement reads only the source of
// a step definition and never runs it: a command written in the body, as one shell string or as a
// program with an argument list, that obviously runs a whole test suite is refused. Helpers that
// a body calls are not read.

// stepLimitItem is the self-check item for a step definition whose body runs a whole test suite;
// the bool is false when it does not.
func stepLimitItem(d verify.StepDef) (verify.StepItem, bool) {
	runner := wholeSuiteRunner(d.Body)
	if runner == "" {
		return verify.StepItem{}, false
	}
	return verify.StepItem{
		Check: "step_limit", Target: d.Pattern, File: d.File,
		Detail: fmt.Sprintf("runs a whole test suite with %s and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the %d s step limit (for example %s)",
			runner, int(verify.StepTimeout.Seconds()), narrowingExample[runner]),
	}, true
}

// narrowingExample names, for each runner, a command that runs one test, so the shaper sees the
// exact edit instead of only the rule.
var narrowingExample = map[string]string{
	"go test":   "go test -run '^TestName$' ./package",
	"mix test":  "mix test test/one_test.exs, or mix test --only tag",
	"npm test":  "npm test -- <one spec file or name filter>",
	"pnpm test": "pnpm test -- <one spec file or name filter>",
	"yarn test": "yarn test <one spec file or name filter>",
}

// srcToken is one token of step definition source: a string literal (its value), a punctuation
// character of an argument list, or anything else.
type srcToken struct {
	str   bool
	value string
}

// wholeSuiteRunner is the runner of the first command in the body that runs a whole test suite
// ("go test", "mix test", "npm test", "pnpm test" or "yarn test"), else "".
func wholeSuiteRunner(body string) string {
	toks := scanSource(body)
	for i := 0; i < len(toks); i++ {
		if !toks[i].str {
			continue
		}
		if args, end, ok := argvAt(toks, i); ok {
			if r := wholeSuite(args); r != "" {
				return r
			}
			i = end
			continue
		}
		for _, words := range shellCommands(toks[i].value) {
			if r := wholeSuite(words); r != "" {
				return r
			}
		}
	}
	return ""
}

// argvAt reads a program name literal, a comma and an array of string literals only starting at
// toks[i]. An array with anything else in it (a variable, a spread) may narrow the run and is not
// judged. It returns the program with its arguments and the index of the closing bracket.
func argvAt(toks []srcToken, i int) ([]string, int, bool) {
	if !slices.Contains([]string{"go", "mix", "npm", "pnpm", "yarn"}, toks[i].value) ||
		i+2 >= len(toks) || toks[i+1] != (srcToken{value: ","}) || toks[i+2] != (srcToken{value: "["}) {
		return nil, 0, false
	}
	words := []string{toks[i].value}
	for j := i + 3; j < len(toks); j++ {
		switch {
		case toks[j] == srcToken{value: "]"}:
			return words, j, true
		case toks[j].str:
			words = append(words, toks[j].value)
			if j+1 < len(toks) && toks[j+1] == (srcToken{value: ","}) {
				j++
			}
		default:
			return nil, 0, false
		}
	}
	return nil, 0, false
}

// wholeSuite is the runner when the program and arguments run a whole test suite, else "".
func wholeSuite(words []string) string {
	words = commandWords(words)
	if len(words) >= 3 && slices.Contains([]string{"sh", "bash", "zsh"}, words[0]) && isCommandFlag(words[1]) {
		for _, inner := range shellCommands(words[2]) {
			if r := wholeSuite(inner); r != "" {
				return r
			}
		}
		return ""
	}
	if len(words) < 2 {
		return ""
	}
	prog, args := words[0], words[1:]
	switch prog {
	case "go":
		if args[0] != "test" || !slices.Contains(args[1:], "./...") {
			return ""
		}
		for _, a := range args[1:] {
			if a == "-run" || a == "--run" || strings.HasPrefix(a, "-run=") || strings.HasPrefix(a, "--run=") ||
				strings.HasPrefix(a, "-test.run") || strings.HasPrefix(a, "--test.run") {
				return ""
			}
		}
		return "go test"
	case "mix":
		if args[0] != "test" {
			return ""
		}
		for i := 1; i < len(args); i++ {
			a := args[i]
			if a == "--only" || a == "--failed" || a == "--stale" || strings.HasPrefix(a, "--only=") {
				return ""
			}
			if slices.Contains(mixValueFlags, a) {
				i++ // the value of a flag that does not select tests is not a path
				continue
			}
			if !strings.HasPrefix(a, "-") {
				return ""
			}
		}
		return "mix test"
	case "npm", "pnpm", "yarn":
		rest := args[1:]
		switch {
		case args[0] == "test":
		case args[0] == "run" && len(rest) > 0 && rest[0] == "test":
			rest = rest[1:]
		default:
			return ""
		}
		for _, a := range rest {
			if !strings.HasPrefix(a, "-") || a == "--" {
				return ""
			}
		}
		return prog + " test"
	}
	return ""
}

// mixValueFlags are the mix test flags that take a separate value and run the same tests.
var mixValueFlags = []string{"--exclude", "--include", "--seed", "--max-failures", "--timeout", "--formatter", "--max-cases", "--slowest", "--trace-timeout"}

// commandWords is the command itself: without a leading ( or { and a trailing ) or }, redirections,
// and the env, exec, time, nohup, timeout and nice prefixes that only wrap it.
func commandWords(words []string) []string {
	var out []string
	skip := false
	for i, w := range words {
		if skip {
			skip = false
			continue
		}
		if i == 0 {
			w = strings.TrimLeft(w, "({")
		}
		if i == len(words)-1 {
			w = strings.TrimRight(w, ")}")
		}
		switch {
		case w == "":
		case isRedirect(w):
			op := strings.TrimLeft(w, "0123456789")
			skip = op == ">" || op == ">>" || op == "<" // a bare operator: the next word is its target
		default:
			out = append(out, w)
		}
	}
	for len(out) > 1 {
		switch w := out[0]; {
		case slices.Contains([]string{"env", "exec", "time", "nohup", "command"}, w), isAssignment(w):
			out = out[1:]
		case w == "timeout":
			out = out[1:]
			for len(out) > 0 && strings.HasPrefix(out[0], "-") {
				out = out[1:]
			}
			if len(out) > 0 {
				out = out[1:] // the duration
			}
		case w == "nice":
			out = out[1:]
			for len(out) > 0 && strings.HasPrefix(out[0], "-") {
				flag := out[0]
				out = out[1:]
				if flag == "-n" && len(out) > 0 {
					out = out[1:]
				}
			}
		default:
			return out
		}
	}
	return out
}

// isRedirect reports whether the word is a redirection such as >out, 2>, >> or <in.
func isRedirect(w string) bool {
	w = strings.TrimLeft(w, "0123456789")
	return strings.HasPrefix(w, ">") || strings.HasPrefix(w, "<")
}

// isCommandFlag reports whether the word is a shell option cluster that includes -c.
func isCommandFlag(w string) bool {
	return len(w) > 1 && w[0] == '-' && strings.Contains(w[1:], "c") && strings.Trim(w[1:], "abcefhilmnptuvxCo") == ""
}

// shellCommands splits one shell string at && || ; | & and newlines, outside quotes, into the
// words of each command, leaving out leading NAME=value assignments.
func shellCommands(s string) [][]string {
	var cmds [][]string
	var words []string
	var cur strings.Builder
	inWord := false
	endWord := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	endCmd := func() {
		endWord()
		for len(words) > 0 && isAssignment(words[0]) {
			words = words[1:]
		}
		if len(words) > 0 {
			cmds = append(cmds, words)
		}
		words = nil
	}
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == '\'' || c == '"':
			inWord = true
			for i++; i < len(rs) && rs[i] != c; i++ {
				if c == '"' && rs[i] == '\\' && i+1 < len(rs) {
					i++
				}
				cur.WriteRune(rs[i])
			}
		case c == '\\' && i+1 < len(rs):
			i++
			inWord = true
			cur.WriteRune(rs[i])
		case c == ';' || c == '|' || c == '&' || c == '\n':
			endCmd()
		case c == ' ' || c == '\t' || c == '\r':
			endWord()
		default:
			inWord = true
			cur.WriteRune(c)
		}
	}
	endCmd()
	return cmds
}

// isAssignment reports whether the word is a NAME=value environment assignment.
func isAssignment(w string) bool {
	name, _, ok := strings.Cut(w, "=")
	if !ok || name == "" {
		return false
	}
	for i, c := range name {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// scanSource tokenises JavaScript source for the judgement: comments are dropped, a string
// literal becomes one token with its unescaped value (a template literal with an interpolation is
// one non-string token, since its value is unknown), the characters , [ ] are tokens of their
// own and anything else is one non-string token per character.
func scanSource(src string) []srcToken {
	var toks []srcToken
	rs := []rune(src)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == '/' && i+1 < len(rs) && rs[i+1] == '/':
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(rs) && rs[i+1] == '*':
			i += 2
			for i+1 < len(rs) && !(rs[i] == '*' && rs[i+1] == '/') {
				i++
			}
			i++
		case c == '\'' || c == '"' || c == '`':
			var v strings.Builder
			dynamic := false
			for i++; i < len(rs) && rs[i] != c; i++ {
				if rs[i] == '\\' && i+1 < len(rs) {
					i++
					switch rs[i] {
					case 'n':
						v.WriteRune('\n')
					case 't':
						v.WriteRune('\t')
					case 'r':
						v.WriteRune('\r')
					default:
						v.WriteRune(rs[i])
					}
					continue
				}
				if c == '`' && rs[i] == '$' && i+1 < len(rs) && rs[i+1] == '{' {
					dynamic = true
				}
				v.WriteRune(rs[i])
			}
			if dynamic {
				toks = append(toks, srcToken{value: "x"})
			} else {
				toks = append(toks, srcToken{str: true, value: v.String()})
			}
		case c == ',' || c == '[' || c == ']':
			toks = append(toks, srcToken{value: string(c)})
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
		default:
			toks = append(toks, srcToken{value: "x"})
		}
	}
	return toks
}
