package shape

import (
	"fmt"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/provider"
	"github.com/KogenAI/kogen-verification/verify"
)

// noToolLimit is the finite protection against repeated prose: a native shaping conversation
// stops at this many answers without any tool call. It is separate from the invalid-submission
// budget.
const noToolLimit = 3

// Reminders for a tool-free answer. They are sent after every such answer, whether or not the
// route can also require a tool call at API level.
const (
	shapeReminder  = "Call an available tool to continue: inspect with read_file, list_files, search_text, probe or documentation, prepare files with propose_files, then submit a complete package or questions. Do not submit an incomplete placeholder."
	refineReminder = "Call submit_refinement with a nonempty files array of complete existing owned artifact contents (edits or readiness acknowledgment), or constraint."
)

// turnCounter counts the tool-free answers and invalid submissions of one native invocation, both
// cumulative across local working-window resets, and whether the next inference must require
// a tool call: only the inference immediately after a tool-free answer. Shape allows
// any preparation or submission tool; Refine retains its terminal acknowledgment.
type turnCounter struct {
	owner, target, reminder string
	noTool, invalid         int
	forced                  bool
}

func newTurns(owner string) *turnCounter {
	if owner == "refine" {
		return &turnCounter{owner: owner, target: "submit_refinement", reminder: refineReminder}
	}
	return &turnCounter{owner: owner, target: "any", reminder: shapeReminder}
}

// answered folds one response in: a tool-free answer schedules a forced inference (reported
// true), any answer with tool calls clears it.
func (t *turnCounter) answered(calls int) bool {
	t.forced = calls == 0
	if t.forced {
		t.noTool++
	}
	return t.forced
}

// recover makes the next inference a recovery inference: it keeps the route's automatic tool
// choice instead of requiring the terminal tool.
func (t *turnCounter) recover() { t.forced = false }

// exhausted reports whether the no-tool ceiling is reached.
func (t *turnCounter) exhausted() bool { return t.noTool >= noToolLimit }

// apply requires a tool call when a forced inference is pending and the
// route can force a tool. Without support the request keeps the existing reminder only.
func (t *turnCounter) apply(req *provider.Request) {
	if t.forced && provider.SupportsToolChoice(req.Role) {
		if t.target == "any" {
			req.RequireTool = true
		} else {
			req.ToolChoice = t.target
		}
	}
}

// announce records the enforcement of the request about to be sent, whether or not the API
// forces it.
func (t *turnCounter) announce(req provider.Request, role project.Role, ev *Evidence) {
	if !t.forced {
		return
	}
	name := role.Provider
	if name == "" {
		name = "chatgpt"
	}
	ev.Record(map[string]any{
		"event": "forced_submit_turn", "owner": t.owner, "target": t.target, "provider": name,
		"api_forcing_supported": req.ToolChoice != "" || req.RequireTool, "no_tool_turns": t.noTool, "invalid_submissions": t.invalid,
	})
}

// counts records the final counters of an invocation.
func (t *turnCounter) counts(ev *Evidence) {
	ev.Record(map[string]any{
		"event": "shaping_turn_counts", "owner": t.owner, "no_tool_turns": t.noTool, "invalid_submissions": t.invalid,
	})
}

// extend adds the counters and pending enforcement to a reset handoff, only when they differ
// from the start, so an ordinary handoff keeps its exact bytes.
func (t *turnCounter) extend(state map[string]any) map[string]any {
	if t.noTool > 0 {
		state["no_tool_turns"] = t.noTool
	}
	if t.forced {
		state["forced_submit_pending"] = t.target
	}
	return state
}

// toolFreeAnswerChars bounds the text retained for a tool-free answer, in Unicode code points.
const toolFreeAnswerChars = 2000

// lastSubmission is the shaper's last submit call that did not end the conversation: a refusal
// (kind outInvalid) or readiness failures returned for acknowledgment (kind outObservations).
// The zero value is no submission.
type lastSubmission struct {
	kind     outcomeKind
	attempt  int
	gate     string   // refusal: the gate
	problems []string // refusal: its problem lines as recorded
	path     string   // observations: the file relative to the project root
	// unchanged: the refusal of a package identical to an earlier refused one.
	unchanged bool
	// earliest: the lines naming the earliest unresolved refusal when it differs; else nil.
	earliest []string
}

