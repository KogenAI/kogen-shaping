Feature: A static check that fails differently on two runs is not attributed to the shaped files
  This feature checks the static gate on a check that fails with different messages on its plain run and its traced re-run, and on checks that must still be refused.

  Scenario: A check failing differently on two runs without naming a shaped file is flaky and unknown
    Given shape-static-check-flaky-not-attributable: the static gate has run a check that fails differently on its two runs and checks that fail the same way twice or name a shaped file
    Then shape-static-check-flaky-not-attributable: the unsteady check has status "flaky"
    And shape-static-check-flaky-not-attributable: the unsteady check exit is 1
    And shape-static-check-flaky-not-attributable: the run with only the unsteady check has outcome "unknown"

  Scenario: The flaky check says so in its progress line and records both failing sets
    Given shape-static-check-flaky-not-attributable: the static gate has run a check that fails differently on its two runs and checks that fail the same way twice or name a shaped file
    Then shape-static-check-flaky-not-attributable: the unsteady check progress line is "kogen: static check unsteady failed (exit 1) with different failures on two runs of the same shaped files: not attributable; the shaped files are not established by this check"
    And shape-static-check-flaky-not-attributable: the unsteady check records the failing lines "[\"Failure: alpha_test\"]" for the plain run and "[\"Error: beta_test\"]" for the traced run

  Scenario: A failure that repeats identically without naming a shaped file is still refused
    Given shape-static-check-flaky-not-attributable: the static gate has run a check that fails differently on its two runs and checks that fail the same way twice or name a shaped file
    Then shape-static-check-flaky-not-attributable: the check identical is "attributable" with files ""
    And shape-static-check-flaky-not-attributable: the progress line of the check identical is "kogen: static check identical failed (exit 1) because of the shaped files: (not named by the check)"

  Scenario: Failures that name a shaped file are still refused
    Given shape-static-check-flaky-not-attributable: the static gate has run a check that fails differently on its two runs and checks that fail the same way twice or name a shaped file
    Then shape-static-check-flaky-not-attributable: the check located is "attributable" with files "features/steps/fixture_steps.js:3:5"
    And shape-static-check-flaky-not-attributable: the check tracedpath is "attributable" with files "features/steps/fixture_steps.js:3:5"
    And shape-static-check-flaky-not-attributable: the run with the three repeating or path-naming checks has outcome "refused"
