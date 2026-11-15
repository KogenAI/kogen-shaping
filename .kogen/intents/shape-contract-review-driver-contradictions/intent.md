# Review driver tests for self-contradictions

Slug: shape-contract-review-driver-contradictions
Model: not applicable — the change is review-instruction text; the evidence check already accepts the quotes needed, so no state or interacting rules are added
Acceptance:
- .kogen/intents/shape-contract-review-driver-contradictions/acceptance.feature
- features/steps/shape-contract-review-driver-contradictions__steps.js
- features/steps/shape-contract-review-driver-contradictions__driver/driver_test.go.txt

The Shape contract review misses acceptance driver tests that contradict themselves, so a Build is spent on an Intent that no implementation can satisfy: the Builder then ends `build_blocked` with a blocker report naming a defect inside the shaped driver test itself, not a conflict with the Intent text. This change extends `contractReviewInstructions` in `shape/contract.go` so the review also reports a finding when an expectation in a candidate step file cannot hold because of the same test's own earlier statements or the fixtures it relies on, in exactly three cases:

1. A record or value the test removed or replaced earlier in the same test is used again later (for example a fixture record destroyed at one line and looked up again at a later line).
2. A created or numbered record collides with a uniqueness rule that repository code the Intent keeps unchanged enforces and that the fixtures already occupy (for example a unique index on `cards(account_id, number)` with fixtures already owning numbers 1-3, or a counter reset that makes the next number repeat one already created).
3. An expected value is computed in a different request or tenant context than the one the checked output is produced in (for example an expected href built outside the `untenanted` block while the page renders untenanted).

The instructions must contain these literal phrases (matched case-insensitively): `removed or replaced earlier in the same test`, `uniqueness rule` and `different request or tenant context`. They must tell the reviewer to give the expectation as `expectation` and, as `contract`, the exact earlier statement from a candidate file in the package (or the exact text of the unchanged repository file) that makes it impossible. They keep the existing sentences `A scenario that fails only because the requested feature is missing is not a finding` and `When unsure, report nothing.` unchanged.

The finding keeps the existing shape. `contractEvidence.evidenced` already accepts a `contract` quote that is an exact (whitespace-normalized) quote of a candidate file in the package, including the same file as the expectation, and rejects an invented one; that behaviour stays unchanged. No new gate, tool or configuration. Which drafts pass stays unchanged unless a draft contains one of these self-contradictions.

The Build also adds a focused model-free regression test next to `shape/contract_cross_scenario_test.go` (in package `shape`): a scripted contract review reporting a finding whose contract quote is an earlier statement of the same step file is accepted as evidenced and refuses the draft; the same finding with an invented quote is not evidenced; and `contractReviewInstructions` contains the three new categories.

## Decisions
- The change is limited to the text of `contractReviewInstructions`; the review and confirmation call flow, finding schema and evidence check stay as they are because `evidenced` already accepts same-file candidate quotes. (decision: D1, round 1) (evidence: shape/contract.go; shape/contract_cross_scenario_test.go) (source: request)
- The new instruction is one added sentence after the cross-scenario comparison sentence and restates that a scenario failing only because the feature is missing is not a finding and that unsure means report nothing. (decision: D2, round 1) (evidence: shape/contract.go) (source: request)
- Acceptance runs a driver Go test, supplied by this Intent as a `.go.txt` helper, in a scratch copy of the checkout, so it needs no test the Build must invent; it uses only pre-existing helpers (`newShpFx`, `shpFiles`, `ccReview`, `ccFindingArgs`). (decision: D3, round 1) (evidence: shape/contract_cross_scenario_test.go) (assumption: the Go toolchain and module cache are available offline in the acceptance environment) (source: request)

## Source comparison
- Request: contract review instructions gain the three self-contradiction categories — addressed by the new sentence in `contractReviewInstructions` (acceptance scenario "The review instructions name the three self-contradiction categories").
- Request: finding uses existing shape and quote evidence, earlier-statement quote must be an exact quote from a candidate file — addressed; the existing `evidenced` already behaves so (scenario "A quote of an earlier statement of the same step file is evidence only when exact").
- Request: scripted review with such a finding is evidenced and refuses the draft — addressed (scenario "A scripted review reporting such a finding refuses the draft").
- Request: the observed `build_blocked` cells motivate the change; unsure reports nothing and a missing-feature failure is not a finding — kept and asserted.
- Request: no new gate, tool or configuration — respected; only instruction text changes.

## Tests
- The instructions name the removed-or-replaced record, uniqueness collision and request-or-tenant context categories and keep the exclusions: a Go test in package shape next to shape/contract_cross_scenario_test.go asserting `contractReviewInstructions` content (unit)
- An exact earlier-statement quote of the same step file is evidenced, an invented one is not: a Go test of `contractEvidence.evidenced` in the same file (unit)
- A scripted contract review with such a finding refuses the draft: a Go test using the scripted shape fixture in the same file (integration)
- All three rules end to end through the acceptance driver: the acceptance.feature scenarios (scenario)

## Feasibility
- none
