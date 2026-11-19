# Exclude timed-out static checks from shaping time

Slug: shape-time-excludes-timed-out-static-checks
Model: not applicable — one duration subtraction and one event field, with no state machine or interacting rules
Acceptance:
- .kogen/intents/shape-time-excludes-timed-out-static-checks/acceptance.feature
- features/steps/shape-time-excludes-timed-out-static-checks__steps.js

A static check that ended `unavailable` because it timed out (`sys.Result.TimedOut`, reason `timed out after 120 s`) tells the shaper nothing about the shaped files, yet its wall time is charged to the 15-minute shaping budget. After this change that wall time is excluded from the charged time, in the same way as cold runtime installation.

- `chargedTime` (shape/shape.go) is the time since the start minus the runtime install time minus the uncharged static time. The budget line (`Time: N elapsed, M remaining`), the budget refusal and the timed attempts all use `chargedTime`, so they use the charged figure.
- The uncharged static time of one timed-out check is its wall time, capped at `verify.StepTimeout` (120 s), so a check excludes at most 120 s per round. A check that passed, failed or ended unavailable for another reason is charged as now, and so are the shaper's own model time, readiness and contract checks.
- The `shape_time` event keeps `static_ms` as the full static wall time and gains `static_uncharged_ms`, the excluded part; the field is present, with 0, when nothing was excluded.
- The budget refusal's lines name the uncharged time next to the install time: where `, install_ms=N` is appended now (the budget line and each attempt line), `, static_uncharged_ms=M` is appended when M is above 0, after `install_ms`.
- `shapingBudget` stays 15 minutes. No gate decision changes. No flag, setting or environment variable is added. If the timed-out-check memo (a timed-out check not run again in a session) is present, its skipped rounds count as zero time and this change still applies to the first run.

The Build adds model-free Go tests in `shape/budget_static_timeout_test.go`, with the existing fake clock (`budgetClock`) and shape fixtures (`stmNew`), so that `go test ./shape -run TestChargedTimeStaticTimeout` runs them: TestChargedTimeStaticTimeoutExcluded (a check timing out after 120 s leaves chargedTime and the budget line's remaining time unchanged; unit), TestChargedTimeFailedStaticCheckCharged (a check failing with exit 1 after 120 s is charged in full; unit), TestShapeTimeReportsUncharged (static_ms full, static_uncharged_ms the excluded part, the budget refusal names it beside install_ms; integration) and TestChargedTimeStaticTimeoutBounded (two timed-out checks exclude at most 120 s each; a wall time above the step timeout is excluded only up to 120 s; unit). Release evidence: those tests do not exist on main, so they fail on main and pass after the change; acceptance here observes the same behaviour through a harness whose test exists independently of the Build.

## Decisions
- The excluded time is measured on the shaping clock (the clock `ShapeInput.Now` sets), because the existing `c.WallMS` of a static check uses the real clock and cannot be driven by the request's fake-clock tests (decision: D1, round 1) (evidence: shape/static.go; shape/shape.go; shape/evidence.go) (source: request)
- `static_uncharged_ms` is always present in `shape_time`, 0 when nothing was excluded, and the refusal suffix appears only when it is above 0, like `install_ms` (decision: D2, round 1)  (evidence: shape/budget.go; shape/evidence.go) (assumption: the user must confirm the zero-valued field and the omitted suffix) (source: request)

## Source comparison
- Expected 1 (subtract in chargedTime, budget line and refusal use it): addressed by the chargedTime bullet and acceptance scenario 1. Source: request.
- Expected 2 (static_ms full, static_uncharged_ms new, refusal names it beside install_ms): addressed by the event and refusal bullets and scenarios 1 and 3. Source: request.
- Expected 3 (pass, fail and other unavailable are charged): scenario 2. Source: request.
- Expected 4 (120 s cap per check): scenario 3 (a 150 s wall time excluded as 120 s, two checks 240 s, the passing check charged). Source: request.
- The tests named in the request live in shape/budget_static_timeout_test.go and are listed above; the acceptance harness does not run them by name because they do not exist before the Build. Source: request.

## Tests
- A timed-out check's wall time is not charged and is reported as static_uncharged_ms with static_ms the full time: shape_time observation through the real Shape entry point (scenario)
- A failing check is charged in full: shape_time observation (scenario)
- At most 120 s is excluded per timed-out check; a passing check is charged: shape_time observation (scenario)
- chargedTime, budget line and refusal lines: TestChargedTimeStaticTimeoutExcluded, TestChargedTimeFailedStaticCheckCharged, TestChargedTimeStaticTimeoutBounded, TestShapeTimeReportsUncharged in shape/budget_static_timeout_test.go (unit)

## Feasibility
- none
