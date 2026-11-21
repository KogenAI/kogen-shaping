package shape

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCutProblemsKeepsKeptRefusalFixSentence(t *testing.T) {
	parallel(t)
	problem := "no wrong implementation names a scenario of it. Fix controls.json so a wrong implementation names this scenario."
	unit := "Do not add a live sponsor to tool exports or change rankings."
	line := func(request string) string {
		return controlsPropertyLine(propertyReport{property: "Sponsors stay out of exports", kind: "kept", request: request}, problem)
	}

	long := line(strings.TrimSpace(strings.Repeat(unit+" ", 8)))
	got := cutProblems([]string{long})
	if len(got) != 1 || utf8.RuneCountInString(got[0]) != 300 {
		t.Fatalf("got %q", got)
	}
	if !strings.HasSuffix(got[0], " ...  | problem: "+problem) || !strings.HasPrefix(got[0], "- Sponsors stay out of exports (kept) | request: Do not add a live sponsor") {
		t.Fatalf("problem and fix sentence not kept whole: %q", got[0])
	}

	short := line(unit)
	if got := cutProblems([]string{short}); len(got) != 1 || got[0] != short {
		t.Fatalf("short line changed: %q", got)
	}

	// A problem too long to leave room for request text keeps today's plain cut.
	huge := "- P (kept) | request: " + strings.Repeat("r", 50) + " | problem: " + strings.Repeat("p", 400)
	if got := cutProblems([]string{huge}); got[0] != huge[:300] {
		t.Fatalf("got %q", got[0])
	}
}
