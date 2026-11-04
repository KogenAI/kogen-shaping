# consumes/kogen-coordinator

This suite states, as tests, exactly what kogen-shaping relies on from kogen-coordinator (module `github.com/KogenAI/kogen-coordinator`, packages `durable`, `git` and `project`) at the version `go.mod` pins:
`v0.0.0-20261102193107-c4a49a4fe84f`. It is one Go package, `consumes`, made only of `_test.go` files, and it is part of `go test ./...`.
If kogen-coordinator changes something listed in the inventory below, a test here fails, so a producer's landing can run this suite instead of building and testing kogen-shaping in full.

## What it pins

Every use of the producer in kogen-shaping's own code (the non-test Go files of `shape` and `cmd/kogen-shape`) is pinned twice:

- At compile time: `TestGitSignatures` and `TestProjectSignatures` assign each function, method (through a method expression), type, field and constant to a variable whose type is written out, for example `var _ func(*os.File) error = durable.Sync`. A changed signature, field type or removed identifier no longer compiles.
- By behaviour, through real calls on what kogen-shaping's call sites depend on:
  - `git`: a repository made in `t.TempDir()` with the `git` binary: `Open` (the root, and the `environment/not_a_git_repo` stop, exit 3), `Head`, `Identity` (never guessed), `ChangedPaths` (staged, unstaged, untracked file by file, ignored left out, a rename as new path then old, pathspecs), `Run` and `OK`/`OKEnv` (stdout, stderr and status kept apart; private index through `GIT_INDEX_FILE`; refused commands), `Show`, `Tree` (the candidate tree in a private index: untracked files in, ignored files out, modes kept, the user's index untouched), `Changed` (no rename detection), `IndexPath`, `Observing` (an observation leaves the index as it is), `Materialize` (a clean cold copy that is its own repository), `OperationInProgress` (merge, cherry-pick, rebase and revert stopped on real conflicts), `TakePin` and `Pin.Changes` (hooks and program-selecting configuration keys named, never values), and the Build lock (`BuildLockPath`, `NewLock`, `AcquireBuild`, `Lock.Held`, `Lock.Release`, the `environment/build_running` stop and the notice for a stale lock).
  - `project`: a `.kogen/project.yaml` and an Intent package written in `t.TempDir()`: the paths (`ConfigPath`, `IntentDir`, `IntentMDPath`, `FeaturePath`, `ModelPath`, `ApprovalPath`, `SourcePath`), `LoadConfig` (every field, role defaults, the three stops with their lines), `LoadIntent`, `ReadIntent` and `CheckIntent` (fields, overlay, `intent/parse` and `intent/lint` lines), `ParseGherkin`, `ParseStages`, `Baseline`, `Hash` (changes when a baselined file changes), `Protected`, `EncodeApproval` and `ReadApproval`, `EncodeJSON` (the exact bytes of the records shape writes), the source receipt (`CaptureOrigin`, `CaptureSource`, `BindSource`, `ValidateSource`, `SourceMessage`, `SourceCard`, `SourceComparison`, `RefreshSource`, `RefineSource`, `ResolveApproval`, `RestoreSource` and the `SourceRef` methods), `Checkout` and `WritePackageFacts`.
  - `durable.Sync` flushes an open file.

The suite asserts nothing kogen-shaping does not use. Other exported identifiers of the producer (for example `git.Repo.Commit` or `project.Template`) do not appear, except `git.Materialized`, `git.Result` and `project.CheckoutFacts`, which appear only to arrange or observe a use.

## How to run it

```sh
go test -race -count=1 ./consumes/kogen-coordinator/
```

It also passes with `-shuffle=on` and `-trimpath`, and under `GOPROXY=off`, `GOFLAGS=-mod=readonly -buildvcs=false -trimpath`.
It needs the `git` binary (2.38 or later, for `ls-files --format`).

## Isolation

`TestMain` unsets every inherited `GIT_*` and `KOGEN_*` variable and `EMAIL`, sets a `HOME` of its own (`os.MkdirTemp`, removed afterwards), and sets `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1` and `GIT_TERMINAL_PROMPT=0`.
Every file lives under `t.TempDir()`; no test reads kogen-shaping's sources, the module cache or another repository, uses the network (no listener, no HTTP server, no port, no `net` package), sleeps, skips or shares state with another test.
Repositories are made with a fixed author and with signing turned off on the command line, so a signing configuration of the caller cannot reach them.
The suite imports only the standard library, the packages of kogen-coordinator and `kogen-loop/sys` (the `*sys.Session` that `git.AcquireBuild` takes and the `*sys.Error` that stops are made of). It imports no package of kogen-shaping.

## How coverage was checked

The uses were listed with go/types: every identifier in kogen-shaping's own non-test code (the packages of module `github.com/KogenAI/kogen-shaping` outside `contract/`, `consumes/` and `features/`) that resolves (`Uses` or `Selections`) to an exported package-level object of a package of `github.com/KogenAI/kogen-coordinator`, named `pkg.Name`, or to a method or field of one of its named types, named `pkg.Type.Member` by the type that declares it, with `pkg` the last element of the import path.
That list (142 uses) was compared with the identifiers named in this suite's test files, listed the same way, and with the table below: each use is named by a test, and the table lists each use once and nothing else.
The listing is the `-list` mode of the acceptance observer, `features/steps/k-consumes-shaping__observer.js`:

```sh
d=$(mktemp -d) && node -e "process.stdout.write(require('./features/steps/k-consumes-shaping__observer.js').source)" > "$d/main.go" && printf 'module observer\n\ngo 1.27\n' > "$d/go.mod" && (cd "$d" && GOFLAGS= GOWORK=off go build -o observer .) && "$d/observer" -list . kogen-coordinator=github.com/KogenAI/kogen-coordinator kogen-executor=github.com/KogenAI/kogen-executor kogen-loop=kogen-loop kogen-verification=github.com/KogenAI/kogen-verification
```

Without `-list` the same program prints `ok` when the suite names every use and this inventory equals the list.

## Inventory

| Use | Tests |
| --- | --- |
| `durable.Sync` | TestGitSignatures, TestDurableSync |
| `git.AcquireBuild` | TestGitSignatures, TestBuildLock, TestBuildLockStale |
| `git.BuildLockPath` | TestGitSignatures, TestBuildLock |
| `git.Lock.Held` | TestGitSignatures, TestBuildLock, TestBuildLockStale |
| `git.Lock.Release` | TestGitSignatures, TestBuildLock, TestBuildLockStale |
| `git.New` | TestGitSignatures, TestRepoTakePin, TestNew, TestRepoChangedPaths, TestRepoIndexPath, TestRepoMaterialize, TestRepoOperationInProgress, TestRepoUnmergedListing, TestCaptureSource |
| `git.NewLock` | TestGitSignatures, TestBuildLock |
| `git.Open` | TestGitSignatures, TestOpen, TestRepoIdentity |
| `git.Pin` | TestGitSignatures, TestRepoTakePin |
| `git.Pin.Changes` | TestGitSignatures, TestRepoTakePin |
| `git.Repo` | TestGitSignatures, TestRepoTakePin, TestOpen, TestNew, TestRepoAt, TestRepoHead, TestRepoIdentity, TestRepoChangedPaths, TestRepoRun, TestRepoOK, TestRepoOKEnv, TestRepoShow, TestRepoTree, TestRepoChanged, TestRepoIndexPath, TestRepoObserving, TestRepoMaterialize, TestRepoOperationInProgress, TestRepoUnmergedListing, TestScratchCheckout |
| `git.Repo.At` | TestGitSignatures, TestRepoAt, TestScratchCheckout |
| `git.Repo.Changed` | TestGitSignatures, TestRepoChanged |
| `git.Repo.ChangedPaths` | TestGitSignatures, TestRepoChangedPaths |
| `git.Repo.Dir` | TestGitSignatures, TestOpen, TestNew |
| `git.Repo.Head` | TestGitSignatures, TestRepoHead |
| `git.Repo.Identity` | TestGitSignatures, TestRepoIdentity |
| `git.Repo.IndexPath` | TestGitSignatures, TestRepoIndexPath |
| `git.Repo.Materialize` | TestGitSignatures, TestRepoMaterialize |
| `git.Repo.OK` | TestGitSignatures, TestRepoOK, TestRepoUnmergedListing |
| `git.Repo.OKEnv` | TestGitSignatures, TestRepoOKEnv |
| `git.Repo.Observing` | TestGitSignatures, TestRepoObserving |
| `git.Repo.OperationInProgress` | TestGitSignatures, TestRepoOperationInProgress |
| `git.Repo.Run` | TestGitSignatures, TestRepoRun |
| `git.Repo.Show` | TestGitSignatures, TestRepoShow |
| `git.Repo.TakePin` | TestGitSignatures, TestRepoTakePin |
| `git.Repo.Tree` | TestGitSignatures, TestRepoTree |
| `git.Result.Status` | TestGitSignatures, TestRepoRun |
| `git.Result.Stderr` | TestGitSignatures, TestRepoRun |
| `git.Result.Stdout` | TestGitSignatures, TestRepoRun |
| `project.Acceptance.Run` | TestProjectSignatures, TestLoadConfig |
| `project.Acceptance.StepsDir` | TestProjectSignatures, TestLoadConfig |
| `project.Approval` | TestProjectSignatures, TestApprovalBytes, TestReadApproval, TestEncodeJSON, TestResolveApproval |
| `project.Approval.ApprovedAt` | TestProjectSignatures, TestReadApproval |
| `project.Approval.Approver` | TestProjectSignatures, TestReadApproval |
| `project.Approval.Base` | TestProjectSignatures, TestReadApproval |
| `project.Approval.SHA256` | TestProjectSignatures, TestReadApproval |
| `project.Approval.Slug` | TestProjectSignatures, TestReadApproval |
| `project.Approval.SourceProvenance` | TestProjectSignatures, TestReadApproval |
| `project.ApprovalPath` | TestProjectSignatures, TestProjectPaths |
| `project.Baseline` | TestProjectSignatures, TestBaselineAndHash |
| `project.BindSource` | TestProjectSignatures, TestCaptureSource, TestBindSource, TestRefineSource |
| `project.CaptureOrigin` | TestProjectSignatures, TestCaptureOrigin |
| `project.CaptureSource` | TestProjectSignatures, TestCaptureSource |
| `project.Check.Name` | TestProjectSignatures, TestLoadConfig |
| `project.Check.Run` | TestProjectSignatures, TestLoadConfig |
| `project.CheckIntent` | TestProjectSignatures, TestCheckIntent, TestCheckIntentStops |
| `project.Checkout` | TestProjectSignatures, TestCheckoutAndPackageFacts |
| `project.Config` | TestProjectSignatures, TestProtected, TestLoadConfig, TestConfigModels, TestConfigAcceptanceConfigured |
| `project.Config.Acceptance` | TestProjectSignatures, TestLoadConfig |
| `project.Config.AcceptanceConfigured` | TestProjectSignatures, TestLoadConfig, TestConfigAcceptanceConfigured |
| `project.Config.Checks` | TestProjectSignatures, TestLoadConfig |
| `project.Config.Fix` | TestProjectSignatures, TestLoadConfig |
| `project.Config.Models` | TestProjectSignatures, TestConfigModels |
| `project.Config.Protect` | TestProjectSignatures, TestLoadConfig |
| `project.Config.Setup` | TestProjectSignatures, TestLoadConfig |
| `project.ConfigPath` | TestProjectSignatures, TestProjectPaths, TestConfigModels, TestLoadConfigStops |
| `project.EncodeApproval` | TestProjectSignatures, TestApprovalBytes |
| `project.EncodeJSON` | TestProjectSignatures, TestEncodeJSON, TestCheckoutAndPackageFacts |
| `project.FeaturePath` | TestProjectSignatures, TestProjectPaths, TestCheckIntent, TestCheckIntentStops, TestRefineSource, TestResolveApproval |
| `project.Hash` | TestProjectSignatures, TestBaselineAndHash, TestCaptureSource, TestCheckoutAndPackageFacts |
| `project.Intent` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.Intent.Acceptance` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.Intent.Decisions` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.Intent.Feasibility` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.Intent.Model` | TestProjectSignatures, TestReadIntent |
| `project.Intent.ModelLine` | TestProjectSignatures, TestLoadIntent |
| `project.Intent.Root` | TestProjectSignatures, TestLoadIntent |
| `project.Intent.Slug` | TestProjectSignatures, TestLoadIntent |
| `project.Intent.SlugLine` | TestProjectSignatures, TestLoadIntent |
| `project.Intent.StepFiles` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.Intent.Tests` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.Intent.Text` | TestProjectSignatures, TestLoadIntent |
| `project.Intent.Title` | TestProjectSignatures, TestLoadIntent |
| `project.IntentDir` | TestProjectSignatures, TestProjectPaths |
| `project.IntentExists` | TestProjectSignatures, TestIntentExistsAndDir |
| `project.IntentMDPath` | TestProjectSignatures, TestProjectPaths, TestReadIntent, TestCheckIntent, TestCheckIntentStops, TestSourceComparison, TestRefineSource, TestCheckoutAndPackageFacts |
| `project.LoadConfig` | TestProjectSignatures, TestLoadConfig, TestConfigAcceptanceConfigured, TestLoadConfigStops |
| `project.LoadIntent` | TestProjectSignatures, TestLoadIntent |
| `project.ModelPath` | TestProjectSignatures, TestProjectPaths |
| `project.ModelSpec` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.ModelSpec.Invariants` | TestProjectSignatures, TestReadIntent |
| `project.ModelSpec.None` | TestProjectSignatures, TestLoadIntent, TestReadIntent |
| `project.ModelSpec.Path` | TestProjectSignatures, TestReadIntent |
| `project.ModelSpec.Samples` | TestProjectSignatures, TestReadIntent |
| `project.ModelSpec.Seed` | TestProjectSignatures, TestReadIntent |
| `project.ModelSpec.Steps` | TestProjectSignatures, TestReadIntent |
| `project.NewSourceRequest` | TestProjectSignatures, TestNewSourceRequest, TestRefineSource |
| `project.Outline.Name` | TestProjectSignatures, TestParseGherkin |
| `project.ParseGherkin` | TestProjectSignatures, TestParseGherkin |
| `project.ParseStages` | TestProjectSignatures, TestParseStages |
| `project.Protected` | TestProjectSignatures, TestProtected |
| `project.ReadApproval` | TestProjectSignatures, TestReadApproval |
| `project.ReadIntent` | TestProjectSignatures, TestReadIntent |
| `project.RefineSource` | TestProjectSignatures, TestRefineSource |
| `project.RefreshSource` | TestProjectSignatures, TestRefreshSource, TestRestoreSource |
| `project.RequireIntentDir` | TestProjectSignatures, TestIntentExistsAndDir |
| `project.ResolveApproval` | TestProjectSignatures, TestResolveApproval |
| `project.RestoreSource` | TestProjectSignatures, TestRestoreSource |
| `project.Role` | TestProjectSignatures, TestEncodeJSON, TestConfigModels |
| `project.Role.BaseURL` | TestProjectSignatures, TestConfigModels |
| `project.Role.Effort` | TestProjectSignatures, TestConfigModels |
| `project.Role.Executor` | TestProjectSignatures, TestConfigModels |
| `project.Role.External` | TestProjectSignatures, TestConfigModels |
| `project.Role.Model` | TestProjectSignatures, TestConfigModels |
| `project.Role.Provider` | TestProjectSignatures, TestConfigModels |
| `project.Scenario` | TestProjectSignatures, TestParseGherkin |
| `project.Scenario.Line` | TestProjectSignatures, TestParseGherkin |
| `project.Scenario.Name` | TestProjectSignatures, TestParseGherkin |
| `project.Scenario.Outline` | TestProjectSignatures, TestParseGherkin |
| `project.Scenario.Steps` | TestProjectSignatures, TestParseGherkin |
| `project.ShapeSource` | TestProjectSignatures, TestCaptureSource |
| `project.ShapeSource.Origin` | TestProjectSignatures, TestCaptureSource |
| `project.ShapeSource.Request` | TestProjectSignatures, TestCaptureSource |
| `project.ShapeSource.Rounds` | TestProjectSignatures, TestCaptureSource |
| `project.ShapeSource.SourcePath` | TestProjectSignatures, TestCaptureSource |
| `project.SourceCard` | TestProjectSignatures, TestSourceMessageAndCard |
| `project.SourceComparison` | TestProjectSignatures, TestSourceComparison |
| `project.SourceMessage` | TestProjectSignatures, TestSourceMessageAndCard |
| `project.SourceOrigin` | TestProjectSignatures, TestCaptureOrigin, TestCaptureSource |
| `project.SourceOrigin.Configuration` | TestProjectSignatures, TestCaptureOrigin, TestCaptureSource |
| `project.SourceOrigin.Head` | TestProjectSignatures, TestCaptureOrigin, TestCaptureSource |
| `project.SourceOrigin.Tree` | TestProjectSignatures, TestCaptureOrigin, TestCaptureSource |
| `project.SourcePath` | TestProjectSignatures, TestProjectPaths |
| `project.SourceRef` | TestProjectSignatures, TestCaptureSource, TestSourceRefValues, TestRefreshSource |
| `project.SourceRef.Archived` | TestProjectSignatures, TestSourceRefValues, TestRefreshSource |
| `project.SourceRef.ContractSHA256` | TestProjectSignatures, TestCaptureSource |
| `project.SourceRef.Equal` | TestProjectSignatures, TestSourceRefValues |
| `project.SourceRef.Map` | TestProjectSignatures, TestSourceRefValues |
| `project.SourceRef.Path` | TestProjectSignatures, TestCaptureSource |
| `project.SourceRequest` | TestProjectSignatures, TestNewSourceRequest |
| `project.Stage` | TestProjectSignatures, TestParseStages |
| `project.Stage.Checks` | TestProjectSignatures, TestParseStages |
| `project.Stage.Goal` | TestProjectSignatures, TestParseStages |
| `project.Stage.ID` | TestProjectSignatures, TestParseStages |
| `project.Stage.Title` | TestProjectSignatures, TestParseStages |
| `project.Step` | TestProjectSignatures, TestParseGherkin |
| `project.Step.Keyword` | TestProjectSignatures, TestParseGherkin |
| `project.Step.Line` | TestProjectSignatures, TestParseGherkin |
| `project.Step.Text` | TestProjectSignatures, TestParseGherkin |
| `project.ValidateSource` | TestProjectSignatures, TestValidateSource |
| `project.WritePackageFacts` | TestProjectSignatures, TestCheckoutAndPackageFacts |
