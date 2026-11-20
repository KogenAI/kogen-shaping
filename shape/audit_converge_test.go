package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// The replay cases under testdata/replay are the six cells in which the contract review refused
// an invented literal again and again: each file is one cell's literals, or its Decision.

type replayLiteral struct {
	Kind    string `json:"kind"` // "step": driven by the step file; "intent": a sentence of intent.md
	Literal string `json:"literal"`
}

type replayCase struct {
	Cell      string          `json:"cell"`
	Request   string          `json:"request"`
	First     []replayLiteral `json:"first"`
	Second    []replayLiteral `json:"second"`
	Weak      string          `json:"weak"`
	WeakValue string          `json:"weak_value"`
	Full      string          `json:"full"`
}

func replayCases(t *testing.T) []replayCase {
	t.Helper()
	paths, err := filepath.Glob("testdata/replay/*.json")
	if err != nil || len(paths) != 6 {
		t.Fatalf("replay cases %v: %v", paths, err)
	}
	var out []replayCase
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var c replayCase
		if err := json.Unmarshal(b, &c); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out = append(out, c)
	}
	return out
}

// convergeFiles is the whole package with the literals driven by the step file or stated by a
// sentence of intent.md.
func convergeFiles(lits []replayLiteral, deferral bool) map[string]any {
	m := shpFiles()
	list := m["files"].([]any)
	for _, raw := range list {
		e := raw.(map[string]any)
		for _, l := range lits {
			switch {
			case l.Kind == "step" && e["path"] == ccStepsPath:
				e["content"] = e["content"].(string) + "\n// " + l.Literal + "\n"
			case l.Kind == "intent" && e["path"] == ccIntentPath:
				e["content"] = strings.Replace(e["content"].(string), "returns True for even integers.\n", "returns True for even integers. "+l.Literal+".\n", 1)
			}
		}
	}
	if deferral {
		m["files"] = append(list, map[string]any{"path": ".kogen/intents/is-even/controls.json", "content": "{\"deferred\":\"the correct implementation is too large to write here\"}"})
	}
	return m
}

func convergeReview(id string, lits []replayLiteral, deferral bool) shpStep {
	inv := []any{}
	for _, l := range lits {
		scenario := "Even numbers"
		if l.Kind == "intent" {
			scenario = ""
		}
		inv = append(inv, map[string]any{"scenario": scenario, "literal": l.Literal, "why": "no source states it"})
	}
	args := map[string]any{"findings": []any{}, "environment": []any{}, "invented": inv}
	if deferral {
		args["guarantees"] = []any{map[string]any{"property": "Delivery", "request": "exactly once", "needs_interrupted": false}}
		args["failure_modes"] = []any{}
	}
	return shpAnswer(shpCall(id, SubmitContractReviewTool, args))
}

// convergeRun shapes a request whose first package drives first and whose second drives second.
func convergeRun(t *testing.T, request string, first, second []replayLiteral, deferral bool) (*shpFx, int, error) {
	f := newShpFx(t, shpSubmit("c1", convergeFiles(first, deferral)), shpSubmit("c2", convergeFiles(second, deferral)), shpSubmit("c3", convergeFiles(second, deferral)))
	f.Prov.contract = []shpStep{convergeReview("r1", first, deferral), convergeReview("r2", second, deferral), convergeReview("r3", second, deferral)}
	exit, err := f.shape(request)
	return f, exit, err
}

func TestReplayInventedLiteralsAreRefusedOnceThenDroppedAndListed(t *testing.T) {
	parallel(t)
	for _, c := range replayCases(t) {
		if len(c.First) == 0 {
			continue
		}
		t.Run(c.Cell, func(t *testing.T) {
			f, exit, err := convergeRun(t, c.Request, c.First, c.Second, false)
			if err != nil || exit != 0 {
				t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr.String())
			}
			// The first submission is a refusal naming every kept literal with its fix line.
			refused := ccRefused(f)
			if len(refused) != 1 {
				t.Fatalf("contract refusals = %d, want 1", len(refused))
			}
			problems := fmt.Sprint(refused[0]["problems"])
			for _, l := range c.First {
				if !strings.Contains(problems, l.Literal) || !strings.Contains(problems, "fix for invented literal") {
					t.Errorf("the refusal lacks %q or its fix line: %s", l.Literal, problems)
				}
			}
			// The second submission drops every kept literal, once each, with its submission number.
			dropped := shpNamed(f.events(), "invented_literal_dropped")
			if len(dropped) != len(c.Second) {
				t.Fatalf("dropped events = %v, want %d", dropped, len(c.Second))
			}
			for i, l := range c.Second {
				if dropped[i]["literal"] != l.Literal || dropped[i]["submission"] != float64(2) || dropped[i]["scenario"] == nil {
					t.Errorf("dropped event %d = %v, want literal %q at submission 2", i, dropped[i], l.Literal)
				}
			}
			// Every such literal is listed under the heading, with the words that the Request does not state them.
			out := f.Stdout.String()
			_, card, ok := strings.Cut(out, "Literals chosen by the Shaper")
			if !ok || !strings.Contains(card, "The Request does not state them") {
				t.Fatalf("no card section:\n%s", out)
			}
			for _, l := range c.Second {
				if !strings.Contains(card, l.Literal) {
					t.Errorf("the card does not list %q", l.Literal)
				}
			}
			var record []inventedRecord
			b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug, inventedDroppedFile))
			if rerr != nil || json.Unmarshal(b, &record) != nil || len(record) != len(c.Second) {
				t.Errorf("the dropped record = %s, %v", b, rerr)
			}
			// The run fits the Shape budget of 900 s.
			for _, e := range shpNamed(f.events(), "shape_time") {
				if ms, _ := e["elapsed_ms"].(float64); time.Duration(ms)*time.Millisecond > 900*time.Second {
					t.Errorf("shape time %v ms exceeds the 900 s budget", ms)
				}
			}
		})
	}
}

