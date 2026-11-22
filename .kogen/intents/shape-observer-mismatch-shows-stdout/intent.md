# Show stdout in observer control mismatches

Slug: shape-observer-mismatch-shows-stdout
Model: not applicable — diagnostic text of one self-check item; no meaningful state
Acceptance:
- .kogen/intents/shape-observer-mismatch-shows-stdout/acceptance.feature
- features/steps/shape-observer-mismatch-shows-stdout__steps.js
- features/steps/shape-observer-mismatch-shows-stdout__driver/driver_test.go.txt

When the step self-check re-runs an observer control and the control's exit code differs from the recorded one, the item's detail keeps its first line (`exit <actual>, recorded exit <recorded>`) and its stderr line (`stderr: ` and the bounded redacted stderr tail, only when stderr is not empty) exactly as today, and adds after them one line starting `stdout: ` followed by the last 4096 bytes of what the control wrote on stdout, redacted with the same redactor as stderr (redaction first, then the cut) and cut at a character boundary so the text is valid UTF-8. When stdout is empty no stdout line is added. On a timeout the stdout line comes before the existing `, timed out` sentence. The compared exit code stays that of the control command itself. A control whose exit code matches the recorded one has no detail, whatever it printed. Each control gets its own stdout capture. Only shape/stepcheck.go and shape/stepcheck_test.go change; the stdout of a control is no longer discarded.

The refusal that motivated this change read `observer layout-observer good: exit 2, recorded exit 0` and `observer layout-observer bad: exit 2, recorded exit 1`, with no stderr line under either, while the recorded runs printed their cause on stdout. After this change such a refusal also shows the stdout line.

## Decisions
- The stdout line follows the stderr line; the stderr tail keeps its own trailing newline and the stdout line follows it with no blank line between them. (source: request)
- The stdout excerpt is the last 4096 bytes after redaction, cut at a character boundary, with the same redactor as stderr; no setting is added. (evidence: shape/stepcheck.go; shape/stepcheck_test.go) (source: request)
- A control that matches its recorded exit code has no detail, and an empty stdout adds no line. (source: request)
- Acceptance drives the real observer self-check through a Go test file overlaid into package shape, because the check is unexported. (evidence: features/steps/k_shape_selfcheck_stderr_steps.js; features/steps/k_shape_selfcheck_stderr_test.go.txt) (source: request)

## Source comparison
- addressed: the stdout line after the exit line and the stderr line, 4096-byte tail, redaction before the cut, character boundary, empty stdout adds nothing (scenarios 1-6); timeout order (scenario 7, with the probe deadline shortened by the driver through verify.ContractProbeDeadline).
- addressed: the motivating refusal lines `observer layout-observer good: exit 2, recorded exit 0` and `observer layout-observer bad: exit 2, recorded exit 1` are the item targets and details this change extends with a stdout line.
- addressed: the exit comparison, how declared observers are found, prompts, docs, README and the verify package are unchanged.

## Tests
- A mismatching control's detail gains the stdout line after the exit and stderr lines: acceptance scenarios 1 and 2 (scenario)
- The stdout line is the redacted last 4096 bytes, valid text: acceptance scenario 3 plus a unit test in shape/stepcheck_test.go (scenario)
- A matching control has no detail and an empty stdout adds no line: acceptance scenarios 4 and 5 (scenario)
- Each control has its own stdout: acceptance scenario 6 (scenario)
- On a timeout the stdout line comes before the timed out sentence: acceptance scenario 7 (scenario)

## Feasibility
- none
