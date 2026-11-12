Feature: Shape splits work that cannot fit one review before any Build starts
  Before an Intent is ready, Shape estimates the mandatory review material of its scope: the
  Intent package and the review inputs (README.md and the testing rules), which every review part
  carries whole. Changed files are not counted, because the review cuts them into parts; but an
  Intent whose estimated number of review parts is over MaxReviewParts (each part is one review
  model call) is refused too. Above the limit, the review ceiling tools.InputBudgetBytes less the
  6000 bytes the review keeps for source, the candidate is refused and the shaper shrinks the
  Intent package or splits the work into separately reviewable Intents. An Intent within both
  bounds is approved as before. The estimate and the limit appear in the shape output.

  Scenario: An Intent within the limit is approved as before and shows its estimate and the limit
    When k-shape-split-oversized-review: the "fits" case is shaped
    Then k-shape-split-oversized-review: the run used a review ceiling of 524288 bytes
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 1 against the limit
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 2 against the limit
    And k-shape-split-oversized-review: the progress shows no split
    And k-shape-split-oversized-review: the result of submission 1 does not contain "Scope check"

  Scenario: An Intent over the limit is refused with its estimate and limit, and its split into parts within the limit is approved
    When k-shape-split-oversized-review: the "oversized" case is shaped
    Then k-shape-split-oversized-review: the run used a review ceiling of 524288 bytes
    And k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the first line of the result of submission 1 is the Intent line for 2 changed files in 1 areas
    And k-shape-split-oversized-review: the result of submission 1 lists the review inputs and the Intent package
    And k-shape-split-oversized-review: the result of submission 1 contains "- engine/core.js: 300000 bytes (changed file, area engine; review sends its diff in parts)"
    And k-shape-split-oversized-review: the result of submission 1 contains "- engine/rules.js: 250000 bytes (changed file, area engine; review sends its diff in parts)"
    And k-shape-split-oversized-review: the item lines of the result of submission 1 add up to the estimate of each unit
    And k-shape-split-oversized-review: the result of submission 1 tells the shaper to shrink the Intent package or split it into parts
    And k-shape-split-oversized-review: no decision check ran before the second shaper request
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 1 against the limit
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 2 against the limit
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the published intent.md contains "## Split"
    And k-shape-split-oversized-review: the published intent.md contains "- engine-rules: "
    And k-shape-split-oversized-review: the progress shows the split into the parts "engine-rules"
    And k-shape-split-oversized-review: the scope check refused 1 submission

  Scenario: An Intent that needs more review parts than MaxReviewParts is refused, and parts that each need fewer are approved
    When k-shape-split-oversized-review: the "many-parts" case is shaped
    Then k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the first line of the result of submission 1 is the review parts line for 11 parts
    And k-shape-split-oversized-review: the result of submission 1 tells the shaper to split the work over fewer review parts
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the progress shows the split into the parts "engine-core,engine-rules"

  Scenario: A staged plan does not make an Intent over the limit fit
    When k-shape-split-oversized-review: the "stages" case is shaped
    Then k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the first line of the result of submission 1 is the Intent line for 2 changed files in 1 areas
    And k-shape-split-oversized-review: the result of submission 1 contains "A ## Stages plan does not lower the estimate"
    And k-shape-split-oversized-review: the Intent is published
    And k-shape-split-oversized-review: the progress shows the split into the parts "engine-rules"

  Scenario: An estimate equal to the limit fits and one byte more is refused
    When k-shape-split-oversized-review: the "edge" case is shaped
    Then k-shape-split-oversized-review: the run used a review ceiling of 524288 bytes
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 1 as 518289 bytes
    And k-shape-split-oversized-review: submission 1 is refused by the scope check
    And k-shape-split-oversized-review: the progress shows the review estimate of submission 2 as 518288 bytes
    And k-shape-split-oversized-review: the scope check refused 1 submission
    And k-shape-split-oversized-review: the decision check ran at least 1 time
    And k-shape-split-oversized-review: no decision check ran before the second shaper request
