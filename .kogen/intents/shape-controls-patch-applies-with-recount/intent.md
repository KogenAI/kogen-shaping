# Apply controls patches with git apply --recount

Slug: shape-controls-patch-applies-with-recount
Model: not applicable — one fixed argument of a git invocation, with no state or interacting rules
Acceptance:
- .kogen/intents/shape-controls-patch-applies-with-recount/acceptance.feature
- features/steps/shape-controls-patch-applies-with-recount__steps.js
- features/steps/shape-controls-patch-applies-with-recount__harness.js

When Shape checks a controls.json implementation that gives a file as `{"patch":"<unified diff>"}`, `writeOverlay` in `shape/controls.go` applies the patch with `git apply` without `--recount`. A model-written patch whose hunk header counts are off by one but whose context and changed lines are right is then refused as "the patch does not apply to the candidate tree", and the shaper spends a round rewriting a patch that was fine. Two observed refusals of this kind were `controls.json: wrong implementation "fixed-width content on the page": src/pages/sponsors.astro: the patch does not apply to the candidate tree` and `controls.json: the correct implementation: scripts/design_site.py: the patch does not apply to the candidate tree: error: patch failed`.

After this change `writeOverlay` applies each patch with `git apply --recount`, so a patch with wrong hunk counts and correct context and lines applies, and the implementation is judged as before. A patch whose context lines do not match the tree is still refused with the same message and the same failed-hunk display as today. Nothing else about the controls rule changes: no new configuration, flag or environment variable, and nothing is removed.

## Decisions
- Both the `--numstat` call and the applying call of `writeOverlay` pass `--recount`; the argument is fixed, with no setting. (evidence: shape/controls.go) (source: request)
- A patch with one context line that does not match the tree is refused with `error: patch failed: <file>:<line>: <hunk header as written in the patch>: <file> from line <line> is: ...`, the failed-hunk form the existing tests pin. (evidence: shape/controls.go; shape/controls_test.go) (source: request)
- One wrong implementation is enough: applying without `--recount`, which refuses the miscounted patch. (source: request)

## Source comparison
- Addressed: "the overlay is applied with git apply --recount, so a patch with wrong hunk counts but correct context and lines applies". Scenario 1 gives a patch whose hunk header counts are one too small and requires no refusal line, with the other implementations still judged as before.
- Addressed: "A patch whose context lines do not match the tree is still refused with the same message and the same failed-hunk display". Scenario 2 changes one context line of the same patch and requires the single refusal line with the failed-hunk display.
- Addressed: the observed refusals `controls.json: wrong implementation "fixed-width content on the page": src/pages/sponsors.astro: the patch does not apply to the candidate tree` and `controls.json: the correct implementation: scripts/design_site.py: the patch does not apply to the candidate tree: error: patch failed` are the refusal form that a miscounted patch no longer triggers; the same form stays for a mismatched context.
- Preserved: "Nothing else about the controls rule changes", "no new configuration". The existing controls tests stay green.

## Tests
- A controls.json patch with a hunk header count one too small applies and the implementation is judged as before: a focused model-free Go test in `shape/controls_test.go` next to the existing patch tests (unit)
- The same patch with one context line changed is refused with the failed-hunk line: the same Go test (unit)
- Both rules end to end through the check of a candidate: acceptance.feature scenarios 1 and 2 (scenario)
- Existing patch overlay and failed-hunk tests keep passing: `TestControlsPatchOverlaysForLargeFiles` and `TestControlsInapplicablePatchShowsFailedHunk` (existing)

## Feasibility
- none
