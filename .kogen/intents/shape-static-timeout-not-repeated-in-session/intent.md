# Skip timed-out static checks later in a session

Slug: shape-static-timeout-not-repeated-in-session
Model: not applicable — the change remembers one fact per check inside a single process run, with no durable state
Acceptance:
- .kogen/intents/shape-static-timeout-not-repeated-in-session/acceptance.feature
- features/steps/shape-static-timeout-not-repeated-in-session__steps.js
- features/steps/shape-static-timeout-not-repeated-in-session__driver/harness_test.go.txt

Within one Shape session, a configured static check that ended unavailable because it timed out is not run again in later rounds (submissions) while its command line and the lockfiles and manifests of the candidate tree are unchanged. The check stays unavailable, never passed, so the static outcome stays unknown and the wording that the shaped files are not established by it stays true. A check that ended unavailable for another reason (program not found, killed by a signal, exit 126 or 127, no output), and every passed or failed check, runs in every round as before; the reuse of the base run of a failing check is unchanged. The memory lives only in the running session: it is not written to any file, and a new or resumed session starts with none. The 120 s step timeout, the list of selected checks, the checks that run a test suite and are never run, the refusal rules for failed checks and the Approve-time gate are unchanged; there is no new flag, setting or environment variable.

A skipped round reports the check in one progress line of the form "kogen: static check NAME unavailable: timed out after 120 s in round 1; not run again in this session", where NAME is the check's name and the round number is that of the round whose run timed out (1 in this form). The check's entry in the static_checks event and in the static-checks record has status unavailable, wall_ms 0, no exit and no output, and the reason "timed out after 120 s in round 1; not run again in this session" (with the actual round number), so every selected check appears in every round.

## Decisions
- The memory is a field of the Shape session that every round's static gate runner receives; it is never persisted and no file, flag, setting or environment variable is added. (decision: D1, round 1) (constraint: nothing is written under the checkout or .kogen/local for it) (evidence: shape/static.go; shape/static_test.go) (source: request)
- Only a result with TimedOut set is remembered, keyed by check name, command line and a digest of the contents of the lockfiles and manifests in the candidate tree that verify's build cache keys on, so a changed mix.lock runs the check again; the digest is of file contents and does not depend on the toolchain version lookup of the cache key, which can be empty. (decision: D2, round 1) (constraint: other unavailable results, passed results and failed results are never remembered) (evidence: shape/static.go; shape/helper_test.go) (assumption: the Build derives the digest from the lockfile and manifest names the build cache already keys on) (source: request)
- The round named in the reason and the progress line is the Shape submission number of the run that timed out; a later skip repeats that number. (decision: D3, round 1) (evidence: shape/static.go) (source: request)
- The skipped check is recorded unavailable with wall_ms 0 and the Outcome stays unknown through the existing unavailable handling. Approve's own gate run starts with no memory, so it is unchanged. (decision: D4, round 1) (evidence: shape/static.go) (source: request)
- Acceptance drives the real Shape entry point with a scripted shaper and fake check commands through the existing fakeStaticChecks seam, using one overlay-only test file that never exists in the repository; the Build separately adds shape/static_timeout_memory_test.go with TestStaticTimeoutMemoryNotRepeated, TestStaticTimeoutMemoryOnlyForTimeouts, TestStaticTimeoutMemoryKeepsFailedChecks and TestStaticTimeoutMemoryResetsOnLockChangeAndNewSession, run by go test ./shape -run TestStaticTimeoutMemory. (decision: D5, round 1) (evidence: shape/clock_bound_test.go; shape/helper_test.go; features/steps/shape-static-checks-record-per-check-time__steps.js) (source: request)

## Source comparison
- Expected 1 (a timed-out check is not run again while command and lockfiles are unchanged, one progress line "kogen: static check NAME unavailable: timed out after 120 s in round 1; not run again in this session", stays unavailable): addressed by scenarios 1 and 4.
- Expected 2 (other unavailable reasons still run every round): addressed by scenario 2.
- Expected 3 (passed or failed checks run every round, base reuse unchanged): addressed by scenario 3.
- Expected 4 (event and record carry status unavailable, wall_ms 0 and a reason that says it was not run again): addressed by scenario 1.
- Expected 5 (memory for one session only, nothing written outside the session): addressed by the new-session scenario 5; a changed lockfile is scenario 4.
- The release-acceptance request to name the test that fails on main and passes after the change: the scenarios of this Intent fail on main at the Then steps asserting 1 start instead of 2 and wall_ms 0.
- Collection scaling: not applicable, the checks are a fixed configured list and no operation over a growing collection of records is added.

## Tests
- A check that timed out in round 1 is not started in round 2 and is recorded unavailable with wall_ms 0: TestStaticTimeoutMemoryNotRepeated (integration)
- Only timeouts are remembered; a check not found in round 1 runs again in round 2: TestStaticTimeoutMemoryOnlyForTimeouts (integration)
- A failed check runs in both rounds while a timed-out check beside it is skipped in round 2: TestStaticTimeoutMemoryKeepsFailedChecks (integration)
- A changed mix.lock runs the timed-out check again and a new session starts with no memory: TestStaticTimeoutMemoryResetsOnLockChangeAndNewSession (integration)
- The five behaviors above through the real Shape entry point: the scenarios of acceptance.feature (scenario)

## Feasibility
- none
