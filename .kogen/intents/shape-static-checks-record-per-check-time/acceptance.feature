Feature: The static checks event records the time of each check

  # Both scenarios shape a request in a small project with a scripted shaper. The sleeping checks
  # take at least 50 ms each. In the first scenario the submission is refused on acceptance red and
  # then acknowledged. In the second, the first package is refused by a static check, the same
  # package is submitted again (the gate reuses its result), then a repaired package is refused on
  # acceptance red and the last submission acknowledges.

  Scenario: Each check, its base run and the copy are timed
    When shape-static-checks-record-per-check-time: the project with the checks "slow", "stale" and "suite" is shaped with one refused submission and an acknowledgment of the unchanged package
    Then shape-static-checks-record-per-check-time: the Intent is published
    And shape-static-checks-record-per-check-time: the first static checks event has the checks "slow", "stale" and "suite" in order
    And shape-static-checks-record-per-check-time: in the first static checks event the check "slow" has wall_ms greater than 0 and base_wall_ms 0
    And shape-static-checks-record-per-check-time: in the first static checks event the check "stale" has wall_ms greater than 0 and base_wall_ms greater than 0
    And shape-static-checks-record-per-check-time: in the first static checks event the check "suite" has wall_ms 0 and base_wall_ms 0
    And shape-static-checks-record-per-check-time: the first static checks event has copy_ms greater than 0

  Scenario: The refusal text of a failing check is unchanged, the failing check is timed and a reused result repeats the timings
    When shape-static-checks-record-per-check-time: the project with the failing check "bad" is shaped with a refused submission, a repaired submission and an acknowledgment
    Then shape-static-checks-record-per-check-time: the Intent is published
    And shape-static-checks-record-per-check-time: the refusal of the static checks gate has the problem lines, in order:
      """
      - check bad: exit 1; files: features/steps/is-even__steps.js:3:5; command: sleep 0.05; if grep -q BAD features/steps/is-even__steps.js; then echo "features/steps/is-even__steps.js:3:5: bad"; exit 1; fi
        | features/steps/is-even__steps.js:3:5: bad
      """
    And shape-static-checks-record-per-check-time: in the first static checks event the check "bad" has wall_ms greater than 0 and base_wall_ms greater than 0
    And shape-static-checks-record-per-check-time: the second static checks event is reused and repeats the copy_ms, wall_ms and base_wall_ms values of the first
