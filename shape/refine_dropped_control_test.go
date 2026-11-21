package shape

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// The fixture draft has two working wrong-implementation controls, run by the real Cucumber runner:
// wrong-even in scenario "Even numbers" and wrong-odd in scenario "Odd numbers".

const rdcSteps = "const { Given, Then } = require('@cucumber/cucumber');\nconst assert = require('node:assert');\nGiven('is-even: the number {int}', function (n) { this.n = n; });\nThen('is-even: it is even', function () { assert.strictEqual(require('../../calc.js').isEven(this.n), true); });\nThen('is-even: it is not even', function () { assert.strictEqual(require('../../calc.js').isEven(this.n), false); });\n"

const rdcGood = "exports.isEven = n => n % 2 === 0;\n"

type rdcWrong struct{ name, scenario, step, calc string }

var (
	rdcEven = rdcWrong{"wrong-even", "Even numbers", "Then is-even: it is even", "exports.isEven = n => false;\n"}
	rdcOdd  = rdcWrong{"wrong-odd", "Odd numbers", "Then is-even: it is not even", "exports.isEven = n => true;\n"}
)

func rdcControls(wrong ...rdcWrong) string {
	var list []any
	for _, w := range wrong {
		list = append(list, map[string]any{"name": w.name, "scenario": w.scenario, "step": w.step, "files": map[string]any{"calc.js": w.calc}})
	}
	b, err := json.Marshal(map[string]any{"correct": map[string]any{"files": map[string]any{"calc.js": rdcGood}}, "wrong": list})
	if err != nil {
		panic(err)
	}
	return string(b) + "\n"
}

// rdcMerged is a control of the odd scenario that supersedes the named original for the reason.
func rdcMerged(original, reason string) rdcWrong {
	return rdcWrong{`wrong-merged supersedes "` + original + `" of "Odd numbers" because ` + reason, "Odd numbers", "Then is-even: it is not even", "exports.isEven = () => true;\n"}
}

// rdcRefine refines a committed draft holding both controls with the replacement and returns the
// controls file after, the one before, the error and the feedback the model got for its submission.
func rdcRefine(t *testing.T, replacement string) (after, before string, err error, feedback string) {
	t.Helper()
	var steps []shpStep
	for i := 1; i <= 3; i++ {
		steps = append(steps, refSubmit(fmt.Sprintf("c%d", i), refControlsPath, replacement))
	}
	f := newRefFx(t, refOpts{}, steps...)
	node, modules := f.S.Getenv("KOGEN_NODE"), f.S.Getenv("KOGEN_CUCUMBER")
	f.write("accept.sh", fmt.Sprintf("#!/bin/sh\nNODE_PATH='%s' '%s' '%s' --strict --require 'features/steps/*.js' --format \"message:$KOGEN_ACCEPTANCE_REPORT\" \"$KOGEN_ACCEPTANCE_FEATURES\"\n",
		modules, node, filepath.Join(modules, "@cucumber/cucumber/bin/cucumber.js")))
	f.write("features/steps/calc_steps.js", "// no steps\n")
	f.write("calc.js", rdcGood)
	f.write(refStepPath, rdcSteps)
	f.write(refControlsPath, rdcControls(rdcEven, rdcOdd))
	f.git("add", "-A")
	f.git("commit", "-qm", "fixture with two working controls")
	before = f.read(refControlsPath)
	_, err = f.refine(f.hash())
	after = f.read(refControlsPath)
	if reqs := f.Prov.requests(); len(reqs) > 1 {
		feedback = shpOutputs(reqs[1])["c1"]
	}
	return after, before, err, feedback
}

func TestRefineRejectsReplacementDroppingControl(t *testing.T) {
	parallel(t)
	after, before, err, feedback := rdcRefine(t, rdcControls(rdcEven))
	if err == nil {
		t.Fatal("a replacement dropping wrong-odd was accepted")
	}
	refText(t, "controls", after, before)
	for _, want := range []string{"wrong-odd", "Odd numbers", "supersedes", "because"} {
		shpContains(t, "feedback", feedback, want)
	}
	if strings.Contains(feedback, `"wrong-even"`) {
		t.Fatalf("the kept control is named as missing: %s", feedback)
	}
}

func TestRefineAcceptsSupersededControlWithReason(t *testing.T) {
	parallel(t)
	replacement := rdcControls(rdcEven, rdcMerged("wrong-odd", "the merged control covers the odd case"))
	after, _, err, feedback := rdcRefine(t, replacement)
	if err != nil {
		t.Fatalf("err %v\n%s", err, feedback)
	}
	refText(t, "controls", after, replacement)
}

func TestRefineKeepsBothControlsJudgedAsBefore(t *testing.T) {
	parallel(t)
	changed := rdcOdd
	changed.calc = "exports.isEven = n => n >= 0;\n"
	replacement := rdcControls(rdcEven, changed)
	after, _, err, feedback := rdcRefine(t, replacement)
	if err != nil {
		t.Fatalf("err %v\n%s", err, feedback)
	}
	refText(t, "controls", after, replacement)
	weak := rdcOdd
	weak.calc = rdcGood
	after, before, err, feedback := rdcRefine(t, rdcControls(rdcEven, weak))
	if err == nil {
		t.Fatal("a retained control that stops catching its defect was accepted")
	}
	refText(t, "controls", after, before)
	shpContains(t, "feedback", feedback, "Wrong-implementation check")
}

func TestRefineRejectsSupersedeWithoutReason(t *testing.T) {
	parallel(t)
	original := rdcControls(rdcEven, rdcOdd)
	for _, c := range []struct {
		name, replacement string
		want              []string
		wantNone          bool
	}{
		{"blank reason", rdcControls(rdcEven, rdcMerged("wrong-odd", "")), []string{"wrong-odd", "Odd numbers", "blank reason", "supersedes", "because"}, false},
		{"whitespace reason", rdcControls(rdcEven, rdcMerged("wrong-odd", "  \t ")), []string{"wrong-odd", "blank reason"}, false},
		{"unknown original", rdcControls(rdcEven, rdcOdd, rdcMerged("wrong-ghost", "it never existed")), []string{"wrong-ghost", "not a control"}, false},
		{"smaller replacement", rdcControls(rdcEven), []string{"wrong-odd", "Odd numbers", "supersedes", "because"}, false},
		{"deferred replacement", `{"deferred":"later"}` + "\n", []string{"wrong-even", "wrong-odd", "supersedes"}, false},
		{"spacing differs", rdcControls(rdcWrong{" wrong-even ", " Even  numbers", rdcEven.step, rdcEven.calc}, rdcOdd), nil, true},
		{"superseded", rdcControls(rdcEven, rdcMerged("wrong-odd", "covered")), nil, true},
		{"deferred original", rdcControls(rdcEven), nil, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			orig := original
			if c.name == "deferred original" {
				orig = `{"deferred":"later"}` + "\n"
			}
			lines := droppedControlLines([]byte(orig), []byte(c.replacement), "is-even")
			text := strings.Join(lines, "\n")
			if c.wantNone {
				if len(lines) > 0 {
					t.Fatalf("unexpected refusal: %s", text)
				}
				return
			}
			if len(lines) == 0 {
				t.Fatal("the replacement was not refused")
			}
			for _, w := range c.want {
				shpContains(t, "refusal", text, w)
			}
		})
	}
}
