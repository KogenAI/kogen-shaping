# Reuse static check base runs within one base tree

Slug: shape-static-check-base-run-once
Model: not applicable — the base-tree result of a check is a pure function of (check name, check command, base tree); the only state is a memo keyed by that triple, with no ordering or interacting rule to model
Acceptance:
- .kogen/intents/shape-static-check-base-run-once/acceptance.feature
- features/steps/shape-static-check-base-run-once__steps.js
- features/steps/shape-static-check-base-run-once__harness.js

When a configured static check fails on the shaped files, the static gate re-runs that check without the shaped files (the base run) to decide whether the failure is pre-existing or attributable. On real Rails projects that base run takes about 110 s and repeats on every Shape round although the result cannot change. Today each round shows the progress `running static checks on the shaped files in a fresh copy: test` and then, after about 110 s, `static check test failed (exit 1) without the shaped files too: pre-existing failure in ...`, with the same base tree and the same check every time.

After this change the base-tree result of a static check is computed once per (check name, check command, base tree) and reused within one Shape run and by the Approve that follows on the same base commit. The run with the shaped files still happens on every round. Which drafts pass, which are refused, the attribution rules and every refusal message stay exactly as they are. The only differences are that the base run is skipped when a result for the same (check name, check command, base tree) exists, and that the progress line of a check judged pre-existing carries the marker ` (base result reused)` right after `without the shaped files too`, for example `kogen: static check test failed (exit 1) without the shaped files too (base result reused): ...` (the rest of the line, `pre-existing failure in <paths>; the shaped files are not established by this check`, is unchanged). The line of a check whose base run actually ran is unchanged, and the same marker is added to the progress line of an attributable failure whose base result was reused (`kogen: static check <name> failed (exit <n>) because of the shaped files (base result reused): ...`). No setting, flag or environment variable is added. A changed base tree (HEAD's tree), check command or check name invalidates the reuse; the reused result is the base run's raw outcome (exit status and output), judged afresh against each round's shaped files and outputs.

## Decisions
- The memo is keyed by check name, check command and the tree of HEAD without the shaped files, and holds the raw base-run result so each round's attribution is computed from it against that round's own output and shaped paths. (constraint: no new configuration; a changed tree, command or name never reuses) (evidence: shape/static.go; shape/static_test.go) (source: request)
- The memo survives across Shape rounds (a new gate runner is built each round) and is retained so that Approve on the same base commit reuses it; where it is stored is the implementation's choice. (constraint: stale entries for another tree, command or name are never used) (evidence: shape/static.go) (source: request)
- The `(base result reused)` marker is added to the progress line of the base-judged outcomes (pre-existing and attributable) only when the base result came from the memo; refusal messages and recorded check statuses are unchanged. (constraint: refusal text byte-identical) (evidence: shape/static.go) (source: request)
- Acceptance observes the reuse across Shape rounds and that distinct check names and distinct commands each get their own base run. Reuse by Approve and invalidation by a changed base tree are requested but are not observed by an automatic step here; they need human evaluation in review of the final tree, and no scenario claims them. (assumption: the reviewer confirms Approve reuse and tree invalidation by reading the implementation) (source: request)
- Memo of the base-run result keyed by (check name, check command, tree of HEAD), holding the raw base-run result and judged afresh each round (decision: D1, round 1) (constraint: no new configuration) (constraint: changed tree, command or name invalidates) (evidence: shape/static.go; shape/static_test.go) (assumption: Where the memo is retained for Approve reuse is left to the implementation; Approve reuse is not covered by acceptance.) (source: request)
- Memo survives Shape rounds and is retained for Approve on the same base commit (decision: D2, round 1) (evidence: shape/static.go) (assumption: Storage location unverified.) (source: request)
- Marker (base result reused) added to progress lines of base-judged outcomes only when the memo supplied the result (decision: D3, round 1) (evidence: shape/static.go) (assumption: The attributable-line marker is my extension and untested.) (source: request)
- Acceptance observes Shape-round reuse and name/command separation; Approve reuse and tree invalidation need human evaluation in review (decision: D4, round 1) (assumption: Reviewer must confirm Approve reuse and tree invalidation by reading the implementation.) (source: request)

## Source comparison
- Request: compute the base-tree result once per (check name, check command, base tree) within one Shape run and the following Approve — addressed by D1 and D2; acceptance observes the Shape part.
- Request: progress line shows `kogen: static check test failed (exit 1) without the shaped files too (base result reused): ...` — addressed in the Intent prose and the acceptance scenario, which pins the exact lines of round 1 and round 2 for three checks.
- Request: shaped-file run still happens each round; passing drafts and refusal messages unchanged — addressed: the scenario shows the check with the shaped files running in both rounds.
- Request: regression test with a scripted shaper refused twice, a marker file counting base executions, reused line in round 2 — addressed by the acceptance scenario and its harness.
- Request: changed tree/command/name invalidates — name and command are observed by the scenario (twin runs its own base run; other has its own); the tree is covered only by D4.

## Tests
- The base run of each failing check executes once per (name, command) across two refused Shape rounds and round 2 prints the reused-result lines: scripted-shaper Shape run counting executions with marker files (scenario)
- The first round still prints the unchanged lines without the marker: same scenario (scenario)
- A differently named check with the same command and a check with a different command each get their own base run: same scenario (scenario)

## Feasibility
- none
