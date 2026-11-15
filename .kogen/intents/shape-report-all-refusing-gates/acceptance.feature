Feature: Shape reports every independent refusing gate in one round
  A submission that fails the readiness gate and a cheap gate comes back with both refusals at
  once. Each case is a small synthetic project shaped by a scripted shaper and a replayed
  acceptance report.

  Scenario: A decision record refusal and a readiness refusal are reported together
    When shape-report-all-refusing-gates: the "both" draft is shaped
    Then shape-report-all-refusing-gates: the first submission is refused by the gates "decision_record,decision_check,literal_check,readiness"
    And shape-report-all-refusing-gates: the first feedback contains "sourced decision 1: no evidence and no uncertainty"
    And shape-report-all-refusing-gates: the first feedback contains "Decision check: "
    And shape-report-all-refusing-gates: the first feedback contains "Literal check: "
    And shape-report-all-refusing-gates: the first feedback contains the readiness refusal text unchanged

  Scenario: Both refusals count as one submission and no contract review runs
    When shape-report-all-refusing-gates: the "both" draft is shaped
    Then shape-report-all-refusing-gates: the first submission counts once against the retry budget
    And shape-report-all-refusing-gates: no contract review ran

  Scenario: A readiness refusal alone keeps its feedback
    When shape-report-all-refusing-gates: the "readiness" draft is shaped
    Then shape-report-all-refusing-gates: the first submission is refused by the gates "readiness"
    And shape-report-all-refusing-gates: the first feedback contains the readiness refusal text unchanged
