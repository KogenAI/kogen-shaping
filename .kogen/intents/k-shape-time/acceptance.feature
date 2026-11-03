Feature: Record shape time and keep shaper commands short
  Every shape run ends its shaping record with an account of where its time went: the shaper's
  model and tools, Kogen's own checks and the remaining gaps, and states it on its last stderr
  line. The claude-code shaper is told to leave readiness to Kogen, keep its commands short and
  never run them in the background or poll.

  Scenario: A claude-code shape records its model, tool, check and gap time
    When a claude-code shape runs rounds "1200:300,600:200" with contract reviews of 500 ms and acceptance of 1 second
    Then the shape is ready with its usual output
    And the shaping record ends with one shape time account before the session end
    And the main shaper rounds are recorded as:
      | round | model_ms | min_wall_ms |
      | 1     | 300      | 1200        |
      | 2     | 200      | 600         |
    And the rounds of session "executor-contract-review-1" are recorded with at least 500 ms and unknown model time
    And the shaper model time is 500 ms
    And the shaper time is the total of the main shaper rounds
    And the shaper time is the shaper model time plus the shaper tools time
    And Kogen checks took at least 1000 ms, of which readiness took at least 1000 ms
    And Kogen checks cover readiness and the rounds of session "executor-contract-review-1"
    And the gaps are the elapsed time minus the shaper and check times
    And the last stderr line states the shape time account

  Scenario: Model time that a round does not report stays unknown
    When a claude-code shape runs rounds "300:250,100:none" with contract reviews of 0 ms and acceptance of 0 seconds
    Then the shape is ready with its usual output
    And the shaping record ends with one shape time account before the session end
    And the main shaper rounds are recorded as:
      | round | model_ms | min_wall_ms |
      | 1     | 250      | 300         |
      | 2     | unknown  | 100         |
    And the shaper model and tools times are unknown
    And the shaper time is the total of the main shaper rounds
    And the gaps are the elapsed time minus the shaper and check times
    And the last stderr line states the shape time account

  Scenario: A native shape records the same account
    When a native shape runs with acceptance of 1 second
    Then the shape is ready with its usual output
    And the shaping record ends with one shape time account before the session end
    And the shaper time is the shaper model time plus the shaper tools time
    And Kogen checks took at least 1000 ms, of which readiness took at least 1000 ms
    And the gaps are the elapsed time minus the shaper and check times
    And the last stderr line states the shape time account

  Scenario: The claude-code shaper is told to keep commands short and never poll
    When a claude-code shape runs rounds "0:10,0:10" with contract reviews of 0 ms and acceptance of 0 seconds
    Then the shape is ready with its usual output
    And the claude-code shaper's task contains this paragraph exactly once:
      """
      Commands: Kogen runs readiness on every submission (the configured setup, the acceptance command, the Quint model and the Feasibility checks) and returns its actual results to this session. Do not repeat that work yourself: do not run the configured setup or checks, the acceptance command or a whole test suite, and do not prototype the implementation. Run only short commands that answer a named concrete uncertainty, such as one focused test, a probe or a tool query, each finishing on its own within 5 minutes. Never start a command in the background and never wait or poll with sleep: when you would need a longer run, submit your complete candidate and use Kogen's results.
      """
