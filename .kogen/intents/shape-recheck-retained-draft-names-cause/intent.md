# Diagnose a refused check on a retained draft

Slug: shape-recheck-retained-draft-names-cause
Model: not applicable — the diagnostic is a pure function of one retained draft and its recorded runs, with no state machine of its own
Acceptance:
- .kogen/intents/shape-recheck-retained-draft-names-cause/acceptance.feature
- features/steps/shape-recheck-retained-draft-names-cause__steps.js
- features/steps/shape-recheck-retained-draft-names-cause__driver/run.js
- features/steps/shape-recheck-retained-draft-names-cause__driver/observe_test.go.txt

Package shape gains a narrow, model-free library diagnostic for a retained Shape draft: `RecheckDraftHash(root, slug string) (string, error)`, `Recheck(ctx, RecheckInput) (RecheckReport, error)`, and the types `RecheckInput`, `RecheckProperty` and `RecheckReport`. The Kogen CLI consumer is a separate later change and is out of scope.

`RecheckInput` has `Project` (the runner checkout holding the retained draft), `Pinned` (a pinned unchanged checkout), `Slug`, `DraftHash` (the package hash of the retained draft that the caller recorded; `RecheckDraftHash` computes the current one with `project.Hash` of `project.Baseline`, the hash approve prints), `Properties` (the retained contract-review properties as `RecheckProperty{Name, Kind, Scenarios}`, `Kind` being `kept` for a kept property; the caller supplies what the retained contract review recorded) and `Timeout` (the limit of one acceptance run; zero means the existing readiness limit). `RecheckReport` has `Verdict`, `Scenario`, `Command`, `ExitCode`, `Class`, `RunnerKind`, `AttributionOpen`, `Unavailable`, `Gate`, `Evidence` (lines) and `Text`.

Recheck copies `Pinned` to scratch copies outside both checkouts, overlays the draft's Intent directory and listed step files from `Project`, and runs the configured setup and the acceptance command with the existing deterministic parts of the contract-review controls check (shape/controls.go: `controlsRun`, the report reader, `referenceFailureLines`): first once on the unchanged tree, then once with the `correct` overlay of the draft's controls.json. It makes no provider or model request, never doubles a timeout, never retries, and never writes to `Project` or `Pinned`: the not-ready latch, the approval state, the readiness records and the draft files are unchanged, whatever the outcome.

`Verdict` is one of the request's three findings, written in its words, or empty when the evidence does not support one:
- `runner failure`: setup fails, or an acceptance run times out at `Timeout`, a tool is missing (exit 127), the command is killed by a signal, or it ends without a readable report of every scenario. `RunnerKind` says which, in words: `timeout`, `missing tool`, `killed command`, `incomplete report` or `setup error`. A setup failure keeps the existing class `setup_failure` of shape-setup-failure-classified-and-retried in `Class` with its command, exit code and last 20 output lines; `Class` is empty otherwise. A runner failure is reported after a single run of the failing command and no later run follows; it is never reported as a draft fault.
- `reference check failed`: a scenario that is not a kept scenario fails with the reference in place at a step, and does not fail the same way on the unchanged tree. `Scenario` and `Command` name it, and `Text` is the existing `referenceFailureLines` text of shape-failing-reference-gets-repair-round, not a copy.
- `bad draft`: a scenario of a `kept` property fails on the unchanged tree. `Scenario` and `Command` name it.

Empty `Verdict`: with `AttributionOpen` true, a scenario that is not kept fails at the same step with the same message on both trees; `Scenario`, `Command` and the evidence of both runs are returned and attribution stays open. With `Unavailable` true, `DraftHash` is empty or differs from the current draft hash, the draft is missing, or `Properties` is empty so a model-dependent gate has no retained evidence; `Gate` names the gate and `Evidence` what is missing, no acceptance run is made and nothing is inferred. With both false, no fault was found: every scenario passes with the reference in place, or only fails on the unchanged tree as a new feature does.

