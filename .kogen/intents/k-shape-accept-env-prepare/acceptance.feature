Feature: Shaped acceptance steps run on a prepared environment
  A shaped acceptance step that fails on the unchanged tree only because Kogen's fresh
  single-commit copy is not prepared for it goes back to the shaper with that fresh-copy
  output, instead of being offered as missing-feature red.

  Scenario: A step that reads a store only the shaping checkout has prepared is sent back
    When the environment journey runs with an acceptance step that depends on "store"
    Then the step comes back to the shaper with its fresh-copy output and its correction is published

  Scenario: A step that depends on Git history the fresh copy lacks is sent back
    When the environment journey runs with an acceptance step that depends on "history"
    Then the step comes back to the shaper with its fresh-copy output and its correction is published
