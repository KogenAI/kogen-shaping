package contract

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/KogenAI/kogen-shaping/shape"
	"kogen-loop/sys"
)

// These cases are copied from shape/response_budget_test.go, which kogen's response_budget_test.go
// runs by name, and run through the exported shape.Shape and shape.Refine with the production
// settings: none configures a response budget, so each sends under the pinned loop's 30 minutes.

func eq[T any](t *testing.T, what string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %v, want %v", what, got, want)
	}
}

// wantThirtyMinutes holds that all requests the transport saw were sent under the library's
// thirty-minute budget (not Build's ten minutes, not a lower setting, not unbounded), and that
// there were at least two.
func wantThirtyMinutes(t *testing.T, p *scripted) {
	t.Helper()
	remaining, missing := p.observed()
	if len(remaining) < 2 || missing != 0 {
		t.Fatalf("observed %d requests with a deadline (%d without), want at least 2 and none without", len(remaining), missing)
	}
	for i, d := range remaining {
		if d > 30*time.Minute || d < 29*time.Minute {
			t.Fatalf("request %d had %s left, want between 29 and 30 minutes", i, d)
		}
	}
}

// budgetEvents are the response_budget events of a run log.
func budgetEvents(events []map[string]any) []map[string]any {
	var out []map[string]any
	for _, e := range events {
		if e["event"] == "response_budget" {
			out = append(out, e)
		}
	}
	return out
}

func wantBudgetRecord(t *testing.T, events []map[string]any) {
	t.Helper()
	got := budgetEvents(events)
	if len(got) != 1 {
		t.Fatalf("%d response_budget events, want exactly one: %v", len(got), got)
	}
	eq(t, "budget", got[0]["budget"], any("30 minutes"))
	eq(t, "budget_ms", got[0]["budget_ms"], any(float64(1800000)))
}

// wantThirtyMinuteText holds that what the user saw reports the provider's unavailable failure of a
// stalled wait, naming the loop's thirty minutes.
func wantThirtyMinuteText(t *testing.T, text string) {
	t.Helper()
	if !strings.Contains(text, "provider/unavailable: ChatGPT response wait exceeded 30 minutes\n") || strings.Contains(text, "30m0s") {
		t.Fatalf("the user saw %q", text)
	}
}

// userSaw is everything a command told the user: its output and its error.
func userSaw(f *fx, err error) string {
	text := f.Stdout.String()
	if e, ok := sys.AsError(err); ok {
		text += e.Render()
	} else if err != nil {
		text += err.Error()
	}
	return text
}

func shapeInput(f *fx) shape.ShapeInput {
	return shape.ShapeInput{Project: f.Root, Slug: slug, Request: "Add calc.is_even(n).", Client: f.Client}
}

// refineDraft is a ready draft whose readiness record is removed, so Refine asks the model.
func refineDraft(t *testing.T, script ...turn) (*fx, string) {
	t.Helper()
	f := newFx(t, script...)
	f.draft()
	hash := f.hash()
	if err := os.Remove(filepath.Join(f.Root, ".kogen/local/readiness/is-even.json")); err != nil {
		t.Fatal(err)
	}
	f.newSession()
	return f, hash
}

func TestShapeBudgetRolesShape(t *testing.T) {
	t.Parallel()
	t.Run("shape sends every request under thirty minutes and records it once", func(t *testing.T) {
		t.Parallel()
		f := newFx(t, submit("c1", filesSubmission()), ack("c2"))
		if _, err := shape.Shape(context.Background(), f.S, shapeInput(f)); err != nil {
			t.Fatalf("%v\n%s", err, f.Stdout)
		}
		wantThirtyMinutes(t, f.Prov)
		eq(t, "client response budget", f.Client.ResponseBudget, time.Duration(0))
		wantBudgetRecord(t, f.shapeEvents())
	})

	t.Run("a stalled shape expires unavailable naming thirty minutes, is not retried and records once", func(t *testing.T) {
		t.Parallel()
		f := newFx(t, stalled(), submit("c1", filesSubmission()), ack("c2"))
		exit, err := shape.Shape(context.Background(), f.S, shapeInput(f))
		e, ok := sys.AsError(err)
		if !ok || e.Class != "shape" || e.Reason != "provider_failed" || e.Exit != 4 || exit != 4 {
			t.Fatalf("exit %d err %v", exit, err)
		}
		wantThirtyMinuteText(t, userSaw(f, err))
		eq(t, "requests", f.Prov.shaperRequests(), 1)
		wantBudgetRecord(t, f.shapeEvents())

		// The retry Shape names is the rerun: it is a new run with its own single record.
		f.newSession()
		if _, err := shape.Shape(context.Background(), f.S, shapeInput(f)); err != nil {
			t.Fatalf("%v\n%s", err, f.Stdout)
		}
		eq(t, "requests after the rerun", f.Prov.shaperRequests() > 1, true)
		eq(t, "records after the rerun", len(budgetEvents(f.shapeEvents())), 2)
	})

	t.Run("refine sends every request under thirty minutes and records it once", func(t *testing.T) {
		t.Parallel()
		f, hash := refineDraft(t, answer(call("p1", "probe", map[string]any{"command": "true"})), refinement("c1", intentPath, intentMD))
		in := shape.RefineInput{Project: f.Root, Slug: slug, Hash: hash, Request: "Clarify the draft.", Client: f.Client}
		if _, err := shape.Refine(context.Background(), f.S, in); err != nil {
			t.Fatal(err)
		}
		wantThirtyMinutes(t, f.Prov)
		eq(t, "client response budget", f.Client.ResponseBudget, time.Duration(0))
		wantBudgetRecord(t, f.refineEvents())
	})

	t.Run("a stalled refine expires unavailable naming thirty minutes and records once", func(t *testing.T) {
		t.Parallel()
		f, hash := refineDraft(t, stalled(), refinement("c1", intentPath, intentMD))
		in := shape.RefineInput{Project: f.Root, Slug: slug, Hash: hash, Request: "Clarify the draft.", Client: f.Client}
		exit, err := shape.Refine(context.Background(), f.S, in)
		if e, ok := sys.AsError(err); !ok || e.Class != "provider" || e.Reason != "unavailable" || e.Exit != 4 || exit != 4 {
			t.Fatalf("exit %d err %v, want provider/unavailable (exit 4)", exit, err)
		}
		wantThirtyMinuteText(t, userSaw(f, err))
		eq(t, "requests", f.Prov.shaperRequests(), 1)
		wantBudgetRecord(t, f.refineEvents())
	})
}
