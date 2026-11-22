# Add direct tests for RecheckDraftHash

Slug: recheck-draft-hash-direct-tests
Model: not applicable — the change only adds regression tests for one existing pure function; it adds no state or interacting rules
Acceptance:
- .kogen/intents/recheck-draft-hash-direct-tests/acceptance.feature
- features/steps/recheck-draft-hash-direct-tests__steps.js

Add `shape/recheck_test.go`, tests in package `shape` that call the public `RecheckDraftHash(root, slug string) (string, error)` directly. The tests build a valid fixture in `t.TempDir()` (a `.kogen/project.yaml` and a draft directory `.kogen/intents/<slug>/` holding `intent.md` and `acceptance.feature`) and check: two calls on the unchanged valid draft return the same nonempty hash; changing the draft's `acceptance.feature` changes the hash; an invalid draft and a missing draft each return an error. Here a missing draft is a draft requested from a root that has no `.kogen/project.yaml`, and an invalid draft is one in a root whose `.kogen/project.yaml` is invalid. Only `shape/recheck_test.go` changes; production code, other Recheck behavior and unrelated package hashing are out of scope.

## Decisions
- An invalid or missing draft returns an error, and the tests assert exactly that: the draft slug requested from a root with no `.kogen/project.yaml` (missing) and from a root whose `.kogen/project.yaml` is invalid (invalid) each return an error. These are the only draft inputs on which the current `RecheckDraftHash` returns an error; a slug whose draft directory is absent inside a valid project returns a hash and no error today, and changing that is production code, which the request puts out of scope, so a test asserting an error there could not pass.  (evidence: shape/recheck.go) (assumption: "missing draft" is tested as a draft in a root without a project configuration, because the function errors for no other missing input) (source: request)
- Acceptance judges the new tests by running them against the real `RecheckDraftHash` and against five scratch copies of `shape/recheck.go` (via `go test -overlay`, no checkout file is written) whose function returns an empty hash, a constant hash, a per-call-unstable hash, a hash that ignores `acceptance.feature`, or swallows the project-configuration error; each copy must make a test in `shape/recheck_test.go` fail. Test names are read from the file, so none is invented. (evidence: shape/recheck.go; shape/precheck_test.go) (source: request)
- No setting, flag or environment variable is added. (source: request)

## Source comparison
- Request acceptance 1 (same nonempty hash on two calls of a valid fixture): addressed by the stable-and-nonempty test and the empty and unstable copies. (source: request)
- Request acceptance 2 (hash changes when `acceptance.feature` changes; an invalid or missing draft returns an error): addressed by the change test, the ignores-`acceptance.feature` and constant copies, and the error test with the swallowed-error copy; "missing draft" is bounded as the first Decision says. (source: request)
- Out of scope and untouched: production code, other recheck behavior, unrelated package hashing. (source: request)

## Tests
- A valid unchanged draft yields the same nonempty hash on two calls: TestRecheckDraftHashIsStableAndNonempty in shape/recheck_test.go (unit)
- Changing the draft's acceptance.feature changes the hash: TestRecheckDraftHashChangesWithDraftFile in shape/recheck_test.go (unit)
- An invalid or missing draft returns an error: TestRecheckDraftHashErrorsWithoutValidProject in shape/recheck_test.go (unit)

## Feasibility
- none
