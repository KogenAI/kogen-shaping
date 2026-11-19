package shape

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// A timed-out static check is remembered for the running session only: these tests shape a request
// through the real entry point with a scripted shaper and fake check commands.

const stmReason = "timed out after 120 s in round 1; not run again in this session"

type stmCheckFn func(n int) sys.Result

type stmFixture struct {
	*shpFx
	mu    sync.Mutex
	calls map[string]int
}

func (s *stmFixture) count(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[name]
}

// stmNew configures the named fake checks (in order) and one submission per id, then an acknowledgment.
func stmNew(t *testing.T, names []string, fns map[string]stmCheckFn, ids ...string) *stmFixture {
	t.Helper()
	return stmNewSessions(t, names, fns, ids)
}

// stmNewSessions scripts one session per list of ids: a different package per id, then an
// acknowledgment of the last one.
func stmNewSessions(t *testing.T, names []string, fns map[string]stmCheckFn, sessions ...[]string) *stmFixture {
	t.Helper()
	var steps []shpStep
	for _, ids := range sessions {
		for i, id := range ids {
			files := shpFiles()
			if i > 0 {
				files["files"].([]any)[2] = map[string]any{"path": "features/steps/is-even__steps.js", "content": shpSteps + "// round " + id + "\n"}
			}
			steps = append(steps, shpSubmit(id, files))
		}
		steps = append(steps, shpAck(ids[len(ids)-1]+"-ack"))
	}
	s := &stmFixture{shpFx: newShpFx(t, steps...), calls: map[string]int{}}
	var y strings.Builder
	y.WriteString("version: 1\ncommands:\n  checks:\n")
	for _, name := range names {
		name, fn := name, fns[name]
		command := "true kogen-stm-unit-" + nonWord.ReplaceAllString(t.Name(), "-") + "-" + name
		fakeStaticChecks.Store(command, func(ctx context.Context) (sys.Result, error) {
			s.mu.Lock()
			s.calls[name]++
			n := s.calls[name]
			s.mu.Unlock()
			return fn(n), nil
		})
		t.Cleanup(func() { fakeStaticChecks.Delete(command) })
		y.WriteString("    - name: " + name + "\n      run: '" + command + "'\n")
	}
	y.WriteString("  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n")
	s.write(".kogen/project.yaml", y.String())
	s.write("mix.lock", "%{}\n")
	s.git("add", "-A")
	s.git("commit", "-qm", "stm checks")
	return s
}

// stmWork takes a measurable time without waiting on the clock, so that wall_ms is above 0.
func stmWork() {
	for start := time.Now(); time.Since(start) < 2*time.Millisecond; {
	}
}

func stmTimeout(int) sys.Result {
	stmWork()
	return sys.Result{Status: 137, TimedOut: true}
}

// stmEntry is the entry of the named check in the nth static_checks event of events.
func stmEntry(t *testing.T, events []map[string]any, n int, name string) map[string]any {
	t.Helper()
	all := shpNamed(events, "static_checks")
	if n > len(all) {
		t.Fatalf("static_checks event %d missing: have %d", n, len(all))
	}
	for _, c := range all[n-1]["checks"].([]any) {
		if c := c.(map[string]any); c["name"] == name {
			return c
		}
	}
	t.Fatalf("check %s missing from static_checks event %d", name, n)
	return nil
}

func (s *stmFixture) shapeOnce(t *testing.T) {
	t.Helper()
	exit, err := s.shape("Add calc.is_even(n).")
	if exit != 0 || err != nil {
		t.Fatalf("shape: exit %d err %v\n%s", exit, err, s.Stderr.String())
	}
}

func TestStaticTimeoutMemoryNotRepeated(t *testing.T) {
	s := stmNew(t, []string{"slow"}, map[string]stmCheckFn{"slow": stmTimeout}, "c1", "c2")
	s.shapeOnce(t)
	if n := s.count("slow"); n != 1 {
		t.Fatalf("slow started %d times, want 1", n)
	}
	ev := s.events()
	first, second := stmEntry(t, ev, 1, "slow"), stmEntry(t, ev, 2, "slow")
	if first["status"] != "unavailable" || first["wall_ms"].(float64) <= 0 {
		t.Fatalf("round 1 entry: %v", first)
	}
	if second["status"] != "unavailable" || second["wall_ms"].(float64) != 0 || second["reason"] != stmReason || second["exit"] != nil || second["output"] != "" {
		t.Fatalf("round 2 entry: %v", second)
	}
	line := "kogen: static check slow unavailable: " + stmReason
	if got := strings.Count(s.Stderr.String(), line+" ("); got != 1 {
		t.Fatalf("progress line printed %d times:\n%s", got, s.Stderr.String())
	}
	rec, err := os.ReadFile(filepath.Join(s.Root, ".kogen/local/static-checks/"+shpSlug+".json"))
	if err != nil || !strings.Contains(string(rec), stmReason) {
		t.Fatalf("static-checks record: %v\n%s", err, rec)
	}
	if got := shpNamed(ev, "static_checks")[1]["outcome"]; got != "unknown" {
		t.Fatalf("outcome %v, want unknown", got)
	}
}

