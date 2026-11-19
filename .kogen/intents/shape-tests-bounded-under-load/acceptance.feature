Feature: The shape package tests are bounded under load
  This feature covers the test code of the shape package. The shape tests take their time from an
  injected clock and fake commands instead of real sleeps, the overlap, control-bound and time-split
  tests assert ordering and injected durations instead of wall-clock thresholds, and the change
  stays within test code and test-only seams.

  Scenario: No shape test waits on the real clock or asserts a wall-clock threshold
    Given shape-tests-bounded-under-load: the test files of the shape package
    Then shape-tests-bounded-under-load: no shape test calls time.Sleep
    And shape-tests-bounded-under-load: no shape test runs a command that sleeps or waits more than one second
    And shape-tests-bounded-under-load: no shape test contains the wall-clock assertion "independent two-second waits took"
    And shape-tests-bounded-under-load: no shape test contains the wall-clock assertion "want [300,600)"

  Scenario: The existing overlap, control-bound and time-split tests still pass and the clock-bound tests pass with them
    When shape-tests-bounded-under-load: I run the tests of the shape package whose names start with the clock-bound prefix and the existing overlap, control-bound and time-split tests once
    Then shape-tests-bounded-under-load: the go command exits 0
    And shape-tests-bounded-under-load: "TestStaticChecksOverlapModelChecks" is reported as passed
    And shape-tests-bounded-under-load: "TestUnpassableControlSkipped" is reported as passed
    And shape-tests-bounded-under-load: "TestShapeTimeSplitsChecksByGate" is reported as passed
    And shape-tests-bounded-under-load: at least 4 clock-bound tests are reported as passed and none is reported as failed or skipped

  Scenario: The change adds the clock-bound tests and changes only test code and test-only seams
    Given shape-tests-bounded-under-load: the test files of the shape package
    And shape-tests-bounded-under-load: the files this change adds or edits
    Then shape-tests-bounded-under-load: every changed file is under shape/, .kogen/ or features/
    And shape-tests-bounded-under-load: no changed Go source of the shape package outside its tests adds an environment variable, flag or setting read
    And shape-tests-bounded-under-load: no shape test is skipped, short-mode gated or given a longer timeout by this change
    And shape-tests-bounded-under-load: the file shape/clock_bound_test.go defines the four TestClockBound tests
