Feature: Contract-check fixtures provide candidate proof
  Scenario: Native and external contract checks use current execution evidence
    When I run the existing contract correction and external Shape regressions with proof capture
    Then all captured candidate reports prove their exact expected steps
