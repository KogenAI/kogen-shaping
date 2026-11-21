Feature: Refine keeps every original control or states why it is superseded
  This feature checks scripted Refine sessions on a fixture Intent draft that has two working wrong-implementation controls, wrong-even in scenario Even numbers and wrong-odd in scenario Odd numbers.

  Scenario: A replacement that drops a control is refused and names it
    When shape-refine-rejects-dropped-control: the "drop" journey runs
    Then shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified

  Scenario: A dropped control superseded with a reason is accepted
    When shape-refine-rejects-dropped-control: the "supersede" journey runs
    Then shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified

  Scenario: A supersede entry with an empty reason is refused
    When shape-refine-rejects-dropped-control: the "noreason" journey runs
    Then shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified

  Scenario: A supersede entry that names a control never in the original is refused
    When shape-refine-rejects-dropped-control: the "unknown" journey runs
    Then shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified

  Scenario: A replacement that keeps both controls and changes one is accepted
    When shape-refine-rejects-dropped-control: the "keep" journey runs
    Then shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified

  Scenario: A replacement that keeps both controls but weakens one is still refused by the wrong-implementation check
    When shape-refine-rejects-dropped-control: the "keepweak" journey runs
    Then shape-refine-rejects-dropped-control: Refine judges the replacement controls as specified
