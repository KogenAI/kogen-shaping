package shape

import (
	"fmt"
	"strings"
	"time"
)

// shapingBudget is the wall-clock time one shaping has, for every project.
const shapingBudget = 15 * time.Minute

// budgetText is a duration in whole minutes, with seconds when it has any.
func budgetText(d time.Duration) string {
	secs := int(d / time.Second)
	if secs%60 == 0 {
		return fmt.Sprintf("%d min", secs/60)
	}
	return fmt.Sprintf("%d min %d s", secs/60, secs%60)
}

// budgetLine tells the shaper how much of the budget is spent and how much is left.
func budgetLine(elapsed time.Duration) string {
	elapsed = max(elapsed, 0)
	return fmt.Sprintf("Time: %s elapsed, %s remaining of the %d-minute shaping budget.",
		budgetText(elapsed), budgetText(max(shapingBudget-elapsed, 0)), int(shapingBudget/time.Minute))
}

// externalBrief is the session note of an external Shape or Refine session: what the session
// has instead of the native tools, and how to spend the time budget.
func externalBrief() string {
	guidance := strings.ReplaceAll(shaperPromptFile("external-guidance"), "{{budget}}", fmt.Sprint(int(shapingBudget/time.Minute)))
	return externalNote + "\n\n" + strings.TrimSuffix(guidance, "\n")
}

// budgetStop is the outcome of a shaping whose budget ran out before its next provider request:
// it names the budget, the time spent, the requests made, and the last refusal if there was one.
func budgetStop(last lastSubmission, answer string, answers int, elapsed time.Duration, requests int) outcome {
	result := noToolStop(last, answer)
	result.detail = fmt.Sprintf("the shaper used its %d-minute shaping budget (%s elapsed, %d requests) without a submission that passed; shaping stopped",
		int(shapingBudget/time.Minute), budgetText(elapsed), requests)
	if answers > 0 {
		first, _, _ := strings.Cut(answer, "\n")
		result.detail += "; last tool-free answer: " + first
	} else {
		result.problems = result.problems[:len(result.problems)-1]
	}
	return result
}

// budgetSpent reports whether the budget leaves no time for another provider request.
func budgetSpent(elapsed time.Duration) bool { return elapsed >= shapingBudget }
