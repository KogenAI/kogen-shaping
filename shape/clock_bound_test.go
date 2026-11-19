package shape

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
	"github.com/KogenAI/kogen-verification/verify"
)

// The tests of this file, and the fake clock and fake commands they stand on, keep the shape
// tests off the real clock: a duration is a number the test moves on an injected clock, and an
// overlap is shown by the order in which things started, never by how long the host took.

// fakeClock is a clock that only moves when a test moves it. A goroutine that sleeps on it
// blocks until the clock has reached its deadline; nothing waits on the real clock.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	pending []*fakeSleeper
	changed chan struct{} // closed, and replaced, whenever the pending sleepers change
}

type fakeSleeper struct {
	until time.Time
	done  chan struct{}
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(1_800_000_000, 0), changed: make(chan struct{})}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// notify wakes whoever waits in await; c.mu is held.
func (c *fakeClock) notify() {
	close(c.changed)
	c.changed = make(chan struct{})
}

// Sleep blocks until the clock has advanced by d, or ctx ends.
func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	if d <= 0 {
		c.mu.Unlock()
		return nil
	}
	w := &fakeSleeper{until: c.now.Add(d), done: make(chan struct{})}
	c.pending = append(c.pending, w)
	c.notify()
	c.mu.Unlock()
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		c.drop(w)
		c.mu.Unlock()
		return ctx.Err()
	}
}

// drop removes a sleeper that is still pending; c.mu is held.
func (c *fakeClock) drop(w *fakeSleeper) {
	for i, p := range c.pending {
		if p == w {
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
			c.notify()
			return
		}
	}
}

// release wakes every sleeper whose deadline the clock has reached; c.mu is held.
func (c *fakeClock) release() {
	var left []*fakeSleeper
	for _, w := range c.pending {
		if w.until.After(c.now) {
			left = append(left, w)
		} else {
			close(w.done)
		}
	}
	c.pending = left
	c.notify()
}

// Advance moves the clock by d and wakes the sleepers it reaches.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.release()
}

// AdvanceToNext moves the clock to the earliest deadline of a pending sleeper and wakes the
// sleepers that reaches. It reports false when nobody sleeps.
func (c *fakeClock) AdvanceToNext() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) == 0 {
		return false
	}
	next := c.pending[0].until
	for _, w := range c.pending {
		if w.until.Before(next) {
			next = w.until
		}
	}
	c.now = next
	c.release()
	return true
}

// awaitSleepers blocks until n goroutines sleep on the clock at once. Its limit only ends a
// test whose sleepers never come together; a passing test does not wait for it.
func (c *fakeClock) awaitSleepers(n int) bool {
	give := time.After(30 * time.Second)
	for {
		c.mu.Lock()
		ok, changed := len(c.pending) >= n, c.changed
		c.mu.Unlock()
		if ok {
			return true
		}
		select {
		case <-changed:
		case <-give:
			return false
		}
	}
}

// fakeStaticChecks maps the text of a configured static check to the function that stands in for
// running it; a check that is not in the map runs as a real command.
var fakeStaticChecks sync.Map

func init() {
	real := staticRunCommand
	staticRunCommand = func(ctx context.Context, s *sys.Session, command, dir string, opts sys.Options) (sys.Result, error) {
		if fn, ok := fakeStaticChecks.Load(strings.TrimPrefix(command, "set -x\n")); ok {
			return fn.(func(context.Context) (sys.Result, error))(ctx)
		}
		return real(ctx, s, command, dir, opts)
	}
}

var nonWord = regexp.MustCompile(`[^A-Za-z0-9]+`)

// fakeStaticCheck registers fn as the run of a static check for the test and returns the command
// to configure for it.
func fakeStaticCheck(t *testing.T, fn func(context.Context) (sys.Result, error)) string {
	t.Helper()
	command := "true kogen-fake-check-" + nonWord.ReplaceAllString(t.Name(), "-")
	fakeStaticChecks.Store(command, fn)
	t.Cleanup(func() { fakeStaticChecks.Delete(command) })
	return command
}

// fakeStaticChecksConfig is a project configuration with the one static check command.
func fakeStaticChecksConfig(command string) string {
	return "version: 1\ncommands:\n  checks:\n    - name: slow\n      run: '" + command + "'\n  acceptance: {run: sh accept.sh, steps_dir: features/steps}\n"
}

