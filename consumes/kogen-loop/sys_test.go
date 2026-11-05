package consumes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/KogenAI/kogen-loop/sys"
)

// Compile-time pins: the types and signatures kogen-shaping's code relies on.
var (
	_ func(class, reason, detail string, exit int) *sys.Error                        = sys.Fail
	_ func(error) (*sys.Error, bool)                                                 = sys.AsError
	_ func(*sys.Error) string                                                        = (*sys.Error).Error
	_ func(*sys.Error) string                                                        = (*sys.Error).Render
	_ error                                                                          = (*sys.Error)(nil)
	_ func(*sys.Error) (*string, *string, *string, *[]string, *int, *map[string]any) = func(e *sys.Error) (*string, *string, *string, *[]string, *int, *map[string]any) {
		return &e.Class, &e.Reason, &e.Detail, &e.Lines, &e.Exit, &e.ProviderFailure
	}
	_ error                                               = (*sys.Interrupted)(nil)
	_ func(*sys.Interrupted) int                          = (*sys.Interrupted).ExitCode
	_ func(*sys.Interrupted) *string                      = func(i *sys.Interrupted) *string { return &i.Signal }
	_ func() *sys.Session                                 = sys.New
	_ func(prefix string) string                          = sys.TmpPath
	_ func(*sys.Session) (*io.Writer, *[]string, *string) = func(s *sys.Session) (*io.Writer, *[]string, *string) {
		return &s.Stdout, &s.Env, &s.Cwd
	}
	_ func(*sys.Session, string)                                                           = (*sys.Session).Out
	_ func(*sys.Session, string)                                                           = (*sys.Session).Puts
	_ func(*sys.Session, string)                                                           = (*sys.Session).Progress
	_ func(*sys.Session, string) string                                                    = (*sys.Session).Getenv
	_ func(*sys.Session) error                                                             = (*sys.Session).Err
	_ func(*sys.Session) func()                                                            = (*sys.Session).WatchSignals
	_ func(*sys.Session, string, string, sys.Options) (sys.Result, error)                  = (*sys.Session).Run
	_ func(*sys.Session, context.Context, string, string, sys.Options) (sys.Result, error) = (*sys.Session).RunContext
	_ func(*sys.Options) (*map[string]string, *time.Duration, *context.Context, *bool)     = func(o *sys.Options) (*map[string]string, *time.Duration, *context.Context, *bool) {
		return &o.Env, &o.Timeout, &o.Context, &o.KillGroupOnExit
	}
	_ func(*sys.Result) (*int, *string, *bool) = func(r *sys.Result) (*int, *string, *bool) {
		return &r.Status, &r.Output, &r.TimedOut
	}
)

// blockUntilKilled is a command that waits without end and without using the CPU: it opens a
// named pipe nobody writes to.
const blockUntilKilled = `mkfifo blocked && read line < blocked`

// shellSession is a Session over a working directory of its own, with the caller's PATH.
func shellSession(t *testing.T) (*sys.Session, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := sys.New()
	s.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "CALLER_VALUE=from-caller"}
	s.Cwd = dir
	return s, dir
}

func TestFail(t *testing.T) {
	e := sys.Fail("candidate", "head_moved", "HEAD is not the baseline", 1)
	if e.Class != "candidate" || e.Reason != "head_moved" || e.Detail != "HEAD is not the baseline" || e.Exit != 1 {
		t.Errorf("sys.Fail: %+v does not keep class, reason, detail and exit", e)
	}
	if len(e.Lines) != 0 || e.ProviderFailure != nil {
		t.Errorf("sys.Fail: %+v must have no detail lines and no provider failure", e)
	}
	for _, exit := range []int{1, 2, 3, 4, 5} {
		if got := sys.Fail("intent", "not_ready", "d", exit).Exit; got != exit {
			t.Errorf("sys.Error.Exit: Fail(..., %d) has exit %d", exit, got)
		}
	}
	var err error = e
	if got, want := err.Error(), "candidate/head_moved: HEAD is not the baseline"; got != want {
		t.Errorf("sys.Error.Error: %q, want %q", got, want)
	}
}

