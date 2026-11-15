# Shorten a title-only lint failure at once

Slug: shape-title-only-lint-fixed-in-place
Model: not applicable — the change is one stateless repair rule in a single submission check
Acceptance:
- .kogen/intents/shape-title-only-lint-fixed-in-place/acceptance.feature
- features/steps/shape-title-only-lint-fixed-in-place__steps.js
- features/steps/shape-title-only-lint-fixed-in-place__harness.js

When the `CheckIntent` call at the start of the intent gate (in the files-submission path of `acceptFiles`, shape/shape.go) fails with lint lines that are all `title:` length problems (the refusal line is `title: the title must be at most 50 characters, not N`, N being the title's length), Shape repairs the title at once, on any attempt, with `shortenTitle`, the way the late fallback does today. The shortened title is written to the proposed intent.md and kept (`keepProposals`), a `title_shortened` event with `original` and `title` is recorded, progress prints exactly one line `kogen: shortened the title to 50 characters: <title>` (no elapsed-time suffix), and `CheckIntent` runs again so the submission continues to the next gates in the same attempt. If the lint has any line that is not a `title:` line, or the title is still invalid after the repair, nothing new is repaired and the refusal text is unchanged, including every non-title line. Titles of 50 characters or fewer are unchanged. The late fallback (after repeated refusals) keeps working as it does and prints the same progress line when it shortens a title. The existing test `TestShapeShortensATooLongTitleOnlyOnTheLastAttempt` (shape/shape_test.go) asserts that a title-only refusal reaches the shaper at attempts 1 and 2; it is replaced by the new behavior and must be updated.

## Decisions
- The repair applies only when every lint line starts with `title:`; the existing late fallback condition is kept and one new condition (title-only lint on any attempt) is added in the files-submission path of `acceptFiles`. (evidence: shape/shape.go) (source: request)
- The progress line is printed with `bareProgress`, so the line is exactly `kogen: shortened the title to 50 characters: <title>` without the ` (N s)` suffix that `progress` adds, and it is printed wherever a title is shortened, including the late fallback. (evidence: shape/shape.go) (source: request)
- No new setting, flag or environment variable is added; 50 characters stays the fixed limit. (source: request)
- Repair title-only lint at once by adding a titleOnlyLint(err) condition to the existing late-fallback guard in acceptFiles, reusing the same shortenTitle block (decision: D1, round 1) (constraint: every lint line must start with title:) (constraint: mixed lint behaves as today) (evidence: shape/shape.go; shape/shape_test.go) (assumption: A scratch check showed the patches apply; the harness was not run end to end, so Kogen readiness is the first run of it.) (source: request)
- Print the progress line with bareProgress so no elapsed-time suffix follows it (decision: D2, round 1) (evidence: shape/shape.go) (assumption: Assumes the scripted fixture routes Progress output into Stderr.) (source: request)

## Source comparison
- Addressed: title-only lint (`title: the title must be at most 50 characters, not N`) repaired with `shortenTitle` on any attempt, kept, recorded as `title_shortened`, one progress line, same-attempt continuation after the `CheckIntent` call (scenario "A 53-character title is shortened at once").
- Addressed: a title line plus another lint line behaves as today, with both lines in the feedback on attempts 1 and 2 (scenario "A title with another lint problem is refused with both lines"); the late fallback still shortens on attempt 3 and records one `title_shortened`.
- Addressed: a 50-character title is unchanged (scenario "A 50-character title is unchanged").
- Addressed: the focused model-free regression test in the shape package, next to the existing title tests (see Tests).
- Superseded: the existing assertion that a title-only refusal is fed back at attempts 1 and 2 (`1:intent 2:intent`) in `TestShapeShortensATooLongTitleOnlyOnTheLastAttempt`.

## Tests
- A title-only lint failure is shortened at attempt 1, kept, recorded once as `title_shortened`, announced by one progress line, and not refused at the intent gate: new subtest in shape/shape_test.go with a scripted shaper and a 53-character title "Add an is-even check for integers and also reject all", asserting the stored intent.md title is the `shortenTitle` result (unit)
- A 53-character title plus one other lint problem is refused with both lines at attempts 1 and 2 as before: update the "another problem remains" subtest in shape/shape_test.go to assert both lines in the first two refusals (unit)
- A 50-character title is unchanged, with no event and no progress line: new subtest in shape/shape_test.go (unit)
- The late fallback still shortens a title after repeated refusals: existing `TestRetryTotalCapPreservesTitleFallback` in shape/submission_retry_test.go with its title now valid at once, so adapt it to still reach the fallback through another lint problem (existing)
- The old last-attempt-only expectation is replaced: update `TestShapeShortensATooLongTitleOnlyOnTheLastAttempt` in shape/shape_test.go (regression)
- End to end through native Shape: the acceptance scenarios run the shape package with a scripted shaper (scenario)

## Feasibility
- evidence: {"id":"title-only-observer","kind":"observer","inputs":["features/steps/shape-title-only-lint-fixed-in-place__steps.js","features/steps/shape-title-only-lint-fixed-in-place__harness.js"],"observers":["features/steps/shape-title-only-lint-fixed-in-place__steps.js"],"checks":[{"name":"good","command":"node features/steps/shape-title-only-lint-fixed-in-place__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/shape-title-only-lint-fixed-in-place__steps.js bad","exit":1,"output":"rejected\n"}]}
