# Accept soft-wrapped request literals in the check

Slug: shape-literal-soft-wrapped-request-literal
Model: not applicable — one stateless string comparison in literalOutcome, with no state or interacting rules
Acceptance:
- .kogen/intents/shape-literal-soft-wrapped-request-literal/acceptance.feature
- features/steps/shape-literal-soft-wrapped-request-literal__steps.js
- features/steps/shape-literal-soft-wrapped-request-literal__harness.js

The literal check refuses a draft that carries a required literal correctly when the request wrote that literal across a soft-wrapped line. Observed: the first submission was refused at the literal_check gate with `literal: 50 cards\nleft | source: request | source_quote: "50 cards\nleft" | status: changed | package_value: 50 cards left | why: Example countdown warning text; the line break in the request is a soft wrap`. The package carried `50 cards left`, exactly the text the request means. The request file wrapped the sentence in the middle of that phrase, the checker copied the line break into the literal, and `literalOutcome` (shape/literal.go) found no package file containing `50 cards<newline>left`, so it reported `changed`.

The literal as the line break splits it, written out with a real line break:

```
50 cards
left
```

After this change, in `literalOutcome`, a required literal that contains a line break which is a soft wrap of the source (a single newline, optionally followed or preceded by indentation or spaces, inside one paragraph; a blank line is a real paragraph break and is not covered) is also accepted as present when the package contains the same text with each such line break and its surrounding spaces replaced by one space. Only this single-space variant is tried. The literal keeps its original text in the event, and the `literal_check` event for a variant match records status `present`. Everything else stays byte for byte: a missing or changed literal without a line break, a literal split by a blank line and a literal whose package value differs in any other character are refused with the same messages as today. No new configuration, and no change to which drafts pass beyond soft-wrapped literals.

Acceptance compiles a harness test through a Go overlay at `shape/shape_literal_soft_wrapped_request_literal_acceptance_test.go` and runs it. The Build must not create that path nor use the identifier `ShapeLiteralSoftWrappedRequestLiteralAcceptance`.

## Decisions
- One narrow change in literalOutcome: keep byte-for-byte inPackage first, add a soft-wrap variant alternative for the literal itself, status present on match with original literal text kept. (constraint: no new configuration) (constraint: blank-line literals and literals without a newline get no variant) (evidence: shape/literal.go) (assumption: Literals with several single line breaks: each is replaced by one space (reading of 'each such line break')) (source: request)
- Acceptance uses an overlay-compiled harness test (path not existing in repo) rather than running a test the Build must invent. (decision: D2, round 1) (evidence: shape/literal_test.go; shape/literal_required_test.go) (assumption: Harness verified in a scratch copy only: red on the unchanged tree, green with the correct patch, red on the byte-for-byte wrong patch.) (source: request)
- Soft-wrapped means no blank line in the literal; each line break with surrounding spaces becomes one space; the variant applies to the literal only. (evidence: shape/literal.go) (assumption: A literal with several single line breaks has each replaced (reading of 'each such line break'.) (source: request)
- One narrow change in literalOutcome: keep byte-for-byte inPackage first, add a soft-wrap variant alternative for the literal itself, status present on match with original literal text kept. (decision: D1, round 1) (constraint: no new configuration) (constraint: blank-line literals and literals without a newline get no variant) (evidence: shape/literal.go) (assumption: Literals with several single line breaks: each is replaced by one space (reading of 'each such line break').) (source: request)
- Soft-wrapped means no blank line in the literal; each line break with surrounding spaces becomes one space; the variant applies to the literal only. (decision: D3, round 1) (evidence: shape/literal.go) (assumption: A literal with several single line breaks has each replaced (reading of 'each such line break').) (source: request)

## Source comparison
- Addressed: a literal `50 cards\nleft` with a package containing `50 cards left` is `present` (scenario 1, also with indentation after the line break; scenario 2 keeps the exact bytes present). The request writes the literal in code as `50 cards<newline>left`.
- Addressed: the same literal with a package containing `50 cards, left` is still `changed` (with a package value) or `missing` (without), with today's feedback lines (scenario 3).
- Addressed: a literal with a blank line inside it is not matched by the variant, and a literal without a line break is still byte for byte (scenarios 4 and 5).
- Addressed: the focused model-free Go test is required in `## Tests`; the acceptance harness is separate.
- Omitted by the request: no new configuration, no change to Refine, prompts or goldens.

## Tests
- A soft-wrapped literal is present when the package has the one-space text, with its original text in the event: add a model-free test in `shape/literal_test.go` that calls `literalOutcome` with `litItem("50 cards\nleft", "request", "50 cards\nleft", "required", "")` and a package containing `50 cards left` (unit)
- A package that changes another character is still refused: the same test with a package containing `50 cards, left` stays `changed` or `missing` as before (unit)
- A literal with a blank line is not matched by the variant: the same test with a literal containing a blank line and a package containing `50 cards left` is not present (unit)
- The rules end to end through `literalOutcome`: the acceptance scenarios of this Intent (scenario)
- Everything else in the literal check is unchanged: the existing tests in `shape/literal_test.go`, `shape/literal_required_test.go` and `shape/literal_provenance_test.go` (existing)

## Feasibility
- none
