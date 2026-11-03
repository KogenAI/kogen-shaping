package shape

// shpQuestionCheck is the question check's answer: one class for each question in order.
func shpQuestionCheck(id string, classes ...string) shpStep {
	items := []any{}
	for i, c := range classes {
		items = append(items, map[string]any{"number": i + 1, "class": c, "why": "a " + c + " question"})
	}
	return shpAnswer(shpCall(id, "submit_question_check", map[string]any{"items": items}))
}

// shpUI is the question check that lets n questions reach the user.
func shpUI(n int) shpStep {
	classes := make([]string, n)
	for i := range classes {
		classes[i] = "ui_ux"
	}
	return shpQuestionCheck("qc", classes...)
}

// shpShaperRequests are the requests that are not question checks.
func shpShaperRequests(reqs []map[string]any) []map[string]any {
	var out []map[string]any
	for _, r := range reqs {
		offered := false
		for _, n := range shpToolNames(r) {
			offered = offered || n == SubmitQuestionCheckTool
		}
		if !offered {
			out = append(out, r)
		}
	}
	return out
}
