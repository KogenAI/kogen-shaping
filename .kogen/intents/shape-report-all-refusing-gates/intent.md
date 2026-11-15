# Report readiness refusals with the cheap gates

Slug: shape-report-all-refusing-gates
Model: not applicable — the change decides which already-existing gate refusals one submission reports; the executable scenarios observe the reported gates and feedback directly
Acceptance:
- .kogen/intents/shape-report-all-refusing-gates/acceptance.feature
- features/steps/shape-report-all-refusing-gates__steps.js
- features/steps/shape-report-all-refusing-gates__harness.js
- features/steps/shape-report-all-refusing-gates__driver/driver.go

Shape stops at the first refusing gate, so a draft with several defects needs several rounds and exhausts the retry budget. The independent cheap gates already come back together in one refusal; the readiness step (checkSubmission's certify, with its unwritten-test, runner-observer, preparation, unpassable-step, not-ready, parse and lint refusals) still runs only when none of them refused. This change runs it on a submission that a cheap gate refused as well, so a draft that fails a cheap gate and the readiness check gets both refusals in one round. Readiness and the cheap gates are independent of each other; the model-backed contract review and wrong-implementation check need a ready package and stay sequential after them.

Nothing is removed. Every gate keeps its own problem text and gate name. The retry budget keeps counting by refuseSet: a submission refused by several gates is one counted refusal.

## Decisions
- When the cheap gates (shape/shape.go cheapGates) refused a submission, checkSubmission still runs readiness on it and holds the readiness refusal as one more refusal of that same submission, after the cheap gates' refusals and last in the combined feedback; the submission is then answered with the combined refusal. (constraint: The readiness refusal is held with the gate name and problem lines it has today: unwritten_tests, runner_observer, preparation, unpassable_step or readiness.) (constraint: The feedback text of every gate, including the readiness one, is unchanged; the texts are joined by a blank line as the cheap gates' are.) (evidence: shape/shape.go; shape/checkorder.go) (source: request)
- Readiness is not a model call, but the contract review and the wrong-implementation check are: they run only when no gate refused, as today, so a draft that fails any gate adds no model call beyond those the cheap gates already make. (decision: D2, round 1) (constraint: The decision, literal and example checks keep skipping when the review scope, step self-check or static checks refused.) (evidence: shape/shape.go; shape/checkorder.go) (source: request)
- When readiness runs on a submission a cheap gate refused and it does not refuse (it is ready, or it returns accepted red observations, or it fails for a reason that is not a refusal), the submission is answered with the cheap gates' combined refusal only; the observations are not acknowledged and no readiness failure replaces the cheap refusal. (decision: D3, round 1)  (evidence: shape/shape.go) (assumption: The request does not say what happens when only the cheap gates refuse; keeping today's feedback for that case is the smallest behavior.) (source: request)
- The retry budget is unchanged: all refusals of one submission are recorded together by recordRefusals as one refuseSet counted once, the first gate naming the refusal, and progress is judged by stillRefused as before. (decision: D4, round 1) (evidence: shape/retrybudget.go; shape/shape.go) (source: request)
- The regression test is a Go test in package shape built on the existing scripted fixtures (newShpFx, shpFiles, orderDecisionRefusal, orderLiteralRefusal): a draft with a decision record refusal and an undefined-step readiness refusal gets one refusal listing both. Acceptance compiles features/steps/shape-report-all-refusing-gates__driver/driver.go as a test file into a copy of the package through a go test overlay and reads what the shaper was told; the Build does not create any file under shape/ named shape_report_all_refusing_gates_observation_test.go. (decision: D5, round 1) (constraint: Existing tests that expected a refused submission to skip readiness are updated to the combined refusal; the test helpers keep their signatures.) (evidence: shape/batch_refusal_test.go; shape/helper_test.go; shape/shape_test.go) (source: request)
- Run readiness on a submission a cheap gate refused and hold its refusal last in the combined feedback, with gate names and texts unchanged. (decision: D1, round 1) (constraint: Contract review and wrong-implementation check stay sequential and run only when nothing refused) (evidence: shape/shape.go; shape/checkorder.go) (assumption: Probe not run; readiness running on cheap-refused drafts assumed safe.) (source: request)

## Source comparison
- Addressed, request Expected: D1 reports the cheap gates' and the readiness refusal in one round; scenario "A decision record refusal and a readiness refusal are reported together".
- Addressed, request Constraints: sequential model-backed checks stay after a ready package (D2); no model call is added (D2); every gate's problem text is unchanged (D1, scenario "A readiness refusal alone keeps its feedback"); the retry budget keeps counting by refuseSet (D4, scenario "Both refusals count as one submission and no contract review runs").
- Addressed, request regression test: the first Tests line.
- Context, not a threshold: the request observed that 9 of 14 real red Builds died in Shape on the 3 same-gate or 6 total refusal budget in shape/retrybudget.go.
- Omitted: no setting or flag; no change to Refine, to the retry limits or to any gate's text.

## Tests
- A draft failing a cheap gate and the readiness check gets both refusals in one round, unchanged texts, counted once, without a contract review: add a Go regression test in package shape using a decision record refusal plus an undefined-step report, and update existing tests that expected readiness to be skipped (regression)
- A readiness refusal alone is reported as before: the "readiness" case of the existing refusal-gate test and the acceptance scenario (existing)
- Surrounding shaping behavior is preserved: run go test ./shape (existing)

## Feasibility
- none
