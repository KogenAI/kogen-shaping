package contract

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-coordinator/project"
	"github.com/KogenAI/kogen-shaping/shape"
	"kogen-loop/sys"
)

const readyDocument = `{"schema_version":1,"kind":"preflight","checkout":{"id":"contract-checkout"},"error":null,"data":{"slug":"is-even","ready":true,"checks":[{"name":"configuration","status":"passed"},{"name":"package","status":"passed"},{"name":"approval","status":"passed"},{"name":"cleanliness","status":"passed"},{"name":"conflict","status":"passed"}],"blockers":[],"limitations":["Setup, checks, hooks, providers, tool availability, signing and full readiness/resume safety were not executed or established. Build revalidates normally."]}}`

// oneJSONLine decodes stdout that must be exactly one JSON line.
func oneJSONLine(t *testing.T, out string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil || strings.Count(out, "\n") != 1 || !strings.HasSuffix(out, "\n") {
		t.Fatalf("not one JSON line: %q (%v)", out, err)
	}
	return doc
}

// TestPreflightJSON holds the document kogen reads from shape.Preflight.
func TestPreflightJSON(t *testing.T) {
	t.Parallel()
	checkout := map[string]any{"id": "contract-checkout"}

	t.Run("an approved draft is ready with five passed checks", func(t *testing.T) {
		t.Parallel()
		f := newFx(t)
		f.committedDraft()
		f.newSession()
		by := "Contract Approver <approver@example.invalid>"
		if exit, err := shape.Approve(context.Background(), f.S, shape.ApproveInput{Project: f.Root, Slug: slug, Hash: f.hash(), By: by}); err != nil || exit != 0 {
			t.Fatalf("approve: exit %d, err %v\n%s", exit, err, f.Stdout)
		}

		f.newSession()
		exit, err := shape.Preflight(context.Background(), f.S, shape.PreflightInput{Project: f.Root, Slug: slug, JSON: true, Checkout: checkout})
		if err != nil || exit != 0 {
			t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stdout)
		}
		var want map[string]any
		if err := json.Unmarshal([]byte(readyDocument), &want); err != nil {
			t.Fatal(err)
		}
		if got := oneJSONLine(t, f.Stdout.String()); !reflect.DeepEqual(got, want) {
			t.Fatalf("JSON %v\nwant %v", got, want)
		}
	})

	t.Run("a repository without .kogen/project.yaml is exit 3 with the configuration error", func(t *testing.T) {
		t.Parallel()
		f := newFx(t)
		dir := t.TempDir()
		for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "empty"}} {
			cmd := exec.Command("git", args...)
			cmd.Dir, cmd.Env = dir, f.Env
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, project.ConfigPath)); err == nil {
			t.Fatal("the repository has a configuration")
		}
		_, cfgErr := project.LoadConfig(dir)
		ce, ok := sys.AsError(cfgErr)
		if !ok {
			t.Fatalf("LoadConfig: %v", cfgErr)
		}

		exit, err := shape.Preflight(context.Background(), f.S, shape.PreflightInput{Project: dir, Slug: slug, JSON: true, Checkout: checkout})
		if err != nil || exit != 3 {
			t.Fatalf("exit %d, err %v\n%s", exit, err, f.Stdout)
		}
		doc := oneJSONLine(t, f.Stdout.String())
		wantErr := map[string]any{"class": ce.Class, "reason": ce.Reason, "detail": ce.Detail, "paths": []any{}}
		if doc["data"] != nil || !reflect.DeepEqual(doc["error"], wantErr) {
			t.Fatalf("JSON %v, want data null and error %v", doc, wantErr)
		}
	})
}