// TestClockBoundStaticChecksOverlap: a static check and a decision check each wait two seconds
// on the injected clock, are started together, and the checks end two injected seconds after
// they started, not four, whatever the real time taken.
func TestClockBoundStaticChecksOverlap(t *testing.T) {
	parallel(t)
	fc := newFakeClock()
	f := newShpFx(t)
	command := fakeStaticCheck(t, func(ctx context.Context) (sys.Result, error) {
		return sys.Result{}, fc.Sleep(ctx, 2*time.Second)
	})
	f.write(".kogen/project.yaml", fakeStaticChecksConfig(command))
	f.git("add", "-A")
	f.git("commit", "-qm", "waiting check")
	sh := f.shaping()
	ev, err := NewEvidenceWithClock(filepath.Join(f.Root, ".kogen/local/shaping", shpSlug), shpSlug, fc.Now)
	if err != nil {
		t.Fatal(err)
	}
	sh.ev = ev
	sh.request, sh.role, sh.started, sh.submissions = orderRequest, sh.cfg.Models["shape"], fc.Now(), 1
	sh.rd = &verify.Readiness{Root: f.Root, Repo: f.Repo, Config: sh.cfg, Tools: fixtureTools(f.S), Evidence: sh.ev}
	for _, file := range shpFiles()["files"].([]any) {
		m := file.(map[string]any)
		f.write(m["path"].(string), m["content"].(string))
	}
	decision, literal := orderDecisionRefusal(), orderLiteralRefusal()
	f.Prov.route = func(req map[string]any) *shpStep {
		if decisionRequest(req) {
			fc.Sleep(context.Background(), 2*time.Second)
			return &decision
		}
		if literalRequest(req) {
			return &literal
		}
		return nil
	}
	// A failed test must not leave a sleeper behind for the fixture's cleanup to wait on.
	t.Cleanup(func() { fc.Advance(time.Hour) })
	started := fc.Now()
	sh.startBatch()
	done := make(chan error, 1)
	go func() { done <- sh.postSubmitChecks(context.Background(), shpScopedIntentMD, "", "overlap-package") }()
	if !fc.awaitSleepers(2) {
		t.Fatal("the static check and the decision check were not waiting at the same time: they did not run together")
	}
	fc.Advance(2 * time.Second)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	shpEq(t, "injected time of the checks", fc.Now().Sub(started), 2*time.Second)
	sh.ev.mu.Lock()
	static, models := sh.ev.intervals["static"], sh.ev.intervals["model_checks"]
	sh.ev.mu.Unlock()
	shpEq(t, "static spans", fmt.Sprint(static), "[[0 2000]]")
	shpEq(t, "model check spans", fmt.Sprint(models), "[[0 2000]]")
	shpEq(t, "time of both", Union(append(append([][2]int64{}, static...), models...)), int64(2000))
	text, _, invalid := sh.settleHeld(sh.collectBatch())
	if !invalid || !strings.HasPrefix(text, "Decision check:") || !strings.Contains(text, "\n\nLiteral check:") {
		t.Fatalf("combined feedback (refused %v): %s", invalid, text)
	}
	shpEq(t, "decision calls", len(f.Prov.decisionRequests()), 1)
	shpEq(t, "literal calls", len(f.Prov.literalRequests()), 1)
}

// controlStepFile is the step file of a control run test that never reaches Cucumber.
const controlStepFile = "throw new Error('stub');"

