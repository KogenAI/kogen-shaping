# Report missing controls.json with contract

Slug: shape-missing-controls-reported-with-findings
Model: not applicable — one refusal now carries lines that were produced one round later; no new state or interacting rules
Acceptance:
- .kogen/intents/shape-missing-controls-reported-with-findings/acceptance.feature
- features/steps/shape-missing-controls-reported-with-findings__steps.js
- features/steps/shape-missing-controls-reported-with-findings__harness.js

When the contract review refuses a draft with confirmed findings and the properties it reviewed include at least one counted observable property (the condition `wrongImplementationCheck` already uses to decide that the draft needs controls.json), the same refusal also carries the controls.json problems that need no model call and no run of any implementation: the line saying the file does not exist, or the problems `parseControls` reports for a controls.json that violates the schema. These lines come after the contract finding lines, in the order `contractConfirmed.all()` already uses for the wrong-implementation lines, under the wrong-implementation headline, so the shaper repairs both in one round.

Unchanged: the runs of the correct and wrong implementations still happen only for a draft whose contract review is clean; a deferred controls.json (a non-blank reason) adds nothing; a draft whose review has no counted observable property gets no controls line; the gate name (`contract_review`), the problem texts, one refusal and the retry budget counting; which drafts pass. The refusal of a clean review without controls.json is exactly today's single line and headline. No new gate, tool or configuration.

## Decisions
- The controls.json problems that need no run are added to the refusal of a contract review that has confirmed findings and a counted observable property, after the finding lines and under the wrong-implementation headline, sharing the preconditions (counted property, deferral, missing file, `parseControls`) with `wrongImplementationCheck` and never starting a run. (evidence: shape/shape.go; shape/controls.go; shape/contract.go) (source: request)
- A review without a counted observable property, a deferred controls.json and a valid controls.json add no line to a refusal that has findings, and nothing runs for it. (evidence: shape/controls.go) (source: request)
- The refusal of a clean review is unchanged: the single line and headline of `wrongImplementationCheck`, with its runs. (evidence: shape/shape.go; shape/controls.go) (source: request)
- The acceptance drives the real Shape flow with a scripted provider, scripted contract review and confirmation, and reads the first feedback to the shaper, the recorded `submission_refused` problems and the count of `kogen: wrong-implementation check: run` progress lines. (decision: D4, round 1) (evidence: shape/contract_check_test.go; shape/property_test.go) (source: request)
- Add the run-free controls.json problems (missing file, parseControls schema problems) to the refusal of a contract review that has confirmed findings and a counted observable property, after the finding lines under the wrong-implementation headline, never starting a run. (decision: D1, round 1) (constraint: gate contract_review and problem texts unchanged) (constraint: deferred controls.json adds nothing) (constraint: no counted property adds nothing) (evidence: shape/shape.go; shape/controls.go; shape/contract.go) (assumption: Exact refusal line counts of the combined feedback were derived from reading contractFeedback, not from running it.) (source: request)
- Acceptance drives the real Shape flow with scripted provider and contract review and reads feedback, submission_refused problems and wrong-implementation run progress lines. (decision: D2, round 1) (evidence: shape/contract_check_test.go; shape/property_test.go) (assumption: The harness was not compiled or run in this session; the end of the scripted Shape flow after the first refusal is deliberately not judged.) (source: request)
- Move the no-run assertions into their own scenario and add a deferred controls.json case, so kept behaviour is asserted by steps that run on the unchanged tree. (decision: D3, round 1) (evidence: shape/controls.go) (assumption: Line counts of the combined feedback come from reading contractFeedback; the first round's observation confirmed the baseline refusal text and the harness reaching the first feedback.) (source: request)

## Source comparison
- Addressed: a review with one finding and one counted observable property, no controls.json: one refusal with the finding and the does-not-exist line (scenario 1).
- Addressed: the same with a controls.json that violates the schema: the finding and the schema problem (scenario 2); no implementation is run in any of these cases (scenario 6).
- Addressed: one finding and no counted property, no controls.json: the finding only (scenario 3).
- Addressed: no finding and a counted property, no controls.json: exactly the single controls.json line and the wrong-implementation headline as today (scenario 4).
- Addressed: a deferred or valid controls.json adds nothing to a refusal with a finding (scenario 5).
- Preserved: gate name, problem texts, retry counting, runs only after a clean review.

## Tests
- A confirmed finding with a counted property and no controls.json is refused with the finding and the missing-file line: a new model-free test in `shape/controls_test.go` using the scripted contract review helpers, and acceptance scenario 1 (regression)
- A controls.json that violates the schema is listed with the finding: a new test in `shape/controls_test.go`, and acceptance scenario 2 (regression)
- No counted property: no controls line: a new test in `shape/controls_test.go`, and acceptance scenario 3 (regression)
- A clean review keeps today's single line and headline: a new test in `shape/controls_test.go`, and acceptance scenario 4 (regression)
- A deferred or valid controls.json adds nothing to a refusal with a finding: a new test in `shape/controls_test.go`, and acceptance scenario 5 (regression)
- No implementation runs for a draft with a finding, whatever its controls.json: a new test in `shape/controls_test.go` counting the scripted runner's runs, and acceptance scenario 6 (regression)
- The existing controls and contract tests stay green: the configured go-test check (existing)

## Feasibility
- none