func TestErrorRender(t *testing.T) {
	e := sys.Fail("intent", "invalid_package", "two problems", 1)
	if got, want := e.Render(), "intent/invalid_package: two problems\n"; got != want {
		t.Errorf("sys.Error.Render: %q, want %q", got, want)
	}
	e.Lines = []string{"changed: a.txt", "changed: b/c.txt"}
	if got, want := e.Render(), "intent/invalid_package: two problems\n  changed: a.txt\n  changed: b/c.txt\n"; got != want {
		t.Errorf("sys.Error.Lines: Render gave %q, want %q", got, want)
	}
	if got, want := e.Error(), "intent/invalid_package: two problems"; got != want {
		t.Errorf("sys.Error.Lines: Error must not include the detail lines, got %q", got)
	}
	e.ProviderFailure = map[string]any{"http_status": 429}
	if e.ProviderFailure["http_status"] != 429 {
		t.Errorf("sys.Error.ProviderFailure: %v does not keep the diagnostics", e.ProviderFailure)
	}
}

func TestAsError(t *testing.T) {
	e := sys.Fail("shape", "refinement_invalid", "no valid correction", 1)
	for name, err := range map[string]error{
		"itself":  e,
		"wrapped": fmt.Errorf("while refining: %w", e),
		"joined":  errors.Join(errors.New("other"), e),
	} {
		got, ok := sys.AsError(err)
		if !ok || got != e {
			t.Errorf("sys.AsError: %s gave %v, %v, want the *sys.Error itself", name, got, ok)
		}
	}
	for name, err := range map[string]error{
		"nil":         nil,
		"plain":       errors.New("boom"),
		"interrupted": &sys.Interrupted{Signal: "SIGINT"},
	} {
		if got, ok := sys.AsError(err); ok || got != nil {
			t.Errorf("sys.AsError: %s gave %v, %v, want nil, false", name, got, ok)
		}
	}
	var target *sys.Error
	if !errors.As(fmt.Errorf("x: %w", e), &target) || target.Reason != "refinement_invalid" {
		t.Errorf("sys.Error: errors.As to *sys.Error failed")
	}
}

func TestInterrupted(t *testing.T) {
	for signal, code := range map[string]int{"SIGINT": 130, "SIGTERM": 143, "SIGHUP": 129} {
		var err error = &sys.Interrupted{Signal: signal}
		var i *sys.Interrupted
		if !errors.As(fmt.Errorf("stopped: %w", err), &i) || i.Signal != signal {
			t.Errorf("sys.Interrupted.Signal: errors.As lost %s", signal)
			continue
		}
		if got := i.ExitCode(); got != code {
			t.Errorf("sys.Interrupted.ExitCode: %s is %d, want %d", signal, got, code)
		}
		if got, want := err.Error(), "interrupted by "+signal; got != want {
			t.Errorf("sys.Interrupted: error text %q, want %q", got, want)
		}
	}
}

func TestNewSession(t *testing.T) {
	s := sys.New()
	if s.Stdout != io.Writer(os.Stdout) {
		t.Errorf("sys.Session.Stdout: sys.New must write to the process's standard output")
	}
	if want, _ := os.Getwd(); s.Cwd != want || s.Cwd == "" {
		t.Errorf("sys.Session.Cwd: %q, want the working directory %q", s.Cwd, want)
	}
	if got, want := strings.Join(s.Env, "\x00"), strings.Join(os.Environ(), "\x00"); got != want {
		t.Errorf("sys.Session.Env: sys.New must hold the process's environment")
	}
	if got := s.Getenv("HOME"); got != os.Getenv("HOME") || got == "" {
		t.Errorf("sys.Session.Getenv: HOME is %q, want %q", got, os.Getenv("HOME"))
	}
	s.Env = []string{"KEY=first", "OTHER=x=y", "KEY=second", "EMPTY="}
	for key, want := range map[string]string{"KEY": "second", "OTHER": "x=y", "EMPTY": "", "MISSING": ""} {
		if got := s.Getenv(key); got != want {
			t.Errorf("sys.Session.Getenv(%q) = %q, want %q", key, got, want)
		}
	}
	s.Stdout = io.Discard
	if s.Stdout != io.Discard {
		t.Errorf("sys.Session.Stdout: the writer cannot be replaced")
	}
}

func TestSessionOutput(t *testing.T) {
	s := sys.New()
	var out, errOut bytes.Buffer
	s.Stdout, s.Stderr = &out, &errOut
	s.Out("no newline")
	s.Out("|")
	s.Puts("a line")
	s.Puts("")
	if got, want := out.String(), "no newline|a line\n\n"; got != want {
		t.Errorf("sys.Session.Out/Puts: stdout %q, want %q", got, want)
	}
	s.Progress("kogen: working")
	s.Progress("kogen: done")
	if got, want := errOut.String(), "kogen: working\nkogen: done\n"; got != want {
		t.Errorf("sys.Session.Progress: stderr %q, want %q", got, want)
	}
	if got := out.String(); got != "no newline|a line\n\n" {
		t.Errorf("sys.Session.Progress: progress reached stdout: %q", got)
	}
}

