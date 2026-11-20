# Drop repeated invented literals in Shape

Slug: shape-invented-contract-converges
Model: not applicable — a bounded change to a review gate and prompt text; the only state is one run-scoped record of an offered rewrite
Acceptance:
- .kogen/intents/shape-invented-contract-converges/acceptance.feature
- features/steps/shape-invented-contract-converges__steps.js

Shape refuses an `invented_contract` finding of the contract review, and the same finding can repeat until the budget ends. After this change the first kept `invented_contract` finding of a Shape run is still a refusal with its fix line, and Kogen records the normalized literal and scenario of each kept finding and that one rewrite was offered. On every later submission of the same run an `invented_contract` finding is dropped, not refused: Kogen writes one shaping event `invented_literal_dropped` per dropped finding (fields `literal`, `scenario`, `submission` number) and lists each dropped literal in the approval card under the heading `Literals chosen by the Shaper`, with the words that the Request does not state them. The rewrite is offered once per run, not once per literal, so a literal that first appears in the second submission is dropped too. A submission whose only findings are dropped ones passes the contract check. Every other finding kind, `uncovered_failure_mode` and every other gate stay refusals; the drop touches only `invented_contract` and never skips the rest of the audit. The drop is Kogen code in shape/audit.go and its caller in shape/contract.go. No new gate, tool, setting or model request is added.

The Shaper's instructions (shape/prompts/shaper.md, pinned by shape/testdata/prompts_golden.json, regenerated deliberately) say that a route, helper, parameter, environment name, state name or number that neither the Request nor a tracked file states is not asserted by the acceptance (the step finds it at run time or asserts the observable behavior the Request names); that a bound such as a query count or a size is added only when the Request asks for it; that an intent.md sentence never adds a mechanism (a single transaction, a count bound) the Request does not require; and that a refused `invented_contract` finding is rewritten into observable behavior on the next submission, not repeated. They also say, and the fix line of a `decision_check` finding that is weaker than a guarantee of the Request (exactly once, never, always, at most once, at least once, every, no) says: state the guarantee in the Request's own words, and put any limit of the repository or the receiver as a separate open limit beside it, never in place of it. The Shaper's Decision says "at-most-once is accepted for a crash mid-send" against the Request "Deliver exactly once."; that Decision stays refused, with a fix line quoting "Deliver exactly once."; a Decision that states "exactly once, including across a crash and a restart" and names the receiver limit as a separate open limit is accepted.

Build notes: the acceptance harness compiles shape/shape_invented_contract_converges_acceptance_test.go from Go source in the step file through a Go overlay; do not create that path and do not use the `sicc` identifier prefix in the package. Keep `newShpFx`, `shpSubmit`, `shpAnswer`, `shpCall`, `shpFiles`, `ccFeaturePath`, `ccStepsPath`, `ccFindingArgs` and the `f.Prov.contract` script in the package tests unchanged. Add the replay cases shape/testdata/replay and shape/audit_converge_test.go as described in Source comparison; the Build's own tests cover the six cells, each shaped to an approved Intent within the Shape budget of 900 s of scripted time.

## Decisions
- The rewrite is offered once per Shape run; the record holds normalized literal and scenario of each kept finding. (source: request)
- The `invented_literal_dropped` event has the fields `literal`, `scenario` and `submission`. (assumption: field names are chosen by Shape, the request names only the event and the three values) (source: request)
- The card heading is exactly `Literals chosen by the Shaper`, shown only when a literal was dropped. (source: request)
- Only `invented_contract` is dropped; `uncovered_failure_mode` stays a refusal even beside a dropped literal. (evidence: shape/audit.go) (source: request)

## Source comparison
- Addressed: parts one and two of the request, items 1 to 6 of the expected behavior, through scenarios 1 to 4 of the feature (drop, once-per-run, prompt text) and the existing tests.
- Replay cases (Build tests under shape/testdata/replay, run by shape/audit_converge_test.go within the Shape budget of 900 s of scripted time), named after the six cells: rails-ft-saas-billing r1 (`contract_review` refuses `invented_contract` for `assert_operator counts.last, :<=, counts.first + 2` and for the sentence "at most 25 SQL queries for 10 or for 100 accounts"); rails-ft-saas-billing r2 (route helpers `account_subscription_path` and `account_billing_portal_path`, the parameter `plan: "unlimited_plus"`, environment names `STRIPE_UNLIMITED_PRICE_ID` and `STRIPE_UNLIMITED_PLUS_PRICE_ID`); rails-ft-data-export-and-import r1 (routes `#{@slug}/account/exports` and `#{@slug}/users/#{@member.id}/data_exports`); rails-ft-data-export-and-import r2 (`assert_operator n, :<=, 500` and the sentence "The whole import runs in one database transaction so failure leaves no rows behind"); rails-ft-webhook-delivery-trust-and-restart r1 and r2 (the Decision above). In each approved result every such literal is absent from the step files and intent.md or listed under `Literals chosen by the Shaper`.
- Not covered by this acceptance file: `uncovered_failure_mode` beside a dropped literal, the decision check fix line and the six replay cases; they are the Build tests named above.

## Tests
- First kept invented literal is refused, later ones are dropped and recorded: sicc harness through Shape and scripted provider (scenario)
- Rewrite offered once per run: second-submission new literal is dropped (scenario)
- Prompt text states the rules: shaper.md contains the new instructions (unit)
- uncovered_failure_mode, decision check fix line and six replay cases: shape/audit_converge_test.go (integration)

## Feasibility
- none
