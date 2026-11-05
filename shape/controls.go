package shape

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-loop/sys"
)

// The wrong-implementation check: acceptance can exercise every property the request asks for
// and still pass an implementation that gets the property wrong. For the observable properties
// of the contract review, Kogen runs the candidate's own acceptance in isolated copies, once with
// the correct implementation and once per wrong implementation that the shaper supplies in the
// candidate's controls.json, and requires that each property is rejected by at least one wrong
// implementation at an assertion about it.

//go:embed controls_preload.js
var controlsPreload string

const (
	controlsMaxRuns  = 8
	controlsMaxFiles = 8
	controlsMaxBytes = 65536

	// controlsTimeCap is the total wall time of the runs of one candidate.
	controlsTimeCap = 300 * time.Second
)

// controlsImpl is one wrong implementation of controls.json.
type controlsImpl struct {
	name, scenario, step string
	files                map[string]string
}

// controlsFile is a valid controls.json.
type controlsFile struct {
	correct map[string]string
	wrong   []controlsImpl
}

// controlsRules decide which paths an implementation may write.
type controlsRules struct {
	steps      string   // the configured steps directory; empty when unset
	acceptance []string // the files listed under the Intent's Acceptance
	protect    []string // the configured protect patterns
}

var controlsCucumberConfigs = []string{"cucumber.js", "cucumber.cjs", "cucumber.mjs", "cucumber.json", "cucumber.yaml", "cucumber.yml"}

