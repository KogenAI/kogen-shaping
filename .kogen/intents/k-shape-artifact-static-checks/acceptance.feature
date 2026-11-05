Feature: Shape establishes that its generated acceptance files pass the configured static checks before approval

  # Each scenario shapes the request "make Count return the slice length" in a small synthetic Go
  # repository (module example.test/staticshape, go 1.27, count/count.go returning 0) with a
  # scripted shaper, the repository's .kogen/project.yaml and the existing configured Cucumber
  # installation, through features/steps/k-shape-artifact-static-checks__harness.js; the Intent
  # quotes every fixture file verbatim. The configured checks are
  #   layout:   test -z "$(gofmt -l count features/steps)"
  #   analysis: go vet ./...
  # The "renamed" and "valid" cases name the same two commands tidy and inspect and put the helper
  # at features/steps/static-proof__counter/main.go. Readiness of the shaped Intent is legitimately
  # red: Count returns 0, so its scenario fails with '0' !== '2'.

  Scenario: An unformatted helper is refused naming layout, unchanged and as an acknowledgment too, until its gofmt output replaces it
    When k-shape-artifact-static-checks: the "repair" case is shaped
    Then k-shape-artifact-static-checks: submission 1 is refused by the static check "layout" with exit 1 naming "features/steps/static-proof__driver/main.go"
    And k-shape-artifact-static-checks: submission 1 shows the output of the static check "layout" containing "(no output)"
    And k-shape-artifact-static-checks: submission 1 names no static check "analysis"
    And k-shape-artifact-static-checks: submission 2 is refused by the static check "layout" with exit 1 naming "features/steps/static-proof__driver/main.go"
    And k-shape-artifact-static-checks: submission 2 is told the package is unchanged since submission 1
    And k-shape-artifact-static-checks: submission 3 is refused by the static check "layout" with exit 1 naming "features/steps/static-proof__driver/main.go"
    And k-shape-artifact-static-checks: submission 4 passes the static checks "layout" and "analysis" with outcome "checked"
    And k-shape-artifact-static-checks: submission 4 receives readiness observations containing "'0' !== '2'"
    And k-shape-artifact-static-checks: the Intent is published after 5 submissions
    And k-shape-artifact-static-checks: the published file "features/steps/static-proof__driver/main.go" is the following, each "<TAB>" standing for a tab:
      """
      package main

      import (
      <TAB>"fmt"

      <TAB>"example.test/staticshape/count"
      )

      func main() {
      <TAB>fmt.Println(count.Count([]string{"a", "b"}))
      }
      """
    And k-shape-artifact-static-checks: the file "count/count.go" is unchanged
    And k-shape-artifact-static-checks: the last fresh static checks ran outside the checkout on the published bytes
    And k-shape-artifact-static-checks: the static check record identifies the published package with outcome "checked"
    And k-shape-artifact-static-checks: approving the published package succeeds

  Scenario: A vet diagnostic in an uncalled, gofmt-clean helper function is refused naming analysis until it is repaired
    When k-shape-artifact-static-checks: the "vet" case is shaped
    Then k-shape-artifact-static-checks: submission 1 is refused by the static check "analysis" with exit 1 naming "features/steps/static-proof__driver/main.go:14"
    And k-shape-artifact-static-checks: submission 1 shows the output of the static check "analysis" containing 'fmt.Printf format %d has arg "not-a-number" of wrong type string'
    And k-shape-artifact-static-checks: submission 1 names no static check "layout"
    And k-shape-artifact-static-checks: submission 2 passes the static checks "layout" and "analysis" with outcome "checked"
    And k-shape-artifact-static-checks: submission 2 receives readiness observations containing "'0' !== '2'"
    And k-shape-artifact-static-checks: the Intent is published after 3 submissions

  Scenario: A configured fixer repairs the helper, only the fixed bytes are accepted, and a later change cannot use the old check or hash
    When k-shape-artifact-static-checks: the "fixer" case is shaped
    Then k-shape-artifact-static-checks: submission 1 passes the static checks "layout" and "analysis" with outcome "checked"
    And k-shape-artifact-static-checks: the Intent is published after 2 submissions
    And k-shape-artifact-static-checks: the published file "features/steps/static-proof__driver/main.go" is the following, each "<TAB>" standing for a tab:
      """
      package main

      import (
      <TAB>"example.test/staticshape/count"
      <TAB>"fmt"
      )

      func main() { fmt.Println(count.Count([]string{"a", "b"})) }
      """
    And k-shape-artifact-static-checks: the file "go.mod" is unchanged
    And k-shape-artifact-static-checks: the file "count/count.go" is unchanged
    And k-shape-artifact-static-checks: the file ".kogen/project.yaml" is unchanged
    And k-shape-artifact-static-checks: the last fresh static checks ran outside the checkout on the published bytes
    And k-shape-artifact-static-checks: the static check record identifies the published package with outcome "checked"
    And k-shape-artifact-static-checks: approving the published hash after the helper changed is refused
    And k-shape-artifact-static-checks: approving the changed package's hash is refused by the static check "analysis" naming "features/steps/static-proof__driver/main.go:11"

  Scenario: An unrelated pre-existing layout failure is reported with its path, not attributed to the helper and not claimed as passed
    When k-shape-artifact-static-checks: the "unrelated" case is shaped
    Then k-shape-artifact-static-checks: submission 1 is not refused by the static checks
    And k-shape-artifact-static-checks: submission 1 records the static check "layout" as "preexisting" naming "count/unrelated.go"
    And k-shape-artifact-static-checks: submission 1 records the static check "layout" without naming "features/steps/static-proof__driver/main.go"
    And k-shape-artifact-static-checks: submission 1 records the static check "layout" with its configured command
    And k-shape-artifact-static-checks: submission 1 records the static check "analysis" as "passed"
    And k-shape-artifact-static-checks: submission 1 records the static checks outcome "unknown"
    And k-shape-artifact-static-checks: stderr contains "kogen: static check layout failed (exit 1) without the shaped files too: pre-existing failure in count/unrelated.go; the shaped files are not established by this check"
    And k-shape-artifact-static-checks: the Intent is published after 2 submissions
    And k-shape-artifact-static-checks: the file "count/unrelated.go" is unchanged
    And k-shape-artifact-static-checks: the file "count/unrelated.go" reads "package count;var Existing=1"

  Scenario: A check whose checker executable is absent is reported unavailable, not as a successful static check
    When k-shape-artifact-static-checks: the "absent" case is shaped
    Then k-shape-artifact-static-checks: submission 1 records the static check "layout" as "unavailable" with the reason "gofmt not found"
    And k-shape-artifact-static-checks: submission 1 records the static check "analysis" as "passed"
    And k-shape-artifact-static-checks: submission 1 records the static checks outcome "unknown"
    And k-shape-artifact-static-checks: stderr contains "kogen: static check layout unavailable: gofmt not found; the shaped files are not established by this check"
    And k-shape-artifact-static-checks: submission 1 is not refused by the static checks
    And k-shape-artifact-static-checks: the Intent is published after 2 submissions

  Scenario: Renamed checks and a renamed helper path with equivalent defects get the same decisions
    When k-shape-artifact-static-checks: the "renamed" case is shaped
    Then k-shape-artifact-static-checks: submission 1 is refused by the static check "tidy" with exit 1 naming "features/steps/static-proof__counter/main.go"
    And k-shape-artifact-static-checks: submission 1 is refused by the static check "inspect" with exit 1 naming "features/steps/static-proof__counter/main.go:4"
    And k-shape-artifact-static-checks: submission 2 passes the static checks "tidy" and "inspect" with outcome "checked"
    And k-shape-artifact-static-checks: the Intent is published after 3 submissions

  Scenario: A valid first submission receives no repair turn because of the static gate
    When k-shape-artifact-static-checks: the "valid" case is shaped
    Then k-shape-artifact-static-checks: submission 1 passes the static checks "tidy" and "inspect" with outcome "checked"
    And k-shape-artifact-static-checks: submission 1 receives readiness observations containing "'0' !== '2'"
    And k-shape-artifact-static-checks: no submission is refused
    And k-shape-artifact-static-checks: the Intent is published after 2 submissions
