'use strict';
// The Go source of the shape-use-studio-rule-snapshot acceptance harness, which
// shape-use-studio-rule-snapshot__steps.js compiles into the shape package's tests through a Go overlay as
// shape/shape_use_studio_rule_snapshot_acceptance_test.go; that path never exists in the repository. The
// source holds no backquote and no dollar-brace, so this raw template keeps it byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of shape-use-studio-rule-snapshot. It drives the real Shape entry with the shape
// tests' scripted local provider (the shaper steps are scripted; the decision check, literal check and
// contract review use the provider's default answers, which match their contracts) and a local fake
// authenticated Studio endpoint with a synthetic token. SRS_CASE names one journey; what the runs
// showed is written as JSON to SRS_OUT for the acceptance steps to judge. The harness judges nothing.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const srsToken = "srs-synthetic-token"

type srsStudio struct {
	mu   sync.Mutex
	rev  int
	mode string // ok, 401, 403 or down
	srv  *httptest.Server
}

func newSrsStudio(t *testing.T) *srsStudio {
	s := &srsStudio{rev: 3, mode: "ok"}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch {
		case s.mode == "down":
			http.Error(w, "down", http.StatusServiceUnavailable)
		case r.Header.Get("Authorization") != "Bearer "+srsToken || s.mode == "401":
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		case s.mode == "403":
			http.Error(w, "forbidden", http.StatusForbidden)
		default:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"product": "kogen", "repo": "kogen-shaping", "rules": []any{map[string]any{
				"id": "rule-repo-1", "revision": s.rev, "scope": "repo", "topic": "tests",
				"text": fmt.Sprintf("SRS rule text revision %d", s.rev), "owner": "team", "overrides": nil}}})
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *srsStudio) set(rev int, mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rev, s.mode = rev, mode
}

func srsConfig(url, repo string) string {
	return "version: 1\nurl: " + url + "\nworkspace_id: optimum\nproduct_id: kogen\nrepo: " + repo + "\ntoken_env: SRS_STUDIO_TOKEN\non_unavailable: unavailable\n"
}

func TestShapeUseStudioRuleSnapshotAcceptance(t *testing.T) {
	name, out := os.Getenv("SRS_CASE"), os.Getenv("SRS_OUT")
	if out == "" {
		t.Fatal("SRS_OUT must be set")
	}
	// Journeys: each is its inputs and what changes between its runs.
	inputs := map[string][]string{
		"live": {"Request text\n"}, "standalone": {"Request text\n"}, "cold_down": {"Request text\n"},
		"changed": {"Request text\n", "1: a\n"}, "answer": {"Request text\n", "1: a\n"}, "outage": {"Request text\n", "1: a\n"},
		"denied": {"Request text\n", "1: a\n"}, "foreign": {"Request text\n", "1: a\n"}, "foreign_down": {"Request text\n", "1: a\n"},
	}
	runs, ok := inputs[name]
	if !ok {
		t.Fatalf("SRS_CASE must name a journey, not %q", name)
	}
	shaper := []shpStep{shpSubmit("s1", shpFiles())}
	if len(runs) == 2 {
		shaper = []shpStep{
			shpSubmit("s1", map[string]any{"questions": []any{shpQuestion("Show details inline or in a panel?", "In a panel", "Inline")}}),
			shpUI(1),
			shpSubmit("s2", shpFiles()),
		}
	}
	f := newShpFx(t, shaper...)
	studio := newSrsStudio(t)
	f.Env = append(f.Env, "SRS_STUDIO_TOKEN="+srsToken)
	f.setReport("PASSED", 0)
	if name == "cold_down" {
		studio.set(3, "down")
	}
	if name != "standalone" {
		f.write(".kogen/studio-rules.yaml", srsConfig(studio.srv.URL, "kogen-shaping"))
		f.git("add", "-A")
		f.git("commit", "-qm", "Configure Studio rules")
	}
	obs := map[string]any{"case": name}
	runs2 := []any{}
	for i, input := range runs {
		if i == 1 {
			switch name {
			case "changed":
				studio.set(4, "ok")
			case "outage":
				studio.set(4, "down")
			case "denied":
				studio.set(3, "401")
			case "foreign":
				studio.set(3, "403")
			case "foreign_down":
				studio.set(3, "down")
				f.write(".kogen/studio-rules.yaml", srsConfig(studio.srv.URL, "other-repo"))
				f.git("add", "-A")
				f.git("commit", "-qm", "Change the Studio scope")
			}
		}
		before := len(f.Prov.allRequests())
		exit, err := f.shape(input)
		e := ""
		if err != nil {
			e = err.Error()
		}
		shaperTotal, shaperCarry, contractTotal, contractCarry, leak := 0, 0, 0, 0, false
		texts := map[string]bool{}
		for _, r := range f.Prov.allRequests()[before:] {
			b, _ := json.Marshal(r)
			body := string(b)
			leak = leak || strings.Contains(body, srsToken)
			has := strings.Contains(body, "SRS rule text revision 3") || strings.Contains(body, "SRS rule text revision 4")
			switch {
			case contractKind(r) != "":
				contractTotal++
				if has {
					contractCarry++
				}
			case decisionRequest(r) || len(shpShaperRequests([]map[string]any{r})) == 0:
			default:
				shaperTotal++
				if has {
					shaperCarry++
				}
			}
			for _, rev := range []string{"3", "4"} {
				if strings.Contains(body, "SRS rule text revision "+rev) {
					texts[rev] = true
				}
			}
		}
		run := map[string]any{"exit": exit, "error": e, "shaper_requests": shaperTotal, "shaper_requests_with_rules": shaperCarry,
			"contract_requests": contractTotal, "contract_requests_with_rules": contractCarry, "token_in_requests": leak,
			"request_revisions": texts, "stderr_has_token": strings.Contains(f.Stderr.String(), srsToken)}
		runs2 = append(runs2, run)
	}
	obs["runs"] = runs2
	receipts, refused := []any{}, []any{}
	if b, err := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping/is-even/events.ndjson")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if line == "" || json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			switch ev["event"] {
			case "studio_rules":
				receipts = append(receipts, ev)
			case "studio_rules_refused":
				refused = append(refused, ev)
			}
			if strings.Contains(line, srsToken) {
				obs["token_in_events"] = true
			}
		}
	}
	obs["receipts"], obs["refusals"] = receipts, refused
	b, err := json.MarshalIndent(obs, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
`;
