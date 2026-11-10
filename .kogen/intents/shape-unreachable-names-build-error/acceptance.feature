Feature: Report useful bounded controls failures
  This feature preserves the compiler diagnostic hidden on 10 November 2026.

  Scenario: old unreachable
    When shape-unreachable-names-build-error: the failure message is:
      """
      go build -o /x/driver ./features/steps/s__driver/main.go:
      # github.com/KogenAI/kogen-review/review
      review/independent.go:771:6: undefined: required
      """
    Then shape-unreachable-names-build-error: the reported reason matches
      """
      go build -o /x/driver ./features/steps/s__driver/main.go: | review/independent.go:771:6: undefined: required | # github.com/KogenAI/kogen-review/review
      """

  Scenario: old wrong reason
    When shape-unreachable-names-build-error: the failure message is:
      """
      go build -o /x/driver ./features/steps/s__driver/main.go:
      # github.com/KogenAI/kogen-review/review
      review/independent.go:771:6: undefined: required
      """
    Then shape-unreachable-names-build-error: the reported reason matches
      """
      go build -o /x/driver ./features/steps/s__driver/main.go: | review/independent.go:771:6: undefined: required | # github.com/KogenAI/kogen-review/review
      """

  Scenario: no location preference
    When shape-unreachable-names-build-error: the failure message is:
      """
      go build -o /x/driver ./features/steps/s__driver/main.go:
      # github.com/KogenAI/kogen-review/review
      review/independent.go:771:6: undefined: required
      """
    Then shape-unreachable-names-build-error: the reported reason matches
      """
      go build -o /x/driver ./features/steps/s__driver/main.go: | review/independent.go:771:6: undefined: required | # github.com/KogenAI/kogen-review/review
      """

  Scenario: truncate assertion
    When shape-unreachable-names-build-error: the failure message is:
      """
      expected true
      """
    Then shape-unreachable-names-build-error: the reported reason matches
      """
      expected true
      """

  Scenario: no extra limit
    When shape-unreachable-names-build-error: the failure has 50 following lines of 1 letters
    Then shape-unreachable-names-build-error: the report has 3 extra lines and 20 characters

  Scenario: no character limit
    When shape-unreachable-names-build-error: the failure has 50 following lines of 130 letters
    Then shape-unreachable-names-build-error: the report has 3 extra lines and 400 characters

  Scenario Outline: character boundary
    When shape-unreachable-names-build-error: the failure has 50 following lines of <width> letters
    Then shape-unreachable-names-build-error: the report has 3 extra lines and <total> characters
    Examples:
      | width | total |
      | 127   | 398   |
      | 128   | 400   |
      | 129   | 400   |
