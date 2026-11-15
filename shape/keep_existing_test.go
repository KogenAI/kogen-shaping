package shape

import (
	"context"
	"strings"
	"testing"
)

// A scripted checker that always answers unsupported: only the keep-existing rule can change the
// outcome.
func TestKeepExistingDecisions(t *testing.T) {
	const limits = "package limits\nconst controlsMaxRuns = 8\nconst controlsMaxFiles = 8\nconst controlsMaxBytes = 65536\nconst controlsTimeCap = 300\n"
	cases := []struct {
		name, line, quote string
		follows, advisory bool
	}{
		{"evidenced keep", "- controlsMaxRuns (8), controlsMaxFiles (8), controlsMaxBytes (65536) and controlsTimeCap (300) keep their values. (evidence: limits.go) (source: request)", "", true, false},
		{"backtick value", "- `controlsMaxBytes = 65536` is unchanged. (evidence: limits.go:4) (source: request)", "", true, false},
		{"missing path", "- controlsMaxRuns (8) keeps its value. (evidence: missing.go) (source: request)", "", false, false},
		{"value not shown", "- controlsMaxRuns (9) keeps its value. (evidence: limits.go) (source: request)", "", false, false},
		{"no values", "- The limits are kept. (evidence: limits.go) (source: request)", "", false, false},
		{"one of two paths missing", "- controlsMaxRuns (8) keeps its value. (evidence: limits.go; missing.go) (source: request)", "", false, false},
		{"new decision without evidence", "- Increase controlsMaxRuns to 9. (source: request)", "Improve controls", false, false},
		{"no keep word", "- controlsMaxRuns is 8. (evidence: limits.go) (source: request)", "Improve controls", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newShpFx(t)
			f.Prov.route = func(req map[string]any) *shpStep {
				if !decisionRequest(req) {
					return nil
				}
				quotes := []any{}
				if c.quote != "" {
					quotes = append(quotes, c.quote)
				}
				answer := shpAnswer(shpCall("d", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{"statement": c.line, "request_quotes": quotes, "verdict": "unsupported", "intent_value": "", "request_value": "", "difference": "", "why": "this adds scope the quoted request passage does not ask for"}}}))
				return &answer
			}
			f.write("limits.go", limits)
			f.git("add", "limits.go")
			f.git("commit", "-qm", "Seed limits")
			f.newSession()
			sh := f.shaping()
			sh.role = sh.cfg.Models["shape"]
			sh.request = "Improve controls; keep the 300 s cap."
			problems, err := sh.decisionCheck(context.Background(), "## Decisions\n"+c.line+"\n")
			if err != nil {
				t.Fatal(err)
			}
			var verdict any
			for _, e := range f.events() {
				if e["event"] == "decision_check" {
					verdict = e["items"].([]any)[0].(map[string]any)["verdict"]
				}
			}
			if c.follows {
				if len(problems) != 0 || verdict != "follows" || len(sh.notes) != 0 {
					t.Fatalf("want follows with no feedback or note, got %v %v %v", verdict, problems, sh.notes)
				}
				return
			}
			if c.advisory {
				if len(problems) != 0 || len(sh.notes) != 1 {
					t.Fatalf("want the existing advisory route, got %v %v", problems, sh.notes)
				}
				return
			}
			if verdict != "unsupported" || !strings.Contains(strings.Join(problems, "\n"), "verdict: unsupported") {
				t.Fatalf("want an unsupported refusal, got %v %v", verdict, problems)
			}
		})
	}
}

func TestKeepExistingClaim(t *testing.T) {
	paths, values, ok := keepExistingClaim("- Keep `a b` at 65536 and x86 and rate_9 (evidence: p.go, ran `keep 3`) (source: request)")
	if !ok || len(paths) != 1 || paths[0] != "p.go, ran `keep 3`" && paths[0] != "p.go" {
		t.Fatalf("paths %v ok %v", paths, ok)
	}
	if got := strings.Join(values, "|"); got != "a b|65536" {
		t.Fatalf("values %q", got)
	}
	if _, _, ok := keepExistingClaim("- Keep 8 (source: request)"); ok {
		t.Fatal("no cited path is not keep-existing")
	}
	if _, _, ok := keepExistingClaim("- Keep 8 (evidence: ran `go test`) (source: request)"); ok {
		t.Fatal("a command is not a cited path")
	}
	if _, _, ok := keepExistingClaim("- Use 8 (evidence: p.go; keep) (source: request)"); ok {
		t.Fatal("a keep word inside the marker does not count")
	}
}
