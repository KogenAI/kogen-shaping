package shape

import (
	"fmt"
	"regexp"
)

// readinessRepeatLimit is how many times an external shaper may be handed the same readiness
// failures before shaping stops. Each return costs a round and a readiness run; when the
// failures have not changed, neither can change what the shaper has to act on.
const readinessRepeatLimit = 3

// readinessVolatile is what differs between two readings of the same failures: the package
// hash, HEAD and evidence file the observations were recorded for, and the scratch directory
// named in a failure's stack.
var readinessVolatile = regexp.MustCompile(`(?m)^(?:Checked baseline|HEAD|Evidence): .*\n?|kogen-readiness-[0-9a-f]+`)

// readinessFingerprint identifies readiness observations by their failures alone.
func readinessFingerprint(observations string) string {
	return ContentHash([]byte(readinessVolatile.ReplaceAllString(observations, "")))
}

// readinessRepeats counts the readiness observations an external shaping handed back, by
// failures.
type readinessRepeats map[string]int

// note counts one return of observations and reports how many times these failures have now
// been returned.
func (r readinessRepeats) note(observations string) int {
	fp := readinessFingerprint(observations)
	r[fp]++
	return r[fp]
}

// readinessRepeatNotice is appended to the feedback of the second return of identical failures.
func readinessRepeatNotice(count int) string {
	return fmt.Sprintf("\n\nThese readiness failures are identical to the ones already returned (%d times). Submitting the package again without changing "+
		"what they name returns them again, and returning them %d times stops shaping. Repair the named assertion or step, or acknowledge it "+
		"as legitimate missing-feature red by resubmitting the exact current artifact.", count, readinessRepeatLimit)
}

// readinessRepeatStop is the outcome of the shaper being handed identical readiness failures
// readinessRepeatLimit times: a repeated identical failure, named, instead of the round cap.
func readinessRepeatStop(attempt, count int, path string) outcome {
	return outcome{
		kind: outNoTool,
		detail: fmt.Sprintf("the same readiness failures were returned %d times without a change to what they name (latest: submission %d); shaping stopped before the round cap",
			count, attempt),
		problems: []string{fmt.Sprintf("last submission (attempt %d): readiness failures returned for acknowledgment, see %s", attempt, path)},
	}
}
