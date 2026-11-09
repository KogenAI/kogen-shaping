# Name fixes for contract-review refusals

Slug: shape-contract-refusal-fix-line
Model: not applicable — each fix message is derived from one confirmed finding
Acceptance:
- .kogen/intents/shape-contract-refusal-fix-line/acceptance.feature
- features/steps/shape-contract-refusal-fix-line__steps.js
- features/steps/shape-contract-refusal-fix-line__harness.js

When Shape's contract review confirms a contradiction, retain the original finding with its scenario, expectation path and quote, and contract path and quote. Put an actionable fix immediately after that finding. The fix names the same scenario, exact expectation to change and exact contract to follow. The shaper's feedback, saved not-ready details and contradiction record keep the corresponding finding and fix. A corrected candidate can pass. Refuted or ungrounded findings remain nonblocking.

## Decisions
- Each confirmed contradiction keeps its current item line and adds one separate, source-specific fix line. (source: request)
- The count is the number of confirmed findings. The event preview keeps its existing limits, and the saved contradiction record keeps the complete text. (source: request)
- Preserve the confirmation rules and scope. Environment, property and wrong-implementation refusals, other Shape gates, and delivery scripts are outside this change. (source: request)

## Source comparison
- Addressed: a confirmed contradiction names the exact expectation to change and contract to follow. The feature's first scenario checks two independent findings.
- Addressed: the feedback and saved not-ready details keep the exact quotes and fix.
- Addressed: corrected, refuted and ungrounded results preserve the existing gate behavior.
- Addressed: the event preview remains bounded and the contradiction record preserves the complete fix.
- Excluded as requested: other refusal types, carried decisions, decision records, the decision classifier and kbuild.sh.

## Tests
- Each confirmed finding has a matching fix and retained quotes: contract-review acceptance (scenario)
- Saved details and long evidence keep the current preview and record boundaries: contract-review acceptance (scenario)
- Corrected, refuted and ungrounded candidates retain the existing outcomes: contract-review acceptance (scenario)
- Existing contract-review and Refine expectations cover the intentionally added diagnostic line: relevant contract tests (regression)
- Other Shape behavior stays covered by the complete suite: go test ./... (existing)

## Feasibility
- none
