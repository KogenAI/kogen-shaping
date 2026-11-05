# consumes/kogen-loop

What kogen-shaping relies on from kogen-loop, as tests. kogen-shaping depends on kogen-loop through
the Go module `github.com/KogenAI/kogen-loop`, at the
version `go.mod` pins. The suite is the consumer's side of the contract: if a kogen-loop change
breaks something here, kogen-shaping breaks with it.

## What it pins

Only what kogen-shaping's own code (`shape/*.go`, `cmd/kogen-shape/main.go`) uses of
`github.com/KogenAI/kogen-loop/provider`, `github.com/KogenAI/kogen-loop/sys` and
`github.com/KogenAI/kogen-loop/tools`, and nothing else:

- Compile-time pins with their types written out (`var _ func(...) ... = pkg.Name`, and one
  pointer-returning function per struct, so a field's type is pinned too) fix the signature of
  every function and method, the type of every field and the interface `provider.Observer`.
  Changing a type or signature of what kogen-shaping uses stops the package from compiling.
- Real calls fix the behaviour, values and formats the call sites depend on: the request body
  `provider.Client.Body` renders for each field of `provider.Request` and `provider.Role` and the
  cache identity of `Scope` and `CacheID`; the offline refusals of `provider.Client.Respond` and
  `provider.RequireLogin` (provider class, exit 4, reasons `login`, `refused` and
  `unsupported_capability`); `provider.Usage.Add` and `Line`; `provider.KnownUsage.Map`; the stops
  made by `sys.Fail` (class, reason, detail, lines, exit, `Render`) and the exit codes of
  `sys.Interrupted`; `sys.Session` output, environment, interrupts (`WatchSignals`) and
  `Run`/`RunContext` with every `sys.Options` field kogen-shaping sets; `sys.TmpPath`; the tool
  names, the exact bytes of their definitions (`tools.Definition`), `tools.Calls`, `Call.Args`,
  `Call.Output`, `tools.Replay`, `tools.DecodeArguments`, `tools.Cut` and `Output.Cut`,
  (as the pinned version defines them: `list_files` and `search_text` say how a path that is not
  plain text is printed; `Cut` keeps whole lines in its head when the text has them and its marker
  then also counts the lines omitted; `Repo` reads NUL-separated `git ls-files -z` and
  `git grep -z -n` output and filters the matches with the pattern itself),
  `tools.Measure` against the inclusive `tools.InputBudgetBytes` (equal to
  `provider.InputBudgetBytes`), `tools.OutputReserveTokens`, `tools.Tools.Run` with `Root`,
  `Enabled`, `Repo`, `Session`, `Probe` and `Documentation`, and `tools.ReadEvent` and
  `tools.Deliver` for a PNG written in a temporary directory.
- Uses that cannot run offline are pinned by signature and by the offline outcome kogen-shaping
  relies on: `provider.Client.Respond` is never sent (its refusals before any transport are
  asserted, and no `provider.Observer` record is delivered for them), `provider.RequireLogin`
  refuses a fresh `HOME` as `provider/login` with exit 4, and `tools.DocumentationTool` is run
  with an operation the tool refuses without any request.

The suite asserts nothing kogen-shaping does not use. Other exported identifiers of kogen-loop
(`tools.Git` and `tools.GitResult` for a fake `Tools.Repo`, the fields of `provider.KnownUsage`,
...) appear only to arrange or observe a use and are not in the inventory.

## Running

```sh
go test -race -count=1 ./consumes/kogen-loop/
```

It also runs in `go test ./...`. Nothing is skipped and no test sleeps; the slowest tests wait
for a 300 ms timeout of a command that blocks on a named pipe.

## Isolation

`TestMain` unsets every inherited `GIT_*` and `KOGEN_*` variable and `EMAIL`, sets a `HOME` of its
own (`os.MkdirTemp`, removed afterwards), `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`
and `GIT_TERMINAL_PROMPT=0`. The tests use no network (no listener, no HTTP server, no `net`
package, no fixed port), write only under `t.TempDir()`, read no host path (not kogen-shaping's
sources, not the module cache, not another repository), share no state and run in any order:
`go test -race -count=1 -shuffle=on ./consumes/kogen-loop/` and `-trimpath` pass. The only
processes are `/bin/sh` commands run through `sys.Session` and one `git init` in a temporary
directory. The suite imports only the standard library and the exported packages of kogen-loop;
it never imports a kogen-shaping package or `github.com/KogenAI/kogen-loop/internal/...`.

## Coverage check

The uses of kogen-loop in kogen-shaping's non-test code are listed with go/types: every
identifier of the non-test Go files of the packages of module `github.com/KogenAI/kogen-shaping`
outside `contract/`, `consumes/` and `features/` that resolves (`Uses` or `Selections`) to an
exported package-level object of a package of the module, `pkg.Name`, or to a method or field of
one of its named types, `pkg.Type.Member`, is a use. This yields the 107 uses of the inventory
below. The same type check of this suite's test files lists the identifiers its tests name; the
two sets are equal, and the inventory has exactly one row per use. The acceptance observer
(`features/steps/k-consumes-shaping__observer.js`) performs this comparison; its `-list` option
prints the uses:

```sh
d=$(mktemp -d) && node -e "process.stdout.write(require('./features/steps/k-consumes-shaping__observer.js').source)" > "$d/main.go" && printf 'module observer\n\ngo 1.27\n' > "$d/go.mod" && (cd "$d" && GOFLAGS= GOWORK=off go build -o observer .) && "$d/observer" -list . kogen-coordinator=github.com/KogenAI/kogen-coordinator kogen-executor=github.com/KogenAI/kogen-executor kogen-loop=github.com/KogenAI/kogen-loop kogen-verification=github.com/KogenAI/kogen-verification
```

## Inventory

| Use | Tests |
| --- | --- |
| `provider.Client` | TestNewClientVersion |
| `provider.Client.Body` | TestRequestBody, TestRoleRoutesBody, TestMeasure |
| `provider.Client.Respond` | TestClientRespondRefusals |
| `provider.Client.Version` | TestNewClientVersion |
| `provider.ContractBlocks` | TestContractBlocks |
| `provider.InputBudgetBytes` | TestInputBudgetBytes |
| `provider.KnownUsage.Map` | TestKnownUsageMap |
| `provider.NewCacheID` | TestNewCacheID |
| `provider.NewClient` | TestNewClientVersion |
| `provider.Observer` | TestObserverInterface |
| `provider.Request` | TestRequestBody, TestClientRespondRefusals |
| `provider.Request.CacheID` | TestRequestBody, TestNewCacheID |
| `provider.Request.Context` | TestRequestBody |
| `provider.Request.Input` | TestRequestBody |
| `provider.Request.Instructions` | TestRequestBody |
| `provider.Request.Role` | TestRequestBody, TestRoleRoutesBody |
| `provider.Request.Scope` | TestRequestBody |
| `provider.Request.ToolChoice` | TestRequestBody, TestClientRespondRefusals |
| `provider.Request.Tools` | TestRequestBody |
| `provider.RequireLogin` | TestRequireLogin |
| `provider.Response` | TestResponseFields, TestDeliver |
| `provider.Response.Items` | TestResponseFields |
| `provider.Response.Known` | TestResponseFields |
| `provider.Response.Usage` | TestDeliver, TestResponseFields |
| `provider.Role` | TestRoleRoutesBody, TestRequireLogin, TestSupportsToolChoice |
| `provider.Role.BaseURL` | TestRoleRoutesBody, TestRequireLogin |
| `provider.Role.Effort` | TestRoleRoutesBody |
| `provider.Role.Model` | TestRoleRoutesBody |
| `provider.Role.Provider` | TestRoleRoutesBody, TestRequireLogin, TestSupportsToolChoice |
| `provider.SupportsToolChoice` | TestSupportsToolChoice |
| `provider.Usage` | TestUsageAddLine, TestResponseFields |
| `provider.Usage.Add` | TestUsageAddLine |
| `provider.Usage.Cached` | TestUsageAddLine |
| `provider.Usage.Input` | TestUsageAddLine |
| `provider.Usage.Line` | TestResponseFields, TestUsageAddLine |
| `provider.Usage.Output` | TestUsageAddLine |
| `provider.Usage.Reasoning` | TestUsageAddLine |
| `sys.AsError` | TestAsError, TestRequireLogin, TestClientRespondRefusals |
| `sys.Error` | TestAsError, TestRequireLogin |
| `sys.Error.Class` | TestFail, TestRequireLogin, TestClientRespondRefusals |
| `sys.Error.Detail` | TestFail, TestRequireLogin, TestClientRespondRefusals |
| `sys.Error.Error` | TestErrorRender |
| `sys.Error.Exit` | TestFail, TestRequireLogin, TestClientRespondRefusals |
| `sys.Error.Lines` | TestErrorRender |
| `sys.Error.ProviderFailure` | TestErrorRender, TestDeliver |
| `sys.Error.Reason` | TestFail, TestRequireLogin, TestClientRespondRefusals |
| `sys.Error.Render` | TestErrorRender |
| `sys.Fail` | TestFail, TestErrorRender, TestAsError |
| `sys.Interrupted` | TestInterrupted, TestWatchSignals |
| `sys.Interrupted.ExitCode` | TestInterrupted, TestWatchSignals |
| `sys.Interrupted.Signal` | TestInterrupted, TestWatchSignals |
| `sys.New` | TestNewSession, TestSessionRun |
| `sys.Options` | TestSessionRun, TestSessionRunTimeout, TestSessionRunContext |
| `sys.Options.Context` | TestSessionRunTimeout, TestSessionRunContext |
| `sys.Options.Env` | TestSessionRun, TestSessionRunContext |
| `sys.Options.KillGroupOnExit` | TestKillGroupOnExit |
| `sys.Options.Timeout` | TestSessionRunTimeout, TestSessionRun |
| `sys.Result.Output` | TestSessionRun |
| `sys.Result.Status` | TestSessionRun, TestSessionRunTimeout |
| `sys.Result.TimedOut` | TestSessionRunTimeout |
| `sys.Session` | TestSessionRun |
| `sys.Session.Cwd` | TestNewSession, TestSessionRun |
| `sys.Session.Env` | TestNewSession, TestSessionRun |
| `sys.Session.Err` | TestWatchSignals |
| `sys.Session.Getenv` | TestNewSession |
| `sys.Session.Out` | TestSessionOutput |
| `sys.Session.Progress` | TestSessionOutput |
| `sys.Session.Puts` | TestSessionOutput |
| `sys.Session.Run` | TestSessionRun, TestSessionRunTimeout, TestKillGroupOnExit |
| `sys.Session.RunContext` | TestSessionRunContext |
| `sys.Session.Stdout` | TestNewSession, TestSessionOutput |
| `sys.Session.WatchSignals` | TestWatchSignals |
| `sys.TmpPath` | TestTmpPath |
| `tools.Call` | TestDeliver, TestToolsRun, TestViewImageAndReadEvent |
| `tools.Call.Args` | TestCallsAndOutput, TestToolsRun |
| `tools.Call.Name` | TestCallsAndOutput |
| `tools.Call.Output` | TestCallsAndOutput |
| `tools.Calls` | TestCallsAndOutput, TestResponseFields |
| `tools.Cut` | TestCut, TestOutputCut |
| `tools.DecodeArguments` | TestDecodeArguments |
| `tools.Definition` | TestDefinition |
| `tools.Deliver` | TestDeliver |
| `tools.Documentation` | TestToolsDocumentation |
| `tools.Documentation.UserAgent` | TestToolsDocumentation |
| `tools.DocumentationTool` | TestToolNames, TestToolsDocumentation |
| `tools.InputBudgetBytes` | TestInputBudgetBytes, TestMeasure |
| `tools.ListFilesTool` | TestToolNames, TestDefinition, TestToolsRepo |
| `tools.Measure` | TestMeasure |
| `tools.Measurement.Fits` | TestMeasure |
| `tools.Measurement.RenderedBytes` | TestMeasure |
| `tools.Output` | TestOutputCut, TestViewImageAndReadEvent |
| `tools.Output.Cut` | TestOutputCut |
| `tools.Output.Text` | TestOutputCut, TestToolsRun |
| `tools.OutputReserveTokens` | TestOutputReserveTokens |
| `tools.ReadEvent` | TestViewImageAndReadEvent |
| `tools.ReadFileTool` | TestToolNames, TestDefinition, TestToolsRun |
| `tools.Replay` | TestReplay |
| `tools.SearchTextTool` | TestToolNames, TestDefinition, TestToolsRepo, TestToolsSession |
| `tools.Tools` | TestToolsRun, TestToolsRepo |
| `tools.Tools.Documentation` | TestToolsDocumentation |
| `tools.Tools.Enabled` | TestToolsRun |
| `tools.Tools.Probe` | TestToolsProbe |
| `tools.Tools.Repo` | TestToolsRepo |
| `tools.Tools.Root` | TestToolsRun |
| `tools.Tools.Run` | TestToolsRun, TestToolsRepo, TestToolsProbe |
| `tools.Tools.Session` | TestToolsSession |
| `tools.ViewImageTool` | TestToolNames, TestDefinition, TestViewImageAndReadEvent |
