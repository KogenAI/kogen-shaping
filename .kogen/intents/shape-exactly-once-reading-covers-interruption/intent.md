# Refuse weakened guarantees in decision check

Slug: shape-exactly-once-reading-covers-interruption
Model: not applicable — one fixed instruction text and one model-free Go test, with no state or interacting rules
Acceptance:
- .kogen/intents/shape-exactly-once-reading-covers-interruption/acceptance.feature
- features/steps/shape-exactly-once-reading-covers-interruption__steps.js
- features/steps/shape-exactly-once-reading-covers-interruption__harness.js

The decision check of Shape (the `decisionInstructions` constant in `shape/decision.go`) lets a Decision that restates an absolute guarantee of the request (for example "exactly once") as only its at-most-once, at-least-once or best-effort half pass when the Decision carries an assumption marker, and the shaped scenarios then pin the weaker behaviour. After this change the instructions tell the checking model to give `contradicts` for such a Decision, or for a concrete result in prose or examples that does the same, even when the Decision carries an assumption marker, with `intent_value` the weaker reading and `request_value` the quoted guarantee. A Decision that keeps the full guarantee, or that states how the interrupted case is covered so that the item is neither lost nor repeated, is not affected. When the full guarantee is not achievable, the Intent must state that limit as an open question for the owner instead of silently picking the weak half.

The change is one added paragraph in `decisionInstructions` (placed before the final "Answer by calling" sentence) and one model-free Go test in the shape package. No new gate, tool, configuration or setting; the refusal text the existing mechanics produce for `contradicts` (which already names both values) is reused unchanged. Which drafts pass stays unchanged except drafts with a weakened-guarantee Decision. The instructions gain no task-specific wording.

The added paragraph contains these literals, which acceptance checks: `at-most-once`, `at-least-once`, `best-effort`, `even when the Decision carries an assumption marker`, `keeps the full guarantee` and `open question for the owner`. It must not contain the words `webhook`, `delivery` or `restart`.

Build notes: the acceptance harness is compiled into the shape package's tests as `shape/ske_acceptance_test.go` through a Go overlay (that path never exists in the repository) and defines `TestSkeAcceptance`; do not create that path or that test name. The Build's own regression test goes in a new `_test.go` file next to `shape/decision_test.go`, with a name that does not start with `TestSke`.

## Decisions
- The weakened-guarantee rule is added to `decisionInstructions` in `shape/decision.go` as one paragraph before "Answer by calling"; the verdict for a weakened guarantee is the existing `contradicts` with the existing `intent_value`, `request_value` and refusal text, so no new gate, tool, configuration or wording of the refusal is added. (evidence: shape/decision.go; shape/decision_test.go) (source: request)
- An assumption marker does not exempt a weakened guarantee: the rule applies even when the Decision carries an assumption marker, extending the existing rule that an assumption does not override an explicit request conflict. (evidence: shape/decision.go) (source: request)
- A Decision that keeps the full guarantee, or states how the interrupted case is covered so the item is neither lost nor repeated, is unaffected; when the full guarantee is not achievable the Intent states the limit as an open question for the owner. (source: request)
- The instructions use no task-specific terms (no webhook, delivery or restart wording); the guarantee examples are generic ("exactly once", "never", "always", "nothing left behind", "every"). (source: request)
- The regression test is model-free and scripted: it feeds scripted decision-check items to the existing decision outcome helpers in `shape/decision.go` (as `shape/decision_test.go` does) and checks the instruction text; one wrong-implementation control, an implementation that still accepts the assumption-marked weakened Decision, is enough. (evidence: shape/decision_test.go) (source: request)
- Acceptance observes the real instruction text and the real outcome rules through a Go overlay harness, because the model's own verdicts cannot be run model-free; the literals pinned in the prose above are the minimum wording Build must use. (assumption: pinning these literals is an acceptable way to observe a prompt-text rule without a model) (source: request)
- Add the weakened-guarantee rule as one paragraph to decisionInstructions in shape/decision.go, reusing the contradicts verdict and existing refusal text (decision: D1, round 1) (constraint: no new gate, tool or configuration) (constraint: no task-specific wording) (evidence: shape/decision.go; shape/decision_test.go) (assumption: Prompt-text rule cannot be run model-free; acceptance pins literals only.) (source: request)
- An assumption marker does not exempt a weakened absolute guarantee (decision: D2, round 1) (constraint: rule applies even when the Decision carries an assumption marker) (evidence: shape/decision.go) (source: request)
- Acceptance uses a Go overlay harness observing the real instruction text and outcome rules (decision: D3, round 1) (evidence: shape/decision_test.go) (assumption: Pinning literals is an acceptable proxy for a prompt rule.) (source: request)

## Source comparison
- Addressed: the decision check treats a Decision or concrete result restating an absolute guarantee as only its weaker half as `contradicts`, even with an assumption marker, naming both values. Scenario 1 (instruction literals plus the scripted contradicts refusal naming `at-most-once on a crash` and `exactly once`).
- Addressed: a Decision keeping the full guarantee, or covering the interrupted case, still passes. Scenario 2.
- Addressed: when the full guarantee is unachievable the Intent states an open question for the owner; no task-specific wording in the instructions. Scenario 3.
- Addressed: focused model-free Go test next to the decision tests, with one wrong-implementation control that still accepts the assumption-marked weakened Decision. Tests section and controls.json.
- Preserved: no new gate, tool or configuration; the other decision-check rules and which drafts pass are unchanged.
- Context only, not behaviour to build: the request's observation text is carried verbatim as "): in all 22 landed cells of hypotheses h20-h27 whose Intent could be read, the Intent defines exactly once as " a pinned weaker reading, and ", and its acceptance scenario " then asserts that weaker reading; this change makes the decision check refuse that pattern.

## Tests
- A weakened-guarantee Decision with an assumption marker is refused with both values named: a Go test in the shape package, next to `shape/decision_test.go`, that scripts a `contradicts` item (intent_value `at-most-once on a crash`, request_value `exactly once`) through the decision outcome helpers and checks the refusal names both (unit)
- A Decision keeping the full guarantee still passes: the same Go test scripts a `follows` item and checks no refusal (unit)
- The instructions carry the weakened-guarantee rule: the same Go test checks `decisionInstructions` contains the rule (unit)
- Instruction text, scripted refusal and unaffected pass observed end to end: acceptance.feature scenarios 1 to 3 (scenario)

## Feasibility
- none
