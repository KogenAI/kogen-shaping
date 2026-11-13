package shape

import (
	"strings"
	"testing"
)

// A checker item can name one sentence of a longer Decisions line. When the request is silent
// on it, the whole line takes the assumption marker instead of the Intent being refused.
func TestDecisionFragmentOfOneLineIsMarkedNotRefused(t *testing.T) {
	parallel(t)
	const request = "Add an export. Admins start it from settings. Reports are kept for a week."
	const long = "- Starting an export queues a job. The job reports 202 when queued. Admins only. (source: request)"
	const other = "- Reports are kept for a week. (source: request)"
	const twin = "- Retries are not supported. The job reports 202 when queued. (source: request)"
	intent := "## Decisions\n" + long + "\n" + other + "\n"
	frag := decisionItem{statement: "The job reports 202 when queued.", quotes: []string{}, verdict: "unsupported", why: "status code not in the request"}

	_, lines, marks := decisionResolve([]decisionItem{frag}, request, intent, nil)
	if len(lines) != 0 {
		t.Fatalf("a silent fragment of one Decisions line was refused: %v", lines)
	}
	if len(marks) != 1 || marks[0].line != long || !strings.Contains(marks[0].marked, "(assumption: status code not in the request) (source: request)") {
		t.Fatalf("the whole line should carry the assumption: %+v", marks)
	}

	// The same sentence in two lines does not identify one line: still refused.
	_, lines, marks = decisionResolve([]decisionItem{frag}, request, intent+twin+"\n", nil)
	if len(lines) == 0 || len(marks) != 0 {
		t.Fatalf("an ambiguous fragment must stay refused: lines=%v marks=%+v", lines, marks)
	}
	// ... and so does a sentence that is not in the Decisions section at all.
	_, lines, marks = decisionResolve([]decisionItem{frag}, request, "## Decisions\n"+other+"\n\n## Notes\nThe job reports 202 when queued.\n", nil)
	if len(lines) == 0 || len(marks) != 0 {
		t.Fatalf("a statement outside Decisions must stay refused: lines=%v marks=%+v", lines, marks)
	}
	// A fragment the request speaks to is not an assumption's business: unchanged refusal.
	spoken := frag
	spoken.quotes = []string{"Admins start it from settings."}
	_, lines, marks = decisionResolve([]decisionItem{spoken}, request, intent, nil)
	if len(lines) == 0 || len(marks) != 0 {
		t.Fatalf("a fragment the request speaks to must stay refused: lines=%v marks=%+v", lines, marks)
	}
}
