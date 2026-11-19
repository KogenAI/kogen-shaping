Feature: Shaping time excludes timed-out static checks
  This feature runs the real Shape entry point with a fake clock and fake static checks. A static
  check that timed out is not charged to the shaping budget, and the shape_time event records the
  excluded part beside the full static wall time. The budget refusal names the excluded part beside
  the install time.

  Background:
    Given shape-time-excludes-timed-out-static-checks: the shaping harness has run

  Scenario: A check that times out after 120 s is excluded
    Then shape-time-excludes-timed-out-static-checks: the shape_time of case "timeout" includes:
      """
      {"static_ms": 120000, "static_uncharged_ms": 120000}
      """

  Scenario: A check that fails after 120 s is charged in full
    Then shape-time-excludes-timed-out-static-checks: the shape_time of case "failed" includes:
      """
      {"static_uncharged_ms": 0}
      """

  Scenario: Each timed-out check excludes at most 120 s and a passing check is charged
    Then shape-time-excludes-timed-out-static-checks: the shape_time of case "bounded" includes:
      """
      {"static_ms": 300000, "static_uncharged_ms": 240000}
      """

  Scenario: The budget refusal charges the passing checks only and names the uncharged time
    Then shape-time-excludes-timed-out-static-checks: the budget refusal reports 29 min elapsed and names static_uncharged_ms 120000 on its figure and attempt lines
