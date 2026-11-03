Feature: Deliberately regenerate synthetic goldens through Go tests
  Scenario Outline: Regenerate recorded expectations rather than hand-merge them
    When I exercise golden regeneration for "<file>" using "<test>"
    Then the golden regeneration cycle succeeds

    Examples:
      | file | test |
      | shape/testdata/init/mix-golden.json | TestInitMixVerificationMatchesElixir |
      | shape/testdata/force_ordinary_golden.json | TestShapeForceWireCompatibility |
