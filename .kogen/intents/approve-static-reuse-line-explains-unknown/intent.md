# Explain unknown in the approve static reuse line

Slug: approve-static-reuse-line-explains-unknown
Model: not applicable — the change is one progress line chosen by a recorded outcome word; there is no state to model
Acceptance:
- .kogen/intents/approve-static-reuse-line-explains-unknown/acceptance.feature
- features/steps/approve-static-reuse-line-explains-unknown__steps.js
- features/steps/approve-static-reuse-line-explains-unknown__driver/harness_test.go.txt

`kogen intent approve` reuses the static-check result Shape recorded for the identical package and prints a progress line. The line ended with the bare recorded outcome word, so `static checks: reusing the result recorded for this package (unknown)` did not tell an operator whether the checks passed. `unknown` is the outcome of a run where a check failed on the base tree too or could not run, so the shaped files are not established by any check.

The reuse line now says what the reused outcome means. `checked` keeps its current text `kogen: static checks: reusing the result recorded for this package (checked)`. `unknown` reads `kogen: static checks: reusing the result recorded for this package (unknown: a check fails without the shaped files too or could not run, so the shaped files are not established by it; see static-checks.json in the Shape record)` with the real record path inserted before the closing parenthesis as ` at <record path>`, where `<record path>` is the path `staticRecordPath` gives for the project root and slug (the file `writeStaticRecord` wrote). `none` (no check configured) reads `kogen: static checks: reusing the result recorded for this package (none: no static check is configured)`.

Nothing else changes: the recorded outcome words, the reuse rule, which packages are approved, every refusal text, and the configuration. Only the progress line of the reuse path in `approveStaticChecks` (shape/static.go) changes. A focused model-free regression test is added next to `TestApproveStaticChecks` in shape/approve_test.go.

## Decisions
- The reuse line of a recorded `unknown` outcome is the request's text with the record path from `staticRecordPath` joined as ` at <record path>` before the closing parenthesis (decision: D1, round 1) (evidence: shape/static.go; shape/static_test.go) (source: request)
- The reuse line of a recorded `none` outcome is `...(none: no static check is configured)`; the request only asks that `none` be said in words, so this wording is chosen here (decision: D2, round 1) (evidence: shape/static.go) (assumption: the operator accepts this wording for the none outcome) (source: request)
- The `checked` line, the recorded outcome words, the reuse rule and the approval decision are untouched; no setting is added (decision: D3, round 1) (evidence: shape/static.go; shape/approve_test.go) (source: request)
- Acceptance drives the real `Approve` through a Go test harness copied into the shape package by a Go build overlay (an unwritten test is not run by name), reading the progress line from the approve session's stderr and counting executions of the configured check with a marker file (decision: D4, round 1) (evidence: shape/approve_test.go; features/steps/fix-k-shape-wrong-implementation__steps.js) (source: request)

## Source comparison
- Observed bare line, from the request: `static checks: reusing the result recorded for this package (unknown)`; superseded by the explained line in acceptance.feature scenario "A recorded unknown outcome is reused and explained".
- Observed Shape output that precedes it in the request, kept as context only and not changed: `static check test failed (exit 1) without the shaped files too: pre-existing failure in app/models/account/data_transfer/manifest.rb:54, ...; the shaped files are not established by this check`; every refusal text stays as it is.
- Expected line, from the request: `kogen: static checks: reusing the result recorded for this package (unknown: a check fails without the shaped files too or could not run, so the shaped files are not established by it; see static-checks.json in the Shape record)` plus the real record path; addressed in the same scenario with the path inserted as ` at <record path>`.
- `checked` reuse line unchanged: addressed in scenario "A recorded checked outcome keeps its line".
- `none` reuse line says so in words: addressed in scenario "A recorded none outcome says so in words".
- Reuse without running the check (marker file count): addressed in the unknown scenario.
- No new configuration, no change to which drafts pass or to any refusal text: omitted by design.

## Tests
- The reuse line of a recorded `unknown` outcome contains `unknown:` and the record path, and the check is not run again: add a regression test next to TestApproveStaticChecks in shape/approve_test.go (regression)
- The reuse line of a recorded `checked` outcome is unchanged: the existing assertion in TestApproveStaticChecks (existing)
- The reuse line of a recorded `none` outcome says so in words: add to the new regression test in shape/approve_test.go (regression)

## Feasibility
- none
