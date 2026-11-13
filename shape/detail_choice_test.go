package shape

import (
	"strings"
	"testing"
)

const detailRequest = "Users can export a report. The report is hosted by a third party and customers get an email when it is done."

// A grounded-quote "unsupported" that only fills in a detail of something the request asks for
// is advisory; one that removes, replaces or drops existing things still refuses.
func TestDecisionGroundedDetailChoiceIsMarked(t *testing.T) {
	cases := []struct {
		name, statement, quote, why string
		refused                     bool
	}{
		{"status code", "a missing report answers `404`", "Users can export a report", "The request does not give a status for a missing report.", false},
		{"label", "the page shows a \"Download\" button", "customers get an email when it is done", "The request says nothing of a Download button.", false},
		{"param", "the export takes param `kind` of `full` or `short`", "Users can export a report", "The request does not specify the param name.", false},
		{"code span with a removal word", "event `report.deleted` marks the export stale", "Users can export a report", "The request is not covered by any Decision for this event.", false},
		{"removal", "The legacy `Report::Cache` mechanism is folded into the new store: nothing remains that reads it.", "Users can export a report", "The request does not ask to remove the cache mechanism.", true},
		{"replace", "The old exporter is replaced by the new job.", "Users can export a report", "Replacing the exporter is a new choice.", true},
	}
	for _, c := range cases {
		intent := "# Intent\n\n## Decisions\n\n- " + c.statement + " (source: request)\n"
		item := decisionItem{statement: c.statement, quotes: []string{c.quote}, verdict: "unsupported", why: c.why}
		_, lines, marks, _ := decisionResolveNotes([]decisionItem{item}, detailRequest, intent, nil)
		if got := len(lines) > 0; got != c.refused {
			t.Fatalf("%s: refused=%v want %v (lines %q marks %v)", c.name, got, c.refused, lines, marks)
		}
		if !c.refused && (len(marks) != 1 || !strings.Contains(marks[0].marked, "(assumption: ")) {
			t.Fatalf("%s: want one assumption mark, got %v", c.name, marks)
		}
		if c.refused && len(marks) != 0 {
			t.Fatalf("%s: refused statement was also marked", c.name)
		}
	}
}
