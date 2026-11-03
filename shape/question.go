package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/KogenAI/kogen-executor/executor"
	"kogen-loop/provider"
	"kogen-loop/sys"
	"kogen-loop/tools"
)

// SubmitQuestionCheckTool is the one tool of the question check call.
const SubmitQuestionCheckTool = "submit_question_check"

const questionInstructions = `You check the questions a shaper wants to ask the user about one feature request. Only UI/UX choices that the request and the user's answers leave open, and mandatory authentication or security handoffs, may reach the user; the shaper investigates and decides everything else itself. You see only the request text, the earlier questions with the user's answers, and the questions. Read-only: do not use any tool except ` + SubmitQuestionCheckTool + `.

Give one item for every question: its number, one class and why (one short reason). The classes:
- ui_ux: a choice of what the user sees or does in the product's interface or interaction (layout, wording, flow, feedback, input behavior) that the request and the earlier answers leave open;
- handoff: a mandatory authentication or security step that only the user can perform, such as signing in or granting access;
- answered: the request or an earlier answer already settles it;
- technical: an implementation, architecture, library, protocol, data or testing choice;
- product_policy: a business, pricing, permission or other product rule that is not a choice of interface or interaction;
- environment: whether a tool, service, account, credential, network or other resource is available where the work runs;
- repository_fact: a fact that the repository's files, history or commands can establish.
Answer by calling ` + SubmitQuestionCheckTool + `.`

// questionClasses are the classes of the question check; the first two reach the user.
var questionClasses = []any{"ui_ux", "handoff", "answered", "technical", "product_policy", "environment", "repository_fact"}

func questionSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"number": map[string]any{"type": "integer"},
			"class":  map[string]any{"type": "string", "enum": questionClasses},
			"why":    map[string]any{"type": "string"},
		},
		"required":             []any{"number", "class", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"items": map[string]any{"type": "array", "items": item}},
		"required":             []any{"items"},
		"additionalProperties": false,
	}
}

func questionToolDef() map[string]any {
	return map[string]any{
		"type": "function", "name": SubmitQuestionCheckTool, "strict": true,
		"description": "Submit the question check: one item per question with its number, class and reason.",
		"parameters":  questionSchema(),
	}
}

func questionInvalid(why string) error {
	return sys.Fail("shape", "question_check_invalid", "the question check answer is not a valid "+SubmitQuestionCheckTool+" payload ("+why+"); nothing was asked", 1)
}

type questionItem struct {
	number     int
	class, why string
}

// wholeNumber is a decoded JSON integer: int64 from native tool arguments, float64 from an
// external result.
func wholeNumber(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), n == math.Trunc(n) && math.Abs(n) < 1<<31
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	}
	return 0, false
}

// parseQuestionCheck is the answer's items, one for each question 1 to count, in question order.
func parseQuestionCheck(args map[string]any, count int) ([]questionItem, error) {
	list, ok := args["items"].([]any)
	if !ok {
		return nil, questionInvalid("items must be a list")
	}
	var items []questionItem
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, questionInvalid("an item is malformed")
		}
		n, okN := wholeNumber(m["number"])
		class, okC := m["class"].(string)
		why, okW := m["why"].(string)
		if !okN || !okC || !okW || !slices.Contains(questionClasses, any(class)) {
			return nil, questionInvalid("an item is malformed")
		}
		items = append(items, questionItem{number: n, class: class, why: why})
	}
	slices.SortFunc(items, func(a, b questionItem) int { return a.number - b.number })
	if len(items) != count {
		return nil, questionInvalid(fmt.Sprintf("expected one item for each question 1 to %d", count))
	}
	for i, it := range items {
		if it.number != i+1 {
			return nil, questionInvalid(fmt.Sprintf("expected one item for each question 1 to %d", count))
		}
	}
	return items, nil
}

// questionsText is the questions as the question check reads them.
func questionsText(questions []map[string]any) string {
	var lines []string
	for i, q := range questions {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, textOf(q["question"])))
		for k, o := range optionsOf(q) {
			line := fmt.Sprintf("   %c) %s", 'a'+k, textOf(o["label"]))
			if r, ok := o["recommended"]; ok && r != nil && r != false {
				line += " (recommended)"
			}
			lines = append(lines, line)
		}
		if q["kind"] == "handoff" {
			lines = append(lines, "   kind: handoff")
		}
	}
	return strings.Join(lines, "\n")
}

const questionHeadline = "Question check: %d of %d question(s) are not UI/UX choices left to the user; nothing was asked. " +
	"Investigate and decide each one yourself: read the repository or run a probe, then record a Decisions line that cites what you read or ran before its source tag, " +
	"as - <decision> (evidence: <repository path>) (source: request) or - <decision> (evidence: ran `<command>`) (source: request). " +
	"Where the repository has no evidence or conflicting evidence, record the uncertainty you decided under instead, as - <decision> (assumption: <the uncertainty>) (source: request); the user confirms or reverses it at approval. " +
	"Ask the user only UI/UX questions that the request and the answers leave open, or a mandatory authentication or security handoff."

