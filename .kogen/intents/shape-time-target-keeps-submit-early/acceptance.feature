Feature: The shaper converges inside the usual target and time stays advisory
  This feature observes the shaper prompts, scripted Shape runs and shape_time accounts with fake clocks and no network.

  Background:
    Given shape-time-target-keeps-submit-early: the supplied scripted observer is available
    When shape-time-target-keeps-submit-early: the scripted observations run

  Scenario: The native shaper prompt tells the shaper to converge inside the usual target
    Then shape-time-target-keeps-submit-early: the "native_prompt" observation equals
      """
      {"first_rounds":true,"late_instruction":true,"late_package_checked":true,"never_a_stop":true,"old_advice_sentence_gone":true}
      """

  Scenario: The external-route guidance tells the shaper to converge with the target filled in
    Then shape-time-target-keeps-submit-early: the "external_guidance" observation equals
      """
      {"first_rounds":true,"late_instruction_15":true,"late_instruction_25":true,"late_package_checked":true,"never_a_stop":true,"old_advice_sentence_gone":true,"placeholders_filled":true}
      """

  Scenario: The time line of every request stays byte for byte and no stop appears after 1500 seconds
    Then shape-time-target-keeps-submit-early: the "rounds" observation equals
      """
      {"at_least_four_requests":true,"first_four_time_lines":["Time: 0 min elapsed. Target: 15 minutes usual, 20 minutes upper end.","Time: 8 min 20 s elapsed. Target: 15 minutes usual, 20 minutes upper end.","Time: 16 min 40 s elapsed. Target: 15 minutes usual, 20 minutes upper end.","Time: 25 min elapsed. Target: 15 minutes usual, 20 minutes upper end."],"names_budget_or_ceiling":false}
      """

  Scenario: A required under 120 seconds literal weakened to 125 seconds stays refused
    Then shape-time-target-keeps-submit-early: the "literal_replay" observation equals
      """
      {"kept":"present","weakened":"missing"}
      """

  Scenario: The shape_time event of a decision check alone is unchanged in every field
    Then shape-time-target-keeps-submit-early: the "event_decision_only" observation equals
      """
      {"acceptance timeout":[],"checks_ms":60000,"contract_ms":0,"elapsed_ms":160000,"event":"shape_time","gap_ms":0,"install_ms":0,"local_check_ms":0,"model_checks_ms":60000,"own check seconds":60,"readiness_ms":0,"setup_ms":0,"shaper_model_ms":null,"shaper_ms":100000,"shaper_tools_ms":null,"static_ms":0,"static_uncharged_ms":0}
      """

  Scenario: The shape_time event of a readiness and static account is unchanged in every field
    Then shape-time-target-keeps-submit-early: the "event_readiness_and_static" observation equals
      """
      {"acceptance timeout":[],"checks_ms":320000,"contract_ms":0,"elapsed_ms":420000,"event":"shape_time","gap_ms":0,"install_ms":0,"local_check_ms":320000,"model_checks_ms":0,"own check seconds":320,"readiness_ms":300000,"setup_ms":0,"shaper_model_ms":null,"shaper_ms":100000,"shaper_tools_ms":null,"static_ms":120000,"static_uncharged_ms":0}
      """
