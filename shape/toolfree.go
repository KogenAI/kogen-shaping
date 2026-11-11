package shape

import "time"

func shapeAnswerLimit(sinceSubmit int) bool {
	return sinceSubmit >= noToolLimit
}

func shapeTurnCounts(turns *turnCounter, ev *Evidence, requests *int, elapsed func() time.Duration) {
	ev.Record(map[string]any{
		"event": "shaping_turn_counts", "owner": turns.owner, "no_tool_turns": turns.noTool,
		"invalid_submissions": turns.invalid, "requests": *requests,
		"budget_seconds": int(shapingBudget / time.Second), "elapsed_seconds": int(elapsed() / time.Second),
	})
}
