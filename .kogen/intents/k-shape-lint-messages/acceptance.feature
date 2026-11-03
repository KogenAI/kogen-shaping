Feature: Shaping's tests expect the coordinator's current lint messages

  kogen-coordinator main (8d24cc7, "Report every Intent lint problem in one refusal") replaced the single
  lint line `decisions: a decision has no source` with one line per unsourced Decision:
  `decisions: <the Decision bullet> must end with `(source: request)` or `(source: answer <n>)``.
  The "pinned" coordinator is the version go.mod requires. The "main" coordinator is
  v0.0.0-20261103113131-ffca7072fe15 (ffca707, coordinator main's code since 8d24cc7; main's later commit
  e26a4d7 changes documentation only), used through a temporary go.work replace. The "mutated" coordinator
  is a temporary copy of the main one whose unsourced-Decision line reads
  `decisions: <the Decision bullet> is unsourced`, neither the pinned nor the main message, so a test that
  still checks the lint correction fails against it. Nothing in the checkout is written.

  Scenario: The lint-correction shape tests pass against the pinned coordinator and coordinator main and still check the unsourced-Decision line
    Given go.mod requires kogen-coordinator "v0.0.0-20261102193107-c4a49a4fe84f"
    When the lint-message shape tests run against the "pinned" coordinator:
      | test                                                                                               |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing           |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing/unsafe    |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing/duplicate |
      | TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals                              |
      | TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness                            |
      | TestRefineLaterProposalsReplaceAPathAndOriginalBytesRevertIt                                     |
    Then each of those lint-message shape tests passed exactly once
    When the lint-message shape tests run against the "mutated" coordinator:
      | test                                                                                               |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing           |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing/unsafe    |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing/duplicate |
      | TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals                              |
      | TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness                            |
      | TestRefineLaterProposalsReplaceAPathAndOriginalBytesRevertIt                                     |
    Then each of those lint-message shape tests failed exactly once
    When the lint-message shape tests run against the "main" coordinator:
      | test                                                                                               |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing           |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing/unsafe    |
      | TestShapeLintCorrectionsKeepEarlierArtifactsAndMalformedMixedPayloadsContributeNothing/duplicate |
      | TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals                              |
      | TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness                            |
      | TestRefineLaterProposalsReplaceAPathAndOriginalBytesRevertIt                                     |
    Then each of those lint-message shape tests passed exactly once
