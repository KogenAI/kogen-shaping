Feature: ShellQuote is pinned by tests with hostile and multiline values
  This feature checks that the new tests of shape/context_test.go pin ShellQuote under /bin/sh for
  empty, spaced, apostrophe, command substitution and line break values. It runs those tests in a
  scratch copy of the shape package, first against the unchanged ShellQuote, which they must pass,
  and then against a wrong version of ShellQuote, which at least one of them must reject.

  Scenario: An empty value that gives no argument is rejected
    Given shell-quote-metacharacter-tests: a scratch copy of the shape package
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the unchanged ShellQuote"
    Then shell-quote-metacharacter-tests: such tests exist and they all pass
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "an empty value gives no argument"
    Then shell-quote-metacharacter-tests: at least one such test fails

  Scenario: A value with spaces that is split is rejected
    Given shell-quote-metacharacter-tests: a scratch copy of the shape package
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the unchanged ShellQuote"
    Then shell-quote-metacharacter-tests: such tests exist and they all pass
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "spaces split the value"
    Then shell-quote-metacharacter-tests: at least one such test fails

  Scenario: An apostrophe escaped with a backslash is rejected
    Given shell-quote-metacharacter-tests: a scratch copy of the shape package
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the unchanged ShellQuote"
    Then shell-quote-metacharacter-tests: such tests exist and they all pass
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "an apostrophe is escaped with a backslash"
    Then shell-quote-metacharacter-tests: at least one such test fails

  Scenario: An apostrophe that is doubled is rejected
    Given shell-quote-metacharacter-tests: a scratch copy of the shape package
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the unchanged ShellQuote"
    Then shell-quote-metacharacter-tests: such tests exist and they all pass
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "an apostrophe is doubled"
    Then shell-quote-metacharacter-tests: at least one such test fails

  Scenario: Double quotes that leave command substitution active are rejected
    Given shell-quote-metacharacter-tests: a scratch copy of the shape package
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the unchanged ShellQuote"
    Then shell-quote-metacharacter-tests: such tests exist and they all pass
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "double quotes leave shell syntax active"
    Then shell-quote-metacharacter-tests: at least one such test fails

  Scenario: A quote that ends at a line break is rejected
    Given shell-quote-metacharacter-tests: a scratch copy of the shape package
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the unchanged ShellQuote"
    Then shell-quote-metacharacter-tests: such tests exist and they all pass
    When shell-quote-metacharacter-tests: the new tests of shape/context_test.go that call ShellQuote run against "the quote ends at a line break"
    Then shell-quote-metacharacter-tests: at least one such test fails
