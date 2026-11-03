# Show stderr tails in self-check refusals

Slug: k-shape-selfcheck-stderr
Model: not applicable — diagnostic formatting adds no meaningful workflow state
Acceptance:
- .kogen/intents/k-shape-selfcheck-stderr/acceptance.feature
- features/steps/k_shape_selfcheck_stderr_steps.js
- features/steps/k_shape_selfcheck_stderr_test.go.txt

When step-selfcheck refuses a failing command, include its bounded, secret-redacted stderr tail in the refusal sent back to the shaper and the final user-facing exhaustion diagnostic. Replace the current exit-only observer diagnostic, not the refusal protocol. Preserve command/check identity, actual and expected exits and timeout information. Use existing verification redaction before excerpting; retain no raw secrets in self-check/refusal events. Use a fixed 4096-byte maximum for the stderr excerpt after redaction (short stderr remains intact; empty stderr adds no misleading cause). No model/provider runs, CLI commands or flags are added. Work from current main, retaining slug ownership checks rather than reverting them to the earlier 33338a6 candidate.

## Decisions
- Surface an actionable bounded stderr tail with secrets redacted when self-check refuses. (source: request)
- Synthetic tests only; no model runs or new CLI commands or flags. (source: request)
- Preserve current main's slug ownership checks and test redaction together with them. (source: request)

## Source comparison
- addressed: request Need and Problem (stderr still absent after 95a6204) are covered by the diagnostic contract above and acceptance's real Shape submission/retry/exhaustion fixture.
- addressed: request landing history (3 Nov 08:53Z, candidate 33338a6, later K-STEP-FILE-COLLISION, TestSelfcheckObserverExhaustionRedactsRetainedEvents failing normal/race checks) is covered by current-main ownership-compatible fixtures, retained-event secret assertions, and required normal/race verification. The historical test name is not presumed present on this tree.
- addressed: request Slug and artifact paths are preserved in this package. README and testing-rules commands remain required; synthetic golden regeneration uses go test ./... -update only when needed, followed by diff inspection and full checks.
- omitted: private ~/Scratch.noindex/kogen-lanes-private/notes-harness/climb/verification/LANE.md L1 and l1probe/results/ are historical evidence references, not available executable fixtures or implementation inputs. Their claimed cause is reproduced synthetically here; no private original is copied.
- addressed: complete fix scope is diagnostic output, boundedness, redaction and current-main compatibility; no unrelated workflow, provider or UI behavior is changed. The 4096-byte excerpt limit settles the unspecified technical bound.

## Tests
- Failing self-check exposes actual command stderr through Shape feedback and exhaustion: supplied overlay regression scenario with real local shell stderr (regression)
- Diagnostic excerpt retains the end, excludes old head and is bounded: supplied overlay long-output refusal plus focused 4095/4096/4097-byte excerpt tests to add during implementation (unit)
- Short and empty stderr preserve useful existing exit/check diagnostics without invented cause: add focused diagnostic tests (unit)
- Redact recognized secrets before truncation and before retention: overlay checks environment and labeled secrets in feedback/events; add boundary-straddling redaction test (regression)
- Current slug ownership validation, retries, publication rules and warnings remain unchanged: existing self-check/ownership tests and normal go test ./... plus go test -race -count=1 ./... (existing)

## Feasibility
- Local dependency inspection confirmed verify.Redactor masks environment and labeled secrets before excerpting; no external protocol uncertainty or documentation lookup is needed. (probe: exit 0)
- evidence: {"id":"stderr-observer","kind":"observer","inputs":["features/steps/k_shape_selfcheck_stderr_test.go.txt"],"observers":["features/steps/k_shape_selfcheck_stderr_steps.js","features/steps/k_shape_selfcheck_stderr_test.go.txt"],"checks":[{"name":"good","command":"node features/steps/k_shape_selfcheck_stderr_steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/k_shape_selfcheck_stderr_steps.js bad","exit":1,"output":"rejected\n"}]}
