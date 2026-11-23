# Ask the caller after a repeated guarantee conflict

Slug: shape-escalate-repeated-guarantee-conflict
Model: not applicable — one counter of reported guarantee contradictions per shaping run decides between the existing refusal and one question; no interacting states beyond the existing submission attempts
Acceptance:
- .kogen/intents/shape-escalate-repeated-guarantee-conflict/acceptance.feature
- features/steps/shape-escalate-repeated-guarantee-conflict__steps.js
- features/steps/shape-escalate-repeated-guarantee-conflict__harness.js

When the decision check reports, in a second submission of the same shaping run, the same contradiction between an explicit guarantee of the request (for example "exactly once") and a draft Decision that states what happens after a crash, Shape stops refusing paraphrases. It ends the run with one question to the caller (the existing questions output, exit 5) asking which crash behavior is required: retrying an unconfirmed delivery or leaving it unresent. Neither option is marked recommended or treated as authorized. The request and the draft are preserved until the caller answers. A draft whose contradiction was reported once and is resolved in the next submission goes through the normal checks and is published as before. Out of scope: changing webhook delivery semantics and weakening the request's explicit guarantees; the decision check's verdicts and feedback for a first report are unchanged.

## Decisions
- The guarantee stays as the request words it: delivery exactly once. The question asks the caller to choose between retrying an unconfirmed delivery and leaving it unresent; the draft is not approved with either weaker guarantee (at-least-once or at-most-once) as equivalent. (source: request)
- Same contradiction means: two different submissions of one shaping run whose decision checks each report a contradicts item whose request_value, normalised with normalizeNotation, is the same explicit guarantee (it matches decisionGuaranteeRe) and whose statement mentions a crash. The statement's wording may differ between the submissions. (assumption: the caller confirms that the same guarantee value, not the same statement text, identifies the repeated contradiction) (source: request)
- On the second report Shape makes no further shaper request and no provider rewrite: it ends with the questions outcome (exit 5, questions.json written) holding one question with exactly two options, one that retries the unconfirmed delivery and one that leaves it unresent, neither marked recommended. The question names the guarantee (request_value) and the draft's value (intent_value). The option wording is the builder's; acceptance only requires that one option mentions retrying and the other mentions leaving the delivery unresent. (assumption: the caller confirms the two options) (source: request)
- The counter is kept in memory per run and counts a submission once; the first report is refused exactly as today. A submission whose decision check reports no such contradiction continues through the normal checks. (evidence: shape/decision.go; shape/shape.go; shape/checkorder.go) (source: request)
- Scope is shape/shape.go, shape/decision.go and shape/submission_retry_test.go; tests are synthetic with scripted models and no live provider. (evidence: shape/submission_retry_test.go; shape/helper_test.go) (source: request)

## Source comparison
- Addressed, acceptance 1: scenario "A guarantee conflict reported twice ends with one question" (case webhook-twice) asserts exit 5, two decision checks, two shaper requests (literal, example and contract check requests are not counted), so no third rewrite is requested.
- Addressed, acceptance 2: scenario "The question offers retry or leave unresent and authorizes neither".
- Addressed, acceptance 3: scenario "A conflict reported once and then resolved is published" (case webhook-resolved).
- Omitted by request: changing webhook delivery semantics.

## Tests
- A guarantee contradiction reported in two submissions asks one question and makes no third shaper request: scenario through Shape with scripted shaper and decision-check models (scenario)
- The question offers retry or leave unresent and recommends neither: same scenario (scenario)
- A contradiction reported once and resolved publishes normally: scenario (scenario)
- The existing retry tests stay unchanged and pass: shape/submission_retry_test.go (existing)

## Feasibility
- none
