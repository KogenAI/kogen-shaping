'use strict';
// Harness of FIX-K-SHAPE-KILL-GROUP-ZOMBIE. It runs kill-group tests of consumes/kogen-loop with the
// Go tool and judges what `go test -v` reports. On Linux every run goes through a small Go program
// that makes itself a child subreaper (prctl PR_SET_CHILD_SUBREAPER) and then either never reaps an
// orphaned process (`hold`, as the Linux Build supervisor does) or reaps each one at once (`reap`,
// as init does). It needs no Cucumber, so the control can use it too.
const fs = require('fs');
const os = require('os');
const path = require('path');
const {spawnSync} = require('child_process');

const PKG = './consumes/kogen-loop/';
const LOOP = 'github.com/KogenAI/kogen-loop';
const RUNNING_TEXT = "of the command's group is still running after Run";

// The top-level tests of consumes/kogen-loop before this change (`go test -list .` at 4f0c335).
const BASELINE_TESTS = [
  'TestUsageAddLine', 'TestKnownUsageMap', 'TestResponseFields', 'TestNewClientVersion',
  'TestRoleRoutesBody', 'TestRequestBody', 'TestClientRespondRefusals', 'TestRequireLogin',
  'TestSupportsToolChoice', 'TestContractBlocks', 'TestNewCacheID', 'TestObserverInterface',
  'TestFail', 'TestErrorRender', 'TestAsError', 'TestInterrupted', 'TestNewSession',
  'TestSessionOutput', 'TestSessionRun', 'TestSessionRunTimeout', 'TestSessionRunContext',
  'TestKillGroupOnExit', 'TestWatchSignals', 'TestTmpPath', 'TestToolNames',
  'TestInputBudgetBytes', 'TestOutputReserveTokens', 'TestDefinition', 'TestCallsAndOutput',
  'TestDecodeArguments', 'TestReplay', 'TestCut', 'TestOutputCut', 'TestMeasure', 'TestToolsRun',
  'TestToolsRepo', 'TestToolsSession', 'TestToolsProbe', 'TestToolsDocumentation',
  'TestViewImageAndReadEvent', 'TestDeliver',
];

// The text kogen-loop's Session.runCaptured holds once; removing it leaves the command's group
// running after Run although KillGroupOnExit is set.
const SWEEP = '\tif opts.KillGroupOnExit {\n\t\tsweepGroup(pgid)\n\t}\n';

const SUBREAPER_SOURCE = `package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// subreaper hold|reap command [arg...] makes itself a child subreaper, runs the command and exits
// with its status. hold never reaps an orphaned process; reap reaps each one at once.
func main() {
	if len(os.Args) < 3 || (os.Args[1] != "hold" && os.Args[1] != "reap") {
		fmt.Fprintln(os.Stderr, "usage: subreaper hold|reap command [arg...]")
		os.Exit(2)
	}
	// PR_SET_CHILD_SUBREAPER is 36; the syscall package does not define it on every architecture.
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, 36, 1, 0); errno != 0 {
		fmt.Fprintln(os.Stderr, "subreaper: prctl(PR_SET_CHILD_SUBREAPER):", errno)
		os.Exit(2)
	}
	file, err := exec.LookPath(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "subreaper:", err)
		os.Exit(2)
	}
	pid, err := syscall.ForkExec(file, os.Args[2:], &syscall.ProcAttr{Env: os.Environ(), Files: []uintptr{0, 1, 2}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "subreaper:", err)
		os.Exit(2)
	}
	target := pid
	if os.Args[1] == "reap" {
		target = -1
	}
	for {
		var ws syscall.WaitStatus
		got, err := syscall.Wait4(target, &ws, 0, nil)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "subreaper: wait:", err)
			os.Exit(2)
		}
		if got != pid {
			continue
		}
		if ws.Signaled() {
			os.Exit(128 + int(ws.Signal()))
		}
		os.Exit(ws.ExitStatus())
	}
}
`;

const isLinux = () => process.platform === 'linux';

function goEnv(extra) {
  return {...process.env, GOFLAGS: '', GOWORK: 'off', ...extra};
}

function tail(text) {
  return text.length > 8000 ? '...' + text.slice(-8000) : text;
}

