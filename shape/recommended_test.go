package shape

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecommendedQuestionBecomesAssumption(t *testing.T) {
	q := shpQuestion("How should non-integers be handled?", "Raise TypeError", "Return false")
	q["options"].([]any)[0].(map[string]any)["recommended"] = true
	line := "- Non-integers raise TypeError. (assumption: use Raise TypeError) (source: request)"
	f := newShpFx(t, shpSubmit("q", map[string]any{"questions": []any{q}}), shpUI(1), shpSubmit("files", shpFilesWithDecisions(line)))
	f.setReport("PASSED", 0)
	exit, err := f.shape("Add calc.is_even(n), which returns true for even integers.")
	if err != nil || exit != 0 {
		t.Fatalf("exit %d: %v", exit, err)
	}
	b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md"))
	if err != nil || !strings.Contains(string(b), line) || f.exists(".kogen/local/shaping/is-even/questions.json") {
		t.Fatalf("intent %s, err %v", b, err)
	}
	if strings.Contains(f.Stdout.String(), "How should non-integers be handled?") {
		t.Fatal("asked recommended question")
	}
}

func TestDecisionSettledSourcesAndConflict(t *testing.T) {
	for _, line := range []string{"- Use TypeError. (source: answer 1)", "- Use TypeError. (assumption: use TypeError) (source: request)"} {
		intent := "## Decisions\n" + line
		item := decisionItem{statement: line, verdict: "unsupported"}
		if _, problems := decisionOutcome([]decisionItem{item}, "Add parity", intent); len(problems) != 0 {
			t.Fatal(problems)
		}
		item = decisionItem{statement: line, verdict: "contradicts", quotes: []string{"Use false"}, intentValue: "TypeError", requestValue: "false", difference: "different"}
		if _, problems := decisionOutcome([]decisionItem{item}, "Use false", intent); len(problems) == 0 {
			t.Fatal("request conflict ignored")
		}
	}
}

func TestFailedSessionClearsQuestionsBeforeFreshSession(t *testing.T) {
	f := newShpFx(t)
	path := ".kogen/local/shaping/is-even/questions.json"
	second, seeded, observed, carried := false, false, false, false
	f.Prov.route = func(req map[string]any) *shpStep {
		if contractKind(req) != "" || decisionRequest(req) || literalRequest(req) {
			return nil
		}
		if !second && !seeded {
			f.write(path, "stale\n")
			seeded = true
		}
		if second && !observed {
			carried = f.exists(path)
			observed = true
		}
		step := shpAnswer()
		return &step
	}
	exit, err := f.shape("Add calc.is_even(n).")
	if err == nil && exit == 0 {
		t.Fatal("first session did not fail")
	}
	f.newSession()
	second = true
	_, _ = f.shape("Add calc.is_even(n).")
	if !observed || carried {
		t.Fatalf("second session observed=%v carried=%v", observed, carried)
	}
}
