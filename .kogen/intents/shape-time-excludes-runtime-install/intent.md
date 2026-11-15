# Exclude runtime install from Shape budget

Slug: shape-time-excludes-runtime-install
Model: not applicable — this changes time accounting and an existing budget predicate, not the shaping lifecycle or its state transitions
Acceptance:
- .kogen/intents/shape-time-excludes-runtime-install/acceptance.feature
- features/steps/shape-time-excludes-runtime-install__steps.js
- features/steps/shape-time-excludes-runtime-install__driver/observe.go
- features/steps/shape-time-excludes-runtime-install__driver/qualify.js

Exclude the one-off managed Claude Code runtime installation from Shape's existing 15-minute shaping budget. Keep wall-clock elapsed time intact, expose installation separately as `install_ms` on every `shape_time` event (including zero on warm hosts), and remove that duration from `gap_ms`. Preserve the identity `gap_ms = elapsed_ms - shaper_ms - checks_ms - install_ms` with non-negative components. Installation must not also count as shaper, tools or checks time. The budget still charges the model, tools, checks and ordinary non-install overhead; no new configuration, timeout or option is introduced.

The progress line `kogen: shape time ...` appends `, runtime install <seconds to one decimal> s` only for positive `install_ms`; warm-host progress stays byte-for-byte unchanged. Budget refusal gate/budget and attempt timing lines additionally name `install_ms=<milliseconds>` only when positive. Other refusal wording, classes, reasons, details and behavior stay unchanged. Do not revise unrelated refusal names or runtime download/authentication behavior.

Add a focused, model-free regression test in package `shape`, using the existing scripted Shape executor and fake clock. Exercise actual `Shape`, its event file, progress output and budget result with measurable runtime preparation; no paid model or real download is needed. The owned acceptance helper supplies its own test through a temporary Go overlay, so acceptance does not depend on a test the Build has yet to invent. The Build must also retain normal package regression coverage outside protected acceptance artifacts.

## Decisions
- Measure the cold runtime preparation interval around executor startup only when the managed runtime transitions from absent to installed; a warm runtime startup is ordinary charged overhead. (decision: D1, round 1) (constraint: Do not exclude all executor startup time or change executor authentication and download behavior.) (evidence: shape/external.go) (source: request)
- Keep the existing 900-second budget boundary and refusal vocabulary; subtract installation from the elapsed time used by that boundary and name positive installation milliseconds on the budget and attempt timing lines. (decision: D2, round 1) (constraint: No new configuration and no warm-host progress or refusal wording changes.) (evidence: shape/budget.go; shape/budget_test.go) (source: request)

## Source comparison
- Addressed: request Expected and Constraints map to the accounting identity, always-present `install_ms`, conditional progress suffix, unchanged warm line and unchanged charged 15-minute budget specified above and in every acceptance scenario. The cold replay tests the supplied elapsed/shaper/checks/install arithmetic and proves a second round remains possible.
- Addressed: the request's AoR climb release, refused data-export cell, second billing cell, runtime version, event filename and observed warm-host gaps are preserved as historical context in acceptance.feature, not represented as new measurements. The replay uses the supplied durations rather than trying to download the historical runtime or access those cell directories.
- Addressed: the requested focused model-free regression is specified above and exercised through Shape by observe.go; boundary cases immediately below and at the unchanged budget are in acceptance.feature. Warm startup is separately charged, distinguishing a broad startup exemption from the requested installation exemption.
- Addressed: landed refusal naming remains the baseline. Only the request-authorized positive `install_ms` addition is permitted on budget gate and timing lines; acceptance checks the original detail and warm refusal lines.
- Omitted: none of the requested behaviors. Superseded: the old attribution of cold installation to unexplained gaps and charged shaping time. Conflicting: no deliberate scope changes. No external release artifacts or provider access are prerequisites.

## Tests
- Exclude cold runtime preparation from charged time: add a focused scripted Shape regression reproducing the historical timings and allowing the acknowledgment round (regression)
- Report installation separately without double counting: assert exact shape_time fields, non-negativity and the elapsed accounting identity in cold and warm scripted runs (regression)
- Preserve conditional progress output: assert the complete cold and unchanged warm shape time lines (scenario)
- Preserve the budget boundary for model, tools and checks: scripted Shape runs immediately below and at the existing charged-time boundary, with installation excluded (regression)
- Preserve refusal text except positive installation attribution: compare budget refusal detail and gate/timing lines with and without an install (scenario)
- Preserve existing time accounting and budget behavior: update the existing exact event expectation for the new zero-valued field and keep useful evidence and budget tests passing (existing)

## Feasibility
- The final Go overlay harness compiled and ran TestShapeRuntimeInstallObservation on the unchanged tree; its warm refusal reached the actual Shape boundary, produced one shape_time event, and matched existing refusal detail and timing lines. Its account currently omits install_ms, which is a requested implementation change. The probe completed successfully without a provider or download.
- Documentation lookup retrieved untrusted upstream executor source at https://raw.githubusercontent.com/KogenAI/kogen-executor/main/executor/executor.go and runtime source at https://raw.githubusercontent.com/KogenAI/kogen-executor/main/executor/claudecode/runtime.go. These are API evidence only, not design authority or proof of the pinned dependency; the first search returned generic GitHub sources rather than a specific authoritative result. The repository's executor startup and successful local fixture establish the relevant integration point.
- evidence: {"id":"output-observer","kind":"observer","inputs":[".kogen/intents/shape-time-excludes-runtime-install/acceptance.feature","features/steps/shape-time-excludes-runtime-install__driver/qualify.js"],"observers":["features/steps/shape-time-excludes-runtime-install__steps.js"],"checks":[{"name":"good","command":"node features/steps/shape-time-excludes-runtime-install__driver/qualify.js good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-time-excludes-runtime-install__driver/qualify.js bad","exit":1,"output":"observer rejected\n"}]}
