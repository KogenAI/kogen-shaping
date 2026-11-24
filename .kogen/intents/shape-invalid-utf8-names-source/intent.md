# Name the source of invalid UTF-8 requests

Slug: shape-invalid-utf8-names-source
Model: not applicable — this is a stateless input diagnostic change
Acceptance:
- .kogen/intents/shape-invalid-utf8-names-source/acceptance.feature
- features/steps/shape-invalid-utf8-names-source__steps.js
- features/steps/shape-invalid-utf8-names-source__driver/observe.js

Make the invalid UTF-8 refusal identify the request source so the caller can locate the offending input. Limit implementation changes to cmd/kogen-shape/main.go and cmd/kogen-shape/main_test.go. Replace the generic invalid UTF-8 diagnostic with one naming the request file or identifying stdin. Do not change request validation rules or Shape behavior. Add regression coverage for both sources while retaining useful existing input tests.

## Decisions
- Invalid UTF-8 read from a request file returns exit code 2 and an error naming that file; invalid UTF-8 read from stdin returns exit code 2 and identifies stdin as the source, not a filesystem path. Preserve request validation rules and Shape behavior. (decision: D1, round 1) (evidence: cmd/kogen-shape/main.go; cmd/kogen-shape/main_test.go) (source: request)
- Both refusals still produce exactly one JSON result on stdout with no Intent. Preserve the documented newline, matching exit_code, null intent, and diagnostic error string; do not print diagnostics outside the JSON result on stdout. (decision: D2, round 1) (evidence: README.md; cmd/kogen-shape/main.go) (source: request)

## Source comparison
- addressed: Request evidence ev-3ff5b4d4 identifies the generic "request must be nonempty UTF-8 text" refusal in cmd/kogen-shape/main.go and the missing invalid UTF-8 coverage in cmd/kogen-shape/main_test.go. The outcome above and both acceptance scenarios replace only that source-less invalid UTF-8 diagnostic and require regression tests in the scoped test file.
- addressed: Request acceptance 1 maps to "Invalid UTF-8 from a request file", checking the actual supplied path rather than accepting a generic error. Request acceptance 2 maps to "Invalid UTF-8 from stdin", requiring stdin and rejecting filesystem-source diagnostics.
- addressed: Request acceptance 3 and README.md's standalone output contract map to the JSON refusal step in both scenarios: actual process status 2, one JSON object and newline, matching exit_code, null intent, and diagnostic string. controls.json includes wrong implementations for a generic file error, stdin misidentified as a path, diagnostics outside JSON, an Intent on refusal, and the wrong process status.
- omitted: No requested behavior is omitted. No validation rule, empty-input behavior, provider integration, or Shape workflow change is authorized. Implementation remains confined to the two request-scoped Go files; the remaining files are acceptance and proof artifacts.

## Tests
- File-source invalid UTF-8 refusal: add CLI regression coverage in cmd/kogen-shape/main_test.go and run the built command with a real invalid request file in acceptance (regression)
- Stdin-source invalid UTF-8 refusal: add CLI regression coverage in cmd/kogen-shape/main_test.go and run the built command with invalid bytes on stdin in acceptance (regression)
- One JSON refusal with no Intent and matching exit status for both sources: parse the complete stdout of the built command in both acceptance scenarios (scenario)
- Existing validation and Shape behavior remain unchanged: retain TestCommandInput and run the configured existing checks (existing)

## Feasibility
- The local standalone command built successfully and the unchanged stdin invalid UTF-8 refusal returned process status 2 with the generic JSON diagnostic. No external capability or documentation lookup is needed for this local diagnostic change. (probe: exit 0, 2.0 s)
- evidence: {"id":"refusal-observer","kind":"observer","inputs":["features/steps/shape-invalid-utf8-names-source__driver/observe.js"],"observers":["features/steps/shape-invalid-utf8-names-source__driver/observe.js"],"checks":[{"name":"good","command":"node features/steps/shape-invalid-utf8-names-source__driver/observe.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/shape-invalid-utf8-names-source__driver/observe.js bad","exit":1,"output":"rejected\n"}]}
