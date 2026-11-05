package shape

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

const questionInstructions = `You check the questions a shaper wants to ask the user about one feature request. Only UI/UX choices that the request and the user's answers leave open, and mandatory authentication or security handoffs, may reach the user; the shaper investigates and decides everything else itself. You see only the request text, the earlier questions with the user's answers, the decisions this shaping recorded with the evidence they cite, the questions, and the cited evidence: the text this shaping read or ran for each (evidence: ...) item a question cites. Read-only: do not use any tool except ` + SubmitQuestionCheckTool + `.

Give one item for every question: its number, one class, why (one short reason) and decision (the id of the recorded decision that settles it when the class is decided, otherwise an empty string). The classes:
- ui_ux: a choice of what the user sees or does in the product's interface or interaction (layout, wording, flow, feedback, input behavior) that the request and the earlier answers leave open;
- handoff: a mandatory authentication or security step that only the user can perform, such as signing in or granting access;
- answered: the request or an earlier answer already settles it;
- decided: a recorded decision settles it, or settles the technical, security, protocol, storage or limit choice it asks about, whatever its wording or the granularity of its options;
- technical: an implementation, architecture, library, protocol, data, storage, limit, security or testing choice, even when it is worded as what the user experiences or chooses;
- product_policy: a business, pricing, permission or other product rule that is not a choice of interface or interaction;
- environment: whether a tool, service, account, credential, network or other resource is available where the work runs, when it is not an environment_prerequisite;
- environment_prerequisite: a resource without which this shaping cannot write the Intent or run its acceptance, as the question's cited evidence shows: the cited text is about the resource the question names and shows it needed and missing, and neither the cited text, the request nor the earlier answers show it optional, needed only to run the finished feature against a real external service, or already stood in for by a fake, stub, simulation or recorded response; the why names the part of the Intent or acceptance that cannot be written or run without it and what in the cited text shows that. A question with no cited text shown, or whose cited text does not support the claimed necessity, is environment, whatever its own wording claims;
- repository_fact: a fact that the repository's files, history or commands can establish.
Answer by calling ` + SubmitQuestionCheckTool + `.`

// questionClasses are the classes of the question check; the first two reach the user.
var questionClasses = []any{"ui_ux", "handoff", "answered", "decided", "technical", "product_policy", "environment", "environment_prerequisite", "repository_fact"}

func questionSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"number":   map[string]any{"type": "integer"},
			"class":    map[string]any{"type": "string", "enum": questionClasses},
			"why":      map[string]any{"type": "string"},
			"decision": map[string]any{"type": "string"},
		},
		"required":             []any{"number", "class", "why", "decision"},
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
	decision   string // the id of the recorded decision that settles a decided question
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
		decision, okD := "", true
		if raw, has := m["decision"]; has {
			decision, okD = raw.(string)
		}
		if !okN || !okC || !okW || !okD || !slices.Contains(questionClasses, any(class)) {
			return nil, questionInvalid("an item is malformed")
		}
		items = append(items, questionItem{number: n, class: class, why: why, decision: decision})
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
		return "record a Decision and continue: proceed without it, with the stand-in the repository shows or one the Intent specifies, as - <decision> (assumption: <the uncertainty>) (source: request); " +
			"Shape stops only for environment questions that cite, as (evidence: <repository path>) or (evidence: ran `<command>`) in the question, what this shaping read or ran that shows the Intent or its acceptance cannot be written or run without the item"
	case "environment_prerequisite":
		return "the cited evidence shows the Intent or its acceptance cannot be written or run without it; if no stand-in can be used or specified, submit only such evidenced environment questions again and Shape stops with an environment fault that names the blocked part and the evidence; otherwise record a Decision and continue"
	}
	return "decide it from what the repository and the request establish"
}

