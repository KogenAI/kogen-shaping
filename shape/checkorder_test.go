package shape

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const orderRequest = "Add calc.is_even(n). Non-integers raise TypeError. Use limit 4242."

// orderDecisionRefusal makes the decision check refuse the intent's one decision.
func orderDecisionRefusal() shpStep {
	return shpAnswer(shpCall("decision-check", SubmitDecisionCheckTool, map[string]any{"items": []any{map[string]any{
		"statement": "- Non-integers raise TypeError. (source: request)", "request_quotes": []any{"Non-integers raise TypeError."},
		"verdict": "contradicts", "intent_value": "TypeError", "request_value": "raise TypeError", "difference": "they differ", "why": "because"}}}))
}

// orderLiteralRefusal makes the literal check find a required literal the package lacks.
func orderLiteralRefusal() shpStep {
	return shpAnswer(shpCall("literal-check", SubmitLiteralCheckTool, map[string]any{"items": []any{map[string]any{
		"literal": "4242", "source": "request", "source_quote": "Use limit 4242.", "package_value": "", "why": "w", "status": "required"}}}))
}

func orderIndex(names []string, name string) int { return slices.Index(names, name) }

func eventNames(events []map[string]any) []string {
	var names []string
	for _, e := range events {
		names = append(names, e["event"].(string))
	}
	return names
}

