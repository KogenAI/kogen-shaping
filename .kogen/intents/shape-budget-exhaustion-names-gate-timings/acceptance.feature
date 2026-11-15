Feature: Shaping budget refusal names gates and timings
  This feature checks the refusal of a shaping whose wall-clock budget ran out after a scripted shaper was refused four times.

  Scenario: Four refused attempts are listed with their gate and timings
    Given shape-budget-exhaustion-names-gate-timings: a scripted shaper refused four times at 225 seconds per model request
    When shape-budget-exhaustion-names-gate-timings: the shaping runs until its budget ends
    Then shape-budget-exhaustion-names-gate-timings: the refusal lists four timed attempts with gate, model time, check time and remaining budget

  Scenario: The budget figure equals the recorded event
    Given shape-budget-exhaustion-names-gate-timings: a scripted shaper refused four times at 225 seconds per model request
    When shape-budget-exhaustion-names-gate-timings: the shaping runs until its budget ends
    Then shape-budget-exhaustion-names-gate-timings: the refusal states a budget figure equal to the shaping_turn_counts budget

  Scenario: The other budget is documented
    Given shape-budget-exhaustion-names-gate-timings: a scripted shaper refused four times at 225 seconds per model request
    When shape-budget-exhaustion-names-gate-timings: the shaping runs until its budget ends
    Then shape-budget-exhaustion-names-gate-timings: the refusal says response_budget is a separate budget

  Scenario: The existing refusal detail is kept
    Given shape-budget-exhaustion-names-gate-timings: a scripted shaper refused four times at 225 seconds per model request
    When shape-budget-exhaustion-names-gate-timings: the shaping runs until its budget ends
    Then shape-budget-exhaustion-names-gate-timings: the refusal still reports 15 min elapsed and 4 requests
