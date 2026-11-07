# Follow the Intent not-found refusal contract

Slug: sh-consumes-k-coord-intent-not-found-names-next
Model: not applicable — a dependency pin and the expected text of three test assertions change; there is no state or interacting rules
Acceptance:
- .kogen/intents/sh-consumes-k-coord-intent-not-found-names-next/acceptance.feature
- features/steps/sh-consumes-k-coord-intent-not-found-names-next__steps.js

This is the consumer row of kogen-coordinator's intended contract change k-coord-intent-not-found-names-next-command. That change landed on purpose, with no compatibility layer, and this repository follows the new contract; no fallback is added and no test or code accepts the old behaviour.

The contract: `project.RequireIntentDir(root, slug)`, for an Intent folder that does not exist, still fails as `intent/not_found` with exit 2, but its Detail is now `Intent does not exist: <slug> (looked in .kogen/intents/<slug>)` (it was `Intent does not exist`) and its first entry in `Lines` is `kogen intent list shows existing Intents; kogen intent shape <slug> <request-file> creates one`. A second line `did you mean: <a>, <b>?` follows when other Intent folders exist; it depends on the neighbours and no test here pins it.

Two tests of this repository failed against the producer's new main:

```
TestApproveMissingIntent	approve_test.go:422: error = intent/not_found: Intent does not exist: unknown (looked in .kogenunknown) (exit 2), want intent/not_found: Intent does not exist (exit 2)
TestIntentExistsAndDir	project_intent_test.go:140: project.RequireIntentDir of a missing folder = intent/not_found: Intent does not exist: absent (looked in .kogenabsent), want intent/not_found exit 2 "Intent does not exist
```

The report text shows the folder as `.kogenunknown` and `.kogenabsent`; run against the producer's new main the real text is `.kogen/intents/unknown` and `.kogen/intents/absent`, which is what the tests expect.

Required edits:
- `go.mod`: require `github.com/KogenAI/kogen-coordinator` at `v0.0.0-20261107115359-b1f2988eb753`, the producer's main resolved through the module mirror (`go get github.com/KogenAI/kogen-coordinator@main`). `go.sum` then holds exactly the module and `go.mod` hash lines of that version (`h1:PnCKKuwyRrlKAssxdYQFpGSfKpWJFv21GQpRDz2earM=` and `/go.mod h1:MMc15p3YwlY71dMxfCdX6QlR2jXJbSxL6YA9Y7971c8=`) for the coordinator and none for the previous version. The other three requirements and the Go version are unchanged, with no `replace`.
- `shape/approve_test.go`, `TestApproveMissingIntent`: the error is `intent/not_found`, exit 2, with Detail exactly `Intent does not exist: unknown (looked in .kogen/intents/unknown)` and `Lines[0]` exactly `kogen intent list shows existing Intents; kogen intent shape unknown <request-file> creates one`. The exit value is still 2.
- `shape/refine_test.go`, `TestRefineRefusalsBeforeTheModel`, subtest `unknown Intent` (slug `missing`): its Detail check, which accepted any text containing `Intent does not exist`, requires the full Detail `Intent does not exist: missing (looked in .kogen/intents/missing)` and `Lines[0]` `kogen intent list shows existing Intents; kogen intent shape missing <request-file> creates one`.
- `consumes/kogen-coordinator/project_intent_test.go`, `TestIntentExistsAndDir`: the missing-folder case (`absent`) requires class `intent`, reason `not_found`, exit 2, Detail exactly `Intent does not exist: absent (looked in .kogen/intents/absent)` and `Lines[0]` exactly `kogen intent list shows existing Intents; kogen intent shape absent <request-file> creates one`. The failure message states that expectation. The cases for an existing folder and for a regular file in place of the folder are unchanged.
- `consumes/kogen-coordinator/README.md`: the pinned version it states is `v0.0.0-20261107115359-b1f2988eb753`.

Nothing else changes: no production code, no other test, no other fixture. Every other test stays green.

## Decisions
- The pin moves to the producer's main as the module mirror resolves it, v0.0.0-20261107115359-b1f2988eb753, in go.mod and go.sum together. (decision: D1, round 1) (constraint: the other three requirements and the go.sum lines for them stay as they are) (evidence: go.mod; consumes/kogen-coordinator/README.md) (assumption: The mirror's main may move before the Build runs; the version was resolved by an external `go list -m -json github.com/KogenAI/kogen-coordinator@main` and the source read from the module cache.) (source: request)
- Tests expect the new Detail naming the slug and the folder .kogen/intents/<slug>, and the next-command line as Lines[0]; the optional did-you-mean line is not pinned. (decision: D2, round 1) (constraint: no assertion accepts the bare Detail) (evidence: shape/approve_test.go; shape/refine_test.go; consumes/kogen-coordinator/project_intent_test.go) (assumption: Producer tests at that version were read from the module cache, not the repository.) (source: request)
- The Refine unknown Intent substring check is tightened so it no longer accepts the old Detail. (decision: D3, round 1) (evidence: shape/refine_test.go; shape/helper_test.go) (source: request)
- Production code of this repository is unchanged; it returns the producer's error as is. (decision: D4, round 1) (evidence: shape/approve.go; shape/refine.go) (source: request)
- Documentation follows the pin: only the version the consumes README states changes. (decision: D5, round 1) (evidence: consumes/kogen-coordinator/README.md) (source: request)

## Source comparison
- Addressed: "move this repository's go.mod pin of the producer to that main (go get against the mirror)". Required edit 1; scenario "go.mod and go.sum pin kogen-coordinator at the producer's new main and nothing else moves".
- Addressed: the two reported failures, `TestApproveMissingIntent` and `TestIntentExistsAndDir`. Required edits 2 and 4; scenarios "The tests that failed pass against the producer's new main", "The consumer suite of kogen-coordinator passes against the producer's new main" and the two scenarios that run the module files as checked out.
- Addressed: "No test or code accepts the old behaviour, and no fallback is added". The Refine substring check is tightened (required edit 3); the scenarios "The tests accept no old behaviour, they fail against the previous producer" and "The consumer test accepts no bare Detail, it fails against the previous producer" run the three tests against `v0.0.0-20261106110407-d74daae36f56` and require them to fail on an assertion.
- Addressed: "Nothing else changes. Keep every other test green." The configured go-test check runs every test; the other three requirements are pinned in the first scenario.
- Addressed with a difference: the report shows the folder as `.kogenunknown`; the real text under the new main is `.kogen/intents/unknown` (see the contract above).
- Omitted: the optional `did you mean` line is not pinned, it depends on which Intent folders exist.

## Tests
- A missing Intent in Approve is refused with the new Detail and next-command line: update TestApproveMissingIntent (unit)
- A missing Intent in Refine is refused with the new Detail and next-command line: update the unknown Intent subtest of TestRefineRefusalsBeforeTheModel (unit)
- project.RequireIntentDir of a missing folder gives the new Detail and next-command line: update TestIntentExistsAndDir (integration)
- The pin is the producer's new main and the tests fail against the previous producer: acceptance scenarios "go.mod and go.sum pin kogen-coordinator at the producer's new main and nothing else moves" and "The tests accept no old behaviour, they fail against the previous producer" (scenario)
- Every other existing test stays green: the configured go-test check (existing)

## Feasibility
- none
