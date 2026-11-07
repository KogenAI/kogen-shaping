# Preserve probe result progress visibility

Slug: k-shape-progress-probe-result
Model: not applicable — this is console reporting, with no meaningful product state
Acceptance:
- .kogen/intents/k-shape-progress-probe-result/acceptance.feature
- features/steps/k-shape-progress-probe-result__progress.js

After each shaping probe command, print one console line containing its exit status and elapsed time without copying probe stdout or stderr into that line. Preserve existing progress lines. Preserve the existing implementation where it already satisfies this contract; strengthen regression coverage rather than introduce a replacement reporting mechanism. This addresses Studio issue c6fbe7f5-1a01-4546-b9e1-cf04d2364a84, KOGEN-005 — Visibility while shaping is in flight, reported after a 709.7 s shaping run. That duration is historical context, not a performance threshold.

The package includes intent.md and acceptance.feature as requested.

## Decisions
- Report both successful and nonzero probe exits without echoing sensitive tool output; leave existing progress lines unchanged. (source: request)
- Preserve the existing result-line format rather than introducing a second result line. (evidence: shape/shape.go) (source: request)

## Source comparison
- Addressed: request acceptance 1–4 map to the status and privacy regression scenarios and the Tests rules below; both successful and nonzero probes require status and elapsed time.
- Addressed: the original observation of numbered model requests, elapsed time and probe commands establishes the existing progress that must remain unchanged.
- Addressed: the supplied Studio issue identifier and 709.7 s observation are retained above, without inventing a latency requirement.
- Addressed: the delivery instruction names intent.md and acceptance.feature; these are the package filenames.
- Superseded: the assumption that result reporting is absent is not an instruction to duplicate the existing result reporting in shape/shape.go.
- Omitted: no requested behavior is deliberately removed; no provider output, new settings or new progress streams are added.

## Tests
- Successful probe completion reports code and elapsed time: retain the existing successful-probe regression through the shaping entry point (regression)
- Nonzero probe completion reports code and elapsed time: retain the existing failure-probe regression through the shaping entry point (regression)
- Result lines exclude probe stdout and stderr: add focused regression assertions using distinct stdout and stderr sentinels for both exit outcomes (regression)
- Existing progress lines remain unchanged: retain existing command and model-progress assertions and add exact progress-line comparisons excluding only variable elapsed fields (regression)

## Feasibility
- none
