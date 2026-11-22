# Choose the invented-literal fix by literal kind

Slug: shape-invented-fix-names-kind-remedy
Model: not applicable — the change is a pure function from a literal to a sentence; it has no state
Acceptance:
- .kogen/intents/shape-invented-fix-names-kind-remedy/acceptance.feature
- features/steps/shape-invented-fix-names-kind-remedy__steps.js

When the contract review refuses an invented literal, the fix line that follows the refusal line always says to find the form or link by its visible text. For a literal that is a test file path, or the module name or input key of code the request asks to add, that remedy cannot apply. `auditInventedFix` in `shape/audit.go` now chooses its remedy by the kind of the literal.

Kinds, tested in this order:
1. Path kind: a literal with no space that contains a slash and ends in a dot and an extension. Its remedy says to list the directory at run time, drive the file found there, and not assert the file name.
2. Code-name kind: a literal that starts with `alias ` or is one token of letters, digits, underscores and dots with no space and no slash. Its remedy says to record the name in a Decision of intent.md as the Shaper choice for the new code, so the approval card lists it, and to assert only the behaviour.
3. Every other literal keeps today's sentence exactly.

Only the text of the fix line changes. The refusal line, which literals are refused, and which drafts pass stay exactly as they are. No new configuration. Out of scope: `shape/prompts`, docs, README, the rule for which literals are invented and the dropping of repeated literals. Focused model-free Go tests are added next to the existing tests in `shape/audit_test.go`.

## Decisions
- The fix line of a refused invented literal is chosen by the kind of the literal: path kind first, then code-name kind, then the default sentence unchanged. (decision: D1, round 1) (evidence: shape/audit.go) (source: request)
- Path kind is a literal with no space that contains a slash and ends in a dot and an extension; its remedy tells the shaper to list the directory at run time, drive the file found there and not assert the file name. (decision: D2, round 1) (evidence: shape/audit.go) (source: request)
- Code-name kind is a literal that starts with `alias ` or is one token of letters, digits, underscores and dots with no space and no slash; its remedy tells the shaper to record the name in a Decision of intent.md as the Shaper choice for the new code, so the approval card lists it, and to assert only the behaviour. (evidence: shape/audit.go) (source: request)
- The refusal line, the set of refused literals and the passing drafts are unchanged; no setting, flag or environment variable is added; only shape/audit.go and shape/audit_test.go change. (evidence: shape/audit.go) (source: request)
- Code-name kind is a literal that starts with `alias ` or is one token of letters, digits, underscores and dots with no space and no slash; its remedy records the name in a Decision of intent.md as the Shaper choice, listed on the approval card, and asserts only the behaviour. (decision: D3, round 1) (evidence: shape/audit.go) (source: request)
- The refusal line, the set of refused literals and the passing drafts are unchanged; no setting is added; only shape/audit.go and shape/audit_test.go change. (decision: D4, round 1) (evidence: shape/audit.go) (source: request)

## Source comparison
- addressed: request "Path kind" and "Code-name kind" remedies; contract: acceptance.feature scenarios for `test/acceptance/external-stall-outcome.test.cjs`, `alias Studio.Loop.Fairness` and `hard_quota`.
- addressed: request "Every other literal keeps today sentence exactly"; contract: acceptance.feature scenario with the table-columns literal and the byte-for-byte sentence.
- addressed: request "The refusal line ... stay exactly as they are"; contract: acceptance.feature outline over the four literals.
- addressed: request "Add focused model-free Go tests"; contract: Tests line below, prose above.
- omitted: shape/prompts, docs, README, the rule for which literals are invented and the dropping of repeated literals; the request puts them out of scope.

## Tests
- Path literal gets the list-the-directory remedy without the words visible text: add a Go test of auditInventedFix in shape/audit_test.go (unit)
- Code names (`alias ` prefix and a single token such as hard_quota) get the Decision and approval card remedy without the words visible text: add Go tests of auditInventedFix in shape/audit_test.go (unit)
- Any other literal keeps today's sentence byte for byte, and the refusal line of each literal is unchanged: add Go tests of auditInventedFix and the refusal line in shape/audit_test.go (unit)
- The fix line and the refusal line observed through the shape package for the four literals: acceptance.feature (scenario)

## Feasibility
- none
