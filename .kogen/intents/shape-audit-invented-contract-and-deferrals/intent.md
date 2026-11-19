# Audit invented contract details and deferrals

Slug: shape-audit-invented-contract-and-deferrals
Model: not applicable — one stateless audit inside the existing contract review call, keyed by the package hash
Acceptance:
- .kogen/intents/shape-audit-invented-contract-and-deferrals/acceptance.feature
- features/steps/shape-audit-invented-contract-and-deferrals__steps.js
- features/steps/shape-audit-invented-contract-and-deferrals__harness.js

Two Intents shaped on hard tasks passed Kogen's gates and were still wrong: one invented route paths, element ids, parameter names, plan keys and a numeric bound that its request never states; the other deferred controls.json and had no scenario for an interrupted delivery, so two wrong implementations passed. This change adds one audit to the existing contract review call (`submit_contract_review`) of the shape package, with two new finding kinds that refuse a candidate. No new model request is made per candidate.

1. `invented_contract`. The review reports, in a new top-level array `invented` of objects `{scenario, literal, why}`, each literal that a scenario step or a step file under Acceptance drives or asserts and that is a route path, element id, parameter name, class or constant name, identifier key or numeric bound. Kogen keeps the report only when the literal occurs in the candidate acceptance.feature or in a step file listed under Acceptance, and does not occur in the request, in the questions and answers, or in a tracked file of the unchanged tree (read from HEAD). The refusal line names the scenario and the literal and says to find it at run time (for example find the form or link by its visible text), to assert only what the request states, or to move the literal to a Decision and stop asserting it. A sentence of intent.md that fixes a mechanism the request does not require and that rules out another correct reading is reported the same way, with the sentence as the literal and scenario empty.
2. `uncovered_failure_mode`. Only when controls.json is a deferral, the review reports in a new top-level array `failure_modes` objects `{property, direction, wrong_implementation, scenario, assertion}`, where direction is one of `too_much`, `too_little`, `interrupted`. For each observable property of the request with a guarantee word (exactly once, in a row, never, at least) there must be at least one `too_much` and one `too_little` entry, and also an `interrupted` entry when the property concerns delivery, ordering or persistence; the review reports the property in a new top-level array `guarantees` of objects `{property, request, needs_interrupted}` with request copied from the Request. Each entry's assertion must be text of a Then step of the named scenario, and the scenario must fail on the unchanged tree according to the readiness observations. The refusal names the property and the missing direction, or the scenario whose assertion is missing or passes on the unchanged tree.

Keys absent from a review payload count as empty arrays. A candidate whose controls.json is not a deferral is not audited for `uncovered_failure_mode`. The audit adds no model request: its findings are Kogen's own evidence checks over the existing review payload, refused with the existing contract refusal headline, and an unchanged package reuses its recorded refusal exactly as today; a changed package is reviewed again. Every other refusal, event, retry count and published draft is unchanged.

## Decisions
- The two new finding kinds are named `invented_contract` and `uncovered_failure_mode` in the refusal lines, and the review payload carries them in the optional top-level arrays `invented`, `guarantees` and `failure_modes` described above, absent meaning empty. (decision: D1, round 1) (constraint: No new model request is added and existing refusal wording is unchanged.) (evidence: shape/contract.go; shape/contract_check_test.go) (assumption: The payload key names are chosen here because the request fixes only the finding kind names.) (source: request)
- Kogen keeps an `invented` report only when the literal appears in the candidate feature or a listed step file and not in the request, the questions and answers, or a file tracked at HEAD, so a literal sourced from the tracked tree or the request is never refused. (decision: D2, round 1) (evidence: shape/contract.go) (source: request)
- A failure-mode entry counts only when its scenario fails on the unchanged tree and its assertion text occurs in a Then step of that scenario in the candidate; a scenario that passes unchanged is a happy path. (decision: D3, round 1) (evidence: shape/contract.go) (assumption: Using the readiness observations as the test for a happy-path assertion is the shaping choice; the request leaves the test open.) (source: request)
- Acceptance is a Go harness compiled into the shape tests by overlay at shape/shape_audit_invented_contract_and_deferrals_acceptance_test.go, which never exists in the repository, using the existing scripted shaper and review helpers; it needs no browser, Rails or Studio. (decision: D4, round 1) (evidence: features/steps/k-shape-controls-minimal-witnesses__steps.js; shape/contract_check_test.go; shape/helper_test.go) (source: request)

## Source comparison
- Addressed: invented_contract finding with scenario and literal and its three remedies (D1, D2, change 1).
- Addressed: uncovered_failure_mode for deferred controls with both directions plus interrupted run, each tied to a failing assertion (D3, change 2).
- Addressed: controls that ran skip the audit, no extra model request, unchanged resubmission reuses the refusal and a changed step file is audited again (acceptance scenarios 6 and 7).
- Omitted: Refine, approve, readiness and other refusals are not touched.
- Lab evidence, not product scope (kept as the request writes it): the climb directory `/home/lanes/work.noindex/aor-climb/hypotheses/h9057-cc-shaping-c000-base2/cells`; the cells `hard-shape-saas-billing__base-r1` and `hard-shape-webhook-delivery-trust-and-restart__base-r1`, each holding `evaluation.json` and `intent/`. In the billing cell the Intent invented the plan keys `unlimited_monthly` and `unlimited_plus_monthly`, the paths `account/subscription` and `account/billing_portal` with a `plan` parameter, the page element `#billing` and an absolute bound of 25 queries; in the webhook cell it had no scenario for an interrupted attempt. Acceptance does not read them; scenarios 1 to 5 reproduce the two failure shapes with the scripted shaper.

## Tests
- invented literal refused naming scenario and literal: scenario 1 of the acceptance feature (scenario)
- sourced literal publishes: scenario 2 (scenario)
- deferred controls missing a direction refused naming property: scenario 3 (scenario)
- complete failure modes publish: scenario 4 (scenario)
- happy-path assertion refused: scenario 5 (scenario)
- real controls skip audit with the same request count: scenario 6 (scenario)
- unchanged resubmission reuses the refusal, a changed step file is audited again: scenario 7 (scenario)

## Feasibility
- none
