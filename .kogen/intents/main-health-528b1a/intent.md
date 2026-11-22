# Restore the blocked report's Next line for the

Slug: main-health-528b1a
Model: not applicable — the change is one fixed output sentence with no state
Acceptance:
- .kogen/intents/main-health-528b1a/acceptance.feature
- features/steps/main-health-528b1a__steps.js

The main-health qualification of the current component main heads fails one assembly scenario, "A required protected edit is reported without creating an Intent or approval" (spec/features/02-shaping.feature:224 of the kogen assembly repository), with `stdout did not contain expected text`. The scenario expects the third line of Shape's blocked report to be exactly:

`Next: revise the request, or review and commit .kogen/project.yaml outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.`

Shape prints a different third line since commit e5d8e26 ("Make the blocked report's Next line name no file"): `Next: revise the request so it needs no edit to the files listed above, or review and commit those files outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.` This Intent restores the third line the assembly spec expects, so the scenario passes again. The spec scenario is not removed, skipped or weakened. The first two lines of the report, the exit code (1), the blocked-path validation and every other behaviour stay unchanged. The new sentence is removed completely from `shape/shape.go`, `shape/shape_test.go`, `shape/contract_test.go` and from the constant `kbrNext` of `features/steps/k-shape-blocked-report-repair__steps.js`, which hold it today and are updated to the restored line. No file other than these four is edited; no flag, setting or environment variable is added.

## Decisions
- The third line of a blocked report is again the one fixed sentence the assembly spec expects, naming .kogen/project.yaml, for every blocked report; it is not special-cased by file or by scenario, and it adds no setting. (decision: D1, round 1) (constraint: one sentence for every blocked report; no per-path branch; spec scenario unchanged) (evidence: shape/shape.go; shape/shape_test.go; shape/contract_test.go) (assumption: the assembled spec at spec/features/02-shaping.feature:224 is the contract and e5d8e26 is the change that broke it, so the fix is made here) (source: request)
- The cause is a conflict between this repository's commit e5d8e26 and the kogen repository's spec scenario, which this repository cannot edit. If the new wording is wanted instead, the owner of that change is the kogen repository (spec/features/02-shaping.feature:224 and its step expectation), and this Intent's restored line must then be reconsidered. (decision: D2, round 1) (evidence: .kogen/intents/shape-blocked-next-line-names-blocked/intent.md; shape/shape.go) (assumption: the earlier Intent shape-blocked-next-line-names-blocked, which asked for the new sentence, is superseded by this one; its own acceptance scenarios stop passing and are not part of the configured checks) (source: request)
- The existing step file features/steps/k-shape-blocked-report-repair__steps.js is edited for its kbrNext constant rather than listed under Acceptance, because the Build must change it and this Intent's acceptance only runs it. (decision: D3, round 1) (evidence: features/steps/k-shape-blocked-report-repair__steps.js) (assumption: that step file is not protected by another Intent) (source: request)

## Source comparison
- Addressed: the failing assembly scenario spec/features/02-shaping.feature:224 and its expected Next line; the third line of the blocked report is the sentence naming .kogen/project.yaml (shape/shape.go printBlocked; request).
- Addressed: the two Go tests and the repair feature step constant that assert the line assert the restored line (shape/shape_test.go, shape/contract_test.go, features/steps/k-shape-blocked-report-repair__steps.js).
- Superseded: the earlier Intent shape-blocked-next-line-names-blocked, whose new sentence is removed.
- Preserved: the first two output lines, exit code 1, no Intent or approval written, no new flag, setting or environment variable, no change to release scripts.

## Tests
- The blocked report for a path protected by the project configuration prints exactly three lines with the restored Next line, exit 1, nothing written: acceptance scenario with a scripted shaper through the real Shape command, run for a `.style.cfg` path, a glob-protected path and `.kogen/project.yaml` (scenario)
- The blocked-report repair feature journeys still pass with the restored constant: acceptance scenario (scenario)
- The Go sources hold only the restored line: update TestShapeBlockedReportIsPrintedAndWritesNothing and TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob, which reproduce the failure on main before the change and pass after it (regression)

## Feasibility
- none
