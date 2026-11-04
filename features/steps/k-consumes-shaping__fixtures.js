'use strict';
// Known-good fixtures of the k-consumes-shaping observer controls (k-consumes-shaping__control.js):
// for each consumes suite, a test file that only references every use of kogen-shaping's code at
// HEAD (no test), and the README inventory of the uses it references, one line each. Data only:
// no step definition.
const refs = {
  'kogen-coordinator': String.raw`package consumes

import (
	"github.com/KogenAI/kogen-coordinator/durable"
	"github.com/KogenAI/kogen-coordinator/git"
	"github.com/KogenAI/kogen-coordinator/project"
)

var _ = durable.Sync
var _ = git.AcquireBuild
var _ = git.BuildLockPath
var _ = (*git.Lock).Held
var _ = (*git.Lock).Release
var _ = git.New
var _ = git.NewLock
var _ = git.Open
var _ *git.Pin
var _ = (*git.Pin).Changes
var _ *git.Repo
var _ = (*git.Repo).At
var _ = (*git.Repo).Changed
var _ = (*git.Repo).ChangedPaths
var _ = func(v *git.Repo) any { return v.Dir }
var _ = (*git.Repo).Head
var _ = (*git.Repo).Identity
var _ = (*git.Repo).IndexPath
var _ = (*git.Repo).Materialize
var _ = (*git.Repo).OK
var _ = (*git.Repo).OKEnv
var _ = (*git.Repo).Observing
var _ = (*git.Repo).OperationInProgress
var _ = (*git.Repo).Run
var _ = (*git.Repo).Show
var _ = (*git.Repo).TakePin
var _ = (*git.Repo).Tree
var _ = func(v *git.Result) any { return v.Status }
var _ = func(v *git.Result) any { return v.Stderr }
var _ = func(v *git.Result) any { return v.Stdout }
var _ = func(v *project.Acceptance) any { return v.Run }
var _ = func(v *project.Acceptance) any { return v.StepsDir }
var _ *project.Approval
var _ = func(v *project.Approval) any { return v.ApprovedAt }
var _ = func(v *project.Approval) any { return v.Approver }
var _ = func(v *project.Approval) any { return v.Base }
var _ = func(v *project.Approval) any { return v.SHA256 }
var _ = func(v *project.Approval) any { return v.Slug }
var _ = func(v *project.Approval) any { return v.SourceProvenance }
var _ = project.ApprovalPath
var _ = project.Baseline
var _ = project.BindSource
var _ = project.CaptureOrigin
var _ = project.CaptureSource
var _ = func(v *project.Check) any { return v.Name }
var _ = func(v *project.Check) any { return v.Run }
var _ = project.CheckIntent
var _ = project.Checkout
var _ *project.Config
var _ = func(v *project.Config) any { return v.Acceptance }
var _ = (*project.Config).AcceptanceConfigured
var _ = func(v *project.Config) any { return v.Checks }
var _ = func(v *project.Config) any { return v.Fix }
var _ = func(v *project.Config) any { return v.Models }
var _ = func(v *project.Config) any { return v.Protect }
var _ = func(v *project.Config) any { return v.Setup }
var _ = project.ConfigPath
var _ = project.EncodeApproval
var _ = project.EncodeJSON
var _ = project.FeaturePath
var _ = project.Hash
var _ *project.Intent
var _ = func(v *project.Intent) any { return v.Acceptance }
var _ = func(v *project.Intent) any { return v.Decisions }
var _ = func(v *project.Intent) any { return v.Feasibility }
var _ = func(v *project.Intent) any { return v.Model }
var _ = func(v *project.Intent) any { return v.ModelLine }
var _ = func(v *project.Intent) any { return v.Root }
var _ = func(v *project.Intent) any { return v.Slug }
var _ = func(v *project.Intent) any { return v.SlugLine }
var _ = (*project.Intent).StepFiles
var _ = func(v *project.Intent) any { return v.Tests }
var _ = func(v *project.Intent) any { return v.Text }
var _ = func(v *project.Intent) any { return v.Title }
var _ = project.IntentDir
var _ = project.IntentExists
var _ = project.IntentMDPath
var _ = project.LoadConfig
var _ = project.LoadIntent
var _ = project.ModelPath
var _ *project.ModelSpec
var _ = func(v *project.ModelSpec) any { return v.Invariants }
var _ = func(v *project.ModelSpec) any { return v.None }
var _ = func(v *project.ModelSpec) any { return v.Path }
var _ = func(v *project.ModelSpec) any { return v.Samples }
var _ = func(v *project.ModelSpec) any { return v.Seed }
var _ = func(v *project.ModelSpec) any { return v.Steps }
var _ = project.NewSourceRequest
var _ = func(v *project.Outline) any { return v.Name }
var _ = project.ParseGherkin
var _ = project.ParseStages
var _ = project.Protected
var _ = project.ReadApproval
var _ = project.ReadIntent
var _ = project.RefineSource
var _ = project.RefreshSource
var _ = project.RequireIntentDir
var _ = project.ResolveApproval
var _ = project.RestoreSource
var _ *project.Role
var _ = func(v *project.Role) any { return v.BaseURL }
var _ = func(v *project.Role) any { return v.Effort }
var _ = func(v *project.Role) any { return v.Executor }
var _ = (*project.Role).External
var _ = func(v *project.Role) any { return v.Model }
var _ = func(v *project.Role) any { return v.Provider }
var _ *project.Scenario
var _ = func(v *project.Scenario) any { return v.Line }
var _ = func(v *project.Scenario) any { return v.Name }
var _ = func(v *project.Scenario) any { return v.Outline }
var _ = func(v *project.Scenario) any { return v.Steps }
var _ *project.ShapeSource
var _ = func(v *project.ShapeSource) any { return v.Origin }
var _ = func(v *project.ShapeSource) any { return v.Request }
var _ = func(v *project.ShapeSource) any { return v.Rounds }
var _ = func(v *project.ShapeSource) any { return v.SourcePath }
var _ = project.SourceCard
var _ = project.SourceComparison
var _ = project.SourceMessage
var _ *project.SourceOrigin
var _ = func(v *project.SourceOrigin) any { return v.Configuration }
var _ = func(v *project.SourceOrigin) any { return v.Head }
var _ = func(v *project.SourceOrigin) any { return v.Tree }
var _ = project.SourcePath
var _ *project.SourceRef
var _ = (*project.SourceRef).Archived
var _ = func(v *project.SourceRef) any { return v.ContractSHA256 }
var _ = (*project.SourceRef).Equal
var _ = (*project.SourceRef).Map
var _ = func(v *project.SourceRef) any { return v.Path }
var _ *project.SourceRequest
var _ *project.Stage
var _ = func(v *project.Stage) any { return v.Checks }
var _ = func(v *project.Stage) any { return v.Goal }
var _ = func(v *project.Stage) any { return v.ID }
var _ = func(v *project.Stage) any { return v.Title }
var _ *project.Step
var _ = func(v *project.Step) any { return v.Keyword }
var _ = func(v *project.Step) any { return v.Line }
var _ = func(v *project.Step) any { return v.Text }
var _ = project.ValidateSource
var _ = project.WritePackageFacts
`,
  'kogen-executor': String.raw`package consumes

import (
	"github.com/KogenAI/kogen-executor/agent"
	"github.com/KogenAI/kogen-executor/executor"
	"github.com/KogenAI/kogen-executor/thinking"
)

var _ = agent.AsAny
var _ = agent.BudgetEvent
var _ = agent.Defs
var _ = agent.Git
var _ = agent.Prompt
var _ = agent.Render
var _ = agent.TestingRules
var _ = agent.WithOriginHint
var _ *executor.Brief
var _ = func(v *executor.Brief) any { return v.Dynamic }
var _ = func(v *executor.Brief) any { return v.Stable }
var _ = executor.ClaudeCode
var _ = (*executor.Descriptor).For
var _ = func(v *executor.Descriptor) any { return v.Name }
var _ = (*executor.Descriptor).ValidateRole
var _ = executor.EndFinish
var _ *executor.Executor
var _ = executor.Executor.Describe
var _ = executor.Executor.Start
var _ = executor.Fail
var _ = func(v *executor.Handle) any { return v.Executor }
var _ = func(v *executor.Handle) any { return v.SessionID }
var _ = func(v *executor.Handle) any { return v.Version }
var _ *executor.Input
var _ = func(v *executor.Input) any { return v.Feedback }
var _ = executor.ReasonProtocol
var _ = executor.ReasonUnavailable
var _ = func(v *executor.RoleCapabilities) any { return v.MaxRounds }
var _ = executor.RoleShape
var _ = executor.RoundLimit
var _ *executor.RoundResult
var _ = func(v *executor.RoundResult) any { return v.Ending }
var _ = func(v *executor.RoundResult) any { return v.Handle }
var _ = func(v *executor.RoundResult) any { return v.Structured }
var _ = func(v *executor.RoundResult) any { return v.Usage }
var _ *executor.Session
var _ = executor.Session.Close
var _ = executor.Session.Handle
var _ = executor.Session.Round
var _ *executor.Task
var _ = func(v *executor.Task) any { return v.Brief }
var _ = func(v *executor.Task) any { return v.Dir }
var _ = func(v *executor.Task) any { return v.Effort }
var _ = func(v *executor.Task) any { return v.Home }
var _ = func(v *executor.Task) any { return v.MaxRounds }
var _ = func(v *executor.Task) any { return v.Model }
var _ = func(v *executor.Task) any { return v.Role }
var _ = func(v *executor.Task) any { return v.Root }
var _ = func(v *executor.Task) any { return v.RunID }
var _ = func(v *executor.Task) any { return v.Schema }
var _ *executor.Usage
var _ = func(v *executor.Usage) any { return v.Cached }
var _ = func(v *executor.Usage) any { return v.CostUSD }
var _ = func(v *executor.Usage) any { return v.Input }
var _ = func(v *executor.Usage) any { return v.Model }
var _ = func(v *executor.Usage) any { return v.Output }
var _ = func(v *executor.Usage) any { return v.Reasoning }
var _ = func(v *executor.Usage) any { return v.Source }
var _ = func(v *executor.Usage) any { return v.State }
var _ = executor.UsagePartial
var _ = thinking.Strip
`,
  'kogen-loop': String.raw`package consumes

import (
	"kogen-loop/provider"
	"kogen-loop/sys"
	"kogen-loop/tools"
)

var _ *provider.Client
var _ = (*provider.Client).Body
var _ = (*provider.Client).Respond
var _ = func(v *provider.Client) any { return v.Version }
var _ = provider.ContractBlocks
var _ = provider.InputBudgetBytes
var _ = (*provider.KnownUsage).Map
var _ = provider.NewCacheID
var _ = provider.NewClient
var _ *provider.Observer
var _ *provider.Request
var _ = func(v *provider.Request) any { return v.CacheID }
var _ = func(v *provider.Request) any { return v.Context }
var _ = func(v *provider.Request) any { return v.Input }
var _ = func(v *provider.Request) any { return v.Instructions }
var _ = func(v *provider.Request) any { return v.Role }
var _ = func(v *provider.Request) any { return v.Scope }
var _ = func(v *provider.Request) any { return v.ToolChoice }
var _ = func(v *provider.Request) any { return v.Tools }
var _ = provider.RequireLogin
var _ *provider.Response
var _ = func(v *provider.Response) any { return v.Items }
var _ = func(v *provider.Response) any { return v.Known }
var _ = func(v *provider.Response) any { return v.Usage }
var _ *provider.Role
var _ = func(v *provider.Role) any { return v.BaseURL }
var _ = func(v *provider.Role) any { return v.Effort }
var _ = func(v *provider.Role) any { return v.Model }
var _ = func(v *provider.Role) any { return v.Provider }
var _ = provider.SupportsToolChoice
var _ *provider.Usage
var _ = (*provider.Usage).Add
var _ = func(v *provider.Usage) any { return v.Cached }
var _ = func(v *provider.Usage) any { return v.Input }
var _ = (*provider.Usage).Line
var _ = func(v *provider.Usage) any { return v.Output }
var _ = func(v *provider.Usage) any { return v.Reasoning }
var _ = sys.AsError
var _ *sys.Error
var _ = func(v *sys.Error) any { return v.Class }
var _ = func(v *sys.Error) any { return v.Detail }
var _ = (*sys.Error).Error
var _ = func(v *sys.Error) any { return v.Exit }
var _ = func(v *sys.Error) any { return v.Lines }
var _ = func(v *sys.Error) any { return v.ProviderFailure }
var _ = func(v *sys.Error) any { return v.Reason }
var _ = (*sys.Error).Render
var _ = sys.Fail
var _ *sys.Interrupted
var _ = (*sys.Interrupted).ExitCode
var _ = func(v *sys.Interrupted) any { return v.Signal }
var _ = sys.New
var _ *sys.Options
var _ = func(v *sys.Options) any { return v.Context }
var _ = func(v *sys.Options) any { return v.Env }
var _ = func(v *sys.Options) any { return v.KillGroupOnExit }
var _ = func(v *sys.Options) any { return v.Timeout }
var _ = func(v *sys.Result) any { return v.Output }
var _ = func(v *sys.Result) any { return v.Status }
var _ = func(v *sys.Result) any { return v.TimedOut }
var _ *sys.Session
var _ = func(v *sys.Session) any { return v.Cwd }
var _ = func(v *sys.Session) any { return v.Env }
var _ = (*sys.Session).Err
var _ = (*sys.Session).Getenv
var _ = (*sys.Session).Out
var _ = (*sys.Session).Progress
var _ = (*sys.Session).Puts
var _ = (*sys.Session).Run
var _ = (*sys.Session).RunContext
var _ = func(v *sys.Session) any { return v.Stdout }
var _ = (*sys.Session).WatchSignals
var _ = sys.TmpPath
var _ *tools.Call
var _ = (*tools.Call).Args
var _ = func(v *tools.Call) any { return v.Name }
var _ = (*tools.Call).Output
var _ = tools.Calls
var _ = tools.Cut
var _ = tools.DecodeArguments
var _ = tools.Definition
var _ = tools.Deliver
var _ *tools.Documentation
var _ = func(v *tools.Documentation) any { return v.UserAgent }
var _ = tools.DocumentationTool
var _ = tools.InputBudgetBytes
var _ = tools.ListFilesTool
var _ = tools.Measure
var _ = func(v *tools.Measurement) any { return v.Fits }
var _ = func(v *tools.Measurement) any { return v.RenderedBytes }
var _ *tools.Output
var _ = (*tools.Output).Cut
var _ = func(v *tools.Output) any { return v.Text }
var _ = tools.OutputReserveTokens
var _ = tools.ReadEvent
var _ = tools.ReadFileTool
var _ = tools.Replay
var _ = tools.SearchTextTool
var _ *tools.Tools
var _ = func(v *tools.Tools) any { return v.Documentation }
var _ = func(v *tools.Tools) any { return v.Enabled }
var _ = func(v *tools.Tools) any { return v.Probe }
var _ = func(v *tools.Tools) any { return v.Repo }
var _ = func(v *tools.Tools) any { return v.Root }
var _ = (*tools.Tools).Run
var _ = func(v *tools.Tools) any { return v.Session }
var _ = tools.ViewImageTool
`,
  'kogen-verification': String.raw`package consumes

import "github.com/KogenAI/kogen-verification/verify"

var _ = verify.Acknowledgment
var _ = verify.CopyTree
var _ = verify.Cut
var _ = func(v *verify.Declaration) any { return v.Checks }
var _ = func(v *verify.Declaration) any { return v.ID }
var _ = func(v *verify.Declaration) any { return v.Kind }
var _ = func(v *verify.Declaration) any { return v.Observers }
var _ = verify.Declarations
var _ = func(v *verify.DeclaredCheck) any { return v.Command }
var _ = func(v *verify.DeclaredCheck) any { return v.Exit }
var _ = func(v *verify.DeclaredCheck) any { return v.Name }
var _ = verify.Diagnostic
var _ *verify.Evidence
var _ *verify.Logger
var _ = verify.NewAttempt
var _ = verify.NewRedactor
var _ *verify.Obligation
var _ = verify.Observations
var _ *verify.Probe
var _ = func(v *verify.Probe) any { return v.Command }
var _ = func(v *verify.Probe) any { return v.Files }
var _ = func(v *verify.Probe) any { return v.Root }
var _ = (*verify.Probe).Run
var _ = func(v *verify.Probe) any { return v.Tools }
var _ *verify.ProbeFile
var _ = func(v *verify.ProbeFile) any { return v.Content }
var _ = func(v *verify.ProbeFile) any { return v.Path }
var _ *verify.ProbeFileError
var _ = (*verify.ProbeFileError).Error
var _ = func(v *verify.ProbeFileError) any { return v.Problems }
var _ *verify.ProbeResult
var _ = func(v *verify.ProbeResult) any { return v.Output }
var _ = func(v *verify.ProbeResult) any { return v.Status }
var _ = func(v *verify.ProbeResult) any { return v.TimedOut }
var _ = func(v *verify.ProbeResult) any { return v.Wall }
var _ = verify.ProbeTimeout
var _ *verify.Readiness
var _ = func(v *verify.Readiness) any { return v.Attempt }
var _ = func(v *verify.Readiness) any { return v.BeforeRecord }
var _ = func(v *verify.Readiness) any { return v.BoundShapeAcceptance }
var _ = (*verify.Readiness).Cached
var _ = (*verify.Readiness).Check
var _ = func(v *verify.Readiness) any { return v.Config }
var _ = (*verify.Readiness).Current
var _ = func(v *verify.Readiness) any { return v.Evidence }
var _ = func(v *verify.Readiness) any { return v.Gate }
var _ = func(v *verify.Readiness) any { return v.Log }
var _ = (*verify.Readiness).RecordPass
var _ = func(v *verify.Readiness) any { return v.Repo }
var _ = func(v *verify.Readiness) any { return v.Root }
var _ = (*verify.Readiness).Setup
var _ = func(v *verify.Readiness) any { return v.Tools }
var _ *verify.Ready
var _ = func(v *verify.Ready) any { return v.AcceptanceExit }
var _ = func(v *verify.Ready) any { return v.Baseline }
var _ = func(v *verify.Ready) any { return v.CandidateTree }
var _ = (*verify.Ready).Failures
var _ = func(v *verify.Ready) any { return v.Feasibility }
var _ = func(v *verify.Ready) any { return v.Head }
var _ = func(v *verify.Ready) any { return v.Intent }
var _ = func(v *verify.Ready) any { return v.Model }
var _ = func(v *verify.Ready) any { return v.SHA256 }
var _ = func(v *verify.Ready) any { return v.Scenarios }
var _ = func(v *verify.Ready) any { return v.StepWarnings }
var _ *verify.Redactor
var _ = (*verify.Redactor).Redact
var _ = verify.RequireRunner
var _ = verify.RequireTools
var _ = verify.RunStepScript
var _ *verify.ScenarioResult
var _ = func(v *verify.ScenarioResult) any { return v.Detail }
var _ = func(v *verify.ScenarioResult) any { return v.Line }
var _ = func(v *verify.ScenarioResult) any { return v.Name }
var _ = func(v *verify.ScenarioResult) any { return v.OutlineExamples }
var _ = func(v *verify.ScenarioResult) any { return v.OutlineLine }
var _ = func(v *verify.ScenarioResult) any { return v.OutlineName }
var _ = func(v *verify.ScenarioResult) any { return v.RunnerObserver }
var _ = func(v *verify.ScenarioResult) any { return v.Status }
var _ = func(v *verify.ScenarioResult) any { return v.Steps }
var _ = func(v *verify.ScenarioResult) any { return v.UnexercisedSteps }
var _ = func(v *verify.ScenarioResult) any { return v.Unwritten }
var _ = verify.StepAnalysis
var _ *verify.StepDef
var _ = func(v *verify.StepDef) any { return v.File }
var _ = func(v *verify.StepDef) any { return v.Pattern }
var _ = func(v *verify.StepDef) any { return v.Used }
var _ *verify.StepItem
var _ = func(v *verify.StepItem) any { return v.Check }
var _ = func(v *verify.StepItem) any { return v.Detail }
var _ = func(v *verify.StepItem) any { return v.File }
var _ = func(v *verify.StepItem) any { return v.OK }
var _ = func(v *verify.StepItem) any { return v.Target }
var _ = func(v *verify.StepItem) any { return v.Warning }
var _ = verify.StepItems
var _ = func(v *verify.StepRaw) any { return v.Defs }
var _ *verify.StepRecord
var _ = func(v *verify.StepRecord) any { return v.Keyword }
var _ = func(v *verify.StepRecord) any { return v.Line }
var _ = func(v *verify.StepRecord) any { return v.Text }
var _ = verify.StepTimeout
var _ = func(v *verify.StepWarning) any { return v.Detail }
var _ = func(v *verify.StepWarning) any { return v.File }
var _ = (*verify.StepWarning).Line
var _ = func(v *verify.StepWarning) any { return v.Pattern }
var _ = verify.StepWarnings
var _ *verify.Tools
var _ = func(v *verify.Tools) any { return v.Cucumber }
var _ = func(v *verify.Tools) any { return v.Node }
var _ = func(v *verify.Tools) any { return v.Quint }
var _ = verify.ToolsFromInstall
`,
};

// uses is the use each reference line of suite s names, in order.
function uses(s) {
  return refs[s].split('\n').filter((l) => l.startsWith('var _ ')).map((l) => {
    let m = /^var _ = func\(v \*(\w+\.\w+)\) any \{ return v\.(\w+) \}$/.exec(l);
    if (m) return `${m[1]}.${m[2]}`;
    m = /^var _ = \(\*(\w+\.\w+)\)\.(\w+)$/.exec(l);
    if (m) return `${m[1]}.${m[2]}`;
    m = /^var _ (?:\*|= )([\w.]+)$/.exec(l);
    if (m) return m[1];
    throw new Error(`unreadable reference line of ${s}: ${l}`);
  });
}

// readme is the control README of suite s: an Inventory table with one row per use.
function readme(s) {
  return `# consumes/${s} (control)\n\n## Inventory\n\n| Use | Tests |\n| --- | --- |\n` + uses(s).map((u) => `| \`${u}\` | refs_test.go |\n`).join('');
}

// refsTest is the control test file of suite s.
function refsTest(s) {
  return refs[s];
}

module.exports = { refsTest, readme };
