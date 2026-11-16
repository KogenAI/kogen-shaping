Feature: The kept assertion repeat outcome names the red scenarios
  This feature checks the progress line and the kept_assertion_repeat event that Shape writes after it repeats the readiness observation of a kept scenario.

  Scenario: One persistent kept red scenario is named
    Given shape-kept-repeat-outcome-names-scenarios: the kept assertion repeat has run on three local fixtures
    Then shape-kept-repeat-outcome-names-scenarios: the progress line for case "one" is 'kogen: kept assertion repeat outcome: persistent kept assertion red in "Admin accounts list links each account" (line 12) (cap 2 executions)'
    And shape-kept-repeat-outcome-names-scenarios: the repeat event for case "one" lists scenarios "Admin accounts list links each account@12" in that order

  Scenario: Persistent kept red scenarios are named in sorted order
    Given shape-kept-repeat-outcome-names-scenarios: the kept assertion repeat has run on three local fixtures
    Then shape-kept-repeat-outcome-names-scenarios: the progress line for case "two" is 'kogen: kept assertion repeat outcome: persistent kept assertion red in "Ant" (line 7), "Zebra" (line 3) (cap 2 executions)'
    And shape-kept-repeat-outcome-names-scenarios: the repeat event for case "two" lists scenarios "Ant@7, Zebra@3" in that order

  Scenario: No persistent kept red names no scenario
    Given shape-kept-repeat-outcome-names-scenarios: the kept assertion repeat has run on three local fixtures
    Then shape-kept-repeat-outcome-names-scenarios: the progress line for case "first-only" is "kogen: kept assertion repeat outcome: no persistent kept assertion red (cap 2 executions)"
    And shape-kept-repeat-outcome-names-scenarios: the repeat event for case "first-only" has no scenarios key
