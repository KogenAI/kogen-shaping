package shape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
	"github.com/KogenAI/kogen-verification/verify"
)

// SubmitLiteralCheckTool is the one tool of the literal check call.
const SubmitLiteralCheckTool = "submit_literal_check"

const literalInstructions = `You check that one candidate Intent package carries the exact literal values that the feature request and the settled answers require. You see the request, the settled answers and the complete candidate package (intent.md, acceptance.feature, its model and its step files). Read-only: do not use any tool except ` + SubmitLiteralCheckTool + `.

List every literal value that the request or a settled answer explicitly requires. Give one item for each, with:
- literal: the value, copied exactly;
- source: request, answer <n> (question n of the latest round of answers) or round <r> answer <n> (question n of an earlier round r);
- source_quote: the passage of that source that requires it, copied exactly;
- status: required, superseded, informal or secret;
- package_value: the value that the package uses in its place, empty when there is none;
- why: one short reason.
A required literal is a value that the request or a settled answer explicitly requires the implementation or an example to use exactly, such as a path, command, name, identifier, number or example input; descriptive text that may be paraphrased is not a literal.
Give status superseded for a value that the request or a later answer explicitly replaces, and status secret for secret material such as a password, token, key or credential; for secret material leave literal, source_quote, package_value and why empty.
Copy each literal exactly as its source writes it, with source_quote copied exactly from that source and containing the literal; Kogen itself compares each literal with the package byte for byte.
Give status informal for a value that appears only in informal background, an aside or an example the source marks optional; Kogen never demands an informal value.
Answer by calling ` + SubmitLiteralCheckTool + `.`

// literalUnknownReason is the reason of the literal check event and progress line of a shaping
// whose original request is not available.
const literalUnknownReason = "the original request is not available"

func literalSchema() map[string]any {
	s := map[string]any{"type": "string"}
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"literal":       s,
			"source":        s,
			"source_quote":  s,
			"package_value": s,
			"why":           s,
			"status":        map[string]any{"type": "string", "enum": []any{"required", "superseded", "informal", "secret"}},
		},
		"required":             []any{"literal", "source", "source_quote", "package_value", "why", "status"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"items": map[string]any{"type": "array", "items": item}},
		"required":             []any{"items"},
		"additionalProperties": false,
	}
}

func literalToolDef() map[string]any {
	return map[string]any{
		"type": "function", "name": SubmitLiteralCheckTool, "strict": true,
		"description": "Submit the literal check: one item per literal value that the request or a settled answer requires, with its source, quote, status and the value the package uses in its place.",
		"parameters":  literalSchema(),
	}
}

func literalInvalid(why string) error {
	return sys.Fail("shape", "literal_check_invalid", "the literal check answer is not a valid "+SubmitLiteralCheckTool+" payload ("+why+"); nothing was published", 1)
}

type literalItem struct {
	literal, source, quote, packageValue, why, status string
}

func parseLiteralCheck(args map[string]any) ([]literalItem, error) {
	list, ok := args["items"].([]any)
	if !ok {
		return nil, literalInvalid("items must be a list")
	}
	if len(args) != 1 {
		return nil, literalInvalid("the payload has fields other than items")
	}
	items := []literalItem{}
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, literalInvalid("an item must be an object")
		}
		var it literalItem
		var okL, okS, okQ, okP, okW, okT bool
		it.literal, okL = m["literal"].(string)
		it.source, okS = m["source"].(string)
		it.quote, okQ = m["source_quote"].(string)
		it.packageValue, okP = m["package_value"].(string)
		it.why, okW = m["why"].(string)
		it.status, okT = m["status"].(string)
		if !okL || !okS || !okQ || !okP || !okW || !okT || len(m) != 6 ||
			!slices.Contains([]string{"required", "superseded", "informal", "secret"}, it.status) {
			return nil, literalInvalid("an item is malformed")
		}
		items = append(items, it)
	}
	return items, nil
}

// literalSources are the texts the literal check's sources name: "request", "answer <n>" for
// question n of the latest round and "round <r> answer <n>" for question n of an earlier round
// r. An answer's text is its resolved answer; an unanswered question has no entry.
func literalSources(request string, rounds []map[string]any) map[string]string {
	sources := map[string]string{"request": request}
	for r, round := range rounds {
		answers := answersOf(round)
		for i, q := range questionsOf(round) {
			text, ok := resolvedAnswer(q, answers[strconv.Itoa(i+1)])
			if !ok {
				continue
			}
			if r == len(rounds)-1 {
				sources[fmt.Sprintf("answer %d", i+1)] = text
			} else {
				sources[fmt.Sprintf("round %d answer %d", r+1, i+1)] = text
			}
		}
	}
	return sources
}

