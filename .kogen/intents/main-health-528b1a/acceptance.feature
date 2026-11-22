Feature: The blocked report's Next line names the project definition
  This feature runs this module's real Shape command with a scripted shaper on small synthetic
  repositories. A blocked report prints three lines whose third line is one fixed sentence
  that names .kogen/project.yaml, as the assembled Kogen shaping spec expects.

  Scenario: A blocked report for a protected style configuration prints the Next line
    When main-health-528b1a: the blocked report for "style" runs
    Then main-health-528b1a: the blocked report prints the Next line naming the project definition

  Scenario: A blocked report for a glob-protected path prints the same Next line
    When main-health-528b1a: the blocked report for "glob" runs
    Then main-health-528b1a: the blocked report prints the Next line naming the project definition

  Scenario: A blocked report for the project definition prints the same Next line
    When main-health-528b1a: the blocked report for "project" runs
    Then main-health-528b1a: the blocked report prints the Next line naming the project definition

  Scenario: The blocked-report repair feature journeys hold the same Next line
    When main-health-528b1a: the repair feature journeys run
    Then main-health-528b1a: the repair feature journeys passed with the project definition Next line

  Scenario: The Go sources hold only the Next line naming the project definition
    When main-health-528b1a: the Go sources are read
    Then main-health-528b1a: the Go sources hold only the Next line naming the project definition
