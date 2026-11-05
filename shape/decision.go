package shape

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-loop/tools"
)

// SubmitDecisionCheckTool is the one tool of the decision check call.
const SubmitDecisionCheckTool = "submit_decision_check"

const decisionInstructions = `You check the stated decisions of one candidate Intent against the feature request that it answers. You see only the request text and the candidate intent.md. Read-only: do not use any tool except ` + SubmitDecisionCheckTool + `.

Give one item for every line of the candidate's "## Decisions" section and for every explicit result the intent states for a concrete input in its prose or examples. For each item give:
- statement: text copied exactly from intent.md (for a Decisions line, the whole line);
- request_quotes: the passages of the request that support or contradict it, each copied exactly from the request (none when the request is silent);
- verdict: follows (the request states the same value), contradicts (the request states a different value) or unsupported (the request does not state it);
- intent_value, request_value and difference: the two values and what makes them differ, for contradicts only;
- why: one short reason.
Compare values, not notation: before comparing, normalise digit grouping (1,000 and 1000), percentages and fractions (50% and 0.5), units of the same quantity (2 minutes and 120 seconds), date layouts, the order of a list whose order does not matter, and letter case where case does not matter.
A statement that restates the request's value in another notation follows the request.
Give contradicts only when the values still differ after that normalisation, such as a different number, a wrongly converted unit or a different order where order matters; then give intent_value (the value as the statement writes it), request_value (the value as a request quote writes it) and difference (what makes them differ). For follows and unsupported give these three as empty strings.
A Decision line may be marked as an assumption with (assumption: <what the user must confirm>) placed before its trailing tag; a marked line keeps its trailing (source: request) or (source: answer N).
Answer by calling ` + SubmitDecisionCheckTool + `.`

// assumptionRule is the feedback line that says exactly how to mark an assumption.
const assumptionRule = "Mark an assumption with (assumption: <what the user must confirm>) before the required trailing tag, e.g. - <decision> (assumption: <what the user must confirm>) (source: request); every Decision line keeps its trailing (source: request) or (source: answer N)."

func decisionSchema() map[string]any {
	s := map[string]any{"type": "string"}
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"statement":      s,
			"request_quotes": map[string]any{"type": "array", "items": s},
			"verdict":        map[string]any{"type": "string", "enum": []any{"follows", "contradicts", "unsupported"}},
			"intent_value":   s,
			"request_value":  s,
			"difference":     s,
			"why":            s,
		},
		"required":             []any{"statement", "request_quotes", "verdict", "intent_value", "request_value", "difference", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"items": map[string]any{"type": "array", "items": item}},
		"required":             []any{"items"},
		"additionalProperties": false,
	}
}

func decisionToolDef() map[string]any {
	return map[string]any{
		"type": "function", "name": SubmitDecisionCheckTool, "strict": true,
		"description": "Submit the decision check: one item per stated decision with its request quotes, verdict and, for a contradiction, the two values.",
		"parameters":  decisionSchema(),
	}
}

func decisionInvalid(why string) error {
	return sys.Fail("shape", "decision_check_invalid", "the decision check answer is not a valid "+SubmitDecisionCheckTool+" payload ("+why+"); nothing was published", 1)
}

type decisionItem struct {
	statement    string
	quotes       []string
	verdict      string
	intentValue  string
	requestValue string
	difference   string
	why          string
}

func parseDecisionCheck(args map[string]any) ([]decisionItem, error) {
	list, ok := args["items"].([]any)
	if !ok {
		return nil, decisionInvalid("items must be a list")
	}
	items := []decisionItem{}
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, decisionInvalid("an item must be an object")
		}
		var it decisionItem
		var okS, okV, okI, okR, okD, okW bool
		it.statement, okS = m["statement"].(string)
		it.verdict, okV = m["verdict"].(string)
		it.intentValue, okI = m["intent_value"].(string)
		it.requestValue, okR = m["request_value"].(string)
		it.difference, okD = m["difference"].(string)
		it.why, okW = m["why"].(string)
		quotes, okQ := m["request_quotes"].([]any)
		if !okS || !okV || !okI || !okR || !okD || !okW || !okQ || len(m) != 7 ||
			(it.verdict != "follows" && it.verdict != "contradicts" && it.verdict != "unsupported") {
			return nil, decisionInvalid("an item is malformed")
		}
		it.quotes = []string{}
		for _, q := range quotes {
			qs, ok := q.(string)
			if !ok {
				return nil, decisionInvalid("a request quote must be a string")
			}
			it.quotes = append(it.quotes, qs)
		}
		items = append(items, it)
	}
	if len(args) != 1 {
		return nil, decisionInvalid("the payload has fields other than items")
	}
	return items, nil
}