// TestClockBoundUnpassableControl: a control run that outlasts its bound on the injected clock is
// stopped at the bound and recorded as skipped with the bound named, and a step file that does
// not load gives its own reason; the control command is a fake and nothing waits.
func TestClockBoundUnpassableControl(t *testing.T) {
	stub := []controlProbe{{name: "stub", body: controlStepFile}}
	t.Run("the bound is exceeded on the injected clock", func(t *testing.T) {
		fc := newFakeClock()
		f, sh, ready, _ := unpassableFixture(t, "", stub, nil)
		sh.controlBound = 2 * time.Second
		var bounds []time.Duration
		sh.controlRun = func(_ context.Context, _, _ string, opts sys.Options) (sys.Result, error) {
			bounds = append(bounds, opts.Timeout)
			// A control run that wants a minute of the injected clock is stopped at its bound.
			if want := time.Minute; want > opts.Timeout {
				fc.Advance(opts.Timeout)
				return sys.Result{Status: -1, TimedOut: true}, nil
			}
			return sys.Result{}, errors.New("the fake control run was meant to exceed its bound")
		}
		started := fc.Now()
		_, reason, err := sh.unpassableControl(context.Background(), ready)
		if err != nil || reason != "the control run exceeded the 2-second bound" {
			t.Fatalf("reason %q, err %v", reason, err)
		}
		shpEq(t, "bounds", fmt.Sprint(bounds), "[2s]")
		shpEq(t, "injected time of the control run", fc.Now().Sub(started), 2*time.Second)

		if err := sh.unpassableGate(context.Background(), ready); err != nil {
			t.Fatalf("gate = %v, want no refusal", err)
		}
		events := shpNamed((&shpFx{t: t, Root: f.Root}).events(), "unpassable_check")
		if len(events) != 1 || events[0]["ok"] != true || events[0]["skipped"] != "the control run exceeded the 2-second bound" {
			t.Fatalf("events = %v", events)
		}
	})
	t.Run("a step file that does not load reports its own reason", func(t *testing.T) {
		_, sh, ready, _ := unpassableFixture(t, "throw new Error('does not load');\n", stub, nil)
		runs := 0
		sh.controlRun = func(context.Context, string, string, sys.Options) (sys.Result, error) {
			runs++
			return sys.Result{Status: 1, Output: "Error: does not load\n"}, nil
		}
		_, reason, err := sh.unpassableControl(context.Background(), ready)
		if err != nil || !strings.HasPrefix(reason, "the control run wrote no report (a step file may not load): ") || !strings.Contains(reason, "does not load") {
			t.Fatalf("reason %q, err %v", reason, err)
		}
		shpEq(t, "control runs", runs, 1)
	})
}

// TestClockBoundTimeCountsOverlapOnce: the decision and literal checks of one gate run together
// for 300 and 500 injected milliseconds beside a 400 ms static check; the shaping time line
// counts the overlap once, in injected durations.
func TestClockBoundTimeCountsOverlapOnce(t *testing.T) {
	parallel(t)
	fc := newFakeClock()
	ev, err := NewEvidenceWithClock(filepath.Join(t.TempDir(), "shaping"), "fixture", fc.Now)
	if err != nil {
		t.Fatal(err)
	}
	ev.AccountTime()
	ctx := context.Background()
	wait := func(d time.Duration) func(context.Context) ([]string, error) {
		return func(ctx context.Context) ([]string, error) { return nil, fc.Sleep(ctx, d) }
	}
	t.Cleanup(func() { fc.Advance(time.Hour) })
	submission, staticDone := make(chan error, 1), make(chan struct{})
	go func() {
		submission <- ev.Checking(func() error {
			var wg sync.WaitGroup
			var static, models error
			wg.Add(2)
			go func() {
				defer wg.Done()
				static = ev.Timed("static", func() error { return fc.Sleep(ctx, 400*time.Millisecond) })
				close(staticDone)
			}()
			go func() {
				defer wg.Done()
				models = ev.Timed("model_checks", func() error {
					_, err := runModelChecks(ctx, wait(300*time.Millisecond), wait(500*time.Millisecond), func(context.Context) ([]string, error) { return nil, nil })
					return err
				})
			}()
			wg.Wait()
			return errors.Join(static, models)
		})
	}()
	// The static check, the decision check and the literal check wait together.
	if !fc.awaitSleepers(3) {
		t.Fatal("the checks did not wait at the same time")
	}
	// The clock moves to each deadline in turn, and only after the check that ended has read it.
	fc.AdvanceToNext() // the decision check, 300 ms
	fc.AdvanceToNext() // the static check, 400 ms
	<-staticDone
	fc.AdvanceToNext() // the literal check, 500 ms
	if err := <-submission; err != nil {
		t.Fatal(err)
	}
	contract := make(chan error, 1)
	go func() {
		contract <- ev.Checking(func() error { return ev.Timed("contract", func() error { return fc.Sleep(ctx, 200*time.Millisecond) }) })
	}()
	if !fc.awaitSleepers(1) {
		t.Fatal("the contract review did not wait")
	}
	fc.AdvanceToNext()
	if err := <-contract; err != nil {
		t.Fatal(err)
	}
	if err := ev.Finish(); err != nil {
		t.Fatal(err)
	}
	a := ev.Spent()
	if a == nil {
		t.Fatal("no time account")
	}
	shpEq(t, "static", a.Static, int64(400))
	shpEq(t, "model checks (300 and 500 together)", a.ModelChecks, int64(500))
	shpEq(t, "contract review", a.Contract, int64(200))
	shpEq(t, "Kogen checks (the static check lies within the model checks)", a.Checks, int64(700))
	shpEq(t, "elapsed", a.Elapsed, int64(700))
	shpEq(t, "gap", a.Gap, int64(0))
	shpEq(t, "line", a.Line(), "kogen: shape time 0.7 s: shaper 0.0 s (model 0.0 s, tools 0.0 s), Kogen checks 0.7 s (static 0.4 s, model checks 0.5 s, contract review 0.2 s, readiness 0.0 s), gaps 0.0 s")
}

