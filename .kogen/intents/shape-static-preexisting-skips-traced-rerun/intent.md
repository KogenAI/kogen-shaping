# Skip the traced rerun of pre-existing static

Slug: shape-static-preexisting-skips-traced-rerun
Model: not applicable — the change reorders two runs inside one function; there is no state to model
Acceptance:
- .kogen/intents/shape-static-preexisting-skips-traced-rerun/acceptance.feature
- features/steps/shape-static-preexisting-skips-traced-rerun__steps.js
- features/steps/shape-static-preexisting-skips-traced-rerun__driver/harness_test.go.txt

When a configured static check fails in the shaped copy, the gate in `staticGateRunner.run` (shape/static.go) ran the check a second time under `set -x` (the traced rerun, `"set -x\n"+c.Run`) right after the failing first run, before it looked up or ran the base result (the check without the shaped files). For a check that also fails without the shaped files, that rerun cannot change the verdict, yet on a large test suite it costs a full extra run in every Shape round.

The traced rerun (`"set -x\n"+c.Run`) now runs only after the base result is known, and only when the failure can still be attributed to the shaped files. A failing check is pre-existing when the base result failed (and was not unavailable) and the first run's output names no path of the shaped files that the base output does not name; such a check does not run the traced rerun. Its recorded paths are the paths of files other than the shaped files that the first run's output names, with their `:<line>[:<column>]` taken from that output; its recorded trace is empty. A failure that is attributable, flaky or unavailable, and a passing check, behave exactly as before, with the same statuses and messages, and the traced rerun still feeds the flaky comparison and the path fallback for the attributable cases. The base result is still computed once per check name, command and base tree and reused by later rounds; nothing else changes (no new configuration, no change to which drafts pass or are refused).

The existing tests whose expectations relied on the trace of a pre-existing check are updated: `TestStaticChecksAttribution` takes the pre-existing path of `stale` from the check's own output (its command now prints the path), and `TestStaticChecksRecordTimes` no longer expects the traced time in the wall time of its pre-existing check.

## Decisions
- The traced rerun (`"set -x\n"+c.Run`) moves from right after the failing first run to after the base lookup, and is skipped for a check judged pre-existing by the existing `attributable` computation, unchanged (constraint: attributable, flaky, unavailable and passed checks keep their status values and messages) (evidence: shape/static.go) (assumption: A scratch copy outside the checkout built with the correct patch of controls.json; the acceptance scenarios were not run by this session, readiness will observe them.) (source: request)
- A pre-existing result lists its paths from the first run's output only, and its trace stays empty in static-checks.json; the paths of an attributable or flaky result still use the trace (evidence: shape/static.go; shape/static_test.go) (assumption: Existing tests TestStaticChecksAttribution (stale) and TestStaticChecksRecordTimes relied on the trace and must be updated by the Build.) (source: request)
- Acceptance drives the real gate through a Go test harness copied into the shape package by a Go build overlay, because a test the Build must write cannot be run by name; the harness checks record every execution in a marker file as `plain` or `traced` (the check text ran under `set -x`) (evidence: features/steps/approve-static-reuse-line-explains-unknown__steps.js; shape/static_test.go) (assumption: The overlay-harness technique is shown by the cited existing step file; the request leaves acceptance mechanics to the builder.) (source: request)
- The Build also adds one focused model-free test in shape/static_test.go for the same behavior, in the style of the existing gate tests there, since the request asks for it (evidence: shape/static_test.go) (assumption: The request's scripted runner is taken to mean the existing real-shell gate fixtures in shape/static_test.go; no other scripted runner for the gate was found.) (source: request)
- The traced rerun moves to after the base lookup and is skipped for a check judged pre-existing by the unchanged attributable computation (decision: D1, round 1) (constraint: attributable, flaky, unavailable and passed checks keep their status values and messages) (evidence: shape/static.go) (assumption: A scratch copy outside the checkout built with the correct patch; the acceptance scenarios were not run by this session.) (source: request)
- Pre-existing results list paths from the first run's output only and record an empty trace (decision: D2, round 1) (evidence: shape/static.go; shape/static_test.go) (assumption: Existing tests TestStaticChecksAttribution (stale) and TestStaticChecksRecordTimes relied on the trace and must be updated by the Build.) (source: request)
- Acceptance runs the real gate through a Go harness copied in by a go build overlay (decision: D3, round 1) (evidence: features/steps/approve-static-reuse-line-explains-unknown__steps.js; shape/static_test.go) (assumption: The overlay-harness technique is shown by the cited existing step file; the request leaves acceptance mechanics to the builder.) (source: request)
- The Build adds one focused model-free test in shape/static_test.go (decision: D4, round 1) (evidence: shape/static_test.go) (assumption: The request's scripted runner is taken to mean the existing real-shell gate fixtures in shape/static_test.go.) (source: request)

## Source comparison
- Pre-existing check runs the check twice in round 1 (shaped, base) and once in a later round (base reused), never under `set -x`, recorded pre-existing with the same paths and an empty trace: addressed in scenarios "A pre-existing failure never runs the traced rerun" and "A pre-existing failure keeps its status, paths and reused base result".
- A check that passes without the shaped files and fails with them still runs the traced rerun and records its trace: addressed in scenario "A failure only the shaped files cause still runs the traced rerun".
- Flaky-not-attributable behavior unchanged (the case a skip-always implementation would break): addressed in scenario "A failure that differs on the traced rerun is still flaky and not attributable".
- Messages, status values and refusal rules of attributable, flaky, unavailable and passed checks unchanged: omitted by design; kept by the existing tests in shape/static_test.go.
- No new configuration: omitted by design.

## Tests
- A check failing identically with and without the shaped files runs twice in round 1, once in a later round, never under `set -x`, and is recorded pre-existing with its paths and an empty trace: add a focused model-free test in shape/static_test.go (unit)
- A check that passes without the shaped files and fails with them still runs the traced rerun and records its trace: add to the same test in shape/static_test.go (unit)
- Flaky, attributable-by-path, unavailable and passed checks keep their statuses and messages: the existing TestStaticChecksFlakyNotAttributable and TestStaticChecksAttribution (existing)
- The recorded wall time of a pre-existing check no longer includes a traced run: update TestStaticChecksRecordTimes in shape/static_test.go (unit)
- The real gate skips the traced rerun for a pre-existing failure and keeps it for attributable and flaky ones: the acceptance scenarios (scenario)

## Feasibility
- none