// resolvedAnswer is an answer's text: the label of the option that a single letter a-z selects,
// else the text as given; false when the question is unanswered.
func resolvedAnswer(question map[string]any, answer any) (string, bool) {
	text, ok := answer.(string)
	if !ok || text == "" {
		return "", false
	}
	if letterAnswer.MatchString(text) {
		if opts := optionsOf(question); int(text[0]-'a') < len(opts) {
			return textOf(opts[text[0]-'a']["label"]), true
		}
	}
	return text, true
}

// literalAnswersText is the answers section of the literal check input.
func literalAnswersText(rounds []map[string]any) string {
	var lines []string
	for r, round := range rounds {
		answers := answersOf(round)
		for i, q := range questionsOf(round) {
			text, ok := resolvedAnswer(q, answers[strconv.Itoa(i+1)])
			if !ok {
				continue
			}
			if r == len(rounds)-1 {
				lines = append(lines, fmt.Sprintf("answer %d: %s", i+1, text))
			} else {
				lines = append(lines, fmt.Sprintf("round %d answer %d: %s", r+1, i+1, text))
			}
		}
	}
	if len(lines) == 0 {
		return "(none)"
	}
	return strings.Join(lines, "\n")
}

type packageFile struct{ path, content string }

// literalInput is the user input of the literal check.
func literalInput(slug, request string, rounds []map[string]any, files []packageFile) string {
	var b strings.Builder
	b.WriteString("Slug: " + slug + "\n\n# Request\n\n" + request + "\n\n# Answers\n\n" + literalAnswersText(rounds) + "\n\n# Candidate package\n\n")
	for _, f := range files {
		b.WriteString("## " + f.path + "\n\n" + f.content + "\n\n")
	}
	return b.String()
}

// candidatePackage reads the files of the candidate package as written to the checkout: the
// Intent's intent.md, its acceptance.feature, its model file when Model: names one, then every
// other Acceptance: path in its listed order, each once.
func (sh *shaping) candidatePackage() ([]packageFile, error) {
	intent, err := project.LoadIntent(sh.root, sh.slug)
	if err != nil {
		return nil, err
	}
	paths := []string{project.IntentMDPath(sh.slug), project.FeaturePath(sh.slug)}
	if intent.Model != nil && !intent.Model.None && intent.Model.Path != "" {
		paths = append(paths, intent.Model.Path)
	}
	paths = append(paths, intent.Acceptance...)
	var files []packageFile
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		b, err := os.ReadFile(filepath.Join(sh.root, p))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		files = append(files, packageFile{p, string(b)})
	}
	return files, nil
}

func literalProblemLine(it literalItem, red *verify.Redactor) string {
	r := red.Redact
	if it.status == "changed" {
		return fmt.Sprintf("literal: %s | source: %s | source_quote: \"%s\" | status: changed | package_value: %s | why: %s",
			r(it.literal), r(it.source), r(it.quote), r(it.packageValue), r(it.why))
	}
	return fmt.Sprintf("literal: %s | source: %s | source_quote: \"%s\" | status: missing | why: %s",
		r(it.literal), r(it.source), r(it.quote), r(it.why))
}