// controlsGlob turns a protect pattern into an anchored regexp, as the baseline reads them.
func controlsGlob(pat string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pat); {
		switch {
		case strings.HasPrefix(pat[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 3
		case strings.HasPrefix(pat[i:], "**"):
			b.WriteString(".*")
			i += 2
		case pat[i] == '*':
			b.WriteString("[^/]*")
			i++
		case pat[i] == '?':
			b.WriteString("[^/]")
			i++
		default:
			j := i
			for j < len(pat) && pat[j] != '*' && pat[j] != '?' {
				j++
			}
			b.WriteString(regexp.QuoteMeta(pat[i:j]))
			i = j
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// machinery reports whether p, a relative normal path, is acceptance machinery.
func (r controlsRules) machinery(p string) bool {
	if p == ".kogen" || strings.HasPrefix(p, ".kogen/") {
		return true
	}
	if steps := strings.TrimRight(r.steps, "/"); steps != "" && (p == steps || strings.HasPrefix(p, steps+"/")) {
		return true
	}
	if slices.ContainsFunc(r.acceptance, func(a string) bool { return path.Clean(a) == p }) {
		return true
	}
	for _, pat := range r.protect {
		pat = strings.TrimRight(pat, "/")
		if controlsGlob(pat).MatchString(p) || strings.HasPrefix(p, pat+"/") {
			return true
		}
	}
	if slices.ContainsFunc(strings.Split(p, "/"), func(seg string) bool { return seg == ".git" || seg == "node_modules" }) {
		return true
	}
	return slices.Contains(controlsCucumberConfigs, p)
}

// checkFiles validates the "files" of one implementation: its files, and the problem lines.
func (r controlsRules) checkFiles(impl string, v any) (map[string]string, []string) {
	files := map[string]string{}
	m, ok := v.(map[string]any)
	if ok {
		for p, c := range m {
			s, isString := c.(string)
			if !isString {
				ok = false
				break
			}
			files[p] = s
		}
	}
	if !ok || len(files) == 0 {
		return nil, []string{fmt.Sprintf(`- controls.json: %s: "files" must map at least one path to its content`, impl)}
	}
	paths := make([]string, 0, len(files))
	size := 0
	for p, c := range files {
		paths = append(paths, p)
		size += len(c)
	}
	sort.Strings(paths)
	var problems []string
	for _, p := range paths {
		switch {
		case !normalPath(p):
			problems = append(problems, fmt.Sprintf("- controls.json: %s: %s: a replacement path must be relative and normal", impl, normalizeWhitespace(p)))
		case r.machinery(p):
			problems = append(problems, fmt.Sprintf("- controls.json: %s: %s: a replacement may not write acceptance machinery (the .kogen/ and steps directories, the Acceptance: files, protected files, .git, node_modules or a Cucumber configuration file)", impl, p))
		}
	}
	if len(files) > controlsMaxFiles || size > controlsMaxBytes {
		problems = append(problems, fmt.Sprintf("- controls.json: %s: it has more than %d files or %d bytes", impl, controlsMaxFiles, controlsMaxBytes))
	}
	return files, problems
}

// controlsScenarios are the indices of the scenarios that name designates: the scenario of that
// name, or every example of the outline of that name.
func controlsScenarios(scenarios []project.Scenario, name string) []int {
	n := normalizeWhitespace(name)
	var out []int
	for i, sc := range scenarios {
		if normalizeWhitespace(sc.Name) == n || (sc.Outline != nil && normalizeWhitespace(sc.Outline.Name) == n) {
			out = append(out, i)
		}
	}
	return out
}

// controlsStepLines are the feature lines of the steps of sc that step names, as the feature
// writes them (or with an outline's placeholders replaced).
func controlsStepLines(sc project.Scenario, step string, feature []string) []int {
	want := normalizeWhitespace(step)
	var out []int
	for _, st := range sc.Steps {
		written := ""
		if st.Line >= 1 && st.Line <= len(feature) {
			written = normalizeWhitespace(feature[st.Line-1])
		}
		if want != "" && (written == want || normalizeWhitespace(st.Keyword+" "+st.Text) == want) && !slices.Contains(out, st.Line) {
			out = append(out, st.Line)
		}
	}
	return out
}

// parseControls reads controls.json; the problem lines are empty when it is valid.
func parseControls(raw []byte, scenarios []project.Scenario, feature []string, rules controlsRules) (*controlsFile, []string) {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, []string{"- controls.json: it is not valid JSON"}
	}
	top, ok := doc.(map[string]any)
	var wrongList []any
	if ok && exactKeys(top, "correct", "wrong") {
		wrongList, ok = top["wrong"].([]any)
	} else {
		ok = false
	}
	if !ok {
		return nil, []string{`- controls.json: it must be an object with exactly the fields "correct" and "wrong"`}
	}
	cf := &controlsFile{}
	var problems []string
	if correct, ok := top["correct"].(map[string]any); !ok || !exactKeys(correct, "files") {
		problems = append(problems, `- controls.json: "correct" must be an object with exactly the field "files"`)
	} else {
		var ps []string
		cf.correct, ps = rules.checkFiles("the correct implementation", correct["files"])
		problems = append(problems, ps...)
	}
	type entry struct {
		m                    map[string]any
		name, scenario, step string
		ok                   bool
	}
	entries := make([]entry, len(wrongList))
	names := map[string]int{}
	for i, raw := range wrongList {
		e := entry{}
		e.m, _ = raw.(map[string]any)
		if e.m != nil && exactKeys(e.m, "name", "scenario", "step", "files") {
			var n, s, st bool
			e.name, n = e.m["name"].(string)
			e.scenario, s = e.m["scenario"].(string)
			e.step, st = e.m["step"].(string)
			e.ok = n && s && st
		}
		if e.ok {
			names[normalizeWhitespace(e.name)]++
		}
		entries[i] = e
	}
	for i, e := range entries {
		if !e.ok {
			problems = append(problems, fmt.Sprintf(`- controls.json: wrong implementation %d must be an object with exactly the fields "name", "scenario", "step" and "files"`, i+1))
			continue
		}
		name := normalizeWhitespace(e.name)
		label := fmt.Sprintf("wrong implementation \"%s\"", name)
		if name == "" {
			label = fmt.Sprintf("wrong implementation %d", i+1)
			problems = append(problems, fmt.Sprintf("- controls.json: wrong implementation %d needs a name that is not blank and differs from the others", i+1))
		} else if names[name] > 1 {
			problems = append(problems, fmt.Sprintf("- controls.json: wrong implementation %d needs a name that is not blank and differs from the others", i+1))
		}
		files, ps := rules.checkFiles(label, e.m["files"])
		problems = append(problems, ps...)
		scenario := normalizeWhitespace(e.scenario)
		idx := controlsScenarios(scenarios, scenario)
		if len(idx) == 0 {
			problems = append(problems, fmt.Sprintf("- controls.json: %s: scenario \"%s\" is not a scenario of acceptance.feature", label, scenario))
		} else if !slices.ContainsFunc(idx, func(k int) bool { return len(controlsStepLines(scenarios[k], e.step, feature)) > 0 }) {
			problems = append(problems, fmt.Sprintf("- controls.json: %s: step \"%s\" is not a step of scenario \"%s\"", label, normalizeWhitespace(e.step), scenario))
		}
		cf.wrong = append(cf.wrong, controlsImpl{name: name, scenario: scenario, step: normalizeWhitespace(e.step), files: files})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return cf, nil
}

// ---- the checked properties ----

// controlsProp is a checked property: the observable, counted property of the review with
// declared scenarios, and the indices of its scenarios in the feature's order.
type controlsProp struct {
	report    propertyReport
	scenarios []int
}

// checkedControlProps are the checked properties of the last review, in its order.
func (sh *shaping) checkedControlProps(scenarios []project.Scenario) []controlsProp {
	var ev contractEvidence
	for _, sc := range scenarios {
		ev.scenarios = append(ev.scenarios, normalizeWhitespace(sc.Name))
		if sc.Outline != nil {
			ev.scenarios = append(ev.scenarios, normalizeWhitespace(sc.Outline.Name))
		}
	}
	var out []controlsProp
	for _, p := range sh.reviewProps {
		if !p.observable || !propertyCounts(p, sh.request, sh.earlier) {
			continue
		}
		declared := ev.declaredScenarios(p.scenarios)
		if len(declared) == 0 {
			continue
		}
		cp := controlsProp{report: p}
		for i, sc := range scenarios {
			if slices.Contains(declared, normalizeWhitespace(sc.Name)) || (sc.Outline != nil && slices.Contains(declared, normalizeWhitespace(sc.Outline.Name))) {
				cp.scenarios = append(cp.scenarios, i)
			}
		}
		out = append(out, cp)
	}
	return out
}

// ---- the report of a run ----

type controlsStep struct {
	line                 int // the feature line of the step; 0 for a hook
	status               string
	kind, message, stack string
}

type controlsCase struct{ steps []controlsStep }

// status is "passed" when every step passed, "failed" when the first step that did not pass
// failed, else "".
func (c *controlsCase) status() string {
	if c == nil {
		return ""
	}
	if first := c.firstUnpassed(); first != nil {
		if first.status == "FAILED" {
			return "failed"
		}
		return ""
	}
	return "passed"
}

func (c *controlsCase) firstUnpassed() *controlsStep {
	if c == nil {
		return nil
	}
	for i := range c.steps {
		if c.steps[i].status != "PASSED" {
			return &c.steps[i]
		}
	}
	return nil
}

// readControlsReport reads a Cucumber Messages report: the case of each scenario line, and
// whether every scenario of expected is accounted exactly once as passed or failed.
func readControlsReport(report []byte, expected []project.Scenario) (map[int]*controlsCase, bool) {
	var msgs []reportMessage
	for _, line := range strings.Split(string(report), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m reportMessage
		if json.Unmarshal([]byte(line), &m) != nil {
			return nil, false
		}
		msgs = append(msgs, m)
	}
	nodes := map[string]astNode{}
	pickles := map[string]*reportMessage{}
	testCases := map[string]*reportTestCase{}
	started := map[string]string{}
	var order []string
	retried := map[string]bool{}
	results := map[string]map[string]*reportMessage{}
	for i := range msgs {
		m := &msgs[i]
		switch {
		case m.GherkinDocument != nil && m.GherkinDocument.Feature != nil:
			astNodes(m.GherkinDocument.Feature.Children, nodes)
		case m.Pickle != nil:
			pickles[m.Pickle.ID] = m
		case m.TestCase != nil:
			testCases[m.TestCase.ID] = m.TestCase
		case m.TestCaseStarted != nil:
			started[m.TestCaseStarted.ID] = m.TestCaseStarted.TestCaseID
			order = append(order, m.TestCaseStarted.ID)
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
	cases := map[int]*controlsCase{}
	count := map[int]int{}
	for _, sid := range order {
		if retried[sid] {
			continue
		}
		tc := testCases[started[sid]]
		if tc == nil || pickles[tc.PickleID] == nil || len(pickles[tc.PickleID].Pickle.AstNodeIDs) == 0 {
			return nil, false
		}
		pickle := pickles[tc.PickleID].Pickle
		line := nodes[pickle.AstNodeIDs[len(pickle.AstNodeIDs)-1]].line
		c := &controlsCase{}
		for _, ts := range tc.TestSteps {
			st := controlsStep{}
			for _, ps := range pickle.Steps {
				if ts.PickleStepID != "" && ps.ID == ts.PickleStepID && len(ps.AstNodeIDs) > 0 {
					st.line = nodes[ps.AstNodeIDs[0]].line
				}
			}
			if f := results[sid][ts.ID]; f != nil {
				r := f.TestStepFinished.TestStepResult
				st.status = r.Status
				st.message = r.Message
				if r.Exception != nil {
					st.kind, st.stack = r.Exception.Type, r.Exception.StackTrace
					if r.Exception.Message != "" {
						st.message = r.Exception.Message
					}
					if st.stack == "" {
						st.stack = r.Exception.Message
					}
				}
			}
			c.steps = append(c.steps, st)
		}
		cases[line] = c
		count[line]++
	}
	complete := len(cases) == len(expected)
	for _, sc := range expected {
		if count[sc.Line] != 1 || cases[sc.Line].status() == "" {
			complete = false
		}
	}
	return cases, complete
}

// ---- the runs ----

// controlsRunResult is what one run observed.
type controlsRunResult struct {
	cases    map[int]*controlsCase
	complete bool
	cutOff   bool     // the time cap ended the run
	roots    []string // the copy's path as given and with symlinks resolved
}

func controlsEnvPath(p string) string {
	if strings.ContainsAny(p, " \t\"\\") {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p) + `"`
	}
	return p
}

// writeOverlay writes each file of an implementation over the copy, creating missing
// directories; no component of a path may be a symlink.
func writeOverlay(tree string, files map[string]string) error {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		dir := tree
		for _, seg := range strings.Split(path.Dir(p), "/") {
			if seg == "." {
				continue
			}
			dir = filepath.Join(dir, seg)
			if st, err := os.Lstat(dir); err == nil && st.Mode()&fs.ModeSymlink != 0 {
				return fmt.Errorf("%s passes through a symlink", p)
			}
		}
		target := filepath.Join(tree, filepath.FromSlash(p))
		if st, err := os.Lstat(target); err == nil && st.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink", p)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(files[p]), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// controlsRun runs the candidate's acceptance once in a fresh copy of the candidate tree with
// the files of each overlay written over it, in order, before the time cap deadline.
func (sh *shaping) controlsRun(ctx context.Context, scenarios []project.Scenario, deadline time.Time, overlays ...map[string]string) (controlsRunResult, error) {
	var res controlsRunResult
	base := sys.TmpPath("kogen-wrong-implementation")
	defer os.RemoveAll(base)
	tree, kit := filepath.Join(base, "tree"), filepath.Join(base, "kit")
	if err := os.MkdirAll(kit, 0o755); err != nil {
		return res, nil
	}
	preload, report := filepath.Join(kit, "preload.js"), filepath.Join(kit, "report.ndjson")
	if err := os.WriteFile(preload, []byte(controlsPreload), 0o644); err != nil {
		return res, nil
	}
	if _, err := sh.repo.Materialize(sh.ready.CandidateTree, tree); err != nil {
		return res, nil
	}
	res.roots = []string{tree}
	if resolved := resolvedPath(tree); resolved != tree {
		res.roots = append(res.roots, resolved)
	}
	for _, files := range overlays {
		if err := writeOverlay(tree, files); err != nil {
			return res, nil
		}
	}
	env := map[string]string{}
	for k, v := range map[string]string{"KOGEN_NODE": sh.rd.Tools.Node, "KOGEN_QUINT": sh.rd.Tools.Quint, "KOGEN_CUCUMBER": sh.rd.Tools.Cucumber} {
		if v != "" {
			env[k] = v
		}
	}
	// command runs one command in the copy within the time that remains; ran is false when the
	// time cap ended it or left no time to start it.
	command := func(cmd string, extra map[string]string) (status int, ran bool, err error) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			res.cutOff = true
			return 0, false, nil
		}
		opts := sys.Options{Env: map[string]string{}, Timeout: remaining, Context: ctx, KillGroupOnExit: true}
		for k, v := range env {
			opts.Env[k] = v
		}
		for k, v := range extra {
			opts.Env[k] = v
		}
		r, err := sh.s.RunContext(ctx, cmd, tree, opts)
		if err != nil {
			return 0, false, err
		}
		if r.TimedOut {
			res.cutOff = true
			return 0, false, nil
		}
		return r.Status, true, nil
	}
	for _, cmd := range sh.cfg.Setup {
		if status, ran, err := command(cmd, nil); err != nil || !ran || status != 0 {
			return res, err
		}
	}
	if sh.cfg.Acceptance == nil || sh.cfg.Acceptance.Run == "" {
		return res, nil
	}
	nodeOptions := "--require " + controlsEnvPath(preload)
	if existing := sh.s.Getenv("NODE_OPTIONS"); existing != "" {
		nodeOptions += " " + existing
	}
	if _, ran, err := command(sh.cfg.Acceptance.Run, map[string]string{
		"KOGEN_ACCEPTANCE_FEATURES": project.FeaturePath(sh.slug),
		"KOGEN_ACCEPTANCE_REPORT":   report,
		"NODE_OPTIONS":              nodeOptions,
	}); err != nil || !ran {
		return res, err
	}
	text, err := os.ReadFile(report)
	if err != nil {
		return res, nil
	}
	res.cases, res.complete = readControlsReport(text, scenarios)
	return res, nil
}

// ---- the judgement ----

// controlsStepLabel is the step line a case's first unpassed step is at.
func controlsStepLabel(st *controlsStep, feature []string) string {
	if st != nil && st.line >= 1 && st.line <= len(feature) {
		return normalizeWhitespace(feature[st.line-1])
	}
	return "a hook"
}

// controlsFirstLine is the first line of a failure message, normalized.
func controlsFirstLine(message string) string {
	return normalizeWhitespace(strings.SplitN(strings.TrimLeft(message, "\r\n"), "\n", 2)[0])
}

var controlsFrameLocation = regexp.MustCompile(`^(.*):(\d+):(\d+)$`)

// controlsFrame is the location the first stack frame of a stack trace names, relative to the
// copy when it is under it, and whether it is one of the step files.
func controlsFrame(stack string, roots, stepFiles []string) (location string, inStepFile bool) {
	for _, l := range strings.Split(stack, "\n") {
		frame, ok := strings.CutPrefix(strings.TrimLeft(l, " \t"), "at ")
		if !ok {
			continue
		}
		frame = strings.TrimSpace(frame)
		if strings.HasSuffix(frame, ")") {
			if i := strings.LastIndex(frame, " ("); i >= 0 {
				frame = frame[i+2 : len(frame)-1]
			}
		}
		frame = strings.TrimPrefix(frame, "file://")
		file, tail := frame, ""
		if m := controlsFrameLocation.FindStringSubmatch(frame); m != nil {
			file, tail = m[1], ":"+m[2]+":"+m[3]
		}
		for _, root := range roots {
			if rel, ok := strings.CutPrefix(file, root+string(filepath.Separator)); ok {
				file = rel
				break
			}
		}
		return file + tail, slices.Contains(stepFiles, file)
	}
	return "an unknown location", false
}

// controlsJudge decides, for one wrong implementation and the scenario instance sc of it, the
// first rule (2 to 5) that fails and the reason; rule 0 means the run is evidence so far.
func controlsJudge(w controlsImpl, sc project.Scenario, c *controlsCase, stepLines []int, feature []string, run controlsRunResult, stepFiles []string) (int, string) {
	if c.status() != "failed" {
		return 2, fmt.Sprintf("wrong implementation \"%s\" passes scenario \"%s\"", w.name, normalizeWhitespace(sc.Name))
	}
	first := c.firstUnpassed()
	if first.line == 0 || !slices.Contains(stepLines, first.line) {
		return 3, fmt.Sprintf("wrong implementation \"%s\" fails scenario \"%s\" at \"%s\", not at \"%s\"", w.name, normalizeWhitespace(sc.Name), controlsStepLabel(first, feature), w.step)
	}
	if first.kind != "AssertionError" {
		kind := first.kind
		if kind == "" {
			kind = "Error"
		}
		return 4, fmt.Sprintf("wrong implementation \"%s\" fails scenario \"%s\" without an assertion: %s: %s", w.name, normalizeWhitespace(sc.Name), kind, controlsFirstLine(first.message))
	}
	if location, ok := controlsFrame(first.stack, run.roots, stepFiles); !ok {
		return 5, fmt.Sprintf("wrong implementation \"%s\" fails scenario \"%s\" with an assertion thrown outside the Intent's step files, at %s", w.name, normalizeWhitespace(sc.Name), location)
	}
	return 0, ""
}

// controlsPropertyLine is the refusal line of a problem with a property.
func controlsPropertyLine(p propertyReport, reason string) string {
	return fmt.Sprintf("- %s (%s) | request: %s | problem: %s", normalizeWhitespace(p.property), p.kind, normalizeWhitespace(p.request), reason)
}

// controlsJudgeAll are the problem lines of the properties no wrong implementation is evidence
// for, given the complete run results of every implementation (index 0 is the correct one).
func controlsJudgeAll(props []controlsProp, cf *controlsFile, scenarios []project.Scenario, feature []string, runs []controlsRunResult, stepFiles []string) []string {
	var lines []string
	for pi, p := range props {
		inP := map[int]bool{}
		for _, k := range p.scenarios {
			inP[k] = true
		}
		satisfied := false
		var reasons []string
		for wi, w := range cf.wrong {
			var instances []int
			for _, k := range controlsScenarios(scenarios, w.scenario) {
				if inP[k] {
					instances = append(instances, k)
				}
			}
			if len(instances) == 0 {
				continue
			}
			run := runs[wi+1]
			if !run.complete {
				reasons = append(reasons, fmt.Sprintf("wrong implementation \"%s\": its run did not report every scenario as passed or failed", w.name))
				continue
			}
			bestRule, reason := -1, ""
			for _, k := range instances {
				rule, why := controlsJudge(w, scenarios[k], run.cases[scenarios[k].Line], controlsStepLines(scenarios[k], w.step, feature), feature, run, stepFiles)
				if rule == 0 {
					bestRule, reason = 0, ""
					break
				}
				if rule > bestRule {
					bestRule, reason = rule, why
				}
			}
			if bestRule == 0 {
				// Rule 6: every scenario of the other checked properties passed.
				for qi, q := range props {
					if qi == pi {
						continue
					}
					for _, k := range q.scenarios {
						if !inP[k] && run.cases[scenarios[k].Line].status() != "passed" && reason == "" {
							reason = fmt.Sprintf("wrong implementation \"%s\" also fails scenario \"%s\", which exercises another property and not this one", w.name, normalizeWhitespace(scenarios[k].Name))
						}
					}
				}
				if reason == "" {
					satisfied = true
					break
				}
			}
			reasons = append(reasons, reason)
		}
		if satisfied {
			continue
		}
		if len(reasons) == 0 {
			reasons = []string{"no wrong implementation in controls.json names a scenario of it"}
		}
		for _, r := range reasons {
			lines = append(lines, controlsPropertyLine(p.report, r))
		}
	}
	return lines
}

// controlsUnreachable are the problem lines of the checked properties with a scenario that did
// not pass in the correct run.
func controlsUnreachable(props []controlsProp, scenarios []project.Scenario, feature []string, correct controlsRunResult) []string {
	var lines []string
	for _, p := range props {
		reason := ""
		if !correct.complete {
			reason = "no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed"
		} else {
			for _, k := range p.scenarios {
				c := correct.cases[scenarios[k].Line]
				if c.status() == "passed" {
					continue
				}
				first := c.firstUnpassed()
				message := ""
				if first != nil {
					message = controlsFirstLine(first.message)
				}
				reason = fmt.Sprintf("no pass is reachable: with the correct implementation in place, scenario \"%s\" fails at \"%s\": %s",
					normalizeWhitespace(scenarios[k].Name), controlsStepLabel(first, feature), message)
				break
			}
		}
		if reason != "" {
			lines = append(lines, controlsPropertyLine(p.report, reason))
		}
	}
	return lines
}

// ---- the check ----

// wrongHeadline is the first line of the wrong-implementation part of a refusal of n problems.
func wrongHeadline(n int, slug string, seconds int) string {
	return fmt.Sprintf("Wrong-implementation check: %d problem(s) with whether acceptance rejects a wrong implementation of each property the request asks for; nothing was published. "+
		"For each property the request or the answers ask for that a step can observe, %s/controls.json supplies a correct implementation, with which every scenario of the property must pass, and wrong implementations, each naming a scenario and the step of it that must reject it. "+
		"At least one wrong implementation of each property must fail that step with an AssertionError thrown by a step file the Intent lists under Acceptance, after the earlier steps of that scenario passed, while every scenario of the other properties still passes. "+
		"Kogen runs the candidate's acceptance once per implementation in an isolated copy, at most %d runs and %d seconds per candidate. "+
		"Make the steps assert the property, or supply a plausible wrong implementation that only gets the property wrong; resubmitting this candidate unchanged is refused again.",
		n, project.IntentDir(slug), controlsMaxRuns, seconds)
}

// wrongImplementationCheck runs the candidate's acceptance against the implementations of its
// controls.json, for the checked properties of the review. It returns the refusal, empty when
// the candidate may go on. It makes no model request.
func (sh *shaping) wrongImplementationCheck(ctx context.Context, hash string) (contractConfirmed, error) {
	if sh.ready == nil || len(sh.reviewProps) == 0 {
		return contractConfirmed{}, nil
	}
	featureBytes, err := os.ReadFile(filepath.Join(sh.root, project.FeaturePath(sh.slug)))
	if err != nil {
		return contractConfirmed{}, err
	}
	scenarios, err := project.ParseGherkin(string(featureBytes))
	if err != nil {
		return contractConfirmed{}, err
	}
	props := sh.checkedControlProps(scenarios)
	if len(props) == 0 {
		return contractConfirmed{}, nil
	}
	intent, err := project.ReadIntent(sh.root, sh.slug, nil)
	if err != nil {
		return contractConfirmed{}, err
	}
	timeCap := sh.in.WrongImplementationTimeCap
	if timeCap <= 0 {
		timeCap = controlsTimeCap
	}
	seconds := int(timeCap / time.Second)
	feature := strings.Split(string(featureBytes), "\n")
	stepFiles := intent.StepFiles()

	started := 0
	begin := time.Now()
	lines, err := sh.controlsLines(ctx, intent, scenarios, feature, stepFiles, props, timeCap, &started)
	if err != nil {
		return contractConfirmed{}, err
	}
	result := "clean"
	var confirmed contractConfirmed
	if len(lines) > 0 {
		result = "refused"
		confirmed = contractConfirmed{wrong: lines, wrongHeadline: wrongHeadline(len(lines), sh.slug, seconds)}
	}
	recorded := lines
	if recorded == nil {
		recorded = []string{}
	}
	sh.ev.Record(map[string]any{"event": "wrong_implementation_check", "package": hash, "result": result, "runs": started,
		"seconds": int(time.Since(begin).Seconds()), "problems": recorded})
	return confirmed, nil
}

// controlsLines are the problem lines of the wrong-implementation check, nil when it is clean.
func (sh *shaping) controlsLines(ctx context.Context, intent *project.Intent, scenarios []project.Scenario, feature, stepFiles []string, props []controlsProp, timeCap time.Duration, started *int) ([]string, error) {
	rel := project.IntentDir(sh.slug) + "/controls.json"
	raw, err := os.ReadFile(filepath.Join(sh.root, rel))
	if errors.Is(err, fs.ErrNotExist) {
		return []string{fmt.Sprintf(`- controls.json: %s does not exist; write it as {"correct":{"files":{"<path>":"<content>"}},"wrong":[{"name":"<name>","scenario":"<scenario>","step":"<step line>","files":{"<path>":"<content>"}}]}`, rel)}, nil
	}
	if err != nil {
		return nil, err
	}
	rules := controlsRules{acceptance: intent.Acceptance, protect: sh.cfg.Protect}
	if sh.cfg.Acceptance != nil {
		rules.steps = sh.cfg.Acceptance.StepsDir
	}
	cf, problems := parseControls(raw, scenarios, feature, rules)
	if len(problems) > 0 {
		return problems, nil
	}
	n := 1 + len(cf.wrong)
	if n > controlsMaxRuns {
		return []string{fmt.Sprintf("- run cap: this candidate needs %d runs (the correct implementation and %d wrong ones), above the cap of %d runs per candidate; nothing was run. Supply at most %d wrong implementations.",
			n, len(cf.wrong), controlsMaxRuns, controlsMaxRuns-1)}, nil
	}
	seconds := int(timeCap / time.Second)
	deadline := time.Now().Add(timeCap)
	timeLine := func(completed int) []string {
		return []string{fmt.Sprintf("- time cap: the runs exceeded the total time cap of %d seconds per candidate after %d of %d runs completed; the check stopped. Make the implementations or the steps faster.", seconds, completed, n)}
	}
	runs := make([]controlsRunResult, 0, n)
	for i := 0; i < n; i++ {
		if time.Until(deadline) <= 0 {
			return timeLine(i), nil
		}
		impl, overlays := "correct implementation", []map[string]string{cf.correct}
		if i > 0 {
			impl = fmt.Sprintf("wrong implementation \"%s\"", cf.wrong[i-1].name)
			overlays = append(overlays, cf.wrong[i-1].files)
		}
		sh.progress(fmt.Sprintf("kogen: wrong-implementation check: run %d of %d (%s)", i+1, n, impl))
		*started++
		run, err := sh.controlsRun(ctx, scenarios, deadline, overlays...)
		if err != nil {
			return nil, err
		}
		if run.cutOff {
			return timeLine(i), nil
		}
		runs = append(runs, run)
		if i == 0 {
			if lines := controlsUnreachable(props, scenarios, feature, run); len(lines) > 0 {
				return lines, nil
			}
		}
	}
	return controlsJudgeAll(props, cf, scenarios, feature, runs, stepFiles), nil
}
