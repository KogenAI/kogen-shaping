# Require kept_steps in strict tool schemas

Slug: shape-strict-tool-schemas
Model: not applicable — the change is a static tool schema and a test over it, with no state
Acceptance:
- .kogen/intents/shape-strict-tool-schemas/acceptance.feature
- features/steps/shape-strict-tool-schemas__steps.js
- features/steps/shape-strict-tool-schemas__driver/observe_test.go.txt

Every ChatGPT-routed Refine fails because OpenAI refuses the strict `submit_contract_review` tool: the property report item schema (`/properties/properties/items`) declares `kept_steps` but its `required` list omits it. Every tool Shape defines with `strict: true` must have a parameter schema valid under OpenAI's strict rules: in every object schema at any depth, every declared property is listed in `required` and `additionalProperties` is `false`.

`kept_steps` becomes a required field of each property report. A property with no kept steps sends an empty list, and an empty list means exactly what an absent field means today. Reports that still omit `kept_steps` (older recorded fixtures) are still accepted as before. Tools defined with `strict: false` are out of scope and unchanged. A focused Go test in package `shape` walks the parameter schema of every tool Shape defines with `strict: true` and fails when any object schema at any depth declares a property that is not in its `required` list or lacks `additionalProperties: false`; the failure message names the offending property or the missing `additionalProperties`. The name of that test contains "Strict" or "Schema".

## Decisions
- `kept_steps` is added to the `required` list of the property report schema in `shape/property.go`, and the parser keeps accepting reports without it; both an absent field and an empty list yield no kept steps. (decision: D1, round 1) (evidence: shape/property.go; shape/contract.go) (source: request)
- The walking test lists the strict tools by their constructors (literal, question, decision, contract review, contract confirmation and the example-check tool built from `exampleSchema()`) and walks `properties`, `items` and `anyOf`/`oneOf`/`allOf` at any depth. (decision: D2, round 1) (evidence: shape/literal.go; shape/question.go; shape/decision.go; shape/contract.go; shape/examples.go) (assumption: the Build must add any further strict tool it finds to the same test) (source: request)
- Acceptance finds the Build's walking test by a top-level test name containing strict or schema (case-insensitive) and does not require any other name, because the Build chooses the test's name. (decision: D3, round 1) (evidence: shape/kept_baseline_test.go) (assumption: the test name will contain Strict or Schema) (source: request)

## Source comparison
- `kept_steps` required in each property report; empty list equals absent field: addressed by the property schema change and scenario "kept_steps is required in a property report and older reports without it are still accepted".
- Reports omitting `kept_steps` stay accepted: addressed by the same scenario (omitted, empty and non-empty lists).
- Strict-false tools unchanged: addressed by leaving shape/policy.go untouched; no acceptance step touches them.
- Focused Go test over every strict tool schema: addressed by the scenarios "Every strict tool's parameter schema is valid under the strict rules" and the two walking-test rejection scenarios, which run the named-by-pattern test and do not use git diff or untracked-file listings.

## Tests
- Every strict tool schema declares every property in required and has additionalProperties false: a Go test in package shape walking all strict tool schemas (unit)
- kept_steps is required in the property report and reports without it are still accepted: observer test plus existing parser behaviour (unit)
- The walking test fails on a nested property missing from required and on a missing additionalProperties false: scenarios with scratch schema mutations (scenario)

## Feasibility
- none
