# Bound native shaping conversations

Slug: shape-bounds-tool-free-answers
Model: .kogen/intents/shape-bounds-tool-free-answers/model.qnt invariants=Bounded,StopAtBoundary seed=1 samples=100 steps=305
Acceptance:
- .kogen/intents/shape-bounds-tool-free-answers/acceptance.feature
- features/steps/shape-bounds-tool-free-answers__steps.js
- features/steps/shape-bounds-tool-free-answers__driver/run.js
- features/steps/shape-bounds-tool-free-answers__driver/cases.go
- features/steps/shape-bounds-tool-free-answers__driver/qualify.js

End native shaping after noToolLimit tool-free answers since the last submission, regardless of intervening preparation tools. A submission resets that count for every outcome that resets sinceSubmit today, including refused questions, invalid submissions and readiness observations. Retain the existing noToolStop refusal text, exit code and events, even its legacy consecutive-times wording. Keep cumulative audit totals distinct from this resettable counter. The existing three-consecutive-answer bound remains effective.

Also bound each native shaping conversation to 300 provider requests: do not send request 301. At request 300, return the noToolStop outcome kind, naming the provider request cap and last tool-free answer if one exists. A shaping conversation ends when it reaches 300 provider requests, with the noToolStop outcome kind; there is no terminal-submission exception. Add integer request_cap and requests fields to the existing shaping_turn_counts event for native Shape, retaining existing counters and a single final event. Request count is conversation-wide and survives working-window resets and submissions.

Put the counting rule and cap in a new small shape/toolfree.go; limit shape/shape.go edits to loop call sites. Do not change Refine, external shaping, provider routing, authentication or the invalid-submission budget. Add failing regression tests first, using fake local shapers, then implement, run affected tests and run go test ./shape. The Go acceptance fixture is injected into a disposable checkout and calls the real Shape entry point using the existing scripted provider; no live provider is required. (evidence: shape/force_test.go; shape/force_wires_test.go; README.md)

## Decisions
- Preserve noToolLimit and the existing refusal wording while applying its bound to tool-free answers since submission, not just consecutive answers. (source: request)
- Keep the new policy isolated in shape/toolfree.go and the native loop wiring minimal. (source: request)
- Use request_cap and requests as the additional shaping_turn_counts field names. (assumption: these non-secret event key names settle the unspecified naming) (source: request)

## Source comparison
- Addressed: Request Change 1 and all four regression bullets are covered by the counting contract above, the model and alternating/reset/cap/consecutive acceptance scenarios. Existing sinceSubmit reset outcome kinds are preserved rather than narrowing reset to valid packages; see shape/shape.go and the reset scenario.
- Addressed: Request Change 2 is covered by the request cap paragraph, cap scenario and bounded model; request_cap and requests settle the otherwise unspecified event key names. Existing noToolStop outcome, refusal exit and events remain required.
- Addressed: Request file boundaries and tests-first sequence are required above: shape/toolfree.go, minimal shape/shape.go loop call sites and go test ./shape. These are implementation requirements, not authorization to copy the bounded control overlay into production.
- Addressed as historical context: 10 November 2026, kogen v0.1.0-dev.3c70abcb, chatgpt, k-docs-concise-task-first (2 h 37 min, 1,110 provider requests, 550 tool_free_answer and 549 forced_submit_turn events) and studio-redesign-preservation (878 requests) motivate the local regression. The repeated answer about mandatory control deletions is diagnostic history, not a required new refusal literal. The executable alternating scenario reproduces its causal tool/prose pattern without replaying paid requests.
- Addressed as execution context, not product scope: OpenCode Go provider/unavailable after 3 attempts without a first byte in 45 s at 21:02Z; SuperGrok grok-4.7 on kogen-runner provider/login rejection at 21:28Z; subsequent duplicated retry bases request chatgpt after its shift cap. These provider retries leave the feature unchanged and require no provider or credential edits.
- Omitted from production scope: unrelated routes are intentionally panic-stubbed only in bounded controls.json overlays to fit the control byte budget. These overlays prove the native acceptance distinctions only; implementation must preserve all unrelated production behavior and pass the full shape package tests. control-generator.py records their construction and is not an implementation recipe.

## Tests
- Bound prose answers across preparation calls: add the alternating fake-shaper regression through Shape (regression)
- Reset after every existing sinceSubmit-reset submission outcome: add invalid-package reset coverage and preserve existing refusal/readiness coverage (regression)
- Cap native provider requests and report cap plus retained answer: add read-only and one-prose-then-read request-cap cases with final event assertions (regression)
- Preserve consecutive prose refusal: retain existing coverage and exercise the third-answer boundary through Shape (regression)
- Preserve surrounding shaping behavior: run go test ./shape after the targeted tests (existing)

## Feasibility
- The unchanged shape package passed go test ./shape; local Go is go1.27.2. The supplied model typechecked. Bounded correct controls exercised the new cases successfully; this is not a delivered implementation. (evidence: ran `go test ./shape`)
- evidence: {"id":"cap-observer","kind":"observer","inputs":[".kogen/intents/shape-bounds-tool-free-answers/controls.json","features/steps/shape-bounds-tool-free-answers__driver/qualify.js"],"observers":["features/steps/shape-bounds-tool-free-answers__driver/run.js","features/steps/shape-bounds-tool-free-answers__driver/cases.go"],"checks":[{"name":"good","command":"node features/steps/shape-bounds-tool-free-answers__driver/qualify.js good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-bounds-tool-free-answers__driver/qualify.js bad","exit":1,"output":"observer rejected\n"}]}
