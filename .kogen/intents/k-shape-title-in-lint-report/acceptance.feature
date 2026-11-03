Feature: k-shape-title-in-lint-report: Restore final-attempt title fallback
  Scenario: Pinned coordinator preserves final-attempt fallback and mixed-problem refusal
    When k-shape-title-in-lint-report: the title regressions run with coordinator "pinned"
    Then k-shape-title-in-lint-report: all title regressions pass exactly once

  Scenario: Main coordinator full reports preserve final-attempt fallback and mixed-problem refusal
    When k-shape-title-in-lint-report: the title regressions run with coordinator "main"
    Then k-shape-title-in-lint-report: all title regressions pass exactly once

  Scenario: Changed title wording before unrelated problems still shortens
    When k-shape-title-in-lint-report: the title regressions run with coordinator "main-title-first"
    Then k-shape-title-in-lint-report: all title regressions pass exactly once

  Scenario: Changed title wording after unrelated problems still shortens
    When k-shape-title-in-lint-report: the title regressions run with coordinator "main-title-last"
    Then k-shape-title-in-lint-report: all title regressions pass exactly once
