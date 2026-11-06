Feature: Shape's scope gate counts the Intent package, not the HEAD size of the files it names
  The scope gate estimates one review of a candidate Intent from the package's own bytes:
  intent.md, every Acceptance: path and the declared model. A tracked file the Intent names in
  backticks no longer adds its size at HEAD. The bound stays tools.InputBudgetBytes (524288 bytes),
  inclusive, and a package whose own bytes are over it is refused with the existing refusal line,
  recorded before the decision check, ending not_ready as before. Each case is a small synthetic
  repository shaped by a scripted shaper.

  Scenario: A package under the bound that names a tracked file larger than the bound passes the scope gate
    When k-shape-scope-gate-counts-package: the "named" case is shaped
    Then k-shape-scope-gate-counts-package: the run used a review bound of 524288 bytes
    And k-shape-scope-gate-counts-package: the Intent names the tracked file data/big.json, which is larger than the bound
    And k-shape-scope-gate-counts-package: the scope check refused 0 submissions
    And k-shape-scope-gate-counts-package: the result of submission 1 does not contain "Scope check"
    And k-shape-scope-gate-counts-package: the decision check ran 2 times
    And k-shape-scope-gate-counts-package: the Intent is published

  Scenario: A package one byte over the bound is refused with the existing refusal line, and at exactly the bound it fits
    When k-shape-scope-gate-counts-package: the "edge" case is shaped
    Then k-shape-scope-gate-counts-package: the run used a review bound of 524288 bytes
    And k-shape-scope-gate-counts-package: the Intent names the tracked file data/big.json, which is larger than the bound
    And k-shape-scope-gate-counts-package: the result of submission 1 is:
      """
      Scope check: the Intent is estimated at 524289 bytes of one review (scenarios 5, acceptance harnesses 2, named files 1 in areas 1), over the 524288-byte bound of one review (tools.InputBudgetBytes).
      - .kogen/intents/is-even/intent.md: 4097 bytes (Intent package)
      - .kogen/intents/is-even/acceptance.feature: 346 bytes (Intent package)
      - features/steps/is-even__steps.js: 210 bytes (Intent package)
      - features/steps/is-even__padding.js: 519516 bytes (Intent package)
      - .kogen/intents/is-even/model.qnt: 120 bytes (Intent package)
      Above the bound, split the work into an ordered ## Stages plan of at least two stages, each with a real gate in its checks and each inside the bound, or narrow the Intent to the requested behaviour. A stage's goal names in backticks the files that stage changes; a named file that no stage goal names counts in every stage, and every stage carries the whole Intent package. Do not satisfy the bound by deleting requested behaviour: every behaviour and example the request asks for keeps its scenario. Resubmitting this candidate unchanged is refused again.
      """
    And k-shape-scope-gate-counts-package: the scope check refused 1 submission
    And k-shape-scope-gate-counts-package: the decision check ran at least 1 time

  Scenario: The estimate counts the declared model and every outline row as before
    When k-shape-scope-gate-counts-package: the "edge" case is shaped
    Then k-shape-scope-gate-counts-package: the result of submission 1 contains "estimated at 524289 bytes"
    And k-shape-scope-gate-counts-package: the result of submission 1 contains "(scenarios 5,"
    And k-shape-scope-gate-counts-package: the result of submission 1 contains "- .kogen/intents/is-even/model.qnt: 120 bytes (Intent package)"

  Scenario: A named path that is not tracked at HEAD or lies in .kogen/ is not counted
    When k-shape-scope-gate-counts-package: the "edge" case is shaped
    Then k-shape-scope-gate-counts-package: the result of submission 1 contains "named files 1 in areas 1)"
    And k-shape-scope-gate-counts-package: the result of submission 1 does not contain "missing/new.js"
    And k-shape-scope-gate-counts-package: the result of submission 1 does not contain ".kogen/notes.txt"
    And k-shape-scope-gate-counts-package: the result of submission 1 does not contain "data/big.json"

  Scenario: A package over the bound is refused before the decision check, recorded, and shaping ends not_ready
    When k-shape-scope-gate-counts-package: the "over" case is shaped
    Then k-shape-scope-gate-counts-package: the run used a review bound of 524288 bytes
    And k-shape-scope-gate-counts-package: the result of submission 1 starts with:
      """
      Scope check: the Intent is estimated at 526213 bytes of one review (scenarios 2, acceptance harnesses 2, named files 0 in areas 0), over the 524288-byte bound of one review (tools.InputBudgetBytes).

      """
    And k-shape-scope-gate-counts-package: the result of submission 1 ends with:
      """
      Do not satisfy the bound by deleting requested behaviour: every behaviour and example the request asks for keeps its scenario. Resubmitting this candidate unchanged is refused again.
      """
    And k-shape-scope-gate-counts-package: the result of submission 2 ends with:
      """
      Resubmitting this candidate unchanged is refused again.
      Unchanged: this package is identical to the one refused at attempt 1; repair it before submitting again.
      """
    And k-shape-scope-gate-counts-package: the scope check refused 3 submissions
    And k-shape-scope-gate-counts-package: the decision check ran 0 times
    And k-shape-scope-gate-counts-package: shaping ended with intent/not_ready and exit 1

  Scenario: The delivered scope tests pass with the package-only count
    When k-shape-scope-gate-counts-package: the delivered scope tests are run
    Then k-shape-scope-gate-counts-package: the delivered scope tests passed
