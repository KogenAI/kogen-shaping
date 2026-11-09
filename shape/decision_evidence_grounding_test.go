package shape

import (
	"context"
	"strings"
	"testing"
)

// A citation is not a waiver when the checker says the value is unsupported.
func TestUnsupportedEvidenceValueIsStillRefused(t *testing.T) {
	parallel(t)
	line := "- Keep maxFiles at 9. (evidence: limits.go) (source: request)"
	_, problems := decisionOutcome([]decisionItem{{statement: line, verdict: "unsupported", why: "limits.go defines 8, not 9"}}, "Improve the controls.", "## Decisions\n"+line+"\n")
	if len(problems) == 0 {
		t.Fatal("an evidence marker hid an unsupported value")
	}
}

func TestDecisionCheckerReceivesObservedCitationBytes(t *testing.T) {
	parallel(t)
	line := "- Keep maxFiles at 8. (evidence: limits.go) (source: request)"
	f := newShpFx(t, shpAnswer(shpCall("d", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": line, "request_quotes": []any{}, "verdict": "follows", "intent_value": "", "request_value": "", "difference": "", "why": "observed current value"}}})))
	f.write("limits.go", "package limits\nconst maxFiles = 8\n")
	f.git("add", "limits.go")
	f.git("commit", "-qm", "Add the observed limit")
	f.newSession()
	sh := f.shaping()
	sh.role = sh.cfg.Models["shape"]
	sh.request = "Improve the controls."
	sh.readPaths = map[string]bool{"limits.go": true}
	sh.readTexts = map[string]string{"limits.go": numberedLines(f.read("limits.go"))}
	if _, err := sh.decisionCheck(context.Background(), "## Decisions\n"+line+"\n"); err != nil {
		t.Fatal(err)
	}
	requests := f.Prov.decisionRequests()
	if len(requests) != 1 {
		t.Fatalf("requests %d", len(requests))
	}
	if !strings.Contains(qcText(requests[0]), "const maxFiles = 8") {
		t.Fatal("checker received the citation name without the observed value")
	}
}

func TestDecisionEvidenceDoesNotReadUnobservedPaths(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.newSession()
	sh := f.shaping()
	text, err := sh.decisionEvidence("## Decisions\n- Preserve the limit. (evidence: calc.js) (source: request)\n")
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "unobserved path", text, "not observed by this shaping")
	if strings.Contains(text, "exports.") {
		t.Fatal("unobserved source body was provided")
	}
}

func TestDecisionEvidenceUsesBaselineNotCheckoutOverrides(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.write("limits.go", "package limits\nconst maxFiles = 8\n")
	f.git("add", "limits.go")
	f.git("commit", "-qm", "Add the baseline limit")
	f.write("limits.go", "package limits\nconst maxFiles = 9\n")
	f.newSession()
	sh := f.shaping()
	sh.readPaths = map[string]bool{"limits.go": true}
	text, err := sh.decisionEvidence("## Decisions\n- Keep the current limit. (evidence: limits.go; limits.go) (source: request)\n")
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "baseline source", text, "const maxFiles = 8")
	if strings.Contains(text, "const maxFiles = 9") {
		t.Fatal("uncommitted candidate overrode baseline evidence")
	}
	shpEq(t, "deduplicated citation", strings.Count(text, "const maxFiles = 8"), 1)
}

func TestDecisionEvidenceRecognizesListedDirectorySpellings(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.write("lib/retained.js", "exports.limit = 8\n")
	f.git("add", "lib/retained.js")
	f.git("commit", "-qm", "Add the retained directory")
	f.newSession()
	sh := f.shaping()
	sh.noteReach("list_files", map[string]any{"path": "lib/"})
	text, err := sh.decisionEvidence("## Decisions\n- Keep helpers under lib. (evidence: lib) (source: request)\n")
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "observed baseline directory", text, "retained.js")
}

func TestDocsDirectoryEvidenceReachesDecisionChecker(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.write("docs/presets/guide.md", "Tracked preset configuration.\n")
	f.write("docs/README.md", "The repository documentation.\n")
	f.git("add", "docs")
	f.git("commit", "-qm", "Add tracked documentation and presets")
	f.newSession()
	sh := f.shaping()
	sh.listedPaths = map[string]bool{"docs": true}

	for _, citation := range []string{"docs/presets", "docs/presets/"} {
		intent := "## Decisions\n- Preserve the preset inventory. (evidence: " + citation + ") (source: request)\n"
		problems, err := sh.evidenceProblems(intent)
		if err != nil || len(problems) != 0 {
			t.Errorf("tracked, listed directory %q: problems %v, error %v", citation, problems, err)
		}
		evidence, err := sh.decisionEvidence(intent)
		if err != nil {
			t.Fatal(err)
		}
		shpContains(t, "tracked directory contents", evidence, "guide.md")
	}

	intent := "## Decisions\n- Preserve the readme. (evidence: docs/README.md) (source: request)\n"
	problems, err := sh.evidenceProblems(intent)
	if err != nil {
		t.Fatal(err)
	}
	shpEq(t, "listing did not read the file", strings.Join(problems, "\n"), "- decision 1: this shaping did not read docs/README.md")
	evidence, err := sh.decisionEvidence(intent)
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "no inherited file read", evidence, "not observed")
	if strings.Contains(evidence, "The repository documentation.") {
		t.Fatal("directory listing supplied an unread file's contents")
	}

	_, _, diagnostic, err := sh.decisionReceipt(1, "docs/presets")
	if err != nil {
		t.Fatal(err)
	}
	shpEq(t, "sourced decision still needs a file", diagnostic,
		"- sourced decision 1: docs/presets is a directory; cite the files this shaping read")

	sh.listedPaths = nil
	intent = "## Decisions\n- Preserve the preset inventory. (evidence: docs/presets) (source: request)\n"
	problems, err = sh.evidenceProblems(intent)
	if err != nil {
		t.Fatal(err)
	}
	shpEq(t, "directory still needs observation", strings.Join(problems, "\n"), "- decision 1: this shaping did not list docs/presets")
}
