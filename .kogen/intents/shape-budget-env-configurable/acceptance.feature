Feature: The shaping budget is settable for one run through the environment
  This feature checks real shapings run with a fake clock and a fake provider, with KOGEN_SHAPE_BUDGET_MINUTES in the session environment.

  Scenario: Unset or empty keeps the 15-minute budget
    Given shape-budget-env-configurable: shapings have run in one process with the budget variable unset, empty, 25 and other values
    Then shape-budget-env-configurable: with the variable unset the time line after 10 seconds is "Time: 0 min 10 s elapsed, 14 min 50 s remaining of the 15-minute shaping budget."
    And shape-budget-env-configurable: with the variable empty the time line after 10 seconds is "Time: 0 min 10 s elapsed, 14 min 50 s remaining of the 15-minute shaping budget."
    And shape-budget-env-configurable: with the variable unset budget_seconds is 900
    And shape-budget-env-configurable: with the variable empty budget_seconds is 900
    And shape-budget-env-configurable: with the variable unset the budget is "not spent" at 899 seconds
    And shape-budget-env-configurable: with the variable unset the budget is "spent" at 900 seconds

  Scenario: 25 sets the budget in the text, the event and the spent check
    Given shape-budget-env-configurable: shapings have run in one process with the budget variable unset, empty, 25 and other values
    Then shape-budget-env-configurable: with the variable 25 the time line after 10 seconds is "Time: 0 min 10 s elapsed, 24 min 50 s remaining of the 25-minute shaping budget."
    And shape-budget-env-configurable: with the variable 25 budget_seconds is 1500
    And shape-budget-env-configurable: with the variable 25 the budget is "not spent" at 900 seconds
    And shape-budget-env-configurable: with the variable 25 the budget is "not spent" at 1499 seconds
    And shape-budget-env-configurable: with the variable 25 the budget is "spent" at 1500 seconds

  Scenario: A shaping that runs out of a 25-minute budget says 25 minutes
    Given shape-budget-env-configurable: shapings have run in one process with the budget variable unset, empty, 25 and other values
    Then shape-budget-env-configurable: the shaping that runs out of budget with the variable 25 ends with a detail starting "the shaper used its 25-minute shaping budget"
    And shape-budget-env-configurable: the first problem line of that shaping that states the budget starts with "shaping budget: 1500 s (25 min"
    And shape-budget-env-configurable: that shaping lists the attempt line "attempt 1: gate validation, model request 1200 s, Kogen checks 0 s, 300 s remaining"
    And shape-budget-env-configurable: that shaping lists the attempt line "attempt 2: gate contract, model request 240 s, Kogen checks 120 s, 0 s remaining, 60 s over"

  Scenario Outline: A value outside 5 to 45 or not a whole number is refused
    Given shape-budget-env-configurable: shapings have run in one process with the budget variable unset, empty, 25 and other values
    Then shape-budget-env-configurable: with the variable "<value>" Shape is refused with exit code 3 and class environment before any provider request
    And shape-budget-env-configurable: with the variable "<value>" the refusal says "shape budget must be a whole number of minutes from 5 to 45"

    Examples:
      | value |
      | 46    |
      | 4     |
      | 0     |
      | -5    |
      | abc   |
      | 25.5  |

  Scenario Outline: The bounds 5 and 45 are accepted
    Given shape-budget-env-configurable: shapings have run in one process with the budget variable unset, empty, 25 and other values
    Then shape-budget-env-configurable: with the variable "<value>" Shape runs without refusal and its time line says "<text>"

    Examples:
      | value | text                            |
      | 5     | of the 5-minute shaping budget  |
      | 45    | of the 45-minute shaping budget |

  Scenario: Two shapings in one process do not share the value
    Given shape-budget-env-configurable: shapings have run in one process with the budget variable unset, empty, 25 and other values
    Then shape-budget-env-configurable: with the variable 25 the time line after 10 seconds is "Time: 0 min 10 s elapsed, 24 min 50 s remaining of the 25-minute shaping budget."
    And shape-budget-env-configurable: with the variable unset the time line after 10 seconds is "Time: 0 min 10 s elapsed, 14 min 50 s remaining of the 15-minute shaping budget."
