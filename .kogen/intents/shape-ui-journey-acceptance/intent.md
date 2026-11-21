# Shape usable journeys for new UI surfaces

Slug: shape-ui-journey-acceptance
Model: not applicable — fixed Shaper policy text, no state machine
Acceptance:
- .kogen/intents/shape-ui-journey-acceptance/acceptance.feature
- features/steps/shape-ui-journey-acceptance__steps.js

Kogen Shape's policy gains one bounded rule: when a request adds or changes a user-facing surface and reachability or visible behavior matters, the acceptance it shapes contains a representative executable journey. The journey starts from an existing entry point of the product, as the actual actor the request names, and goes through to the requested visible result. A direct URL visit or a self-menu-only path does not satisfy it, and neither does a count or a placeholder in place of the exact entry the fixtures define. Existing conventions supply the default without needless questions; only an unresolved choice that materially changes the UX is asked before readiness. Non-UI work and changes where the requirement is irrelevant get no browser-test mandate. Both the instructions Shape sends to the Shaper and the instructions of its contract review carry this rule. Out of scope: ElixirDrops, its queue, and Studio's delivery safeguard (split and incomplete-slice approval). The Shaper's instructions must contain these exact sentences, each on one line: "The representative executable journey starts from an existing entry point of the product, as the actual actor the request names, and goes through to the requested visible result.", "A direct URL visit or a self-menu-only path does not satisfy the journey", "an exact entry the fixtures define must be visible", "A count, a placeholder or a heading alone does not show the visible result.", "use the existing browser or end-to-end tooling", "Follow an existing convention for the default and ask no question it settles.", "Ask the caller only when an unresolved choice materially changes the UX", "Non-UI work and changes where reachability or visible behavior is irrelevant get no browser-test mandate." The contract review instructions must contain the direct-URL sentence.

## Decisions
- The journey rule is policy text for the Shaper and the contract review, observed through the instructions Shape actually sends to the model at its entry point; no new setting or flag is added. (evidence: shape/policy.go; shape/contract.go; shape/prompts/shaper.md) (assumption: confirm that prompt policy, not a new deterministic readiness check, is the intended bounded improvement) (source: request)
- The default comes from existing conventions and questions are asked only for an unresolved choice that materially changes the UX. (source: request)
- Because the Shaper's prompt bytes change, the recorded policy goldens are regenerated deliberately as the README describes. (evidence: README.md) (assumption: confirm that regenerating the policy goldens with the README's update command is acceptable for this deliberate policy change) (source: request)

## Source comparison
- Request items 1 and 2 (guest on an existing Drop following another contributor's attribution to the profile; known public comment visible on the Comments tab as that exact entry; direct-URL, self-menu-only, count-only and placeholder variants fail) are carried as the policy sentences about the existing entry point, the actual actor, the exact visible entry and the rejected shortcuts, with a wrong-implementation control per shortcut. Addressed.
- Request item 3 (conventions give the default; unresolved material UX choice is surfaced before readiness; no generic browser-test mandate for non-UI work) is carried by the last three policy sentences. Addressed.
- ElixirDrops, its queue and Studio delivery safeguards: omitted as the request excludes them.

## Tests
- A UI-surface request yields a journey from an existing entry point as the actual actor, rejecting direct URL and self-menu-only variants: scenario through Shape's entry point reading the Shaper and contract review instructions (scenario)
- A known public entry must be visible; a count or placeholder fails: same scenario harness (scenario)
- Conventions settle defaults, material UX choices are asked, non-UI work gets no mandate: same scenario harness (scenario)

## Feasibility
- none
