# State all-fields-required in check item texts

Slug: shape-check-items-state-all-fields-required
Model: not applicable — the change is fixed wording in two instruction texts and in tool schema property descriptions; there is no state to model
Acceptance:
- .kogen/intents/shape-check-items-state-all-fields-required/acceptance.feature
- features/steps/shape-check-items-state-all-fields-required__steps.js
- features/steps/shape-check-items-state-all-fields-required__driver/harness_test.go.txt

The literal check model sometimes omits `status` from an item and the decision check model sometimes omits `why`, so the submission is rejected and the session pays a second model call. The literal check instructions (`literalInstructions` in `shape/literal.go`) and the decision check instructions (`decisionInstructions` in `shape/decision.go`) now state that every item carries every field, including `status` for a literal item and `why` for a decision item, and that a field with nothing to say is the empty string, never omitted. The literal tool schema (`literalSchema`) gets a `description` on its `status` and `why` properties and the decision tool schema (`decisionSchema`) gets a `description` on its `why` property saying the same.

This is wording only. The schemas' `required` lists, enums, `additionalProperties` and strictness, the parsers (`parseLiteralCheck`, `parseDecisionCheck`), which drafts pass, and the existing instruction sentences stay exactly as they are; in particular a secret literal item still leaves `literal`, `source_quote`, `package_value` and `why` as empty strings. No setting, flag or environment variable is added. Making malformed-item errors name the field is a separate request and is not done here.

Acceptance is the model-free Go test `TestSciafrAcceptanceHarness` in the `shape` package. Each acceptance step runs that named test with `go test -run` (it is supplied to the shape package through a Go overlay from the Intent's driver file, so it exists before the Build) and checks its result: the test must pass, and what it observed about the instruction texts, the tool schema descriptions and the required lists must match the feature. The Build also adds its own focused model-free Go test in the `shape` package with the same checks.

## Decisions
- Both instruction texts get one added sentence line after the item field list stating that every item carries every field, naming `status` (literal) or `why` (decision), and that a field with nothing to say is the empty string, never omitted. (evidence: shape/literal.go; shape/decision.go) (source: request)
- The literal schema's `status` and `why` properties and the decision schema's `why` property each get a `description` saying the property is present on every item and never omitted; the `why` descriptions also say that nothing to say is the empty string. (constraint: the required lists, enums, additionalProperties, strictness and parsers stay exactly as they are) (evidence: shape/literal.go; shape/decision.go) (source: request)
- The existing sentence that tells the literal model to leave `literal`, `source_quote`, `package_value` and `why` empty for secret material is kept unchanged, because an existing test requires that exact line. (evidence: shape/literal.go; shape/literal_test.go) (source: request)
- For the decision check's `request_quotes`, which is a list rather than a string, the instruction text says an empty list is the empty value. (evidence: shape/decision.go) (assumption: the request names the empty string, but request_quotes is an array in the schema, so an empty list is the only valid empty value) (source: request)
- Acceptance steps run the named Go test `TestSciafrAcceptanceHarness` in the shape package and check its result, not git diff or untracked-file listings; the Intent supplies that test through a Go overlay so it runs on the unchanged tree, the pattern an existing Intent's steps use. (evidence: features/steps/approve-static-reuse-line-explains-unknown__steps.js; features/steps/approve-static-reuse-line-explains-unknown__driver/harness_test.go.txt) (source: request)
- The Build adds a focused model-free Go test in the shape package, in a new test file using the package's existing `parallel(t)` helper. (evidence: shape/literal_test.go; shape/decision_test.go) (source: request)
- Both instruction texts get one added sentence line stating that every item carries every field, naming status (literal) or why (decision), and that a field with nothing to say is the empty string, never omitted. (decision: D1, round 1) (constraint: existing instruction sentences stay) (evidence: shape/literal.go; shape/decision.go) (source: request)
- Literal status and why properties and decision why property each get a description; required lists, enums, additionalProperties, strictness and parsers unchanged. (decision: D2, round 1) (constraint: required lists, enums, additionalProperties, strictness and parsers stay exactly as they are) (evidence: shape/literal.go; shape/decision.go) (source: request)
- The existing secret-material sentence in the literal instructions is kept unchanged. (decision: D3, round 1) (evidence: shape/literal.go; shape/literal_test.go) (source: request)
- For decision request_quotes (an array) the instruction names an empty list as the empty value. (decision: D4, round 1) (evidence: shape/decision.go) (assumption: The request names the empty string but request_quotes is an array in the schema.) (source: request)
- Acceptance observes texts and schemas through a Go harness copied in by a Go overlay, because the Build writes its own Go test and acceptance may not run an invented test. (decision: D5, round 1) (evidence: features/steps/approve-static-reuse-line-explains-unknown__steps.js; features/steps/approve-static-reuse-line-explains-unknown__driver/harness_test.go.txt) (source: request)
- The Build's own Go test lives in a new test file in the shape package using the existing parallel(t) helper. (decision: D6, round 1) (evidence: shape/literal_test.go; shape/decision_test.go) (source: request)

## Source comparison
- Request: both instructions state that every item carries every field, including status for a literal and why for a decision item, and that a field with nothing to say is the empty string, never omitted. Addressed by D1 and the instruction scenarios of acceptance.feature.
- Request: the literal and decision tool schemas get a description on the status and why properties. Addressed by D2 and the schema description scenarios of acceptance.feature.
- Request: wording only; required lists, enums and parsers unchanged; secret items still leave the named fields empty strings. Addressed by D2, D3 and the unchanged-schema scenario of acceptance.feature.
- Request: no change to which drafts pass and no new configuration. Honoured: no parser or setting is touched.
- Request: a focused model-free Go test in the shape package, run by acceptance. Addressed by D5, D6 and under Tests.
- Request: the malformed-item error naming the field is a separate request. Omitted on purpose.

## Tests
- The literal check instructions state that every item carries every field, including status, with an empty string and never omitted: a focused model-free Go test in the shape package checking `literalInstructions` (unit)
- The decision check instructions state the same, including why: a focused model-free Go test in the shape package checking `decisionInstructions` (unit)
- The literal status and why descriptions and the decision why description exist and say never omitted: a focused model-free Go test in the shape package reading `literalSchema` and `decisionSchema` (unit)
- The required lists, enums and the existing secret sentence are unchanged: the same Go test compares the required lists and enums, and the existing `TestLiteralInstructions` and parser tests keep passing (existing)
- The texts and schemas observed through a real Go test run: the acceptance feature runs the named test `TestSciafrAcceptanceHarness` and checks its result (scenario)

## Feasibility
- none
