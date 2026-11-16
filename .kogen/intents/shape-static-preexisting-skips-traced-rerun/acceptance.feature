Feature: A static check that also fails without the shaped files skips the traced rerun
  This feature runs the real static gate of the shape package over three configured checks. Each check appends "plain" or "traced" to a marker file every time it executes, "traced" when its text ran under set -x, so the run order and count are observed.

  Scenario: A pre-existing failure never runs the traced rerun
    Given shape-static-preexisting-skips-traced-rerun: the static gate has judged a preexisting, an attributable and a flaky check
    Then shape-static-preexisting-skips-traced-rerun: the preexisting check ran 2 times in round 1, 0 of them under set -x
    And shape-static-preexisting-skips-traced-rerun: the preexisting check in round 1 is recorded "preexisting" with paths "app.txt:3:1" and an empty trace
    And shape-static-preexisting-skips-traced-rerun: the preexisting check ran 1 times in round 2, 0 of them under set -x
    And shape-static-preexisting-skips-traced-rerun: the preexisting check in round 2 is recorded "preexisting" with paths "app.txt:3:1" and an empty trace

  Scenario: A pre-existing failure keeps its status, paths and reused base result
    Given shape-static-preexisting-skips-traced-rerun: the static gate has judged a preexisting, an attributable and a flaky check
    Then shape-static-preexisting-skips-traced-rerun: the preexisting check in round 1 is recorded "preexisting" with paths "app.txt:3:1"
    And shape-static-preexisting-skips-traced-rerun: the static run outcome of the preexisting check is "unknown"
    And shape-static-preexisting-skips-traced-rerun: the preexisting check in round 2 is recorded "preexisting" with paths "app.txt:3:1"
    And shape-static-preexisting-skips-traced-rerun: the preexisting check in round 2 reports that the base result was reused

  Scenario: A failure only the shaped files cause still runs the traced rerun
    Given shape-static-preexisting-skips-traced-rerun: the static gate has judged a preexisting, an attributable and a flaky check
    Then shape-static-preexisting-skips-traced-rerun: the attributable check ran 3 times in round 1, 1 of them under set -x
    And shape-static-preexisting-skips-traced-rerun: the attributable check is recorded "attributable" with a non-empty trace
    And shape-static-preexisting-skips-traced-rerun: the static run outcome of the attributable check is "refused"

  Scenario: A failure that differs on the traced rerun is still flaky and not attributable
    Given shape-static-preexisting-skips-traced-rerun: the static gate has judged a preexisting, an attributable and a flaky check
    Then shape-static-preexisting-skips-traced-rerun: the flaky check ran 3 times in round 1, 1 of them under set -x
    And shape-static-preexisting-skips-traced-rerun: the flaky check is recorded "flaky" with a non-empty trace
    And shape-static-preexisting-skips-traced-rerun: the flaky check records failing lines "Failure: alpha_test" and traced failing lines "Error: beta_test"
    And shape-static-preexisting-skips-traced-rerun: the static run outcome of the flaky check is "unknown"