func TestReplayWeakenedGuaranteeIsRefusedWithTheRequestsWords(t *testing.T) {
	parallel(t)
	for _, c := range replayCases(t) {
		if c.Weak == "" {
			continue
		}
		t.Run(c.Cell, func(t *testing.T) {
			run := func(statement, verdict, intentValue string) string {
				it := decisionItem{statement: statement, quotes: []string{c.Request}, verdict: verdict, intentValue: intentValue, requestValue: c.Request, difference: "d", why: "w"}
				_, refusal, _, _ := decisionResolveNotes([]decisionItem{it}, c.Request, "## Decisions\n"+statement+"\n", nil)
				return strings.Join(refusal, "\n")
			}
			weak := run(c.Weak, "contradicts", c.WeakValue)
			if !strings.Contains(weak, `"`+c.Request+`"`) || !strings.Contains(weak, "restate the guarantee in the Request's own words") ||
				!strings.Contains(weak, "separate open limit") {
				t.Errorf("the weakened Decision's refusal:\n%s", weak)
			}
			if full := run(c.Full, "follows", ""); full != "" {
				t.Errorf("the full Decision was refused:\n%s", full)
			}
		})
	}
}

// A request conflict that states no guarantee keeps its general fix line.
func TestDecisionFixLineWithoutGuaranteeIsUnchanged(t *testing.T) {
	parallel(t)
	it := decisionItem{statement: "- Use red. (source: request)", quotes: []string{"Use blue."}, verdict: "contradicts", intentValue: "red", requestValue: "Use blue.", difference: "d", why: "w"}
	_, refusal, _, _ := decisionResolveNotes([]decisionItem{it}, "Use blue.", "## Decisions\n- Use red. (source: request)\n", nil)
	if got := strings.Join(refusal, "\n"); !strings.Contains(got, `Replace "red" with the request's requirement "Use blue."`) {
		t.Errorf("refusal:\n%s", got)
	}
}

func TestAMixedSubmissionKeepsUncoveredFailureModeRefused(t *testing.T) {
	parallel(t)
	lits := []replayLiteral{{Kind: "step", Literal: "bound = 40"}}
	f, exit, err := convergeRun(t, "Deliver exactly once.", lits, lits, true)
	if err == nil || exit == 0 {
		t.Fatalf("exit %d, err %v: the deferral without a failure mode passed", exit, err)
	}
	var invented, uncovered int
	for _, e := range ccRefused(f) {
		p := fmt.Sprint(e["problems"])
		if strings.Contains(p, "invented_contract") {
			invented++
		}
		if strings.Contains(p, "uncovered_failure_mode") {
			uncovered++
		}
	}
	if invented != 1 || uncovered < 2 {
		t.Errorf("invented_contract refusals %d (want 1), uncovered_failure_mode refusals %d (want >= 2)", invented, uncovered)
	}
	dropped := shpNamed(f.events(), "invented_literal_dropped")
	if len(dropped) == 0 || dropped[0]["literal"] != "bound = 40" {
		t.Errorf("dropped = %v", dropped)
	}
}

func TestApproveCardListsDroppedLiteralsOnlyWhenThereAreAny(t *testing.T) {
	parallel(t)
	root := t.TempDir()
	dir := filepath.Join(root, ".kogen/local/shaping/demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(dir, inventedDroppedFile)
	os.WriteFile(rec, []byte(`[{"scenario":"Even numbers","literal":"bound = 40","submission":2}]`), 0o644)
	card := func(lines []string) string {
		s := sys.NewWith(nil, "")
		var out strings.Builder
		s.Stdout = &out
		ready := &verify.Ready{Intent: &project.Intent{Title: "T", Text: "# T\n"}}
		approveCard(s, "demo", strings.Repeat("a", 64), "Me", ready, []string{"x.md"}, nil, lines)
		return out.String()
	}
	got := card(inventedDroppedLines(root, "demo"))
	want := "Literals chosen by the Shaper\n  The Request does not state them; check that each is acceptable:\n  - scenario: Even numbers | literal: \"bound = 40\"\nFiles\n"
	if !strings.Contains(got, want) {
		t.Fatalf("card:\n%s", got)
	}
	os.WriteFile(rec, []byte(`[]`), 0o644)
	if got := card(inventedDroppedLines(root, "demo")); strings.Contains(got, "Literals chosen by the Shaper") {
		t.Fatalf("a run without dropped literals shows the heading:\n%s", got)
	}
}
