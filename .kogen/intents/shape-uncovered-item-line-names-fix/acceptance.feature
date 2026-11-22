Feature: An uncovered request item says how to fix it
  This feature checks the lines that follow the refusal lines for an item no wrong implementation of controls.json names a scenario of and for an item no scenario exercises.

  Scenario: A kept item whose scenario no wrong implementation names gets a fix line
    Given shape-uncovered-item-line-names-fix: a kept item named "Existing tool exports stay published and free of sponsors" with request text "Existing tool exports stay published and free of sponsors" declares 1 scenario and a wrong implementation names only another scenario
    Then shape-uncovered-item-line-names-fix: the lines are exactly:
      """
      - Existing tool exports stay published and free of sponsors (kept) | request: Existing tool exports stay published and free of sponsors | problem: no wrong implementation in controls.json names a scenario of it. Add a wrong implementation to controls.json that breaks this unchanged behaviour and names a scenario of it and the step line that must reject it.
      - fix for property Existing tool exports stay published and free of sponsors: add a wrong implementation to controls.json whose scenario is "Add keeps working" and whose step is the Then step that fails when the behaviour is broken; the wrong implementation must break the kept behaviour
      """

  Scenario: A new item gets a fix line without the kept sentence
    Given shape-uncovered-item-line-names-fix: a new item named "Two blocked steps are joined by and" with request text "Two blocked steps are joined by and" declares 1 scenario and a wrong implementation names only another scenario
    Then shape-uncovered-item-line-names-fix: there are 2 lines
    And shape-uncovered-item-line-names-fix: line 2 contains "- fix for property Two blocked steps are joined by and: add a wrong implementation to controls.json whose scenario is \"Add keeps working\""
    And shape-uncovered-item-line-names-fix: line 2 does not contain "kept behaviour"

  Scenario: An item with no declared scenario is told to add a scenario
    Given shape-uncovered-item-line-names-fix: a new item named "Two blocked steps are joined by and" with request text "Two blocked steps are joined by and" declares 0 scenarios and no wrong implementation is given
    Then shape-uncovered-item-line-names-fix: there are 2 lines
    And shape-uncovered-item-line-names-fix: line 2 contains "- fix for property Two blocked steps are joined by and: add a scenario for it to acceptance.feature"
    And shape-uncovered-item-line-names-fix: line 2 does not contain "Add keeps working"

  Scenario: Many declared scenarios are quoted three and then counted
    Given shape-uncovered-item-line-names-fix: a kept item named "Existing tool exports stay published and free of sponsors" with request text "Existing tool exports stay published and free of sponsors" declares 5 scenarios and no wrong implementation is given
    Then shape-uncovered-item-line-names-fix: there are 2 lines
    And shape-uncovered-item-line-names-fix: line 2 contains "whose scenario is one of \"Add keeps working\", \"Mul works\", \"Sub works\" and 2 more and whose step"
    And shape-uncovered-item-line-names-fix: line 2 does not contain "Neg works"

  Scenario: An item no scenario exercises gets a fix line
    Given shape-uncovered-item-line-names-fix: a new item named "Index linked from workspace context and initialize instructions" with request text "Index linked from workspace context and initialize instructions" has a confirmed uncovered problem
    Then shape-uncovered-item-line-names-fix: the lines are exactly:
      """
      - Index linked from workspace context and initialize instructions (new) | request: Index linked from workspace context and initialize instructions | problem: no scenario exercises it | confirmed: no scenario
      - fix for property Index linked from workspace context and initialize instructions: add a scenario whose Then step observes it, or, when no automatic step can observe it, record in a Decision that it needs human evaluation
      """

  Scenario: A covered item gets no fix line
    Given shape-uncovered-item-line-names-fix: a kept item named "Existing tool exports stay published and free of sponsors" with request text "Existing tool exports stay published and free of sponsors" declares 1 scenario and a wrong implementation rejects it at its step
    Then shape-uncovered-item-line-names-fix: there are 0 lines

  Scenario: Another problem kind gets no fix line
    Given shape-uncovered-item-line-names-fix: a new item named "Two blocked steps are joined by and" with request text "Two blocked steps are joined by and" has a confirmed passes_unchanged problem
    Then shape-uncovered-item-line-names-fix: there are 1 line
    And shape-uncovered-item-line-names-fix: no line starts with "- fix for"
