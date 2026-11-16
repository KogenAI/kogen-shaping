# List the last submission in the budget refusal

Slug: shape-budget-stop-lists-last-submission-checks
Model: not applicable — the change only adds a line to a refusal text and holds no state
Acceptance:
- .kogen/intents/shape-budget-stop-lists-last-submission-checks/acceptance.feature
- features/steps/shape-budget-stop-lists-last-submission-checks__steps.js
- features/steps/shape-budget-stop-lists-last-submission-checks__driver/harness_test.go.txt

When a shaping ends in the budget refusal and its last submission was checked but not refused by a gate, the lines after the budget line also list that submission in the same format as a refused attempt: `attempt N: gate <the check that ended the run>, model request X s, Kogen checks Y s, Z s remaining`. This is refusal text only: the budget, which checks run past it, the events and which drafts pass do not change, and no setting is added. The lines of refused attempts stay exactly as they are.

The extra line is for the last submission only (the one the shaper received readiness failures for, to acknowledge), listed after the refused attempts. Z is the budget left when its checks ended, never below 0. When its checks ended past the budget, the line ends with `, N s over` (N whole seconds past the budget, truncated), and only then; a refused attempt's line never has it. `<the check that ended the run>` is `contract` when the contract review or confirmation ran in that submission, otherwise `readiness`. The existing `install_ms` suffix applies to the extra line as to the others.

## Decisions
- The extra line uses the refused-attempt format and a new optional `, N s over` suffix, shown only when the checks ended past the budget and only on this line (assumption: the gate names `contract` and `readiness` name the last check that ran; the request leaves the names open) (source: request)
- Only the last submission is listed, and only when it was checked without a gate refusal, so earlier unrefused submissions add no lines (evidence: shape/budget.go; shape/shape.go) (source: request)
- The acceptance harness is a Go test file copied into the shape package through a Go overlay so that no test name has to be invented by the Build (evidence: shape/budget_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (source: request)
- The extra line uses the refused-attempt format plus an optional `, N s over` suffix on the last unrefused submission only (decision: D1, round 1) (evidence: shape/budget.go; shape/shape.go) (assumption: The gate names contract and readiness for the unrefused line are my choice) (source: request)
- Only the last submission is listed, when checked without a gate refusal (decision: D2, round 1) (evidence: shape/budget.go; shape/shape.go) (source: request)
- Acceptance copies a Go harness test into the shape package through a Go overlay so no test name must be invented (decision: D3, round 1) (evidence: shape/budget_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (source: request)

## Source comparison
- Addressed: the budget refusal also lists the last checked, unrefused submission as `attempt N: gate <the check that ended the run>, model request X s, Kogen checks Y s, Z s remaining`, with `, N s over` appended when the checks ended past the budget; see the Acceptance scenario (source: request).
- Unchanged: refused-attempt lines, the budget, the checks that run past it, the events and which drafts pass (source: request).
- Omitted: any new setting or configuration (source: request).

## Tests
- The budget refusal lists the last unrefused submission with 0 s remaining and the seconds over: add the model-free Go test `TestBudgetStopListsTheLastSubmissionChecks` in shape/budget_test.go using its scripted shaper helpers and clock (unit)
- Refused-attempt lines stay exactly as they are: the existing budget tests in shape/budget_test.go (existing)
- The refusal of a scripted external shaping lists both attempts through the real entry point: the acceptance scenario (scenario)

## Feasibility
- none
