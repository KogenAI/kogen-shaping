'use strict';
// The Go source of the k-shape-recovery-completes-package acceptance harness. The steps compile it
// into the shape package's tests through a Go overlay as shape/ksr_acceptance_test.go; that path
// never exists in the repository. The source holds no backquote and no dollar-brace, so this raw
// template keeps it byte for byte.
module.exports = String.raw`package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
)

// The acceptance harness of k-shape-recovery-completes-package: one scripted native Shape journey
// per run against the loopback provider (no model) in the synthetic is-even repository. KSR_SCRIPT
// names the journey. The run prints one KSR line with what a user and the provider observed.

const ksrRecovery = "Recovery before submitting again:"

const (
	ksrIntentPath  = ".kogen/intents/is-even/intent.md"
	ksrFeaturePath = ".kogen/intents/is-even/acceptance.feature"
	ksrStepsPath   = "features/steps/is-even__steps.js"
)

func ksrFile(path, content string) any { return map[string]any{"path": path, "content": content} }

func ksrFiles(files ...any) map[string]any { return map[string]any{"files": files} }

// ksrUserTexts are the user texts a request carries, in order.
func ksrUserTexts(req map[string]any) []string {
	var out []string
	for _, it := range forceConversation(req) {
		m, _ := it.(map[string]any)
		if m["role"] != "user" {
			continue
		}
		switch c := m["content"].(type) {
		case string:
			out = append(out, c)
		case []any:
			for _, b := range c {
				bm, _ := b.(map[string]any)
				if s, ok := bm["text"].(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// ksrRows are the events of one name without their event field, in order.
func ksrRows(events []map[string]any, name string) []any {
	rows := []any{}
	for _, e := range shpNamed(events, name) {
		delete(e, "event")
		rows = append(rows, e)
	}
	return rows
}

func TestKSRAcceptance(t *testing.T) {
	w := responsesWire
	call := func(id, name string, args map[string]any) map[string]any { return shpCall(id, name, args) }
	submit := func(id string, args map[string]any) shpStep { return w.calls(call(id, "submit", args)) }
	readme := call("read", "read_file", map[string]any{"path": "README.md"})
	intent := ksrFile(ksrIntentPath, shpScopedIntentMD)
	feature := ksrFile(ksrFeaturePath, shpFeature)
	steps := ksrFile(ksrStepsPath, shpSteps)
	both := map[string]any{"questions": []any{shpQuestion("Should odd numbers be reported too?")}, "files": []any{intent, feature, steps}}
	whole := map[string]string{ksrIntentPath: shpScopedIntentMD, ksrFeaturePath: shpFeature, ksrStepsPath: shpSteps}
	f := w.newShape(t)
	f.setReport("PASSED", 0)
	var script []shpStep
	var published map[string]string
	switch name := os.Getenv("KSR_SCRIPT"); name {
	case "missing-step-written":
		script = []shpStep{
			w.calls(readme, call("c1", "submit", ksrFiles(intent, feature))),
			w.calls(call("add", "propose_files", ksrFiles(steps))),
			submit("c2", ksrFiles(steps)),
		}
		published = whole
	case "missing-step-prose":
		script = []shpStep{
			w.calls(readme, call("c1", "submit", ksrFiles(intent, feature))),
			w.prose("The step file is still unfinished; the package needs more work."),
			submit("c2", ksrFiles(intent, feature)),
			w.calls(call("add", "propose_files", ksrFiles(steps))),
			submit("c3", ksrFiles(steps)),
		}
		published = whole
	case "intent-only":
		script = []shpStep{
			w.calls(readme, call("c1", "submit", ksrFiles(intent))),
			submit("c2", ksrFiles(feature)),
			submit("c3", ksrFiles(steps)),
		}
		published = whole
	case "envelope-repaired":
		script = []shpStep{
			w.calls(readme, call("c1", "submit", both)),
			submit("c2", ksrFiles(intent, feature, steps)),
		}
		published = whole
	case "envelope-repeated":
		script = []shpStep{
			w.calls(readme, call("c1", "submit", both)),
			w.prose("The package is complete; I will submit it again."),
			submit("c2", both),
			submit("c3", both),
			submit("c4", both),
			submit("c5", both),
			w.prose("unused"),
		}
	case "valid-first":
		script = []shpStep{w.calls(readme, call("c1", "submit", ksrFiles(intent, feature, steps)))}
		published = whole
	default:
		t.Fatalf("unknown KSR_SCRIPT %q", name)
	}
	f.Prov.seq = script
	exit, err := f.shape("Add calc.is_even(n).")
	obs := map[string]any{"exit": exit, "stop": ""}
	if err != nil {
		if e, ok := sys.AsError(err); ok {
			obs["exit"], obs["stop"] = e.Exit, e.Render()
		} else {
			obs["exit"], obs["stop"] = -1, err.Error()
		}
	}
	reqs := shpShaperRequests(f.Prov.requests())
	recovery := []int{}
	seen := 0
	for i, r := range reqs {
		n := 0
		for _, s := range ksrUserTexts(r) {
			if strings.HasPrefix(s, ksrRecovery) {
				n++
			}
		}
		if n > seen {
			recovery = append(recovery, i+1)
		}
		seen = n
	}
	obs["requests"], obs["recovery"] = len(reqs), recovery
	outputs := map[string]string{}
	if len(reqs) > 0 {
		outputs = shpOutputs(reqs[len(reqs)-1])
	}
	obs["outputs"] = outputs
	events := f.events()
	obs["refused"] = ksrRows(events, "submission_refused")
	obs["prechecks"] = ksrRows(events, "submission_precheck")
	invalid := -1
	if counts := shpNamed(events, "shaping_turn_counts"); len(counts) == 1 {
		if v, ok := counts[0]["invalid_submissions"].(float64); ok {
			invalid = int(v)
		}
	}
	obs["invalid"] = invalid
	obs["intent_written"] = f.exists(".kogen/intents/is-even")
	var mismatch []string
	if published != nil {
		paths := make([]string, 0, len(published))
		for p := range published {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			if !f.exists(p) || f.read(p) != published[p] {
				mismatch = append(mismatch, p)
			}
		}
	}
	obs["published"] = published != nil && exit == 0 && err == nil && len(mismatch) == 0
	obs["unpublished"] = mismatch
	b, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("KSR %s\n", b)
}
`;
