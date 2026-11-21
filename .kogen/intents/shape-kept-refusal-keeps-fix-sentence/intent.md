# Keep the fix sentence in a cut kept-step refusal

Slug: shape-kept-refusal-keeps-fix-sentence
Model: not applicable — a pure string transformation of one refusal line, with no state
Acceptance:
- .kogen/intents/shape-kept-refusal-keeps-fix-sentence/acceptance.feature
- features/steps/shape-kept-refusal-keeps-fix-sentence__steps.js
- features/steps/shape-kept-refusal-keeps-fix-sentence__driver/harness_test.go.txt

When Shape records a refusal, each problem line is cut to 300 characters. For a kept-step refusal line of the form `- <property> (kept) | request: <quoted request text> | problem: <problem and fix sentence>`, the cut currently removes the end of the line, so a long quoted request text pushes the problem and its fix sentence out and the shaper sees a line such as `- Existing tool exports stay published and free of sponsors (kept) | request: Do not add a live sponsor to tool exports or change rankings. | problem:` ending in an empty `problem:`. The cut now shortens the quoted request text first, marking the cut with the existing ` ... ` marker, and keeps the problem and fix sentence whole. A line of at most 300 characters is unchanged byte for byte, and no gate decision changes. This adds no setting, flag or output.

## Decisions
- A refusal line over 300 characters that has a quoted request text followed by ` | problem: ` has only the request text shortened, with the existing ` ... ` cut marker, so that the line is 300 characters long, and the problem and everything after it stay whole. (evidence: shape/checkorder.go; shape/controls.go; shape/checkorder_cut_test.go) (source: request)
- A line of at most 300 characters is returned byte-identical to today's, and a line without a `| request:` quote followed by `| problem:` keeps today's cut (the ` | why: ` rule or a plain cut at 300 characters). (evidence: shape/checkorder.go; shape/checkorder_cut_test.go) (source: request)
- The fix lives in `cutProblems`, which cuts every recorded refusal line, because the `(kept) | request: ... | problem: ...` lines are built by `controlsPropertyLine` and by the property item line and reach `cutProblems` unchanged, while `keptBaselineRefusals` builds lines that carry no quoted request text; `keptBaselineRefusals` therefore stays as it is. (evidence: shape/kept_baseline.go; shape/controls.go; shape/checkorder.go) (assumption: the request names keptBaselineRefusals, whose lines carry no `| request:` quote; the quoted-request lines the request describes are the property lines, and the single cut that all of them pass through is where they are fixed) (source: request)
- A focused model-free unit test is added in the shape package next to shape/kept_baseline_test.go: a kept property whose request text is over 300 runes yields a cut line that still contains its whole problem and fix sentence, and a short one equals the uncut line. (evidence: shape/kept_baseline_test.go; shape/checkorder_cut_test.go) (source: request)
- A refusal line over 300 characters with a quoted request text followed by ' | problem: ' has only the request text shortened with the existing ' ... ' marker; the problem and what follows stay whole; lines of at most 300 characters are byte-identical. (decision: D1, round 1) (constraint: no new configuration) (constraint: cutProblems bounds lines without a request/problem structure as today) (evidence: shape/checkorder.go; shape/controls.go; shape/checkorder_cut_test.go; shape/kept_baseline.go; shape/kept_baseline_test.go) (assumption: keptBaselineRefusals lines carry no quoted request text; the fix is placed in cutProblems, which all property lines pass through. Scratch probe on a copy: correct passed both scenarios, both wrong variants failed their named steps.) (source: request)

## Source comparison
- Long kept-step line keeps its problem and fix sentence, with the request text shortened by the cut marker: addressed by acceptance scenario "A long request text is shortened and the problem and fix sentence stay whole" and its wrong implementation, which cuts the whole line at 300 runes.
- Short kept-step line is byte-identical to today's: addressed by acceptance scenario "A short refusal line is unchanged" and its wrong implementation, which puts the problem before the request text for every line.
- No new configuration, and no change to the gate decisions: no step or test of this Intent changes a gate.

## Tests
- A kept-step line over 300 runes keeps its whole problem and fix sentence, with the request text shortened by the cut marker: a unit test in shape beside kept_baseline_test.go, plus the scenario "A long request text is shortened and the problem and fix sentence stay whole" (unit)
- A short kept-step line is byte-identical to today's: a unit test in the same place, plus the scenario "A short refusal line is unchanged" (unit)
- Existing cut rules (the ` | why: ` rule, the plain cut at 300 runes, the 20-line limit) stay as they are: the existing checkorder_cut_test.go tests (existing)

## Feasibility
- none
