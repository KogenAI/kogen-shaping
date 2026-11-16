# Name red scenarios in the kept repeat outcome

Slug: shape-kept-repeat-outcome-names-scenarios
Model: not applicable — the change is one progress line and one event field built from an already computed list; there is no state to model
Acceptance:
- .kogen/intents/shape-kept-repeat-outcome-names-scenarios/acceptance.feature
- features/steps/shape-kept-repeat-outcome-names-scenarios__steps.js
- features/steps/shape-kept-repeat-outcome-names-scenarios__driver/harness_test.go.txt

When Shape repeats the readiness observation of kept scenarios, its outcome progress line `kogen: kept assertion repeat outcome: persistent kept assertion red (cap 2 executions)` does not say which scenario is red, and the `kept_assertion_repeat` event carries only `execution_cap`, `outcome` and `package`. The line is printed before the contract confirmation judges whether the red belongs to a kept property, so on real runs it reads like a failure nothing acts on and nothing in the log names the scenario.

When the repeat finds scenarios with the same assertion red in both observations, the progress line names them, in the order of the existing sorted `lines` of the kept-assertion repeat block of `shape/contract.go`: `kogen: kept assertion repeat outcome: persistent kept assertion red in "Admin accounts list links each account" (line 12), "Billing portal opens" (line 31) (cap 2 executions)`. The event gets a `scenarios` list of `{"name", "line"}` objects in the same order. When that list is empty (the outcome is no persistent red, or the red comes only from baselineRefusals) the progress line and the event stay exactly as they are today and the event has no `scenarios` key.

Nothing else changes: which drafts pass or are refused, every refusal text, the repeat's execution cap of 2, the reference observations file text, and the configuration (no new setting). The existing tests that assert the current line text for the no-names cases keep passing. Only the kept-assertion repeat block of `shape/contract.go` changes, plus a focused model-free regression test in the shape package next to `kept_mixed_test.go`.

## Decisions
- The names are the scenarios of the existing `lines`, sorted as `lines` is sorted (the full sentences, so `Ant` precedes `Zebra` whatever their lines), rendered as `"<name>" (line <n>)` joined by `, ` after ` in ` and before ` (cap 2 executions)`; the event `scenarios` list uses the same order (evidence: shape/contract.go; shape/kept_mixed_test.go) (source: request)
- When no scenario is named the progress line, the event keys and the reference observations file text are untouched, and the `scenarios` key is absent rather than empty (evidence: shape/contract.go; shape/unpassable_reuse_test.go) (source: request)
- Acceptance drives the real `contractCheck` repeat through a Go test harness copied into the shape package by a Go build overlay (an unwritten test is not run by name), with real local Cucumber observations of kept scenarios, reading stderr and the `kept_assertion_repeat` event; the progress line is compared after removing the trailing elapsed ` (<n> s)` suffix the progress writer adds (evidence: shape/kept_mixed_test.go; shape/unpassable_reuse_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (source: request)
- Names follow the existing sorted lines order, rendered as "<name>" (line <n>) joined by ", " after " in "; event scenarios list in the same order (decision: D1, round 1) (constraint: no change to refusal texts, cap or reference file) (evidence: shape/contract.go; shape/kept_mixed_test.go) (assumption: The correct patch and harness were checked in a scratch overlay probe in this session only; Kogen readiness is authoritative.) (source: request)
- With no named scenarios the progress line, event keys and reference file stay unchanged and the scenarios key is absent (decision: D2, round 1) (evidence: shape/contract.go; shape/unpassable_reuse_test.go) (assumption: Red from baselineRefusals only is not separately exercised by acceptance.) (source: request)
- Acceptance drives the real contractCheck via a Go harness copied in with a Go build overlay, comparing the progress line after removing the trailing elapsed suffix (decision: D3, round 1) (evidence: shape/kept_mixed_test.go; shape/unpassable_reuse_test.go; features/steps/approve-static-reuse-line-explains-unknown__steps.js) (assumption: Progress lines carry a trailing ' (<n> s)' suffix observed in a scratch run.) (source: request)

## Source comparison
- Observed bare line `kogen: kept assertion repeat outcome: persistent kept assertion red (cap 2 executions)` and the event with only `execution_cap`, `outcome`, `package`, from the request: superseded when scenarios are red in both observations; addressed in scenarios "One persistent kept red scenario is named" and "Persistent kept red scenarios are named in sorted order".
- Observed progress lines on the real Rails run (context only, unchanged for the repeat start line): `kogen: repeating kept assertion observation (execution 2 of 2 maximum) (323 s)` followed by `kogen: kept assertion repeat outcome: persistent kept assertion red (cap 2 executions) (345 s)`; the trailing `(323 s)` and `(345 s)` are the elapsed-time suffix the progress writer appends, which this change does not alter. The repeat start line is not changed.
- Observed log context, not changed: the round 2 `contract_review` event is `result: clean` with 0 findings, and the `property_check` event dismisses the red assertion `GET /admin/accounts returns 404` (the new admin tools, not the kept property). This change only adds the scenario names to the repeat line and event so such a red can be traced; the review and property check are untouched.
- Expected event for one red scenario, from the request: the event `scenarios` is [{"name": "Admin accounts list links each account", "line": 12}]; addressed in "One persistent kept red scenario is named", which checks the list entries have exactly the keys `name` and `line` and the values `Admin accounts list links each account` and 12.
- Expected line with several scenarios in sorted order and the `scenarios` event list: addressed in "Persistent kept red scenarios are named in sorted order" (Ant at line 7 before Zebra at line 3).
- Empty `lines` keeps the line and the event unchanged, with no `scenarios` key: addressed in "No persistent kept red names no scenario" (a scenario red in the first observation only).
- The red coming only from baselineRefusals keeps today's line and event by the same empty-list rule; not separately exercised.
- Refusal texts, which drafts pass, the cap, the reference observations file and configuration: omitted by design, unchanged.

## Tests
- The progress line and event name one persistent kept red scenario: add a regression test in shape/kept_repeat_names_test.go (regression)
- Several persistent kept red scenarios are named in the sorted order of the existing lines: same regression test (regression)
- No persistent red leaves the line unchanged and the event without a scenarios key: same regression test (regression)
- Existing line text assertions for the no-names cases still pass: TestPersistentKeptAssertionRepeatCap and TestUnpassableCompletedReuse (existing)

## Feasibility
- none
