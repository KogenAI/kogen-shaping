Feature: The runner-observer refusal shows the failing step's message
  This feature covers the runner-observer refusal of Shape. When a step failed although the test
  runner reported a passing run, the refusal item ends with the first non-blank line of the failing
  step's message, so the shaper can see which check rejected the runner's real output. The item of
  an anchored regular expression used as a literal name filter stays as it is.

  Background:
    Given shape-runner-observer-refusal-shows-step-failure: the runner-observer refusal harness has run

  Scenario: A passing-run item ends with the first line of the step message
    Then shape-runner-observer-refusal-shows-step-failure: the only item of "passing" is "- Even numbers (line 2): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips; step message: assertion failed in scenario Even numbers"

  Scenario: A multi-line message contributes only its first non-blank line
    Then shape-runner-observer-refusal-shows-step-failure: the only item of "blank-lead" is "- Even numbers (line 2): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips; step message: first real line"

  Scenario: A first line longer than 200 runes is cut at 200 runes
    Then shape-runner-observer-refusal-shows-step-failure: the only item of "long" is "- Even numbers (line 2): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips; step message: " followed by 200 "é" characters

  Scenario: A first line of exactly 200 runes is kept whole
    Then shape-runner-observer-refusal-shows-step-failure: the only item of "exact" is "- Even numbers (line 2): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips; step message: " followed by 200 "a" characters

  Scenario: A scenario without a detail gets the item exactly as before
    Then shape-runner-observer-refusal-shows-step-failure: the only item of "nodetail" is "- Even numbers (line 2): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips"

  Scenario: The literal-filter item is byte-identical even when the step has a detail
    Then shape-runner-observer-refusal-shows-step-failure: the only item of "literal" is "- Odd numbers (line 5): the name filter -n ^test_widget_passes$ is a regular expression that the runner matched as a literal test name, so nothing ran: 0 runs, 0 assertions, 0 failures, 0 errors, 0 skips"

  Scenario: Both kinds in one refusal keep their order and each its own form
    Then shape-runner-observer-refusal-shows-step-failure: the items of "mixed" are:
      """
      ["- Even numbers (line 2): the step failed although the test runner reported a passing run: 1 runs, 3 assertions, 0 failures, 0 errors, 0 skips; step message: first line", "- Odd numbers (line 5): the name filter -n ^test_widget_passes$ is a regular expression that the runner matched as a literal test name, so nothing ran: 0 runs, 0 assertions, 0 failures, 0 errors, 0 skips"]
      """