// realWaitPatterns are what a shape test may not spell: a call that sleeps on the real clock, a
// shell sleep of more than one second and a node timer of more than 1000 ms. They are built from
// pieces so that this file does not name what it refuses.
var (
	realSleepCall = regexp.MustCompile(`\b` + "time" + `\.` + "Sleep" + `\s*\(`)
	shellSleep    = regexp.MustCompile(`\b` + "sleep" + `\s+([0-9]+(?:\.[0-9]+)?)`)
	nodeTimer     = regexp.MustCompile("setTimeout" + `\([^,)]*,\s*(\d+)`)
)

// realWaits lists the lines of a Go test file that wait on the real clock for more than a second
// through a helper: they are read without their whole-line comments.
func realWaits(name, text string) []string {
	var found []string
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		where := name + ":" + strconv.Itoa(i+1)
		if realSleepCall.MatchString(line) {
			found = append(found, where+": a real sleep call")
		}
		for _, m := range shellSleep.FindAllStringSubmatch(line, -1) {
			if n, _ := strconv.ParseFloat(m[1], 64); n > 1 {
				found = append(found, where+": a shell sleep of "+m[1]+" seconds")
			}
		}
		for _, m := range nodeTimer.FindAllStringSubmatch(line, -1) {
			if n, _ := strconv.Atoi(m[1]); n > 1000 {
				found = append(found, where+": a node timer of "+m[1]+" ms")
			}
		}
	}
	return found
}

// TestClockBoundNoRealSleepsInShape fails when a shape test sleeps on the real clock or runs a
// command that waits longer than one second, and shows the scan finds each of them.
func TestClockBoundNoRealSleepsInShape(t *testing.T) {
	parallel(t)
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no shape test files found: %v", err)
	}
	var found []string
	for _, name := range files {
		text, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		found = append(found, realWaits(name, string(text))...)
	}
	if len(found) != 0 {
		t.Fatalf("shape tests wait on the real clock for more than a second; use the injected clock or a fake command:\n%s", strings.Join(found, "\n"))
	}
	// The scan itself: what it must refuse and what it lets through.
	refused := []string{
		"\t" + "time" + ".Sleep(10 * time.Millisecond)",
		"run: 'sl" + "eep 2; true'",
		"command := \"sl" + "eep 1.5\"",
		"await new Promise(r => set" + "Timeout(r, 1001));",
	}
	for _, line := range refused {
		if len(realWaits("x", line)) != 1 {
			t.Errorf("the scan lets %q through", line)
		}
	}
	allowed := []string{
		"run: 'sl" + "eep 0.2; true'",
		"while ! test -f x; do sl" + "eep 1; done",
		"await new Promise(r => set" + "Timeout(r, 1000));",
		"// " + "time" + ".Sleep(time.Hour)",
	}
	for _, line := range allowed {
		if got := realWaits("x", line); len(got) != 0 {
			t.Errorf("the scan refuses %q: %v", line, got)
		}
	}
}
