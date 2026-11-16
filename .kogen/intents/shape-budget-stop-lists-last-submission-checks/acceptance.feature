Feature: The shaping budget refusal lists the last submission's checks

  This feature covers a shaping that ends in the budget refusal after a submission that was checked
  but not refused by a gate.

  Scenario: The budget refusal lists the last submission whose checks ended past the budget
    Given shape-budget-stop-lists-last-submission-checks: a scripted shaping whose second submission passes the cheap gates and whose checks end 60 s past the budget has stopped
    Then shape-budget-stop-lists-last-submission-checks: the refusal lists the line "attempt 1: gate validation, model request 600 s, Kogen checks 0 s, 300 s remaining"
    Then shape-budget-stop-lists-last-submission-checks: the refusal lists the line "attempt 2: gate contract, model request 240 s, Kogen checks 120 s, 0 s remaining, 60 s over"
    Then shape-budget-stop-lists-last-submission-checks: the refusal lists 2 attempt lines
