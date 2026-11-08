# Normalise shape example comparisons

Slug: shape-example-check-normalises-format
Model: not applicable — format equivalence is a stateless comparison; existing submission lifecycle remains unchanged
Acceptance:
- .kogen/intents/shape-example-check-normalises-format/acceptance.feature
- features/steps/shape-example-check-normalises-format__steps.js
- features/steps/shape-example-check-normalises-format__harness.js

Make the requested example-check behaviour available on this tree, which currently deliberately omits the old example check. Restore the requested per-Examples-row classification and refusal for real contradictions, not raw-string differences. Do not treat absence of a rule-derived value as a contradiction. Preserve other readiness and approval checks. This deliberately supersedes the no-example-check expectation in shape/examples_revert_test.go; update that test rather than retaining mutually exclusive requirements.

The local synthetic-provider acceptance protocol uses submit_example_check with an items array. Each item has scenario, expected, derived, verdict and why strings. A missing rule-derived value is represented by an empty derived string. The example_check event records items with their final verdict, and submission_refused with gate example_check records actual contradiction refusal. This fixes the testable boundary without requiring live credentials. Normalisation must happen in code, not only in model instructions: a scripted model may call format-equivalent values contradicts.

Normalise braced JSON, unbraced JSON member lists and the supplied plain key:value spelling for comparison. Ignore format-only braces, member quoting and insignificant whitespace, and compare object members by value rather than textual spelling/order. Preserve meaningful string contents and array order; do not strip punctuation indiscriminately. Expected `"status":0,"out":"61610a0a"` and derived `{"status":0,"out":"61610a0a"}` or `status:0, out:61610a0a` must follow. Changing status to 1 must contradict and refuse. Missing derived output must not contradict or cause an example-check refusal.

## Decisions
- Values equal after normalising format are follows; only real value differences are contradicts. (source: request)
- An example whose value the rules do not give is not a contradiction. (source: request)
- (assumption: the request does not require unsupported values to block) A missing rule-derived value is unsupported and does not itself refuse submission. (source: request)

## Source comparison
- Addressed: the original problem and Studio issue 002072a5-c189-4ae8-81ed-dbb2c0a3db9c are covered by the comparison rules above and the executable Examples below. Historical eu2 runs/shc2-eu2-r1/cells/framecrc-go/r1/g1 shaping-events.ndjson lines 24–25 and 30–31, intent/not_ready and eight hidden losses in the 65-pair screen are supplied motivation, not local fixtures or claims of reproduction of private records.
- Addressed: both supplied equivalent spellings, real-difference refusal, not-given non-contradiction and passing Go suite are retained in acceptance and Tests.
- Superseded: shape/examples_revert_test.go currently asserts no example check occurs; the requested follows/contradicts behaviour requires replacing that expectation. No other readiness gate is removed.
- Addressed: delivery constraints use the requested slug, owned prefixed bindings and paths, source-tagged Decisions and no private-record lookup. README, testing rules and configured checks remain in force; unrelated historical testing-policy examples are unchanged, not feature scope.

## Tests
- Format-only JSON/member-list and key:value differences follow: add focused comparison tests using the supplied values and acceptance through Shape with a scripted false contradiction (regression)
- Real differences contradict and refuse submission: add comparison tests and assert Shape records the example-check refusal (integration)
- Missing rule-derived values are not contradictions: add comparison tests and assert unsupported without example-check refusal (unit)
- Existing Go behaviour remains passing: run go test ./... and configured checks after updating deliberately superseded gate expectations (existing)

## Feasibility
- evidence: {"id":"example-observer","kind":"observer","inputs":["features/steps/shape-example-check-normalises-format__harness.js","features/steps/shape-example-check-normalises-format__steps.js"],"observers":["features/steps/shape-example-check-normalises-format__steps.js"],"checks":[{"name":"good","command":"node features/steps/shape-example-check-normalises-format__steps.js good","exit":0,"output":"observer accepted\n"},{"name":"bad","command":"node features/steps/shape-example-check-normalises-format__steps.js bad","exit":1,"output":"observer rejected\n"}]}
