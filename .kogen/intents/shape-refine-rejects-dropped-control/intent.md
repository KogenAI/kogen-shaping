# Refuse a Refine controls file that drops a control

Slug: shape-refine-rejects-dropped-control
Model: not applicable — a stateless comparison of two controls files within one Refine check
Acceptance:
- .kogen/intents/shape-refine-rejects-dropped-control/acceptance.feature
- features/steps/shape-refine-rejects-dropped-control__steps.js

When Refine receives a replacement `controls.json`, every wrong-implementation control of the draft's stored `controls.json` (identified by its `name` and `scenario`) must be present in the replacement, or be explicitly superseded in the replacement with a stated reason that names the original control. Otherwise Refine rejects the replacement with a refusal that names each missing control and says what is needed to supersede it, and nothing is retained: the draft's files, hash and approval stay as they were. Today `controlsEditCheck` (shape/refine.go) judges only the controls listed in the replacement and never compares them with the stored original, so a replacement that drops a control passes.

A supersede with an empty reason, or one that names a control that was never in the stored original, is rejected; a supersede with a blank reason does not count, so the original it names is missing. A superseding control is itself judged like any other control of the replacement.

Out of scope and unchanged: how a single control is judged, the cap on controls, the contract review, and a replacement whose stored original is deferred or holds no readable wrong implementation (nothing to compare). Only shape/refine.go changes, plus new model-free test files in shape/.

## Decisions
- Every original control, by name and scenario, must be present in a replacement controls file or be superseded in it with a stated reason naming the original control. (source: request)
- Refine refuses a replacement with a missing control, naming each missing control and what is needed to supersede it, and retains nothing. (source: request)
- A superseded control with an empty reason, or naming a control never in the original, is rejected. (source: request)
- The controls file schema accepts exactly the fields correct and wrong and only shape/refine.go may change, so a supersede is stated in the name of a replacement control as the text `supersedes "<original name>" of "<original scenario>" because <reason>`, and the refusal text for a missing control contains the control's name, its scenario, the word `supersedes` and the word `because`. (evidence: shape/controls.go) (assumption: the supersede statement is written in a replacement control's name in this form) (source: request)
- A deferred replacement of a stored original that has wrong implementations drops all of them and cannot carry a supersede statement, so it is refused naming each. (assumption: deferring controls does not count as superseding them) (source: request)
- Names and scenarios are compared after whitespace normalisation, as the controls parser does. (evidence: shape/controls.go) (source: request)
- A replacement that keeps every original control is judged exactly as today, including the wrong-implementation run that refuses a retained control that stops catching its defect. (evidence: shape/refine.go; shape/refine_test.go) (source: request)

## Source comparison
- Addressed: scenario 1, a replacement holding one of two controls is refused naming the dropped one (`drop` journey).
- Addressed: scenario 2, the dropped control superseded with a non-empty reason is accepted (`supersede` journey).
- Addressed: scenario 3, a blank reason and an unknown original are refused (`noreason`, `unknown` journeys).
- Addressed: scenario 4, both controls kept and one changed is judged as today, and a weakened retained control is still refused by the wrong-implementation check (`keep`, `keepweak` journeys; the existing test TestRefineRefusesAControlsReplacementWhoseWrongImplementationPasses stays).
- Interpretation: the `supersede` journey keeps two controls in total (one kept, one superseding), so it does not distinguish refusing every replacement with fewer controls; the Build's own unit test of the comparison covers a smaller replacement.
- Omitted: the request's named tests are not run by acceptance because Build must write them; the journeys observe Refine's behavior instead, with the real Cucumber runner on the fixture draft.
- Release evidence: TestRefineRejectsReplacementDroppingControl fails on main and passes after the change.

## Tests
- A replacement that drops an original control is refused, naming it: TestRefineRejectsReplacementDroppingControl (integration)
- A dropped control superseded with a non-empty reason is accepted: TestRefineAcceptsSupersededControlWithReason (integration)
- A supersede with a blank reason, or naming a control not in the original, is refused: TestRefineRejectsSupersedeWithoutReason (unit)
- A replacement that keeps both controls is judged as before: TestRefineKeepsBothControlsJudgedAsBefore (integration) and the existing TestRefineRefusesAControlsReplacementWhoseWrongImplementationPasses (existing)

## Feasibility
- none
