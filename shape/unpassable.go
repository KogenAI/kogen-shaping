package shape

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// The unpassable-step check: an acceptance step definition for which no pass is reachable fails
// the same way for every implementation and every repository state, so the Build could only stop
// on it. Readiness counts such a step as ordinary missing-feature red; this check tells them
// apart with one control run of the candidate's own acceptance in which everything acceptance
// code obtains from outside itself is a stand-in (unpassable_preload.js).

//go:embed unpassable_preload.js
var unpassablePreload string

//go:embed unpassable_support.js
var unpassableSupport string

// unpassableBound bounds the control run.
var unpassableBound = verify.StepTimeout

// The verdicts of one judged step definition.
const (
	verdictTimeout     = "timeout"
	verdictPassed      = "passed"
	verdictStandInUsed = "stand_in_used"
	verdictDifferent   = "different_failure"
	verdictUnpassable  = "unpassable"
)

// unpassableStep is one judged step definition, at its first failing use.
type unpassableStep struct {
	step       string // acceptance.feature line <L>: <Keyword> <text>
	definition string // <file> line <n>
	pattern    string
	verdict    string
	message    string // the first line of the readiness failure message
}

// unpassableError is the refusal of a candidate with step definitions no implementation can pass.
type unpassableError struct{ items []string }

func (e *unpassableError) Error() string { return strings.SplitN(e.text(), "\n", 2)[0] }

func (e *unpassableError) text() string {
	return fmt.Sprintf("Acceptance has steps that no implementation can pass: %d step definition(s) fail the same way whatever the repository contains, so no pass is reachable; nothing was published. "+
		"Each step must be able to pass once the requested behaviour exists: run the program, call the code or inspect its output, and compare what it observed with an assertion that shows the actual value (for example assert.strictEqual). "+
		"Rewrite these steps; resubmitting them unchanged is refused again.", len(e.items)) +
		"\n" + strings.Join(e.items, "\n")
}

// item is the refusal line of an unpassable definition.
func (s unpassableStep) item() string {
	message := []rune(s.message)
	if len(message) > 200 {
		message = message[:200]
	}
	return fmt.Sprintf("- %s (step definition %q, %s): no pass is reachable: it fails with %q whatever the repository contains", s.step, s.pattern, s.definition, string(message))
}

// ---- the control run's observations ----

type controlDef struct {
	pattern string
	file    string
	line    int
}

// controlStep is one Gherkin step of a scenario of the control run.
type controlStep struct {
	line    int
	def     string // the id of its one definition, "" when it has none or several
	status  string // the Cucumber Messages status, upper case
	detail  string // the failure detail as readiness words it, "" unless the step failed
	standIn bool   // a stand-in was used in the scenario up to and including this step
}

type controlCase struct{ steps []controlStep }

// controlRun is what the control run observed: the definitions and, per scenario line, the steps.
type controlRun struct {
	defs  map[string]controlDef
	cases map[int]*controlCase
	dups  map[int]bool // scenario lines reported more than once
}

func (c *controlRun) caseAt(line int) *controlCase {
	if c == nil || c.dups[line] {
		return nil
	}
	return c.cases[line]
}

var (
	failureLocation = regexp.MustCompile(`^acceptance\.feature line (\d+): `)
	timeoutFailure  = regexp.MustCompile(`function timed out, ensure the promise resolves within \d+ milliseconds`)
)

