package shape

import (
	"fmt"
	"strings"

	"github.com/KogenAI/kogen-verification/verify"
)

// The preparation gate: a failed Given or Background step, or a failed Before hook, on the
// unchanged tree is a defect of the acceptance package, not missing-feature red. The gate judges
// the readiness run's own result: it adds no run and knows nothing of the repository.

// preparationSignature is the failure signature of a refused preparation failure, with the
// attempt that refused it.
type preparationSignature struct {
	signature string
	attempt   int
}

// preparationError is the refusal of a candidate whose acceptance fails in its preparation.
type preparationError struct {
	scenarios int
	items     []string
}

func (e *preparationError) Error() string { return strings.SplitN(e.text(), "\n", 2)[0] }

func (e *preparationError) text() string {
	return fmt.Sprintf("Acceptance preparation failed on the unchanged tree: %d scenario(s) fail before they reach the behaviour under test, so this is not missing-feature red; nothing was published. "+
		"A failed Given or Background step or Before hook, or a When or Then step that fails exactly as a refused preparation did, is a defect of the acceptance package: "+
		"repair the preparation so that it succeeds on the unchanged tree and the consumer of its fixture accepts it. "+
		"Keep the scenario and its outcome assertions, and do not change production code to accept a malformed fixture. "+
		"Resubmitting or acknowledging this package unchanged is refused again.", e.scenarios) +
		"\n" + strings.Join(e.items, "\n")
}

// preparationFailure is one scenario judged a preparation failure.
type preparationFailure struct {
	items     []string // the scenario's item lines
	signature string   // the failure signature, "" when empty
}

// failureSignature is the failure detail without its first line (the location), redacted, without
// stack frames and without trailing whitespace.
func failureSignature(detail string, redactor *verify.Redactor) string {
	_, rest, _ := strings.Cut(detail, "\n")
	return strings.TrimRight(withoutFrames(redactor.Redact(rest)), " \t\r\n\f\v")
}

// preparationFailures judges every failed scenario of a ready result; known are the signatures of
// the preparation failures refused earlier in this run.
func preparationFailures(ready *verify.Ready, known []preparationSignature, redactor *verify.Redactor) []preparationFailure {
	var out []preparationFailure
	for _, sc := range ready.Scenarios {
		if sc.Status != "failed" || sc.Detail == nil {
			continue
		}
		where, _, _ := strings.Cut(*sc.Detail, "\n")
		sig := failureSignature(*sc.Detail, redactor)
		why := preparationReason(sc, where, sig, known)
		if why == "" {
			continue
		}
		out = append(out, preparationFailure{items: preparationItem(sc, where, why, ready.AcceptanceExit, redactor), signature: sig})
	}
	return out
}

// preparationReason is the reason a failed scenario is a preparation failure, "" when it is not.
func preparationReason(sc verify.ScenarioResult, where, sig string, known []preparationSignature) string {
	if strings.HasPrefix(where, "a hook") {
		for _, st := range sc.Steps {
			if !hasStepLine(sc.UnexercisedSteps, st.Line) {
				return ""
			}
		}
		return "a Before hook"
	}
	m := failureLocation.FindStringSubmatch(where)
	if m == nil {
		return ""
	}
	var line int
	fmt.Sscan(m[1], &line)
	failed := -1
	for i, st := range sc.Steps {
		if st.Line == line {
			failed = i
			break
		}
	}
	if failed < 0 {
		return ""
	}
	own := sc.Line
	if sc.OutlineLine != nil {
		own = *sc.OutlineLine
	}
	if line < own {
		return "a Background step"
	}
	switch strings.TrimSpace(sc.Steps[failed].Keyword) {
	case "Given":
		return "a Given step"
	case "And", "But", "*":
		for i := failed - 1; i >= 0; i-- {
			switch strings.TrimSpace(sc.Steps[i].Keyword) {
			case "Given":
				return "a Given step"
			case "When", "Then":
				return earlierPreparation(sig, known)
			}
		}
		return earlierPreparation(sig, known)
	}
	return earlierPreparation(sig, known)
}

// earlierPreparation is the reason of a When or Then failure whose signature equals one refused
// earlier, naming the earliest such attempt.
func earlierPreparation(sig string, known []preparationSignature) string {
	if sig == "" {
		return ""
	}
	attempt := 0
	for _, k := range known {
		if k.signature == sig && (attempt == 0 || k.attempt < attempt) {
			attempt = k.attempt
		}
	}
	if attempt == 0 {
		return ""
	}
	return fmt.Sprintf("the same failure as the preparation refused at attempt %d", attempt)
}

func hasStepLine(steps []verify.StepRecord, line int) bool {
	for _, st := range steps {
		if st.Line == line {
			return true
		}
	}
	return false
}

// preparationItem is the item of one scenario: its header, the bounded output and the steps that
// did not run.
func preparationItem(sc verify.ScenarioResult, where, why string, exit int, redactor *verify.Redactor) []string {
	items := []string{fmt.Sprintf("- %s (acceptance.feature line %d): %s failed (%s); acceptance exit %d", sc.Name, sc.Line, where, why, exit)}
	_, rest, _ := strings.Cut(*sc.Detail, "\n")
	for _, l := range strings.Split(verify.Diagnostic(redactor.Redact(rest)), "\n") {
		if l == "" {
			items = append(items, "")
		} else {
			items = append(items, "  "+l)
		}
	}
	for _, st := range sc.UnexercisedSteps {
		items = append(items, fmt.Sprintf("  not run: acceptance.feature line %d: %s %s", st.Line, strings.TrimSpace(st.Keyword), st.Text))
	}
	return items
}

// preparationGate refuses a ready result in which a scenario fails in its preparation, and
// remembers the failures it refused for the rest of this Shape run.
func (sh *shaping) preparationGate(ready *verify.Ready) error {
	redactor := verify.NewRedactor(sh.s.Env, sh.s.Getenv("HOME"))
	failures := preparationFailures(ready, sh.preparationRefused, redactor)
	if len(failures) == 0 {
		return nil
	}
	e := &preparationError{scenarios: len(failures)}
	for _, f := range failures {
		e.items = append(e.items, f.items...)
	}
	for _, f := range failures {
		if f.signature != "" {
			sh.preparationRefused = append(sh.preparationRefused, preparationSignature{signature: f.signature, attempt: sh.submissions})
		}
	}
	return e
}
