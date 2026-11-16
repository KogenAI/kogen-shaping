# Print progress for external Shape checks

Slug: shape-external-checks-print-progress-lines
Model: not applicable — progress-only instrumentation adds no meaningful domain state
Acceptance:
- .kogen/intents/shape-external-checks-print-progress-lines/acceptance.feature
- features/steps/shape-external-checks-print-progress-lines__steps.js
- features/steps/shape-external-checks-print-progress-lines__driver/observe_test.go.txt

Remove the silent gap while an external Shape runs its decision and request-literal checks. Change only progress output in the external branches of shape/decision.go and shape/literal.go, and add one focused, model-free Go regression test in the shape package. Do not change check requests, results, events, refusals, draft eligibility, native output, configuration, or any other tooling. No browser, Rails or Studio work belongs to this change.

## Decisions
- Print each external check's existing native start wording once before its executor round, followed after that round by one check-specific finished line with that round's elapsed whole seconds, not the shaping session's total elapsed time. Preserve the native branches exactly. (source: request)
- Use the existing progress formatter for external start lines and bare progress for finished lines, timing x.round with time.Now and truncating time.Since to whole seconds so the finished line has only one duration suffix. (decision: D1, round 1) (evidence: shape/shape.go; shape/decision.go; shape/literal.go) (source: request)
- Add one focused Go regression test using the existing scripted external-executor helpers used by the decision and literal tests; exercise both completed checks across checked submissions, asserting one start and finish per round and start-before-round/finish-after-round ordering. Keep useful existing native tests unchanged. (evidence: shape/lineup_shape_test.go; shape/decision_test.go; shape/literal_test.go) (source: request)
- Preserve all existing check inputs, schemas, payload parsing, accounting, events, feedback, refusals and publication decisions; no new configuration or production abstraction is needed. (source: request)

## Source comparison
The request's Expected paragraph is addressed by the first two Decisions and the executable external scenario: both exact start strings and both check-specific finished strings are carried in acceptance.feature. The start formatter keeps its existing session-elapsed suffix; the new finished format reports only the round duration as an integer number of seconds. The request's native counterexample is addressed by the same scenario's native execution and comparison to an actually observed native output record, normalizing only changing numeric measurements. Native response wording and usage/wall_ms fields remain unchanged. The scope and constraints are addressed by the prose, preservation Decision, and the single focused regression test; the existing external decision/literal tests remain evidence for unchanged requests, events and refusal behavior. The user's historical AoR release, cells, timings, event names and lab path are retained as contextual observations in acceptance.feature; the executable local reproducer tests the same silent branches without claiming access to those external files or promising historical durations. No requested behavior is deliberately omitted or superseded; the obsolete silent external progress behavior is replaced, not retained as an option. The one wrong implementation in controls.json restores silent external branches, as requested.

## Tests
- External decision and literal rounds announce their execution exactly once before and once after each completed round with whole-second duration: add one focused model-free Go test in shape using shapeExec, and exercise the real Shape entry point with a local scripted executor in acceptance (regression)
- Native check progress is unchanged: preserve existing native tests and compare native Shape progress against the recorded baseline, allowing numeric timing and token measurements to vary (regression)
- Check requests, results, events, refusals and draft eligibility stay unchanged: retain TestExternalShapeDecisionCheck and TestExternalShapeLiteralCheck unchanged and preserve the existing check pipelines (existing)

## Feasibility
- The existing focused decision/literal fixtures and native passing-submission fixture ran successfully with go test ./shape -run '^(TestExternalShapeDecisionCheck|TestExternalShapeLiteralCheck|TestPassingSubmissionReachesCertifyAsBefore)$' -count=1 (probe: exit 0). A built, overlay-injected Go observer also recorded a successful native Shape's actual progress without calling a live model (probe: exit 0).
- evidence: {"id":"progress-observer","kind":"observer","inputs":["shape/decision.go","shape/literal.go",".kogen/intents/shape-external-checks-print-progress-lines/controls.json",".kogen/intents/shape-external-checks-print-progress-lines/observer-expectations.json",".kogen/intents/shape-external-checks-print-progress-lines/native-observation.json"],"observers":["features/steps/shape-external-checks-print-progress-lines__steps.js","features/steps/shape-external-checks-print-progress-lines__driver/observe_test.go.txt"],"checks":[{"name":"good","command":"node features/steps/shape-external-checks-print-progress-lines__steps.js qualify good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-external-checks-print-progress-lines__steps.js qualify bad","exit":1,"output":"observer rejected\n"}]}
