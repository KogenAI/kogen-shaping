Feature: The wrong-implementation check returns infrastructure errors instead of refusing the candidate
  The wrong-implementation check runs the candidate's acceptance in a fresh copy outside the
  checkout. When Kogen cannot prepare that copy (its temporary directory, the preload file or the
  materialized candidate tree) or a configured setup command fails there, the check returns an
  error, so Shape stops with that error instead of refusing the candidate with "no pass is
  reachable" and spending the shaper's correction allowance. A failure to write the candidate's own
  controls.json files over the copy is still the candidate's problem and is refused as before. A
  contract check that stops before its review leaves no properties of an earlier review behind for
  the wrong-implementation check to use. The existing tests of the wrong-implementation check and
  the contract check stay green.

  Background:
    Given fix-k-shape-wrong-implementation: the wrong-implementation check harness has run

  Scenario: A candidate tree that cannot be materialized stops the check with an error
    Then fix-k-shape-wrong-implementation: the error of "materialize" is not empty
    And fix-k-shape-wrong-implementation: the observation of "materialize" includes:
      """
      {"refusal": [], "runs": 1, "events": []}
      """

  Scenario: A temporary directory that cannot be created stops the check with an error
    Then fix-k-shape-wrong-implementation: the error of "kit" is not empty
    And fix-k-shape-wrong-implementation: the observation of "kit" includes:
      """
      {"refusal": [], "runs": 1, "events": []}
      """

  Scenario: A failing setup command stops the check with an error naming the command and its status
    Then fix-k-shape-wrong-implementation: the error of "setup" contains 'setup command "exit 3" exited with status 3'
    And fix-k-shape-wrong-implementation: the observation of "setup" includes:
      """
      {"refusal": [], "runs": 1, "events": []}
      """

  Scenario: A correct implementation that cannot be written over the copy is still refused
    Then fix-k-shape-wrong-implementation: the observation of "overlay" includes:
      """
      {
        "error": "",
        "refusal": [
          "- add keeps working (kept) | request: add | problem: no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed",
          "- sub keeps working (kept) | request: sub | problem: no pass is reachable: with the correct implementation in place, the run did not report every scenario as passed or failed"
        ],
        "runs": 1,
        "events": ["refused"]
      }
      """

  Scenario: A contract check that stops before its review leaves no stale review properties
    Then fix-k-shape-wrong-implementation: the observation of "stale" includes:
      """
      {"contract_error": true, "review_properties": 0, "error": "", "refusal": [], "runs": 0, "events": []}
      """

  Scenario: The existing tests of the wrong-implementation check and the contract check stay green
    When fix-k-shape-wrong-implementation: I run these existing shape tests:
      """
      TestParseControls
      TestControlsJudge
      TestControlsJudgeAll
      TestControlsUnreachable
      TestControlsRefusalJoinsContractRefusal
      TestControlsPreloadFreezesAssert
      TestWrongImplementationRealRuns
      TestWrongImplementationCaps
      TestWrongImplementationCheckRecordsAndSkips
      TestContractEvidenceRule
      TestContractPayloadValidation
      TestContractCheckLastRefusalByAnotherGateWritesNoRecord
      TestContractVerdictCounting
      TestContractFormats
      TestContractEnvironmentEvidenceRule
      TestContractCheckNumbersPropertyProblemsAfterContradictions
      TestContractCheckRefusesAConfirmedContradictionAndPublishesItsCorrection
      TestContractCheckKeptContradictionEndsNotReadyAndApprovalRefusesIt
      TestContractCheckExternalShape
      TestContractCheckEnvironmentFindings
      TestContractCheckUnevidencedAndRefutedFindingsNeverRefuse
      TestContractCheckReadOnlyToolRoundsAndIncompleteReviews
      """
    Then fix-k-shape-wrong-implementation: each of them is reported as passed exactly once, and no test or subtest failed or was skipped
