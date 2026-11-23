# Pin ExitCode for wrapped errors in tests

Slug: exit-code-wrapped-errors-tested
Model: not applicable — the change only adds tests of a stateless function
Acceptance:
- .kogen/intents/exit-code-wrapped-errors-tested/acceptance.feature
- features/steps/exit-code-wrapped-errors-tested__steps.js

This change adds focused Go tests to `shape/context_test.go` that pin the behavior of `ExitCode` in `shape/common.go` for wrapped errors. `ExitCode` is not meant to change: its current behavior is the behavior to pin. Three rules are pinned. A `*sys.Error` found through wrapping returns its embedded exit code. A `*sys.Interrupted` found through wrapping returns its signal-specific exit code. An unrelated wrapped error returns 70. The existing assertions on an unknown error and an unwrapped `sys.Error` stay. Out of scope: command exit-code policy, and any change to `sys.Error` or `sys.Interrupted`.

## Decisions
- The tests live in `shape/context_test.go` and call `ExitCode` with wrapped errors. Acceptance finds them at run time as the test functions of that file whose body calls `ExitCode(`, so their names are free. (evidence: shape/common.go; shape/context_test.go) (source: request)
- A wrapped `*sys.Error` returns its embedded exit code. The test uses an exit code other than 70. (evidence: shape/common.go) (source: request)
- A wrapped `*sys.Interrupted` returns its signal-specific exit code. The tests cover at least two different signals, so one fixed code for every interruption fails them. (evidence: shape/common.go) (source: request)
- A wrapped unrelated error (for example `os.ErrNotExist`) still returns 70. (evidence: shape/common.go; shape/context_test.go) (source: request)
- `shape/common.go` keeps its `ExitCode` unchanged; the change is test-only. Acceptance proves each test is meaningful by running them in a scratch copy of the `shape` package against the unchanged `ExitCode` (must pass) and against wrong versions (a direct type assertion for a Kogen error, no interruption handling, a direct type assertion for an interruption, one fixed code for every interruption, any wrapped error treated as a Kogen error), each of which at least one test must reject. (evidence: shape/common.go) (source: request)

## Source comparison
- Request item 1 (wrapped sys.Error returns its code; direct type assertion is wrong) — addressed by the scenario "A direct type assertion for a Kogen error is rejected".
- Request item 2 (wrapped sys.Interrupted returns its signal code; omitted handling or no unwrapping is wrong) — addressed by the scenarios on missing interruption handling, a direct type assertion for an interruption and a fixed code for every interruption.
- Request item 3 (unrelated wrapped error returns 70; treating any wrapped error as a Kogen error is wrong) — addressed by the scenario "Treating any wrapped error as a Kogen error is rejected".
- Out of scope and omitted: command exit-code policy, changes to sys.Error or sys.Interrupted.

## Tests
- A wrapped sys.Error returns its embedded exit code: a test in shape/context_test.go (unit)
- A wrapped sys.Interrupted returns its signal-specific exit code, for at least two signals: a test in shape/context_test.go (unit)
- An unrelated wrapped error returns 70: a test in shape/context_test.go (unit)
- The tests pass on the real ExitCode and fail on each wrong ExitCode: acceptance.feature scenarios run them in a scratch copy of the shape package (scenario)

## Feasibility
- none
