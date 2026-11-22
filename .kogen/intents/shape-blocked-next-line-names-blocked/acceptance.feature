Feature: The blocked report's Next line names no file
  This feature runs this module's real Shape command with a scripted shaper on small synthetic
  repositories. A blocked report prints three lines whose third line is one fixed sentence
  that does not name the project definition.

  Scenario: A blocked report for a glob-protected path prints the new Next line
    When shape-blocked-next-line-names-blocked: the blocked report for "glob" runs
    Then shape-blocked-next-line-names-blocked: the blocked report satisfies the contract

  Scenario: A blocked report for the project definition prints the same Next line
    When shape-blocked-next-line-names-blocked: the blocked report for "project" runs
    Then shape-blocked-next-line-names-blocked: the blocked report satisfies the contract

  Scenario: A blocked report for two dependency manifests prints the same Next line
    When shape-blocked-next-line-names-blocked: the blocked report for "manifest" runs
    Then shape-blocked-next-line-names-blocked: the blocked report satisfies the contract

  Scenario: The blocked-report repair feature journeys hold the new Next line
    When shape-blocked-next-line-names-blocked: the repair feature journeys run
    Then shape-blocked-next-line-names-blocked: the repair feature journeys passed with the new constant

  Scenario: The Go sources hold only the new Next line
    When shape-blocked-next-line-names-blocked: the Go sources are read
    Then shape-blocked-next-line-names-blocked: the Go sources hold only the new line
