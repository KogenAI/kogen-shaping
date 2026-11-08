# Preserve corrections across shaping retries

Slug: k-shape-refine-retry-keeps-corrections
Model: .kogen/intents/k-shape-refine-retry-keeps-corrections/model.qnt invariants=Retained,Atomic seed=1 samples=100 steps=10
Acceptance:
- .kogen/intents/k-shape-refine-retry-keeps-corrections/acceptance.feature
- features/steps/k-shape-refine-retry-keeps-corrections__steps.js

Within one refinement or shaping invocation, retain well-formed unpublished proposals even when lint or readiness rejects their candidate. Reconstruct each retry from the frozen original plus all accumulated proposals, with latest supplied content winning per path. Omitted paths retain previous proposed content. Precheck, readiness and publication must consume the same complete candidate. Failure must not modify the published package. Preserve existing validation and ownership restrictions; no cross-invocation proposal replay is requested.

The reported Skeptic.bot refinement fb4ef144cf2d11bc submitted four corrected artifacts (Intent, Gherkin, JS, ExUnit) in submission 1666; rejection concerned an unsourced Intent decision. Submission 1858 supplied only Intent.md, losing the other three corrections. This repair covers arbitrary owned artifacts, not only the three-file acceptance fixture. Replace reset-and-overlay-only-latest behavior, not the frozen original used for rollback.

## Decisions
- Retain omitted earlier corrections within one invocation: “a new submission overlays on top of them, so files it omits keep the earlier proposed content.” (source: request)
- Keep publication atomic: “Failure still leaves the original package untouched.” (source: request)
- Apply the same cumulative semantics to shaping correction precheck and publication: “The adopted repair is cumulative proposals within one invocation.” (source: request)

## Source comparison
- Addressed KOGEN-024 (b1e22f8a-3936-43bd-9387-5fe517d14b7e): the outcome paragraph and refinement scenario preserve submission 1's B and C after submission 2 supplies only A; the historical four-artifact example above remains in scope.
- Addressed KOGEN-026 (afe52432-512f-4ad3-981a-605b602a0db9): the same outcome applies to shaping precheck, readiness and publication, exercised by the shaping scenario.
- Addressed request “failed refinement leaves the published package unchanged” through the failed-refinement scenario; “existing shaping suite passes” through the existing configured go-test check (GOFLAGS=-trimpath go test ./...), rather than duplicating the full suite inside the time-limited acceptance runner.
- No requested behavior is omitted or superseded. The acceptance fixture uses the existing Intent/Gherkin/JS three-artifact reproducer for A/B/C; ExUnit is a historical artifact type, not a new runtime dependency.

## Tests
- Rejected first candidate followed by a narrower retry retains all earlier owned corrections: TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness (regression)
- A failed refinement leaves published artifacts and approval unchanged: TestRefineFailedAndCancelledApprovedCorrectionsPreservePackageRawApprovalAndIndex (integration)
- Shaping correction precheck and final publication use accumulated proposals: TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals (regression)
- Existing shaping behavior remains intact: configured go-test check, GOFLAGS=-trimpath go test ./... (existing)

## Feasibility
- Local named-test runner format and filters were checked against the existing tests; model typechecking was checked after renaming a reserved action. The full shaping suite takes longer than the acceptance runner's limit, so it remains in configured checks. No external dependency or protocol change is required.
