# Report blocked kept steps in one refusal line

Slug: shape-kept-blocked-lines-one-refusal
Model: not applicable — a pure function from two observations to refusal lines, with no state
Acceptance:
- .kogen/intents/shape-kept-blocked-lines-one-refusal/acceptance.feature
- features/steps/shape-kept-blocked-lines-one-refusal__steps.js
- features/steps/shape-kept-blocked-lines-one-refusal__driver/harness_test.go.txt

When several kept steps of one scenario never run on the unchanged tree because an earlier step failed, the contract review wrote one refusal line per kept step, all with the same long failure text, which fills the 20-line refusal cap. Kept steps that never run are now reported in one refusal line per scenario, property and failure text. The line is the existing blocked-step sentence with the single word `line` replaced by `lines` and the numbers listed in ascending order separated by a comma and a space, the last two joined by the word `and` (for two numbers: the two numbers joined by `and`). A single blocked step keeps the sentence with `line` and its number exactly as today. A red kept step keeps its own line as today. Lines for different scenarios, different properties or different failure texts stay separate, and the refusals stay sorted as today. The change is confined to `shape/kept_baseline.go` and `shape/kept_baseline_test.go`; the red-step wording, the property and step matching, the prompts, the docs and the 20-line and 300-character cuts are unchanged. No setting, flag or output is added.

## Decisions
- Blocked kept steps (never run in both unchanged-tree observations) are grouped by property name, scenario and failure text, and each group gives exactly one refusal line, so three blocked steps on lines 3, 4 and 5 of one property give one line containing `lines 3, 4 and 5` and the failure text once. (decision: D1, round 1) (constraint: no new setting, flag or output) (evidence: shape/kept_baseline.go; shape/kept_baseline_test.go) (source: request)
- The grouped line is the existing blocked-step sentence with `line %d` replaced by `lines` and the ascending numbers joined with `, ` and the last two with ` and `; two numbers read `lines 4 and 5`; a single blocked step keeps `line 5` exactly as today, never `lines 5`. (decision: D2, round 1) (constraint: numbers ascending whatever the order in which the property names them) (evidence: shape/kept_baseline.go) (source: request)
- Only steps blocked in both observations are listed; a step red or blocked in one observation only gives no refusal, as today. (decision: D3, round 1) (evidence: shape/kept_baseline.go; shape/kept_baseline_test.go) (source: request)
- A red kept step keeps its own refusal line with its own line number and is never part of a group; the refusals stay sorted as today. (decision: D4, round 1) (evidence: shape/kept_baseline.go) (source: request)
- The grouped sentence keeps the existing grammar (`lines 3, 4 and 5 is meant to assert ...`) because the request replaces only the single word `line`. (decision: D5, round 1)  (evidence: shape/kept_baseline.go) (assumption: the user accepts the plural noun followed by the unchanged verb `is`) (source: request)
- Acceptance drives the unexported `keptStepStates` and `keptBaselineRefusals` with a Go test harness copied in through a Go overlay, because the behaviour is internal to the shape package. (decision: D6, round 1) (evidence: shape/kept_baseline.go; shape/kept_baseline_test.go) (source: request)

## Source comparison
- Three blocked kept steps give one refusal with `lines 3, 4 and 5` and the failure text once: addressed by the scenario "Three blocked kept steps of one property give one refusal" and its wrong implementation emitting one line per step.
- Two blocked steps named in the order 5 then 4 give `lines 4 and 5`, and a step that ran in the second observation is not listed: addressed by the scenario "Two blocked kept steps are listed in ascending order and a step that ran is left out" and its wrong implementations (comma join, descending order, first-observation-only).
- Unchanged kept behaviour (single blocked step keeps `line 5`, a red step alone keeps its line, one-observation steps give no refusal): addressed by the scenarios "A single blocked kept step keeps the sentence with line", "A red kept step alone keeps its own line" and "Kept steps that are red or blocked in one observation only give no refusal"; wrong implementations write `lines 5` for one step or refuse a step seen once.
- Different properties and scenarios stay separate: addressed by the scenario "Different properties and different scenarios stay in separate refusals" and the wrong implementation grouping by scenario alone.
- A red step is not folded into the group: addressed by the scenario "A red kept step is not grouped with the blocked kept steps after it" and the wrong implementation that folds it in.
- Out of scope and unchanged: shape/prompts, docs, README, the red-step wording, the property and step matching, the 20-line and 300-character cuts.

## Tests
- Blocked kept steps of one scenario, property and failure text give one refusal listing their lines in ascending order: a unit test in shape/kept_baseline_test.go beside TestKeptBaselineRefusalCases, plus the scenarios "Three blocked kept steps of one property give one refusal" and "Two blocked kept steps are listed in ascending order and a step that ran is left out" (unit)
- A single blocked step, a red step alone and a one-observation step keep today's lines and refusals: the existing TestKeptBaselineRefusalCases, plus the scenarios for the single step, the red step and one observation only (existing)
- Different properties and scenarios stay separate and a red step is not grouped: a unit test in shape/kept_baseline_test.go, plus the scenarios "Different properties and different scenarios stay in separate refusals" and "A red kept step is not grouped with the blocked kept steps after it" (unit)

## Feasibility
- none