// run runs argv in dir and returns its exit status and its standard output followed by its
// standard error.
function run(dir, argv, extraEnv) {
  const r = spawnSync(argv[0], argv.slice(1), {
    cwd: dir, env: goEnv(extraEnv), encoding: 'utf8', maxBuffer: 256 * 1024 * 1024, timeout: 20 * 60 * 1000,
  });
  if (r.error) throw new Error(`${argv.join(' ')}: ${r.error.message}`);
  return {command: argv.join(' '), status: r.status, signal: r.signal, output: (r.stdout || '') + (r.stderr || '')};
}

function must(r) {
  if (r.status !== 0) throw new Error(`${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}:\n${tail(r.output)}`);
  return r;
}

// subreaperSource writes the subreaper program into a new directory and returns it.
function subreaperSource() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'fix-k-shape-kill-group-zombie-subreaper-'));
  fs.writeFileSync(path.join(dir, 'main.go'), SUBREAPER_SOURCE);
  fs.writeFileSync(path.join(dir, 'go.mod'), 'module subreaper\n\ngo 1.27\n');
  return dir;
}

let builtSubreaper = null;

// subreaper is the path of the subreaper program, built once for this platform (Linux only).
function subreaper() {
  if (!builtSubreaper) {
    const dir = subreaperSource();
    must(run(dir, ['go', 'build', '-o', 'subreaper', '.']));
    builtSubreaper = path.join(dir, 'subreaper');
  }
  return builtSubreaper;
}

// buildSubreaperFor cross-builds the subreaper program for linux/<arch> and returns the run.
function buildSubreaperFor(arch) {
  const dir = subreaperSource();
  return run(dir, ['go', 'build', '-o', 'subreaper', '.'], {GOOS: 'linux', GOARCH: arch, CGO_ENABLED: '0'});
}

// newTests are the top-level tests of the package in dir that the unchanged package does not have.
function newTests(dir) {
  const r = must(run(dir, ['go', 'test', '-count=1', '-list', '.', PKG]));
  return r.output.split('\n').filter(l => /^Test\w*$/.test(l) && !BASELINE_TESTS.includes(l));
}

// reports are the top-level `--- PASS|FAIL|SKIP: Name (Ns)` lines of go test -v, in order.
function reports(output) {
  const out = [];
  for (const m of output.matchAll(/^--- (PASS|FAIL|SKIP): (\S+) \((\d+(?:\.\d+)?)s\)$/gm)) {
    out.push({result: {PASS: 'passed', FAIL: 'failed', SKIP: 'skipped'}[m[1]], name: m[2], seconds: Number(m[3])});
  }
  return out;
}

// runTests runs the named tests of the package in dir verbosely, count times. mode is hold, reap
// or plain: on Linux hold and reap run under the subreaper program; plain, and every mode
// elsewhere, run go test directly.
function runTests(dir, names, count, mode) {
  let argv = ['go', 'test', `-count=${count}`, '-v', '-run', '^(' + names.join('|') + ')$', PKG];
  if (isLinux() && mode !== 'plain') argv = [subreaper(), mode, ...argv];
  const r = run(dir, argv);
  r.reports = reports(r.output);
  return r;
}

function copyTree(from, to) {
  fs.cpSync(from, to, {recursive: true});
  const writable = p => {
    const st = fs.lstatSync(p);
    if (st.isSymbolicLink()) return;
    fs.chmodSync(p, st.mode | 0o200);
    if (st.isDirectory()) for (const e of fs.readdirSync(p)) writable(path.join(p, e));
  };
  writable(to);
}

