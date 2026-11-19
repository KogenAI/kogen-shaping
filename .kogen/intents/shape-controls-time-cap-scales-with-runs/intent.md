# Scale the controls time cap with the planned runs

Slug: shape-controls-time-cap-scales-with-runs
Model: not applicable — one arithmetic rule and message text inside the existing wrong-implementation check, with no state that outlives a check
Acceptance:
- .kogen/intents/shape-controls-time-cap-scales-with-runs/acceptance.feature
- features/steps/shape-controls-time-cap-scales-with-runs__steps.js
- features/steps/shape-controls-time-cap-scales-with-runs__harness.js

The wrong-implementation check (`shape/controls.go`, `controlsLines`) stops with a time cap refusal when the runs of one candidate together pass a fixed 300 seconds, however many runs the candidate makes (at most 8), and tells the shaper to make the implementations or steps faster, which it cannot do when the project's own run is slow. After this change:

1. The time cap of a candidate is the larger of 300 seconds and 60 seconds per planned run, where the planned runs are the correct run plus the wrong implementations (n). A candidate of 8 runs has 480 seconds; a candidate of 3 runs keeps 300 seconds. The cap is computed from n in `controlsLines`; the existing 300 second constant is the base, and the 60 second per-run figure is a new constant (no flag, setting or environment variable).
2. The time of the check (its start, its deadline and each remaining-time test) is read through the shaping's injected clock, `sh.ev.clock()`, instead of `time.Now`/`time.Until`, so a test can fake the time the runs take.
3. The time cap line is exactly, for a run stopped after `C` of `n` runs completed under a cap of `S` seconds (`A` = whole seconds elapsed on the clock since the check began, divided by `C`, integer division):
   `- time cap: the runs exceeded the total time cap of S seconds per candidate (n planned runs; the cap is the larger of 300 seconds and 60 seconds per planned run) after C of n runs completed; the check stopped. The completed runs took A seconds on average.` followed by, when the correct run alone took more than half of S seconds (measured on the clock), ` The project's own run is the slow part: the correct implementation's run alone took R seconds, more than half of the cap. Make the project's setup or acceptance command faster, or supply fewer wrong implementations.` and otherwise ` Make the implementations or the steps faster.` (R = whole seconds the correct run took).
4. The `wrong_implementation_check` event gains the fields `time_cap_seconds` (S) and `planned_runs` (n) for every check that plans runs; its other fields are unchanged.
5. A run that finished is never discarded because of the cap: when the cap stops the check after C runs, the refusal has the time cap line first, then, for each finished wrong implementation the judgement rejects (rules 2 to 5 of `controlsJudge`), the usual property problem line of that finding (`controlsPropertyLine`). Properties whose wrong implementations have not run yet add no line. Finished wrong implementations that the judgement accepts add no line.

`controlsMaxRuns` (8), `controlsMaxFiles`, `controlsMaxBytes`, the judgement of each run, the wrong-implementation refusal headline and the shaping budget as a whole do not change. Existing tests that pass a base cap to `controlsLines` or expect the old text are updated to the new rule and text. Add `shape/controls_time_cap_test.go` with model-free Go tests `TestControlsTimeCapEightRunsAtFiftySeconds`, `TestControlsTimeCapSmallCandidateKeepsBase`, `TestControlsTimeCapSlowProjectMessage` and `TestControlsTimeCapKeepsFinishedVerdicts`, using the existing controls fixtures (`controlsReal`) and a fake clock for the runs.

## Decisions
- The per-candidate cap is max(300 s, 60 s × n), n = 1 + wrong implementations, so it never passes 480 s with `controlsMaxRuns` at 8; 60 s per run is a new constant and nothing is configurable. (constraint: controlsMaxRuns, controlsMaxFiles and controlsMaxBytes keep their values.) (evidence: shape/controls.go) (source: request)
- The check reads time through the shaping's injected clock `sh.ev.clock()`, which already exists for tests, so acceptance can fake the duration of runs without waiting. (evidence: shape/evidence.go; shape/clock_bound_test.go) (source: request)
- The time cap line has the exact text in the outline above, naming the cap and the planned runs and giving the average run time in whole seconds; when the correct run alone took more than half the cap it names the project's own run as the slow part. The refusal headline text is left unchanged. (evidence: shape/controls.go) (assumption: The exact wording of the new time cap line is chosen here, not by the request; the user confirms it.) (source: request)
- When the cap stops the check, verdicts of finished wrong implementations that the judgement rejects are reported after the time cap line, and the event records `time_cap_seconds` and `planned_runs`. (decision: D4, round 1) (evidence: shape/controls.go) (assumption: The request says a finished run keeps its verdict; reporting it as the usual property problem line after the time cap line is this Intent's reading.) (source: request)
- Per-candidate time cap is max(300 s, 60 s x planned runs), planned runs = correct run + wrong implementations; the 60 s figure is a new constant, nothing configurable. (decision: D1, round 1) (constraint: controlsMaxRuns, controlsMaxFiles and controlsMaxBytes keep their values) (evidence: shape/controls.go) (assumption: none) (source: request)
- The check reads time through the shaping's injected clock sh.ev.clock() so acceptance fakes run durations. (decision: D2, round 1) (evidence: shape/evidence.go; shape/clock_bound_test.go) (assumption: I did not run the harness; readiness will show whether the fake clock via trace line count works.) (source: request)
- Exact wording of the new time cap line, and finished rejected wrong verdicts reported after it, are chosen by this Intent. (decision: D3, round 1) (evidence: shape/controls.go) (assumption: Wording is not given by the request; user confirms it.) (source: request)

## Source comparison
- Addressed: cap scales as the larger of 300 s and 60 s per planned run (change 1; acceptance scenarios 1 and 2).
- Addressed: refusal text, event and problems line state the cap and planned runs (changes 3 and 4; scenarios 2 and 3 for the text, scenario 1 for the event).
- Addressed: average run time and the project-run-is-slow message when the correct run took more than half the cap (change 3; scenario 3).
- Addressed: finished runs keep their verdicts when the cap is passed (change 5; scenario 4). Unchanged: controlsMaxRuns, controlsMaxFiles, controlsMaxBytes, judgement, no new flag, setting or environment variable.
- Release evidence: the Build cannot run the assembly suite; on main before the change `go test ./shape -run TestControlsTimeCap` fails (the four tests and the acceptance scenarios fail) and after the change they pass.

## Tests
- A candidate of 8 planned runs of 50 s each (400 s) is not stopped and is judged on all 8 runs: TestControlsTimeCapEightRunsAtFiftySeconds (integration)
- A candidate of 3 planned runs taking 400 s is stopped at 300 s and the refusal names the cap of 300 s and 3 planned runs: TestControlsTimeCapSmallCandidateKeepsBase (integration)
- A candidate of 8 planned runs whose correct run takes 300 s is stopped at 480 s with the project-run-is-slow message and the average: TestControlsTimeCapSlowProjectMessage (integration)
- A wrong implementation that passed before the cap stopped the check is still reported: TestControlsTimeCapKeepsFinishedVerdicts (integration)
- Existing cap and run-cap tests keep working under the new rule: TestWrongImplementationCaps (existing)

## Feasibility
- none
