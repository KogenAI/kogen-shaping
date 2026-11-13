# Shape the simplest necessary behavior

Slug: simplify-shaping-necessary-behavior
Model: not applicable — this edits fixed policy text and constants; it has no state
Acceptance:
- .kogen/intents/simplify-shaping-necessary-behavior/acceptance.feature
- features/steps/simplify-shaping-necessary-behavior__steps.js

The Shaper's policy (shape/prompts/shaper.md) requires a demonstrated variability need before it proposes a setting, flag, environment variable or option; a fixed value is a constant; one fixed request (for example a fixed timeout) yields one behavior with no configuration; no unused feature or compatibility is proposed. Its rules on step-file ownership, withdrawing proposals and the advisory review estimate are reconciled with real requirements while keeping collision and protected-file safety. The old wording is replaced, not kept beside the new one.

The request names shape/toolfree.go:8 as keeping a 300-request cap. In the current tree that line is `return sinceSubmit >= noToolLimit`, a plain constant (3); no 300-request cap exists there, so nothing is removed for it and no cap is added.

Out of scope: the wall-clock 15-minute shaping budget (`shapingBudget`, `ShapeInput.Now`) and the scope-enumeration problem. Both stay unchanged.

## Decisions
- The policy sentence on configuration is reworded to require a demonstrated variability need that the request or current code shows, with fixed values as constants and no unused feature or compatibility; the old sentence "an actual variability need that must stay supported" is removed. (evidence: shape/prompts/shaper.md) (source: request)
- The policy states that Shape may reuse an existing step file or helper of the Intent when it is slug-owned and unchanged, and that only collisions with another Intent's files and protected paths are refused; no stack-specific rule refuses a feasible correction or reuse. (evidence: shape/prompts/shaper.md; shape/shape.go) (assumption: the exact wording is the Builder's, the acceptance pins the key phrases) (source: request)
- The policy states that a proposal can be withdrawn when it is no longer needed, and that review-scope estimates are advisory: an estimate that only predicts future review scope is reported, not refused. (evidence: shape/scope.go; shape/prompts/shaper.md) (assumption: the request's estimate refusal is read as policy text, not a change of scope.go limits, which the scope-enumeration work owns) (source: request)
- shape/toolfree.go:8 is `return sinceSubmit >= noToolLimit` with `noToolLimit = 3` a constant; the 300-request cap the request names is not present, so no change is made there. `submissionRetryLimit` and `noToolLimit` stay constants with no test-only override. `ShapeInput.Now` stays, because the budget correction owns it. (evidence: shape/toolfree.go; shape/force.go; shape/shape.go; shape/budget.go) (source: request)
- Goldens under shape/testdata change only for the prompt bytes this edit changes, regenerated with `go test ./... -update`. (evidence: README.md) (source: request)

## Source comparison
- Fixed-timeout request yields one behavior, no configuration: addressed in the policy sentence and acceptance scenario 1.
- Genuine variability need stays supported: acceptance scenario 2.
- Old forms removed completely (grep): acceptance scenario 3.
- Feasible correction or reuse not refused by a stack-specific rule: acceptance scenario 4.
- shape/toolfree.go:8 "300-request cap": superseded; the current code has no such cap (see above).
- Readiness and ownership checks still pass, full go test: the configured checks; the other line numbers named in the request no longer match the code, so the current text is the reference.
- 15-minute budget and scope enumeration: omitted by request.

## Tests
- The policy requires a demonstrated variability need and constants for fixed values: acceptance scenarios 1 and 2 (scenario)
- Old wording is gone: grep scenario (scenario)
- Reuse and withdrawal are not refused by a stack-specific rule: scenario 4 (scenario)
- Existing readiness, ownership and policy golden tests: existing suite (existing)

## Feasibility
- none
