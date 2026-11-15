Feature: Support evidenced keep-existing Decisions
  This feature exercises Shape's decision_check gate with a scripted checker that always answers
  unsupported for the Decision, so only the mechanical keep-existing rule can change the outcome.
  It does not claim a live model evaluation. Studio issue kogen/ed7490a5-37ca-4fc0-b7c6-30664f7efcbd.

  Scenario: Keep existing limits shown by the cited file
    Given shape-keep-existing-decisions-supported: the case
      """
      {"Statement":"controlsMaxRuns (8), controlsMaxFiles (8), controlsMaxBytes (65536) and controlsTimeCap (300) keep their values.","Citation":"limits.go","Expect":"follows"}
      """
    When shape-keep-existing-decisions-supported: decision_check evaluates the candidate
    Then shape-keep-existing-decisions-supported: the final verdict and feedback match the case

  Scenario: Keep existing value citing a missing path is refused
    Given shape-keep-existing-decisions-supported: the case
      """
      {"Statement":"controlsMaxRuns (8) keeps its value.","Citation":"missing.go","Expect":"unsupported"}
      """
    When shape-keep-existing-decisions-supported: decision_check evaluates the candidate
    Then shape-keep-existing-decisions-supported: the final verdict and feedback match the case

  Scenario: Keep existing value not shown by the cited file is refused
    Given shape-keep-existing-decisions-supported: the case
      """
      {"Statement":"controlsMaxRuns (9) keeps its value.","Citation":"limits.go","Expect":"unsupported"}
      """
    When shape-keep-existing-decisions-supported: decision_check evaluates the candidate
    Then shape-keep-existing-decisions-supported: the final verdict and feedback match the case

  Scenario: A new Decision that adds scope beyond a quoted request passage is still refused
    Given shape-keep-existing-decisions-supported: the case
      """
      {"Statement":"Increase controlsMaxRuns to 9.","Citation":"","Quotes":["Improve controls"],"Why":"this adds scope the quoted request passage does not ask for","Expect":"unsupported"}
      """
    When shape-keep-existing-decisions-supported: decision_check evaluates the candidate
    Then shape-keep-existing-decisions-supported: the final verdict and feedback match the case