func TestSessionRun(t *testing.T) {
	s, dir := shellSession(t)
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := s.Run("pwd; echo out; echo err >&2; printf '%s,%s' \"$CALLER_VALUE\" \"$ADDED_VALUE\"; exit 3", sub, sys.Options{Env: map[string]string{"ADDED_VALUE": "from-options"}})
	if err != nil {
		t.Fatalf("sys.Session.Run: %v", err)
	}
	if want := sub + "\nout\nerr\nfrom-caller,from-options"; res.Output != want {
		t.Errorf("sys.Session.Run: output %q, want %q (working directory, combined streams, caller and Options.Env)", res.Output, want)
	}
	if res.Status != 3 || res.TimedOut {
		t.Errorf("sys.Result.Status: %+v, want status 3 and no timeout", res)
	}

	res, err = s.Run("echo fine", dir, sys.Options{})
	if err != nil || res.Status != 0 || res.Output != "fine\n" {
		t.Errorf("sys.Result.Output: %+v, %v, want status 0 and output \"fine\\n\"", res, err)
	}
	res, err = s.Run("kill -9 $$", dir, sys.Options{})
	if err != nil || res.Status != 128+9 {
		t.Errorf("sys.Result.Status: a killed command is %+v, %v, want status 137", res, err)
	}
	res, err = s.Run("echo within the limit", dir, sys.Options{Timeout: time.Minute})
	if err != nil || res.TimedOut || res.Status != 0 {
		t.Errorf("sys.Options.Timeout: a command within its limit gave %+v, %v", res, err)
	}
}

func TestSessionRunTimeout(t *testing.T) {
	s, dir := shellSession(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := s.Run(blockUntilKilled, dir, sys.Options{Timeout: 300 * time.Millisecond, Context: ctx, KillGroupOnExit: true})
	if err != nil {
		t.Fatalf("sys.Options.Timeout: %v", err)
	}
	if !res.TimedOut || res.Status == 0 {
		t.Errorf("sys.Result.TimedOut: %+v, want a timed-out command with a nonzero status", res)
	}
}

func TestSessionRunContext(t *testing.T) {
	s, dir := shellSession(t)
	done, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := s.RunContext(done, "touch ran", dir, sys.Options{Context: done})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("sys.Session.RunContext: a done context gave %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "ran")); statErr == nil {
		t.Errorf("sys.Session.RunContext: the command started although the context was done (%+v)", res)
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	res, err = s.RunContext(ctx, "echo $ADDED_VALUE", dir, sys.Options{
		Env: map[string]string{"ADDED_VALUE": "kept"}, Timeout: time.Minute, Context: ctx, KillGroupOnExit: true,
	})
	if err != nil || res.Output != "kept\n" || res.Status != 0 || res.TimedOut {
		t.Errorf("sys.Session.RunContext: the options of the unpassable control gave %+v, %v", res, err)
	}

	deadline, stopDeadline := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer stopDeadline()
	if _, err := s.RunContext(deadline, blockUntilKilled, dir, sys.Options{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("sys.Session.RunContext: a context that ends a running command gave %v, want context.DeadlineExceeded", err)
	}
}

// running reports whether process pid runs. kill(pid, 0) also succeeds on a zombie, a process that
// has exited but that its parent has not reaped (under a parent that is a child subreaper, a
// killed group member stays one until the subreaper reaps it), so where /proc/<pid>/stat can be
// read a process whose state field, the first after the last ')', is Z is gone. Without /proc
// kill(pid, 0) decides.
//
// Follow-up in kogen-loop, not edited here: contract/sys_test.go TestRunKillGroupOnExit has the same
// weakness, its waitGone polls kill(-pgid, 0) until ESRCH, which an unreaped zombie in the group
// never gives.
func running(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return true
	}
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 {
		return true
	}
	fields := strings.Fields(string(stat[end+1:]))
	return len(fields) == 0 || fields[0] != "Z"
}

// checkKillGroupOnExit runs a command that leaves a process behind in its group, once with
// KillGroupOnExit, where the process must not be running afterwards, and once without, where it must.
func checkKillGroupOnExit(t *testing.T) {
	t.Helper()
	s, dir := shellSession(t)
	// The command leaves a process behind in its group and prints its process ID.
	const leaveBehind = `mkfifo blocked; (read line < blocked) >/dev/null 2>&1 </dev/null & echo $!`
	pidOf := func(res sys.Result) int {
		var pid int
		if _, err := fmt.Sscanf(res.Output, "%d", &pid); err != nil || pid <= 1 {
			t.Fatalf("sys.Options.KillGroupOnExit: no process ID in output %q", res.Output)
		}
		return pid
	}

	res, err := s.Run(leaveBehind, dir, sys.Options{KillGroupOnExit: true})
	if err != nil || res.Status != 0 {
		t.Fatalf("sys.Options.KillGroupOnExit: %+v, %v", res, err)
	}
	if pid := pidOf(res); running(pid) {
		syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("sys.Options.KillGroupOnExit: process %d of the command's group is still running after Run", pid)
	}

	if err := os.Remove(filepath.Join(dir, "blocked")); err != nil {
		t.Fatal(err)
	}
	res, err = s.Run(leaveBehind, dir, sys.Options{})
	if err != nil {
		t.Fatalf("sys.Session.Run: %v", err)
	}
	pid := pidOf(res)
	left := running(pid)
	syscall.Kill(pid, syscall.SIGKILL)
	if !left {
		t.Errorf("sys.Options.KillGroupOnExit: without the option process %d must outlive its command", pid)
	}
}

func TestKillGroupOnExit(t *testing.T) {
	checkKillGroupOnExit(t)
}

// childSubreaperEnv marks the re-run of the test binary that runs the check as a child subreaper.
const childSubreaperEnv = "CONSUMES_KILL_GROUP_CHILD"

// TestKillGroupOnExitUnderChildSubreaper runs the kill-group check in a child process that is a
// child subreaper and never reaps, so the group member KillGroupOnExit killed stays a zombie of it.
// The test process itself never becomes a subreaper.
func TestKillGroupOnExitUnderChildSubreaper(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("child subreapers exist on Linux only")
	}
	if os.Getenv(childSubreaperEnv) == "1" {
		if err := becomeChildSubreaper(); err != nil {
			t.Fatalf("prctl(PR_SET_CHILD_SUBREAPER): %v", err)
		}
		checkKillGroupOnExit(t)
		return
	}
	const name = "TestKillGroupOnExitUnderChildSubreaper"
	cmd := exec.Command(os.Args[0], "-test.run=^"+name+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), childSubreaperEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary as a child subreaper failed: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("--- PASS: "+name)) {
		t.Errorf("the test binary as a child subreaper did not report %s as passed:\n%s", name, out)
	}
}