// questionInstruction is what the shaper does with a refused question of the class.
func questionInstruction(class string) string {
	switch class {
	case "answered":
		return "the request or an earlier answer settles it; follow it and keep its (source: request) or (source: answer N)"
	case "environment":
		return "do not ask about the environment; proceed without it and record that Decision, or, if the request cannot be shaped without it, submit only such environment questions again and Shape ends with an environment fault"
	}
	return "decide it from what the repository and the request establish"
}

// questionClassify makes the question check call: one fresh read-only request or session.
func (sh *shaping) questionClassify(ctx context.Context, questions []map[string]any) ([]questionItem, error) {
	earlier := sh.earlier
	if earlier == "" {
		earlier = "none"
	}
	input := "Slug: " + sh.slug + "\n\n# Request\n\n" + sh.request + "\n\n# Earlier questions and answers\n\n" + earlier + "\n\n# Questions\n\n" + questionsText(questions)
	var args map[string]any
	sh.questionChecks++
	if sh.ext != nil {
		x, err := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"),
			fmt.Sprintf("%s/executor-question-%d", sh.ev.Dir, sh.questionChecks), sh.slug, sh.role, questionInstructions+"\n\n"+input, "", mustJSON(questionSchema()), sh.ev, nil)
		if err != nil {
			return nil, err
		}
		defer x.close()
		res, err := x.round(ctx, "")
		sh.questionRounds += x.rounds
		if err != nil {
			return nil, err
		}
		if res.Ending != executor.EndFinish {
			return nil, questionInvalid("the session ended without a result")
		}
		if args, err = submissionArgs(res.Structured); err != nil {
			return nil, questionInvalid("not a JSON object")
		}
	} else {
		sh.progress("kogen: checking the questions")
		revision, err := CurrentRevision(sh.repo)
		if err != nil {
			return nil, err
		}
		cacheID := startConversation(sh.ev, "question-check")
		req := conversationRequest(provider.Request{
			Role: ProviderRole(sh.role), Instructions: questionInstructions,
			Input: []any{map[string]any{"role": "user", "content": input}}, Tools: []any{questionToolDef()},
			Scope: sh.ev.Dir, Context: map[string]any{"phase": "shape", "role": "shape", "slug": sh.slug},
		}, cacheID)
		call := sh.ev.BeginCall("shape", "", revision.Map(), "question check", sh.role).Sized(sh.client, req)
		t0 := time.Now()
		resp, err := sh.client.Respond(ctx, sh.s, req, call)
		call.End(resp, err)
		if err != nil {
			if e, ok := sys.AsError(err); ok && e.Class == "provider" {
				return nil, sh.providerFailed(e)
			}
			return nil, err
		}
		sh.progress(fmt.Sprintf("kogen: question check response %s wall_ms=%d", resp.Usage.Line(), time.Since(t0).Milliseconds()))
		sh.questionUsage = sh.questionUsage.Add(resp.Usage)
		calls := tools.Calls(resp.Items)
		if len(calls) != 1 || calls[0].Name != SubmitQuestionCheckTool {
			return nil, questionInvalid("expected one " + SubmitQuestionCheckTool + " call")
		}
		args = calls[0].Args()
	}
	return parseQuestionCheck(args, len(questions))
}

// questionGate is the question check of a questions submission. refused holds the question
// lines of a refusal (nil when every question may reach the user), with the feedback text; an
// environment-only refusal that repeats, or that is the last allowed submission, is the
// environment fault instead.
func (sh *shaping) questionGate(ctx context.Context, questions []map[string]any) (text string, refused []string, err error) {
	items, err := sh.questionClassify(ctx, questions)
	if err != nil {
		return "", nil, err
	}
	events := make([]any, len(items))
	asked, environment := 0, 0
	for i, it := range items {
		events[i] = map[string]any{"number": it.number, "question": textOf(questions[i]["question"]), "class": it.class, "why": it.why}
		switch it.class {
		case "ui_ux", "handoff":
			asked++
			continue
		case "environment":
			environment++
		}
		refused = append(refused, fmt.Sprintf("- question %d (%s): %s | why: %s | %s", it.number, it.class, textOf(questions[i]["question"]), it.why, questionInstruction(it.class)))
	}
	sh.ev.Record(map[string]any{"event": "question_check", "items": events, "asked": asked, "refused": len(items) - asked})
	if refused == nil {
		return "", nil, nil
	}
	sh.refused("question_check", refused)
	if environment == len(items) {
		sh.environmentRefusals++
		if sh.environmentRefusals >= 2 || sh.priorInvalid >= submissionRetryLimit(sh.in.SubmissionRetryLimit)-1 {
			e := sys.Fail("environment", "shape_environment", "shaping cannot go on without something only your environment can provide; nothing was asked or published", 3)
			for _, q := range questions {
				e.Lines = append(e.Lines, "- "+textOf(q["question"]))
			}
			return "", nil, e
		}
	}
	return strings.Join(append([]string{fmt.Sprintf(questionHeadline, len(refused), len(items))}, refused...), "\n"), refused, nil
}
