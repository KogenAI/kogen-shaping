# Decide recommended answers without asking

Slug: k-shape-decide-not-ask-recommended
Model: .kogen/intents/k-shape-decide-not-ask-recommended/model.qnt invariants=NoStale,RecommendedIsAssumed seed=17 samples=100 steps=20
Acceptance:
- .kogen/intents/k-shape-decide-not-ask-recommended/acceptance.feature
- features/steps/k-shape-decide-not-ask-recommended__steps.js
- features/steps/k-shape-decide-not-ask-recommended__harness.js

Make shaping converge by deciding questions that can carry a recommended answer and recording the selected answer as an explicit assumption in the Intent instead of asking the caller. Enforce the behavior at the submission/session boundary as well as in the shaper policy: a scripted runner submitting such a question must not make it reach the caller. Continue shaping toward a checked Intent, preserving the assumption in its Decisions. A recommendation is not a user answer; do not mislabel it as one. Use the existing assumption marker and decision recording conventions rather than inventing a new provenance format.

Allow a material product question with no recommended answer to reach the caller. Replace the requirement that every question have exactly one recommendation with optional recommendation; keep unrelated question validation and technical-question restrictions. Mandatory authentication/security handoffs still require the actual human action; a recommended option is not evidence that the action happened.

Remove stale `.kogen/local/shaping/<slug>/questions.json` on failed-session termination or before starting the next fresh session, so the next runner sees no questions carried over from failure. Preserve valid pending questions and answer/resume behavior for a successful questions session. Make the decision check accept decisions supported by recorded assumptions or answers, instead of treating absence from the initial request alone as a refusal. Update both the decision-check instructions/context and mechanical outcome handling as needed. Keep genuine request-conflict checks and other readiness checks; do not solve convergence by increasing the three-submission package budget or weakening package gates.

The reported source is Studio kogen issue `5f0b8ad9-cb24-4626-b7c0-1aa5a15dd5bb`, “Shaping asks decidable questions every session; stale questions.json; answer-blind decision check”. Its reproducing lab `lab-shaping-converge-20261107 r8` reports eight failing rows, 25 questions including 11 follow-ups, and zero of eight ready; every session asks one question. The supplied diagnostic locations were `shaper.md:21`, `shape.go:2182`, `shape.go:799`, `decision.go:18` and `decision.go:195`. These are historical diagnostics, not performance thresholds or new retry requirements. The acceptance uses deterministic local fake-provider sessions rather than replaying an unavailable lab dataset.

## Decisions
- Decide recommended product answers as recorded assumptions without asking; leave unrecommended material product questions askable. (source: request)
- Failed sessions must not carry stale questions into the next session. (source: request)
- Accept answer-sourced and assumption-marked decision lines without requiring their values to appear in the initial request. (source: request)
- Keep the complete Go suite green. (source: request)

## Source comparison
- Addressed: request acceptance 1 and the reported forced recommendation (`shaper.md:21`, `shape.go:2182`) are covered by the recommended-answer session, publication, assumption, zero-question and no-pending assertions in acceptance.feature, and the first two paragraphs above. The existing forced recommended-option requirement is superseded, not preserved.
- Addressed: request acceptance 2 and stale `questions.json` (`shape.go:799`) are covered by the failed-session fixture and next-session artifact observation in acceptance.feature and the lifecycle paragraph above.
- Addressed: request acceptance 3 and answer-blind checks (`decision.go:18`, `decision.go:195`) are covered by both source cases in acceptance.feature and the decision-check contract above.
- Addressed: request acceptance 4 is the unrecommended product-question scenario. Acceptance 5 remains mandatory through the configured full Go test check, not a duplicate Cucumber scenario: an actual unchanged-tree go test ./... passed but took 230.9 seconds, exceeding the acceptance runner's 120-second cap. This changes placement of verification, not required product scope. The named issue, lab metrics and source locations are retained above as historical context, not claimed measurements of this candidate.
- Omitted from executable replay: the unavailable eight-row lab dataset and Studio service are not repository fixtures; the requested repository-verifiable fake sessions reproduce the specified behavior instead. No requested product behavior is omitted. The observation that package gates exhaust three submissions does not authorize changing that budget.

## Tests
- Recommended questions are decided without caller interaction and recorded as assumptions in a ready Intent: add fake-provider Shape session regression and update changed question-policy tests and deliberate policy goldens (regression)
- Questions without recommendations remain askable: add fake-provider Shape session with an unmarked two-option product question and update option-validation tests (scenario)
- Failed sessions cannot leak stale questions into the next session: add failed-session file lifecycle regression observing the artifact at the next provider request (integration)
- Answer-sourced and assumption-marked lines pass decision checking even when absent from the initial request: add focused outcome tests and update decision-check context tests to cover the recorded source (unit)
- Existing repository behavior outside these changes remains valid: run go test ./... and configured checks; regenerate synthetic policy goldens only by the documented update command for deliberate policy changes (existing)

## Feasibility
- The local Quint model typechecked and its two invariants ran for 20 samples of 10 steps with exit 0. An earlier model used the reserved built-in name fail; it was corrected to abortSession. An attempted scratch lookup of privately retained proposals found no files; it provides no observer evidence. No external dependency facts or provider research are required.
- go test ./... passed on the unchanged tree (probe: exit 0, 230.9 s); this is baseline suite evidence, not evidence of the requested fixes. The first acceptance attempt timed out during the duplicate full-suite scenario; that scenario was removed in favor of the mandatory configured full-suite gate.
- evidence: {"id":"observations","kind":"observer","inputs":["features/steps/k-shape-decide-not-ask-recommended__harness.js","features/steps/k-shape-decide-not-ask-recommended__steps.js"],"observers":["features/steps/k-shape-decide-not-ask-recommended__steps.js"],"checks":[{"name":"good","command":"node features/steps/k-shape-decide-not-ask-recommended__steps.js good","exit":0,"output":"observer accepted good\n"},{"name":"bad","command":"node features/steps/k-shape-decide-not-ask-recommended__steps.js bad","exit":1,"output":"observer rejected bad\n"}]}
