package shape

import (
	"fmt"
	"strings"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// shapingBudget is the default charged time one shaping has, excluding cold runtime installation and the wall time of timed-out static checks.
const shapingBudget = 15 * time.Minute

// budgetEnv names the environment variable that sets the budget of one shaping in minutes.
const budgetEnv = "KOGEN_SHAPE_BUDGET_MINUTES"

const (
	minBudgetMinutes = 5
	maxBudgetMinutes = 45
)

// budgetFromEnv is the budget of one shaping: the default when the variable is unset or empty,
// otherwise the whole number of minutes it holds, which must be from 5 to 45.
func budgetFromEnv(getenv func(string) string) (time.Duration, error) {
	v := getenv(budgetEnv)
	if v == "" {
		return shapingBudget, nil
	}
	minutes := 0
	ok := len(v) <= 3
	for _, c := range v {
		if c < '0' || c > '9' {
			ok = false
			break
		}
		if ok {
			minutes = minutes*10 + int(c-'0')
		}
	}
	if !ok || minutes < minBudgetMinutes || minutes > maxBudgetMinutes {
		return 0, sys.Fail("environment", "shape_budget", "kogen: shape budget must be a whole number of minutes from 5 to 45", 3)
	}
	return time.Duration(minutes) * time.Minute, nil
}

// budgetOr is budget, or the default budget when it is unset (zero).
func budgetOr(budget time.Duration) time.Duration {
	if budget <= 0 {
		return shapingBudget
	}
	return budget
}

// budgetText is a duration in whole minutes, with seconds when it has any.
func budgetText(d time.Duration) string {
	secs := int(d / time.Second)
	if secs%60 == 0 {
		return fmt.Sprintf("%d min", secs/60)
	}
	return fmt.Sprintf("%d min %d s", secs/60, secs%60)
}

// budgetLine tells the shaper how much of the budget is spent and how much is left.
// It states the default budget; budgetLineFor states the budget of one shaping.
func budgetLine(elapsed time.Duration) string { return budgetLineFor(shapingBudget, elapsed) }

// budgetLineFor is budgetLine for a shaping with the given budget.
func budgetLineFor(budget, elapsed time.Duration) string {
	elapsed = max(elapsed, 0)
	return fmt.Sprintf("Time: %s elapsed, %s remaining of the %d-minute shaping budget.",
		budgetText(elapsed), budgetText(max(budget-elapsed, 0)), int(budget/time.Minute))
}

// externalBrief is the session note of an external Shape or Refine session: what the session
// has instead of the native tools, and how to spend the time budget.
// It states the default budget; externalBriefFor states the budget of one shaping.
func externalBrief() string { return externalBriefFor(shapingBudget) }

// externalBriefFor is externalBrief for a shaping with the given budget.
func externalBriefFor(budget time.Duration) string {
	guidance := strings.ReplaceAll(shaperPromptFile("external-guidance"), "{{budget}}", fmt.Sprint(int(budget/time.Minute)))
	return externalNote + "\n\n" + strings.TrimSuffix(guidance, "\n")
}

// budgetStop is the outcome of a shaping whose budget ran out before its next provider request:
// it names the budget, the time spent, the requests made, and the last refusal if there was one.
func budgetStop(budget time.Duration, last lastSubmission, answer string, answers int, elapsed time.Duration, requests int, attempts []timedAttempt, install, uncharged time.Duration) outcome {
	result := noToolStop(last, answer)
	if last.attempt > 0 {
		result.problems = result.problems[1:]
	}
	result.detail = fmt.Sprintf("the shaper used its %d-minute shaping budget (%s elapsed, %d requests) without a submission that passed; shaping stopped",
		int(budget/time.Minute), budgetText(elapsed), requests)
	if answers > 0 {
		first, _, _ := strings.Cut(answer, "\n")
		result.detail += "; last tool-free answer: " + first
	} else {
		result.problems = result.problems[:len(result.problems)-1]
	}
	result.problems = append(result.problems, attemptLines(budget, attempts, install, uncharged)...)
	return result
}

// timedAttempt is one refused submission, or the last submission when it was checked without a refusal: the first refusing gate, the time of the model request
// that produced it, the time Kogen spent checking it and the shaping budget left when that check ended.
type timedAttempt struct {
	attempt   int
	gate      string
	model     time.Duration
	check     time.Duration
	remaining time.Duration
	over      time.Duration // how far past the budget the check ended; set only on the last unrefused submission
}

// wholeSeconds is a duration in whole seconds, truncated and never below 0.
func wholeSeconds(d time.Duration) int { return int(max(d, 0) / time.Second) }

// attemptLines state the shaping budget, as the shaping_turn_counts event records it, and
// one line per refused submission, oldest first.
func attemptLines(budget time.Duration, attempts []timedAttempt, install, uncharged time.Duration) []string {
	lines := []string{fmt.Sprintf("shaping budget: %d s (%d min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)",
		shapingBudgetSeconds(budget), int(budget/time.Minute))}
	suffix := ""
	if install > 0 {
		suffix = fmt.Sprintf(", install_ms=%d", install.Milliseconds())
	}
	if uncharged > 0 {
		suffix += fmt.Sprintf(", static_uncharged_ms=%d", uncharged.Milliseconds())
	}
	lines[0] += suffix
	for _, a := range attempts {
		lines = append(lines, fmt.Sprintf("attempt %d: gate %s, model request %d s, Kogen checks %d s, %d s remaining",
			a.attempt, a.gate, wholeSeconds(a.model), wholeSeconds(a.check), wholeSeconds(a.remaining))+overText(a.over)+suffix)
	}
	return lines
}

// overText is the suffix of an attempt line whose checks ended past the budget.
func overText(over time.Duration) string {
	if over <= 0 {
		return ""
	}
	return fmt.Sprintf(", %d s over", wholeSeconds(over))
}

// shapingBudgetSeconds is the shaping budget in seconds, the figure of the shaping_turn_counts event.
func shapingBudgetSeconds(budget time.Duration) int { return int(budget / time.Second) }

// budgetSpent reports whether the budget leaves no time for another provider request.
func budgetSpent(budget, elapsed time.Duration) bool { return elapsed >= budget }
