Feature: Shape splits work that cannot fit one review before any Build starts
  Before an Intent is ready, Shape estimates the mandatory review material of its scope: the
  Intent package, the review inputs (README.md and the testing rules) and the changed files the
  Intent names. Above the review ceiling, tools.InputBudgetBytes, the candidate is refused and the
  shaper splits the work into separately reviewable Intents, each part under the ceiling. An
  Intent under the ceiling is approved as before. The estimate and the ceiling appear in the
  shape output.

  Scenario: An Intent under the ceiling is approved as before and shows its estimate and the ceiling
    When k-shape-split-oversized-review: the "fits" case is shaped
    Then k-shape-split-oversized-review: the run used a review ceiling of 524288 bytes
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 1 against the ceiling
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 2 against the ceiling
    And k-shape-split-oversized-review: the progress shows no split
    And k-shape-split-oversized-review: the result of submission 1 does not contain "Scope check"

  Scenario: An Intent over the ceiling is refused with its estimate and ceiling, and its split into parts under the ceiling is approved
    When k-shape-split-oversized-review: the "oversized" case is shaped
    Then k-shape-split-oversized-review: the run used a review ceiling of 524288 bytes
    And k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the first line of the result of submission 1 is the Intent line for 2 changed files in 1 areas
    And k-shape-split-oversized-review: the result of submission 1 lists the review inputs and the Intent package
    And k-shape-split-oversized-review: the result of submission 1 contains "- engine/core.js: 300000 bytes (changed file, area engine)"
    And k-shape-split-oversized-review: the result of submission 1 contains "- engine/rules.js: 250000 bytes (changed file, area engine)"
    And k-shape-split-oversized-review: the item lines of the result of submission 1 add up to the estimate of each unit
    And k-shape-split-oversized-review: the result of submission 1 tells the shaper to split the work into separately reviewable Intents
    And k-shape-split-oversized-review: no decision check ran before the second shaper request
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 1 against the ceiling
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 2 against the ceiling
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the published intent.md contains "## Split"
    And k-shape-split-oversized-review: the published intent.md contains "- engine-rules: "
    And k-shape-split-oversized-review: the progress shows the split into the parts "engine-rules"
    And k-shape-split-oversized-review: the scope check refused 1 submission

  Scenario: A split part that is itself over the ceiling is refused, and parts that each fit are approved
    When k-shape-split-oversized-review: the "part-over" case is shaped
    Then k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the result of submission 1 has the part line for "engine-all" with 2 changed files in 1 areas
    And k-shape-split-oversized-review: the result of submission 1 does not refuse the Intent itself
    And k-shape-split-oversized-review: the item lines of the result of submission 1 add up to the estimate of each unit
    And k-shape-split-oversized-review: the progress does not name the part "engine-all"
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the progress shows the split into the parts "engine-core,engine-rules"

  Scenario: A staged plan does not make an Intent over the ceiling fit
    When k-shape-split-oversized-review: the "stages" case is shaped
    Then k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the first line of the result of submission 1 is the Intent line for 2 changed files in 1 areas
    And k-shape-split-oversized-review: the result of submission 1 contains "A ## Stages plan does not lower the estimate"
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the progress shows the split into the parts "engine-rules"

  Scenario: An estimate equal to the ceiling fits and one byte more is refused
    When k-shape-split-oversized-review: the "edge" case is shaped
    Then k-shape-split-oversized-review: the run used a review ceiling of 524288 bytes
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 1 as 524289 bytes
    And k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 2 as 524288 bytes
    And k-shape-split-oversized-review: the scope check refused 1 submission
    And k-shape-split-oversized-review: the decision check ran at least 1 time
    And k-shape-split-oversized-review: no decision check ran before the second shaper request
