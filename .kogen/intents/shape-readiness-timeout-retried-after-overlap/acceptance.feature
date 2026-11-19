Feature: Overlapped readiness timeout is repeated alone
  This feature checks that Shape repeats an acceptance timeout of the overlapped readiness observation once, alone, and never repeats anything else.

  Scenario: A timeout beside a running static check is repeated alone and passes
    Given shape-readiness-timeout-retried-after-overlap: a scripted readiness that times out on its first call and passes on its second, with a static check running beside it
    When shape-readiness-timeout-retried-after-overlap: the submission's readiness is joined
    Then shape-readiness-timeout-retried-after-overlap: the submission is ready
    And shape-readiness-timeout-retried-after-overlap: the scripted readiness was called 2 times
    And shape-readiness-timeout-retried-after-overlap: the second call began after the static check ended
    And shape-readiness-timeout-retried-after-overlap: the progress line was written
    And shape-readiness-timeout-retried-after-overlap: one repeat event carries the wall time of the first run
    And shape-readiness-timeout-retried-after-overlap: no refusal was recorded

  Scenario: Two timeouts are refused with one added line after exactly two calls
    Given shape-readiness-timeout-retried-after-overlap: a scripted readiness that times out on both calls, with a static check running beside it
    When shape-readiness-timeout-retried-after-overlap: the submission's readiness is joined
    Then shape-readiness-timeout-retried-after-overlap: the submission is refused as an acceptance timeout whose last line is "Readiness was repeated alone after the concurrent checks ended; both runs timed out." after the existing timeout lines
    And shape-readiness-timeout-retried-after-overlap: the scripted readiness was called 2 times

  Scenario: A timeout with nothing overlapping is refused at once
    Given shape-readiness-timeout-retried-after-overlap: a scripted readiness that times out with no static check or model check beside it
    When shape-readiness-timeout-retried-after-overlap: the submission's readiness is joined
    Then shape-readiness-timeout-retried-after-overlap: the submission is refused as an acceptance timeout with exactly the existing lines
    And shape-readiness-timeout-retried-after-overlap: the scripted readiness was called 1 times
    And shape-readiness-timeout-retried-after-overlap: no repeat event or progress line was written

  Scenario: A failing acceptance assertion beside a running static check is not repeated
    Given shape-readiness-timeout-retried-after-overlap: a scripted readiness that refuses for a failing acceptance assertion, with a static check running beside it
    When shape-readiness-timeout-retried-after-overlap: the submission's readiness is joined
    Then shape-readiness-timeout-retried-after-overlap: the submission is refused as "acceptance failed" with its existing lines
    And shape-readiness-timeout-retried-after-overlap: the scripted readiness was called 1 times
    And shape-readiness-timeout-retried-after-overlap: no repeat event or progress line was written
