# consumes/kogen-verification

What kogen-shaping relies on from kogen-verification (module `github.com/KogenAI/kogen-verification`, package `verify`), stated as tests, at the version `go.mod` pins now (`v0.0.0-20261105003813-ac4f6a9a4877`). It is the consumer's half of the two-sided contract: if kogen-verification changes anything below, this suite fails before kogen-shaping does.

## What the suite pins

Only what kogen-shaping's own code (`shape/*.go` and `cmd/kogen-shape/main.go`) uses, and each use two ways:

- **Compile time**: the type or signature of every function, method, type, field and constant is pinned with its types written out (for example `var _ func(string) string = verify.Cut`), so a changed signature, field type or constant type stops the package from compiling. `verify.Evidence` and `verify.Logger`, the interfaces kogen-shaping implements, are pinned by assigning an implementation, and the readiness fields by assigning them to typed variables.
- **Behaviour**, through real calls: the bounds of `verify.Cut` and `verify.Diagnostic`; what `verify.Redactor` masks (secret-named environment variables, saved logins under a home, labeled secrets, invalid UTF-8); the paths `verify.ToolsFromInstall` resolves under an install root; the `environment/tool_missing` and `environment/acceptance_unavailable` refusals of `verify.RequireTools` and `verify.RequireRunner`; `verify.CopyTree` and `verify.Probe.Run` on a copy of a checkout (exit status, combined output, tools as `KOGEN_*`, files laid over the copy, the `*verify.ProbeFileError` refusal); `verify.StepAnalysis`, `verify.StepItems`, `verify.StepWarnings` and the outcome of `verify.RunStepScript` for a Node that prints a result and for one that cannot run; the declarations read by `verify.Declarations` and the refusals of malformed evidence; and `verify.Readiness` end to end on a Git repository with an Intent in a `t.TempDir()`: `Check`, `Cached`, `Current`, `RecordPass`, `Setup`, `Gate`, `BeforeRecord`, `Attempt`, `BoundShapeAcceptance`, the `verify.Ready` and `verify.ScenarioResult` it returns (outline, unexercised steps, unwritten tests, runner observation), `verify.Observations` (each failed scenario carries its `Failure phase:` line) and `verify.Acknowledgment`, and the obligations of declared feasibility evidence.

Uses that need a signed-in harness, a model, the network or a host-installed Node, Cucumber or Quint are pinned by their signature and by the offline outcome kogen-shaping relies on: the acceptance command of the readiness check is a shell command that copies a prepared Cucumber Messages report, `verify.RunStepScript` runs a shell script standing for Node (and a Node that does not exist), and the tool paths of `verify.ToolsFromInstall` are only resolved. Identifiers of other Kogen dependencies appear only where a `verify` signature needs their values (`git.Repo`, `project.Config`, `project.Intent`, `*sys.Session`).

The suite asserts nothing kogen-shaping does not use; other exported identifiers of kogen-verification appear only to arrange or observe a use.

## Running it

```sh
go test -race -count=1 ./consumes/kogen-verification/
```

It is also part of `go test ./...`. It needs `git` and a POSIX `sh`, like kogen-shaping itself; no Node, Cucumber or Quint.

## Isolation

- `TestMain` unsets every inherited `GIT_*` and `KOGEN_*` variable and `EMAIL`, sets a `HOME` of its own (`os.MkdirTemp`, removed afterwards), `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1` and `GIT_TERMINAL_PROMPT=0`.
- No network: no listener, no HTTP server, no fixed port, no `net` import.
- No host paths: every file is under `t.TempDir()`, and tests that run probes or readiness checks also point `TMPDIR` at their own folder. Nothing reads kogen-shaping's sources, the module cache or another repository.
- No shared state between tests, no sleep, no skip; the suite is deterministic under `-race -count=1 -shuffle=on` and `-trimpath`.
- The suite imports only the standard library and packages of the four Kogen dependency modules, never a package of kogen-shaping, an internal package or any other module.

## How coverage was checked

A go/types listing of every identifier of the module `github.com/KogenAI/kogen-verification` that the non-test Go code of kogen-shaping (outside `contract/`, `consumes/` and `features/`: today `shape` and `cmd/kogen-shape`) uses: each identifier that resolves (`Uses` or `Selections`) to an exported package-level object of a package of the module, named `pkg.Name`, or to a method or field of one of its named types, named `pkg.Type.Member` by the declaring type. That list has 110 uses. The same analysis of this suite's test files must name every one of them, and the inventory below must list exactly them, once each. The acceptance observer of the k-consumes-shaping Intent (`features/steps/k-consumes-shaping__observer.js`) computes both and compares them; run it as that Intent's acceptance does:

