Feature: A required literal that the request soft-wrapped is carried by the package's one-space text
  This feature checks the literal check of Shape against a literal that the request wrote across a
  soft-wrapped line. Such a literal counts as present when the package carries it with each line
  break and its surrounding spaces replaced by one space. Everything else is still compared byte
  for byte.

  Background:
    Given shape-literal-soft-wrapped-request-literal: the literal check harness has run

  Scenario: A soft-wrapped literal is present when the package has it on one line
    Then shape-literal-soft-wrapped-request-literal: the observation of "soft wrap" includes:
      """
      {"status": "present", "literal": "50 cards\nleft", "present": 1, "missing": 0, "changed": 0, "lines": []}
      """
    And shape-literal-soft-wrapped-request-literal: the observation of "soft wrap with indent" includes:
      """
      {"status": "present", "literal": "50 cards\n    left", "present": 1, "missing": 0, "changed": 0, "lines": []}
      """

  Scenario: A literal that the package carries with its line break byte for byte is still present
    Then shape-literal-soft-wrapped-request-literal: the observation of "exact bytes" includes:
      """
      {"status": "present", "present": 1, "lines": []}
      """

  Scenario: A package that changes another character of the literal is still refused
    Then shape-literal-soft-wrapped-request-literal: the observation of "changed text" includes:
      """
      {"status": "changed", "present": 0, "changed": 1, "lines": ["Literal check: 1 required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:", "literal: 50 cards\nleft | source: request | source_quote: \"50 cards\nleft\" | status: changed | package_value: 50 cards, left | why: w"]}
      """
    And shape-literal-soft-wrapped-request-literal: the observation of "missing text" includes:
      """
      {"status": "missing", "present": 0, "missing": 1, "lines": ["Literal check: 1 required literal values are not carried exactly by the package; copy each value byte for byte into the Intent or its acceptance where the Intent uses it (descriptive text may stay paraphrased), or restore the substituted value:", "literal: 50 cards\nleft | source: request | source_quote: \"50 cards\nleft\" | status: missing | why: w"]}
      """

  Scenario: A literal split by a blank line is still compared byte for byte
    Then shape-literal-soft-wrapped-request-literal: the observation of "blank line" includes:
      """
      {"status": "missing", "present": 0, "missing": 1}
      """

  Scenario: A literal without a line break is still compared byte for byte
    Then shape-literal-soft-wrapped-request-literal: the observation of "no line break" includes:
      """
      {"status": "missing", "present": 0, "missing": 1}
      """