// scratchCopy is a new module directory holding the checkout's go.mod, go.sum and
// consumes/kogen-loop, built against a copy of the pinned kogen-loop. With keepRunning the copy
// of kogen-loop does not sweep the command's group, so a process it leaves behind keeps running.
// plant adds the named Go test files to the package.
function scratchCopy(root, {keepRunning = false, plant = {}} = {}) {
  const top = fs.mkdtempSync(path.join(os.tmpdir(), 'fix-k-shape-kill-group-zombie-copy-'));
  const mod = path.join(top, 'mod');
  const loop = path.join(top, 'loop');
  fs.mkdirSync(path.join(mod, 'consumes'), {recursive: true});
  for (const f of ['go.mod', 'go.sum']) fs.copyFileSync(path.join(root, f), path.join(mod, f));
  copyTree(path.join(root, 'consumes', 'kogen-loop'), path.join(mod, 'consumes', 'kogen-loop'));
  const loopDir = must(run(root, ['go', 'list', '-m', '-f', '{{.Dir}}', LOOP])).output.trim();
  copyTree(loopDir, loop);
  if (keepRunning) {
    const sysGo = path.join(loop, 'sys', 'sys.go');
    const text = fs.readFileSync(sysGo, 'utf8');
    const n = text.split(SWEEP).length - 1;
    if (n !== 1) throw new Error(`${LOOP} sys/sys.go holds the KillGroupOnExit sweep ${n} times, want once; the copy cannot keep the group running`);
    fs.writeFileSync(sysGo, text.replace(SWEEP, ''));
  }
  for (const [name, source] of Object.entries(plant)) fs.writeFileSync(path.join(mod, 'consumes', 'kogen-loop', name), source);
  must(run(mod, ['go', 'mod', 'edit', `-replace=${LOOP}=${loop}`]));
  return mod;
}

// The judges return problems; an empty list means the run shows what is wanted.

function describe(r) {
  return `${r.command} exited ${r.status}${r.signal ? ' (' + r.signal + ')' : ''}; its output:\n${tail(r.output)}`;
}

function exitProblems(r, code) {
  return r.status === code ? [] : [`want exit ${code}: ${describe(r)}`];
}

function reportedProblems(r, name, result, times = 1) {
  const mine = r.reports.filter(x => x.name === name);
  const got = mine.filter(x => x.result === result).length;
  if (got === times && mine.length === times) return [];
  return [`${name} is reported ${mine.map(x => x.result).join(', ') || 'never'}; want ${result} ${times} time(s): ${describe(r)}`];
}

function failedWithProblems(r, name, text) {
  const problems = reportedProblems(r, name, 'failed');
  if (!r.output.includes(text)) problems.push(`the output does not contain ${JSON.stringify(text)}: ${describe(r)}`);
  return problems;
}

function oneNewTestProblems(names) {
  if (names.length === 1) return [];
  return [`the package has ${names.length} top-level tests that the unchanged package does not have (${names.join(', ') || 'none'}); want exactly one (go test -count=1 -list . ${PKG})`];
}

// newTestProblems: every new test is reported with result, at least once; with atLeast, each of its
// reports took at least that many seconds.
function newTestProblems(r, names, result, atLeast) {
  const problems = oneNewTestProblems(names);
  for (const name of names) {
    const mine = r.reports.filter(x => x.name === name);
    if (!mine.length || mine.some(x => x.result !== result)) {
      problems.push(`the new test ${name} is reported ${mine.map(x => x.result).join(', ') || 'never'}; want ${result}: ${describe(r)}`);
    } else if (atLeast !== undefined && mine.some(x => x.seconds < atLeast)) {
      problems.push(`the new test ${name} took ${mine.map(x => x.seconds + 's').join(', ')}; want at least ${atLeast}s, as a kill-group check takes while a child subreaper holds the killed group member as a zombie: ${describe(r)}`);
    }
  }
  return problems;
}

function durationProblems(r, name, {atLeast, lessThan}) {
  const mine = r.reports.filter(x => x.name === name);
  if (!mine.length) return [`${name} is never reported: ${describe(r)}`];
  const bad = mine.filter(x => (atLeast !== undefined && x.seconds < atLeast) || (lessThan !== undefined && x.seconds >= lessThan));
  if (!bad.length) return [];
  const want = atLeast !== undefined ? `at least ${atLeast}s` : `less than ${lessThan}s`;
  return [`${name} took ${mine.map(x => x.seconds + 's').join(', ')}; want ${want} each: ${describe(r)}`];
}

module.exports = {
  PKG, RUNNING_TEXT, BASELINE_TESTS, isLinux, run, must, tail, newTests, reports, runTests,
  scratchCopy, buildSubreaperFor, exitProblems, reportedProblems, failedWithProblems,
  oneNewTestProblems, newTestProblems, durationProblems,
};
