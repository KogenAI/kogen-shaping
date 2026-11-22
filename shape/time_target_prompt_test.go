package shape

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

const (
	timeTargetLate = "submit the best complete package you have and spend later rounds only on the refusals Kogen names, with no new exploration of files you have not read."
	timeTargetGate = "A late package is still checked by every ordinary gate."
)

// TestTimeTargetPromptTellsShaperToConverge pins the submit-early instruction in both prompts.
func TestTimeTargetPromptTellsShaperToConverge(t *testing.T) {
	native := shaperPrompt("shaper")
	for _, want := range []string{
		"Crossing the target is never a stop, but converge inside it",
		"submit a complete package in the first rounds",
		"Once the elapsed time stated in the turn is within about 3 minutes of the usual target or past it, " + timeTargetLate,
		timeTargetGate,
	} {
		if !strings.Contains(native, want) {
			t.Errorf("native shaper prompt lacks %q", want)
		}
	}
	if strings.Contains(native, "The target is advice, not a stop") {
		t.Error("native shaper prompt keeps the old advice sentence")
	}
	for _, minutes := range []int{15, 25} {
		ext := externalBriefFor(time.Duration(minutes) * time.Minute)
		for _, want := range []string{
			"Crossing the target is never a stop, but converge inside it",
			"submit a complete package in the first rounds",
			"Once the elapsed time stated in the round is within about 3 minutes of the usual target of " + itoa(minutes) + " minutes or past it, " + timeTargetLate,
			timeTargetGate,
		} {
			if !strings.Contains(ext, want) {
				t.Errorf("external guidance for %d minutes lacks %q", minutes, want)
			}
		}
		if strings.Contains(ext, "The target is advice, not a stop") || strings.Contains(ext, "{{") {
			t.Errorf("external guidance for %d minutes keeps the old sentence or an unfilled placeholder", minutes)
		}
	}
}

// TestTimeTargetPromptNoElapsedStop: three 500 s rounds (two refused submissions, then a read) still get a fourth request.
func TestTimeTargetPromptNoElapsedStop(t *testing.T) {
	w := responsesWire
	bad := func(p string) shpStep {
		return w.shapeSubmit(p, map[string]any{"files": []any{map[string]any{"path": p, "content": "x\n"}}})
	}
	f := w.newShape(t, bad("a.go"), bad("b.go"), w.readReadme("r3"), w.readReadme("r4"))
	clock := &shpClock{t: time.Unix(1_800_000_000, 0)}
	f.Now = clock.now
	f.Prov.route = func(req map[string]any) *shpStep {
		if !decisionRequest(req) && !literalRequest(req) {
			clock.t = clock.t.Add(500 * time.Second)
		}
		return nil
	}
	_, err := f.shape("Bound shaping.")
	msg := ""
	if err != nil {
		msg = strings.ToLower(err.Error())
		if e, ok := sys.AsError(err); ok {
			msg += " " + strings.ToLower(e.Detail+" "+strings.Join(e.Lines, " "))
		}
	}
	if strings.Contains(msg, "budget") || strings.Contains(msg, "ceiling") {
		t.Errorf("the run names a budget or ceiling: %s", msg)
	}
	var lines []string
	for _, r := range f.Prov.requests() {
		all := timeLineRE.FindAllString(shpBody(r), -1)
		if len(all) > 0 {
			lines = append(lines, all[len(all)-1])
		}
	}
	if len(f.Prov.requests()) < 4 || len(lines) < 4 {
		t.Fatalf("requests = %d, time lines = %d, want at least four of each", len(f.Prov.requests()), len(lines))
	}
	want := []string{
		"Time: 0 min elapsed. Target: 15 minutes usual, 20 minutes upper end.",
		"Time: 8 min 20 s elapsed. Target: 15 minutes usual, 20 minutes upper end.",
		"Time: 16 min 40 s elapsed. Target: 15 minutes usual, 20 minutes upper end.",
		"Time: 25 min elapsed. Target: 15 minutes usual, 20 minutes upper end.",
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("time line %d = %q, want %q", i+1, lines[i], w)
		}
	}
}

