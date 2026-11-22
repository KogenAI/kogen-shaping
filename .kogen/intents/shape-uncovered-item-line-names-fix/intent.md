# Add a fix line to uncovered-item refusals

Slug: shape-uncovered-item-line-names-fix
Model: not applicable — pure functions from review results to refusal lines, with no state
Acceptance:
- .kogen/intents/shape-uncovered-item-line-names-fix/acceptance.feature
- features/steps/shape-uncovered-item-line-names-fix__steps.js
- features/steps/shape-uncovered-item-line-names-fix__driver/harness_test.go.txt

When contract review finds a request item that no scenario or control covers, the refusal line says only what is missing and no fix line follows it, while every other kind of contract-review problem is followed by a line starting with `- fix for`. Two problem kinds change: `no wrong implementation in controls.json names a scenario of it` (controls judging, shape/controls.go) and `no scenario exercises it` (property check, shape/property.go). Each such problem line is followed by exactly one line `- fix for property <item>: ...`, where <item> is the item's name as the problem line shows it, in the order of the problem lines. The controls fix quotes the scenarios the review declared for the item (at most 3, then ` and N more`) and says to add a wrong implementation to controls.json whose scenario is one of them and whose step is the Then step that fails when the behaviour is broken; for a kept item it adds `; the wrong implementation must break the kept behaviour`; when no scenario is declared it says to add a scenario for it to acceptance.feature first and quotes none. The property fix says to add a scenario whose Then step observes the item, or, when no automatic step can observe it, to record in a Decision that it needs human evaluation. Which drafts pass, and every existing refusal line, stay exactly as they are; the only change is the added fix line. No setting, flag or output is added and the line cut rules are unchanged.

## Decisions
- The controls fix line is `- fix for property <item>: add a wrong implementation to controls.json whose scenario is <"A"> (or one of <"A", "B", "C" and N more>) and whose step is the Then step that fails when the behaviour is broken`, followed for a kept item only by `; the wrong implementation must break the kept behaviour`; with no declared scenario it is `- fix for property <item>: add a scenario for it to acceptance.feature, then add a wrong implementation to controls.json whose scenario is that scenario and whose step is the Then step that fails when the behaviour is broken`. It follows the problem line `no wrong implementation in controls.json names a scenario of it` only. (decision: D1, round 1) (constraint: no new setting, flag or environment variable) (evidence: shape/controls.go; shape/audit.go) (assumption: the item in the fix line is the property name the problem line shows, and the exact wording of the fix text is the Shaper's choice) (source: request)
- The property fix line is `- fix for property <item>: add a scenario whose Then step observes it, or, when no automatic step can observe it, record in a Decision that it needs human evaluation`, one after each confirmed `no scenario exercises it` item line. (decision: D2, round 1) (evidence: shape/property.go) (assumption: the exact wording of the fix text is the Shaper's choice) (source: request)
- Which drafts pass, and every existing refusal line, stay exactly as they are; the only change is the added fix line. Fix lines are separate lines after their problem line, in the order of the problem lines. (decision: D3, round 1) (constraint: line cutting is out of scope and unchanged) (evidence: shape/controls.go; shape/property.go; shape/checkorder.go) (source: request)
- An existing headline states the same number of problems as before, so the added fix lines are not counted in it; the property headline is built in shape/contract.go from the number of property lines, beside the named files. (decision: D4, round 1) (evidence: shape/contract.go; shape/controls.go) (assumption: a one-line edit in shape/contract.go to keep the property headline count is accepted, because the named files alone cannot keep that count) (source: request)

## Source comparison
- Kept item with a scenario but no wrong implementation naming it gets a fix line quoting the scenario and containing controls.json: addressed by scenario "A kept item whose scenario no wrong implementation names gets a fix line".
- The kept fix says the wrong implementation must break the kept behaviour, the new one does not: addressed by that scenario and "A new item gets a fix line without the kept sentence".
- Item with no declared scenario gets a fix that says to add a scenario and quotes none: addressed by "An item with no declared scenario is told to add a scenario"; five declared scenarios quote three and the count: "Many declared scenarios are quoted three and then counted".
- Uncovered item with no scenario gets a fix naming a Then step and a Decision for human evaluation: addressed by "An item no scenario exercises gets a fix line".
- Covered items get no fix line, byte for byte as before: addressed by "A covered item gets no fix line" and "Another problem kind gets no fix line".
- Out of scope and untouched: shape/prompts, docs, README, the other refusal kinds, the line cut rules.

## Tests
- A controls problem line of an unnamed item is followed by its fix line, kept and new, with one, no and many declared scenarios: update the exact-line expectations in shape/controls_test.go and add a focused test of the fix texts (unit)
- An uncovered property problem is followed by its fix line and other reasons are not: update the expectations of applyPropertyVerdicts in shape/property_test.go and add a focused test (unit)
- The headlines state the number of problems, not of lines: assert in shape/controls_test.go and shape/property_test.go that a refusal with fix lines states the number of problems (unit)
- Covered items and other reasons produce unchanged lines: the scenarios "A covered item gets no fix line" and "Another problem kind gets no fix line" (scenario)

## Feasibility
- none