// literalOutcome applies Kogen's rules to the model's items: the final status of each, the
// event, and the feedback lines (nil when no problem).
func literalOutcome(items []literalItem, sources map[string]string, files []packageFile, red *verify.Redactor) (map[string]any, []string) {
	inPackage := func(s string) bool {
		for _, f := range files {
			if strings.Contains(f.content, s) {
				return true
			}
		}
		return false
	}
	counts := map[string]int{}
	events := make([]any, len(items))
	var problems []string
	for i := range items {
		it := &items[i]
		src, hasSource := sources[it.source]
		switch {
		case it.status == "secret" || red.Redact(it.literal) != it.literal ||
			(hasSource && strings.Contains(src, it.literal) && !strings.Contains(red.Redact(src), it.literal)):
			it.status = "secret"
		case strings.TrimSpace(it.literal) == "" || !hasSource || !strings.Contains(src, it.quote) || !strings.Contains(it.quote, it.literal):
			it.status = "unverified"
		case it.status == "superseded" || it.status == "informal":
		case inPackage(it.literal):
			it.status = "present"
		case strings.TrimSpace(it.packageValue) != "" && it.packageValue != it.literal && inPackage(it.packageValue):
			it.status = "changed"
		default:
			it.status = "missing"
		}
		counts[it.status]++
		if it.status == "secret" {
			events[i] = map[string]any{"source": red.Redact(it.source), "status": "secret"}
			continue
		}
		events[i] = map[string]any{"literal": red.Redact(it.literal), "source": red.Redact(it.source), "source_quote": red.Redact(it.quote),
			"status": it.status, "package_value": red.Redact(it.packageValue), "why": red.Redact(it.why)}
		if it.status == "missing" || it.status == "changed" {
			problems = append(problems, literalProblemLine(*it, red))
		}
	}
	event := map[string]any{"event": "literal_check", "status": "compared", "present": counts["present"], "missing": counts["missing"],
		"changed": counts["changed"], "superseded": counts["superseded"], "informal": counts["informal"], "secret": counts["secret"],
		"unverified": counts["unverified"], "items": events}
	if len(problems) == 0 {
		return event, nil
	}
	head := fmt.Sprintf("Literal check: %d required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:", len(problems))
	return event, append([]string{head}, problems...)
}

// literalCheck makes one fresh read-only call on the request, the settled answers and the
// complete candidate package, and returns the feedback lines, nil when the candidate may go on.
// With no original request it compares nothing and records unknown.
func (sh *shaping) literalCheck(ctx context.Context) ([]string, error) {
	if strings.TrimSpace(sh.request) == "" {
		sh.ev.Record(map[string]any{"event": "literal_check", "status": "unknown", "reason": literalUnknownReason})
		sh.progress("kogen: request literal check unknown: " + literalUnknownReason + "; required literals were not compared")
		return nil, nil
	}
	files, err := sh.candidatePackage()
	if err != nil {
		return nil, err
	}
	input := literalInput(sh.slug, sh.request, sh.rounds, files)
	var args map[string]any
	sh.literalChecks++
	if sh.ext != nil {
		x, err := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"),
			fmt.Sprintf("%s/executor-literal-%d", sh.ev.Dir, sh.literalChecks), sh.slug, sh.role, literalInstructions+"\n\n"+input, "", mustJSON(literalSchema()), sh.ev, nil)
		if err != nil {
			return nil, err
		}
		defer x.close()
		res, err := x.round(ctx, "")
		sh.literalRounds += x.rounds
		if err != nil {
			return nil, err
		}
		if res.Ending != executor.EndFinish {
			return nil, literalInvalid("the session ended without a result")
		}
		if args, err = submissionArgs(res.Structured); err != nil {
			return nil, literalInvalid("not a JSON object")
		}
	} else {
		sh.progress("kogen: checking the required literals")
		revision, err := CurrentRevision(sh.repo)
		if err != nil {
			return nil, err
		}
		cacheID := startConversation(sh.ev, "literal-check")
		req := conversationRequest(provider.Request{
			Role: ProviderRole(sh.role), Instructions: literalInstructions,
			Input: []any{map[string]any{"role": "user", "content": input}}, Tools: []any{literalToolDef()},
			Scope: sh.ev.Dir, Context: map[string]any{"phase": "shape", "role": "shape", "slug": sh.slug},
		}, cacheID)
		call := sh.ev.BeginCall("shape", "", revision.Map(), "literal check", sh.role).Sized(sh.client, req)
		t0 := time.Now()
		resp, err := sh.client.Respond(ctx, sh.s, req, call)
		call.End(resp, err)
		if err != nil {
			if e, ok := sys.AsError(err); ok && e.Class == "provider" {
				return nil, sh.providerFailed(e)
			}
			return nil, err
		}
		sh.progress(fmt.Sprintf("kogen: literal check response %s wall_ms=%d", resp.Usage.Line(), time.Since(t0).Milliseconds()))
		sh.literalUsage = sh.literalUsage.Add(resp.Usage)
		calls := tools.Calls(resp.Items)
		if len(calls) != 1 || calls[0].Name != SubmitLiteralCheckTool {
			return nil, literalInvalid("expected one " + SubmitLiteralCheckTool + " call")
		}
		args = calls[0].Args()
	}
	items, err := parseLiteralCheck(args)
	if err != nil {
		return nil, err
	}
	home := sh.s.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	event, lines := literalOutcome(items, literalSources(sh.request, sh.rounds), files, verify.NewRedactor(sh.s.Env, home))
	sh.ev.Record(event)
	return lines, nil
}
