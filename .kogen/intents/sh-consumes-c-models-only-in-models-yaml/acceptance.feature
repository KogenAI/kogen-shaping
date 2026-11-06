Feature: kogen-shaping follows kogen-coordinator's c-models-only-in-models-yaml contract
  kogen-coordinator d74daae (the producer's new main) reads model roles only from
  .kogen/models.yaml and .kogen/local/models.yaml and refuses a models: key in
  .kogen/project.yaml. This repository pins that main and its tests and fixtures write the lineup
  only in .kogen/models.yaml, with no fallback to the old place.

  Scenario: go.mod and go.sum pin kogen-coordinator at the producer's new main
    Then sh-consumes-c-models-only-in-models-yaml: go.mod requires "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261106110407-d74daae36f56"
    And sh-consumes-c-models-only-in-models-yaml: go.sum has the line "github.com/KogenAI/kogen-coordinator v0.0.0-20261106110407-d74daae36f56 h1:8VmqsU9EArCYwqOTw8e9uuS54nRSlUHSUaDl8W+lYqs="
    And sh-consumes-c-models-only-in-models-yaml: go.sum has the line "github.com/KogenAI/kogen-coordinator v0.0.0-20261106110407-d74daae36f56/go.mod h1:MMc15p3YwlY71dMxfCdX6QlR2jXJbSxL6YA9Y7971c8="
    And sh-consumes-c-models-only-in-models-yaml: go.sum has no line for "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261105022408-e7bc7ce9c169"

  Scenario: The kogen-coordinator consumer suite passes against the new main
    When sh-consumes-c-models-only-in-models-yaml: every Go test of package "./consumes/kogen-coordinator" is run
    Then sh-consumes-c-models-only-in-models-yaml: the run passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestBaselineAndHash" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestBindSource" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestCaptureOrigin" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestCaptureSource" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestCheckIntent" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestCheckIntentStops" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestCheckoutAndPackageFacts" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestConfigModels" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestIntentExistsAndDir" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestLoadConfig" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestLoadConfigStops" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestLoadIntent" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestRefineSource" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestRefreshSource" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestResolveApproval" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestRestoreSource" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestSourceComparison" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestSourceMessageAndCard" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestValidateSource" passed
    And sh-consumes-c-models-only-in-models-yaml: go.mod requires "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261106110407-d74daae36f56"

  Scenario: The shape tests that failed against the new main pass against it
    When sh-consumes-c-models-only-in-models-yaml: the Go tests "TestClaudeLineupNativeCompatibility TestClaudeLineupShapeCheckoutGuard TestClaudeLineupShapeSubmission TestClaudeLineupUnsupportedCapabilities TestExecutorRoundTiming TestExternalShapeDecisionCheck TestExternalShapeLiteralCheck TestExternalShapeReceivesTheRunnerObserverRefusalAsFeedback TestExternalShapeReceivesTheUnwrittenTestsRefusalAsFeedback TestRefineAnthropicInvalidProviderFailureReachesTheResult TestRefineGivesTheExternalRefinerShapesRefusalText TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady TestShapeAnthropicInvalidProviderFailureReachesTheDiagnostic TestShapeDecisionRecordExternal TestShapeForceWireCompatibility TestShapeTimeWithQuestions TestShapeDecisionEvidenceExternal TestExternalShapeQuestionCheck TestExternalShapeQuestionCheckCitedEvidence TestShapeExternalPrecheckIsNotCounted" of package "./shape" are run
    Then sh-consumes-c-models-only-in-models-yaml: the run passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupNativeCompatibility" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeSubmission" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupUnsupportedCapabilities" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExternalShapeDecisionCheck" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExternalShapeLiteralCheck" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExternalShapeReceivesTheRunnerObserverRefusalAsFeedback" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExternalShapeReceivesTheUnwrittenTestsRefusalAsFeedback" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestRefineAnthropicInvalidProviderFailureReachesTheResult" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestRefineGivesTheExternalRefinerShapesRefusalText" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestScopeRefusalIsRecordedBeforeTheDecisionCheckAndEndsNotReady" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestShapeAnthropicInvalidProviderFailureReachesTheDiagnostic" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestShapeDecisionRecordExternal" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestShapeForceWireCompatibility" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestShapeTimeWithQuestions" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestShapeDecisionEvidenceExternal" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExternalShapeQuestionCheck" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExternalShapeQuestionCheckCitedEvidence" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestShapeExternalPrecheckIsNotCounted" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/a_failed_removal_names_the_files_and_publishes_nothing" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/a_file_the_session_untracked_is_not_removed" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/created_files_are_reported_once_in_the_evidence" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/created_untracked_files_are_removed,_pre-existing_ones_stay,_a_stat_refresh_is_equal" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/kogen's_own_run_evidence_is_not_a_checkout_edit" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/refine_mutation_preserves_the_package,_approval_and_the_detected_edit" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/shape_change_a_mode" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/shape_delete_a_file" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/shape_edit_a_tracked_file" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/shape_edit_a_tracked_ignored_file" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/shape_move_HEAD" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeCheckoutGuard/shape_stage_a_change" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeSubmission/a_model-supplied_provenance_line_is_not_a_Kogen_documentation_record" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeSubmission/questions_keep_the_numbered_CLI_and_the_next_invocation_is_a_fresh_session" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupShapeSubmission/readiness_feedback_and_exact_red_acknowledgment_stay_with_one_session" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupUnsupportedCapabilities/a_descriptor_without_the_shape_role_is_refused" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestClaudeLineupUnsupportedCapabilities/limitations_are_reported_and_research_is_not_a_Kogen_record" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/a_fractional_value" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/a_missing_file" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/a_null_value" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/a_string_value" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/an_absent_value" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/an_earlier_line_when_the_last_is_unusable" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/no_result_line" passed
    And sh-consumes-c-models-only-in-models-yaml: the test "TestExecutorRoundTiming/the_last_result_line" passed
    And sh-consumes-c-models-only-in-models-yaml: go.mod requires "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261106110407-d74daae36f56"

  Scenario: No Go file keeps the detection of the old lineup place
    Then sh-consumes-c-models-only-in-models-yaml: no Go file of the module mentions "models_moved"
