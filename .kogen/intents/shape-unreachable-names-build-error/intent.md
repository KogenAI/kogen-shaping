# Report build errors in controls refusals

Slug: shape-unreachable-names-build-error
Model: not applicable — bounded string formatting has no meaningful state
Acceptance:
- .kogen/intents/shape-unreachable-names-build-error/acceptance.feature
- features/steps/shape-unreachable-names-build-error__steps.js
- features/steps/shape-unreachable-names-build-error__driver/probe_test.go

Expose the diagnostic behind an echoed command in controls refusal reasons instead of reporting only the command. Add shape/failmessage.go with controlsFailureMessage(message string) string. Use it only at the controlsUnreachable reason and the wrong-implementation “fails scenario ... without an assertion” reason. Keep controls.go production edits to those two call sites; leave controlsFirstLine and the Assertion event field unchanged. No changes to acceptance judgment, event schema, or execution are requested.

Normalize each line with the existing whitespace normalization, discard empty lines, and retain the first non-empty line. For multiline messages (including when the first line ends in a colon), append following non-empty lines using ` | `. Prefer lines containing a path:line or path:line:col location over other following lines, preserving source order within each group. Retain at most three extra lines and 400 Unicode characters total including separators; truncate the final included line as necessary, without appending an empty separator. A single-line assertion remains unchanged unless it exceeds the total bound. Empty input produces an empty reason.

During implementation, add failing regression tests first, observe their failures against the old formatter, then implement. Cover the supplied historical build diagnostic, unchanged single-line assertions, both bounds with 50 following lines, and the controlsUnreachable problem line naming the undefined identifier. Add focused unit tests for the formatter and regression tests for its two reason entry points. Run `go test ./shape` after implementation; retain useful existing tests and update expectations deliberately changed by multiline reporting.

## Decisions
- Preserve the first line while adding bounded location-preferred diagnostic context in only the two requested refusal reasons. (source: request)
- Leave controlsFirstLine and the Assertion event field unchanged; the acceptance harness observes the actual event extraction as well as both refusal entry points. (evidence: shape/controls.go) (source: request)
- Use stable location-first ordering, count Unicode characters including separators, and truncate the last included line to fit the bound. These settle the requested preference and character cap without adding new settings. (source: request)

## Source comparison
- Addressed: Request Change maps to the formatter contract above, shape/failmessage.go and exactly two controls.go call-site edits; controlsFirstLine and Assertion remain unchanged.
- Addressed: Request's 10 November 2026 observation on review-reminder-reads-unread-parts, kogen v0.1.0-dev.cd805d8f, contract_review explains the failure hidden by first-line reporting. acceptance.feature carries the exact supplied /x/driver reproducer, package diagnostic and undefined required diagnostic through both real reason entry points. It does not reproduce or change that historical project's build.
- Addressed: Request's four regression requirements map to historical-diagnostic, single-line and 50-line acceptance cases, focused implementation tests, and the required `go test ./shape` run. Existing multiline refusal expectations must change rather than suppressing the additional diagnostic.
- Omitted: no requested behavior. No scope is superseded or intentionally conflicting.

## Tests
- Retain the build diagnostic in controlsUnreachable reasons: reproduce the supplied message through controlsUnreachable and compare the full problem line (regression)
- Retain the diagnostic in non-assertion wrong-implementation reasons: exercise controlsJudge with the same command failure (regression)
- Prefer location-bearing lines while preserving normalized first and following lines: add focused controlsFailureMessage tests and compare refusal outputs (unit)
- Preserve one-line assertion reporting and the Assertion event field: compare one-line and historical multiline observations (regression)
- Bound diagnostic context to three extra lines and 400 characters: test 50-line inputs at the line limit and around the character limit (unit)
- Preserve other shape behavior: run go test ./shape after updating changed multiline expectations (existing)

## Feasibility
- The isolated Go-overlay harness compiles the actual shape package tests and emits JSON observations, not compiler diagnostics as payload. Initial controls preparation found and corrected a conflicting regexp variable name and an incomplete scenario fixture. Final observer controls accept the candidate formatter and reject the old unreachable call site.
- evidence: {"id":"failure-observer","kind":"observer","inputs":[".kogen/intents/shape-unreachable-names-build-error/controls.json"],"observers":["features/steps/shape-unreachable-names-build-error__steps.js","features/steps/shape-unreachable-names-build-error__driver/probe_test.go"],"checks":[{"name":"good","command":"node features/steps/shape-unreachable-names-build-error__steps.js good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-unreachable-names-build-error__steps.js bad","exit":1,"output":"observer rejected\n"}]}