// withoutFrames removes the stack-frame lines (those that start with "at" after their leading
// whitespace) of a failure detail.
func withoutFrames(detail string) string {
	var kept []string
	for _, l := range strings.Split(detail, "\n") {
		if !strings.HasPrefix(strings.TrimLeft(l, " \t"), "at ") {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

// judgeUnpassable judges every step definition that is the failing step of a failed scenario of
// the readiness result against the control run. A non-empty reason means the readiness result
// and the control run do not match and nothing can be judged.
func judgeUnpassable(ready *verify.Ready, ctl *controlRun) (steps []unpassableStep, reason string) {
	type use struct {
		different bool
		standIn   bool
		timedOut  bool
	}
	type judged struct {
		step unpassableStep
		uses []use
	}
	var order []string
	byDef := map[string]*judged{}
	passedDefs := map[string]bool{}

	for _, sc := range ready.Scenarios {
		if sc.Status != "passed" && sc.Status != "failed" {
			continue
		}
		c := ctl.caseAt(sc.Line)
		if c == nil || len(c.steps) != len(sc.Steps) {
			return nil, fmt.Sprintf("scenario at line %d is not matched in the control report", sc.Line)
		}
		for i, st := range sc.Steps {
			if c.steps[i].line != st.Line {
				return nil, fmt.Sprintf("step at line %d is not matched in the control report", st.Line)
			}
		}
		failing := -1 // the index of the failing step, -1 when the scenario failed in a hook
		detail := ""
		if sc.Status == "failed" && sc.Detail != nil {
			detail = *sc.Detail
			if m := failureLocation.FindStringSubmatch(detail); m != nil {
				for i, st := range sc.Steps {
					if fmt.Sprint(st.Line) == m[1] {
						failing = i
					}
				}
				if failing < 0 {
					return nil, fmt.Sprintf("the failing step of the scenario at line %d is not matched in the control report", sc.Line)
				}
			}
		}
		passedUpTo := len(c.steps) // a hook failure leaves every step of the scenario passed
		if failing >= 0 {
			passedUpTo = failing
		}
		for _, st := range c.steps[:passedUpTo] {
			if st.def == "" {
				return nil, fmt.Sprintf("step at line %d has no single definition in the control report", st.line)
			}
			passedDefs[st.def] = true
		}
		if failing < 0 {
			continue
		}

		cs := c.steps[failing]
		if cs.def == "" {
			return nil, fmt.Sprintf("step at line %d has no single definition in the control report", cs.line)
		}
		lines := strings.Split(detail, "\n")
		j := byDef[cs.def]
		if j == nil {
			def := ctl.defs[cs.def]
			j = &judged{step: unpassableStep{step: strings.TrimRight(lines[0], " \t"), definition: fmt.Sprintf("%s line %d", def.file, def.line), pattern: def.pattern}}
			if len(lines) > 1 {
				j.step.message = lines[1]
			}
			byDef[cs.def] = j
			order = append(order, cs.def)
		}
		first := -1 // the first step of the control scenario that did not pass
		for i, s := range c.steps {
			if s.status != "PASSED" {
				first = i
				break
			}
		}
		same := first == failing && cs.status == "FAILED" && withoutFrames(cs.detail) == withoutFrames(detail)
		j.uses = append(j.uses, use{
			timedOut:  timeoutFailure.MatchString(detail),
			standIn:   cs.standIn,
			different: !same,
		})
	}

	for _, key := range order {
		j := byDef[key]
		var timedOut, standIn, different bool
		for _, u := range j.uses {
			timedOut = timedOut || u.timedOut
			standIn = standIn || u.standIn
			different = different || (!u.standIn && u.different)
		}
		switch {
		case timedOut:
			j.step.verdict = verdictTimeout
		case passedDefs[key]:
			j.step.verdict = verdictPassed
		case standIn:
			j.step.verdict = verdictStandInUsed
		case different:
			j.step.verdict = verdictDifferent
		default:
			j.step.verdict = verdictUnpassable
		}
		steps = append(steps, j.step)
	}
	return steps, ""
}

// ---- the Cucumber Messages report of the control run ----

type reportMessage struct {
	GherkinDocument *struct {
		Feature *struct {
			Children []reportChild `json:"children"`
		} `json:"feature"`
	} `json:"gherkinDocument"`
	Pickle *struct {
		ID         string   `json:"id"`
		AstNodeIDs []string `json:"astNodeIds"`
		Steps      []struct {
			ID         string   `json:"id"`
			Text       string   `json:"text"`
			AstNodeIDs []string `json:"astNodeIds"`
		} `json:"steps"`
	} `json:"pickle"`
	StepDefinition *struct {
		ID      string `json:"id"`
		Pattern struct {
			Source string `json:"source"`
		} `json:"pattern"`
		SourceReference struct {
			URI      string `json:"uri"`
			Location struct {
				Line int `json:"line"`
			} `json:"location"`
		} `json:"sourceReference"`
	} `json:"stepDefinition"`
	TestCase        *reportTestCase `json:"testCase"`
	TestCaseStarted *struct {
		ID         string `json:"id"`
		TestCaseID string `json:"testCaseId"`
	} `json:"testCaseStarted"`
	TestStepFinished *struct {
		TestCaseStartedID string `json:"testCaseStartedId"`
		TestStepID        string `json:"testStepId"`
		TestStepResult    struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Exception *struct {
				Type       string `json:"type"`
				Message    string `json:"message"`
				StackTrace string `json:"stackTrace"`
			} `json:"exception"`
		} `json:"testStepResult"`
	} `json:"testStepFinished"`
	TestCaseFinished *struct {
		TestCaseStartedID string `json:"testCaseStartedId"`
		WillBeRetried     bool   `json:"willBeRetried"`
	} `json:"testCaseFinished"`
}

type reportTestCase struct {
	ID        string `json:"id"`
	PickleID  string `json:"pickleId"`
	TestSteps []struct {
		ID                string   `json:"id"`
		PickleStepID      string   `json:"pickleStepId"`
		StepDefinitionIDs []string `json:"stepDefinitionIds"`
	} `json:"testSteps"`
}

type reportChild struct {
	Background *reportScenario `json:"background"`
	Scenario   *reportScenario `json:"scenario"`
	Rule       *struct {
		Children []reportChild `json:"children"`
	} `json:"rule"`
}

type reportScenario struct {
	ID       string     `json:"id"`
	Location reportLine `json:"location"`
	Steps    []struct {
		ID       string     `json:"id"`
		Keyword  string     `json:"keyword"`
		Location reportLine `json:"location"`
	} `json:"steps"`
	Examples []struct {
		TableBody []struct {
			ID       string     `json:"id"`
			Location reportLine `json:"location"`
		} `json:"tableBody"`
	} `json:"examples"`
}

type reportLine struct {
	Line int `json:"line"`
}

type astNode struct {
	line    int
	keyword string
}

func astNodes(children []reportChild, out map[string]astNode) {
	for _, ch := range children {
		if ch.Rule != nil {
			astNodes(ch.Rule.Children, out)
			continue
		}
		sc := ch.Scenario
		if sc == nil {
			sc = ch.Background
		}
		if sc == nil {
			continue
		}
		out[sc.ID] = astNode{line: sc.Location.Line}
		for _, ex := range sc.Examples {
			for _, row := range ex.TableBody {
				out[row.ID] = astNode{line: row.Location.Line}
			}
		}
		for _, st := range sc.Steps {
			out[st.ID] = astNode{line: st.Location.Line, keyword: strings.TrimSpace(st.Keyword)}
		}
	}
}

// failureDetail words a failed step the way readiness does: where it is, then the first 20 lines
// of its message.
func failureDetail(where, message string) string {
	lines := strings.Split(message, "\n")
	text := strings.Join(lines[:min(20, len(lines))], "\n")
	if len(lines) > 20 {
		text += fmt.Sprintf("\n[... %d lines omitted ...]", len(lines)-20)
	}
	return verify.Cut(strings.TrimRight(where+"\n"+text, " \t\r\n"))
}

// standInUses reads the control run's log: per pickle and pickle step, whether a stand-in was
// used up to the end of that step (the whole scenario for a step that never ended).
func standInUses(log []byte) (through map[string]map[string]bool, used map[string]bool) {
	through, used = map[string]map[string]bool{}, map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(string(log)))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var r struct{ K, Pickle, Step string }
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		switch r.K {
		case "use":
			used[r.Pickle] = true
		case "afterStep":
			if through[r.Pickle] == nil {
				through[r.Pickle] = map[string]bool{}
			}
			through[r.Pickle][r.Step] = used[r.Pickle]
		}
	}
	return through, used
}

