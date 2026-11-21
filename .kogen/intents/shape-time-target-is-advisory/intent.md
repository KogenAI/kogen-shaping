# Make Shape time a target, not a stop

Slug: shape-time-target-is-advisory
Model: not applicable — the change is time accounting and guidance text, with no state machine of its own
Acceptance:
- .kogen/intents/shape-time-target-is-advisory/acceptance.feature
- features/steps/shape-time-target-is-advisory__steps.js
- features/steps/shape-time-target-is-advisory__driver/run.js
- features/steps/shape-time-target-is-advisory__driver/observe_test.go.txt

Shape's elapsed time becomes a target, not a stop. 15 minutes is the usual guidance and 20 minutes is the upper end of the performance target. Crossing either point never refuses the next model request on the native route or the external route: shaping continues whenever it can make useful progress, and a valid late draft goes through every ordinary check. The hard whole-session clock guard in shape/budget.go and shape/shape.go (budgetSpent before each request, budgetStop and its "used its shaping budget" outcome) is removed, with no other elapsed cutoff in its place. KOGEN_SHAPE_BUDGET_MINUTES (the 900 second default and its 5 to 45 range) is kept for compatibility and changes the stated guidance only. The separate response budget of one provider response wait and the shapebench wall-time cap are unchanged.

The per-request time line and the prompt guidance state the target and the elapsed time and no longer print a remaining-of-budget figure or say the shaper used a shaping budget. Per request the line is exactly `Time: <elapsed> elapsed. Target: <N> minutes usual, <N+5> minutes upper end.` where elapsed is the wall time of the whole shaping in whole minutes (with seconds when it has any, as budgetText writes it) and N is 15 by default. The no-submit attempt lines and shaping_turn_counts keep reporting the figures they report now apart from deadline wording.

Context cancellation still ends further work and keeps its reason; a provider or command timeout remains a different outcome from the elapsed target. A literal such as a required under 120 seconds requirement that a draft weakens to 125 seconds is refused after the target is crossed exactly as before it; no test, approved literal or gate is relaxed by elapsed time.

The fixed invalid-submission retry limit (submissionRetryLimit = 3 and the retry budget in shape/retrybudget.go) is a different mechanism from elapsed time: this change leaves it exactly as it is, so three consecutive submissions refused at the same gate still end shaping whatever the clock says. The elapsed-time acceptance therefore never scripts three same-gate refusals.

Local check time is the union of the wall time of the readiness observation of a submission (the acceptance run on the unchanged tree and the kept-assertion repeat) and of its static checks. Where the two overlap, the overlap counts once. A shaper request, a tool call and a model check that run while no readiness observation or static check runs are not local check time. The shape_time event keeps every field it has today, including static_uncharged_ms with its present meaning (the wall time of timed-out static checks), and gains one field, local_check_ms. With no readiness observation and no static check, local_check_ms is 0 and every other field is unchanged. Examples: a shaper request of 100 seconds followed by a static check of 120 seconds and a readiness observation of 300 seconds started together gives local_check_ms 300000 and elapsed 400 seconds; a static check starting 200 seconds after the readiness observation and ending 20 seconds after it gives 320000; a decision check of 60 seconds alone after a 100 second request gives local_check_ms 0 and elapsed 160 seconds; three rounds of 100 seconds plus a 400 second readiness observation give a wall clock of 1500 seconds and local_check_ms 1200000, with a fourth request still made. Separately, a scripted shaping of three rounds of 500 seconds each (two refused submissions at one gate, then a read round) still makes a fourth request after 1500 seconds of wall clock.

The change is limited to shape/budget.go, shape/shape.go, shape/evidence.go and new model-free test files in package shape.

## Decisions
- Remove the elapsed-time refusal of the next model request on native and external routes and keep shaping while it can make progress; no other elapsed cutoff replaces it. (decision: D1, round 1) (evidence: shape/budget.go; shape/shape.go) (source: request)
- The per-request time line is exactly `Time: <elapsed> elapsed. Target: <N> minutes usual, <N+5> minutes upper end.` with N the configured guidance minutes (15 default); it states no remaining figure and no deadline wording. (decision: D2, round 1) (evidence: shape/budget.go) (assumption: the exact wording of the new line is chosen by the shaper) (source: request)
- Local check time is the union of the wall time of the readiness observation of a submission (the acceptance run on the unchanged tree and the kept-assertion repeat) and of its static checks; where the two overlap, the overlap counts once. The shape_time event keeps every field it has today, including static_uncharged_ms with its present meaning (the wall time of timed-out static checks), and gains one field, local_check_ms. (decision: D3, round 1) (evidence: shape/evidence.go) (source: request)
- KOGEN_SHAPE_BUDGET_MINUTES keeps its name, 5 to 45 range and 900 second default and changes guidance only. (decision: D4, round 1) (evidence: shape/budget.go) (source: request)
- A weakened literal (under 120 seconds weakened to 125 seconds) is refused regardless of elapsed time; the literal check gains no time input. (decision: D5, round 1) (evidence: shape/literal.go) (source: request)
- The invalid-submission retry limit in shape/retrybudget.go and submissionRetryLimit stay unchanged; the fourth-request scenario stays below it by scripting two refused submissions and a read round, not three same-gate refusals. (decision: D6, round 2) (evidence: shape/retrybudget.go; shape/shape.go) (source: request)
- Three rounds of a 100 second shaper request each followed by a 400 second readiness observation give a wall clock of 1500 seconds and a shape_time local_check_ms of 1200000, the readiness observations being the only local check time. (decision: D7, round 2) (evidence: shape/evidence.go) (source: request)

## Source comparison
- Addressed: no refusal of the next request at 15, 20 or 45 minutes; a later valid draft passes all ordinary checks (acceptance scenario 1).
- Addressed: cancellation retains its reason and is not reported as a target, budget or timeout outcome (scenario 2).
- Addressed: the under 120 seconds literal weakened to 125 seconds is still refused (scenario 3), observed through the literal check, which takes no clock input.
- Addressed: exact time-line text with target, elapsed time and no remaining figure (scenario 4).
- Addressed: local_check_ms union semantics (decisions D3 and D7), the decision-only case with a golden of every other field, and the 1200000 ms local check time of three readiness rounds (scenarios 5, 6, 7 and 9).
- Corrected (round 2, builder-reported contradiction, independently checked in shape/retrybudget.go: refuseSet ends shaping at 3 consecutive same-gate refusals, so the old fixture of three refused a.go, b.go, c.go submissions never reached a fourth request whatever the clock): scenario 8 now scripts two refused submissions then a read round, each 500 s, and still expects a fourth request after 1500 s. Superseded: the earlier fixture of three same-gate refusals. Retry limit behaviour is unchanged and out of scope.
- Omitted: docs, README, contract folder and cmd/shapebench are out of scope.

## Tests
- No elapsed refusal at 15, 20 or 45 minutes: model-free observation of a scripted native shaping with a fake clock through Shape (scenario)
- Cancellation keeps its own outcome: observation of a cancelled Shape run (scenario)
- A weakened literal stays refused: observation of the literal check on the under 120 seconds and 125 seconds packages (unit)
- Time line states target and elapsed time only: exact-text comparison from the same run (scenario)
- local_check_ms union, decision-only and readiness-round accounts: Evidence-level fake-clock observation of the shape_time event (integration)
- Fourth request after 1500 seconds of long rounds, two of them refused submissions: scripted native shaping (scenario)
- Budget stop and the old budget line are removed: update the existing budget tests in shape/budget_test.go and shape/budget_stop_test.go that expect a budget stop or the old budget line (existing)

## Feasibility
- none
