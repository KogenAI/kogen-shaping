# Overlap readiness with checks and reuse decisions

Slug: shape-kogen-checks-consume-shape-budget
Model: not applicable — the change is wall-clock scheduling and memoization inside one function chain; no stateful domain rules need a model
Acceptance:
- .kogen/intents/shape-kogen-checks-consume-shape-budget/acceptance.feature
- features/steps/shape-kogen-checks-consume-shape-budget__overlay.js

This Intent makes Kogen's per-submission checks in the `shape` package take less wall-clock time without changing which drafts pass, which are refused, any refusal text, the order of held refusals, the retry accounting or the recorded events (`submission_refused` and the rest).

Today `cheapGates`, then `applyAssumptionMarks`, then `certify` (which calls `sh.rd.Check`), then the contract review run strictly one after another; the Build reschedules this chain.

1. The readiness observation of a submitted package starts together with the static checks and the decision, literal and example checks, instead of after them. Its result is used exactly as today. If the cheap gates refuse, a readiness refusal is added after theirs. If the package's identical copy was already refused by the contract review, the readiness result is not reported. If the cheap gates pass, the contract review follows. If applying assumption marks changes the package after the overlapped observation started, that observation is discarded and readiness runs again on the marked package, as today. Events, progress lines and held refusals from the overlapped work are recorded after the work joins, in today's fixed order (static, decision, literal, example, then readiness), never in completion order. The static checks and the readiness observation keep their own separate working copies.
2. Within one Shape run, the decision check reuses its earlier model answer when its whole input is byte-identical to an earlier submission's: the request, the earlier answers, the normalized candidate intent.md and the decision evidence. A reused answer makes no new model or executor call. Everything after the answer still runs per submission as today: repeated-statement notes ("unchanged since submission N", where N is the earlier submission; for example "unchanged since submission 1"), the refused-statement ledger, assumption marks and the refusal text. Any changed byte of that input calls the model as today.

No new configuration, option or environment variable is added. The budget limits, wording and the `kogen: shape time` line format are unchanged. The contract review, the kept-assertion repeat, the literal and example checks and the wrong-implementation controls are not changed.

Acceptance runs focused model-free Go tests of the shape package. They live as data in `.kogen/intents/shape-kogen-checks-consume-shape-budget/overlay_test.go.txt`; the step file copies that file into `shape/` for the run, runs one named test and removes it. The tests use the existing scripted-provider fixtures of the shape package tests. Scenario 1: static check 4 s, decision check 4 s, readiness acceptance step 4 s, one passing submission under 6.5 s (a sequential implementation takes at least 8 s). Scenario 2: the decision check refuses after 3 s, readiness refuses at once; the refusal lists decision first, then readiness. Scenario 3: a second submission that changes only a step file makes no second decision-checker call and gets the same refusal with the "unchanged since submission 1" note; a third submission changing one decision line calls the checker again and uses its new answer.

## Decisions
- Readiness is overlapped with both the static checks and the model checks, because the static checks and readiness already use separate working copies and this Go project has no shared database server (decision: D1, round 1) (constraint: separate working copies stay) (evidence: shape/shape.go; shape/checkorder.go) (assumption: no shared resource makes concurrent static and readiness runs unsafe in this project) (source: request)
- Decision-answer reuse is memoized per Shape run, keyed on the exact decision-check input (request, earlier answers, normalized candidate intent.md, decision evidence); repeated-statement notes, the refused-statement ledger and assumption marks are still computed per submission from the reused answer (decision: D2, round 1) (constraint: no new model or executor call on a key hit) (evidence: shape/decision.go) (source: request)
- Deferred controls: the correct implementation spans the scheduling of checkSubmission and the decision cache, so controls.json is deferred and the Build writes the implementation (decision: D3, round 1) (assumption: wrong implementations are caught by the three scenarios' Go tests) (source: request)

## Source comparison
- Readiness overlap, join-then-record order, discard on assumption-mark change, and held readiness refusal after cheap refusals: addressed by scenarios 1 and 2 and the test overlay.
- Decision-answer reuse for byte-identical input and fresh call on any changed byte: addressed by scenario 3.
- Unchanged: contract review, kept-assertion repeat, literal and example checks, wrong-implementation controls, budget wording and the `kogen: shape time` line.
- Measurement context from the request, kept as literals: lab directory `/home/lanes/work.noindex/aor-climb/hypotheses`; cells `h23-release-bbf56fdc-vs-891c91b3/cells/*__release-bbf56fdc*` and `h22-chatgpt-shape-claude-build-vs-baseline-891c91b3/cells/rails-ft-saas-billing__chatgpt-shape-claude-build`; runs `chatgpt-sol-terra` and `chatgpt-shape-claude-build`; evidence files `shaping-events.ndjson` with `shape_time` and `static_checks` events; progress lines `checking step code` and `checking readiness`; `executor-decision` rounds; stderr `decision check response` with `wall_ms=73969`. They explain the problem and are not behavior to implement.

## Tests
- Readiness overlaps the static and model checks: TestOverlayReadinessOverlapsCheapChecks, one passing submission under 6.5 s with static 4 s, decision 4 s and readiness 4 s (integration)
- Combined refusals keep today's order and text: TestOverlayCombinedRefusalsKeepOrder, decision refusal after 3 s then readiness refusal (integration)
- Decision answer reused for identical input only: TestOverlayDecisionAnswerReuse, one checker call across two identical-input submissions and a new call when a decision line changes (unit)

## Feasibility
- none