// TestTimeTargetPromptLiteralStillRefusedLate: the literal check has no time input.
func TestTimeTargetPromptLiteralStillRefusedLate(t *testing.T) {
	req := "The build must finish in under 120 seconds."
	model := []literalItem{litItem("under 120 seconds", "request", "under 120 seconds", "required", "")}
	status := func(pkg string) string {
		items := withRequestLiterals(req, model)
		ev, _ := literalOutcome(items, map[string]string{"request": req}, []packageFile{{"intent.md", pkg}}, verify.NewRedactor(nil, ""))
		for _, raw := range ev["items"].([]any) {
			m := raw.(map[string]any)
			if m["literal"] == "under 120 seconds" {
				return m["status"].(string)
			}
		}
		return "absent"
	}
	if got := status("The build finishes in under 120 seconds."); got != "present" {
		t.Errorf("kept literal status = %q, want present", got)
	}
	if got := status("The build finishes in under 125 seconds."); got != "missing" {
		t.Errorf("weakened literal status = %q, want missing", got)
	}
}

// TestTimeTargetPromptShapeTimeEventUnchanged pins every field of shape_time as recorded before the change.
func TestTimeTargetPromptShapeTimeEventUnchanged(t *testing.T) {
	account := func(build func(ev *Evidence, clock *shpClock)) string {
		ev, clock, dir := newTestEvidence(t)
		ev.AccountTime()
		build(ev, clock)
		if err := ev.Finish(); err != nil {
			t.Fatal(err)
		}
		got := shpNamed(readEvents(t, dir), "shape_time")
		if len(got) != 1 {
			t.Fatalf("shape_time events: %d", len(got))
		}
		b, err := json.Marshal(got[0])
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	round := func(ev *Evidence, clock *shpClock, ms int64) {
		a := ev.Millis()
		clock.advance(ms)
		ev.ShaperRound(a, ev.Millis(), nil)
	}
	decisionOnly := account(func(ev *Evidence, clock *shpClock) {
		round(ev, clock, 100000)
		b := ev.Millis()
		clock.advance(60000)
		ev.Interval("model_checks", b, b+60000)
		ev.CheckSpan(b, b+60000)
	})
	if want := `{"acceptance timeout":[],"checks_ms":60000,"contract_ms":0,"elapsed_ms":160000,"event":"shape_time","gap_ms":0,"install_ms":0,"local_check_ms":0,"model_checks_ms":60000,"own check seconds":60,"readiness_ms":0,"setup_ms":0,"shaper_model_ms":null,"shaper_ms":100000,"shaper_tools_ms":null,"static_ms":0,"static_uncharged_ms":0}`; decisionOnly != want {
		t.Errorf("decision-only shape_time =\n%s\nwant\n%s", decisionOnly, want)
	}
	readinessStatic := account(func(ev *Evidence, clock *shpClock) {
		round(ev, clock, 100000)
		b := ev.Millis()
		clock.advance(320000)
		ev.Interval("readiness", b, b+300000)
		ev.Interval("static", b+200000, b+320000)
		ev.CheckSpan(b, b+320000)
	})
	if want := `{"acceptance timeout":[],"checks_ms":320000,"contract_ms":0,"elapsed_ms":420000,"event":"shape_time","gap_ms":0,"install_ms":0,"local_check_ms":320000,"model_checks_ms":0,"own check seconds":320,"readiness_ms":300000,"setup_ms":0,"shaper_model_ms":null,"shaper_ms":100000,"shaper_tools_ms":null,"static_ms":120000,"static_uncharged_ms":0}`; readinessStatic != want {
		t.Errorf("readiness and static shape_time =\n%s\nwant\n%s", readinessStatic, want)
	}
}

var timeLineRE = regexp.MustCompile(`Time: [^"\\]*\.`)
