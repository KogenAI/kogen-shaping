Feature: kogen-shaping follows kogen-coordinator's intent-not-found-names-next-command contract
  The producer's new main v0.0.0-20261107115359-b1f2988eb753 makes project.RequireIntentDir refuse a
  missing Intent folder as intent/not_found, exit 2, with the Detail
  "Intent does not exist: <slug> (looked in .kogen/intents/<slug>)" and a first extra line naming
  the next command: "kogen intent list shows existing Intents; kogen intent shape <slug> <request-file> creates one".
  This repository pins that main and its tests describe only that contract: they fail against the
  previous producer v0.0.0-20261106110407-d74daae36f56, with no fallback to the bare Detail.

  Scenario: go.mod and go.sum pin kogen-coordinator at the producer's new main and nothing else moves
    Then sh-consumes-k-coord-intent-not-found-names-next: go.mod requires "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261107115359-b1f2988eb753"
    And sh-consumes-k-coord-intent-not-found-names-next: go.mod requires "github.com/KogenAI/kogen-executor" at "v0.0.0-20261105010505-8d465842f9df"
    And sh-consumes-k-coord-intent-not-found-names-next: go.mod requires "github.com/KogenAI/kogen-loop" at "v0.0.0-20261105005409-31036b2c02e8"
    And sh-consumes-k-coord-intent-not-found-names-next: go.mod requires "github.com/KogenAI/kogen-verification" at "v0.0.0-20261105003813-ac4f6a9a4877"
    And sh-consumes-k-coord-intent-not-found-names-next: go.mod has no replace directive
    And sh-consumes-k-coord-intent-not-found-names-next: go.sum has the line "github.com/KogenAI/kogen-coordinator v0.0.0-20261107115359-b1f2988eb753 h1:PnCKKuwyRrlKAssxdYQFpGSfKpWJFv21GQpRDz2earM="
    And sh-consumes-k-coord-intent-not-found-names-next: go.sum has the line "github.com/KogenAI/kogen-coordinator v0.0.0-20261107115359-b1f2988eb753/go.mod h1:MMc15p3YwlY71dMxfCdX6QlR2jXJbSxL6YA9Y7971c8="
    And sh-consumes-k-coord-intent-not-found-names-next: go.sum lists "github.com/KogenAI/kogen-coordinator" only at "v0.0.0-20261107115359-b1f2988eb753"

  Scenario: The tests that failed pass against the producer's new main
    When sh-consumes-k-coord-intent-not-found-names-next: I run the Go tests "TestApproveMissingIntent TestRefineRefusalsBeforeTheModel" of package "./shape" against kogen-coordinator "v0.0.0-20261107115359-b1f2988eb753"
    Then sh-consumes-k-coord-intent-not-found-names-next: the run passed
    And sh-consumes-k-coord-intent-not-found-names-next: the test "TestApproveMissingIntent" passed
    And sh-consumes-k-coord-intent-not-found-names-next: the test "TestRefineRefusalsBeforeTheModel/unknown_Intent" passed

  Scenario: The consumer suite of kogen-coordinator passes against the producer's new main
    When sh-consumes-k-coord-intent-not-found-names-next: I run the Go tests "TestIntentExistsAndDir" of package "./consumes/kogen-coordinator" against kogen-coordinator "v0.0.0-20261107115359-b1f2988eb753"
    Then sh-consumes-k-coord-intent-not-found-names-next: the run passed
    And sh-consumes-k-coord-intent-not-found-names-next: the test "TestIntentExistsAndDir" passed

  Scenario: The tests accept no old behaviour, they fail against the previous producer
    When sh-consumes-k-coord-intent-not-found-names-next: I run the Go tests "TestApproveMissingIntent TestRefineRefusalsBeforeTheModel" of package "./shape" against kogen-coordinator "v0.0.0-20261106110407-d74daae36f56"
    Then sh-consumes-k-coord-intent-not-found-names-next: the test "TestApproveMissingIntent" failed
    And sh-consumes-k-coord-intent-not-found-names-next: the test "TestRefineRefusalsBeforeTheModel/unknown_Intent" failed

  Scenario: The consumer test accepts no bare Detail, it fails against the previous producer
    When sh-consumes-k-coord-intent-not-found-names-next: I run the Go tests "TestIntentExistsAndDir" of package "./consumes/kogen-coordinator" against kogen-coordinator "v0.0.0-20261106110407-d74daae36f56"
    Then sh-consumes-k-coord-intent-not-found-names-next: the test "TestIntentExistsAndDir" failed

  Scenario: With the module files as checked out the approve and refine refusals pass
    When sh-consumes-k-coord-intent-not-found-names-next: I run the Go tests "TestApproveMissingIntent TestRefineRefusalsBeforeTheModel" of package "./shape" with the module files as checked out
    Then sh-consumes-k-coord-intent-not-found-names-next: the run passed
    And sh-consumes-k-coord-intent-not-found-names-next: the test "TestApproveMissingIntent" passed
    And sh-consumes-k-coord-intent-not-found-names-next: go.mod requires "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261107115359-b1f2988eb753"

  Scenario: With the module files as checked out the consumer test of kogen-coordinator passes
    When sh-consumes-k-coord-intent-not-found-names-next: I run the Go tests "TestIntentExistsAndDir" of package "./consumes/kogen-coordinator" with the module files as checked out
    Then sh-consumes-k-coord-intent-not-found-names-next: the run passed
    And sh-consumes-k-coord-intent-not-found-names-next: the test "TestIntentExistsAndDir" passed
    And sh-consumes-k-coord-intent-not-found-names-next: go.mod requires "github.com/KogenAI/kogen-coordinator" at "v0.0.0-20261107115359-b1f2988eb753"