// readControl reads the Cucumber Messages report and the use log of the control run, rooted at
// root; a non-empty reason means the report cannot be read.
func readControl(report, log []byte, root string) (*controlRun, string) {
	var msgs []reportMessage
	for _, line := range strings.Split(string(report), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m reportMessage
		if json.Unmarshal([]byte(line), &m) != nil {
			return nil, "the control report is invalid"
		}
		msgs = append(msgs, m)
	}
	if len(msgs) == 0 {
		return nil, "the control run wrote no report"
	}
	nodes := map[string]astNode{}
	run := &controlRun{defs: map[string]controlDef{}, cases: map[int]*controlCase{}, dups: map[int]bool{}}
	pickles := map[string]*reportMessage{}
	testCases := map[string]*reportTestCase{}
	started := map[string]string{}
	var startedOrder []string
	retried := map[string]bool{}
	results := map[string]map[string]*reportMessage{} // started id -> test step id -> finished message
	for i := range msgs {
		m := &msgs[i]
		switch {
		case m.GherkinDocument != nil && m.GherkinDocument.Feature != nil:
			astNodes(m.GherkinDocument.Feature.Children, nodes)
		case m.Pickle != nil:
			pickles[m.Pickle.ID] = m
		case m.StepDefinition != nil:
			ref := m.StepDefinition.SourceReference
			file := ref.URI
			for _, prefix := range []string{root, resolvedPath(root)} {
				if rel, ok := strings.CutPrefix(file, prefix+string(filepath.Separator)); ok {
					file = rel
				}
			}
			run.defs[m.StepDefinition.ID] = controlDef{pattern: m.StepDefinition.Pattern.Source, file: file, line: ref.Location.Line}
		case m.TestCase != nil:
			testCases[m.TestCase.ID] = m.TestCase
		case m.TestCaseStarted != nil:
			started[m.TestCaseStarted.ID] = m.TestCaseStarted.TestCaseID
			startedOrder = append(startedOrder, m.TestCaseStarted.ID)
		case m.TestCaseFinished != nil && m.TestCaseFinished.WillBeRetried:
			retried[m.TestCaseFinished.TestCaseStartedID] = true
		case m.TestStepFinished != nil:
			f := m.TestStepFinished
			if results[f.TestCaseStartedID] == nil {
				results[f.TestCaseStartedID] = map[string]*reportMessage{}
			}
			results[f.TestCaseStartedID][f.TestStepID] = m
		}
	}
	through, anyUse := standInUses(log)
	for _, sid := range startedOrder {
		if retried[sid] {
			continue
		}
		tc := testCases[started[sid]]
		if tc == nil || pickles[tc.PickleID] == nil {
			return nil, "the control report is invalid"
		}
		pickle := pickles[tc.PickleID].Pickle
		if len(pickle.AstNodeIDs) == 0 {
			return nil, "the control report is invalid"
		}
		caseLine := nodes[pickle.AstNodeIDs[len(pickle.AstNodeIDs)-1]].line
		c := &controlCase{}
		for _, ts := range tc.TestSteps {
			if ts.PickleStepID == "" {
				continue
			}
			var text string
			var node astNode
			for _, ps := range pickle.Steps {
				if ps.ID == ts.PickleStepID && len(ps.AstNodeIDs) > 0 {
					text, node = ps.Text, nodes[ps.AstNodeIDs[0]]
				}
			}
			st := controlStep{line: node.line}
			if len(ts.StepDefinitionIDs) == 1 {
				st.def = ts.StepDefinitionIDs[0]
			}
			if f := results[sid][ts.ID]; f != nil {
				r := f.TestStepFinished.TestStepResult
				st.status = r.Status
				if st.status == "FAILED" {
					message := r.Message
					if message == "" && r.Exception != nil {
						message = r.Exception.Message
					}
					st.detail = failureDetail(fmt.Sprintf("acceptance.feature line %d: %s %s", node.line, node.keyword, text), message)
				}
			}
			if used, ok := through[tc.PickleID][ts.PickleStepID]; ok {
				st.standIn = used
			} else {
				st.standIn = anyUse[tc.PickleID]
			}
			c.steps = append(c.steps, st)
		}
		if _, dup := run.cases[caseLine]; dup {
			run.dups[caseLine] = true
		}
		run.cases[caseLine] = c
	}
	return run, ""
}

