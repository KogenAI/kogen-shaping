package shape

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

func TestWholeSuiteRunner(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"go shell", `execSync('go test ./...')`, "go test"},
		{"go shell with env, chain and flags", `execSync("CGO_ENABLED=0 GOFLAGS=-v make x; cd a && go test -count=1 ./... | tee out")`, "go test"},
		{"go after ||", "execSync('false || go test ./...')", "go test"},
		{"go argv", `spawnSync('go', ['test', '-count=1', './...'])`, "go test"},
		{"two concurrent go suites", `spawn('go', ['test', './...']); spawn('go', ['test', './...'])`, "go test"},
		{"mix shell", `execSync('mix test')`, "mix test"},
		{"mix flags only", `spawnSync('mix', ['test', '--trace', '--cover'])`, "mix test"},
		{"npm shell", `execSync('npm test')`, "npm test"},
		{"npm argv", `spawnSync('npm', ['test'])`, "npm test"},
		{"pnpm run test", `execSync('pnpm run test --silent')`, "pnpm test"},
		{"yarn test", `spawnSync('yarn', ['test'])`, "yarn test"},
		{"first runner wins", `execSync('mix test'); execSync('go test ./...')`, "mix test"},

		{"go one package", `execSync('go test ./shape')`, ""},
		{"go -run", `spawnSync('go', ['test', '-run', '^TestOne$', './...'])`, ""},
		{"go -run=", `execSync("go test -run='^TestOne$' ./...")`, ""},
		{"go -test.run", `execSync('go test -test.run TestOne ./...')`, ""},
		{"go build", `execSync('go build ./...')`, ""},
		{"mix file", `execSync('mix test test/one_test.exs')`, ""},
		{"mix only", `spawnSync('mix', ['test', '--only', 'tag'])`, ""},
		{"mix failed", `execSync('mix test --failed')`, ""},
		{"mix stale", `execSync('mix test --stale')`, ""},
		{"npm forwarded", `spawnSync('npm', ['test', '--', '--grep', 'one'])`, ""},
		{"npm run other script", `execSync('npm run lint')`, ""},
		{"npm test with a value", `execSync('npm test one.spec.js')`, ""},
		{"argument list with a variable", `spawnSync('go', ['test', pkg])`, ""},
		{"argument list with a spread", `spawnSync('npm', ['test', ...extra])`, ""},
		{"template with interpolation", "execSync(`go test ./... ${flags}`)", ""},
		{"comment only", "// go test ./...\n/* mix test */ run()", ""},
		{"helper call", `runSuite()`, ""},
		{"quoted operator", `execSync("go test -run 'a|b' ./...")`, ""},
	}
	for _, c := range cases {
		if got := wholeSuiteRunner(c.body); got != c.want {
			t.Errorf("%s: wholeSuiteRunner(%q) = %q, want %q", c.name, c.body, got, c.want)
		}
	}
}

func TestStepLimitItemFollowsStepTimeout(t *testing.T) {
	d := verify.StepDef{File: "features/steps/x__steps.js", Pattern: "x: I run the suite", Body: `execSync('go test ./...')`}
	it, ok := stepLimitItem(d)
	if !ok {
		t.Fatal("stepLimitItem: a whole-suite body must be refused")
	}
	want := fmt.Sprintf("runs a whole test suite with go test and no narrowing; narrow it to one named test or package, or split the step, so it finishes below the %d s step limit (for example go test -run '^TestName$' ./package)", int(verify.StepTimeout.Seconds()))
	if it.Check != "step_limit" || it.Target != d.Pattern || it.File != d.File || it.Detail != want || it.OK || it.Warning {
		t.Errorf("stepLimitItem = %+v, want check step_limit, target %q, file %q, detail %q", it, d.Pattern, d.File, want)
	}
	d.Body = `execSync('go test ./shape')`
	if it, ok := stepLimitItem(d); ok || strings.Contains(it.Detail, "step limit") {
		t.Errorf("stepLimitItem: a one-package body must be accepted, got %+v", it)
	}
}

// Command shapes a step body wraps around a whole-suite run. Each one runs the whole suite and
// must be refused; the narrowed twins below each must stay accepted.
func TestWholeSuiteRunnerWrappedCommands(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"subshell", `execSync('(cd app && mix test)')`, "mix test"},
		{"sh -c", `execSync("sh -c 'cd app && mix test'")`, "mix test"},
		{"bash -lc", `execSync('bash -lc "go test -count=1 ./..."')`, "go test"},
		{"env prefix", `execSync('env MIX_ENV=test mix test')`, "mix test"},
		{"timeout prefix", `execSync('timeout 100 mix test')`, "mix test"},
		{"nice prefix", `execSync('nice -n 10 go test ./...')`, "go test"},
		{"stderr redirect", `execSync('mix test 2>&1')`, "mix test"},
		{"output redirect", `execSync('npm test > out.log')`, "npm test"},
		{"mix exclude value", `execSync('mix test --exclude integration')`, "mix test"},
		{"mix seed value", `spawnSync('mix', ['test', '--seed', '0'])`, "mix test"},

		{"subshell narrowed", `execSync('(cd app && mix test test/one_test.exs)')`, ""},
		{"sh -c narrowed", `execSync("sh -c 'mix test --only tag'")`, ""},
		{"env prefix narrowed", `execSync('env MIX_ENV=test go test -run TestOne ./...')`, ""},
		{"redirect narrowed", `execSync('mix test test/one_test.exs 2>&1')`, ""},
		{"mix exclude with file", `execSync('mix test --exclude slow test/one_test.exs')`, ""},
		{"timeout other command", `execSync('timeout 100 make lint')`, ""},
	}
	for _, c := range cases {
		if got := wholeSuiteRunner(c.body); got != c.want {
			t.Errorf("%s: wholeSuiteRunner(%q) = %q, want %q", c.name, c.body, got, c.want)
		}
	}
}
