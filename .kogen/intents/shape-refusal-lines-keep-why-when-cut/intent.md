# Preserve why in truncated Shape refusal lines

Slug: shape-refusal-lines-keep-why-when-cut
Model: not applicable — this is a stateless bounded string transformation
Acceptance:
- .kogen/intents/shape-refusal-lines-keep-why-when-cut/acceptance.feature
- features/steps/shape-refusal-lines-keep-why-when-cut__steps.js
- features/steps/shape-refusal-lines-keep-why-when-cut__driver/probe_test.go

Change only the refusal-record transformation in `cutProblems` in `shape/checkorder.go`, with focused model-free Go regression coverage next to `shape/checkorder_test.go`. Replace the current first-300-runes truncation only for overlong lines containing ` | why: `. Keep the beginning of the line, then exactly ` ... `, then the why segment starting with ` | why: `. Bound that entire why segment, including its delimiter, to its first 150 runes. Give the beginning the remaining rune budget within 300; it must retain at least 60 runes. Preserve Unicode rune boundaries. Leave lines of 300 runes or fewer byte-for-byte unchanged, and leave overlong lines without the delimiter cut to their first 300 runes exactly as before. Continue returning only the first 20 problem lines in their original order.

Only the recorded problem lines and the refusal summary built from them change. Do not change gate verdicts, which drafts pass or are refused, or any text fed back to the shaper. Keep the contract_review and decision_check producers unchanged. Do not add configuration or raise any limit. No provider, prompt, policy-golden, UI, or other workflow changes belong to this fix.

## Decisions
- Keep the implementation confined to cutProblems, retaining its current callers for recorded refusals and summary inputs; do not truncate the original model feedback text. (evidence: shape/checkorder.go; shape/shape.go; shape/checkorder_test.go) (source: request)
- Count both limits in runes; the 150-rune why-segment budget includes ` | why: ` and the 300-rune result budget includes ` ... `. The remaining head budget is at least 145 runes and therefore satisfies the requested minimum of 60. (source: request)
- Use focused Go tests of the actual package-private function, not a CLI or live model scenario, and do not introduce a state model. The acceptance bridge is supplied now and calls the real function through a temporary Go test overlay; Build must also add ordinary focused regression coverage beside the existing checkorder tests. (evidence: shape/checkorder.go; shape/checkorder_test.go; .kogen/project.yaml) (source: request)

## Source comparison
- Addressed: Request Expected and Constraints map to the transformation and exclusions above and to every case in acceptance.feature: the 300-rune threshold, the literal delimiters, the capped 150-rune why segment, the retained head, no-why compatibility, Unicode counting, unchanged short lines and the ordered 20-line limit. Existing prefix-only truncation for overlong why lines is superseded, not retained as a compatibility mode.
- Addressed: Request Scope note maps to one narrow production edit in shape/checkorder.go and focused Go regression tests next to shape/checkorder_test.go. Acceptance directly exercises cutProblems; controls.json supplies a bounded correct patch and one wrong patch restoring the current tail cut. There is no model, new option, raised limit or unrelated cleanup.
- Addressed as reported historical context, not independently reproduced: Request AoR counts, glob, attempt/gate, cell identifier, supplied truncated contract fragment and stdout.txt suffix are preserved in acceptance.feature. Its first executable case uses that fragment in a 600-rune synthetic completion. The historical lost why is unknown; the test explicitly labels its reason synthetic rather than inventing historical evidence.
- Addressed: Request source paths shape/contract.go and shape/decision.go identify unchanged producers, not additional edit targets. Their formats already put ` | why: ` after expectation/contract or decision detail. (evidence: shape/contract.go; shape/decision.go)
- Omitted: no requested behavior. The acceptance uses the scoped function boundary rather than new workflow integration fixtures, consistent with the explicit narrow scope. Existing feedback coverage remains useful and must not be weakened. (evidence: shape/checkorder_test.go)

## Tests
- Preserve the reason on overlong refusal lines: add focused Go regression coverage next to shape/checkorder_test.go for the supplied synthetic contract case and decision-shaped input, checking exact bounded output through cutProblems (regression)
- Cap the why segment while preserving the head and rune limit: cover adjacent 149, 150 and 151-rune why segments and non-ASCII input through cutProblems (unit)
- Preserve the existing threshold and no-why behavior: cover adjacent 299, 300 and 301-rune lines plus an overlong line without the delimiter through cutProblems (unit)
- Preserve the ordered first-20-lines limit: cover adjacent 19, 20 and 21-line batches through cutProblems (unit)
- Preserve the uncut feedback delivered to the shaper and the existing refusal decisions: retain TestDecisionAndLiteralRefusalsAreBothReported and other existing refusal tests unchanged, and keep the production edit inside cutProblems (existing)

## Feasibility
- The local Go 1.27.2 and Node v24.21.0 tools were observed. A focused scratch compile and execution of TestRefusalCutAcceptanceBridge with an unchanged short input returned exit 0, `PASS` and a JSON observation containing the original line. This qualifies the bridge's named-test route, not the unimplemented transformation.
- evidence: {"id":"bounded-line-observer","kind":"observer","inputs":[".kogen/intents/shape-refusal-lines-keep-why-when-cut/acceptance.feature"],"observers":["features/steps/shape-refusal-lines-keep-why-when-cut__steps.js"],"checks":[{"name":"good","command":"node features/steps/shape-refusal-lines-keep-why-when-cut__steps.js good","exit":0,"output":"observer accepted fixture\n"},{"name":"bad","command":"node features/steps/shape-refusal-lines-keep-why-when-cut__steps.js bad","exit":1,"output":"observer rejected corruptions\n"}]}