// repairable reports whether a recovery inference may follow the submission's first tool-free
// answer: readiness failures, or a refusal of a package that is not an unchanged one.
func (l lastSubmission) repairable() bool {
	return l.kind == outObservations || (l.kind == outInvalid && !l.unchanged)
}

// recoveryEvent is the recovery_turn event of the recovery inference with the given request number.
func (l lastSubmission) recoveryEvent(request int) map[string]any {
	e := map[string]any{"event": "recovery_turn", "owner": "shape", "request": request, "attempt": l.attempt, "cause": "readiness_failures"}
	if l.kind == outInvalid {
		e["cause"], e["gate"] = "refused", l.gate
	}
	return e
}

// recoveryMessage is the user message of a recovery inference.
func (l lastSubmission) recoveryMessage() string {
	const rest = "This inference does not require submit: investigate with read_file, list_files, search_text, probe or documentation, add or correct files with propose_files, then call submit with the repaired package"
	if l.kind == outInvalid {
		return fmt.Sprintf("Recovery before submitting again: submission attempt %d was refused at gate %s and nothing was published. Your accumulated proposals are kept, as the cumulative proposal manifest lists. %s. An unchanged package is refused again. The refusal:\n%s",
			l.attempt, l.gate, rest, strings.Join(l.problems, "\n"))
	}
	return fmt.Sprintf("Recovery before submitting again: submission attempt %d returned readiness failures for acknowledgment (see %s) and nothing was published. Your accumulated proposals are kept, as the cumulative proposal manifest lists. %s, or acknowledge legitimate missing-feature red as the readiness feedback describes.",
		l.attempt, l.path, rest)
}

// noToolStop is the outcome of reaching the no-tool limit: it names the last submission the
// shaper was asked to act on and the first line of its last tool-free answer.
func noToolStop(last lastSubmission, answer string) outcome {
	lines := []string{"the shaper ended without calling submit"}
	var detail string
	switch last.kind {
	case outInvalid:
		detail = fmt.Sprintf("the shaper answered without calling any tool %d times since its last submission (no-tool limit); that submission (attempt %d) was refused at gate %s", noToolLimit, last.attempt, last.gate)
		lines = append(lines, fmt.Sprintf("last refusal (attempt %d, gate %s):", last.attempt, last.gate))
		lines = append(lines, last.problems...)
		lines = append(lines, last.earliest...)
	case outObservations:
		detail = fmt.Sprintf("the shaper answered without calling any tool %d times since its last submission (no-tool limit); that submission (attempt %d) returned readiness failures that were not acknowledged", noToolLimit, last.attempt)
		lines = append(lines, fmt.Sprintf("last submission (attempt %d): readiness failures returned for acknowledgment, see %s", last.attempt, last.path))
	default:
		detail = fmt.Sprintf("the shaper answered without calling any tool %d times (no-tool limit); no submission was made", noToolLimit)
	}
	first, _, _ := strings.Cut(answer, "\n")
	if first == "" {
		first = "(empty)"
	}
	lines = append(lines, "last tool-free answer: "+first)
	return outcome{kind: outNoTool, detail: detail, problems: lines}
}

// recordToolFreeAnswer records the text of a response without a tool call, redacted, trimmed and
// bounded, and returns the recorded text.
func (sh *shaping) recordToolFreeAnswer(items []map[string]any, noTool int, redactor *verify.Redactor) string {
	var parts []string
	for _, it := range items {
		if it["type"] != "message" || it["role"] != "assistant" {
			continue
		}
		var content []map[string]any
		switch c := it["content"].(type) {
		case []map[string]any:
			content = c
		case []any:
			for _, e := range c {
				if m, ok := e.(map[string]any); ok {
					content = append(content, m)
				}
			}
		}
		for _, part := range content {
			if part["type"] != "output_text" {
				continue
			}
			if t, ok := part["text"].(string); ok {
				parts = append(parts, t)
			}
		}
	}
	text := strings.TrimSpace(redactor.Redact(strings.Join(parts, "\n")))
	runes := []rune(text)
	event := map[string]any{
		"event": "tool_free_answer", "owner": "shape", "request": sh.requests, "no_tool_turns": noTool,
		"chars": len(runes), "truncated": len(runes) > toolFreeAnswerChars,
	}
	if len(runes) > toolFreeAnswerChars {
		text = string(runes[:toolFreeAnswerChars])
	}
	if len(runes) > 0 {
		event["text"] = text
	}
	sh.ev.Record(event)
	return text
}
