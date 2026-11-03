Feature: kogen-shaping works with the model lineup in .kogen/models.yaml and stays green on coordinator main

  kogen-coordinator's K-MODELS-FILE (green candidate 2a4c69a, Build b04dd3bd; not on kogen-coordinator main yet)
  reads the team lineup from .kogen/models.yaml, puts that file in the approval baseline after .kogen/project.yaml
  and refuses a .kogen/project.yaml that still has models:. Its change outside tests is the fixture
  features/steps/testdata/k_models_file_shaping/coordinator.patch. The "pinned" coordinator is the
  kogen-coordinator version go.mod requires. The "K-MODELS-FILE" coordinator is a temporary copy of that module
  with the patch applied, used through a temporary go.work replace; nothing in the checkout changes. The harness
  features/steps/k_models_file_shaping_test.go.txt is overlaid into ./shape and built into each coordinator's
  shape test binary. It shapes in the shape package's fixture project with its scripted Claude Code shaper, whose
  team lineup is in .kogen/models.yaml for the K-MODELS-FILE coordinator and under models: in
  .kogen/project.yaml for the pinned one.

  Scenario: The shape tests the K-MODELS-FILE landing check failed pass against both coordinators
    When the shape tests run against the "pinned" coordinator:
      | test                                                                       |
      | TestClaudeLineupShapeSubmission                                            |
      | TestWorkersReceiveBoundedLongLineReadsAndReplayTheDeliveredDisplayExactly  |
      | TestBatchAnExistingEditableStepIsAValidDraftTarget                         |
      | TestBatchOwnershipAndBoundaryRefusalPrecedesCallsAndNeverAltersTheRawIndex |
      | TestShapeBatchInterruptCancelsEveryJob                                     |
    Then each of those shape tests passed exactly once
    When the shape tests run against the "K-MODELS-FILE" coordinator:
      | test                                                                       |
      | TestClaudeLineupShapeSubmission                                            |
      | TestWorkersReceiveBoundedLongLineReadsAndReplayTheDeliveredDisplayExactly  |
      | TestBatchAnExistingEditableStepIsAValidDraftTarget                         |
      | TestBatchOwnershipAndBoundaryRefusalPrecedesCallsAndNeverAltersTheRawIndex |
      | TestShapeBatchInterruptCancelsEveryJob                                     |
    Then each of those shape tests passed exactly once

  Scenario: The shape tests that depend on where the lineup lives pass against the K-MODELS-FILE coordinator
    When the shape tests run against the "K-MODELS-FILE" coordinator:
      | test                                                         |
      | TestClaudeLineupNativeCompatibility                          |
      | TestApproveCardOutput                                        |
      | TestRefineWritesTheCorrectionAndPrintsTheLines               |
      | TestShapeAnthropicInvalidProviderFailureReachesTheDiagnostic |
      | TestShapeBatchRunsWorkersAndReturnsTheirResults              |
      | TestContextBlocksCarryTheContractsAndEarlierRounds           |
      | TestShapeForceWireCompatibility                              |
    Then each of those shape tests passed exactly once

  Scenario: Shape takes the shape role from the lineup the coordinator reads and names .kogen/models.yaml in the protection contract only when the baseline has it
    When the models-file harness shapes against the "pinned" coordinator
    Then the models-file observation includes:
      """
      {"layout": "project-yaml",
       "shaper_sessions": 1,
       "model": "team-shape-model",
       "effort": "high",
       "automatic_baseline": "automatic baseline: every file in `.kogen/intents/is-even/`, every path listed under the Intent's `Acceptance:` (including step files), and `.kogen/project.yaml`.",
       "automatic_paths": "automatic paths currently present or listed: [\".kogen/project.yaml\"]"}
      """
    When the models-file harness shapes against the "K-MODELS-FILE" coordinator
    Then the models-file observation includes:
      """
      {"layout": "models-file",
       "shaper_sessions": 1,
       "model": "team-shape-model",
       "effort": "high",
       "automatic_baseline": "automatic baseline: every file in `.kogen/intents/is-even/`, every path listed under the Intent's `Acceptance:` (including step files), `.kogen/project.yaml` and `.kogen/models.yaml`.",
       "automatic_paths": "automatic paths currently present or listed: [\".kogen/project.yaml\", \".kogen/models.yaml\"]"}
      """
