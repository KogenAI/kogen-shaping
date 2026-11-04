# consumes/kogen-executor

What kogen-shaping relies on from kogen-executor (module `github.com/KogenAI/kogen-executor`, at the
version `go.mod` pins). Each test states, against the producer's exported API only, a use that
kogen-shaping's own code (`shape/*.go`, `cmd/kogen-shape/main.go`) makes of the packages `agent`,
`executor` and `thinking`, so that a producer change kogen-shaping would feel fails here.

## What it pins

- `agent`: the testing-rules prompt and `Render` that build the model-facing text, `Defs` and
  `AsAny` that build the Tools list of a provider request, `BudgetEvent`'s run-log record,
  `WithOriginHint`'s extra line, and `Git`'s adapter of a `git.Repo` to `tools.Git` (nil stays nil;
  `Run`, `Head` and `Tree` agree with the repository).
- `executor`: the role, executor, ending, reason and usage-state words; the stops made by `Fail`
  and `RoundLimit` (class, reason, detail, exit); a descriptor's `Name`, `For` and `ValidateRole`;
  and the `Executor` and `Session` methods with the `Task`, `Input`, `Brief`, `RoundResult`,
  `Handle` and `Usage` values passed through them, each called through an interface value or a
  method expression. The compile-time pins write the types out (for example
  `var _ func(int) *sys.Error = executor.RoundLimit`).
- `thinking.Strip`: which blocks leave a kept message, and that its input is not mutated.

Nothing here starts a real executor, the `claude` CLI or a network call: `executor.ClaudeCode` is
the executor name constant and `Executor.Start` is called on a local fake, so the suite pins its
signature and the values that cross it. A fake implementing an interface is never what covers an
interface method: the tests name `Describe`, `Start`, `Round`, `Handle` and `Close` through
`executor.Executor` and `executor.Session` values.

## Running

```sh
go test -race -count=1 ./consumes/kogen-executor/
```

## Isolation

`TestMain` unsets every inherited `GIT_*` and `KOGEN_*` variable and `EMAIL`, sets a `HOME` of its
own (`os.MkdirTemp`, removed afterwards), `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1` and
`GIT_TERMINAL_PROMPT=0`. The suite imports no `net` package and starts no listener; every file it
writes is under `t.TempDir()`; tests share no state and pass under `-race -count=1 -shuffle=on` and
`-trimpath`.

## Coverage

The uses were listed with go/types: every identifier in kogen-shaping's non-test code (the packages
outside `contract/`, `consumes/` and `features/`, today `shape` and `cmd/kogen-shape`) that resolves
to an exported package-level object of a package of this module (`pkg.Name`), or to a method or
field of one of its named types (`pkg.Type.Member`). The same analysis over this directory's test
files must name each of them, and the inventory below must equal that list: 61 uses, each row naming
the tests that assert it. Other exported identifiers of the producer, or of the other Kogen
dependencies, appear in the tests only to arrange or observe a use.

## Inventory

| Use | Tests |
| --- | --- |
| `agent.AsAny` | TestAgentAsAny |
| `agent.BudgetEvent` | TestAgentBudgetEvent |
| `agent.Defs` | TestAgentDefs |
| `agent.Git` | TestAgentGit |
| `agent.Prompt` | TestAgentPromptTestingRules |
| `agent.Render` | TestAgentRender |
| `agent.TestingRules` | TestAgentPromptTestingRules |
| `agent.WithOriginHint` | TestAgentWithOriginHint |
| `executor.Brief` | TestExecutorInterfaces |
| `executor.Brief.Dynamic` | TestExecutorInterfaces |
| `executor.Brief.Stable` | TestExecutorInterfaces |
| `executor.ClaudeCode` | TestExecutorConstants |
| `executor.Descriptor.For` | TestExecutorDescriptor |
| `executor.Descriptor.Name` | TestExecutorDescriptor |
| `executor.Descriptor.ValidateRole` | TestExecutorDescriptor |
| `executor.EndFinish` | TestExecutorConstants, TestExecutorInterfaces |
| `executor.Executor` | TestExecutorInterfaces |
| `executor.Executor.Describe` | TestExecutorInterfaces |
| `executor.Executor.Start` | TestExecutorInterfaces |
| `executor.Fail` | TestExecutorFail |
| `executor.Handle.Executor` | TestExecutorInterfaces |
| `executor.Handle.SessionID` | TestExecutorInterfaces |
| `executor.Handle.Version` | TestExecutorInterfaces |
| `executor.Input` | TestExecutorInterfaces |
| `executor.Input.Feedback` | TestExecutorInterfaces |
| `executor.ReasonProtocol` | TestExecutorConstants, TestExecutorFail |
| `executor.ReasonUnavailable` | TestExecutorConstants, TestExecutorFail |
| `executor.RoleCapabilities.MaxRounds` | TestExecutorDescriptor |
| `executor.RoleShape` | TestExecutorConstants, TestExecutorDescriptor |
| `executor.RoundLimit` | TestExecutorRoundLimit |
| `executor.RoundResult` | TestExecutorInterfaces |
| `executor.RoundResult.Ending` | TestExecutorInterfaces |
| `executor.RoundResult.Handle` | TestExecutorInterfaces |
| `executor.RoundResult.Structured` | TestExecutorInterfaces |
| `executor.RoundResult.Usage` | TestExecutorInterfaces |
| `executor.Session` | TestExecutorInterfaces |
| `executor.Session.Close` | TestExecutorInterfaces |
| `executor.Session.Handle` | TestExecutorInterfaces |
| `executor.Session.Round` | TestExecutorInterfaces |
| `executor.Task` | TestExecutorInterfaces |
| `executor.Task.Brief` | TestExecutorInterfaces |
| `executor.Task.Dir` | TestExecutorInterfaces |
| `executor.Task.Effort` | TestExecutorInterfaces |
| `executor.Task.Home` | TestExecutorInterfaces |
| `executor.Task.MaxRounds` | TestExecutorInterfaces |
| `executor.Task.Model` | TestExecutorInterfaces |
| `executor.Task.Role` | TestExecutorInterfaces |
| `executor.Task.Root` | TestExecutorInterfaces |
| `executor.Task.RunID` | TestExecutorInterfaces |
| `executor.Task.Schema` | TestExecutorInterfaces |
| `executor.Usage` | TestExecutorUsage |
| `executor.Usage.Cached` | TestExecutorUsage |
| `executor.Usage.CostUSD` | TestExecutorUsage |
| `executor.Usage.Input` | TestExecutorUsage |
| `executor.Usage.Model` | TestExecutorUsage |
| `executor.Usage.Output` | TestExecutorUsage |
| `executor.Usage.Reasoning` | TestExecutorUsage |
| `executor.Usage.Source` | TestExecutorUsage |
| `executor.Usage.State` | TestExecutorUsage |
| `executor.UsagePartial` | TestExecutorConstants, TestExecutorUsage |
| `thinking.Strip` | TestThinkingStrip |
