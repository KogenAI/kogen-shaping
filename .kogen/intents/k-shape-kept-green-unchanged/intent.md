# Refuse red kept-behaviour scenarios

Slug: k-shape-kept-green-unchanged
Model: not applicable — validation of observed acceptance results, not a new product state machine
Acceptance:
- .kogen/intents/k-shape-kept-green-unchanged/acceptance.feature
- features/steps/k-shape-kept-green-unchanged__steps.js
- features/steps/k-shape-kept-green-unchanged__harness.js

For every scenario exercising a counted kept property, use unchanged-tree readiness observations to refuse a persistent assertion contradiction before publication. Feedback must name the scenario and its expected and observed values and instruct the shaper to derive expectations from unchanged code or the scenario's own fixture, not another fixture or the request's prose. Do not change or drop scenarios automatically. Keep missing-file, missing-command and environment failures in existing feasibility/environment handling. New properties are not subject to this check.

Repeat the existing unchanged-tree acceptance observation once for an assertion-red kept scenario: at most two observation executions total for the candidate under this check. Record that cap and the repeat outcome in progress/readiness evidence. Never repeat indefinitely. A persistent assertion red remains a contradiction; a missing prerequisite remains environment/feasibility evidence. This settles the bounded-repeat technical detail without adding a setting.

## Decisions
- Preserve existing behaviour when the request says to keep or pin it; an incorrect pinned assertion is not missing-feature red. (source: request)
- Keep green kept scenarios on the existing publication path without extra refusal or provider requests; requests without kept properties retain their exact provider-request sequence. No additional model request is introduced on that path. (source: request)
- A transient kept assertion failure that becomes green on repeat must not be refused. (source: request)

## Source comparison
- Addressed: Problem and Need, including shape/property.go's new-only checks and shape/contract.go's missing-feature exemption, by the kept-only readiness rule above. Existing review/confirmation remains available; neither an assertion contradiction nor an environment failure may be reclassified as new behaviour just to publish it.
- Addressed: synthetic acceptance 1, request "pin the current output of function f for input A; no behaviour change", by correct mode: f(A) returns observed-A, candidate expects wrong-A, feedback refuses and a corrected resubmission publishes.
- Addressed: case 2 by green mode, with no extra provider request; case 3 by mixed mode, request "keep behaviour B and add C", refusing B only while C's new-behaviour red remains acceptable; case 4 by missing mode loading absent-fixture.txt; case 5 by plain mode; case 6 by flaky mode and the recorded two-execution cap.
- Addressed: synthetic local repositories, scripted shaper and contract review, no live task/provider. All acceptance bindings and helpers are owned by this Intent and use the requested prefix and paths.
- No requested behaviour is deliberately omitted or superseded. The missing-prerequisite case uses the expressly allowed missing-fixture alternative, not a missing binding that would invalidate acceptance itself.

## Tests
- Persistent kept assertion red is refused with scenario, expected/observed values and correction guidance, then corrected resubmission publishes: correct synthetic Shape journey (regression)
- Green kept scenarios publish without extra refusal or model request: green journey (integration)
- Mixed kept/new properties refuse only the kept contradiction: mixed journey (regression)
- Missing prerequisites remain on existing environment/feasibility handling: missing-fixture journey (integration)
- No kept property preserves the provider-request sequence: plain journey (regression)
- A red-then-green kept observation is not refused and repeats are bounded and recorded: flaky journey; add focused persistent-red repeat-cap coverage in production tests (integration)

## Feasibility
- evidence: {"id":"observer-controls","kind":"observer","inputs":["features/steps/k-shape-kept-green-unchanged__steps.js","features/steps/k-shape-kept-green-unchanged__harness.js"],"observers":["features/steps/k-shape-kept-green-unchanged__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-kept-green-unchanged__steps.js good","exit":0,"output":"good observations accepted\n"},{"name":"bad","command":"node features/steps/k-shape-kept-green-unchanged__steps.js bad","exit":1,"output":"bad observations rejected\n"}]}
