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

func TestFileLineCitationUsesObservedBaseline(t *testing.T) {
	parallel(t)
	f := newShpFx(t)
	f.write("limits.go", "package limits\nconst maxFiles = 8\n")
	f.git("add", "limits.go")
	f.git("commit", "-qm", "Add baseline limit")
	f.write("limits.go", "package limits\nconst maxFiles = 9\n")
	f.newSession()
	sh := f.shaping()
	sh.readPaths = map[string]bool{"limits.go": true}
	for _, citation := range []string{"limits.go:2", "`limits.go:2`", "limits.go"} {
		intent := "## Decisions\n- Keep maxFiles at 8. (evidence: " + citation + ") (source: request)\n"
		problems, err := sh.evidenceProblems(intent)
		if err != nil || len(problems) != 0 {
			t.Fatalf("%s: problems %v, error %v", citation, problems, err)
		}
		evidence, err := sh.decisionEvidence(intent)
		if err != nil {
			t.Fatal(err)
		}
		shpContains(t, "baseline", evidence, "const maxFiles = 8")
		if strings.Contains(evidence, "const maxFiles = 9") {
			t.Fatal("checkout override supplied as evidence")
		}
	}
	sh.readPaths = nil
	intent := "## Decisions\n- Keep maxFiles at 8. (evidence: limits.go:2) (source: request)\n"
	problems, err := sh.evidenceProblems(intent)
	if err != nil || len(problems) == 0 {
		t.Fatalf("unobserved file: problems %v, error %v", problems, err)
	}
	evidence, err := sh.decisionEvidence(intent)
	if err != nil {
		t.Fatal(err)
	}
	shpContains(t, "unobserved file", evidence, "not observed")
}

func TestEvidenceItemLineSuffix(t *testing.T) {
	for _, tc := range []struct{ citation, path string }{
		{"limits.go:1", "limits.go"}, {"limits.go:002", "limits.go"},
		{"limits.go:0", "limits.go:0"}, {"limits.go:00", "limits.go:00"},
		{"limits.go:", "limits.go:"}, {"limits.go:-1", "limits.go:-1"},
		{"limits.go:2x", "limits.go:2x"},
	} {
		if got := parseEvidenceItem(tc.citation).path; got != tc.path {
			t.Errorf("%q: got %q, want %q", tc.citation, got, tc.path)
		}
	}
}

// Scripted verdicts exercise gate handling and evidence transport, not model reasoning.
func TestDecisionCheckEvidencedAndUnsupported(t *testing.T) {
	for _, tc := range []struct {
		name, statement, citation, evidence, verdict string
		probe                                        bool
	}{
		{"unchanged file", "Keep maxFiles at 8.", "limits.go:2", "const maxFiles = 8", "follows", false},
		{"unchanged behavior", "Keep helpers in features/steps.", "ran `observe helpers`", "helpers directory: features/steps", "follows", true},
		{"new choice", "Increase maxFiles to 9.", "", "", "unsupported", false},
		{"wrong value", "Keep maxFiles at 9.", "limits.go:2", "const maxFiles = 8", "unsupported", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			line := "- " + tc.statement
			if tc.citation != "" {
				line += " (evidence: " + tc.citation + ")"
			}
			line += " (source: request)"
			f := newShpFx(t)
			f.Prov.route = func(req map[string]any) *shpStep {
				if !decisionRequest(req) {
					return nil
				}
				answer := shpAnswer(shpCall("d", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": line, "request_quotes": []any{}, "verdict": tc.verdict, "intent_value": "", "request_value": "", "difference": "", "why": "scripted result"}}}))
				return &answer
			}
			f.write("limits.go", "package limits\nconst maxFiles = 8\n")
			f.git("add", "limits.go")
			f.git("commit", "-qm", "Add baseline")
			f.newSession()
			sh := f.shaping()
			sh.role = sh.cfg.Models["shape"]
			sh.request = "Improve controls."
			sh.readPaths = map[string]bool{"limits.go": true}
			if tc.probe {
				sh.probed = map[string]bool{"observe helpers": true}
				sh.probeTexts = map[string]string{"observe helpers": "helpers directory: features/steps\n"}
			}
			intent := "## Decisions\n" + line + "\n"
			problems, err := sh.evidenceProblems(intent)
			if err != nil || len(problems) != 0 {
				t.Fatalf("evidence: %v, %v", problems, err)
			}
			feedback, err := sh.decisionCheck(context.Background(), intent)
			if err != nil {
				t.Fatal(err)
			}
			if (len(feedback) == 0) != (tc.verdict == "follows") {
				t.Fatalf("feedback: %v", feedback)
			}
			if tc.verdict == "unsupported" && !strings.Contains(strings.Join(feedback, "\n"), "verdict: unsupported") {
				t.Fatalf("feedback: %v", feedback)
			}
			requests := f.Prov.decisionRequests()
			if len(requests) != 1 {
				t.Fatalf("requests: %d", len(requests))
			}
			input := qcText(requests[0])
			if tc.evidence != "" {
				shpContains(t, "delivered evidence", input, tc.evidence)
			}
			instructions, _ := requests[0]["instructions"].(string)
			for _, policy := range []string{"if it shows the stated current value, give follows without requiring a request quote.", "A new choice with neither evidence nor a recorded assumption remains unsupported.", "A citation name alone is not proof.", "Give unsupported when the observed evidence does not establish the claimed value", "Explicit request conflicts still contradict"} {
				shpContains(t, "checker policy", instructions, policy)
			}
			found := false
			for _, event := range f.events() {
				if event["event"] == "decision_check" {
					found = true
					if event[tc.verdict] != float64(1) {
						t.Fatalf("event: %v", event)
					}
				}
			}
			if !found {
				t.Fatal("missing decision_check event")
			}
		})
	}
}
