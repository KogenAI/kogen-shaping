Feature: Refine may correct the Intent's own controls file
  This feature checks a scripted Refine session on a fixture Intent draft whose controls file holds a home path.

  Scenario: A controls file without a home path replaces the one with a home path
    When shape-refine-accepts-controls-file-edit: the "accept" journey runs
    Then shape-refine-accepts-controls-file-edit: Refine handles the controls file as specified

  Scenario: A controls file whose wrong-implementation control is weaker is refused
    When shape-refine-accepts-controls-file-edit: the "weak" journey runs
    Then shape-refine-accepts-controls-file-edit: Refine handles the controls file as specified

  Scenario: An edit outside the owned files is refused with the existing text, the refused path and the owned files
    When shape-refine-accepts-controls-file-edit: the "outside" journey runs
    Then shape-refine-accepts-controls-file-edit: Refine handles the controls file as specified
