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
