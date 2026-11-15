# Show the step failure in runner-observer refusals

Slug: shape-runner-observer-refusal-shows-step-failure
Model: not applicable — one pure formatting change of a refusal line with no state
Acceptance:
- .kogen/intents/shape-runner-observer-refusal-shows-step-failure/acceptance.feature
- features/steps/shape-runner-observer-refusal-shows-step-failure__steps.js
- features/steps/shape-runner-observer-refusal-shows-step-failure__harness.js

When a step fails although the test runner reported a passing run, the runner-observer refusal says only that, without the failure message of the step, so the shaper cannot see which check rejected the runner's real output. Today `runnerObserverGate` (shape/shape.go) writes each item as `- %s (line %d): %s` (scenario name, line, the runner-observer text), for example `- <scenario> (line N): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips`. An item of that passing-run kind now ends with `; step message: <first non-blank line of the failing step's message>`, that line cut at 200 runes. The failing step's message is the scenario result's `ScenarioResult.Detail` (kogen-verification verify/report.go), which already carries it; no new data is needed. The text before the suffix is exactly as before. Items of the other kind (an anchored regular expression used as a literal name filter) and the headline are unchanged. A scenario without a detail (nil, or no non-blank line) gets its item exactly as before. The same scenarios are refused as before; no configuration is added and kogen-verification is not changed.

The Build also updates the expectation constants of shape/runner_observer_test.go (the passing-run item `runnerObserverItem1` and what is built from it) so the existing tests of that file pass with the new suffix.

## Decisions
- The suffix is added only to items whose runner-observer text starts with `the step failed although the test runner reported a passing run`; the literal-filter item is never changed. (decision: D1, round 1) (evidence: shape/shape.go; shape/runner_observer_test.go) (source: request)
- The line used is the first line of the detail that is non-blank after trimming whitespace; it is used trimmed of surrounding whitespace and cut to at most 200 runes (not bytes); a line of exactly 200 runes is kept whole. (decision: D2, round 1) (assumption: trimming surrounding whitespace of the chosen line is acceptable) (source: request)
- No setting, flag or new data: the detail already exists on the scenario result (ScenarioResult.Detail). (decision: D3, round 1) (evidence: shape/shape.go) (source: request)

## Source comparison
- Suffix `; step message: <first non-blank line>` cut at 200 runes: addressed in shape/shape.go runnerObserverGate and acceptance scenarios "A passing-run item ends...", "A multi-line message...", "A first line longer than 200 runes...", "A first line of exactly 200 runes...".
- Literal-filter item and headline unchanged; no-detail item unchanged: addressed by the scenarios "The literal-filter item is byte-identical...", "A scenario without a detail..." and "Both kinds in one refusal...".
- Updating the expectation constants of shape/runner_observer_test.go: addressed by the Build; go-test check runs them.

## Tests
- A passing-run item ends with `; step message: ` and the first non-blank line of the step message, cut at 200 runes; other items unchanged: focused Go test of runnerObserverGate through the acceptance scenarios (unit)
- The existing runner-observer refusal tests expect the new suffix: update the constants in shape/runner_observer_test.go (existing)

## Feasibility
- none
