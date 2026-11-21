Feature: A narrow model-free diagnostic names the cause of a refused check on a retained Shape draft
  This feature observes scripted diagnostics of retained drafts held in temporary runner and pinned checkouts, run with the real Cucumber runner and no provider.

  Background:
    Given shape-recheck-retained-draft-names-cause: the supplied recheck observer is available
    When shape-recheck-retained-draft-names-cause: the scripted rechecks run

  Scenario: A correct reference that fails its own acceptance is named with its scenario and originating command
    Then shape-recheck-retained-draft-names-cause: the "reference" observation equals
      """
      {"verdict":"reference check failed","scenario":"S","class":"","runner_kind":"","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":true,"reference_text":true,"has_evidence":true,"acceptance_runs":0,"state_unchanged":true}
      """

  Scenario: A kept assertion that fails on the unchanged tree is a bad draft, not a reference failure
    Then shape-recheck-retained-draft-names-cause: the "kept" observation equals
      """
      {"verdict":"bad draft","scenario":"S","class":"","runner_kind":"","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":true,"reference_text":false,"has_evidence":true,"acceptance_runs":0,"state_unchanged":true}
      """

  Scenario: The same failure on both trees of a scenario that is not kept leaves attribution open with the evidence
    Then shape-recheck-retained-draft-names-cause: the "ambiguous" observation equals
      """
      {"verdict":"","scenario":"S","class":"","runner_kind":"","attribution_open":true,"unavailable":false,"names_gate":false,"names_command":true,"reference_text":false,"has_evidence":true,"acceptance_runs":0,"state_unchanged":true}
      """

  Scenario: A passing focused diagnostic finds no fault and leaves the latch and approval state unchanged
    Then shape-recheck-retained-draft-names-cause: the "passing" observation equals
      """
      {"verdict":"","scenario":"","class":"","runner_kind":"","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":false,"reference_text":false,"has_evidence":false,"acceptance_runs":0,"state_unchanged":true}
      """

  Scenario: A timed-out runner stays a runner failure after a single run
    Then shape-recheck-retained-draft-names-cause: the "timeout" observation equals
      """
      {"verdict":"runner failure","scenario":"","class":"","runner_kind":"timeout","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":false,"reference_text":false,"has_evidence":true,"acceptance_runs":1,"state_unchanged":true}
      """

  Scenario: A missing tool and a killed command stay runner failures
    Then shape-recheck-retained-draft-names-cause: the "missingtool" observation equals
      """
      {"verdict":"runner failure","scenario":"","class":"","runner_kind":"missing tool","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":false,"reference_text":false,"has_evidence":true,"acceptance_runs":1,"state_unchanged":true}
      """
    And shape-recheck-retained-draft-names-cause: the "killed" observation equals
      """
      {"verdict":"runner failure","scenario":"","class":"","runner_kind":"killed command","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":false,"reference_text":false,"has_evidence":true,"acceptance_runs":1,"state_unchanged":true}
      """

  Scenario: A failing setup is reported with the setup_failure class and no acceptance run
    Then shape-recheck-retained-draft-names-cause: the "setup" observation equals
      """
      {"verdict":"runner failure","scenario":"","class":"setup_failure","runner_kind":"setup error","attribution_open":false,"unavailable":false,"names_gate":false,"names_command":false,"reference_text":false,"has_evidence":true,"acceptance_runs":0,"state_unchanged":true}
      """

  Scenario: A mismatched draft makes no inference and changes no state
    Then shape-recheck-retained-draft-names-cause: the "mismatch" observation equals
      """
      {"verdict":"","scenario":"","class":"","runner_kind":"","attribution_open":false,"unavailable":true,"names_gate":true,"names_command":false,"reference_text":false,"has_evidence":true,"acceptance_runs":0,"state_unchanged":true}
      """

  Scenario: A model-dependent gate without retained evidence makes no inference and names the gate
    Then shape-recheck-retained-draft-names-cause: the "nomodel" observation equals
      """
      {"verdict":"","scenario":"","class":"","runner_kind":"","attribution_open":false,"unavailable":true,"names_gate":true,"names_command":false,"reference_text":false,"has_evidence":true,"acceptance_runs":0,"state_unchanged":true}
      """
