package shape

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

func timeoutRefusal() error {
	e := sys.Fail("intent", "not_ready", "acceptance timed out", 1)
	e.Lines = []string{"acceptance timed out: command exceeds the 120-second limit", "Acceptance ran for 120.2s before it was stopped."}
	return e
}

// An acceptance timeout of an overlapped observation is observed once more, alone; only that
// refusal, and only with overlap, is repeated.
func TestJoinReadinessRepeatsOverlappedTimeoutOnce(t *testing.T) {
	assertion := sys.Fail("intent", "not_ready", "acceptance failed", 1)
	for _, tc := range []struct {
		name       string
		overlapped bool
		results    []error // the error of each call
		calls      int32
		wantErr    string // "" for ready
		wantLast   string
	}{
		{"repeat passes", true, []error{timeoutRefusal(), nil}, 2, "", ""},
		{"both time out", true, []error{timeoutRefusal(), timeoutRefusal()}, 2, "acceptance timed out", readinessTimeoutRepeated},
		{"no overlap", false, []error{timeoutRefusal(), nil}, 1, "acceptance timed out", "Acceptance ran for 120.2s before it was stopped."},
		{"other refusal", true, []error{assertion, nil}, 1, "acceptance failed", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newShpFx(t)
			sh := f.shaping()
			sh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S), Evidence: sh.ev}
			var calls atomic.Int32
			sh.readinessObserver = func(context.Context, []string) (certification, error) {
				n := int(calls.Add(1))
				if err := tc.results[n-1]; err != nil {
					return certification{}, err
				}
				return certification{text: "accepted: the Intent is ready", out: outcome{kind: outReady}}, nil
			}
			run := sh.startReadiness(context.Background(), nil, "pkg")
			<-run.done
			run.overlapped = tc.overlapped
			cert, err := sh.joinReadiness(context.Background(), nil, run, "pkg")
			if calls.Load() != tc.calls {
				t.Fatalf("readiness called %d times, want %d", calls.Load(), tc.calls)
			}
			repeats := shpNamed(f.events(), "readiness_repeated")
			if tc.calls == 2 {
				if len(repeats) != 1 || repeats[0]["package"] != "pkg" {
					t.Fatalf("repeat events %v", repeats)
				}
				if _, ok := repeats[0]["first_wall_ms"].(float64); !ok {
					t.Fatalf("first_wall_ms is not a number: %v", repeats[0])
				}
				if !strings.Contains(f.Stderr.String(), "kogen: readiness timed out while other checks ran; repeating it alone") {
					t.Fatalf("progress line missing: %q", f.Stderr.String())
				}
			} else if len(repeats) != 0 {
				t.Fatalf("unexpected repeat events %v", repeats)
			}
			if tc.wantErr == "" {
				if err != nil || cert.out.kind != outReady {
					t.Fatalf("want ready, got %v", err)
				}
				return
			}
			e, ok := sys.AsError(err)
			if !ok || e.Detail != tc.wantErr {
				t.Fatalf("want refusal %q, got %v", tc.wantErr, err)
			}
			if tc.wantLast != "" && e.Lines[len(e.Lines)-1] != tc.wantLast {
				t.Fatalf("last line %q, want %q", e.Lines[len(e.Lines)-1], tc.wantLast)
			}
		})
	}
}
