Feature: Refuse acceptance steps no implementation can pass
  Shape refuses an acceptance step whose failure no implementation and no repository state can turn
  into a pass, names it and says why, and keeps approving steps that fail only because the feature
  is missing.

  Scenario: A step that throws unconditionally is refused and not published
    When the unpassable-step journey runs in "stub" mode
    Then the unpassable-step contract passes

  Scenario: A step that asserts false after reading a file is refused
    When the unpassable-step journey runs in "assertfalse" mode
    Then the unpassable-step contract passes

  Scenario: A step that calls a missing function stays approved
    When the unpassable-step journey runs in "missing" mode
    Then the unpassable-step contract passes

  Scenario: A step that passes for another input value stays approved
    When the unpassable-step journey runs in "inputs" mode
    Then the unpassable-step contract passes

  Scenario: A candidate whose only step is a stub is refused despite a clean contract check
    When the unpassable-step journey runs in "only" mode
    Then the unpassable-step contract passes

  Scenario: A step waiting on an unavailable service is an environment problem
    When the unpassable-step journey runs in "service" mode
    Then the unpassable-step contract passes

  Scenario: A repaired step is accepted on the next submission
    When the unpassable-step journey runs in "repair" mode
    Then the unpassable-step contract passes
