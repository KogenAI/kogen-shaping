Feature: Batch the step-ownership check per step-file pair

  The step-ownership check of approve, preflight and Refine starts at most one Node + Cucumber
  loader per (own step file, other step file) pair and matches every step text of both Intents in
  it, with the release's verdicts. Each case runs in a scratch Git repository whose KOGEN_NODE
  wrapper logs one line per invocation and then runs the real Node with the same arguments.

  Scenario: Thirty earlier Intents without collisions cost at most one loader per step-file pair
    When k-approve-binding-check-batched: the "pairs" case runs
    Then k-approve-binding-check-batched: "syn-new" is approved after its preview and the approval with the preview's digest
    And k-approve-binding-check-batched: the preview's step-ownership check of "syn-new" started between 1 and 30 Node loaders
    And k-approve-binding-check-batched: preflight's step-ownership check of "syn-new" started between 1 and 30 Node loaders and its package check passed

  Scenario: A copy of syn-17's binding in syn-new's step file is refused with the release's text
    When k-approve-binding-check-batched: the "one-collision" case runs
    Then k-approve-binding-check-batched: the preview of "syn-new" is refused with exit 1 and this rendered error:
      """
      intent/step_ownership_collision: Intents syn-new and syn-17 have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/syn-17__steps.js
        features/steps/syn-new__steps.js
        features/steps/syn-17__steps.js
      """
    And k-approve-binding-check-batched: preflight of "syn-new" reports exactly this step-ownership blocker:
      """
      {"class": "intent", "reason": "step_ownership_collision", "detail": "Intents syn-new and syn-17 have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/syn-17__steps.js", "paths": ["features/steps/syn-new__steps.js", "features/steps/syn-17__steps.js"]}
      """

  Scenario: Of two collisions the first in Intent order is refused
    When k-approve-binding-check-batched: the "two-collisions" case runs
    Then k-approve-binding-check-batched: the preview of "syn-new" is refused with exit 1 and this rendered error:
      """
      intent/step_ownership_collision: Intents syn-new and syn-03 have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/syn-03__steps.js
        features/steps/syn-new__steps.js
        features/steps/syn-03__steps.js
      """
    And k-approve-binding-check-batched: preflight of "syn-new" reports exactly this step-ownership blocker:
      """
      {"class": "intent", "reason": "step_ownership_collision", "detail": "Intents syn-new and syn-03 have overlapping step bindings in features/steps/syn-new__steps.js and features/steps/syn-03__steps.js", "paths": ["features/steps/syn-new__steps.js", "features/steps/syn-03__steps.js"]}
      """

  Scenario: A step file that throws on load causes no refusal
    When k-approve-binding-check-batched: the "load-failure" case runs
    Then k-approve-binding-check-batched: "syn-new" is approved after its preview and the approval with the preview's digest
    And k-approve-binding-check-batched: the preview's step-ownership check of "syn-new" started between 1 and 30 Node loaders

  Scenario: Two Intents shaped in separate worktrees both approve from one checkout
    When k-approve-binding-check-batched: the "worktrees" case runs
    Then k-approve-binding-check-batched: "syn-alpha" is approved after its preview and the approval with the preview's digest
    And k-approve-binding-check-batched: "syn-beta" is approved after its preview and the approval with the preview's digest
    And k-approve-binding-check-batched: preflight's step-ownership check of "syn-alpha" started between 1 and 1 Node loaders and its package check passed
    And k-approve-binding-check-batched: preflight's step-ownership check of "syn-beta" started between 1 and 1 Node loaders and its package check passed
