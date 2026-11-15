# Identify repeated refused decision statements

Slug: shape-repeated-decision-statement-says-so
Model: .kogen/intents/shape-repeated-decision-statement-says-so/model.qnt invariants=NotesOnlyForEarlierRefusals seed=17 samples=100 steps=12
Acceptance:
- .kogen/intents/shape-repeated-decision-statement-says-so/acceptance.feature
- features/steps/shape-repeated-decision-statement-says-so__steps.js
- features/steps/shape-repeated-decision-statement-says-so__driver/observe_test.go
- features/steps/shape-repeated-decision-statement-says-so__driver/qualify.js

Make a repeated decision-check refusal recognizable without changing its verdict or any existing feedback line. In shape/decision.go, when a currently refused statement equals a statement refused in an earlier submission of this shaping, append exactly one problem line immediately after that statement's existing `fix for` line:

`note: this statement is unchanged since submission <n> and was refused for the same reason; change the statement itself, not the text around it`

Replace <n> with the earlier submission number. Match the statement text, not the intent line number, the whole draft, surrounding prose, or model explanation. Use exactly the existing whitespace normalization used to locate the statement in the intent; all other bytes remain significant. Track only statements actually refused, not passed, advisory, or automatically marked assumptions. A first refusal and a changed statement retain today's output exactly. A later passing verdict remains passing and receives no refusal note. Retain earlier refusal history across intervening submissions within the active shaping run; it is not a new persisted cache or setting.

Do not skip checks, change model consultation, alter verdicts, quotes, fixes, or any existing output line, or optimize away the static/model check runs mentioned in the observation. This adds feedback, not a new refusal rule, policy prompt, retry policy, compatibility form, or user option. Keep existing behavior tests and add the focused model-free scripted regression next to the shape package's decision-check tests, including the requested first-versus-second refusal and changed-statement counterexample.

## Decisions
- Use the existing statement-location whitespace normalization, not case folding, punctuation removal, value-notation normalization, or draft equality. (evidence: shape/decisions.go; shape/decision.go) (source: request)
- Name the earliest earlier submission that refused the normalized statement within the active shaping run. (decision: D1, round 1) (constraint: Track only actual refusals; preserve the first refusal number) (evidence: shape/shape.go; shape/decision.go) (assumption: When several earlier submissions refused the same statement, use the earliest one's submission number) (source: request)
- Keep the scripted acceptance independent of the implementation under change and require a focused permanent regression in the shape package. (evidence: shape/helper_test.go; shape/decision_test.go; .kogen/project.yaml) (source: request)

## Source comparison
- Addressed: Request Expected and Constraints map to the exact additive note above and acceptance.feature's repeated/changed real-submission scenarios, full statement/quote/verdict/fix comparisons, whitespace case, submission-number gap, and nonadjacent recurrence. Existing wording is not replaced; only repeated refusals gain the note.
- Addressed: Request regression-test paragraph maps to the required model-free shape-package regression and the owned scripted-provider overlay driver. The driver invokes the real Shape submission flow for the successive-submission examples and the real decision-check entry point for focused history controls; it does not require the Build to invent a named test before acceptance can run.
- Addressed: Request Observed describes 12 of 78 refusals from 4fb52c19 to 30cf6619, chiefly rails-ft-webhook-delivery-trust-and-restart and “Deliver exactly once.” The supplied h7/h8/h10 cell names and attempt transitions are retained in acceptance.feature as historical provenance, with executable local reproductions of the repeated statement, moved line, and changed-statement counterexample. These are synthetic scripted verdicts, not a claim of access to the original cells' run/cells/*/r*/*/shaping-events.ndjson.
- Addressed: the reported full static and model runs remain unchanged; no requested pass/refuse rule or consultation is superseded. The requested slug and artifact/step prefixes are used throughout the complete package. Omitted: no requested behavior; historical private run artifacts are not recreated or required as runtime inputs.

## Tests
- Repeated refused statement gains the exact note after its fix while the first refusal is unchanged: add a model-free scripted regression alongside shape/decision_test.go and exercise two real submissions in acceptance (regression)
- Changed statement receives no note and retains existing refusal text: exercise the changed second submission and assert original feedback (regression)
- Statement identity uses existing whitespace normalization and earlier submission numbers rather than line/check numbers: exercise a whitespace-only variant with a gap in submission numbers (regression)
- Earlier refusal remains recognizable across an intervening different statement: exercise nonadjacent recurrence naming the earliest refusal (regression)
- Feedback does not change gate verdicts or model consultation: assert one scripted checker call per candidate, unchanged quotes/verdict/reason, no history note in checker input, and both passing/refused transitions (integration)

## Feasibility
- The local Quint model typechecked and ran 100 samples of 12 steps with seed 17 without an invariant violation; it abstracts normalized statements as two identities and distinguishes passed from refused history. (probe: exit 0)
- evidence: {"id":"scripted-refusal-observer","kind":"observer","inputs":[".kogen/intents/shape-repeated-decision-statement-says-so/controls.json",".kogen/intents/shape-repeated-decision-statement-says-so/acceptance.feature"],"observers":["features/steps/shape-repeated-decision-statement-says-so__steps.js","features/steps/shape-repeated-decision-statement-says-so__driver/observe_test.go","features/steps/shape-repeated-decision-statement-says-so__driver/qualify.js"],"checks":[{"name":"good","command":"\"$KOGEN_NODE\" features/steps/shape-repeated-decision-statement-says-so__driver/qualify.js good","exit":0,"output":"observer accepted real correct implementation\n"},{"name":"bad","command":"\"$KOGEN_NODE\" features/steps/shape-repeated-decision-statement-says-so__driver/qualify.js bad","exit":1,"output":"observer rejected corruptions\n"}]}
