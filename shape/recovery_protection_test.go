package shape

import (
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
)

// A retained unapproved draft must be repairable through Shape. Describing its
// own artifacts as a protected-edit blocker prevents the documented recovery.
func TestFailedDraftRecoveryDistinguishesBuildProtection(t *testing.T) {
	parallel(t)
	f, sh := newGuardedShaping(t)
	for _, raw := range shpFiles()["files"].([]any) {
		file := raw.(map[string]any)
		f.write(file["path"].(string), file["content"].(string))
	}
	f.write(".kogen/local/shaping/is-even/not-ready.json", `{"gate":"decision_check","problems":["repair the draft"]}`)
	for _, tc := range []struct {
		name, path string
		blocked    bool
	}{
		{"own intent", ".kogen/intents/is-even/intent.md", false},
		{"own acceptance", ".kogen/intents/is-even/acceptance.feature", false},
		{"own steps", "features/steps/is-even__steps.js", false},
		{"project configuration", project.ConfigPath, true},
		{"configured protection", "guard.txt", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sub, problems, err := sh.validateBlocked(map[string]any{"paths": []any{tc.path}, "reason": "This path needs changing."})
			if err != nil {
				t.Fatal(err)
			}
			if (sub.blocked && len(problems) == 0) != tc.blocked {
				t.Fatalf("blocked=%v problems=%v; expected %v", sub.blocked, problems, tc.blocked)
			}
			if !tc.blocked && !strings.Contains(strings.Join(problems, "\n"), "unapproved draft") {
				t.Fatalf("missing supported recovery: %v", problems)
			}
		})
	}
	cfg := *sh.cfg
	cfg.Protect = append(append([]string(nil), cfg.Protect...), "features/steps/is-even__steps.js")
	sh.cfg = &cfg
	if sub, problems, err := sh.validateBlocked(map[string]any{"paths": []any{"features/steps/is-even__steps.js"}, "reason": "Explicitly protected"}); err != nil || !sub.blocked || len(problems) != 0 {
		t.Fatalf("configured protection lost: %+v %v %v", sub, problems, err)
	}
}

func TestShapeRecoveryContextKeepsFrozenProtectionSeparate(t *testing.T) {
	parallel(t)
	f, sh := newGuardedShaping(t)
	for _, raw := range shpFiles()["files"].([]any) {
		file := raw.(map[string]any)
		f.write(file["path"].(string), file["content"].(string))
	}
	f.write(".kogen/local/shaping/is-even/not-ready.json", `{"gate":"decision_check","problems":["repair the draft"]}`)
	protected, err := project.Protected(f.Root, sh.cfg)
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := contextBlocks(promptInput{root: f.Root, slug: shpSlug, cfg: sh.cfg, state: &state{Request: "Repair the original request."}, protected: protected, selected: "navigation"})
	if err != nil {
		t.Fatal(err)
	}
	text := blocks[0].(map[string]any)["text"].(string)
	for _, want := range []string{"unapproved draft", "Other declared helpers must be read and resubmitted", "Configured protect patterns, project configuration and other Intents' artifacts remain protected", "Draft paths editable through Shape recovery:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("recovery context omits %q: %s", want, text)
		}
	}
	// The same filename cannot be borrowed from another Intent during recovery.
	other := strings.Replace(f.read(".kogen/intents/is-even/intent.md"), "Slug: is-even", "Slug: other-one", 1)
	f.write(".kogen/intents/other-one/intent.md", other)
	if _, err := project.LoadIntent(f.Root, "other-one"); err != nil {
		t.Fatal(err)
	}
	owned, err := recoveryOwnedPaths(f.Root, shpSlug, sh.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if owned["features/steps/is-even__steps.js"] {
		t.Fatal("a step declared by another Intent became recovery-owned")
	}
}

func TestShapeRetriesOwnDraftInsteadOfReportingProtectedConflict(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("first", shpFiles()), shpAck("again"), shpAck("last"))
	f.setReport("UNDEFINED", 1)
	if exit, err := f.shape("Add calc.is_even(n)."); err == nil || exit == 0 {
		t.Fatalf("defective first draft unexpectedly ready: %d %v", exit, err)
	}
	f.setReport("FAILED", 1)
	f.Prov.mu.Lock()
	f.Prov.seq = append(f.Prov.seq,
		shpSubmit("repair-block", map[string]any{"blocked": map[string]any{"paths": []any{".kogen/intents/is-even/intent.md"}, "reason": "The retained draft must change."}}),
		shpSubmit("repair", shpFiles()), shpAck("repair-ack"))
	f.Prov.mu.Unlock()
	if exit, err := f.shape("Add calc.is_even(n)."); err != nil || exit != 0 {
		t.Fatalf("supported recovery stopped as protected: %d %v\n%s", exit, err, f.Stderr)
	}
	if f.exists(".kogen/local/shaping/is-even/not-ready.json") {
		t.Fatal("successful native recovery left the refusal latch")
	}
	var saw bool
	for _, request := range shpShaperRequests(f.Prov.requests()) {
		if feedback, ok := shpOutputs(request)["repair-block"]; ok {
			saw = true
			if !strings.Contains(feedback, "unapproved draft") || !strings.Contains(feedback, "resubmitting every declared owned acceptance helper") {
				t.Fatalf("unsupported recovery feedback: %s", feedback)
			}
		}
	}
	if !saw {
		t.Fatal("no recovery feedback was delivered")
	}
}
