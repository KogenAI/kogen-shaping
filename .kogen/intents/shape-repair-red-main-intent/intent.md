# Let Shape repair named failing checks on red main

Slug: shape-repair-red-main-intent
Model: not applicable — decision rules over observed check results, no state machine
Acceptance:
- .kogen/intents/shape-repair-red-main-intent/acceptance.feature
- features/steps/shape-repair-red-main-intent__steps.js
- features/steps/shape-repair-red-main-intent__harness.js

When a request declares that it repairs named failing checks on main, Shape runs those named checks on the starting revision and records their failures as the repair target (red before, green after) instead of refusing because main is red. Every other required test stays a kept check that must pass on the unchanged tree and after the change. A named check that is already green on the starting revision is refused with a message saying that the named check is not failing. Kogen's own check time (own check seconds) and any acceptance timeout (a step that times out at 120 s) are reported on their own lines, apart from the product's defect, so a timeout or slow Kogen check is not shown as a failed test. Requests without a repair declaration behave as today, including refusal of a red main. Out of scope: Build, Refine and the 15 minute shaping budget. Changes stay in shape/, with model-free tests in shape/ and fixtures under shape/testdata.

## Decisions
- A request declares a repair when its text says that it repairs the named failing check(s) on main; the fixture states this as a sentence naming the check in double quotes. No setting or flag is added. (assumption: the request names no declaration form; the Shaper settles recognition as a sentence naming the check and says it repairs it) (source: request)
- Named checks are run on the starting revision; each one that fails there is recorded as the target (red before, green after) in one progress line naming the check; a target never counts as a kept check and never causes the red-on-unchanged-tree refusal. (evidence: shape/kept_baseline.go) (source: request)
- Every required test not named in the declaration remains a kept check: if it is red on the unchanged tree it is refused exactly as today. (evidence: shape/kept_baseline.go) (source: request)
- A named check that passes on the starting revision is refused with a message containing `not failing` and the check's name. (source: request)
- The `shape_time` event reports Kogen's own check time under the key `own check seconds` (a number) and the checks that ended in a timeout under the key `acceptance timeout` (an array of check names); a check that times out at 120 s is reported there and is not added to any failed-test or product-defect list. (evidence: shape/budget_static_timeout_test.go) (assumption: the request names the lines own check seconds and acceptance timeout; carrying them as shape_time keys is the Shaper's choice) (source: request)
- A request without a repair declaration on a red main is refused exactly as today. (evidence: shape/kept_baseline.go) (source: request)

## Source comparison
- Addressed: request scenarios 1, 2, 3 and 5 by feature scenarios on a synthetic fixture repo, run through the real Shape entry; scenario 4 by the timing scenario with a fake clock and a fake slow check that times out at 120 s, reporting `own check seconds` and `acceptance timeout`. Test names in the request are the Build's tests to add; acceptance observes behaviour and does not run them.
- Addressed: Release evidence, the unit and integration tests named in the request are added by the Build in shape/.
- Omitted: Build, Refine and the shaping budget, as the request scopes out.

## Tests
- Repair target on red main is recorded, not refused: TestShapeRepairRedMainTarget (integration)
- Other checks stay kept and a broken one is refused: TestShapeRepairKeepsOtherChecks (integration)
- Named check green on main is refused: TestShapeRepairNamedCheckGreen (unit)
- Own check time and timeout reported apart from product defect: TestShapeReportsOwnTimeAndTimeoutApart (integration)
- Red main without declaration still refused: TestShapeRedMainWithoutRepairRefused (unit)

## Feasibility
- none
