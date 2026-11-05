'use strict';
// The Go source of the k-shape-sourced-decision-continuity acceptance harness, which
// k-shape-sourced-decision-continuity__steps.js compiles into the shape package's tests through a Go
// overlay as shape/k_shape_sourced_decision_continuity_acceptance_test.go; that path never exists in
// the repository. The source holds no backquote and no dollar-brace, so this raw template keeps it
// byte for byte.
module.exports = String.raw`package shape

// The acceptance harness of k-shape-sourced-decision-continuity: synthetic shaping journeys of
// one or more Shape runs, continued through the existing answer input, in the shape tests'
// is-even project with an invented parcel audit fixture committed on top. The native cases play
// the shaper and Kogen's question check with the shape package's scripted local provider; the
// external case plays them with the scripted claude-code executor of the shape tests. KSDC_CASE
// names one case; what the runs showed is written as JSON to KSDC_OUT for the acceptance steps to
// judge. The harness judges nothing itself.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-loop/sys"
)

const ksdcBT = "\x60"

const ksdcRequest = "Make parcel inspection obey the existing safety policy; preserve normal reads. The new audit panel's visual presentation is open.\n"

// The parcel audit fixture: the safety policy, an unrelated retention note and the audit code.
var ksdcFixture = map[string]string{
	"docs/policy.md":    "# Parcel safety policy\n\nAn unreadable parcel record produces an incomplete audit observation with safe diagnostics; record bytes are never included.\n",
	"docs/retention.md": "# Retention\n\nAudit observations are kept for 30 days.\n",
	"src/audit.go": "package audit\n\n// Observation is one audit observation of a parcel inspection.\ntype Observation struct {\n\tParcel      string\n\tComplete    bool\n\tDiagnostics []string\n}\n\n" +
		"// Incomplete is the observation of a parcel whose record could not be read: safe diagnostics only.\nfunc Incomplete(parcel, reason string) Observation {\n\treturn Observation{Parcel: parcel, Diagnostics: []string{\"unreadable record: \" + reason}}\n}\n",
}

// ksdcChangedPolicy is the safety policy as a later commit changes it.
const ksdcChangedPolicy = "# Parcel safety policy\n\nAn unreadable parcel record produces an incomplete audit observation with safe diagnostics and the record's size; record bytes are never included.\n"

const (
	ksdcProbeOK   = "grep -n Incomplete src/audit.go"
	ksdcProbeFail = "grep -n 'never included' src/audit.go"
)

func ksdcRan(command string) string { return "ran " + ksdcBT + command + ksdcBT }

func ksdcQ(text string, options ...string) map[string]any {
	list := []any{}
	for i, o := range options {
		list = append(list, map[string]any{"label": o, "recommended": i == 0})
	}
	return map[string]any{"question": text, "options": list}
}

var (
	ksdcUnreadable = ksdcQ("What should a user experience on an unreadable record: reject the operation or save an incomplete observation?", "Save an incomplete observation", "Reject the operation")
	ksdcPanel      = ksdcQ("Show audit details inline or in an expandable panel?", "In an expandable panel", "Inline")
	ksdcBytes      = ksdcQ("For incomplete parcel observations, should record bytes be attached or only safe diagnostics?", "Only safe diagnostics", "Attach record bytes")
	ksdcRestated   = ksdcQ("When a parcel record cannot be read, should the saved observation carry the raw record or just its diagnostics?", "Just its diagnostics", "The raw record")
	ksdcOrder      = ksdcQ("Should the audit panel list the newest observation first or the oldest first?", "Newest first", "Oldest first")
)

const (
	ksdcTextD1      = "An unreadable parcel record produces an incomplete audit observation with safe diagnostics, built with audit.Incomplete"
	ksdcTextD2      = "Normal parcel reads keep their current behavior"
	ksdcTextD3      = "An unreadable parcel record produces an incomplete audit observation with safe diagnostics and the record's size, built with audit.Incomplete"
	ksdcConstraint  = "record bytes are never included"
	ksdcUncertainty = "the repository has no test of normal reads to cite"
	ksdcWhy         = "docs/policy.md now also requires the record's size in the diagnostics"
)

func ksdcList(items ...string) []any {
	out := []any{}
	for _, it := range items {
		out = append(out, it)
	}
	return out
}

// The decisions the shaper records: the incomplete-observation choice with its evidence and
// constraint, the normal-read choice with its stated uncertainty, their versions that cite
// unread or failed evidence, and the revision after the policy changed.
var (
	ksdcD1      = map[string]any{"decision": ksdcTextD1, "evidence": ksdcList("docs/policy.md", "src/audit.go", ksdcRan(ksdcProbeOK)), "constraints": ksdcList(ksdcConstraint)}
	ksdcD1Bad   = map[string]any{"decision": ksdcTextD1, "evidence": ksdcList("docs/policy.md", "src/audit.go", "docs/retention.md", ksdcRan(ksdcProbeFail)), "constraints": ksdcList(ksdcConstraint)}
	ksdcD1Ext   = map[string]any{"decision": ksdcTextD1, "evidence": ksdcList("docs/policy.md", "src/audit.go"), "constraints": ksdcList(ksdcConstraint)}
	ksdcD2      = map[string]any{"decision": ksdcTextD2, "evidence": ksdcList(), "constraints": ksdcList(), "uncertainty": ksdcUncertainty}
	ksdcD2Bad   = map[string]any{"decision": ksdcTextD2}
	ksdcD3      = map[string]any{"decision": ksdcTextD3, "evidence": ksdcList("docs/policy.md", "src/audit.go"), "constraints": ksdcList(ksdcConstraint), "revises": "D1", "why": ksdcWhy}
	ksdcLineD1  = "- " + ksdcTextD1 + " (decision: D1, round 1) (constraint: " + ksdcConstraint + ") (evidence: docs/policy.md; src/audit.go; " + ksdcRan(ksdcProbeOK) + ") (source: request)"
	ksdcLineD1X = "- " + ksdcTextD1 + " (decision: D1, round 1) (constraint: " + ksdcConstraint + ") (evidence: docs/policy.md; src/audit.go) (source: request)"
	ksdcLineD2  = "- " + ksdcTextD2 + " (decision: D2, round 1) (assumption: " + ksdcUncertainty + ") (source: request)"
	ksdcLineD3  = "- " + ksdcTextD3 + " (decision: D3, round 3) (constraint: " + ksdcConstraint + ") (evidence: docs/policy.md; src/audit.go) (revises: D1 because " + ksdcWhy + ") (source: request)"
	ksdcLineBad = "- Incomplete parcel observations attach the record bytes for debugging (source: request)"
)

// ksdcIntent is the shape tests' scoped is-even Intent with these Decisions lines.
func ksdcIntent(lines ...string) string {
	return strings.Replace(shpScopedIntentMD, "- Non-integers raise TypeError. (source: request)", strings.Join(lines, "\n"), 1)
}

func ksdcPackage(intent string, decisions ...any) map[string]any {
	p := map[string]any{"files": []any{
		map[string]any{"path": ".kogen/intents/is-even/intent.md", "content": intent},
		map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature},
		map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps},
	}}
	if len(decisions) > 0 {
		p["decisions"] = decisions
	}
	return p
}

func ksdcAsk(q map[string]any, decisions ...any) map[string]any {
	a := map[string]any{"questions": []any{q}}
	if len(decisions) > 0 {
		a["decisions"] = decisions
	}
	return a
}

// ksdcCheck is a question check answer for one question: its class, why and decision.
func ksdcCheck(class, why, decision string) map[string]any {
	return map[string]any{"items": []any{map[string]any{"number": 1, "class": class, "why": why, "decision": decision}}}
}

const (
	ksdcWhyTechnical = "rejecting or saving an unreadable record is a security and data-handling choice, whatever its wording"
	ksdcWhyPanel     = "how audit details are shown is an interface choice the request leaves open"
	ksdcWhyOrder     = "the order of the audit panel's list is an interface choice"
	ksdcWhyBytes     = "D1 settles it: record bytes are never included"
	ksdcWhyRestated  = "D1 settles it in other words: an incomplete observation carries safe diagnostics, never the record"
)

var (
	ksdcReads = shpAnswer(
		shpCall("t1", "read_file", map[string]any{"path": "docs/policy.md"}),
		shpCall("t2", "read_file", map[string]any{"path": "src/audit.go"}),
		shpCall("t3", "probe", map[string]any{"command": ksdcProbeOK}),
		shpCall("t4", "probe", map[string]any{"command": ksdcProbeFail}))
	ksdcReread = shpAnswer(shpCall("t5", "read_file", map[string]any{"path": "docs/policy.md"}))
	// ksdcFirstRound is the first run of the native journeys: the disguised technical question,
	// the reads and probes, decisions citing an unread file and a failed probe, then the
	// recorded decisions with the open presentation question.
	ksdcFirstRound = []shpStep{
		shpSubmit("s1", ksdcAsk(ksdcUnreadable)),
		ksdcReads,
		shpSubmit("s2", ksdcAsk(ksdcPanel, ksdcD1Bad, ksdcD2Bad)),
		shpSubmit("s3", ksdcAsk(ksdcPanel, ksdcD1, ksdcD2)),
	}
	ksdcFirstChecks = []map[string]any{ksdcCheck("technical", ksdcWhyTechnical, ""), ksdcCheck("ui_ux", ksdcWhyPanel, "")}
)

type ksdcRun struct {
	input  string
	before map[string]string // files committed before the run
}

type ksdcCase struct {
	runs     []ksdcRun
	shaper   []shpStep
	checks   []map[string]any
	external [][]ksdcRound // the external case: the scripted rounds of each run
	want     string        // the intent.md a published journey writes; "" when it publishes nothing
}

// ksdcRound is one scripted external round: the main shaper's or a question check session's.
type ksdcRound struct {
	label  string
	result any
}

func ksdcSteps(parts ...[]shpStep) []shpStep {
	var out []shpStep
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

var ksdcCases = map[string]ksdcCase{
	"ordinary": {
		runs:   []ksdcRun{{input: ksdcRequest}},
		shaper: []shpStep{shpSubmit("s1", ksdcPackage(shpScopedIntentMD))},
		want:   shpScopedIntentMD,
	},
	"settle": {
		runs: []ksdcRun{{input: ksdcRequest}, {input: "1: a\n"}},
		shaper: ksdcSteps(ksdcFirstRound, []shpStep{
			shpSubmit("s4", ksdcAsk(ksdcBytes)),
			shpSubmit("s5", ksdcPackage(ksdcIntent(ksdcLineBad))),
			shpSubmit("s6", ksdcPackage(ksdcIntent(ksdcLineD1, ksdcLineD2), ksdcD1, ksdcD2)),
		}),
		checks: append(append([]map[string]any{}, ksdcFirstChecks...), ksdcCheck("decided", ksdcWhyBytes, "D1")),
		want:   ksdcIntent(ksdcLineD1, ksdcLineD2),
	},
	"revise": {
		runs: []ksdcRun{{input: ksdcRequest}, {input: "1: a\n"}, {input: "1: a\n", before: map[string]string{"docs/policy.md": ksdcChangedPolicy}}},
		shaper: ksdcSteps(ksdcFirstRound, []shpStep{
			shpSubmit("s4", ksdcAsk(ksdcOrder)),
			shpSubmit("s5", ksdcAsk(ksdcRestated)),
			shpSubmit("s6", ksdcPackage(ksdcIntent(ksdcLineD3, ksdcLineD2), ksdcD3)),
			ksdcReread,
			shpSubmit("s7", ksdcPackage(ksdcIntent(ksdcLineD3, ksdcLineD2), ksdcD3)),
		}),
		checks: append(append([]map[string]any{}, ksdcFirstChecks...), ksdcCheck("ui_ux", ksdcWhyOrder, ""), ksdcCheck("decided", ksdcWhyRestated, "D1")),
		want:   ksdcIntent(ksdcLineD3, ksdcLineD2),
	},
	"repeat": {
		runs: []ksdcRun{{input: ksdcRequest}, {input: "1: a\n"}},
		shaper: ksdcSteps(ksdcFirstRound, []shpStep{
			shpSubmit("s4", ksdcAsk(ksdcBytes)),
			shpSubmit("s5", ksdcAsk(ksdcRestated)),
			shpSubmit("s6", ksdcAsk(ksdcBytes)),
		}),
		checks: append(append([]map[string]any{}, ksdcFirstChecks...), ksdcCheck("decided", ksdcWhyBytes, "D1"), ksdcCheck("decided", ksdcWhyRestated, "D1"), ksdcCheck("decided", ksdcWhyBytes, "D1")),
	},
	"external": {
		runs: []ksdcRun{{input: ksdcRequest}, {input: "1: a\n"}},
		external: [][]ksdcRound{
			{
				{"x1", ksdcAsk(ksdcUnreadable)},
				{"q1", ksdcCheck("technical", ksdcWhyTechnical, "")},
				{"x2", ksdcAsk(ksdcPanel, ksdcD1, ksdcD2)},
				{"x3", ksdcAsk(ksdcPanel, ksdcD1Ext, ksdcD2)},
				{"q2", ksdcCheck("ui_ux", ksdcWhyPanel, "")},
			},
			{
				{"x4", ksdcAsk(ksdcBytes)},
				{"q3", ksdcCheck("decided", ksdcWhyBytes, "D1")},
				{"x5", ksdcPackage(ksdcIntent(ksdcLineD1X, ksdcLineD2))},
			},
		},
		want: ksdcIntent(ksdcLineD1X, ksdcLineD2),
	},
}

// ksdcIsCheck is whether a request is a question check: it offers submit_question_check.
func ksdcIsCheck(req map[string]any) bool {
	for _, n := range shpToolNames(req) {
		if n == "submit_question_check" {
			return true
		}
	}
	return false
}

// ksdcUserText is the text of the request's first user message: a string, or its input_text
// blocks joined.
func ksdcUserText(req map[string]any) string {
	input, _ := req["input"].([]any)
	for _, it := range input {
		m, _ := it.(map[string]any)
		if m == nil || m["role"] != "user" {
			continue
		}
		switch c := m["content"].(type) {
		case string:
			return c
		case []any:
			var parts []string
			for _, b := range c {
				if bm, _ := b.(map[string]any); bm != nil {
					if s, ok := bm["text"].(string); ok {
						parts = append(parts, s)
					}
				}
			}
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

var (
	ksdcWall     = regexp.MustCompile("(?m)^wall: [0-9.]+ s$")
	ksdcSubmitID = regexp.MustCompile("^s[0-9]+$")
	ksdcToolID   = regexp.MustCompile("^t[0-9]+$")
	ksdcManifest = regexp.MustCompile("(?s)\n\nCumulative proposal manifest: .*$")
)

func ksdcNorm(s string) string { return ksdcWall.ReplaceAllString(s, "wall: <wall> s") }

// ksdcSection is the recorded-decisions section of a shaper's context, "" without one.
func ksdcSection(text string) string {
	i := strings.Index(text, "# Your recorded decisions")
	if i < 0 {
		return ""
	}
	rest := text[i:]
	if j := strings.Index(rest, "\n\n# The Intent to write"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// ksdcNormJSON is a decoded JSON value with every probe wall line normalised.
func ksdcNormJSON(v any) any {
	switch x := v.(type) {
	case string:
		return ksdcNorm(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = ksdcNormJSON(e)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, e := range x {
			out[k] = ksdcNormJSON(e)
		}
		return out
	}
	return v
}

const ksdcLedger = ".kogen/local/shaping/is-even/decisions.json"

func TestKShapeSourcedDecisionContinuityAcceptance(t *testing.T) {
	name, out := os.Getenv("KSDC_CASE"), os.Getenv("KSDC_OUT")
	c, ok := ksdcCases[name]
	if !ok || out == "" {
		t.Fatalf("KSDC_CASE must name a case (ordinary, settle, revise, repeat, external), not %q, and KSDC_OUT must be set", name)
	}
	f := newShpFx(t, c.shaper...)
	var mu sync.Mutex
	checks := c.checks
	f.Prov.route = func(req map[string]any) *shpStep {
		if !ksdcIsCheck(req) {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		if len(checks) == 0 {
			return &shpStep{status: 400, code: "invalid_request_error"}
		}
		st := shpAnswer(shpCall("qc", "submit_question_check", checks[0]))
		checks = checks[1:]
		return &st
	}
	for p, content := range ksdcFixture {
		f.write(p, content)
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "Add the parcel audit fixture")
	if c.external != nil {
		f.useExternalShape()
	}
	f.setReport("PASSED", 0)

	obs := map[string]any{"case": name}
	exits, errs, lastLines := []any{}, []any{}, []any{}
	errorLines := []string{}
	shaperCounts, checkCounts := []any{}, []any{}
	var checkInputs, checkInstructions []string
	var checkSchema any
	var previous []byte
	var labels []string
	for i, run := range c.runs {
		n := fmt.Sprint(i + 1)
		if len(run.before) > 0 {
			for p, content := range run.before {
				f.write(p, content)
			}
			f.git("add", "-A")
			f.git("commit", "-qm", "Change the fixture")
		}
		before := len(f.Prov.allRequests())
		var exit int
		var err error
		var x *shapeExec
		if c.external != nil {
			rounds := c.external[i]
			x = &shapeExec{t: t}
			for _, r := range rounds {
				r := r
				x.rounds = append(x.rounds, func(int, string) (executor.RoundResult, error) {
					labels = append(labels, r.label)
					return result(r.result)(0, "")
				})
			}
			labels = nil
			exit, err = f.shapeWith(x, run.input)
		} else {
			exit, err = f.shape(run.input)
		}
		stdout := f.Stdout.String()
		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		lastLines = append(lastLines, lines[len(lines)-1])
		if exit == 5 {
			obs["stdout_"+n] = stdout
		}
		e := ""
		errorLines = []string{}
		if err != nil {
			e = err.Error()
			if se, ok := sys.AsError(err); ok {
				exit = se.Exit
				errorLines = append(errorLines, se.Lines...)
			}
		}
		exits, errs = append(exits, exit), append(errs, e)
		obs["stderr_"+n] = f.Stderr.String()

		shaper, checksN := 0, 0
		if x != nil {
			for k, task := range x.starts {
				if strings.Contains(task.Dir, "/executor-question-") {
					checksN++
					stable := task.Brief.Stable
					j := strings.Index(stable, "\n\nSlug: ")
					if j < 0 {
						checkInputs = append(checkInputs, stable)
						continue
					}
					checkInstructions = append(checkInstructions, stable[:j])
					checkInputs = append(checkInputs, ksdcNorm(stable[j+2:]))
					if checkSchema == nil {
						json.Unmarshal(task.Schema, &checkSchema)
					}
				} else if k == 0 {
					obs["context_decisions_"+n] = ksdcSection(task.Brief.Stable)
				}
			}
			last := ""
			for k, label := range labels {
				if label[0] != 'x' {
					continue
				}
				shaper++
				if last != "" && k < len(x.inputs) && x.inputs[k].Feedback != "" {
					obs["feedback_"+last] = ksdcNorm(ksdcManifest.ReplaceAllString(x.inputs[k].Feedback, ""))
				}
				last = label
			}
		} else {
			first := true
			for _, r := range f.Prov.allRequests()[before:] {
				if contractKind(r) != "" || decisionRequest(r) {
					continue
				}
				if ksdcIsCheck(r) {
					checksN++
					checkInputs = append(checkInputs, ksdcNorm(ksdcUserText(r)))
					if s, ok := r["instructions"].(string); ok {
						checkInstructions = append(checkInstructions, s)
					}
					if checkSchema == nil {
						if tl, _ := r["tools"].([]any); len(tl) == 1 {
							if m, _ := tl[0].(map[string]any); m != nil {
								checkSchema = m["parameters"]
							}
						}
					}
					continue
				}
				shaper++
				if first {
					obs["context_decisions_"+n] = ksdcSection(ksdcUserText(r))
					first = false
				}
				for id, text := range shpOutputs(r) {
					switch {
					case ksdcSubmitID.MatchString(id):
						obs["feedback_"+id] = ksdcNorm(text)
					case ksdcToolID.MatchString(id):
						obs["output_"+id] = ksdcNorm(text)
					}
				}
			}
		}
		shaperCounts, checkCounts = append(shaperCounts, shaper), append(checkCounts, checksN)

		b, rerr := os.ReadFile(filepath.Join(f.Root, ksdcLedger))
		if rerr == nil {
			var v any
			if jerr := json.Unmarshal(b, &v); jerr != nil {
				obs["ledger_"+n] = "not JSON: " + string(b)
			} else {
				obs["ledger_"+n] = ksdcNormJSON(v)
			}
			if st, serr := os.Stat(filepath.Join(f.Root, ksdcLedger)); serr == nil {
				obs["ledger_mode_"+n] = fmt.Sprintf("%04o", st.Mode().Perm())
			}
		} else {
			obs["ledger_"+n] = nil
		}
		if i > 0 {
			obs["ledger_unchanged_"+n] = rerr == nil && previous != nil && bytes.Equal(b, previous)
		}
		previous = nil
		if rerr == nil {
			previous = b
		}
	}
	obs["exits"], obs["errors"], obs["error_lines"], obs["last_stdout_lines"] = exits, errs, errorLines, lastLines
	obs["shaper_requests"], obs["question_check_requests"] = shaperCounts, checkCounts
	for k, in := range checkInputs {
		obs[fmt.Sprintf("question_check_input_%d", k+1)] = in
	}
	if len(checkInstructions) > 0 {
		obs["question_check_instructions"] = checkInstructions[0]
	}
	obs["question_check_schema"] = checkSchema

	gates, classes, decided, carried, recorded := []any{}, []any{}, []any{}, []any{}, []any{}
	if b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/local/shaping/is-even/events.ndjson")); rerr == nil {
		for _, line := range strings.Split(string(b), "\n") {
			var ev map[string]any
			if line == "" || json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			switch ev["event"] {
			case "submission_refused":
				gates = append(gates, ev["gate"])
			case "question_check":
				cs, ds := []any{}, []any{}
				items, _ := ev["items"].([]any)
				for _, it := range items {
					if m, _ := it.(map[string]any); m != nil {
						cs, ds = append(cs, m["class"]), append(ds, m["decision"])
					}
				}
				classes, decided = append(classes, cs), append(decided, ds)
			case "carried_evidence":
				carried = append(carried, ev["items"])
			case "decisions_recorded":
				delete(ev, "event")
				recorded = append(recorded, ev)
			}
		}
	}
	obs["refused_gates"], obs["question_check_classes"], obs["question_check_decisions"] = gates, classes, decided
	obs["carried_evidence"], obs["decisions_recorded"] = carried, recorded
	obs["intent_exists"] = f.exists(".kogen/intents/is-even/intent.md")
	published := false
	if b, rerr := os.ReadFile(filepath.Join(f.Root, ".kogen/intents/is-even/intent.md")); rerr == nil {
		published = c.want != "" && string(b) == c.want && f.exists(".kogen/local/readiness/is-even.json")
		if !published {
			obs["intent_md"] = string(b)
		}
	}
	obs["published"] = published
	b, jerr := json.MarshalIndent(obs, "", " ")
	if jerr != nil {
		t.Fatal(jerr)
	}
	if werr := os.WriteFile(out, b, 0o644); werr != nil {
		t.Fatal(werr)
	}
}
`;
