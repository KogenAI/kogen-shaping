package shape

import (
	"context"
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

// ---- the judgment of readiness results against control observations ----

const (
	judgeFile   = "features/steps/s_steps.js"
	judgeFailed = "acceptance.feature line 3: Then it works"
)

func judgeDetail(line int, message string) *string {
	d := fmt.Sprintf("acceptance.feature line %d: Then it works\n%s", line, message)
	return &d
}

func judgeScenario(line int, status string, detail *string, stepLines ...int) verify.ScenarioResult {
	sc := verify.ScenarioResult{Name: fmt.Sprintf("S%d", line), Line: line, Status: status, Detail: detail}
	for _, l := range stepLines {
		sc.Steps = append(sc.Steps, verify.StepRecord{Keyword: "Then ", Text: "it works", Line: l})
	}
	return sc
}

// judgeCase is a control scenario whose steps all have the given definition.
func judgeCase(def string, steps ...controlStep) *controlCase {
	for i := range steps {
		if steps[i].def == "" {
			steps[i].def = def
		}
	}
	return &controlCase{steps: steps}
}

func judgeRun(cases map[int]*controlCase) *controlRun {
	return &controlRun{
		defs:  map[string]controlDef{"d1": {pattern: "it works", file: judgeFile, line: 5}, "d2": {pattern: "it also works", file: judgeFile, line: 6}},
		cases: cases,
		dups:  map[int]bool{},
	}
}

func controlFailure(line int, message string) controlStep {
	return controlStep{line: line, status: "FAILED", detail: failureDetail(fmt.Sprintf("acceptance.feature line %d: Then it works", line), message)}
}

func controlPassed(line int) controlStep { return controlStep{line: line, status: "PASSED"} }

func TestJudgeUnpassable(t *testing.T) {
	const stack = "\n    at World.<anonymous> (/tmp/scratch-a/features/steps/s_steps.js:5:34)\n    at run (node:internal/x:1:1)"
	const otherStack = "\n    at World.<anonymous> (/private/tmp/scratch-b/features/steps/s_steps.js:5:34)"
	long := strings.Repeat("x", 250)
	tests := []struct {
		name      string
		ready     []verify.ScenarioResult
		run       *controlRun
		verdict   string
		reason    string
		firstLine string
	}{
		{
			name:    "the same failure with other stack frames is unpassable",
			ready:   []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: coverage incomplete"+stack), 3)},
			run:     judgeRun(map[int]*controlCase{2: judgeCase("d1", controlFailure(3, "Error: coverage incomplete"+otherStack))}),
			verdict: verdictUnpassable,
		},
		{
			name:    "a different message is a different failure",
			ready:   []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: coverage incomplete"+stack), 3)},
			run:     judgeRun(map[int]*controlCase{2: judgeCase("d1", controlFailure(3, "Error: other"+stack))}),
			verdict: verdictDifferent,
		},
		{
			name:    "a control that passes the step is a different failure",
			ready:   []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: no"), 3)},
			run:     judgeRun(map[int]*controlCase{2: judgeCase("d1", controlPassed(3))}),
			verdict: verdictDifferent,
		},
		{
			name:    "a control that fails at another step is a different failure",
			ready:   []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(4, "Error: no"), 3, 4)},
			run:     judgeRun(map[int]*controlCase{2: judgeCase("d1", controlStep{def: "d2", line: 3, status: "FAILED", detail: failureDetail("acceptance.feature line 3: Then it works", "Error: no")}, controlStep{line: 4, status: "SKIPPED"})}),
			verdict: verdictDifferent,
		},
		{
			name:  "a stand-in used before the failing step wins over a different failure",
			ready: []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "TypeError: isEven is not a function"), 3)},
			run: func() *controlRun {
				c := controlFailure(3, "Error: stand-in used")
				c.standIn = true
				return judgeRun(map[int]*controlCase{2: judgeCase("d1", c)})
			}(),
			verdict: verdictStandInUsed,
		},
		{
			name:  "a stand-in used with the same failure is still stand_in_used",
			ready: []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: no"), 3)},
			run: func() *controlRun {
				c := controlFailure(3, "Error: no")
				c.standIn = true
				return judgeRun(map[int]*controlCase{2: judgeCase("d1", c)})
			}(),
			verdict: verdictStandInUsed,
		},
		{
			name: "a timeout is never unpassable",
			ready: []verify.ScenarioResult{judgeScenario(2, "failed",
				judgeDetail(3, "Error: function timed out, ensure the promise resolves within 500 milliseconds"+stack), 3)},
			run: judgeRun(map[int]*controlCase{2: judgeCase("d1",
				controlFailure(3, "Error: function timed out, ensure the promise resolves within 500 milliseconds"+stack))}),
			verdict: verdictTimeout,
		},
		{
			name: "a pass in another scenario makes the definition reachable",
			ready: []verify.ScenarioResult{
				judgeScenario(2, "passed", nil, 3),
				judgeScenario(5, "failed", judgeDetail(6, "Error: no"), 6),
			},
			run: judgeRun(map[int]*controlCase{
				2: judgeCase("d1", controlPassed(3)),
				5: judgeCase("d1", controlFailure(6, "Error: no")),
			}),
			verdict: verdictPassed,
		},
		{
			name:  "a step before the failing step passed",
			ready: []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(4, "Error: no"), 3, 4)},
			run: judgeRun(map[int]*controlCase{2: judgeCase("d1", controlPassed(3),
				controlFailure(4, "Error: no"))}),
			verdict: verdictPassed,
		},
		{
			name: "a timeout wins over a pass",
			ready: []verify.ScenarioResult{
				judgeScenario(2, "passed", nil, 3),
				judgeScenario(5, "failed", judgeDetail(6, "Error: function timed out, ensure the promise resolves within 5000 milliseconds"), 6),
			},
			run: judgeRun(map[int]*controlCase{
				2: judgeCase("d1", controlPassed(3)),
				5: judgeCase("d1", controlFailure(6, "Error: function timed out, ensure the promise resolves within 5000 milliseconds")),
			}),
			verdict: verdictTimeout,
		},
		{
			name: "a stand-in use in any failing use outranks a different failure in another",
			ready: []verify.ScenarioResult{
				judgeScenario(2, "failed", judgeDetail(3, "Error: no"), 3),
				judgeScenario(5, "failed", judgeDetail(6, "Error: no"), 6),
			},
			run: func() *controlRun {
				c := controlFailure(6, "Error: no")
				c.standIn = true
				return judgeRun(map[int]*controlCase{2: judgeCase("d1", controlPassed(3)), 5: judgeCase("d1", c)})
			}(),
			verdict: verdictStandInUsed,
		},
		{
			name:  "a scenario that failed in a hook leaves its steps passed",
			ready: []verify.ScenarioResult{judgeScenario(2, "failed", func() *string { d := "a hook at hooks.js line 3\nError: boom"; return &d }(), 3), judgeScenario(5, "failed", judgeDetail(6, "Error: no"), 6)},
			run: judgeRun(map[int]*controlCase{
				2: judgeCase("d1", controlPassed(3)),
				5: judgeCase("d1", controlFailure(6, "Error: no")),
			}),
			verdict: verdictPassed,
		},
		{
			name:      "the message of the item is its first line, cut to 200 characters",
			ready:     []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: "+long+"\nsecond line"), 3)},
			run:       judgeRun(map[int]*controlCase{2: judgeCase("d1", controlFailure(3, "Error: "+long+"\nsecond line"))}),
			verdict:   verdictUnpassable,
			firstLine: "Error: " + long[:193],
		},
		{
			name:   "a scenario missing from the control report cannot be judged",
			ready:  []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: no"), 3)},
			run:    judgeRun(map[int]*controlCase{}),
			reason: "scenario at line 2 is not matched in the control report",
		},
		{
			name:   "a failing step the control report does not have cannot be judged",
			ready:  []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(7, "Error: no"), 3)},
			run:    judgeRun(map[int]*controlCase{2: judgeCase("d1", controlFailure(3, "Error: no"))}),
			reason: "the failing step of the scenario at line 2 is not matched in the control report",
		},
		{
			name:   "a failing step without one definition cannot be judged",
			ready:  []verify.ScenarioResult{judgeScenario(2, "failed", judgeDetail(3, "Error: no"), 3)},
			run:    judgeRun(map[int]*controlCase{2: {steps: []controlStep{controlFailure(3, "Error: no")}}}),
			reason: "step at line 3 has no single definition in the control report",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			steps, reason := judgeUnpassable(&verify.Ready{Scenarios: tc.ready}, tc.run)
			if reason != tc.reason {
				t.Fatalf("reason = %q, want %q", reason, tc.reason)
			}
			if tc.reason != "" {
				return
			}
			if len(steps) != 1 || steps[0].verdict != tc.verdict {
				t.Fatalf("steps = %+v, want one %s", steps, tc.verdict)
			}
			if tc.firstLine != "" && !strings.HasSuffix(steps[0].item(), fmt.Sprintf("it fails with %q whatever the repository contains", tc.firstLine)) {
				t.Fatalf("item = %q, want the message %q", steps[0].item(), tc.firstLine)
			}
		})
	}
}