func TestWatchSignals(t *testing.T) {
	s := sys.New()
	if err := s.Err(); err != nil {
		t.Fatalf("sys.Session.Err: a new session reports %v", err)
	}
	stop := s.WatchSignals()
	defer stop()
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Context().Done():
	case <-time.After(30 * time.Second):
		t.Fatalf("sys.Session.WatchSignals: SIGTERM did not interrupt the session")
	}
	err := s.Err()
	var i *sys.Interrupted
	if !errors.As(err, &i) || i.Signal != "SIGTERM" || i.ExitCode() != 143 {
		t.Errorf("sys.Session.Err: after SIGTERM got %v, want *sys.Interrupted SIGTERM with exit code 143", err)
	}
	if again := s.Err(); again == nil {
		t.Errorf("sys.Session.Err: the interrupt did not stay")
	}
}

func TestTmpPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	path := sys.TmpPath("kogen-context-index")
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(filepath.Join(os.TempDir(), "kogen-context-index-")) + hexSuffix + `$`)
	if !pattern.MatchString(path) || filepath.Dir(path) != filepath.Clean(os.TempDir()) {
		t.Errorf("sys.TmpPath: %q is not <tmp>/kogen-context-index-<16 hex>", path)
	}
	if _, err := os.Lstat(path); err == nil {
		t.Errorf("sys.TmpPath: %q already exists; the path must only be unused", path)
	}
	seen := map[string]bool{path: true}
	for range 50 {
		next := sys.TmpPath("kogen-context-index")
		if seen[next] {
			t.Fatalf("sys.TmpPath: %q repeated", next)
		}
		seen[next] = true
	}
	if other := sys.TmpPath("kogen-refine"); !strings.HasPrefix(filepath.Base(other), "kogen-refine-") {
		t.Errorf("sys.TmpPath: %q does not start with its prefix", other)
	}
}
