Feature: Probe completion visibility during shaping
  Scenario: Successful probe status remains visible
    Then k-shape-progress-probe-result: the existing shaping regression "TestShapeProbeRunsOutsideTheProjectAndItsResultIsRecorded" passes

  Scenario: Nonzero probe status remains visible
    Then k-shape-progress-probe-result: the existing shaping regression "TestShapeProbeFailuresAreFindingsNotErrors" passes