// questionClassify makes the question check call: one fresh read-only request or session.
func (sh *shaping) questionClassify(ctx context.Context, questions []map[string]any) ([]questionItem, [][]string, error) {
	earlier := sh.earlier
	if earlier == "" {
		earlier = "none"
	}
	input := "Slug: " + sh.slug + "\n\n# Request\n\n" + sh.request + "\n\n# Earlier questions and answers\n\n" + earlier +
		sh.ledger.checkInputText(sh.checkoutDigest) + "\n\n# Questions\n\n" + questionsText(questions)
	cited, shown, err := sh.citedEvidence(questions)
	if err != nil {
		return nil, nil, err
	}
	if cited != "" {
		input += "\n\n# Cited evidence\n\n" + cited
	}
	var args map[string]any
	sh.questionChecks++
	if sh.ext != nil {
		x, err := startExternal(ctx, sh.ext, sh.extMax, sh.repo, sh.root, sh.s.Getenv("HOME"),
			fmt.Sprintf("%s/executor-question-%d", sh.ev.Dir, sh.questionChecks), sh.slug, sh.role, questionInstructions+"\n\n"+input, "", mustJSON(questionSchema()), sh.ev, nil)
		if err != nil {
			return nil, nil, err
		}
		defer x.close()
		res, err := x.round(ctx, "")
		sh.questionRounds += x.rounds
		if err != nil {
			return nil, nil, err
		}
		if res.Ending != executor.EndFinish {
			return nil, nil, questionInvalid("the session ended without a result")
		}
		if args, err = submissionArgs(res.Structured); err != nil {
			return nil, nil, questionInvalid("not a JSON object")
		}
	} else {
		sh.progress("kogen: checking the questions")
		revision, err := CurrentRevision(sh.repo)
		if err != nil {
			return nil, nil, err
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
				return nil, nil, sh.providerFailed(e)
			}
			return nil, nil, err
		}
		sh.progress(fmt.Sprintf("kogen: question check response %s wall_ms=%d", resp.Usage.Line(), time.Since(t0).Milliseconds()))
		sh.questionUsage = sh.questionUsage.Add(resp.Usage)
		calls := tools.Calls(resp.Items)
		if len(calls) != 1 || calls[0].Name != SubmitQuestionCheckTool {
			return nil, nil, questionInvalid("expected one " + SubmitQuestionCheckTool + " call")
		}
		args = calls[0].Args()
	}
	items, err := parseQuestionCheck(args, len(questions))
	return items, shown, err
}

// citedEvidence is the "# Cited evidence" blocks of the questions' (evidence: ...) items, ""
// when no question cites a nonempty item, and for each question the cited items it shows, as cited.
func (sh *shaping) citedEvidence(questions []map[string]any) (string, [][]string, error) {
	shown := make([][]string, len(questions))
	var blocks []string
	var tracked map[string]bool
	for i, q := range questions {
		for _, citation := range evidenceCitations(textOf(q["question"])) {
			for _, raw := range splitCitation(citation) {
				it := parseEvidenceItem(raw)
				if it.empty {
					continue
				}
				cited := strings.TrimSpace(raw)
				head := fmt.Sprintf("Question %d cites ", i+1)
				if it.command != "" {
					switch text, ok := sh.probeTexts[it.command]; {
					case sh.ext != nil:
						blocks = append(blocks, head+cited+": not shown (Kogen keeps no output of the external session's commands)")
					case !ok:
						blocks = append(blocks, head+cited+": not shown (no probe of this shaping ran `"+it.command+"`)")
					default:
						blocks = append(blocks, head+cited+" (probe output):\n"+strings.TrimRight(text, "\n"))
						shown[i] = append(shown[i], cited)
					}
					continue
				}
				if tracked == nil {
					out, err := sh.repo.OK("ls-tree", "-r", "-z", "--name-only", "HEAD")
					if err != nil {
						return "", nil, err
					}
					tracked = map[string]bool{}
					for _, f := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
						tracked[f] = true
					}
				}
				p := cleanCitedPath(it.path)
				switch {
				case strings.HasSuffix(it.path, "/") || !tracked[p]:
					blocks = append(blocks, head+cited+": not shown ("+it.path+" is not a file tracked at HEAD)")
				case sh.ext != nil && !sh.inspected:
					blocks = append(blocks, head+cited+": not shown (this shaping's session read nothing, so "+it.path+" was not read)")
				case sh.ext != nil:
					data, rerr := os.ReadFile(filepath.Join(sh.root, filepath.FromSlash(p)))
					if rerr != nil {
						out, err := sh.repo.OK("show", "HEAD:"+p)
						if err != nil {
							return "", nil, err
						}
						data = []byte(out)
					}
					blocks = append(blocks, head+it.path+" (read from the checkout):\n"+strings.TrimRight(numberedLines(string(data)), "\n"))
					shown[i] = append(shown[i], cited)
				case !sh.readPaths[p]:
					blocks = append(blocks, head+cited+": not shown (this shaping did not read "+it.path+")")
				default:
					blocks = append(blocks, head+it.path+" (read by this shaping):\n"+strings.TrimRight(sh.readTexts[p], "\n"))
					shown[i] = append(shown[i], cited)
				}
			}
		}
	}
	return strings.Join(blocks, "\n\n"), shown, nil
}

