Feature: The fix line of a refused invented literal follows the kind of the literal
  This feature runs the shape package's own handling of an invented literal that no source states
  and that the acceptance drives, and reads the fix line and the refusal line it gives.

  Scenario: A file path is told to list the directory at run time
    When shape-invented-fix-names-kind-remedy: the shape package gives each of the four invented literals its fix line and refusal line
    Then shape-invented-fix-names-kind-remedy: the fix line for "test/acceptance/external-stall-outcome.test.cjs" contains "list the directory at run time"
    And shape-invented-fix-names-kind-remedy: the fix line for "test/acceptance/external-stall-outcome.test.cjs" contains "do not assert the file name"
    And shape-invented-fix-names-kind-remedy: the fix line for "test/acceptance/external-stall-outcome.test.cjs" does not contain "visible text"

  Scenario: A name of new code that starts with alias is told to record a Decision
    When shape-invented-fix-names-kind-remedy: the shape package gives each of the four invented literals its fix line and refusal line
    Then shape-invented-fix-names-kind-remedy: the fix line for "alias Studio.Loop.Fairness" contains "Decision of intent.md"
    And shape-invented-fix-names-kind-remedy: the fix line for "alias Studio.Loop.Fairness" contains "approval card"
    And shape-invented-fix-names-kind-remedy: the fix line for "alias Studio.Loop.Fairness" does not contain "visible text"
    And shape-invented-fix-names-kind-remedy: the fix line for "alias Studio.Loop.Fairness" does not contain "list the directory"

  Scenario: A single token name of new code gets the same remedy
    When shape-invented-fix-names-kind-remedy: the shape package gives each of the four invented literals its fix line and refusal line
    Then shape-invented-fix-names-kind-remedy: the fix line for "hard_quota" contains "Decision of intent.md"
    And shape-invented-fix-names-kind-remedy: the fix line for "hard_quota" is the fix line for "alias Studio.Loop.Fairness" with its own literal
    And shape-invented-fix-names-kind-remedy: the fix line for "hard_quota" does not contain "visible text"

  Scenario: Any other literal keeps the visible text sentence
    When shape-invented-fix-names-kind-remedy: the shape package gives each of the four invented literals its fix line and refusal line
    Then shape-invented-fix-names-kind-remedy: the fix line for "The table has one column each for Shape, Build and Review" is byte for byte:
      """
      - fix for invented literal "The table has one column each for Shape, Build and Review": find it at run time (for example find the form or link by its visible text), assert only what the request states, or move it to a Decision and stop asserting it
      """

  Scenario Outline: The refusal line of each literal is unchanged
    When shape-invented-fix-names-kind-remedy: the shape package gives each of the four invented literals its fix line and refusal line
    Then shape-invented-fix-names-kind-remedy: the refusal line for "<literal>" is byte for byte:
      """
      - invented_contract | scenario: S | literal: "<literal>" | why: W | no source states it: not the request, the answers or a tracked file
      """

    Examples:
      | literal                                                    |
      | test/acceptance/external-stall-outcome.test.cjs            |
      | alias Studio.Loop.Fairness                                 |
      | hard_quota                                                 |
      | The table has one column each for Shape, Build and Review  |
