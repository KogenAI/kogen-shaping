Feature: ExitCode of wrapped errors is pinned by tests
  This feature checks that the tests of shape/context_test.go pin ExitCode for wrapped Kogen
  errors, wrapped interruptions and unrelated wrapped errors. It runs those tests in a scratch
  copy of the shape package, first against the unchanged ExitCode, which they must pass, and then
  against a wrong version of ExitCode, which at least one of them must reject.

  Scenario: A direct type assertion for a Kogen error is rejected
    Given exit-code-wrapped-errors-tested: a scratch copy of the shape package
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "the unchanged ExitCode"
    Then exit-code-wrapped-errors-tested: such tests exist and they all pass
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "direct type assertion for a Kogen error"
    Then exit-code-wrapped-errors-tested: at least one such test fails

  Scenario: Missing interruption handling is rejected
    Given exit-code-wrapped-errors-tested: a scratch copy of the shape package
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "the unchanged ExitCode"
    Then exit-code-wrapped-errors-tested: such tests exist and they all pass
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "no interruption handling"
    Then exit-code-wrapped-errors-tested: at least one such test fails

  Scenario: A direct type assertion for an interruption is rejected
    Given exit-code-wrapped-errors-tested: a scratch copy of the shape package
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "the unchanged ExitCode"
    Then exit-code-wrapped-errors-tested: such tests exist and they all pass
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "direct type assertion for an interruption"
    Then exit-code-wrapped-errors-tested: at least one such test fails

  Scenario: A fixed code for every interruption is rejected
    Given exit-code-wrapped-errors-tested: a scratch copy of the shape package
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "the unchanged ExitCode"
    Then exit-code-wrapped-errors-tested: such tests exist and they all pass
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "one fixed code for every interruption"
    Then exit-code-wrapped-errors-tested: at least one such test fails

  Scenario: Treating any wrapped error as a Kogen error is rejected
    Given exit-code-wrapped-errors-tested: a scratch copy of the shape package
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "the unchanged ExitCode"
    Then exit-code-wrapped-errors-tested: such tests exist and they all pass
    When exit-code-wrapped-errors-tested: the tests of shape/context_test.go that call ExitCode run against "any wrapped error is a Kogen error"
    Then exit-code-wrapped-errors-tested: at least one such test fails
