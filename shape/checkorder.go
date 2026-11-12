package shape

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/KogenAI/kogen-coordinator/project"
)

// modelRefusal is the feedback lines of one model check that refused a submission.
type modelRefusal struct {
	gate  string
	lines []string // the check's headline, when it has one, then its problems
	head  bool     // whether lines starts with a headline
}

func (r modelRefusal) problems() []string {
	if r.head {
		return r.lines[1:]
	}
	return r.lines
}

func (r modelRefusal) text() string {
	if r.gate == "example_check" {
		return "Examples contradict the rules:\n" + strings.Join(r.lines, "\n")
	}
	return strings.Join(r.lines, "\n")
}

// cutProblems are the problem lines a refusal records: at most 20, each at most 300 characters.
func cutProblems(problems []string) []string {
	if len(problems) > 20 {
		problems = problems[:20]
	}
	cut := make([]string, len(problems))
	for i, p := range problems {
		if r := []rune(p); len(r) > 300 {
			p = string(r[:300])
		}
		cut[i] = p
	}
	return cut
}

// runModelChecks runs the decision, literal and example checks together and waits for all of
// them. It returns the refusals of the checks that refused, in that fixed order. When a check
// fails, the others are cancelled and the first failure is returned.
func runModelChecks(ctx context.Context, decision, literal, example func(context.Context) ([]string, error)) ([]modelRefusal, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	checks := []struct {
		gate string
		head bool
		run  func(context.Context) ([]string, error)
	}{{"decision_check", true, decision}, {"literal_check", true, literal}, {"example_check", false, example}}
	lines := make([][]string, len(checks))
	var (
		wg    sync.WaitGroup
		once  sync.Once
		first error
	)
	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if lines[i], err = c.run(ctx); err != nil {
				once.Do(func() { first = err; cancel() })
			}
		}()
	}
	wg.Wait()
	if first != nil {
		return nil, first
	}
	var refusals []modelRefusal
	for i, c := range checks {
		if len(lines[i]) > 0 {
			refusals = append(refusals, modelRefusal{gate: c.gate, lines: lines[i], head: c.head})
		}
	}
	return refusals, nil
}

// heldRefusal is one gate's refusal of a submission whose other independent gates still run: the
// gate, its bounded problem lines and the feedback text the gate gives on its own.
type heldRefusal struct {
	gate     string
	problems []string
	text     string
}

// holdText gives the refusal a gate just held its feedback text.
func (sh *shaping) holdText(text string) {
	if n := len(sh.batch); n > 0 {
		sh.batch[n-1].text = text
	}
}

// startBatch makes the refusals that follow wait for settleBatch.
func (sh *shaping) startBatch() {
	sh.batching, sh.batch = true, nil
}

// settleBatch ends the collection and, when a gate refused, records every refusal once and
// returns the one feedback that names each refusing gate's problems, grouped by gate.
func (sh *shaping) settleBatch() (string, outcome, bool) {
	held := sh.batch
	sh.batching, sh.batch = false, nil
	if h := sh.heldRecord; h != nil {
		sh.heldRecord = nil
		held = append([]heldRefusal{*h}, held...)
	}
	if len(held) == 0 {
		return "", outcome{}, false
	}
	sh.recordRefusals(held)
	return combinedRefusal(held)
}

func combinedRefusal(held []heldRefusal) (string, outcome, bool) {
	var texts, problems []string
	for _, h := range held {
		texts = append(texts, h.text)
		problems = append(problems, h.problems...)
	}
	return strings.Join(texts, "\n\n"), outcome{kind: outInvalid, problems: problems}, true
}

// reportHeldRecord adds a decision record refusal that no later gate reported to the refusal a
// submission got from a gate that does not collect.
func (sh *shaping) reportHeldRecord(text string, out outcome) (string, outcome) {
	h := sh.heldRecord
	sh.heldRecord = nil
	if h == nil || out.kind != outInvalid {
		return text, out
	}
	sh.refusedAlso(h.gate, h.problems)
	text, out, _ = combinedRefusal([]heldRefusal{*h, {problems: out.problems, text: text}})
	return text, out
}

// postSubmitChecks runs the checks after a submission is written: the step self-check, the static
// checks and the decision, literal and example checks. Each refusing check is held with its
// problems, in that order, for the combined refusal.
func (sh *shaping) postSubmitChecks(ctx context.Context, intent, evidence, hash string) error {
	sh.progress("kogen: checking step code")
	feedback, err := sh.stepSelfcheck(ctx)
	if err != nil {
		return err
	}
	if feedback != "" {
		sh.refused("step_selfcheck", strings.Split(feedback, "\n")[1:])
		sh.holdText(feedback)
	}
	if text, out, err := sh.staticChecks(ctx, hash); err != nil {
		return err
	} else if out.kind == outInvalid {
		sh.holdText(text)
	}
	// A package the review scope, the step self-check or the static checks refuse has to change
	// before a model can judge it, so those refusals still skip the model checks.
	if slices.ContainsFunc(sh.batch, func(h heldRefusal) bool {
		return h.gate == scopeGate || h.gate == "step_selfcheck" || h.gate == staticGate
	}) {
		return nil
	}
	refusals, err := runModelChecks(ctx,
		func(ctx context.Context) ([]string, error) {
			return sh.decisionCheckWithEvidence(ctx, intent, evidence)
		},
		sh.literalCheck, sh.exampleCheck)
	if err != nil {
		return err
	}
	for _, r := range refusals {
		sh.refused(r.gate, r.problems())
		sh.holdText(r.text())
	}
	return nil
}

// refusedAlso records a further refusal of the submission refused by sh.refused: its event and
// its place among the unresolved refusals, without counting the submission against the retry
// budget a second time.
func (sh *shaping) refusedAlso(gate string, problems []string) {
	cut := cutProblems(problems)
	sh.unresolved = append(sh.unresolved, shapeRefusal{attempt: sh.submissions, gate: gate, problems: cut})
	sh.ev.Record(map[string]any{"event": "submission_refused", "attempt": sh.submissions, "gate": gate, "problems": cut,
		"consecutive": sh.retry.consecutive, "total": sh.retry.total})
}

// applyAssumptionMarks writes the assumption markers the decision check found missing into
// intent.md and records the rewrite.
func (sh *shaping) applyAssumptionMarks() error {
	marks := sh.pendingMarks
	sh.pendingMarks = nil
	if len(marks) == 0 {
		return nil
	}
	path := filepath.Join(sh.root, project.IntentMDPath(sh.slug))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	intent := string(data)
	var done []any
	for _, m := range marks {
		if strings.Contains(intent, m.line) {
			intent = strings.ReplaceAll(intent, m.line, m.marked)
			done = append(done, map[string]any{"line": m.line, "marked": m.marked, "assumption": m.assumption})
		}
	}
	if len(done) == 0 {
		return nil
	}
	if err := os.WriteFile(path, []byte(intent), 0o666); err != nil {
		return err
	}
	sh.ev.Record(map[string]any{"event": "decision_marked", "lines": done})
	return nil
}