func TestStaticTimeoutMemoryOnlyForTimeouts(t *testing.T) {
	tool := func(n int) sys.Result {
		stmWork()
		if n == 1 {
			return sys.Result{Status: 127, Output: "/bin/sh: gofmt: command not found\n"}
		}
		return sys.Result{}
	}
	s := stmNew(t, []string{"tool"}, map[string]stmCheckFn{"tool": tool}, "c1", "c2")
	s.shapeOnce(t)
	if n := s.count("tool"); n != 2 {
		t.Fatalf("tool started %d times, want 2", n)
	}
	ev := s.events()
	if got := stmEntry(t, ev, 1, "tool")["status"]; got != "unavailable" {
		t.Fatalf("round 1: %v", got)
	}
	if got := stmEntry(t, ev, 2, "tool"); got["status"] != "passed" || got["wall_ms"].(float64) <= 0 {
		t.Fatalf("round 2: %v", got)
	}
}

func TestStaticTimeoutMemoryKeepsFailedChecks(t *testing.T) {
	bad := func(n int) sys.Result {
		stmWork()
		if n <= 2 { // the plain run and the base run of round 1
			return sys.Result{Status: 1, Output: "boom\n"}
		}
		return sys.Result{}
	}
	s := stmNew(t, []string{"bad", "slow"}, map[string]stmCheckFn{"bad": bad, "slow": stmTimeout}, "c1", "c2")
	s.shapeOnce(t)
	ev := s.events()
	if got := stmEntry(t, ev, 1, "bad")["status"]; got != "preexisting" {
		t.Fatalf("bad in round 1: %v", got)
	}
	if got := stmEntry(t, ev, 2, "bad"); got["status"] != "passed" || got["wall_ms"].(float64) <= 0 {
		t.Fatalf("bad in round 2: %v", got)
	}
	if got := stmEntry(t, ev, 2, "slow"); got["status"] != "unavailable" || got["wall_ms"].(float64) != 0 {
		t.Fatalf("slow in round 2: %v", got)
	}
	if n := s.count("slow"); n != 1 {
		t.Fatalf("slow started %d times, want 1", n)
	}
}

func TestStaticTimeoutMemoryResetsOnLockChangeAndNewSession(t *testing.T) {
	// A check that passes rewrites mix.lock during round 2, so round 3's candidate has another lock.
	var s *stmFixture
	writer := func(n int) sys.Result {
		if n == 2 {
			if err := os.WriteFile(filepath.Join(s.Root, "mix.lock"), []byte("%{\"dep\" => 2}\n"), 0o644); err != nil {
				t.Error(err)
			}
		}
		return sys.Result{}
	}
	s = stmNew(t, []string{"slow", "writer"}, map[string]stmCheckFn{"slow": stmTimeout, "writer": writer}, "c1", "c2", "c3")
	s.shapeOnce(t)
	ev := s.events()
	if got := stmEntry(t, ev, 2, "slow"); got["wall_ms"].(float64) != 0 || got["reason"] != stmReason {
		t.Fatalf("round 2: %v", got)
	}
	third := stmEntry(t, ev, 3, "slow")
	if third["wall_ms"].(float64) <= 0 || strings.Contains(third["reason"].(string), "not run again") {
		t.Fatalf("round 3 must run the check again: %v", third)
	}
	if n := s.count("slow"); n != 2 {
		t.Fatalf("slow started %d times, want 2", n)
	}

	// A new session on the same tree starts with no memory.
	n := stmNewSessions(t, []string{"slow"}, map[string]stmCheckFn{"slow": stmTimeout}, []string{"c1", "c2"}, []string{"d1", "d2"})
	n.shapeOnce(t)
	n.git("clean", "-fdq")
	before := len(shpNamed(n.events(), "static_checks"))
	n.shapeOnce(t)
	ev = n.events()
	if got := n.count("slow"); got != 2 {
		t.Fatalf("slow started %d times over two sessions, want 2", got)
	}
	if got := stmEntry(t, ev, before+1, "slow"); got["wall_ms"].(float64) <= 0 || strings.Contains(got["reason"].(string), "not run again") {
		t.Fatalf("new session round 1: %v", got)
	}
}