```sh
d=$(mktemp -d) && node -e "process.stdout.write(require('./features/steps/k-consumes-shaping__observer.js').source)" > "$d/main.go" && printf 'module observer\n\ngo 1.27\n' > "$d/go.mod" && (cd "$d" && GOFLAGS= GOWORK=off go build -o observer .) && "$d/observer" -list . kogen-verification=github.com/KogenAI/kogen-verification
```

## Inventory

One row per use; the other cells name the tests that assert it.

| Use | Tests |
| --- | --- |
| `verify.Acknowledgment` | TestReadinessRedObservations |
| `verify.CopyTree` | TestCopyTree |
| `verify.Cut` | TestCut |
| `verify.Declaration.Checks` | TestDeclarations |
| `verify.Declaration.ID` | TestDeclarations |
| `verify.Declaration.Kind` | TestDeclarations |
| `verify.Declaration.Observers` | TestDeclarations |
| `verify.Declarations` | TestDeclarations, TestDeclarationsRefuseInvalidEvidence |
| `verify.DeclaredCheck.Command` | TestDeclarations |
| `verify.DeclaredCheck.Exit` | TestDeclarations |
| `verify.DeclaredCheck.Name` | TestDeclarations |
| `verify.Diagnostic` | TestDiagnostic |
| `verify.Evidence` | TestReadinessCheckAndKeptEvidence, TestReadinessFeasibilityObligations |
| `verify.Logger` | TestReadinessCheckAndKeptEvidence |
| `verify.NewAttempt` | TestReadinessAttemptRefusal |
| `verify.NewRedactor` | TestNewRedactorMasksEnvironmentSecrets, TestNewRedactorMasksSavedLogins, TestRedactorRedactMasksLabeledSecrets |
| `verify.Obligation` | TestReadinessFeasibilityObligations |
| `verify.Observations` | TestReadinessRedObservations |
| `verify.Probe` | TestProbeRun |
| `verify.Probe.Command` | TestProbeRun |
| `verify.Probe.Files` | TestProbeRun, TestProbeFileError |
| `verify.Probe.Root` | TestProbeRun |
| `verify.Probe.Run` | TestProbeRun, TestProbeFileError |
| `verify.Probe.Tools` | TestProbeRun |
| `verify.ProbeFile` | TestProbeRun, TestProbeFileError |
| `verify.ProbeFile.Content` | TestProbeRun, TestProbeFileError |
| `verify.ProbeFile.Path` | TestProbeRun, TestProbeFileError |
| `verify.ProbeFileError` | TestProbeFileError |
| `verify.ProbeFileError.Error` | TestProbeFileError |
| `verify.ProbeFileError.Problems` | TestProbeFileError |
| `verify.ProbeResult` | TestProbeRun |
| `verify.ProbeResult.Output` | TestProbeRun |
| `verify.ProbeResult.Status` | TestProbeRun |
| `verify.ProbeResult.TimedOut` | TestProbeRun |
| `verify.ProbeResult.Wall` | TestProbeRun |
| `verify.ProbeTimeout` | TestProbeTimeout |
| `verify.Readiness` | TestReadinessCheckAndKeptEvidence |
| `verify.Readiness.Attempt` | TestReadinessAttemptRefusal |
| `verify.Readiness.BeforeRecord` | TestReadinessGateAndBeforeRecord |
| `verify.Readiness.BoundShapeAcceptance` | TestReadinessBoundShapeAcceptance |
| `verify.Readiness.Cached` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations, TestReadinessRecordPass, TestReadinessFeasibilityObligations |
| `verify.Readiness.Check` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations, TestReadinessGateAndBeforeRecord, TestReadinessAttemptRefusal, TestReadinessFeasibilityObligations |
| `verify.Readiness.Config` | TestReadinessCheckAndKeptEvidence |
| `verify.Readiness.Current` | TestReadinessCheckAndKeptEvidence |
| `verify.Readiness.Evidence` | TestReadinessCheckAndKeptEvidence, TestReadinessFeasibilityObligations |
| `verify.Readiness.Gate` | TestReadinessGateAndBeforeRecord |
| `verify.Readiness.Log` | TestReadinessCheckAndKeptEvidence |
| `verify.Readiness.RecordPass` | TestReadinessRecordPass |
| `verify.Readiness.Repo` | TestReadinessCheckAndKeptEvidence |
| `verify.Readiness.Root` | TestReadinessCheckAndKeptEvidence |
| `verify.Readiness.Setup` | TestReadinessSetup |
| `verify.Readiness.Tools` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations |
| `verify.Ready.AcceptanceExit` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations |
| `verify.Ready.Baseline` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready.CandidateTree` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready.Failures` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations |
| `verify.Ready.Feasibility` | TestReadinessCheckAndKeptEvidence, TestReadinessFeasibilityObligations |
| `verify.Ready.Head` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready.Intent` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready.Model` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready.SHA256` | TestReadinessCheckAndKeptEvidence |
| `verify.Ready.Scenarios` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations |
| `verify.Ready.StepWarnings` | TestReadinessRecordPass |
| `verify.Redactor` | TestRedactorRedactMasksLabeledSecrets |
| `verify.Redactor.Redact` | TestRedactorRedactMasksLabeledSecrets, TestNewRedactorMasksEnvironmentSecrets, TestNewRedactorMasksSavedLogins |
| `verify.RequireRunner` | TestRequireRunner |
| `verify.RequireTools` | TestRequireTools |
| `verify.RunStepScript` | TestRunStepScript |
| `verify.ScenarioResult` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations |
| `verify.ScenarioResult.Detail` | TestReadinessRedObservations |
| `verify.ScenarioResult.Line` | TestReadinessCheckAndKeptEvidence |
| `verify.ScenarioResult.Name` | TestReadinessCheckAndKeptEvidence |
| `verify.ScenarioResult.OutlineExamples` | TestReadinessCheckAndKeptEvidence |
| `verify.ScenarioResult.OutlineLine` | TestReadinessCheckAndKeptEvidence |
| `verify.ScenarioResult.OutlineName` | TestReadinessCheckAndKeptEvidence |
| `verify.ScenarioResult.RunnerObserver` | TestReadinessRedObservations |
| `verify.ScenarioResult.Status` | TestReadinessCheckAndKeptEvidence, TestReadinessRedObservations |
| `verify.ScenarioResult.Steps` | TestReadinessCheckAndKeptEvidence |
| `verify.ScenarioResult.UnexercisedSteps` | TestReadinessRedObservations |
| `verify.ScenarioResult.Unwritten` | TestReadinessRedObservations |
| `verify.StepAnalysis` | TestStepAnalysis |
| `verify.StepDef` | TestStepItems, TestRunStepScript |
| `verify.StepDef.File` | TestStepItems, TestRunStepScript |
| `verify.StepDef.Pattern` | TestStepItems, TestRunStepScript |
| `verify.StepDef.Used` | TestStepItems, TestRunStepScript |
| `verify.StepItem` | TestStepItems |
| `verify.StepItem.Check` | TestStepItems |
| `verify.StepItem.Detail` | TestStepItems |
| `verify.StepItem.File` | TestStepItems |
| `verify.StepItem.OK` | TestStepItems |
| `verify.StepItem.Target` | TestStepItems |
| `verify.StepItem.Warning` | TestStepItems, TestStepWarnings |
| `verify.StepItems` | TestStepItems |
| `verify.StepRaw.Defs` | TestRunStepScript, TestStepItems |
| `verify.StepRecord` | TestReadinessCheckAndKeptEvidence |
| `verify.StepRecord.Keyword` | TestReadinessCheckAndKeptEvidence |
| `verify.StepRecord.Line` | TestReadinessCheckAndKeptEvidence |
| `verify.StepRecord.Text` | TestReadinessCheckAndKeptEvidence |
| `verify.StepTimeout` | TestStepTimeout |
| `verify.StepWarning.Detail` | TestStepWarnings |
| `verify.StepWarning.File` | TestStepWarnings |
| `verify.StepWarning.Line` | TestStepWarnings |
| `verify.StepWarning.Pattern` | TestStepWarnings |
| `verify.StepWarnings` | TestStepWarnings |
| `verify.Tools` | TestTools, TestToolsFromInstall |
| `verify.Tools.Cucumber` | TestTools, TestToolsFromInstall |
| `verify.Tools.Node` | TestTools, TestToolsFromInstall |
| `verify.Tools.Quint` | TestTools, TestToolsFromInstall |
| `verify.ToolsFromInstall` | TestToolsFromInstall |
