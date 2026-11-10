package shape

import (
	"fmt"
	"strings"
)

const shapeRequestCap = 300

func shapeAnswerLimit(sinceSubmit int) bool {
	return sinceSubmit >= noToolLimit
}

func shapeRequestStop(last lastSubmission, answer string, answers int) outcome {
	result := noToolStop(last, answer)
	result.detail = fmt.Sprintf("the shaper reached the provider request cap of %d; shaping stopped", shapeRequestCap)
	if answers > 0 {
		first, _, _ := strings.Cut(answer, "\n")
		result.detail += "; last tool-free answer: " + first
	} else {
		result.problems = result.problems[:len(result.problems)-1]
	}
	return result
}

func shapeTurnCounts(turns *turnCounter, ev *Evidence, requests *int) {
	ev.Record(map[string]any{
		"event": "shaping_turn_counts", "owner": turns.owner, "no_tool_turns": turns.noTool,
		"invalid_submissions": turns.invalid, "request_cap": shapeRequestCap, "requests": *requests,
	})
}
