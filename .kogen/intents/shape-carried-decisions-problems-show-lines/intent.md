# Show the refused line in carried-decision problems

Slug: shape-carried-decisions-problems-show-lines
Model: not applicable — one stateless gate extends the text of two of its problem lines; no retained state or interacting rule is added
Acceptance:
- .kogen/intents/shape-carried-decisions-problems-show-lines/acceptance.feature
- features/steps/shape-carried-decisions-problems-show-lines__steps.js
- features/steps/shape-carried-decisions-problems-show-lines__harness.js

The carried decisions gate (`carriedDecisionsGate` in shape/ledger.go) refuses a draft whose Decisions section misuses a recorded decision identity, but two of its problems name only the identity, so the shaper must search the whole Decisions section for the line to change. For example the observed refusal was the single line `- D7 is not a recorded decision that stands; remove its identity or submit a sourced decision`, with no hint which Decisions line carries D7. After this change both problems show the line they are about, so the fix is a local edit. Which drafts pass or are refused does not change; the same lines are refused and only their text grows.

## Decisions
- The problem `- <id> is not a recorded decision that stands; remove its identity or submit a sourced decision` becomes `- <id> is not a recorded decision that stands; remove its identity or submit a sourced decision: <the Intent Decisions line as written>`, where the added line is the Intent's Decisions line as written, whole and uncut. (evidence: shape/ledger.go; shape/decision_record_test.go) (source: request)
- The problem `- <id> does not match its recorded line` becomes `- <id> does not match its recorded line; keep it as: <the recorded line from the ledger> (Intent has: <the Intent line as written>)`, where the recorded line is the standing decision's ledger line exactly as the third problem kind `- <id> (round <n>) is not in Decisions; keep it as: <line>` already prints it, and the Intent line is the Intent's Decisions line as written, whole and uncut. (evidence: shape/ledger.go) (source: request)
- The wording before the added text stays exactly as it is, so the existing refusal text and the `sh.refused` gate name (`carried_decisions`) are unchanged: the gate keeps calling `sh.refused("carried_decisions", problems)`. The `appears more than once`, `use one recorded decision per line` and `- <id> (round <n>) is not in Decisions; keep it as: <line>` problems are unchanged. (evidence: shape/ledger.go) (source: request)
- The added line text is not cut; no setting, flag or environment variable is added. (evidence: shape/ledger.go) (source: request)

## Source comparison
- Addressed, request "both problem kinds show the Intent line they are about": scenario "The carried-decision problems show the lines they are about" expects the three exact problem lines for a ledger with superseded D1 and standing D2 (the request's `D7` example is the same kind as D1: a recorded identity that no longer stands).
- Addressed, request `- <id> is not a recorded decision that stands; remove its identity or submit a sourced decision: <the Intent Decisions line as written>`: the first problem line of that scenario.
- Addressed, request `- <id> does not match its recorded line; keep it as: <the recorded line from the ledger> (Intent has: <the Intent line as written>)`: the second problem line of that scenario.
- Addressed, request "the wording before the added text stays exactly as it is, so the existing refusal text and the sh.refused gate name are unchanged": the exact problem lines of that scenario begin with the unchanged wording, and the `sh.refused` call with `carried_decisions` is not edited.
- Addressed, request `- <id> (round <n>) is not in Decisions; keep it as: <line>` unchanged: the third problem line of that scenario.
- Addressed, request "no change to which drafts pass or are refused": the same three problems, in the same order, are refused as before; only text grows.
- Addressed, request "one wrong implementation: still prints the identity only": controls.json.
- Omitted by request: the 300-character cut on recorded events is a separate request.

## Tests
- Both identity problems show their Intent line, and the does-not-match problem also the recorded ledger line: add a model-free unit test to shape/decision_record_test.go with a ledger of superseded D1 and standing D2 and a draft carrying D1's line and a changed D2 line (unit)
- The same refusal through the acceptance harness, which runs the gate on that ledger and draft: acceptance.feature (scenario)

## Feasibility
- none
