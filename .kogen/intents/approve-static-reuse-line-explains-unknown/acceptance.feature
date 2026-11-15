Feature: The approve static reuse line explains the reused outcome
  This feature checks the progress line that approve prints when it reuses the static-check result recorded for the package.

  Scenario: A recorded checked outcome keeps its line
    Given approve-static-reuse-line-explains-unknown: approve has reused a recorded checked, unknown and none static-check result
    Then approve-static-reuse-line-explains-unknown: the checked reuse line is "kogen: static checks: reusing the result recorded for this package (checked)"

  Scenario: A recorded unknown outcome is reused and explained
    Given approve-static-reuse-line-explains-unknown: approve has reused a recorded checked, unknown and none static-check result
    Then approve-static-reuse-line-explains-unknown: the configured check ran 1 time in total
    And approve-static-reuse-line-explains-unknown: the unknown reuse line is "kogen: static checks: reusing the result recorded for this package (unknown: a check fails without the shaped files too or could not run, so the shaped files are not established by it; see static-checks.json in the Shape record at <record path>)"

  Scenario: A recorded none outcome says so in words
    Given approve-static-reuse-line-explains-unknown: approve has reused a recorded checked, unknown and none static-check result
    Then approve-static-reuse-line-explains-unknown: the none reuse line is "kogen: static checks: reusing the result recorded for this package (none: no static check is configured)"
