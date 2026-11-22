# Resolve a controls scenario field with its keyword

Slug: shape-controls-scenario-keyword
Model: not applicable — one name lookup in the controls.json parser, with no state or interacting rules
Acceptance:
- .kogen/intents/shape-controls-scenario-keyword/acceptance.feature
- features/steps/shape-controls-scenario-keyword__steps.js
- features/steps/shape-controls-scenario-keyword__harness.js

When Shape parses a controls.json wrong implementation, `controlsScenarios` in `shape/controls.go` matches the scenario field only against the plain name of a scenario or of an outline. A field that repeats the Gherkin keyword of the scenario line, such as `Scenario: Add keeps working` or `Scenario Outline: Mul <a> times <b>`, is refused as "is not a scenario of acceptance.feature" although it names a real scenario, and the shaper spends another round. In a recorded attempt all 7 wrong implementations were refused this way, each naming a scenario with the keyword in front, for example `Scenario: A chapter 09 text without the intent block is rejected naming the page` and, for an outline, `Scenario Outline: One extra space in a chapter 09 page is rejected for every page`; the feature had those scenarios under the plain names.

After this change a scenario field is resolved by its plain name first, exactly as before. Only when no scenario or outline has that name, one leading keyword of `Scenario:`, `Scenario Outline:` or `Scenario Template:` followed by whitespace is removed, once, and the rest is looked up as a plain name. The parsed control keeps the plain name, so the witness choice, the run and every later message treat it as if the plain name had been written. A name that matches nothing even after the removal is refused with the same line as today, naming the field as written. The step field rules do not change (a step without its keyword is still refused). Nothing is removed and no setting is added.

For a wrong implementation named n1 whose scenario field is `Missing`, the refusal line is exactly:

`- controls.json: wrong implementation "n1": scenario "Missing" is not a scenario of acceptance.feature`

and for the field `Scenario: Missing` the same line names `Scenario: Missing`.

## Decisions
- The scenario field is looked up as a plain name first; the keyword is removed only when that finds no scenario or outline, so a scenario whose own name starts with the keyword text still resolves to itself. (decision: D1, round 1) (evidence: shape/controls.go; shape/controls_test.go) (source: request)
- The keyword is removed once: `Scenario:`, `Scenario Outline:` or `Scenario Template:` at the start, followed by whitespace; a field with the keyword twice is refused as not a scenario of acceptance.feature. (decision: D2, round 1) (evidence: shape/controls.go) (source: request)
- The parsed control stores the plain name, so the witness choice returns every example of an outline for the outline form. (decision: D3, round 1) (evidence: shape/controls.go) (source: request)
- A name that matches nothing after the removal is refused with the line `- controls.json: wrong implementation "n1": scenario "Missing" is not a scenario of acceptance.feature` (for the field `Missing`; the quoted scenario is the field as written); no fallback to any scenario. (decision: D4, round 1) (evidence: shape/controls.go; shape/controls_test.go) (source: request)

## Source comparison
- Addressed: "a scenario field is resolved by its plain name first, exactly as today"; scenarios 4 and the plain cases of scenarios 1 and 2.
- Addressed: removal of one leading `Scenario:`, `Scenario Outline:` or `Scenario Template:`; scenarios 1 and 2 (the Template form is observed in scenario 2). The recorded refused fields `Scenario: A chapter 09 text without the intent block is rejected naming the page` and `Scenario Outline: One extra space in a chapter 09 page is rejected for every page` are of the kind that now resolve to their plain names.
- Addressed: unknown names refused with the same line, with and without a keyword: `- controls.json: wrong implementation "n1": scenario "Missing" is not a scenario of acceptance.feature`; scenario 3.
- Addressed: keyword text that is part of a scenario's own name; scenario 4. Keyword twice; scenario 5.
- Out of scope and unchanged: shape/prompts, docs, README, the step field rules, every other package.

## Tests
- A scenario field with a keyword parses to the plain name, outline and template forms give every example, unknown names are refused, keyword-named scenarios resolve first, the keyword is removed once: new cases in `TestParseControls` and a witness case in `shape/controls_test.go` (unit)
- The same rules through the real parser and witness choice: acceptance.feature scenarios 1 to 5 (scenario)
- Existing parse cases keep passing: `TestParseControls` (existing)

## Feasibility
- none
