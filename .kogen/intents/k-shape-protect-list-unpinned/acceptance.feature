Feature: k-shape-protect-list-unpinned: coordinator-owned baseline
  Scenario Outline: Consumer regressions tolerate coordinator baseline evolution
    When k-shape-protect-list-unpinned: I run the regression suite with coordinator "<coordinator>" and extra automatic path "<path>"
    Then k-shape-protect-list-unpinned: the regression suite exits 0

    Examples:
      | coordinator | path                        |
      | pinned      |                             |
      | 482d565     |                             |
      | 482d565     | .kogen/future-baseline.yaml |
