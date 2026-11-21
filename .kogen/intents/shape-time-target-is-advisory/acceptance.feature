Feature: Shape time is a target, not a stop
  This feature observes scripted Shape runs and shape_time accounts with fake clocks and no network.
  The one field the change adds to the shape_time event (its local check time in milliseconds) is found at run time as the only key the event did not have before.

  Background:
    Given shape-time-target-is-advisory: the supplied scripted Shape time observer is available
    When shape-time-target-is-advisory: the scripted shapings and time accounts run

  Scenario: A scripted draft advances across the 15, 20 and 45 minute points of wall clock and the later valid draft passes
    Then shape-time-target-is-advisory: the "target" observation equals
      """
      {"exit":0,"failed":false,"requests":4,"time_lines":["Time: 0 min elapsed. Target: 15 minutes usual, 20 minutes upper end.","Time: 15 min elapsed. Target: 15 minutes usual, 20 minutes upper end.","Time: 20 min elapsed. Target: 15 minutes usual, 20 minutes upper end.","Time: 45 min elapsed. Target: 15 minutes usual, 20 minutes upper end."]}
      """

  Scenario: Cancellation ends work with its own reason
    Then shape-time-target-is-advisory: the "cancel" observation equals
      """
      {"failed":true,"requests":2,"mentions_cancellation":true,"mentions_target_budget_or_timeout":false}
      """

  Scenario: A required under 120 seconds literal weakened to 125 seconds stays refused
    Then shape-time-target-is-advisory: the "literal_replay" observation equals
      """
      {"kept":"present","weakened":"missing"}
      """

  Scenario: The time line states the target and elapsed time without deadline wording
    Then shape-time-target-is-advisory: the "text" observation equals
      """
      {"first":"Time: 0 min elapsed. Target: 15 minutes usual, 20 minutes upper end.","last":"Time: 45 min elapsed. Target: 15 minutes usual, 20 minutes upper end."}
      """

  Scenario: After a shaper request of 100 seconds, a static check of 120 seconds and a readiness observation of 300 seconds started together count once
    Then shape-time-target-is-advisory: the "overlap_together" observation equals
      """
      {"added_field_count":1,"added_field_ms":300000,"elapsed_ms":400000}
      """

  Scenario: A static check starting 200 seconds after the readiness observation and ending 20 seconds after it counts the union
    Then shape-time-target-is-advisory: the "overlap_staggered" observation equals
      """
      {"added_field_count":1,"added_field_ms":320000,"elapsed_ms":420000}
      """

  Scenario: A decision check of 60 seconds alone leaves local check time 0 and elapsed 160 seconds
    Then shape-time-target-is-advisory: the "decision_only" observation equals
      """
      {"event":"shape_time","elapsed_ms":160000,"shaper_ms":100000,"shaper_model_ms":null,"shaper_tools_ms":null,"install_ms":0,"checks_ms":60000,"static_ms":0,"static_uncharged_ms":0,"model_checks_ms":60000,"own check seconds":60,"acceptance timeout":[],"contract_ms":0,"readiness_ms":0,"gap_ms":0,"added_field_count":1,"added_field_ms":0}
      """

  Scenario: Three long rounds of 500 seconds, two of them refused submissions, still allow a fourth request
    Then shape-time-target-is-advisory: the "rounds_requests" observation equals
      """
      {"at_least_four_requests":true,"names_budget_or_ceiling":false}
      """

  Scenario: Three readiness observations of 400 seconds after 100 seconds each reach a wall clock of 1500 seconds
    Then shape-time-target-is-advisory: the "rounds_account" observation equals
      """
      {"added_field_count":1,"added_field_ms":1200000,"elapsed_ms":1500000}
      """
