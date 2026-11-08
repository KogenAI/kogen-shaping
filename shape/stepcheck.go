package shape

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-verification/verify"
)

// The step self-check: before readiness, Kogen checks the step code of the candidate package
// with its own Node and Cucumber, so a broken step file comes back as feedback instead of
// wasting a readiness run or a Build.

const selfcheckHeadline = "Step self-check failed; nothing was published. Fix the step files and resubmit."

// selfcheckScenarioFeedback identifies unbound steps at their feature locations, so the
// shaper receives the same actionable diagnostics as readiness. matches holds, for each
// ambiguous step text, the definitions Cucumber's matcher matches for it in load order; an
// ambiguous step without them keeps the location-less line. A step of an expanded Outline
// example names its Examples row line and is reported once per example; a plain step once.
func selfcheckScenarioFeedback(items []verify.StepItem, scenarios []project.Scenario, matches map[string][]verify.StepDef) string {
	type instance struct {
		step project.Step
		row  int
	}
	var lines []string
	seen := map[instance]bool{}
	for _, it := range items {
		if it.OK {
			continue
		}
		ambiguous := it.Check == "steps" && strings.HasPrefix(it.Detail, "ambiguous step") && len(matches[it.Target]) > 0
		if it.Check == "steps" && (it.Detail == "undefined step" || ambiguous) {
			for _, sc := range scenarios {
				for _, st := range sc.Steps {
					if st.Text != it.Target {
						continue
					}
					in, row := instance{step: st}, ""
					if sc.Outline != nil {
						in.row, row = sc.Line, fmt.Sprintf(" (Examples row line %d)", sc.Line)
					}
					if seen[in] {
						continue
					}
					seen[in] = true
					if !ambiguous {
						lines = append(lines, fmt.Sprintf("acceptance.feature line %d: undefined step: %s%s%s", st.Line, st.Keyword, st.Text, row))
						continue
					}
					defs := make([]string, len(matches[st.Text]))
					for i, d := range matches[st.Text] {
						defs[i] = d.Pattern + " in " + d.File
					}
					lines = append(lines, fmt.Sprintf("acceptance.feature line %d: ambiguous step: %s%s%s matches %d definitions: %s",
						st.Line, st.Keyword, st.Text, row, len(defs), strings.Join(defs, "; ")))
				}
			}
		} else {
			lines = append(lines, "- "+it.Check+" "+it.Target+": "+it.Detail)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return selfcheckHeadline + "\n" + strings.Join(lines, "\n")
}

// stepSelfcheck runs the self-check on the written candidate and records its one event. It
// returns the feedback text, empty when no item is a defect. Unused definitions of the
// Intent's own step files are warnings: the event lists them with their definition pattern and
// file, and they never make feedback.
func (sh *shaping) stepSelfcheck(ctx context.Context) (string, error) {
	intent, scenarios, err := project.CheckIntent(sh.root, sh.slug, nil)
	if err != nil {
		return "", nil // the package is not readable; readiness reports why
	}
	own, files, texts := verify.StepAnalysis(sh.root, intent, scenarios)

	var items []verify.StepItem
	matches := map[string][]verify.StepDef{}
	if len(own) > 0 {
		node, cucumber := sh.stepTool(sh.rd.Tools.Node, "KOGEN_NODE"), sh.stepTool(sh.rd.Tools.Cucumber, "KOGEN_CUCUMBER")
		raw, _, err := verify.RunStepScript(ctx, node, cucumber, sh.root, sh.s.Env, files, texts)
		if err != nil {
			return "", err
		}
		items = verify.StepItems(own, raw)
		// Cucumber's own matcher names the definitions of an ambiguous step: run it again for
		// that one text; the definitions it marks used are exactly those that match.
		for _, it := range items {
			if it.Check != "steps" || it.OK || !strings.HasPrefix(it.Detail, "ambiguous step") {
				continue
			}
			one, _, err := verify.RunStepScript(ctx, node, cucumber, sh.root, sh.s.Env, files, []string{it.Target})
			if err != nil {
				return "", err
			}
			for _, d := range one.Defs {
				if d.Used {
					matches[it.Target] = append(matches[it.Target], d)
				}
			}
		}
		for _, sc := range scenarios {
			for _, st := range sc.Steps {
				if !strings.HasPrefix(st.Text, sh.slug+": ") {
					items = append(items, verify.StepItem{Check: "ownership", Target: st.Text, Detail: "step sentence must start with " + sh.slug + ": "})
				}
			}
		}
		for _, d := range raw.Defs {
			if d.Used && !slices.Contains(own, d.File) {
				items = append(items, verify.StepItem{Check: "ownership", Target: d.Pattern, File: d.File, Detail: "binding comes from a step file outside this Intent: " + d.File})
			}
			if slices.Contains(own, d.File) && !strings.HasPrefix(strings.TrimPrefix(d.Pattern, "^"), sh.slug+": ") {
				items = append(items, verify.StepItem{Check: "ownership", Target: d.Pattern, File: d.File, Detail: "definition must start with " + sh.slug + ": "})
			}
		}
		for _, d := range raw.Defs {
			if d.Used && slices.Contains(own, d.File) {
				if it, refused := stepLimitItem(d); refused {
					items = append(items, it)
				}
			}
		}
	}
	obs, err := sh.observerItems(ctx, intent)
	if err != nil {
		return "", err
	}
	items = append(items, obs...)

	all, ok := make([]any, 0, len(items)), true
	for _, it := range items {
		m := map[string]any{"check": it.Check, "target": it.Target, "ok": it.OK, "detail": it.Detail}
		if it.File != "" {
			m["file"] = it.File
		}
		if it.Warning {
			m["warning"] = true
		}
		all = append(all, m)
		ok = ok && it.OK
	}
	event := map[string]any{"event": "step_selfcheck", "ok": ok, "items": all}
	if warnings := verify.StepWarnings(items); len(warnings) > 0 {
		list := make([]any, len(warnings))
		for i, w := range warnings {
			list[i] = map[string]any{"pattern": w.Pattern, "file": w.File, "detail": w.Detail}
		}
		event["warnings"] = list
	}
	sh.ev.Record(event)
	return selfcheckScenarioFeedback(items, scenarios, matches), nil
}

// stepTool is a tool path of the readiness tools, else the environment variable.
func (sh *shaping) stepTool(configured, env string) string {
	if configured != "" {
		return configured
	}
	return sh.s.Getenv(env)
}

// observerItems re-runs the controls of every declared observer and compares each exit code
// with the recorded one.
func (sh *shaping) observerItems(ctx context.Context, intent *project.Intent) ([]verify.StepItem, error) {
	decls, err := verify.Declarations(intent)
	if err != nil {
		return nil, nil // invalid declarations are readiness's to report
	}
	redactor := verify.NewRedactor(sh.s.Env, sh.s.Getenv("HOME"))
	var items []verify.StepItem
	for _, d := range decls {
		if d.Kind != "observer" {
			continue
		}
		for _, c := range d.Checks {
			r, err := sh.ev.Probe(func() (verify.ProbeResult, error) {
				r, err := (verify.Probe{Root: sh.root, Command: "{\n" + c.Command + "\n} >/dev/null", Tools: sh.rd.Tools}).Run(ctx, sh.s)
				r.Output = redactor.Redact(r.Output)
				return r, err
			})
			if err != nil {
				return nil, err
			}
			it := verify.StepItem{Check: "observer", Target: d.ID + " " + c.Name, OK: r.Status == c.Exit && !r.TimedOut}
			if !it.OK {
				it.Detail = selfcheckObserverDetail(r.Status, c.Exit, r.Output)
				if r.TimedOut {
					it.Detail += fmt.Sprintf(", timed out: %s/%s (%v) exceeds the %d-second limit", d.ID, c.Name, d.Observers, int(verify.ProbeTimeout.Seconds()))
				}
			}
			items = append(items, it)
		}
	}
	return items, nil
}

// selfcheckStderrTail bounds an already-redacted diagnostic by bytes.
func selfcheckStderrTail(stderr string) string {
	if len(stderr) > 4096 {
		tail := stderr[len(stderr)-4096:]
		for len(tail) > 0 && !utf8.RuneStart(tail[0]) {
			tail = tail[1:]
		}
		return tail
	}
	return stderr
}

func selfcheckObserverDetail(actual, expected int, stderr string) string {
	detail := fmt.Sprintf("exit %d, recorded exit %d", actual, expected)
	if tail := selfcheckStderrTail(stderr); tail != "" {
		detail += "\nstderr: " + tail
	}
	return detail
}