Ordinary Shape acceptance, full approval and every elapsed-time rule are unchanged. The regression case's retained draft stays at the runner checkout and a copy goes to the ops notes; that is an operational step outside this repository.

## Decisions
- The diagnostic is a library in package shape (`Recheck`, `RecheckDraftHash`) with no CLI, flag or environment variable; it takes the pinned checkout and the retained properties as input. (evidence: shape/controls.go; shape/approve.go) (source: request)
- It reuses `controlsRun`, the report reader and `referenceFailureLines` of shape/controls.go and the `setup_failure` class of shape/shape.go instead of cloning their text. (evidence: shape/controls.go; shape/shape.go; .kogen/intents/shape-failing-reference-gets-repair-round/intent.md; .kogen/intents/shape-setup-failure-classified-and-retried/intent.md) (source: request)
- Whether a property is kept comes only from the caller-supplied retained contract-review properties; without them the model-dependent outcome is unavailable with a Gate named and no inference. (evidence: shape/kept_baseline.go; shape/controls.go) (assumption: the retained review is supplied by the later CLI consumer as `RecheckProperty` values) (source: request)
- Runner outcomes (timeout, missing tool, setup error, killed command, unreadable report) stay a runner failure after one run, with no retry, no timeout doubling and no provider request. (evidence: shape/readiness_repeat.go) (source: request)
- Recheck never writes to either checkout; a passing diagnostic leaves the not-ready latch and existing approval state unchanged. (evidence: shape/draft_refusal.go; shape/approve.go) (source: request)
- A scenario that fails the same way on both trees and is not kept leaves attribution open, returned with evidence and no verdict. (assumption: "same way" means the same failing step line and normalized message) (source: request)
- The diagnostic is a library in package shape (Recheck, RecheckDraftHash) with no CLI, flag or environment variable; it takes the pinned checkout and retained properties as input. (decision: D1, round 1) (evidence: shape/controls.go; shape/approve.go) (source: request)
- It reuses controlsRun, the report reader and referenceFailureLines and the setup_failure class instead of cloning them. (decision: D2, round 1) (evidence: shape/controls.go; shape/shape.go) (source: request)
- Kept classification comes only from caller-supplied retained contract-review properties; without them the verdict is unavailable naming gate contract_review. (decision: D3, round 1) (evidence: shape/kept_baseline.go) (assumption: The later CLI consumer supplies the retained review as RecheckProperty values.) (source: request)
- Runner outcomes stay runner_failure after one run, with no retry, timeout doubling or provider request. (decision: D4, round 1) (evidence: shape/readiness_repeat.go) (source: request)
- Recheck never writes to either checkout; latch and approval state stay unchanged. (decision: D5, round 1) (evidence: shape/draft_refusal.go; shape/approve.go) (source: request)
- A non-kept scenario failing the same way on both trees is ambiguous with evidence and no attribution. (decision: D6, round 1) (assumption: Same way means the same failing step line and normalized message.) (source: request)

## Source comparison
- Addressed: correct-reference failure with scenario and command (scenario 1); kept failure on the unchanged tree distinguished (scenario 2); ambiguity left open (scenario 3); passing diagnostic leaves state unchanged (scenario 4).
- Addressed: timeout after a single run, missing tool, killed command and setup error as runner failures (scenarios 5 to 7).
- Addressed: mismatched draft and model-dependent gate make no inference and change no state (scenarios 8 and 9).
- Omitted: the Kogen CLI consumer, and keeping the regression case draft at the runner checkout with a copy in the ops notes (operational, outside this repository). Signals and real 120-second timeouts are scripted with a 3-second Timeout.

## Tests
- Correct-reference failure names scenario and command, kept failure is a bad draft, same failure leaves attribution open, passing is unchanged: scripted rechecks with the real Cucumber runner (scenario)
- Timeout, missing tool, killed command and setup error stay runner failures after a single run: same harness (scenario)
- Mismatched draft and missing model evidence make no inference and change no state: same harness (scenario)

## Feasibility
- none
