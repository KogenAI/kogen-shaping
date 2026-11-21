# Check a request before Shape spends a model call

Slug: request-check-before-shape-spend
Model: not applicable — the check is a pure function of one request text and one checkout, with no state machine
Acceptance:
- .kogen/intents/request-check-before-shape-spend/acceptance.feature
- features/steps/request-check-before-shape-spend__steps.js
- features/steps/request-check-before-shape-spend__driver/run.js
- features/steps/request-check-before-shape-spend__driver/observe_test.go.txt

`kogen intent shape` (the entry point is `shape.Shape` in shape/shape.go) first runs a model-free request check. No provider request, shaper round or Kogen check starts before this check ends. It looks for four findings, each with one treatment: (a) the request names a protected file (one `project.Protected` returns) as something to edit: the edit demand is removed from the request, a line is added that names the file and says the change must not touch it, and Kogen prints `request repaired:` followed by a plain-language description of the change; the repaired text is what the shaper receives and what the Intent keeps as its request text (state.json `request`, the same text the captured source holds). (b) two scenarios have the same observable outcome, written in different words or not: the request is always refused and scenarios are never merged. (c) a scenario has no observable outcome: refused. (d) a referenced path does not exist in the checkout: refused. A refusal happens before any provider request, prints `request/needs_rewrite:` and one line per finding saying how to rewrite the request; the line for (b) names both scenario numbers, the line for (d) contains the missing path. A request with no finding behaves byte-identically to today: same events, same text, and the repaired line is never printed. No new flag, service or protected list; no docs change.

## Decisions
- The check lives in the shape package entry point and reuses `project.Protected`; no new protected list. (evidence: shape/shape.go; shape/context.go) (source: request)
- The repaired request replaces `st.Request`, so the shaper, state.json and the captured source all hold the same text. (evidence: shape/shape.go) (source: request)
- Fixtures of the acceptance write a request in this form, which the build may replace by the form the code and request format rules already use, provided its tests write fixtures in that form: scenarios are the numbered items under a `## Acceptance` heading, each stating its observable outcome after `Outcome:`; referenced paths are backticked tokens with a `/` or a file extension. Two scenarios with the same observable outcome are refused whatever their wording. (assumption: the repository states no request format, so the fixture form is the Shaper's choice) (source: request)
- Refusal is an error of class request, reason needs_rewrite, so it prints `request/needs_rewrite:`. (assumption: exit code left to the build) (source: request)
- A request with no finding produces the same events and text as a golden recorded from today's behaviour: the event names in order, standard output, and standard error with durations normalized. (evidence: shape/shape.go) (source: request)

## Source comparison
- Addressed: findings a to d and the clean request (acceptance scenarios 1 to 5). Scenario 5 compares the event names, standard output and normalized standard error with a golden recorded from today's unchanged code.

## Tests
- Protected-file demand is repaired and the fake client receives the repaired text: model-free test with a temporary checkout (scenario)
- Duplicate outcome, missing outcome and missing path are refused with zero client calls: same harness (scenario)
- A clean request produces the golden events and text: same harness (scenario)

## Feasibility
- none