// A static refusal discards even refusing model answers.
func TestStaticRefusalDiscardsTheModelChecks(t *testing.T) {
	parallel(t)
	bad := shpFiles()
	bad["files"].([]any)[2].(map[string]any)["content"] = shpSteps + "\n// BAD\n"
	f := newShpFx(t, shpSubmit("c1", bad), shpSubmit("c2", bad), shpSubmit("c3", bad))
	f.write(".kogen/project.yaml", "version: 1\ncommands:\n  checks:\n    - name: unit\n      run: 'true'\n    - name: steps\n      run: '! grep -q BAD features/steps/is-even__steps.js'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	decision, literal := orderDecisionRefusal(), orderLiteralRefusal()
	f.Prov.route = func(req map[string]any) *shpStep {
		if decisionRequest(req) {
			return &decision
		}
		if literalRequest(req) {
			return &literal
		}
		return nil
	}
	f.git("add", "-A")
	f.git("commit", "-qm", "static check")
	_, err := f.shape(orderRequest)
	if err == nil {
		t.Fatal("a statically refused package was published")
	}
	events := f.events()
	if n := len(shpNamed(events, "submission_refused")); n != 3 {
		t.Fatalf("%d refusals, want 3", n)
	}
	for _, e := range shpNamed(events, "submission_refused") {
		shpEq(t, "gate", e["gate"], staticGate)
	}
	// The identical package gets the literal check each time; the decision answer is reused.
	if n, m := len(f.Prov.decisionRequests()), len(f.Prov.literalRequests()); n != 1 || m != 3 {
		t.Fatalf("%d decision and %d literal model calls, want 1 and 3", n, m)
	}
	for _, name := range []string{"decision_check", "literal_check"} {
		if n := len(shpNamed(events, name)); n != 0 {
			t.Fatalf("%d %s events after a static refusal", n, name)
		}
	}
}

// The three model checks run together, not one after another: each holds until all three have
// started, which only happens when they run at once.
func TestModelChecksRunConcurrently(t *testing.T) {
	parallel(t)
	var running, peak atomic.Int32
	together := make(chan struct{})
	slow := func(context.Context) ([]string, error) {
		n := running.Add(1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		if n == 3 {
			close(together)
		}
		select {
		case <-together:
		case <-time.After(30 * time.Second):
			running.Add(-1)
			return nil, errors.New("the other model checks never started")
		}
		running.Add(-1)
		return nil, nil
	}
	got, err := runModelChecks(context.Background(), slow, slow, slow)
	if err != nil || len(got) != 0 {
		t.Fatalf("refusals %v, err %v", got, err)
	}
	if peak.Load() != 3 {
		t.Fatalf("%d model checks ran at once, want 3", peak.Load())
	}
}

// A check that errors ends the others and the error is returned once all are done.
func TestModelChecksStopOnError(t *testing.T) {
	parallel(t)
	boom := context.Canceled
	_, err := runModelChecks(context.Background(),
		func(context.Context) ([]string, error) { return nil, boom },
		func(ctx context.Context) ([]string, error) { <-ctx.Done(); return nil, ctx.Err() },
		func(context.Context) ([]string, error) { return nil, nil })
	if err != boom {
		t.Fatalf("err %v", err)
	}
}

// When decision and literal both refuse, the refusal carries both problems, decision first, each
// with today's text.
func TestDecisionAndLiteralRefusalsAreBothReported(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpSubmit("c2", shpFiles()), shpSubmit("c3", shpFiles()))
	decision, literal := orderDecisionRefusal(), orderLiteralRefusal()
	f.Prov.route = func(req map[string]any) *shpStep {
		switch {
		case decisionRequest(req):
			return &decision
		case literalRequest(req):
			return &literal
		}
		return nil
	}
	_, err := f.shape(orderRequest)
	if err == nil {
		t.Fatal("a refused package was published")
	}
	reqs := f.Prov.requests()
	if len(reqs) < 2 {
		t.Fatalf("%d shaper requests", len(reqs))
	}
	fed := shpOutputs(reqs[1])["c1"]
	di, li := strings.Index(fed, "Decision check: "), strings.Index(fed, "Literal check: ")
	if di < 0 || li < 0 || di > li {
		t.Fatalf("feedback lacks both checks, decision first:\n%s", fed)
	}
	if !strings.Contains(fed, "statement: - Non-integers raise TypeError. (source: request) | request_quotes: \"Non-integers raise TypeError.\" | verdict: contradicts | intent_value: TypeError | request_value: raise TypeError | difference: they differ | why: because") ||
		!strings.Contains(fed, "literal: 4242 | source: request | source_quote: \"Use limit 4242.\" | status: missing | why: w") {
		t.Fatalf("feedback lacks today's problem lines:\n%s", fed)
	}
	events := f.events()
	var gates []string
	for _, ev := range shpNamed(events, "submission_refused") {
		gates = append(gates, ev["gate"].(string))
	}
	if len(gates) < 2 || gates[0] != "decision_check" || gates[1] != "literal_check" {
		t.Fatalf("refusal gates %v", gates)
	}
	// One submission counts once against the retry budget.
	shpEq(t, "attempt 1 total", shpNamed(events, "submission_refused")[0]["total"], shpNamed(events, "submission_refused")[1]["total"])
}

// A submission that passes every check reaches certify as before: the local checks, then the
// model checks, then the readiness events, with the same events overall.
func TestPassingSubmissionReachesCertifyAsBefore(t *testing.T) {
	parallel(t)
	f := newShpFx(t, shpSubmit("c1", shpFiles()), shpSubmit("c2", map[string]any{"files": []any{map[string]any{"path": ".kogen/intents/is-even/acceptance.feature", "content": shpFeature}}}))
	exit, err := f.shape(orderRequest)
	if err != nil || exit != 0 {
		t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stderr)
	}
	names := eventNames(f.events())
	for _, name := range []string{"step_selfcheck", "static_checks", "decision_check", "literal_check", "unpassable_check", "contract_review", "readiness_review_reused"} {
		shpEq(t, name, strings.Count(strings.Join(names, ","), name+","), 1)
	}
	self, static := orderIndex(names, "step_selfcheck"), orderIndex(names, "static_checks")
	decision, literal := orderIndex(names, "decision_check"), orderIndex(names, "literal_check")
	unpassable, review := orderIndex(names, "unpassable_check"), orderIndex(names, "contract_review")
	if !(self < static && static < min(decision, literal) && max(decision, literal) < unpassable && unpassable < review) {
		t.Fatalf("events out of order: %v", names)
	}
}

func TestModelErrorCancelsStaticBeforeOtherModelsFinish(t *testing.T) {
	staticCancelled := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	want := errors.New("decision failed")
	go func() {
		_, err := runModelChecksCancel(context.Background(),
			func(context.Context) ([]string, error) { return nil, want },
			func(context.Context) ([]string, error) { <-release; return nil, nil },
			func(context.Context) ([]string, error) { return nil, nil },
			func() { close(staticCancelled) })
		done <- err
	}()
	select {
	case <-staticCancelled:
	case <-time.After(time.Second):
		close(release)
		<-done
		t.Fatal("static cancellation waited for another model")
	}
	close(release)
	if err := <-done; err != want {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestEmptyLiteralResultIsHeldUntilPublication(t *testing.T) {
	f := newShpFx(t)
	sh := f.shaping()
	sh.request = ""
	var publish func()
	if _, err := sh.literalCheck(context.WithValue(context.Background(), modelPublicationKey{}, &publish)); err != nil {
		t.Fatal(err)
	}
	if len(shpNamed(f.events(), "literal_check")) != 0 || publish == nil {
		t.Fatal("empty request result was not held")
	}
	publish()
	if len(shpNamed(f.events(), "literal_check")) != 1 {
		t.Fatal("empty request result was not published")
	}
}
