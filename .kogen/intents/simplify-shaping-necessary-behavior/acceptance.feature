Feature: Shape the simplest necessary behavior
  This feature checks the Shaper policy text that the models receive.

  Scenario: A fixed request yields one behavior with no configuration
    Then simplify-shaping-necessary-behavior: the shaper policy contains "demonstrated variability need"
    And simplify-shaping-necessary-behavior: the shaper policy contains "a fixed value is a constant"
    And simplify-shaping-necessary-behavior: the shaper policy contains "no setting, flag or environment variable"

  Scenario: A genuine variability requirement is still supported
    Then simplify-shaping-necessary-behavior: the shaper policy contains "when the request or the current code demonstrates a variability need that must stay supported"

  Scenario: Old forms are removed completely
    Then simplify-shaping-necessary-behavior: no shape source or prompt contains "an actual variability need that must stay supported"

  Scenario: Feasible reuse or correction is not refused by a stack-specific rule
    Then simplify-shaping-necessary-behavior: the shaper policy contains "reuse an unchanged step file or helper that this Intent owns"
    And simplify-shaping-necessary-behavior: the shaper policy contains "only a collision with another Intent's files or a protected path is refused"
    And simplify-shaping-necessary-behavior: the shaper policy contains "review estimates are advisory"
