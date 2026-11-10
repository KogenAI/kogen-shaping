package shape

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
	own, files, texts := verify.StepAnalysisIn(sh.root, sh.stepsDir(), intent, scenarios)

	var items []verify.StepItem
	matches := map[string][]verify.StepDef{}
	quoteHint := false
	specialHint := false
	if len(own) > 0 {
		node, cucumber := sh.stepTool(sh.rd.Tools.Node, "KOGEN_NODE"), sh.stepTool(sh.rd.Tools.Cucumber, "KOGEN_CUCUMBER")
		raw, _, err := verify.RunStepScript(ctx, node, cucumber, sh.root, sh.s.Env, files, texts)
		if err != nil {
			return "", err
		}
		items = verify.StepItems(own, raw)
		for _, it := range items {
			if it.Check == "steps" && it.Detail == "undefined step" && unescapedStringArgument(it.Target, raw.Defs) {
				quoteHint = true
				break
			}
		}
		var undefined []string
		for _, it := range items {
			if it.Check == "steps" && it.Detail == "undefined step" {
				undefined = append(undefined, it.Target)
			}
		}
		if len(undefined) > 0 {
			specialHint = literalExpressionHint(ctx, node, cucumber, sh.root, sh.s.Env, files, own, undefined)
		}
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
	feedback := selfcheckScenarioFeedback(items, scenarios, matches)
	if quoteHint {
		feedback += "\n- Escape inner quotes inside the {string} argument (including quotes in an Examples cell), or use a DocString to preserve raw fixture bytes."
	}
	if specialHint {
		feedback += `
- Cucumber Expressions read / as alternation, ( ) as optional text and { } as parameters; escape literal characters (in JavaScript strings use \\/, \\( and \\)) or rephrase the definition.`
	}
	return feedback, nil
}

// Inspect the registered source patterns before finalize: Cucumber omits definitions with
// unknown parameter types from its finalized library. Use its expression parser and registry
// for the hypothetical literal interpretation, without altering verification's matcher.
const literalExpressionScript = `
const fs = require('fs'), path = require('path');
const cc = require('@cucumber/cucumber');
const { CucumberExpression } = require('@cucumber/cucumber-expressions');
const inp = JSON.parse(fs.readFileSync(0, 'utf8'));
const b = cc.supportCodeLibraryBuilder;
let n = 0;
b.reset(process.cwd(), () => String(n++));
for (const f of inp.files) { try { require(path.resolve(f)); } catch (e) {} }
const registry = b.parameterTypeRegistry;
let found = false;
for (const d of b.stepDefinitionConfigs) {
  // Cucumber's stack-based URI extraction can strip parentheses from the checkout path.
  if (typeof d.pattern !== 'string' || !inp.own.some(f => {
    const absolute = path.resolve(f), uri = path.resolve(d.uri);
    return absolute === uri || absolute.replace(/[()]/g, '') === uri;
  })) continue;
  let p = '', changed = false;
  for (let i = 0; i < d.pattern.length; i++) {
    const c = d.pattern[i];
    if (c === '\\') { p += c + (d.pattern[++i] || ''); continue; }
    if (c === '{') {
      const end = d.pattern.indexOf('}', i + 1);
      if (end >= 0 && registry.lookupByTypeName(d.pattern.slice(i + 1, end))) {
        p += d.pattern.slice(i, end + 1); i = end; continue;
      }
    }
    if ('/(){'.includes(c)) { p += '\\'; changed = true; }
    // A closing brace outside a parameter must also be escaped for the parser.
    if (c === '}') p += '\\';
    p += c;
  }
  if (!changed) continue;
  try {
    const expr = new CucumberExpression(p, registry);
    if (inp.texts.some(t => expr.match(t) !== null)) { found = true; break; }
  } catch (e) {}
}
process.stdout.write('\nKOGEN_LITERAL_HINT:' + JSON.stringify(found) + '\n');
`

func literalExpressionHint(ctx context.Context, node, cucumber, root string, env, files, own, texts []string) bool {
	in, _ := json.Marshal(map[string]any{"files": files, "own": own, "texts": texts})
	cmd := exec.CommandContext(ctx, node, "-e", literalExpressionScript)
	cmd.Dir = root
	cmd.Env = append(slices.Clone(env), "NODE_PATH="+cucumber)
	cmd.Stdin = bytes.NewReader(in)
	out, _ := cmd.Output()
	for _, line := range strings.Split(string(out), "\n") {
		if line == "KOGEN_LITERAL_HINT:true" {
			return true
		}
	}
	return false
}

// unescapedStringArgument recognizes the narrow case where a matching literal
// prefix and suffix surround a double-quoted argument with raw inner quotes.
func unescapedStringArgument(text string, defs []verify.StepDef) bool {
	for _, d := range defs {
		prefix, suffix, found := strings.Cut(d.Pattern, "{string}")
		if !found || strings.Contains(suffix, "{string}") {
			continue
		}
		value, starts := strings.CutPrefix(text, prefix)
		value, ends := strings.CutSuffix(value, suffix)
		if starts && ends && len(value) > 2 && value[0] == '"' && value[len(value)-1] == '"' &&
			strings.Contains(strings.ReplaceAll(value[1:len(value)-1], `\"`, ""), `"`) {
			return true
		}
	}
	return false
}

func (sh *shaping) stepsDir() string {
	if sh.cfg != nil && sh.cfg.Acceptance != nil && sh.cfg.Acceptance.StepsDir != "" {
		return sh.cfg.Acceptance.StepsDir
	}
	return strings.TrimSuffix(verify.StepsFolder, "/")
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
