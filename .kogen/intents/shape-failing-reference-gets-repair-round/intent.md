# Give a failing reference a Shape repair round

Slug: shape-failing-reference-gets-repair-round
Model: not applicable — the change is one refusal path and one extra shaper round, with no state machine of its own
Acceptance:
- .kogen/intents/shape-failing-reference-gets-repair-round/acceptance.feature
- features/steps/shape-failing-reference-gets-repair-round__steps.js
- features/steps/shape-failing-reference-gets-repair-round__driver/run.js
- features/steps/shape-failing-reference-gets-repair-round__driver/observe_test.go.txt

When contract review finds that a scenario fails with the reference implementation (the `correct` overlay of controls.json) in place, the fault is in the shaper's own reference or test, so the shaper gets a repair round. The refusal returned to the shaper names the failing command, its exit code and the first failing assertion lines, and says the reference or the test is at fault. Shape then makes at least one further shaper request, however much time has elapsed and whether or not the 15 to 20 minute target has been crossed, unless the run was cancelled or ended by an independent provider, command, account or custody outcome. A first reference failure never counts as a repeated failure for the existing no-progress evidence (shape/retrybudget.go).

If the draft still fails with the reference in place after that extra round, it is refused with exactly today's refusal text plus one more line, `reference and test were not repaired in the extra round`. The extra round is granted once: no further repair request follows.

Which drafts pass, and every other refusal text, are unchanged. A refusal that is not about the reference (for example a contradicting scenario) gets no extra round and is handled exactly as today. This change adds no elapsed-time rule and does not read or change the Shape target, KOGEN_SHAPE_BUDGET_MINUTES or any time field; shape-time-target-is-advisory owns those. The change is limited to package shape (shape/controls.go, shape/shape.go, shape/retrybudget.go as needed) and its model-free tests with a fake client and scripted checks.

## Decisions
- A reference failure is the contract-review refusal in which the correct implementation of controls.json does not pass a scenario (the "no pass is reachable: with the correct implementation in place" problem); no other refusal is treated as one. (decision: D1, round 1) (evidence: shape/controls.go; shape/shape.go) (source: request)
- The refusal given to the shaper for a reference failure contains the failing command, its exit code and the first failing assertion lines, and states that the reference or the test is at fault. (decision: D2, round 1) (evidence: shape/controls.go) (assumption: the exact wording around the command, exit code and assertion lines is chosen by the builder) (source: request)
- Shape makes at least one further shaper request after a first reference failure, whatever the elapsed time, unless the run was cancelled or ended by an independent provider, command, account or custody outcome; a first reference failure is not counted as a repeated failure in shape/retrybudget.go. (decision: D3, round 1) (evidence: shape/retrybudget.go; shape/shape.go) (source: request)
- A draft that still fails with the reference in place after the extra round is refused with today's refusal text plus the line `reference and test were not repaired in the extra round`, after exactly two shaper requests. (decision: D4, round 1) (source: request)
- A refusal not about the reference gets no extra round and behaves as today; the change reads no elapsed time. (decision: D5, round 1) (evidence: shape/budget.go; shape/retrybudget.go) (source: request)

## Source comparison
- Addressed: a reference failure at 1300 s names command, exit code and assertion lines and a second request is made (scenario 1).
- Addressed: a second request that fixes the reference passes (scenario 2).
- Addressed: an unrepaired reference is refused with today's text plus the fixed line after two requests (scenario 3).
- Addressed: a contradicting-scenario refusal gets no extra round (scenario 4).
- Omitted: Shape target, environment variable and time fields (owned by shape-time-target-is-advisory); red main branch repair (owned by shape-repair-red-main-intent).

## Tests
- Reference failure refusal names command, exit code and assertion lines and a second request follows at 1300 s: scripted native shaping with a fake client and clock (scenario)
- A second request that repairs the reference yields a passing Intent: same harness (scenario)
- An unrepaired reference is refused with the extra line after two requests: same harness (scenario)
- A refusal not about the reference gets no extra round: same harness (scenario)

## Feasibility
- none
