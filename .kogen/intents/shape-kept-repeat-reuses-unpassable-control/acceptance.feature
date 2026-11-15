Feature: Reuse completed unpassable control observations
  This feature reproduces the duplicate unpassable_check seen in AoR climb release
  30cf6619, cell h9-shape-sonnet-high-vs-baseline-30cf6619/cells/rails-ft-saas-billing__shape-sonnet-high,
  and its __r2 and __r3 cells: round 1 had one event and rounds 2 and 3 each had
  two identical stand_in_used events for features/steps/rails-ft-saas-billing__billing.js
  line 13. The run was refused at 744 s; acceptance_executions was 5 for 3 packages.
  The repeats ran at 430 to 462 s and 669 to 702 s and ended persistent kept assertion red.
  The executable local analogue counts actual control subprocesses and readiness runs,
  not elapsed time or external historical artifacts.

  Background:
    Given shape-kept-repeat-reuses-unpassable-control: the local control fixture is available

  Scenario: Reuse the control during a kept assertion repeat
    When shape-kept-repeat-reuses-unpassable-control: the "repeat" journey runs
    Then shape-kept-repeat-reuses-unpassable-control: the observation is
      """
      {"controls":1,"readiness":2,"events":1,"repeats":1,"same_package":true,"same_tree":true,"statuses":["failed","failed"]}
      """

  Scenario: Invalidate reuse after changing package bytes
    When shape-kept-repeat-reuses-unpassable-control: the "package" journey runs
    Then shape-kept-repeat-reuses-unpassable-control: the observation is
      """
      {"controls":2,"readiness":2,"events":2,"repeats":0,"same_package":false,"same_tree":false,"statuses":["failed","failed"]}
      """

  Scenario: Invalidate reuse after changing the candidate tree
    When shape-kept-repeat-reuses-unpassable-control: the "tree" journey runs
    Then shape-kept-repeat-reuses-unpassable-control: the observation is
      """
      {"controls":2,"readiness":2,"events":2,"repeats":0,"same_package":true,"same_tree":false,"statuses":["failed","failed"]}
      """

  Scenario: Retry a skipped control on the repeat
    When shape-kept-repeat-reuses-unpassable-control: the "skipped" journey runs
    Then shape-kept-repeat-reuses-unpassable-control: the observation is
      """
      {"controls":2,"readiness":2,"events":2,"repeats":1,"skipped":true,"same_package":true,"same_tree":true,"statuses":["failed","failed"]}
      """

  Scenario: Preserve a completed refusal exactly
    When shape-kept-repeat-reuses-unpassable-control: the "refusal" journey runs
    Then shape-kept-repeat-reuses-unpassable-control: the observation is
      """
      {"controls":1,"readiness":0,"events":1,"repeats":0,"refusal":true,"refusal_text":"Acceptance has steps that no implementation can pass: 1 step definition(s) fail the same way whatever the repository contains, so no pass is reachable; nothing was published. Each step must be able to pass once the requested behaviour exists: run the program, call the code or inspect its output, and compare what it observed with an assertion that shows the actual value (for example assert.strictEqual). Rewrite these steps; resubmitting them unchanged is refused again.\n- acceptance.feature line 3: Then step kept (step definition \"step kept\", features/steps/is_even_steps.js line 2): no pass is reachable: it fails with \"Error: stub\" whatever the repository contains"}
      """

  Scenario: Retry after a control execution error
    When shape-kept-repeat-reuses-unpassable-control: the "error" journey runs
    Then shape-kept-repeat-reuses-unpassable-control: the observation is
      """
      {"controls":1,"readiness":0,"events":1,"repeats":0,"error":true}
      """
