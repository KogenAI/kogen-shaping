# Quote the refused decision in record problems

Slug: shape-decision-record-problems-quote-decision
Model: not applicable — one stateless gate rewrites the text of its problem lines; no retained state or interacting rule is added
Acceptance:
- .kogen/intents/shape-decision-record-problems-quote-decision/acceptance.feature
- features/steps/shape-decision-record-problems-quote-decision__steps.js
- features/steps/shape-decision-record-problems-quote-decision__harness.js

The decision record gate (`decisionRecordGate` in shape/ledger.go) names a refused sourced decision only by its position, so a shaper with several decisions has to guess which one to repair. After this change every problem line of that gate that begins `- sourced decision <n>: ` also quotes the submitted decision. The text of each reason after the colon, the reasons' order, and which submissions pass or are refused stay unchanged.

## Decisions
- A problem line that began `- sourced decision <n>: ` becomes `- sourced decision <n> ("<quote>"): <reason>`, where `<quote>` is the decision text of decision `<n>` and `<reason>` is today's reason text exactly. Every such line of the gate is rewritten, whichever check produced it (missing evidence and uncertainty, revises problems, evidence-item problems). Lines of the decision notes (`revises ... so ... stands`) are not problem lines and stay as they are. (evidence: shape/ledger.go; shape/decision_record_test.go) (source: request)
- The quote is the decision text with each run of whitespace made one space, cut to its first 120 characters (runes) with no ellipsis; a text of 120 characters or fewer is quoted whole. Whitespace is normalised so a multi-line decision cannot split a problem line. (evidence: shape/ledger.go) (assumption: collapsing whitespace is a detail the request does not state; it only matters for decisions that contain line breaks or repeated spaces) (source: request)
- The quote shrinks further so the whole line is at most 300 characters, the cut that refusals record, keeping the reason whole; when no room is left the quote is empty (`("")`). A reason that alone exceeds the room is still cut at 300 by the existing refusal cut. (evidence: shape/checkorder.go) (source: request)
- The quote gets no extra processing: it is carried into the problem lines like any other text in them. (evidence: shape/checkorder.go) (source: request)
- No setting, flag or environment variable is added. Existing expectations in shape/decision_record_test.go (and shape/decision_evidence_grounding_test.go, shape/batch_refusal_test.go) of `- sourced decision <n>: ` lines are updated to the new form. (evidence: shape/decision_record_test.go; shape/decision_evidence_grounding_test.go; shape/batch_refusal_test.go) (source: request)

## Source comparison
- Addressed, request "each problem line ... also quotes the submitted decision ... first 120 characters": the required form is `- sourced decision <n> ("<first 120 characters of the decision text>"): ...`, where the text after the colon is the unchanged reason. Scenarios "The refused second decision is named and quoted", "A long decision is quoted to its first 120 characters" and "A decision of exactly 120 characters is quoted whole".
- Addressed, observed refusal lines that name a decision only by position, which this change quotes: `- sourced decision 1: no evidence and no uncertainty; cite what this shaping read or ran, or state the uncertainty you decided under` (the reason text kept unchanged in the scenarios), `- sourced decision 8: no evidence and no uncertainty; ...`, and `- sourced decision 1: D1 because Kogen's decision check found the at-most-once/no-retry wording narrowed the request's exactly-once requirement is not a recorded decision that stands` (the last is a decision check line outside the decision record gate; it is not changed).
- Addressed, constraint "problem lines still fit the 300 character cut ... the quote shrinks": scenario "The quote shrinks so the whole line keeps its reason within 300 characters".
- Addressed, constraint "reason texts and their order are unchanged": the exact problem-line expectations in the scenarios.
- Addressed, request "a submission with two sourced decisions where only the second has neither evidence nor uncertainty": case `second-unsourced`, ordinal 2 and quote `Round halves up`.

## Tests
- Problem lines quote the decision, cut at 120 characters, shrunk to keep 300: add a model-free regression test to shape/ledger_test.go covering two decisions with only the second unsourced (ordinal 2 and its quote), a long text cut at 120, a 120-character text quoted whole, and a crowded line of 300 characters ending in the unchanged reason (regression)
- Existing exact problem-line expectations take the quoted form: update shape/decision_record_test.go, shape/decision_evidence_grounding_test.go and shape/batch_refusal_test.go (unit)
- The four cases end to end through native Shape with a scripted shaper: acceptance.feature (scenario)

## Feasibility
- none
