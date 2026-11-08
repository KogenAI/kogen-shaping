Feature: Cumulative unpublished correction proposals
  Scenario: Refinement retains B and C when only A is corrected after lint rejection
    Then k-shape-refine-retry-keeps-corrections: existing test "TestRefineNarrowRetryPreservesAllEarlierProposalsUntilMergedReadiness" passes

  Scenario: Failed refinement leaves the published package untouched
    Then k-shape-refine-retry-keeps-corrections: existing test "TestRefineFailedAndCancelledApprovedCorrectionsPreservePackageRawApprovalAndIndex" passes

  Scenario: Shaping precheck and publication retain earlier omitted artifacts
    Then k-shape-refine-retry-keeps-corrections: existing test "TestShapeInvalidSubmitIsRetriedWithFeedbackAndKeepsEarlierProposals" passes
