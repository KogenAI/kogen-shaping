# Make the shaping budget settable by environment

Slug: shape-budget-env-configurable
Model: not applicable — the change reads one value once per shaping and passes it to text and checks; there is no state to model
Acceptance:
- .kogen/intents/shape-budget-env-configurable/acceptance.feature
- features/steps/shape-budget-env-configurable__steps.js
- features/steps/shape-budget-env-configurable__driver/harness_test.go.txt

Shape reads the environment variable `KOGEN_SHAPE_BUDGET_MINUTES` once at the start of a shaping, through the session's environment. Unset or empty, the budget is 15 minutes exactly as today, with unchanged text and event fields. Otherwise the value must be a whole number of minutes from 5 to 45 and that many minutes is the budget of that shaping only. Every place that states or enforces the budget uses that one value: the time line the shaper sees after each request, the check that no further provider request starts once the budget is spent, the external session brief, the outcome text of a shaping that ran out of budget, the per-attempt remaining and over figures, and the budget_seconds field of the shaping_turn_counts event. A value outside 5 to 45 or not a whole number is refused before any provider request with the failure `kogen: shape budget must be a whole number of minutes from 5 to 45` (exit code 3, class environment); it is never clamped or ignored. The value lives in the one shaping that read it and in no package-level state, so two shapings in one process with different values do not affect each other.

Unchanged: the separate response budget of one provider response wait, the static check step limit, the exclusion of cold runtime installation and of timed-out static check wall time from the charged time, and the shapebench wall-time cap. No new configuration file field and no new command line flag. Existing tests keep passing without edits to their expectations. Scope: shape/budget.go (the budget becomes a per-shaping value and the text helpers take it), shape/shape.go (read the variable at the start, pass the value where the budget is used), shape/toolfree.go (the budget_seconds field), and the new tests in shape/budget_test.go and shape/budget_stop_test.go; docs, README, the contract folder and cmd/shapebench are out of scope. The new tests are model-free: a fake clock, a fake provider client, no network.

## Decisions
- The variable is read once per shaping from the session's environment and the parsed value is held by that shaping, not by package-level state (decision: D1, round 1) (evidence: shape/shape.go; shape/budget.go) (source: request)
- Unset or empty means exactly 15 minutes with byte-for-byte unchanged text and event fields (decision: D2, round 1) (evidence: shape/budget.go; shape/budget_test.go) (source: request)
- A value that is not a whole number of minutes from 5 to 45 is refused before any provider request with `kogen: shape budget must be a whole number of minutes from 5 to 45`, exit code 3, class environment, and is never clamped or ignored (decision: D3, round 1) (evidence: shape/shape.go) (source: request)
- The same value is used by the time line, the spent check, the external brief, the budget stop text, the remaining and over figures and budget_seconds (decision: D4, round 1) (evidence: shape/budget.go; shape/toolfree.go) (source: request)
- Existing helper names stay callable with the 15-minute default for existing tests and for Refine, which is out of scope (decision: D5, round 1) (evidence: shape/budget_static_timeout_test.go; shape/refine.go) (assumption: Refine keeps the 15-minute budget) (source: request)
- Acceptance drives real Shape through a Go harness copied into the shape package by a Go build overlay, since an unwritten test is not run by name (decision: D6, round 1) (evidence: shape/budget_test.go; shape/helper_test.go) (source: request)

## Source comparison
- Default 15 minutes, time line, 900 s event field and spent boundary at 900 not 899 seconds: addressed in scenario "Unset or empty keeps the 15-minute budget".
- Value 25: text, event field 1500 and spent boundary: addressed in scenario "25 sets the budget in the text, the event and the spent check".
- Outcome text, budget line and remaining and over arithmetic for 25: addressed in scenario "A shaping that runs out of a 25-minute budget says 25 minutes"; its first problem line stating the budget is `shaping budget: 1500 s (25 min`; the existing refusal lines that this shaping prints before it are unchanged because the default text stays byte-for-byte the same.
- Refusal of 46, 4, 0, -5, abc and 25.5 and acceptance of 5 and 45: addressed in the two outline scenarios; the exact failure text is checked as contained in the rendered failure.
- No state shared between two shapings in one process: addressed in the last scenario.
- The external session brief uses the same value: stated in D4, not asserted by acceptance.
- Docs, README, contract folder, cmd/shapebench, new flags or configuration fields: omitted by the request.

## Tests
- Time line, budget_seconds and spent boundary with the variable unset, empty and 25: new tests in shape/budget_test.go with a fake clock and fake provider (unit)
- Outcome text, budget line and remaining and over figures with 25: new test in shape/budget_stop_test.go (unit)
- Refusal of 46, 4, 0, -5, abc and 25.5 before any provider request and acceptance of 5 and 45: new test in shape/budget_test.go (unit)
- Two shapings in one process with 25 then unset: new test in shape/budget_test.go (unit)
- Default behaviour unchanged: existing budget tests (existing)

## Feasibility
- The Go harness was run on the unchanged tree in a scratch copy: it compiled, passed and showed the 15-minute default and no refusal for any value. (probe 1: exit 0, 15 s)
