Feature: Shape repairs named failing checks on a red main
  This feature runs Shape on a synthetic fixture repository.

  Scenario: A declared repair of the red check is the target
    When shape-repair-red-main-intent: Shape processes "target"
    Then shape-repair-red-main-intent: the repair policy holds

  Scenario: Every other check stays kept
    When shape-repair-red-main-intent: Shape processes "keeps"
    Then shape-repair-red-main-intent: the repair policy holds

  Scenario: A named check that is green is refused
    When shape-repair-red-main-intent: Shape processes "green"
    Then shape-repair-red-main-intent: the repair policy holds

  Scenario: Own time and timeout are reported apart
    When shape-repair-red-main-intent: Shape processes "timing"
    Then shape-repair-red-main-intent: the repair policy holds

  Scenario: A red main without a declaration is refused
    When shape-repair-red-main-intent: Shape processes "norepair"
    Then shape-repair-red-main-intent: the repair policy holds