func TestJudgeUnpassableSteps(t *testing.T) {
	ready := &verify.Ready{Scenarios: []verify.ScenarioResult{
		judgeScenario(2, "failed", judgeDetail(3, "Error: first"), 3),
		judgeScenario(5, "failed", judgeDetail(6, "Error: second"), 6),
		judgeScenario(8, "failed", judgeDetail(9, "Error: first again"), 9),
	}}
	run := judgeRun(map[int]*controlCase{
		2: judgeCase("d2", controlFailure(3, "Error: first")),
		5: judgeCase("d1", controlFailure(6, "Error: second")),
		8: judgeCase("d2", controlFailure(9, "Error: first again")),
	})
	steps, reason := judgeUnpassable(ready, run)
	if reason != "" || len(steps) != 2 {
		t.Fatalf("steps = %+v, reason %q: want one entry per definition", steps, reason)
	}
	want := []unpassableStep{
		{step: judgeFailed, definition: judgeFile + " line 6", pattern: "it also works", verdict: verdictUnpassable, message: "Error: first"},
		{step: "acceptance.feature line 6: Then it works", definition: judgeFile + " line 5", pattern: "it works", verdict: verdictUnpassable, message: "Error: second"},
	}
	for i := range want {
		if steps[i] != want[i] {
			t.Errorf("steps[%d] = %+v, want %+v (ordered by first failing scenario, at its first failing use)", i, steps[i], want[i])
		}
	}
}

