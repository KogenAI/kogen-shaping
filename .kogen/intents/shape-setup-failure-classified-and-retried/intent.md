# Classify and retry Shape setup failures

Slug: shape-setup-failure-classified-and-retried
Model: not applicable — the change is failure classification, one retry and time accounting of one setup step, with no state machine of its own
Acceptance:
- .kogen/intents/shape-setup-failure-classified-and-retried/acceptance.feature
- features/steps/shape-setup-failure-classified-and-retried__steps.js
- features/steps/shape-setup-failure-classified-and-retried__driver/run.js
- features/steps/shape-setup-failure-classified-and-retried__driver/observe_test.go.txt

The configured commands.setup step that Shape runs once before its first model request (in shape/shape.go, through the readiness Setup call) is classified and retried. A setup command that exits non-zero, is killed by a signal or times out is reported as `setup_failure`, never as a draft fault. The report gives the command, the exit code and the last 20 lines of the command's output, and is recorded as a `setup_failure` event in the shaping events with the fields command, exit_code and output (the last 20 lines), once per failed attempt. After the first failure Shape pauses 10 seconds (a wait that ends early only when the context ends) and runs the whole setup once more. If the retry passes, shaping continues normally. If the retry fails too, Shape refuses with the class `environment`, and the refusal text has the command, the exit code and the last 20 lines of output. A setup that passes the first time behaves exactly as today.

The seconds spent in this initial setup (every attempt, and the 10 second pause) are reported as `setup_ms` in the shape_time event. They are not part of the Shape elapsed time: `elapsed_ms` of the event and the elapsed figure of the per-request time line exclude them, so the target comparison of shape-time-target-is-advisory never sees them. `checks_ms` no longer includes the initial setup and `gap_ms` keeps its identity elapsed_ms - shaper_ms - checks_ms - install_ms. The change adds no elapsed-time rule of its own. For example, a scripted setup of 90 seconds followed by a 100 second shaper request reports setup_ms 90000 and a Shape elapsed time of 100 seconds, not 190 seconds. Only the initial Shape setup counts for `setup_ms`; setup runs made later by readiness or static checks on fresh copies are unchanged, and the acceptance observer counts setup runs only up to the first model request, because those later runs are expected.

Drafts that pass and the text of every other refusal are unchanged. The change is limited to the Shape package setup step (shape/shape.go, shape/evidence.go) and new model-free tests in package shape. The acceptance observer runs a scripted shell setup command through the real Shape entry point with ShapeInput.Now as a real clock plus scripted advances.

## Decisions
- A failed Shape setup (non-zero exit, killed by a signal, or timed out) is reported as `setup_failure` with the command, exit code and last 20 output lines and is never reported as a draft fault. (decision: D1, round 1) (evidence: shape/shape.go) (source: request)
- After a first failure Shape waits 10 seconds and runs the setup once more; a failing retry refuses with the class `environment`, carrying the command, exit code and last 20 output lines. A passing retry lets shaping continue. (decision: D2, round 1) (evidence: shape/shape.go; consumes/kogen-verification/readiness_test.go) (source: request)
- setup_ms in the shape_time event is the union of the wall time of the initial setup attempts and the pause, and elapsed_ms, checks_ms and the time line exclude it; a scripted setup of 90 seconds followed by a 100 second shaper request reports setup_ms 90000 and a Shape elapsed time of 100 seconds; gap_ms keeps its identity. (decision: D3, round 1) (evidence: shape/evidence.go; shape/budget.go) (assumption: the progress line may state setup separately; only the event field is required) (source: request)
- The 10 second pause is a fixed constant with no setting; acceptance waits it really, so no test seam is added. (decision: D4, round 2) (evidence: shape/shape.go) (assumption: the pause is a real wait observed through the clock the caller passes as ShapeInput.Now) (source: request)
- A setup that passes first time, drafts that pass and every other refusal text are unchanged; this change adds no elapsed-time rule. (decision: D5, round 1) (evidence: shape/shape.go) (source: request)

## Source comparison
- Addressed: scripted setup fails once and passes on retry; shaping continues and shape_time shows the retry through setup_ms (scenario 1).
- Addressed: setup fails twice; class `environment`, text with command, exit code and the last 20 lines (scenario 2).
- Addressed: a scripted setup of 90 seconds followed by a 100 second shaper request gives setup_ms 90000 (observed in a range from 90000) and elapsed about 100 seconds, not 190 (scenario 3).
- Addressed: signal and timeout use the same failure path as a non-zero exit; they are not scripted in acceptance because the wall time they need is not feasible in a model-free test.
- Omitted: README, contract folder and other refusals.

## Tests
- First setup failure is retried once after a 10 second pause and shaping continues: model-free scripted-setup Shape run (scenario)
- Two setup failures refuse with class environment and the command, exit code and last 20 lines: same harness (scenario)
- setup_ms is reported and excluded from the elapsed time: Shape run with scripted clock advances (scenario)
- Existing time-account tests keep passing with setup absent from checks: existing shape time tests (existing)

## Feasibility
- none
