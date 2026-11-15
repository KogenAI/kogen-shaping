# Record per-check time in static checks events

Slug: shape-static-checks-record-per-check-time
Model: not applicable — the change adds wall-clock measurements to an existing event; it has no state machine or interacting rules
Acceptance:
- .kogen/intents/shape-static-checks-record-per-check-time/acceptance.feature
- features/steps/shape-static-checks-record-per-check-time__steps.js
- features/steps/shape-static-checks-record-per-check-time__harness.js

The `static_checks` event of a Shape run carries attempt, outcome, reused, dir, head, tree, sha256, files and per-check name, run, status, exit, paths, output and trace, but no time. A slow static gate (for example a Rails project whose `bin/rails test` check takes about 150 s per attempt while the suite itself reports 31 s) therefore cannot be attributed to the check run, the base-tree run or the fresh copy.

After this change each entry of `checks` in the `static_checks` event carries `wall_ms`, the wall time of the run of that check on the candidate in the fresh copy (the plain run plus the traced re-run of a failing check), and `base_wall_ms`, the wall time of the base-tree run of that check (the run without the shaped files) when that run happened in this gate run. The event carries `copy_ms`, the time to build the fresh copy of the candidate and its seeded dependencies. All three are integers in milliseconds. A check that was not run (it runs the test suite) has `wall_ms` 0, a check whose base run did not happen (it passed or was unavailable, or its base result was reused from an earlier round) has `base_wall_ms` 0, and a gate run that made no copy (no check applies) has `copy_ms` 0. When the event reports `reused` true (the identical candidate is submitted again after a static refusal), it repeats the timings recorded for the original run of that candidate (the same `copy_ms`, `wall_ms` and `base_wall_ms` values), not new ones. The recorded static check result file keeps the same additive fields. Older events without the fields stay readable; the fields are additive.

Nothing else changes: which drafts pass, the refusal text, the checks that run, and the `shape_time` accounting (`checks_ms` still equals the sum it records today) stay exactly as they are. The existing progress lines `running static checks ...` and `static check <name> failed ...` do not change, except that the line announcing a finished run may, but need not, end with ` (<n> s)`. No setting, flag or environment variable is added.

## Decisions
- `wall_ms`, `base_wall_ms` and `copy_ms` are added to the `static_checks` event as additive integer millisecond fields on each `checks` entry and on the event; a not-run value is 0. (constraint: no new configuration; older events stay readable) (evidence: shape/static.go; shape/static_test.go) (source: request)
- `wall_ms` covers the plain run and the traced re-run of a failing check on the candidate; `base_wall_ms` covers only a base run that actually executed in that gate run, so a base result reused from the memo gives 0.  (evidence: shape/static.go) (assumption: whether the traced re-run counts toward wall_ms is my reading of "the time of the run of that check on the candidate") (source: request)
- A reused `static_checks` event repeats the timings stored with the original run instead of measuring anew; the timings live in the retained static run so the event and the recorded result file agree. (evidence: shape/static.go) (source: request)
- The progress lines `running static checks ...` and `static check <name> failed ...` are not changed; the optional ` (<n> s)` suffix on a finished-run line is not added. (constraint: refusal text byte-identical) (evidence: shape/static.go; shape/static_test.go) (assumption: the optional suffix is omitted) (source: request)
- Acceptance drives a real Shape run with a scripted shaper through a Go test overlay, reads the `static_checks` events, and pins the refusal problem lines byte for byte. The reused result is observed by submitting the identical refused package twice. The regression test in shape/static_test.go next to the existing static gate tests is written by the Build. (evidence: shape/static_test.go; features/steps/shape-static-check-base-run-once__steps.js) (source: request)

## Source comparison
- Request: `wall_ms` per check, `base_wall_ms` when the base run happens, `copy_ms` on the event, integers in ms, 0 when not run — addressed by the prose and the timing scenario.
- Request: reused result records the original timing again — addressed by the refusal scenario (second, reused event equals the first).
- Request: refusal text byte-identical, no change to which drafts pass — addressed by the refusal scenario that pins the problem lines and the publication after repair.
- Request: progress lines `running static checks ...` and `static check <name> failed ...` unchanged except an optional ` (<n> s)` — addressed: decided unchanged.
- Request: `shape_time` accounting stays — addressed by prose; not separately observed here, the existing evidence tests cover it.
- Request: model-free regression test in shape/static_test.go — addressed in Tests; written by the Build.

## Tests
- Each check entry records `wall_ms` greater than 0 when it ran, `base_wall_ms` greater than 0 when its base run ran, 0 otherwise, and the event records `copy_ms`: scripted-shaper Shape run with checks that sleep (scenario)
- A reused result records the original timings again: refusal scenario (scenario)
- The refusal text of a failing check is byte-identical to the one produced before: refusal scenario (scenario)
- Focused model-free test with a configured sleeping check, a reused result and an unchanged refusal text: new test in shape/static_test.go (unit)

## Feasibility
- none