// excerpt is the first 200 characters of command output on one line.
func excerpt(output string) string {
	text := []rune(strings.Join(strings.Fields(output), " "))
	if len(text) > 200 {
		text = text[:200]
	}
	return string(text)
}

func resolvedPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// ---- the control run ----

// unpassableControl runs the candidate's acceptance once with Kogen's own Node and Cucumber, in a
// fresh scratch directory of the candidate tree where everything acceptance code obtains from
// outside itself is a stand-in. A non-empty reason means no control could be observed.
func (sh *shaping) unpassableControl(ctx context.Context, ready *verify.Ready) (*controlRun, string, error) {
	node := sh.stepTool(sh.rd.Tools.Node, "KOGEN_NODE")
	cucumber := sh.stepTool(sh.rd.Tools.Cucumber, "KOGEN_CUCUMBER")
	if node == "" {
		return nil, "Node is unavailable", nil
	}
	cli := filepath.Join(cucumber, "@cucumber/cucumber/bin/cucumber.js")
	if cucumber == "" {
		return nil, "Cucumber is unavailable", nil
	}
	if _, err := os.Stat(cli); err != nil {
		return nil, "Cucumber is unavailable", nil
	}
	base := sys.TmpPath("kogen-unpassable")
	defer os.RemoveAll(base)
	tree, kit := filepath.Join(base, "tree"), filepath.Join(base, "kit")
	if err := os.MkdirAll(kit, 0o755); err != nil {
		return nil, "the control directory could not be created", nil
	}
	if _, err := sh.repo.Materialize(ready.CandidateTree, tree); err != nil {
		return nil, "the candidate tree could not be materialized", nil
	}
	seedIgnored(sh.root, tree)
	stepsDir := sh.stepsDir()
	_, files, _ := verify.StepAnalysisIn(tree, stepsDir, ready.Intent, nil)
	if len(files) == 0 {
		return nil, "the candidate has no step files", nil
	}
	preload, support := filepath.Join(kit, "preload.js"), filepath.Join(kit, "support.js")
	log, report := filepath.Join(kit, "uses.ndjson"), filepath.Join(kit, "report.ndjson")
	for path, text := range map[string]string{preload: unpassablePreload, support: unpassableSupport} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return nil, "the control directory could not be written", nil
		}
	}
	command := []string{ShellQuote(node), "-r", ShellQuote(preload), ShellQuote(cli), "--strict", "--require", ShellQuote(support)}
	for _, f := range files {
		command = append(command, "--require", ShellQuote(f))
	}
	command = append(command, "--format", ShellQuote("message:"+report), ShellQuote(project.FeaturePath(sh.slug)))
	res, err := sh.s.RunContext(ctx, strings.Join(command, " "), tree, sys.Options{
		Env:             map[string]string{"NODE_PATH": cucumber, "KOGEN_UNPASSABLE_LOG": log, "KOGEN_UNPASSABLE_STEPS_DIR": stepsDir},
		Timeout:         unpassableBound,
		Context:         ctx,
		KillGroupOnExit: true,
	})
	if err != nil {
		return nil, "", err
	}
	if res.TimedOut {
		return nil, fmt.Sprintf("the control run exceeded the %d-second bound", int(unpassableBound.Seconds())), nil
	}
	text, err := os.ReadFile(report)
	if err != nil {
		return nil, "the control run wrote no report (a step file may not load): " + excerpt(res.Output), nil
	}
	uses, _ := os.ReadFile(log)
	run, reason := readControl(text, uses, tree)
	return run, reason, nil
}

// unpassableGate refuses a ready result in which a step definition can not pass whatever the
// repository contains. It runs only for a result with a failed scenario, records one event and
// refuses only on positive evidence.
func (sh *shaping) unpassableGate(ctx context.Context, ready *verify.Ready) error {
	if !ready.Failures() {
		return nil
	}
	run, reason, err := sh.unpassableControl(ctx, ready)
	if err != nil {
		return err
	}
	var steps []unpassableStep
	if reason == "" {
		steps, reason = judgeUnpassable(ready, run)
	}
	if reason != "" {
		sh.ev.Record(map[string]any{"event": "unpassable_check", "ok": true, "skipped": reason, "steps": []any{}})
		return nil
	}
	e := &unpassableError{}
	list := make([]any, 0, len(steps))
	for _, s := range steps {
		list = append(list, map[string]any{"step": s.step, "definition": s.definition, "pattern": s.pattern, "verdict": s.verdict})
		if s.verdict == verdictUnpassable {
			e.items = append(e.items, s.item())
		}
	}
	sh.ev.Record(map[string]any{"event": "unpassable_check", "ok": len(e.items) == 0, "steps": list})
	if len(e.items) > 0 {
		return e
	}
	return nil
}
