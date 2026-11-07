# Restore decisions during refused draft revision

Slug: sh-revision-drops-carried-decisions
Model: not applicable — this repair is a text transformation, not a new workflow state machine
Acceptance:
- .kogen/intents/sh-revision-drops-carried-decisions/acceptance.feature
- features/steps/sh-revision-drops-carried-decisions__steps.js

Repair carried-decision omissions during revision of a refused draft. Consume the exact keep-lines named by the refusal, restore them in Decisions, and preserve the decisions and unrelated content already present. Use the repair in Shape's actual refused-draft revision path before the next candidate is checked; do not weaken the carried-decisions gate, bypass verification, or publish a refused candidate. A resolvable omission must not exhaust retries merely because the revision omitted the same supplied lines again.

The final request note, “Retry 3: probe statements dropped, replay requirement removed”, narrows this delivery to carried decisions. Probe/test-result statement cleanup and provider replay are not required by this Intent. This does not authorize inventing request quotes or relaxing any existing gate.

## Decisions
- A revision keeps every Decision the draft still depends on and restores the exact lines the refusal names. (source: request)
- Keep gate enforcement and use its supplied keep-lines as repair inputs rather than guessing historical decision text. (evidence: shape/ledger.go) (source: request)
- Provide the text transformation as the package-internal callable interface repairCarriedDecisionDraft(draft string, problems []string) string, and wire it into refused-draft revision; this is not a new public API or CLI. (evidence: shape/ledger.go; shape/shape.go) (source: request)
- Treat the final Retry 3 note as superseding probe-statement cleanup and replay requirements, while retaining direct automated refused-draft revision tests. (assumption: the final note removes these requirements from this delivery rather than describing an already corrected draft) (source: request)

## Source comparison
- Addressed: the request's 7 Nov 16:19Z docs incident, k-docs-concise-task-first, on the kogen seat/MacBook, stopped after 756 s with intent/not_ready and carried_decisions because D9–D16 from round 1 were omitted despite exact keep-lines being available. The acceptance fixture uses those IDs and round with explicitly synthetic text, not invented historical originals.
- Addressed: “a revision keeps every decision the draft still depends on” and “acts on exactly what the refusal names” are covered by exact restoration, preservation of an already-present decision and unrelated text, and idempotence scenarios.
- Superseded: the earlier probe/timing/unsupported-statement cleanup and question_check scope is removed by the final Retry 3 note. The reported cross-host counts (decision_check 5, literal_check 3, contract_review 2, carried_decisions 1, step_ownership 1, readiness 1) and 16 other refusals motivate the request but are not a new requirement to repair every gate.
- Addressed: direct automated tests feed refused drafts and real carried_decisions-format problem lines through the revision transformation. Replay is not required. Historical evidence locations kogen-macbook-runner ~/work.noindex/shape-k-docs-concise-task-first.out lines 16–40 and build-runtime/attempts/k-docs-concise-task-first-2c574494d4ad46d1bd7664f1ace5e1d1.json (exit 5) are provenance, not files acceptance assumes are available.
- Omitted as optional: github.com/KogenAI/kogen-shaping commit 5a8feee on shaper-repair is permitted reuse, not a dependency or proof of correctness.

## Tests
- Restore all exact keep-lines named by a carried-decision refusal: direct callable regression with the reported IDs and round (regression)
- Preserve decisions and unrelated draft content while repairing only named omissions: exact whole-draft comparison (unit)
- Do not duplicate restored lines on another revision: apply the repair twice and compare complete outputs (unit)
- Wire the repair into refused-draft revision without weakening checks: add an affected Shape integration regression using existing local scripted fixtures (integration)

## Feasibility

The following is the required declaration of acceptance-observer controls, not a product Decision, request-derived concrete example, or claim about a probe's observed outcome. Its commands and expected outputs specify control execution; readiness must run them. This declaration follows the repository's shaping-evidence format. (evidence: shape/prompts/feasibility-guidance.md)

- evidence: {"id":"revision-observer","kind":"observer","inputs":["features/steps/sh-revision-drops-carried-decisions__steps.js"],"observers":["features/steps/sh-revision-drops-carried-decisions__steps.js"],"checks":[{"name":"good","command":"node features/steps/sh-revision-drops-carried-decisions__steps.js good","exit":0,"output":"accepted\n"},{"name":"bad","command":"node features/steps/sh-revision-drops-carried-decisions__steps.js bad","exit":1,"output":"rejected\n"}]}
