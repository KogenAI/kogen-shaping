package shape

import (
	"fmt"
	"strings"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// shapingBudget is the default usual-guidance target of one shaping. It is a target, not a stop.
const shapingBudget = 15 * time.Minute

// upperEndMinutes is how many minutes past the usual target the upper end of the performance target lies.
const upperEndMinutes = 5

// budgetEnv names the environment variable that sets the usual-guidance target of one shaping in minutes.
const budgetEnv = "KOGEN_SHAPE_BUDGET_MINUTES"

const (
	minBudgetMinutes = 5
	maxBudgetMinutes = 45
)

// budgetFromEnv is the usual-guidance target of one shaping: the default when the variable is unset or empty,
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

// budgetLine tells the shaper the elapsed time of the shaping and the time target.
// It states the default target; budgetLineFor states the target of one shaping.
func budgetLine(elapsed time.Duration) string { return budgetLineFor(shapingBudget, elapsed) }

// budgetLineFor is budgetLine for a shaping whose usual-guidance target is budget. The target is
// advice: crossing it never refuses a request.
func budgetLineFor(budget, elapsed time.Duration) string {
	minutes := int(budget / time.Minute)
	return fmt.Sprintf("Time: %s elapsed. Target: %d minutes usual, %d minutes upper end.",
		budgetText(max(elapsed, 0)), minutes, minutes+upperEndMinutes)
}

// externalBrief is the session note of an external Shape or Refine session: what the session
// has instead of the native tools, and how to use the time target.
// It states the default target; externalBriefFor states the target of one shaping.
func externalBrief() string { return externalBriefFor(shapingBudget) }

// externalBriefFor is externalBrief for a shaping with the given budget.
func externalBriefFor(budget time.Duration) string {
	minutes := int(budget / time.Minute)
	guidance := strings.ReplaceAll(shaperPromptFile("external-guidance"), "{{budget}}", fmt.Sprint(minutes))
	guidance = strings.ReplaceAll(guidance, "{{upper}}", fmt.Sprint(minutes+upperEndMinutes))
	return externalNote + "\n\n" + strings.TrimSuffix(guidance, "\n")
}

// shapingBudgetSeconds is the shaping budget in seconds, the figure of the shaping_turn_counts event.
func shapingBudgetSeconds(budget time.Duration) int { return int(budget / time.Second) }
