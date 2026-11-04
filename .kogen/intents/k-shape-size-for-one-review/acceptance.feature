Feature: Shape refuses an Intent too large for one review before any Build
  Before an Intent is ready, Shape estimates the bytes one review of it needs from what the Intent
  states: its own package (intent.md, the acceptance feature, its harnesses), and every repository
  file it names in backticks. Above the review stage's bound, tools.InputBudgetBytes, the candidate
  is refused with a line naming the counts and the bound, and the shaper splits the work into an
  ordered staged plan whose stages each fit, or narrows it to the requested behaviour. An Intent
  inside the bound keeps today's route and model requests.

  Scenario: An Intent inside the bound keeps today's route with no scope line and no extra model request
    When k-shape-size-for-one-review: the "small" case is shaped
    Then k-shape-size-for-one-review: the run used a review bound of 524288 bytes
    And k-shape-size-for-one-review: the Intent is published
    And k-shape-size-for-one-review: the result of submission "c1" does not contain "Scope check"
    And k-shape-size-for-one-review: the result of submission "c1" contains "Readiness observations"
    And k-shape-size-for-one-review: the published acceptance.feature has 3 scenarios
    And k-shape-size-for-one-review: the run made 2 shaper, 2 decision check and 1 contract check requests

  Scenario: An Intent of 40 scenarios in 12 areas is refused naming its counts and the bound, and its staged plan is accepted
    When k-shape-size-for-one-review: the "wide" case is shaped
    Then k-shape-size-for-one-review: the run used a review bound of 524288 bytes
    And k-shape-size-for-one-review: submission "c1" is refused by the scope check
    And k-shape-size-for-one-review: a line of the result of submission "c1" contains "Scope check: the Intent is estimated at", "(scenarios 40, acceptance harnesses 1, named files 12 in areas 12)" and "over the 524288-byte bound of one review (tools.InputBudgetBytes)"
    And k-shape-size-for-one-review: the result of submission "c1" contains "- alpha/module.js: 48000 bytes (named file, area alpha)"
    And k-shape-size-for-one-review: the result of submission "c1" contains "- lima/module.js: 48000 bytes (named file, area lima)"
    And k-shape-size-for-one-review: the result of submission "c1" contains "- features/steps/is-even__steps.js: "
    And k-shape-size-for-one-review: the result of submission "c1" contains "split the work into an ordered ## Stages plan"
    And k-shape-size-for-one-review: the result of submission "c1" contains "narrow the Intent to the requested behaviour"
    And k-shape-size-for-one-review: the result of submission "c2" does not contain "Scope check"
    And k-shape-size-for-one-review: the Intent is published
    And k-shape-size-for-one-review: the published intent.md contains "## Stages"
    And k-shape-size-for-one-review: the published acceptance.feature has 40 scenarios
    And k-shape-size-for-one-review: the run made 3 shaper, 2 decision check and 1 contract check requests

  Scenario: An over-scoped Intent narrowed to the requested behaviour is accepted
    When k-shape-size-for-one-review: the "narrow" case is shaped
    Then k-shape-size-for-one-review: submission "c1" is refused by the scope check
    And k-shape-size-for-one-review: a line of the result of submission "c1" contains "Scope check: the Intent is estimated at", "(scenarios 40, acceptance harnesses 1, named files 12 in areas 12)" and "over the 524288-byte bound of one review (tools.InputBudgetBytes)"
    And k-shape-size-for-one-review: the result of submission "c2" does not contain "Scope check"
    And k-shape-size-for-one-review: the Intent is published
    And k-shape-size-for-one-review: the published intent.md does not contain "## Stages"
    And k-shape-size-for-one-review: the published acceptance.feature has 6 scenarios
    And k-shape-size-for-one-review: the published acceptance.feature contains "Scenario: bravo behaviour 6"

  Scenario: A request that needs every behaviour is staged with nothing dropped, and deleting a requested example is refused
    When k-shape-size-for-one-review: the "every" case is shaped
    Then k-shape-size-for-one-review: submission "c1" is refused by the scope check
    And k-shape-size-for-one-review: the result of submission "c1" contains "Do not satisfy the bound by deleting requested behaviour"
    And k-shape-size-for-one-review: the result of submission "c2" does not contain "Scope check"
    And k-shape-size-for-one-review: a line of the result of submission "c2" contains "lima behaviour 36 returns built 36", "problem: no scenario exercises it" and "confirmed: synthetic confirmation"
    And k-shape-size-for-one-review: the result of submission "c3" does not contain "Scope check"
    And k-shape-size-for-one-review: the Intent is published
    And k-shape-size-for-one-review: the published intent.md contains "## Stages"
    And k-shape-size-for-one-review: the published acceptance.feature has 40 scenarios
    And k-shape-size-for-one-review: the published acceptance.feature contains "Scenario: lima behaviour 36"
    And k-shape-size-for-one-review: the run made 4 shaper, 3 decision check and 3 contract check requests

  Scenario: The bound is the review stage's tools.InputBudgetBytes, and changing it changes the verdict
    When k-shape-size-for-one-review: the "wide-unstaged" case is shaped
    Then k-shape-size-for-one-review: the run used a review bound of 524288 bytes
    And k-shape-size-for-one-review: submission "c1" is refused by the scope check
    And k-shape-size-for-one-review: the Intent is not published
    When k-shape-size-for-one-review: the "wide-unstaged" case is shaped with tools.InputBudgetBytes set to 1048576
    Then k-shape-size-for-one-review: the run used a review bound of 1048576 bytes
    And k-shape-size-for-one-review: the result of submission "c1" does not contain "Scope check"
    And k-shape-size-for-one-review: the Intent is published
    When k-shape-size-for-one-review: the "heavy" case is shaped with tools.InputBudgetBytes set to 1048576
    Then k-shape-size-for-one-review: the run used a review bound of 1048576 bytes
    And k-shape-size-for-one-review: submission "c1" is refused by the scope check
    And k-shape-size-for-one-review: a line of the result of submission "c1" contains "Scope check: the Intent is estimated at", "named files 12 in areas 12" and "over the 1048576-byte bound of one review (tools.InputBudgetBytes)"
    And k-shape-size-for-one-review: the result of submission "c1" contains "- alpha/module.js: 90000 bytes (named file, area alpha)"
    And k-shape-size-for-one-review: the result of submission "c2" does not contain "Scope check"
    And k-shape-size-for-one-review: the Intent is published
