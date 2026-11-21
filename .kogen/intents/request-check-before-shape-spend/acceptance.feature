Feature: A request check runs before Shape spends a model call
  This feature observes Shape on temporary checkouts with a scripted fake client.

  Scenario: A protected-file edit demand is repaired
    Then request-check-before-shape-spend: the "repair" observation equals
      """
      {"printed":true,"calls":true,"demand":true,"limit_line":true,"rest":true,"received":true}
      """

  Scenario: Two scenarios with the same outcome are refused
    Then request-check-before-shape-spend: the "duplicate" observation equals
      """
      {"marker":true,"names":true,"calls":0}
      """

  Scenario: A scenario without an outcome is refused
    Then request-check-before-shape-spend: the "nooutcome" observation equals
      """
      {"marker":true,"names":true,"calls":0}
      """

  Scenario: A missing referenced path is refused
    Then request-check-before-shape-spend: the "missing" observation equals
      """
      {"marker":true,"names":true,"calls":0}
      """

  Scenario: A clean request produces the golden events and text
    Then request-check-before-shape-spend: the "clean" observation equals
      """
      {"printed":false,"refused":false,"calls":true,"stored":true,"received":true,"events":["response_budget","cache_identity","context_selection","provider_call","cache_identity","provider_call","question_check","shaping_turn_counts","usage_totals","phase_duration","phase_duration","phase_duration","phase_duration","phase_duration","phase_duration","shape_time","session_end"],"stdout":"Questions for is-even:\n1. Which wording?\n   a) keep\n   b) shorter\nAnswer one per line as <n>: <letter or your words>, e.g. printf '1: a\\n' | kogen intent shape is-even -\nOr enter all: recommended to use every question's recommended option.\n","stderr":"kogen: checking the acceptance runner\nkogen: shaping (request 1, T s)\nkogen: shape response input=100 cached=40 output=30 reasoning=10 wall_ms=W (T s)\nkogen: checking the questions (T s)\nkogen: question check response input=101 cached=40 output=30 reasoning=10 wall_ms=W (T s)\nkogen: shape time T s: shaper T s (model T s, tools T s), Kogen checks T s (static T s, model checks T s, contract review T s, readiness T s), gaps T s\n"}
      """
