# Support evidenced keep-existing Decisions

Slug: shape-keep-existing-decisions-supported
Model: not applicable — stateless classification of one Decision against repository bytes, no workflow state
Acceptance:
- .kogen/intents/shape-keep-existing-decisions-supported/acceptance.feature
- features/steps/shape-keep-existing-decisions-supported__steps.js
- features/steps/shape-keep-existing-decisions-supported__driver/probe_test.go

Shape's decision check must support a Decision that keeps existing values or behaviour when the repository evidence it cites shows those values, without requiring a request quote, even when the checking model answered unsupported. A keep-existing Decision whose cited path is missing, or whose cited file bytes do not show the stated values, stays refused as unsupported. Every other Decision is classified exactly as today. This closes the 8 Nov observation that 6 of 9 intent/not_ready failures were Shape's own gates, among them decision_check refusing `controlsMaxRuns (8), controlsMaxFiles (8), controlsMaxBytes (65536) and controlsTimeCap keep their values` because the request quotes only the 300 s cap (Studio issue kogen/ed7490a5-37ca-4fc0-b7c6-30664f7efcbd).

The checker policy text and the evidence supplier already exist (`decisionInstructions` and `decisionEvidence`); the model's verdict alone is not reliable, so the gate decides mechanically. In `decisionResolveNotes` (or the code that calls it), classify an item whose model verdict is unsupported as follows. A statement is keep-existing when its text, outside the `(evidence: ...)` marker and trailing tags, contains one of the words keep, keeps, kept, unchanged, preserve, preserves, retain or retains (case-insensitive, whole word) and its marker cites at least one file path. Its values are the backtick-quoted spans and the decimal numbers (digit runs not touching letters or underscores) of that same text. The values are shown when every cited path is a tracked file at HEAD and every value occurs as a substring of the combined HEAD bytes of the cited files; a statement with no values is never shown. Shown: the item's verdict becomes follows, with no refusal and no advisory note. Not shown: the item is refused as unsupported (feedback line `statement: ... | verdict: unsupported | why: ...`) and does not take the existing advisory route for evidence-marked statements. The event item records the final verdict. No setting, flag or option is added.

Add focused model-free regression coverage in the shape package using the existing scripted provider: a keep-existing Decision with valid evidence passes, the missing-path and values-not-shown cases are refused, and a planted unsupported new Decision with no evidence is still refused. Keep all existing decision-check tests passing.

## Decisions
- Decide keep-existing support mechanically from cited HEAD bytes in addition to the model verdict, because the model answered unsupported on such Decisions while repository evidence existed. (evidence: shape/decision.go; shape/decision_evidence.go) (source: request)
- Leave the checker policy text, the evidence supplier and all non-keep-existing classification unchanged. (evidence: shape/decision.go; shape/decision_evidence.go) (source: request)
- Use the same local scripted-provider approach as the earlier keep-unchanged Intent: a compiled test overlay in the shape package whose scripted checker returns unsupported, so only Kogen's mechanical rule can change the outcome. (evidence: features/steps/shape-decision-check-keep-unchanged__driver/probe_test.go; features/steps/shape-decision-check-keep-unchanged__steps.js) (source: request)
- Keep-existing detection words and value extraction are the fixed rule above, chosen as the simplest deterministic reading of the request. (assumption: the user confirms the keyword list and that the values must literally occur in the cited file bytes) (source: request)

## Source comparison
- Addressed: a keep-existing Decision citing an existing repository path that shows the value is supported without a request quote (scenarios 1).
- Addressed: a missing cited path or bytes that do not show the value stay refused (scenarios 2 and 3).
- Addressed: unsupported new Decisions are still refused and every other Decision is unchanged (scenario 4, existing tests).
- Addressed: the controlsMaxRuns (8), controlsMaxFiles (8), controlsMaxBytes (65536), controlsTimeCap and 300 s literals are in the executable fixture. The request's model-free regression tests are the acceptance driver plus focused shape tests.

## Tests
- Keep-existing Decision with valid evidence is supported without a quote: scripted unsupported checker, observe follows and no feedback (regression)
- Missing cited path stays refused: observe unsupported feedback (regression)
- Cited file not showing the value stays refused: observe unsupported feedback (regression)
- Planted new Decision without evidence stays refused: observe unsupported feedback (regression)
- Existing decision-check behaviour unchanged: existing shape decision tests (existing)

## Feasibility
- none
