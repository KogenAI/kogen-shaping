Feature: Isolate step files and bindings across worktrees
  New packages own their step files and text. Existing approved sharing is usable
  only while its dependencies remain unchanged. These component journeys run the
  existing Shape and Approve command implementations with real Git and Cucumber.

  Scenario: Shape concurrently and land either order by merge or rebase
    When the step-file ownership journey runs in "parallel" mode
    Then the step-file ownership journey satisfies the contract

  Scenario: Refuse a resolved merge that adopts the other Intent's step file
    When the step-file ownership journey runs in "foreign" mode
    Then the step-file ownership journey satisfies the contract

  Scenario: Refuse mixed step text even with separate filenames
    When the step-file ownership journey runs in "text" mode
    Then the step-file ownership journey satisfies the contract

  Scenario: Keep unchanged legacy approved shared files usable
    When the step-file ownership journey runs in "legacy" mode
    Then the step-file ownership journey satisfies the contract

  Scenario: Refuse changed legacy approved shared bytes
    When the step-file ownership journey runs in "legacy-change" mode
    Then the step-file ownership journey satisfies the contract

  Scenario: Refuse newly introduced sharing of a grandfathered file
    When the step-file ownership journey runs in "new-sharing" mode
    Then the step-file ownership journey satisfies the contract
