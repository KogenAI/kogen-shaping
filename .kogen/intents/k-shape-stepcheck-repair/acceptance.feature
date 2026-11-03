Feature: Step-code defects return to the shaper
  A concrete go run exit-code defect must be repairable within the existing correction budget.

  Scenario: Repair a rejected step package before publishing
    When the step-check correction journey runs in "repair" mode
    Then the step-check correction contract passes

  Scenario: Stop only after correction attempts are exhausted
    When the step-check correction journey runs in "exhaust" mode
    Then the step-check correction contract passes
