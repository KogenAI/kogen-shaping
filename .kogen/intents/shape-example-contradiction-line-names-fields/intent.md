# Name differing fields in example contradiction

Slug: shape-example-contradiction-line-names-fields
Model: not applicable — the change is the text of one refusal line built from four strings; there is no state to model
Acceptance:
- .kogen/intents/shape-example-contradiction-line-names-fields/acceptance.feature
- features/steps/shape-example-contradiction-line-names-fields__steps.js
- features/steps/shape-example-contradiction-line-names-fields__driver/harness_test.go.txt

When the example check refuses a draft because an Examples row contradicts the rules, the refusal line shows the whole expected and derived values and nothing else. Real rows give values 150 to 250 characters long, so the one differing field is hard to find, and `cutProblems` cuts a line over 300 runes at its tail, which loses the derived value and the checker's why. A refusal whose derived value merely lacks one field looked wrong, and the record did not carry the why that explained it.

Only the text of the contradiction lines changes. One helper in shape/examples.go, `exampleContradictionLine(scenario, expected, derived, why string) string`, builds the line and `exampleCheck` calls it where it now formats the line with `fmt.Sprintf`.

- When `expected` and `derived` both parse (with `exampleValue`) as JSON objects, the line is the scenario name, `: differs in `, the names of the top-level fields whose values differ, are missing from derived or exist only in derived, sorted alphabetically and joined by `, `, then ` | expected `, the expected value, ` | rules give `, the derived value, then the why segment. A field differs when `exampleEqual` finds its two values unequal.
- In every other case (a value that is not an object, or does not parse) the line is the scenario name, `: expected `, the expected value, `, rules give `, the derived value, then the why segment.
- The why segment is ` | why: ` followed by the why with each whitespace run replaced by one space and the ends trimmed. When the why is empty after that, no segment is added, so the line has no trailing ` | why: `.
- The field names come first and the why segment is last, so the existing `cutProblems` cut (which keeps the ` | why: ` segment of a line over 300 runes) keeps both. `cutProblems` is not changed.

Nothing else changes: which drafts pass or are refused (`exampleVerdict`, `exampleEqual`), the `example_check` event, the schema and the instructions given to the checker stay as they are, and no configuration is added. The model-free Go test file shape/examples_line_test.go is added next to shape/examples_test.go. Its test functions are named `TestExampleContradictionLine` and `TestExampleContradictionLineCut`; the name `TestExampleContradictionLineHarness` is taken by the acceptance harness.

Each of the five acceptance scenarios has its own wrong-implementation control in `.kogen/intents/shape-example-contradiction-line-names-fields/controls.json`, which names the scenario exactly as acceptance.feature writes it.

## Decisions
- The helper is `exampleContradictionLine(scenario, expected, derived, why string) string` in shape/examples.go and `exampleCheck` uses it; the request does not name it, so this name and signature are chosen here (evidence: shape/examples.go) (assumption: the helper name and signature are acceptable) (source: request)
- The differing top-level fields are found with the existing `exampleEqual` on each field value, so a numerically equal or reordered value does not count as differing, matching the verdict the line explains (decision: D2, round 1) (evidence: shape/examples.go; shape/examples_test.go) (assumption: semantic comparison per field, not textual, is what the operator wants) (source: request)
- The why is normalised with `strings.Fields` joined by one space; an empty result adds no segment, and `cutProblems` is unchanged because it already keeps a trailing ` | why: ` segment (decision: D3, round 1) (evidence: shape/checkorder.go) (source: request)
- Acceptance drives the helper and `cutProblems` through a Go test harness copied into the shape package by a Go build overlay, one `go test` run per Examples row passed through an environment variable; on a tree without the helper the step substitutes the old line format so the harness still runs and the Then steps fail on the line text (decision: D4, round 1) (evidence: features/steps/approve-static-reuse-line-explains-unknown__steps.js; shape/checkorder.go) (source: request)
- Helper exampleContradictionLine(scenario, expected, derived, why string) string in shape/examples.go builds the contradiction line; exampleCheck calls it (decision: D1, round 1) (constraint: verdict logic, exampleEqual, cutProblems, the example_check event, schema and checker instructions unchanged) (evidence: shape/examples.go; shape/checkorder.go; shape/examples_test.go) (assumption: Helper name is chosen here; the correct and wrong overlays were compiled and run in an external scratch overlay only, not through Kogen.) (source: request)

## Source comparison
- Observed refusal line, from the request: `Log HTTP outcomes without retries — code 299: expected {"http_status":204,"posts":1,"deliveries":1,"payload_matches":true,"signed":true,"state":"completed","code":299,"error_present":false,"signature_hidden":true,"html_has_outcome":true}, rules give {"posts":1,"deliveries":1,"payload_matches":true,"` cut inside the derived value with no why; superseded by the field-naming line, addressed in scenario "Cut refusal line keeps the differing field and the reason".
- Missing field in derived, from the request: the old line `Log outcome: expected {"http_status":204,"code":299}, rules give {"code":299}` is superseded by `Log outcome: differs in http_status | expected ...`; addressed in scenario "Contradiction line names a field the rules leave out".
- Several fields, sorted, one only in derived: the line says `differs in retries, state` and not `differs in state`; addressed in scenario "Contradiction line lists every differing field in alphabetical order".
- Non-object values keep the two-value form: the line for a list never starts the object form `Order: differs in `; addressed in scenario "Contradiction line for a list keeps the two values".
- Why on one line, empty why omitted: addressed in scenario "Contradiction line puts the why on one line and omits an empty why".
- Wrong-implementation controls in controls.json, one per scenario: carried in `.kogen/intents/shape-example-contradiction-line-names-fields/controls.json`.
- No change to cutProblems, verdict logic, the example_check event, the schema, the checker instructions or configuration: omitted by design.
- Existing tests that assert the old line: none found by searching for `rules give`; nothing to update.

## Tests
- A contradiction line names a field the rules leave out: add TestExampleContradictionLine in shape/examples_line_test.go, case "Log outcome" (unit)
- A contradiction line lists every differing field in alphabetical order, including one only in derived: add the "Retry" case to TestExampleContradictionLine in shape/examples_line_test.go (unit)
- A contradiction line for non-object values keeps the two-value form: add the "Order" case to TestExampleContradictionLine in shape/examples_line_test.go (unit)
- The why is on one line and omitted when empty: add the whitespace and empty-why cases to TestExampleContradictionLine in shape/examples_line_test.go (unit)
- A cut refusal line keeps the differing field and the why: add TestExampleContradictionLineCut in shape/examples_line_test.go, passing the line through cutProblems (unit)

## Feasibility
- none
