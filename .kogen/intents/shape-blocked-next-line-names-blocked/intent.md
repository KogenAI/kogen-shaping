# Make the blocked report's Next line name no file

Slug: shape-blocked-next-line-names-blocked
Model: not applicable — the change is one fixed output sentence with no state
Acceptance:
- .kogen/intents/shape-blocked-next-line-names-blocked/acceptance.feature
- features/steps/shape-blocked-next-line-names-blocked__steps.js

The blocked report printed by Shape prints three lines. The third line is a fixed sentence that sent every blocked report to `.kogen/project.yaml`, even when the blocked files were others (for example `cli/go.mod` and `cli/go.sum`). The third line no longer names `.kogen/project.yaml`. It reads exactly:

`Next: revise the request so it needs no edit to the files listed above, or review and commit those files outside Build, then shape again; preserve checks and actual check configuration. There is no protected-edit override.`

The line is the same for every blocked report and does not change with the kind of file. The first two lines, the blocked-path validation, the exit code (1), prompts, docs and every other file stay unchanged. The old sentence is removed completely from `shape/shape.go`, `shape/shape_test.go`, `shape/contract_test.go` and from the constant `kbrNext` in `features/steps/k-shape-blocked-report-repair__steps.js`, which hold it today and are updated to the new line.

## Decisions
- The Next line of a blocked report is one fixed sentence, the exact text above, for every blocked report; no per-file or per-file-type advice, no new setting. (evidence: shape/shape.go) (source: request)
- The existing step file features/steps/k-shape-blocked-report-repair__steps.js is edited (its kbrNext constant) rather than listed under Acceptance, because the Build must change it and this Intent's acceptance only reads it and runs it. (evidence: features/steps/k-shape-blocked-report-repair__steps.js; shape/shape_test.go; shape/contract_test.go) (assumption: the earlier Intent's step file is not otherwise protected from this edit) (source: request)
- The Next line of a blocked report is one fixed sentence for every blocked report, with no per-file advice. (decision: D1, round 1) (constraint: No setting or per-file-type branch) (evidence: shape/shape.go) (assumption: none) (source: request)
- The earlier Intent's step file k-shape-blocked-report-repair__steps.js is edited for its kbrNext constant, not listed under Acceptance. (decision: D2, round 1) (evidence: features/steps/k-shape-blocked-report-repair__steps.js; shape/shape_test.go; shape/contract_test.go) (assumption: Assumes that step file is not protected by another Intent.) (source: request)

## Source comparison
- Addressed: the third line of the blocked report no longer names .kogen/project.yaml and reads exactly the new sentence (shape/shape.go printBlocked; request).
- Addressed: the two Go tests and the repair feature step constant that assert the old line now assert the new line (shape/shape_test.go, shape/contract_test.go, features/steps/k-shape-blocked-report-repair__steps.js).
- Preserved: the first two output lines, exit code 1, no Intent or approval written, blocked-path validation.
- Omitted by request: prompts, docs and every other file.

## Tests
- The blocked report for a glob-protected path prints exactly the three lines with the new Next line, exit 1, nothing written: acceptance scenario with a scripted shaper through the real Shape command (scenario)
- The blocked report for .kogen/project.yaml prints the same new Next line: acceptance scenario (scenario)
- The blocked report for cli/go.mod and cli/go.sum prints the same new Next line and its third line does not contain .kogen/project.yaml: acceptance scenario (scenario)
- The blocked-report repair feature's journeys still pass with the new step constant and the Go sources hold only the new line: acceptance scenario (scenario)
- The Go tests that assert the blocked Next line expect the new line: update TestShapeBlockedReportIsPrintedAndWritesNothing and TestShapeBlockedReportRecognizesProtectedPathsMatchedByAGlob (unit)

## Feasibility
- none
