# Split shape time checks by gate

Slug: shape-time-splits-checks-by-gate
Model: not applicable — the change only reports durations of existing checks and holds no state
Acceptance:
- .kogen/intents/shape-time-splits-checks-by-gate/acceptance.feature
- features/steps/shape-time-splits-checks-by-gate__steps.js

The shape time account lumps every Kogen check into `checks_ms`. The `shape_time` event and the progress line `kogen: shape time ...` now also name the Kogen checks by gate: static checks, model checks (decision, literal, example), contract review (review, confirmation and wrong-implementation check) and readiness. Each gate is the union of its wall-clock spans, so overlapping spans count once per gate. The new `shape_time` fields are `static_ms`, `model_checks_ms` and `contract_ms`; `readiness_ms` already exists. The existing fields keep their names and meaning, and `checks_ms` stays the union of all Kogen check spans. A run without a contract review reports `contract_ms` 0. The progress line reads `Kogen checks <checks> (static <s>, model checks <m>, contract review <c>, readiness <r>)`, each figure in seconds with one decimal as `seconds` already formats them; the `unknown` handling of model time and the rest of the line are unchanged. No change to which drafts pass, to any refusal text or to the order of the checks. The existing time account tests that quote the old progress line or the old exact `shape_time` event are updated to the new form.

The gates correspond to evidence Kogen already records: the `static_checks` events (static), the `executor-decision`, `executor-literal` and example sessions (model checks), and the `executor-contract-review` and `executor-contract-confirm` sessions plus the wrong-implementation check (contract review).

## Decisions
- The gate spans are timed where the gates already run: the static checks and the model checks (decision, literal, example) in postSubmitChecks (shape/checkorder.go) and the contract review (review, confirmation and wrong-implementation check) in checkSubmission (shape/shape.go), through the existing Evidence.Timed phase intervals, read back by Finish in shape/evidence.go. (decision: D1, round 1) (evidence: shape/evidence.go; shape/checkorder.go; shape/shape.go; shape/contract.go) (source: request)
- Refine's contract review is not split, because only Shape's run records a shape_time account. (decision: D2, round 1) (evidence: shape/evidence.go) (assumption: the user confirms that Refine needs no gate split) (source: request)

## Source comparison
- Addressed: `shape_time` gains `static_ms`, `model_checks_ms`, `contract_ms`; the progress line names static, model checks, contract review and readiness (source: request; acceptance.feature, both scenarios).
- Addressed: spans that overlap count once per gate and `checks_ms` stays the union of all of them (acceptance.feature, first scenario: the decision and literal checks overlap).
- Addressed: a run without a contract review reports `contract_ms` 0 (acceptance.feature, second scenario).
- Addressed: the request's evidence names `static_checks` events and the `executor-contract-review`, `executor-contract-confirm`, `executor-decision` and `executor-literal` sessions as where the time lies; they define the gates and need no change.
- Unchanged: existing field names and meaning, one-decimal `seconds`, `unknown` model time, refusal texts, check order.

## Tests
- shape_time and the progress line split the Kogen checks by gate with per-gate values at most checks_ms: add a model-free scripted shaping run test with a configured sleeping check and scripted decision, literal and contract answers next to TestShapeTimeAccount in shape/evidence_test.go (integration)
- a run without a contract review reports contract_ms 0 and still names all four gates: the same test file, a run refused before any review (integration)
- the existing time account and runtime install tests quote the new line and event: update TestShapeTimeAccount, TestShapeTimeLine and TestShapeRuntimeInstallAccounting (existing)

## Feasibility
- none