// normalizeNotation is normalizeWhitespace followed by removing every comma that has a digit
// just before it and exactly three digits just after it, the third not followed by another
// digit. Case is kept.
func normalizeNotation(s string) string {
	s = normalizeWhitespace(s)
	isDigit := func(i int) bool { return i >= 0 && i < len(s) && s[i] >= '0' && s[i] <= '9' }
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == ',' && isDigit(i-1) && isDigit(i+1) && isDigit(i+2) && isDigit(i+3) && !isDigit(i+4) {
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// decisionOutcome applies the mechanical rules: final items (verdicts replaced where a
// statement, quote or value is not found, or where two named values are the same value), the
// event, and the feedback lines (nil when no problem).
func decisionOutcome(items []decisionItem, request, intent string) (map[string]any, []string) {
	counts := map[string]int{}
	events := make([]any, len(items))
	var problems []string
	for i := range items {
		it := &items[i]
		supported := containsNormalized(intent, it.statement)
		for _, q := range it.quotes {
			supported = supported && containsNormalized(request, q)
		}
		if !supported {
			it.verdict = "unsupported"
		}
		if it.verdict == "contradicts" {
			named := normalizeWhitespace(it.intentValue) != "" && containsNormalized(it.statement, it.intentValue)
			found := false
			for _, q := range it.quotes {
				found = found || containsNormalized(q, it.requestValue)
			}
			named = named && normalizeWhitespace(it.requestValue) != "" && found
			switch {
			case !named:
				it.verdict = "unsupported"
			case normalizeNotation(it.intentValue) == normalizeNotation(it.requestValue):
				it.verdict = "follows"
			}
		}
		counts[it.verdict]++
		events[i] = map[string]any{"statement": it.statement, "request_quotes": it.quotes, "verdict": it.verdict,
			"intent_value": it.intentValue, "request_value": it.requestValue, "difference": it.difference, "why": it.why}
		if it.verdict == "follows" || markedStatement(it.statement) {
			continue
		}
		quoted := make([]string, len(it.quotes))
		for j, q := range it.quotes {
			quoted[j] = `"` + q + `"`
		}
		if it.verdict == "contradicts" {
			problems = append(problems, fmt.Sprintf("statement: %s | request_quotes: %s | verdict: contradicts | intent_value: %s | request_value: %s | difference: %s | why: %s",
				it.statement, strings.Join(quoted, ", "), it.intentValue, it.requestValue, it.difference, it.why))
			continue
		}
		problems = append(problems, fmt.Sprintf("statement: %s | request_quotes: %s | verdict: unsupported | why: %s", it.statement, strings.Join(quoted, ", "), it.why))
	}
	event := map[string]any{"event": "decision_check", "follows": counts["follows"], "contradicts": counts["contradicts"], "unsupported": counts["unsupported"], "items": events}
	if len(problems) == 0 {
		return event, nil
	}
	head := fmt.Sprintf("Decision check: %d stated decisions are not supported by the request; follow the request, cite the repository evidence that decides it with (evidence: ...), or mark the decision as an assumption the user must confirm:", len(problems))
	return event, append([]string{head, assumptionRule}, problems...)
}

// decisionCheck makes one fresh read-only call on the candidate intent.md and returns the
// feedback lines, nil when the candidate may go on.
func (sh *shaping) decisionCheck(ctx context.Context, intent string) ([]string, error) {
	input := "Slug: " + sh.slug + "\n\n# Request\n\n" + sh.request + "\n\n# Candidate intent.md\n\n" + intent
	var args map[string]any
	sh.decisionChecks++
	if sh.ext != nil {
		x, err := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"),
			fmt.Sprintf("%s/executor-decision-%d", sh.ev.Dir, sh.decisionChecks), sh.slug, sh.role, decisionInstructions+"\n\n"+input, "", mustJSON(decisionSchema()), sh.ev, nil)
		if err != nil {
			return nil, err
		}
		defer x.close()
		res, err := x.round(ctx, "")
		sh.decisionRounds += x.rounds
		if err != nil {
			return nil, err
		}
		if res.Ending != executor.EndFinish {
			return nil, decisionInvalid("the session ended without a result")
		}
		if args, err = submissionArgs(res.Structured); err != nil {
			return nil, decisionInvalid("not a JSON object")
		}
	} else {
		sh.progress("kogen: checking the stated decisions")
		revision, err := CurrentRevision(sh.repo)
		if err != nil {
			return nil, err
		}
		cacheID := startConversation(sh.ev, "decision-check")
		req := conversationRequest(provider.Request{
			Role: ProviderRole(sh.role), Instructions: decisionInstructions,
			Input: []any{map[string]any{"role": "user", "content": input}}, Tools: []any{decisionToolDef()},
			Scope: sh.ev.Dir, Context: map[string]any{"phase": "shape", "role": "shape", "slug": sh.slug},
		}, cacheID)
		call := sh.ev.BeginCall("shape", "", revision.Map(), "decision check", sh.role).Sized(sh.client, req)
		t0 := time.Now()
		resp, err := sh.client.Respond(ctx, sh.s, req, call)
		call.End(resp, err)
		if err != nil {
			if e, ok := sys.AsError(err); ok && e.Class == "provider" {
				return nil, sh.providerFailed(e)
			}
			return nil, err
		}
		sh.progress(fmt.Sprintf("kogen: decision check response %s wall_ms=%d", resp.Usage.Line(), time.Since(t0).Milliseconds()))
		sh.decisionUsage = sh.decisionUsage.Add(resp.Usage)
		calls := tools.Calls(resp.Items)
		if len(calls) != 1 || calls[0].Name != SubmitDecisionCheckTool {
			return nil, decisionInvalid("expected one " + SubmitDecisionCheckTool + " call")
		}
		args = calls[0].Args()
	}
	items, err := parseDecisionCheck(args)
	if err != nil {
		return nil, err
	}
	event, lines := decisionOutcome(items, sh.request, intent)
	sh.ev.Record(event)
	return lines, nil
}

// markedStatement is a statement that carries an assumption or evidence marker: the decision
// check does not count it as a problem.
func markedStatement(statement string) bool {
	return strings.Contains(statement, "(assumption:") || strings.Contains(statement, "(evidence:")
}
