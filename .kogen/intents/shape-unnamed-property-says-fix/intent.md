# Say what to add for an unnamed property

Slug: shape-unnamed-property-says-fix
Model: not applicable — a pure function from controls results to refusal lines, with no state
Acceptance:
- .kogen/intents/shape-unnamed-property-says-fix/acceptance.feature
- features/steps/shape-unnamed-property-says-fix__steps.js
- features/steps/shape-unnamed-property-says-fix__driver/harness_test.go.txt

When the judging of controls.json finds a property that no wrong implementation names a scenario of, the refusal line ends with the empty-looking reason `no wrong implementation in controls.json names a scenario of it`, and the shaper resubmits the same controls. The reason stays as it is and now gets one sentence after a full stop and a space that says what to add. For a kept property the sentence is `Add a wrong implementation to controls.json that breaks this unchanged behaviour and names a scenario of it and the step line that must reject it.` For a new property it is `Add a wrong implementation to controls.json that gets this property wrong and names a scenario of it and the step line that must reject it.` Every other reason (a wrong implementation whose run did not report every scenario, or that is not rejected at its step) is unchanged and gets no such sentence, and a property that a wrong implementation rejects gets no line. The change is confined to shape/controls.go and shape/controls_test.go; it adds no setting, flag or output, and the existing 300-character cut of recorded refusal lines (which keeps a kept line's problem whole by shortening the request text) is unchanged.

## Decisions
- The reason for a property that no wrong implementation names stays `no wrong implementation in controls.json names a scenario of it` and gets one sentence after a full stop and a space: for a kept property `Add a wrong implementation to controls.json that breaks this unchanged behaviour and names a scenario of it and the step line that must reject it.`, for a new property `Add a wrong implementation to controls.json that gets this property wrong and names a scenario of it and the step line that must reject it.` (decision: D1, round 1) (constraint: scope is shape/controls.go and shape/controls_test.go only) (constraint: no new setting, flag or environment variable) (evidence: shape/controls.go; shape/controls_test.go) (source: request)
- Every other reason of the judging (a wrong implementation whose run did not report every scenario as passed or failed, or that is not rejected at its step) is unchanged and gets no such sentence; a property that a wrong implementation rejects at its step gets no line. (decision: D2, round 1) (evidence: shape/controls.go) (source: request)
- The sentence is added where the reason is chosen in controlsJudgeAll, so the existing cut in shape/checkorder.go, which shortens a kept line's request text and keeps the problem whole, then preserves the whole kept sentence; shape/checkorder.go is not edited. (decision: D3, round 1) (evidence: shape/controls.go; shape/checkorder.go) (assumption: the existing cutRequestText behaviour at HEAD is what keeps the sentence whole; a scratch run on a copy of the checkout with the correct implementation showed the exact lines of every acceptance scenario) (source: request)

## Source comparison
- Kept property with no wrong implementation gets the exact line with the kept sentence: addressed by scenario "A kept property with no wrong implementation says what to add"; its wrong implementation drops the sentence.
- New property that no wrong implementation names gets the new sentence: addressed by scenario "A new property that no wrong implementation names says what to add"; its wrong implementation uses the kept sentence for it.
- Reasons for a wrong run that did not report every scenario, or that fails at another step, stay unchanged with no sentence: addressed by scenarios "A wrong run that did not report every scenario keeps its own reason" and "A wrong implementation rejected at another step keeps its own reason"; the wrong implementation appends the sentence to every reason.
- A property that a wrong implementation rejects at its step gets no line: addressed by scenario "A property that a wrong implementation rejects at its step gets no line"; its wrong implementation adds a line for a satisfied property.
- A long kept request text is cut to exactly 300 characters and the whole kept sentence stays at the end: addressed by scenario "A long kept request text is cut and the whole sentence stays"; its wrong implementation cuts the end of the line.
- Out of scope and untouched: shape/prompts, docs, README, shape/checkorder.go, the other reasons of controls judging, which scenarios count as naming a property.

## Tests
- A kept property no wrong implementation names gets the reason and the kept sentence, and a new one the new sentence: update the two expectations in shape/controls_test.go (TestControlsJudgeAll and TestWrongImplementationRealRuns "no wrong implementation") to the exact lines (unit)
- Other reasons carry no sentence and a rejected property gets no line: the existing incomplete-run and real-run expectations in shape/controls_test.go stay unchanged (existing)
- A long kept line cut to 300 characters ends with the whole kept sentence: the scenario "A long kept request text is cut and the whole sentence stays", through controlsJudgeAll and the cut that records refusals (scenario)

## Feasibility
- none