// numberedLines is text numbered as read_file numbers it: `<n>\t<line>`, a final newline ending
// the last line.
func numberedLines(text string) string {
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, l := range lines {
		lines[i] = fmt.Sprintf("%d\t%s", i+1, l)
	}
	return strings.Join(lines, "\n")
}

// classCount is a question class and the number of questions refused in it.
type classCount struct {
	class string
	n     int
}

// refusalCountsLines is the last error line of spent retries: the classes of the refused
// questions with their counts, nil when no question was refused.
func (sh *shaping) refusalCountsLines() []string {
	if len(sh.questionRefusals) == 0 {
		return nil
	}
	parts := make([]string, len(sh.questionRefusals))
	for i, c := range sh.questionRefusals {
		parts[i] = fmt.Sprintf("%s x%d", c.class, c.n)
	}
	return []string{"question check refusals: " + strings.Join(parts, ", ")}
}

func (sh *shaping) countRefusal(class string) {
	for i := range sh.questionRefusals {
		if sh.questionRefusals[i].class == class {
			sh.questionRefusals[i].n++
			return
		}
	}
	sh.questionRefusals = append(sh.questionRefusals, classCount{class, 1})
}

// questionGate is the question check of a questions submission. refused holds the question
// lines of a refusal (nil when every question may reach the user), with the feedback text; the
// second evidenced environment-only submission of the run is the environment fault instead.
func (sh *shaping) questionGate(ctx context.Context, questions []map[string]any) (text string, refused []string, err error) {
	items, shown, err := sh.questionClassify(ctx, questions)
	if err != nil {
		return "", nil, err
	}
	events := make([]any, len(items))
	asked, evidenced := 0, true
	for i, it := range items {
		var settled recordedDecision
		if it.class == "decided" {
			var ok bool
			if settled, ok = sh.ledger.standingByID(it.decision); !ok {
				it.class = "technical"
			}
		}
		event := map[string]any{"number": it.number, "question": textOf(questions[i]["question"]), "class": it.class, "why": it.why}
		if it.class == "decided" {
			event["decision"] = settled.ID
		}
		events[i] = event
		if it.class == "ui_ux" || it.class == "handoff" {
			asked++
			evidenced = false
			continue
		}
		instruction := it.class
		if it.class == "environment_prerequisite" && len(shown[i]) == 0 {
			instruction = "environment"
		}
		if instruction != "environment_prerequisite" {
			evidenced = false
		}
		sh.countRefusal(it.class)
		guidance := questionInstruction(instruction)
		if it.class == "decided" {
			guidance = fmt.Sprintf("%s (round %d) settles it: %s; do not ask it again. Complete the choice from %s, its constraints and its evidence, and keep it in the Intent's Decisions as: %s",
				settled.ID, settled.Round, settled.Decision, settled.ID, settled.intentLine())
		}
		refused = append(refused, fmt.Sprintf("- question %d (%s): %s | why: %s | %s", it.number, it.class, textOf(questions[i]["question"]), it.why, guidance))
	}
	sh.ev.Record(map[string]any{"event": "question_check", "items": events, "asked": asked, "refused": len(items) - asked})
	if refused == nil {
		return "", nil, nil
	}
	sh.refused("question_check", refused)
	if evidenced {
		sh.evidencedRefusals++
		if sh.evidencedRefusals >= 2 {
			e := sys.Fail("environment", "shape_environment", "shaping cannot go on without something only your environment can provide; nothing was asked or published", 3)
			for i, q := range questions {
				e.Lines = append(e.Lines, "- "+textOf(q["question"]), "  blocks: "+items[i].why, "  evidence: "+strings.Join(shown[i], "; "))
			}
			return "", nil, e
		}
	}
	return strings.Join(append([]string{fmt.Sprintf(questionHeadline, len(refused), len(items))}, refused...), "\n"), refused, nil
}
