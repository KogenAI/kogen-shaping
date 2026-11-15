Feature: A missing literal shows the nearest package text
  This feature covers the literal check feedback of Shape. A required literal that is missing
  from the package carries the text the model reported as the closest match, in the same
  package_value field that a changed literal uses. The status the check assigns, the headline and
  the line of a changed literal do not change.

  Background:
    Given shape-literal-missing-shows-near-package-text: the literal outcome of each case has been observed

  Scenario: A missing literal with a near package value shows it after the status
    Then shape-literal-missing-shows-near-package-text: the problem line of case "missing with near value" is:
      """
      literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: missing | package_value: /demo_two | why: w
      """

  Scenario: A missing literal without a near package value keeps the old line
    Then shape-literal-missing-shows-near-package-text: the problem line of case "missing without near value" is:
      """
      literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: missing | why: w
      """

  Scenario: A changed literal renders as before
    Then shape-literal-missing-shows-near-package-text: the problem line of case "changed" is:
      """
      literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: changed | package_value: /demo-one | why: w
      """

  Scenario: The near package value is redacted
    Then shape-literal-missing-shows-near-package-text: the problem line of case "missing with secret near value" is:
      """
      literal: /demo-two | source: request | source_quote: "Register /demo_one and /demo-two." | status: missing | package_value: near [REDACTED] | why: w
      """

  Scenario: The headline and the statuses are unchanged
    Then shape-literal-missing-shows-near-package-text: the headline of case "missing with near value" is:
      """
      Literal check: 1 required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:
      """
    And shape-literal-missing-shows-near-package-text: the status of case "missing with near value" is "missing"
    And shape-literal-missing-shows-near-package-text: the status of case "changed" is "changed"
