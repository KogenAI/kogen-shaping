# Repeat an overlapped readiness timeout alone

Slug: shape-readiness-timeout-retried-after-overlap
Model: not applicable — the change is one bounded retry decision on a single readiness result, with no state that outlives a submission
Acceptance:
- .kogen/intents/shape-readiness-timeout-retried-after-overlap/acceptance.feature
- features/steps/shape-readiness-timeout-retried-after-overlap__steps.js

Shape starts the readiness observation of a submitted package together with the static checks and the model checks. When that observation ends in the acceptance-timeout refusal only because the host was shared with those checks, the submission must not be refused for it. This Intent makes Shape repeat the observation of the same package once, alone, after the concurrent checks have finished.

The refusal that triggers the repeat is the existing one: class `intent`, reason `not_ready`, detail `acceptance timed out`, first line starting `acceptance timed out`. The repeat happens only when at least one static check or model check of the same submission ran while the observation ran. A timeout where nothing overlapped, and every other readiness refusal (for example a failing acceptance assertion), is returned as it is today and is never repeated. The repeat happens at most once per package hash. Its result replaces the timeout. If the repeat also times out, the refusal is the existing refusal of the repeat run with one added last line: `Readiness was repeated alone after the concurrent checks ended; both runs timed out.` The time limit is unchanged and nothing is configurable. A repeat that does not refuse adds nothing to the refusal count of the retry budget.

Progress line, written when the repeat starts: `kogen: readiness timed out while other checks ran; repeating it alone` (Shape's progress writer appends its usual elapsed-time suffix). The events record the repeat as one event `readiness_repeated` with `package` (the package hash) and `first_wall_ms` (the wall time in milliseconds of the first, timed-out run, a number).

The change is in shape/readiness_overlap.go joinReadiness and its caller in shape/shape.go (`cheapGates`, which starts the observation and runs the other checks, marks the observation as overlapped when a static check or model check ran while it had not ended), plus test files of package shape; no other production file changes. It adds this test-only seam inside package `shape`, in those two files or in test files: a field `readinessObserver func(ctx context.Context, files []string) (certification, error)` on `shaping`; when non-nil it is called instead of `sh.certify` for every readiness observation of a package (the overlapped one started by `startReadiness`, the fallback of `joinReadiness`, and the repeat). `cheapGates` returns the started observation in its result's `readiness` field, which `joinReadiness` takes.

## Decisions
- The repeat is triggered only by the acceptance-timeout refusal with overlap; all other readiness refusals are returned unchanged and never repeated. (decision: D1, round 1) (evidence: shape/readiness_overlap.go; shape/shape.go) (source: request)
- At most one repeat per package hash, with no new configuration and no change to the 120-second limit. (decision: D2, round 1) (evidence: shape/readiness_overlap.go) (source: request)
- Overlap is marked by the caller in shape/shape.go `cheapGates`, which runs the other checks beside the observation; a static check whose stored result is reused without running does not count. (decision: D3, round 1) (evidence: shape/shape.go; shape/static.go) (assumption: that a reused static result is not an overlap, since it ran no process beside readiness) (source: request)
- The seam is the `readinessObserver` field on `shaping`, so scripted readiness and call counting need no other production file. (decision: D4, round 1) (evidence: shape/readiness_overlap.go; shape/static_overlap_test.go) (source: request)
- The second timeout keeps the existing lines and adds the one last line `Readiness was repeated alone after the concurrent checks ended; both runs timed out.` (decision: D5, round 1) (source: request)
- Acceptance drives the real `cheapGates` (with a real configured sleeping static check) and `joinReadiness` of the shape package through a Go overlay test, following the pattern of an existing intent's steps. (decision: D6, round 1) (evidence: shape/static_overlap_test.go; features/steps/k-shape-artifact-static-checks__steps.js) (source: request)

## Source comparison
- Repeat once, alone, after an overlapped acceptance timeout; the repeat result replaces the timeout: addressed by scenario 1 (acceptance.feature) and Decisions D1, D2.
- Second timeout refused with an added line, exactly two calls: addressed by scenario 2 and D5.
- Timeout without overlap refused at once with one call: addressed by scenario 3.
- Non-timeout refusal during overlap refused with its text and one call: addressed by scenario 4.
- Progress line `kogen: readiness timed out while other checks ran; repeating it alone` and the repeat event with the first wall time: addressed by scenario 1 (event `readiness_repeated`, `first_wall_ms`).
- No new configuration, no limit change, no retry-budget count for a successful repeat: addressed in the prose; scenario 1 checks that no refusal event is recorded.
- The change is in shape/readiness_overlap.go joinReadiness and its caller in shape/shape.go, with the seam in those files or shape test files: addressed in the prose and D3, D4.
- The request says scenarios 3 and 4 count calls through the new seam and fail on the unchanged tree because the seam does not exist: addressed; the harness uses `readinessObserver`.

## Tests
- Overlapped acceptance timeout then pass is ready, with progress line and repeat event: regression test in package shape with a scripted readiness and a scripted sleeping static check (regression)
- Two timeouts refuse with the added line after exactly two calls: test in package shape (regression)
- A timeout with no overlapping check is refused with one call: test in package shape (regression)
- A non-timeout readiness refusal during overlap is refused with one call: test in package shape (regression)

## Feasibility
- none