func TestUnpassableRefusalText(t *testing.T) {
	e := &unpassableError{items: []string{
		unpassableStep{step: judgeFailed, definition: judgeFile + " line 5", pattern: "it works", message: "Error: coverage incomplete"}.item(),
	}}
	want := "Acceptance has steps that no implementation can pass: 1 step definition(s) fail the same way whatever the repository contains, so no pass is reachable; nothing was published. " +
		"Each step must be able to pass once the requested behaviour exists: run the program, call the code or inspect its output, and compare what it observed with an assertion that shows the actual value (for example assert.strictEqual). " +
		"Rewrite these steps; resubmitting them unchanged is refused again.\n" +
		"- acceptance.feature line 3: Then it works (step definition \"it works\", features/steps/s_steps.js line 5): no pass is reachable: it fails with \"Error: coverage incomplete\" whatever the repository contains"
	shpEq(t, "refusal text", e.text(), want)
}

func TestWithoutFrames(t *testing.T) {
	got := withoutFrames("acceptance.feature line 3: Then x\nError: boom\n    at a (b:1:1)\n\tat c (d:2:2)\nat the end is not a frame start: at x\nattached")
	want := "acceptance.feature line 3: Then x\nError: boom\nattached"
	if got != want {
		t.Fatalf("withoutFrames = %q, want %q", got, want)
	}
}

// ---- the control run with the embedded preload, real Node and Cucumber ----

type controlProbe struct {
	name      string
	body      string
	standIn   bool
	status    string
	detailHas string
}

