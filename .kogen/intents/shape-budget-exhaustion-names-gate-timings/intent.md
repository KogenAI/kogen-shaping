# Name gates and timings in shaping budget refusal

Slug: shape-budget-exhaustion-names-gate-timings
Model: not applicable — the change only reports already-measured durations in a refusal text; it adds no state machine and changes no gate decision.
Acceptance:
- .kogen/intents/shape-budget-exhaustion-names-gate-timings/acceptance.feature
- features/steps/shape-budget-exhaustion-names-gate-timings__steps.js
- features/steps/shape-budget-exhaustion-names-gate-timings__harness.js

Shape ends in shape/invalid_output after its 15-minute shaping budget (class `shape`, reason `invalid_output`, exit 1). That refusal's detail begins "the shaper used its 15-minute shaping budget"; it now says how the time was spent. Its lines list, for every refused submission in order, the refusing gate, the time of the model request that produced the submission, the time Kogen spent checking it, and the shaping budget left when that check ended. It also states the shaping budget as a figure in seconds that equals the `budget_seconds` of the `shaping_turn_counts` event of the same run, and says that the separate `response_budget` event (the limit on waiting for one provider response) is a different budget from the shaping budget.

Exact additions to the refusal's lines (`Lines`; the existing detail sentence, its elapsed/request counts, the last-refusal lines and the last tool-free answer line stay as they are):
- one line `shaping budget: <S> s (<M> min, wall-clock for the whole shaping, all rounds; the response_budget event is the separate limit on one provider response wait)`, where `<S>` is the shaping budget in seconds (900) and `<M>` in minutes (15);
- then one line per refused submission, oldest first: `attempt <N>: gate <gate>, model request <A> s, Kogen checks <B> s, <R> s remaining`, where `<N>` is the submission number, `<gate>` the gate that refused it (the first refusal when several gates refused), `<A>` the whole seconds from sending the model request (native shaper) or starting the round (external shaper) until its answer was received, `<B>` the whole seconds Kogen spent checking that submission, and `<R>` the whole seconds of the shaping budget left when that check ended, never below 0.

Boundaries: no change to which drafts pass or are refused, to the budget itself (15 minutes), to when shaping stops (before the next provider request once the elapsed time reaches the budget; a request already in flight finishes), or to the `response_budget` event, which belongs to the provider library and is not edited. Not changed: the stop for repeated tool-free answers, retries, and readiness. Attempts that were not refused at a gate (observations returned for acknowledgment) are not listed.

## Decisions
- The refusal text reports the shaping budget as a seconds figure taken from the same constant that the `shaping_turn_counts` event records, so both always agree. (decision: D1, round 1) (evidence: shape/budget.go; shape/toolfree.go) (source: request)
- The '30 minutes' `response_budget` event is the provider library's per-response wait limit, a different budget from the 15-minute shaping budget; the difference is documented in the refusal text rather than by changing the event or either value. (decision: D2, round 1) (evidence: shape/response_budget_test.go; shape/budget.go) (source: request)
- Which drafts pass is unchanged and the stop rule is unchanged: the budget is checked before each provider request, and an in-flight request finishes. Therefore a draft is not cut by the shaping cap while the response wait still has room; this is documented, not altered. (decision: D3, round 1) (constraint: no change to which drafts pass) (evidence: shape/budget.go; shape/budget_test.go) (source: request)
- Only refused submissions are listed as attempts, each identified by the submission number and the first refusing gate recorded in `submission_refused`. (decision: D4, round 1) (evidence: shape/shape.go; shape/retrybudget.go) (source: request)
- Durations are measured on the shaping's own clock (the one that drives `shaping_turn_counts`), as whole seconds truncated, so test clocks control them. (decision: D5, round 1) (evidence: shape/shape.go; shape/evidence.go) (source: request)

## Source comparison
- Shape ends in shape/invalid_output after its 15-minute shaping budget: the refusal class `shape`, reason `invalid_output` and its 15-minute wording are kept; source request.
- Budget refusal lists per attempt the refusing gate, model-request time, Kogen check time and remaining budget: addressed by the `attempt <N>: ...` lines; source request.
- `response_budget` event and refusal name the same budget: the figures are two different budgets; addressed by documenting the difference in the refusal and making the shaping budget figure equal `shaping_turn_counts.budget_seconds`; source request ("or the difference is documented").
- A draft one round from passing is not cut by a stale cap when the other budget still has room: documented as unchanged (D3); no behavior change, per the constraint that which drafts pass does not change.
- Related queued work (shape-report-all-refusing-gates, shape-keep-existing-decisions-supported): out of scope.
- Focused model-free regression test, a scripted shaper refused four times yields four timed attempts and a budget figure equal to the event's: addressed in acceptance.feature.

## Tests
- A budget-exhausted shaping refusal lists one timed attempt line per refused submission with gate, model time, check time and remaining budget: a model-free scripted-shaper test refused four times that asserts four attempt lines (scenario)
- The refusal's shaping budget figure equals the `shaping_turn_counts` budget_seconds of the same run: the same scripted test compares the two (scenario)
- The refusal documents that `response_budget` is a separate budget: the same scripted test checks the sentence (scenario)
- Which drafts pass and when shaping stops are unchanged: the existing tests in shape/budget_test.go (existing)

## Feasibility
- none
