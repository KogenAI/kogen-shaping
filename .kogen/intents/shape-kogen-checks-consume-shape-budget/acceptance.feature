Feature: Kogen's shape checks take less wall-clock time
  This feature runs focused model-free Go tests of the shape package.

  Scenario: Readiness overlaps the cheap checks
    Given shape-kogen-checks-consume-shape-budget: the Go test overlay is available
    When shape-kogen-checks-consume-shape-budget: the shape test TestOverlayReadinessOverlapsCheapChecks is run
    Then shape-kogen-checks-consume-shape-budget: the shape test passed

  Scenario: Combined refusals keep today's order and text
    Given shape-kogen-checks-consume-shape-budget: the Go test overlay is available
    When shape-kogen-checks-consume-shape-budget: the shape test TestOverlayCombinedRefusalsKeepOrder is run
    Then shape-kogen-checks-consume-shape-budget: the shape test passed

  Scenario: Decision answer reused for identical input
    Given shape-kogen-checks-consume-shape-budget: the Go test overlay is available
    When shape-kogen-checks-consume-shape-budget: the shape test TestOverlayDecisionAnswerReuse is run
    Then shape-kogen-checks-consume-shape-budget: the shape test passed