// unpassableFixture writes a candidate whose steps are the probes, one scenario each, and returns
// the shaping, its readiness result (with every scenario failed) and the scenario lines.
func unpassableFixture(t *testing.T, header string, probes []controlProbe, extra map[string]string) (*shpFx, *shaping, *verify.Ready, map[string]int) {
	t.Helper()
	f := newShpFx(t)
	var feature, steps strings.Builder
	feature.WriteString("Feature: Controls\n")
	steps.WriteString("const {Then} = require('@cucumber/cucumber');\nconst assert = require('node:assert/strict');\nconst fs = require('node:fs');\n" + header)
	for _, p := range probes {
		fmt.Fprintf(&feature, "  Scenario: %s\n    Then step %s\n", p.name, p.name)
		fmt.Fprintf(&steps, "Then('step %s', async function () { %s });\n", p.name, p.body)
	}
	f.write(".kogen/intents/is-even/intent.md", shpIntentMD)
	f.write(project.FeaturePath(shpSlug), feature.String())
	f.write("features/steps/is_even_steps.js", steps.String())
	f.write("calc.js", "exports.add = (a, b) => a + b;\n")
	f.write("features/steps/calc_steps.js", "// existing steps\n")
	f.write("features/steps/helper.js", "exports.answer = 42;\n")
	for p, text := range extra {
		f.write(p, text)
	}
	intent, scenarios, err := project.CheckIntent(f.Root, shpSlug, nil)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := f.Repo.Tree(f.Repo.Head(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ready := &verify.Ready{Intent: intent, CandidateTree: tree}
	lines := map[string]int{}
	for _, sc := range scenarios {
		lines[sc.Name] = sc.Line
		r := verify.ScenarioRecord(sc)
		r.Status = "failed"
		ready.Scenarios = append(ready.Scenarios, r)
	}
	sh := f.shaping()
	sh.rd = &verify.Readiness{}
	return f, sh, ready, lines
}

func TestUnpassableControlStandIns(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	probes := []controlProbe{
		{name: "throws", body: "throw new Error('stub');", status: "FAILED", detailHas: "Error: stub"},
		{name: "read unused", body: "fs.readFileSync('README.md', 'utf8'); throw new Error('stub');", status: "FAILED", detailHas: "Error: stub"},
		{name: "read used", body: "fs.readFileSync('README.md', 'utf8').trim(); throw new Error('stub');", standIn: true, status: "FAILED"},
		{name: "missing file unused", body: "fs.readFileSync('missing.txt'); assert.ok(false);", status: "FAILED", detailHas: "AssertionError"},
		{name: "missing file listed", body: "Object.keys(fs.readdirSync('missing'));", standIn: true, status: "FAILED"},
		{name: "missing file opened", body: "assert.ok(fs.openSync('missing.txt') > 0);", standIn: true, status: "FAILED"},
		{name: "files globbed", body: "assert.strictEqual(fs.globSync('*.txt').length, 1);", standIn: true, status: "FAILED"},
		{name: "module used", body: "require('../../calc').isEven(2);", standIn: true, status: "FAILED"},
		{name: "module destructured while loading", body: "isEven(2);", standIn: true, status: "FAILED"},
		{name: "module unresolvable unused", body: "require('no-such-module-xyz'); throw new Error('stub');", status: "FAILED", detailHas: "Error: stub"},
		{name: "module unresolvable used", body: "require('no-such-module-xyz').run();", standIn: true, status: "FAILED"},
		{name: "module awaited", body: "await require('../../calc');", standIn: true, status: "FAILED"},
		{name: "module converted", body: "String(require('../../calc'));", standIn: true, status: "FAILED"},
		{name: "module member checked", body: "'add' in require('../../calc');", standIn: true, status: "FAILED"},
		{name: "module constructed", body: "new (require('../../calc'))();", standIn: true, status: "FAILED"},
		{name: "exists", body: "fs.existsSync('README.md'); throw new Error('stub');", standIn: true, status: "FAILED"},
		{name: "access", body: "try { fs.accessSync('README.md'); } catch (e) {} throw new Error('stub');", standIn: true, status: "FAILED"},
		{name: "asynchronous call", body: "await fs.promises.readFile('README.md'); throw new Error('stub');", standIn: true, status: "FAILED"},
		{name: "environment", body: "process.env.HOME; throw new Error('stub');", standIn: true, status: "FAILED"},
		{name: "network fetch", body: "await fetch('http://127.0.0.1:9/');", standIn: true, status: "FAILED"},
		{name: "network socket", body: "require('node:net').connect(9, '127.0.0.1').unref();", standIn: true, status: "FAILED"},
		{name: "network request", body: "require('node:http').get('http://127.0.0.1:9/').on('error', () => {});", standIn: true, status: "FAILED"},
		{name: "network socket unused", body: "require('node:net').connect(9, '127.0.0.1'); throw new Error('stub');", status: "FAILED", detailHas: "Error: stub"},
		{name: "promisified process", body: "await require('node:util').promisify(require('node:child_process').execFile)('touch', [" + fmt.Sprintf("%q", marker) + "]);", status: "FAILED", detailHas: "function timed out"},
		{name: "by other code", body: "assert.strictEqual(require('../../calc'), 7);", status: "FAILED", detailHas: "AssertionError"},
		{name: "process", body: "require('node:child_process').spawnSync('touch', [" + fmt.Sprintf("%q", marker) + "]); throw new Error('stub');", status: "FAILED", detailHas: "Error: stub"},
		{name: "steps folder", body: "assert.strictEqual(require('./helper').answer, 42);", status: "PASSED"},
	}
	_, sh, ready, lines := unpassableFixture(t, "const {isEven} = require('../../calc');\nrequire('@cucumber/cucumber').setDefaultTimeout(1000);\n", probes, nil)
	run, reason, err := sh.unpassableControl(context.Background(), ready)
	if err != nil || reason != "" {
		t.Fatalf("control run: reason %q, err %v", reason, err)
	}
	for _, p := range probes {
		c := run.caseAt(lines[p.name])
		if c == nil || len(c.steps) != 1 {
			t.Errorf("%s: no control scenario at line %d", p.name, lines[p.name])
			continue
		}
		st := c.steps[0]
		if st.status != p.status || st.standIn != p.standIn || !strings.Contains(st.detail, p.detailHas) {
			t.Errorf("%s: status %s, stand-in used %v, detail %q; want %s, %v, containing %q", p.name, st.status, st.standIn, st.detail, p.status, p.standIn, p.detailHas)
		}
		if d := run.defs[st.def]; d.file != "features/steps/is_even_steps.js" || d.pattern != "step "+p.name {
			t.Errorf("%s: definition %+v", p.name, d)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Errorf("the control run started a process")
	}
}

func TestUnpassableControlUsesConfiguredStepFolder(t *testing.T) {
	const folder = "test/intents/steps"
	const stepPath = folder + "/is_even_steps.js"
	marker := filepath.Join(t.TempDir(), "started")
	steps := fmt.Sprintf(`const { Then } = require('@cucumber/cucumber');
Then('step process', function () {
  require('node:child_process').spawnSync('touch', [%q]);
  throw new Error('stub');
});
`, marker)
	_, sh, ready, lines := unpassableFixture(t, "", []controlProbe{{name: "process"}}, map[string]string{
		project.ConfigPath:            strings.ReplaceAll(shpGuardConfig, "features/steps", folder),
		project.IntentMDPath(shpSlug): strings.ReplaceAll(shpIntentMD, "features/steps", folder),
		stepPath:                      steps,
	})
	run, reason, err := sh.unpassableControl(context.Background(), ready)
	if err != nil || reason != "" {
		t.Fatalf("control run: reason %q, err %v", reason, err)
	}
	scenario := run.caseAt(lines["process"])
	if scenario == nil || len(scenario.steps) != 1 || run.defs[scenario.steps[0].def].file != stepPath || scenario.steps[0].status != "FAILED" {
		t.Fatalf("custom definition did not run in the control: %+v", scenario)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("control did not isolate the custom step's child process: %v", err)
	}
}

func TestUnpassableControlSkipped(t *testing.T) {
	stub := []controlProbe{{name: "stub", body: "throw new Error('stub');"}}
	t.Run("Node is unavailable", func(t *testing.T) {
		f, sh, ready, _ := unpassableFixture(t, "", stub, nil)
		var env []string
		for _, kv := range f.Env {
			if !strings.HasPrefix(kv, "KOGEN_NODE=") {
				env = append(env, kv)
			}
		}
		sh.s = sys.NewWith(env, f.Root)
		_, reason, err := sh.unpassableControl(context.Background(), ready)
		if err != nil || reason != "Node is unavailable" {
			t.Fatalf("reason %q, err %v", reason, err)
		}
	})
	t.Run("Cucumber is unavailable", func(t *testing.T) {
		_, sh, ready, _ := unpassableFixture(t, "", stub, nil)
		sh.rd.Tools.Cucumber = t.TempDir()
		_, reason, err := sh.unpassableControl(context.Background(), ready)
		if err != nil || reason != "Cucumber is unavailable" {
			t.Fatalf("reason %q, err %v", reason, err)
		}
	})
	t.Run("a step file does not load", func(t *testing.T) {
		_, sh, ready, _ := unpassableFixture(t, "throw new Error('does not load');\n", stub, nil)
		_, reason, err := sh.unpassableControl(context.Background(), ready)
		if err != nil || !strings.HasPrefix(reason, "the control run wrote no report") {
			t.Fatalf("reason %q, err %v", reason, err)
		}
	})
	t.Run("the bound is exceeded", func(t *testing.T) {
		bound := unpassableBound
		unpassableBound = 2 * time.Second
		t.Cleanup(func() { unpassableBound = bound })
		_, sh, ready, _ := unpassableFixture(t, "", []controlProbe{{name: "slow", body: "await new Promise(r => setTimeout(r, 60000));"}}, nil)
		start := time.Now()
		_, reason, err := sh.unpassableControl(context.Background(), ready)
		if err != nil || reason != "the control run exceeded the 2-second bound" {
			t.Fatalf("reason %q, err %v", reason, err)
		}
		if time.Since(start) > 30*time.Second {
			t.Fatalf("the control run was not stopped at its bound: %v", time.Since(start))
		}
	})
	t.Run("the gate refuses nothing and records skipped", func(t *testing.T) {
		_, sh, ready, _ := unpassableFixture(t, "throw new Error('does not load');\n", stub, nil)
		if err := sh.unpassableGate(context.Background(), ready); err != nil {
			t.Fatalf("gate = %v, want no refusal", err)
		}
		f := &shpFx{t: t, Root: sh.root}
		events := shpNamed(f.events(), "unpassable_check")
		if len(events) != 1 || events[0]["ok"] != true || !strings.HasPrefix(fmt.Sprint(events[0]["skipped"]), "the control run wrote no report") {
			t.Fatalf("events = %v", events)
		}
		if steps, ok := events[0]["steps"].([]any); !ok || len(steps) != 0 {
			t.Fatalf("steps = %v, want an empty list", events[0]["steps"])
		}
	})
}

func TestUnpassableGate(t *testing.T) {
	t.Run("a result without a failed scenario records nothing", func(t *testing.T) {
		_, sh, ready, _ := unpassableFixture(t, "", []controlProbe{{name: "stub", body: "throw new Error('stub');"}}, nil)
		ready.Scenarios[0].Status = "passed"
		if err := sh.unpassableGate(context.Background(), ready); err != nil {
			t.Fatal(err)
		}
		events, _ := os.ReadFile(filepath.Join(sh.root, ".kogen/local/shaping", shpSlug, "events.ndjson"))
		if strings.Contains(string(events), "unpassable_check") {
			t.Fatalf("the check recorded an event: %s", events)
		}
	})
	t.Run("an unpassable step is refused and a reachable one is not", func(t *testing.T) {
		probes := []controlProbe{
			{name: "stub", body: "throw new Error('coverage incomplete');"},
			{name: "calls", body: "require('../../calc').isEven(2);"},
		}
		_, sh, ready, lines := unpassableFixture(t, "", probes, nil)
		for i := range ready.Scenarios {
			// The readiness failures the same steps give when the feature is missing.
			message := "Error: coverage incomplete\n    at World.<anonymous> (/elsewhere/features/steps/is_even_steps.js:5:34)"
			if ready.Scenarios[i].Line == lines["calls"] {
				message = "TypeError: require(...).isEven is not a function"
			}
			d := fmt.Sprintf("acceptance.feature line %d: Then step %s\n%s", ready.Scenarios[i].Steps[0].Line, ready.Scenarios[i].Name, message)
			ready.Scenarios[i].Detail = &d
		}
		err := sh.unpassableGate(context.Background(), ready)
		refusal, ok := err.(*unpassableError)
		if !ok || len(refusal.items) != 1 || !strings.HasPrefix(refusal.items[0], fmt.Sprintf("- acceptance.feature line %d: Then step stub (step definition \"step stub\", features/steps/is_even_steps.js line ", lines["stub"]+1)) {
			t.Fatalf("gate = %#v, want a refusal of the stub only", err)
		}
		f := &shpFx{t: t, Root: sh.root}
		events := shpNamed(f.events(), "unpassable_check")
		if len(events) != 1 || events[0]["ok"] != false {
			t.Fatalf("events = %v", events)
		}
		verdicts := map[string]string{}
		for _, raw := range events[0]["steps"].([]any) {
			m := raw.(map[string]any)
			verdicts[fmt.Sprint(m["pattern"])] = fmt.Sprint(m["verdict"])
			if !strings.HasPrefix(fmt.Sprint(m["definition"]), "features/steps/is_even_steps.js line ") {
				t.Errorf("definition = %v", m["definition"])
			}
		}
		if verdicts["step stub"] != verdictUnpassable || verdicts["step calls"] != verdictStandInUsed || len(verdicts) != 2 {
			t.Fatalf("verdicts = %v", verdicts)
		}
	})
}
