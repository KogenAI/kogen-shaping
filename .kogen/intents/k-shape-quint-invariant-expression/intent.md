# Clarify Quint invariant expression guidance

Slug: k-shape-quint-invariant-expression
Model: not applicable — this wording change has no meaningful state
Acceptance:
- .kogen/intents/k-shape-quint-invariant-expression/acceptance.feature
- features/steps/k-shape-quint-invariant-expression__text.js

Clarify the shaper instructions (Shape.instructions) and matching shaping prose with one concise example: multiple invariants use one Boolean conjunction expression, not a comma list. Change only this wording and affected prompt goldens; add no stage, model, gate, timeout, or runtime behavior. Preserve the comma-separated Intent metadata format. The full test suite remains a final configured repository check, not a timed acceptance step, and must pass without reducing the gate.

## Decisions
- Update the embedded shaping prose, which feeds ShaperInstructions, and regenerate affected policy goldens deliberately rather than hand-merging hashes. (evidence: shape/prompts/shaper.md; shape/context.go; shape/policy_test.go; shape/policy_golden_test.go) (source: request)
- Keep the configured full repository checks unchanged and inspect the changed text and matching prompt snapshot directly in acceptance. (evidence: .kogen/project.yaml) (source: request)

## Source comparison
- Addressed: KOGEN-025, Studio issue e9521c19-5b78-4939-b97d-1f813f1d1766 requests explicit conjunction guidance and comma-list rejection; the prose boundary above and both acceptance scenarios cover it. The request's Shape.instructions surface is provided here by ShaperInstructions and its embedded prose. (evidence: shape/context.go)
- Addressed: the supplied invalid and valid flag examples are preserved in acceptance. ElixirDrops' four failed initial shapes and correction about 190 seconds later motivate the copy fix; no log processing or timing behavior is requested.
- Addressed: acceptance recovery clarification of 8 Nov keeps the full suite as the existing final configured check, not an acceptance subprocess. Affected prompt golden tests must pass, including TestShaperPolicyGolden and TestPromptsAreByteEqualToTheElixirFunctions; regenerate synthetic goldens using the documented update workflow, inspect the diff, and run normal checks. (evidence: shape/policy_test.go; shape/policy_golden_test.go)
- Superseded: the current indirect comma distinction is replaced by explicit invalid-comma wording. No runtime grammar change, new stage, model or gate is included.

## Tests
- Explicit conjunction guidance and comma rejection: inspect the embedded shaper prose with the supplied flag examples (regression)
- Matching model-facing prompt snapshot: inspect the same wording in the policy snapshot and update affected prompt goldens (regression)
- Full suite remains passing: run unchanged configured repository checks after the deliberate golden update (existing)

## Feasibility
- none
