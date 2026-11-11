package shape

import (
	"context"
	"strings"
	"sync"
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

// postSubmitChecks runs the checks after a submission is written. The local ones (the step
// self-check and the static checks) come first, and a refusal by either skips the model checks.
// Otherwise the decision, literal and example checks run together and every refusing check
// reports its problems, in that order. done is whether the submission was refused.
func (sh *shaping) postSubmitChecks(ctx context.Context, intent, evidence, hash string) (text string, out outcome, done bool, err error) {
	sh.progress("kogen: checking step code")
	feedback, err := sh.stepSelfcheck(ctx)
	if err != nil {
		return "", outcome{}, true, err
	}
	if feedback != "" {
		problems := strings.Split(feedback, "\n")[1:]
		sh.refused("step_selfcheck", problems)
		return feedback, outcome{kind: outInvalid, problems: problems}, true, nil
	}
	if text, out, err := sh.staticChecks(ctx, hash); err != nil || out.kind == outInvalid {
		return text, out, true, err
	}
	refusals, err := runModelChecks(ctx,
		func(ctx context.Context) ([]string, error) {
			return sh.decisionCheckWithEvidence(ctx, intent, evidence)
		},
		sh.literalCheck, sh.exampleCheck)
	if err != nil {
		return "", outcome{}, true, err
	}
	if len(refusals) == 0 {
		return "", outcome{}, false, nil
	}
	var texts, problems []string
	for i, r := range refusals {
		if i == 0 {
			sh.refused(r.gate, r.problems())
		} else {
			sh.refusedAlso(r.gate, r.problems())
		}
		texts = append(texts, r.text())
		problems = append(problems, r.problems()...)
	}
	return strings.Join(texts, "\n\n"), outcome{kind: outInvalid, problems: problems}, true, nil
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
