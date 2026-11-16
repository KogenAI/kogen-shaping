# Drop false no-submit claim from budget stops

Slug: shape-budget-stop-drops-no-submit-line
Model: not applicable — this is refusal-copy filtering, not a change to shaping state or transitions
Acceptance:
- .kogen/intents/shape-budget-stop-drops-no-submit-line/acceptance.feature
- features/steps/shape-budget-stop-drops-no-submit-line__steps.js
- features/steps/shape-budget-stop-drops-no-submit-line__driver/run.js
- features/steps/shape-budget-stop-drops-no-submit-line__driver/observe_test.go.txt

Correct the misleading opening detail line of a shaping-budget refusal. When at least one submission was made, omit only `the shaper ended without calling submit`. When no submission was made, retain that line. Preserve the detail sentence, last refusal and earliest unresolved refusal blocks, optional tool-free-answer text, shaping-budget line and attempt lines, byte-for-byte and in their existing order apart from the omitted line. The no-tool-limit refusal is entirely unchanged, including its opening line even after refused submissions.

This is refusal text only: do not change the 15-minute budget, which drafts pass, events, submission counting, request scheduling or the no-tool-limit behavior. Keep implementation edits narrowly in budgetStop/noToolStop in shape/budget.go and shape/force.go as necessary, plus one focused model-free Go regression test next to shape/budget_test.go. Add that test as shape/budget_stop_test.go, using the existing scripted shaper helpers and clock, with cases for two refused submissions followed by budget exhaustion and for no submission followed by budget exhaustion. The first case must lack the misleading line and retain both attempt lines; the second must keep the line. Existing no-tool tests remain useful and unchanged. No new settings, dependencies, abstractions or compatibility forms are needed.

## Decisions
- Remove the no-submit opening line only from budget refusals after a submission; preserve every other refusal line and its order, and leave no-tool-limit refusals unchanged. (decision: D1, round 1) (evidence: shape/budget.go; shape/force.go) (source: request)
- Use the existing scripted Shape fixtures and clock for one focused model-free Go regression test and for independent acceptance observation through Shape. (decision: D2, round 1) (evidence: shape/budget_test.go; shape/force_wires_test.go; shape/helper_test.go) (source: request)
- Treat this as copy filtering over existing stop information, without changing the budget, draft acceptance, events or shaping state and without a Quint model. (decision: D3, round 1) (evidence: shape/budget.go; shape/force.go) (source: request)

## Source comparison
- Addressed: Request Expected and Constraints require conditional removal of the exact no-submit line, not a new diagnostic or a change to stopping logic. The outcome paragraphs and all acceptance cases preserve the detail, refusal blocks, optional answer, budget and ordered attempts; the retained-line scenario preserve the unchanged opening line.
- Addressed: Request's observed AoR climb release, cell path, stdout.txt quotation, shaping-events.ndjson evidence, h19 observation and three refused gates are retained as historical context in acceptance.feature. The executable reproduction deliberately uses the requested two cheap refused scripted submissions rather than accessing those external lab files or a live model.
- Addressed: Request asks for one focused model-free Go test beside shape/budget_test.go, using its helpers and clock with submitted and never-submitted budget cases. The implementation boundary above specifies that test; the supplied acceptance harness independently exercises the real Shape entry point and does not depend on a test the Build has yet to write.
- Addressed: Scope note permits budgetStop/noToolStop in shape/budget.go and shape/force.go plus the one test; no broader production work is authorized. controls.json contains one bounded correct overlay and one wrong overlay that drops the opening line for every stop, rejected by the combined retained-line scenario and independently by the no-tool observer control.
- Addressed: Intent slug, file paths, step prefixes and helper placement follow the request's Intent-to-write instructions. README.md and .kogen/project.yaml supply the Go and Cucumber conventions used here. The supplied testing-rules text governs testing; docs/testing-rules.md is absent in this checkout.
- Omitted: no requested behavior or settled non-secret input is deliberately omitted. Conflicting or superseded: only the misleading line on submitted budget stops is superseded; there is no change to the no-tool-limit wording or any events.

## Tests
- Submitted budget stops omit the misleading line and preserve detail and ordered refusal and attempt lines: add one model-free regression test with a two-refusal clocked case and compare the full structured Shape refusal in acceptance (regression)
- Never-submitted budget stops retain the opening line: cover the no-submission case in that same Go test and compare its full Shape refusal in acceptance (regression)
- Other refusal copy remains unchanged, including tool-free-answer detail and no-tool-limit output: compare complete observed detail and line arrays in the preserving acceptance cases and retain existing TestShapeForceNoToolBound coverage (regression)

## Feasibility
- The local toolchain reports Go 1.27.2 and Node v24.21.0; the existing focused TestBudgetEndsBeforeTheNextRequest passed. (probe: exit 0, 4.8 s)
- The initial observation fixture serialized before parallel subtests completed and produced an empty object; repaired it to serialize in cleanup after all scripted runs finish. The repaired baseline harness reached all four Shape stops and recorded the misleading line in both submitted budget cases, with the preserved cases intact. (probes: exit 0, 4.7 s and exit 0, 5.1 s)
- Generated the bounded correct and wrong implementation controls programmatically, without editing production checkout files. (probe: exit 0, 0.1 s)
- Final-byte observer controls accepted all four real correct-overlay observations and independently rejected both the never-submitted budget and no-tool-limit outputs of the all-stops wrong overlay. (probes: exit 0, 4.8 s and exit 1, 4.8 s)
- evidence: {"id":"budget-copy-observer","kind":"observer","inputs":[".kogen/intents/shape-budget-stop-drops-no-submit-line/acceptance.feature",".kogen/intents/shape-budget-stop-drops-no-submit-line/controls.json","features/steps/shape-budget-stop-drops-no-submit-line__driver/observe_test.go.txt","features/steps/shape-budget-stop-drops-no-submit-line__driver/run.js"],"observers":["features/steps/shape-budget-stop-drops-no-submit-line__driver/run.js","features/steps/shape-budget-stop-drops-no-submit-line__driver/observe_test.go.txt"],"checks":[{"name":"good","command":"node features/steps/shape-budget-stop-drops-no-submit-line__driver/run.js good","exit":0,"output":"observer accepted all scripted refusals\n"},{"name":"bad","command":"node features/steps/shape-budget-stop-drops-no-submit-line__driver/run.js bad","exit":1,"output":"observer rejected both retained-line regressions\n"}]}
